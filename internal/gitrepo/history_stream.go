package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// HistoryStream reads one pinned Git traversal through a bounded channel. Next
// has one caller at a time; Close may run concurrently to release backpressure.
type HistoryStream struct {
	ctx     context.Context
	cancel  context.CancelFunc
	tips    []string
	commits chan Commit
	done    chan struct{}
	err     error // Written before done closes.
	pending *Commit
}

// OpenHistoryStream continues already pinned tips from offset. Unlike repeated
// HistoryPage calls, it skips the prefix just once for the entire traversal.
func (r *Repository) OpenHistoryStream(ctx context.Context, tips []string, offset int) (*HistoryStream, error) {
	if tips == nil || offset < 0 {
		return nil, fmt.Errorf("无效的历史流范围")
	}
	for _, tip := range tips {
		if !oidRE.MatchString(tip) {
			return nil, fmt.Errorf("无效的历史起始 SHA")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	stream := &HistoryStream{ctx: ctx, cancel: cancel, tips: append([]string{}, tips...), commits: make(chan Commit, 1), done: make(chan struct{})}
	if len(tips) == 0 {
		close(stream.commits)
		close(stream.done)
		return stream, nil
	}
	go func() {
		output := historyStreamOutput{ctx: ctx, commits: stream.commits}
		args := []string{"log", "-z", historyFormat, "--topo-order", "--skip=" + strconv.FormatInt(int64(offset), 10), "--stdin", "--"}
		stream.err = runOutputContext(ctx, r.Root, strings.Join(stream.tips, "\n")+"\n", &output, args...)
		if stream.err == nil && len(output.record) != 0 {
			stream.err = errors.New("无效的提交历史记录")
		}
		close(stream.done)
		close(stream.commits)
	}()
	return stream, nil
}

func (s *HistoryStream) Close() { s.cancel() }

func (s *HistoryStream) read(ctx context.Context) (Commit, error) {
	if err := ctx.Err(); err != nil {
		return Commit{}, err
	}
	select {
	case <-ctx.Done():
		return Commit{}, ctx.Err()
	case commit, ok := <-s.commits:
		if ok {
			return commit, nil
		}
		<-s.done
		if s.err != nil {
			return Commit{}, s.err
		}
		return Commit{}, io.EOF
	}
}

// Next returns at most limit rows plus one retained lookahead. No complete
// history string or expanded prefix is buffered in memory.
func (s *HistoryStream) Next(limit int) (HistoryPage, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	return s.next(ctx, limit)
}

func (s *HistoryStream) next(ctx context.Context, limit int) (HistoryPage, error) {
	if limit < 1 || limit > 2000 {
		return HistoryPage{}, fmt.Errorf("无效的历史分页范围")
	}
	if err := s.ctx.Err(); err != nil {
		return HistoryPage{}, err
	}
	page := HistoryPage{Tips: append([]string{}, s.tips...)}
	if s.pending != nil {
		page.Commits = append(page.Commits, *s.pending)
		s.pending = nil
	}
	for len(page.Commits) <= limit {
		commit, err := s.read(ctx)
		if errors.Is(err, io.EOF) {
			return page, nil
		}
		if err != nil {
			s.Close()
			return HistoryPage{}, err
		}
		if len(page.Commits) == limit {
			s.pending = &commit
			page.HasMore = true
			return page, nil
		}
		page.Commits = append(page.Commits, commit)
	}
	return page, nil
}

type historyStreamOutput struct {
	ctx     context.Context
	commits chan<- Commit
	record  []byte
	fields  int
}

func (w *historyStreamOutput) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		end := bytes.IndexByte(p, 0)
		count := len(p)
		if end >= 0 {
			count = end + 1
		}
		if len(w.record)+count > outputLimit {
			return 0, errors.New("提交历史记录超过 4 MiB 显示上限")
		}
		w.record = append(w.record, p[:count]...)
		p = p[count:]
		if end < 0 {
			continue
		}
		w.fields++
		if w.fields != historyFieldCount {
			continue
		}
		commits, err := parseHistory(string(w.record))
		if err != nil {
			return 0, err
		}
		select {
		case w.commits <- commits[0]:
		case <-w.ctx.Done():
			return 0, w.ctx.Err()
		}
		w.record = w.record[:0]
		w.fields = 0
	}
	return n, nil
}
