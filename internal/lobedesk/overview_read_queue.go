package lobedesk

import (
	"context"
	"slices"
	"sync"
)

// The worker owns Git reads; the UI only publishes viewport priorities or
// retries. The queue never reads UI state from its worker goroutine.
type overviewReadQueue struct {
	mu      sync.Mutex
	pending []detailFileKey
	visible []detailFileKey
	wake    chan struct{}
}

func newOverviewReadQueue(keys []detailFileKey) *overviewReadQueue {
	return &overviewReadQueue{pending: slices.Clone(keys), wake: make(chan struct{}, 1)}
}

func (q *overviewReadQueue) prioritize(keys []detailFileKey) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if slices.Equal(q.visible, keys) {
		return
	}
	q.visible = slices.Clone(keys)
	wanted := make(map[detailFileKey]bool, len(keys))
	for _, key := range keys {
		wanted[key] = true
	}
	ordered := make([]detailFileKey, 0, len(q.pending))
	for _, key := range q.pending {
		if wanted[key] {
			ordered = append(ordered, key)
		}
	}
	for _, key := range q.pending {
		if !wanted[key] {
			ordered = append(ordered, key)
		}
	}
	q.pending = ordered
}

func (q *overviewReadQueue) push(key detailFileKey) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if slices.Contains(q.pending, key) {
		return false
	}
	q.pending = append(q.pending, key)
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return true
}

func (q *overviewReadQueue) next(ctx context.Context) (detailFileKey, bool) {
	for {
		if ctx.Err() != nil {
			return detailFileKey{}, false
		}
		q.mu.Lock()
		if len(q.pending) > 0 {
			key := q.pending[0]
			q.pending = q.pending[1:]
			q.mu.Unlock()
			return key, true
		}
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return detailFileKey{}, false
		case <-q.wake:
		}
	}
}
