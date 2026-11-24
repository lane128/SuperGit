package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

const childWorkerLimit = 4

func (r *Repository) Submodules(ctx context.Context) ([]Submodule, error) {
	rows, err := r.SubmoduleInventory(ctx)
	if err != nil {
		return nil, err
	}
	details := make(map[string]Submodule)
	var mu sync.Mutex
	r.InspectSubmodules(ctx, rows, func(m Submodule) { mu.Lock(); details[m.Path] = m; mu.Unlock() })
	for i, m := range rows {
		if d, ok := details[m.Path]; ok {
			d.Name, d.Recorded, d.Index = m.Name, m.Recorded, m.Index
			rows[i] = d
		}
	}
	return rows, ctx.Err()
}

// InspectSubmodules schedules each valid checkout once and streams completed rows.
// The callback may run concurrently on up to four workers.
func (r *Repository) InspectSubmodules(ctx context.Context, rows []Submodule, publish func(Submodule)) {
	r.inspectSubmodules(ctx, rows, childWorkerLimit, publish)
}

func (r *Repository) inspectSubmodules(ctx context.Context, rows []Submodule, workers int, publish func(Submodule)) {
	jobs := make(chan string)
	var wg sync.WaitGroup
	for range min(workers, len(rows)) {
		wg.Go(func() {
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				m := r.SubmoduleCheckout(ctx, path)
				if ctx.Err() == nil {
					publish(m)
				}
			}
		})
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.Path] || row.Error != "" {
			continue
		}
		seen[row.Path] = true
		select {
		case jobs <- row.Path:
		case <-ctx.Done():
		}
	}
	close(jobs)
	wg.Wait()
}

func (r *Repository) SubmoduleInventory(ctx context.Context) ([]Submodule, error) {
	if _, err := os.Lstat(filepath.Join(r.Root, ".gitmodules")); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	out, err := run(ctx, r.Root, "", "config", "--null", "--file", ".gitmodules", "--get-regexp", `^submodule\..*\.path$`)
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	var result []Submodule
	var paths []string
	for _, record := range strings.Split(out, "\x00") {
		key, path, ok := strings.Cut(record, "\n")
		if !ok {
			continue
		}
		m := Submodule{Name: strings.TrimSuffix(strings.TrimPrefix(key, "submodule."), ".path"), Path: path}
		if err := r.path(path); err != nil {
			m.Error = err.Error()
			result = append(result, m)
			continue
		}
		paths = append(paths, path)
		m.Pending = true
		result = append(result, m)
	}
	if len(paths) > 0 {
		readLinks := func(args []string, index bool) (map[string]string, error) {
			out, err := run(ctx, r.Root, "", append(args, paths...)...)
			if err != nil {
				return nil, err
			}
			links := map[string]string{}
			for _, entry := range strings.Split(out, "\x00") {
				meta, name, ok := strings.Cut(entry, "\t")
				f := strings.Fields(meta)
				if !ok || len(f) != 3 || f[0] != "160000" {
					continue
				}
				if !index {
					links[name] = f[2]
				} else if f[2] == "0" {
					links[name] = f[1]
				}
			}
			return links, nil
		}
		index, err := readLinks([]string{"ls-files", "--stage", "-z", "--"}, true)
		if err != nil {
			return nil, err
		}
		var tree map[string]string
		head, err := r.optionalRef(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
		if err != nil {
			return nil, err
		}
		if head != "" {
			tree, err = readLinks([]string{"ls-tree", "-r", "-z", head, "--"}, false)
			if err != nil {
				return nil, err
			}
		}
		for i := range result {
			result[i].Index = index[result[i].Path]
			result[i].Recorded = tree[result[i].Path]
		}
	}
	// .gitmodules keeps the order modules were added, and the config read
	// returns that order. Every list the app shows (sidebar, Workspaces,
	// command palette) reads this slice, so sort it once here instead of in
	// each renderer.
	slices.SortStableFunc(result, func(a, b Submodule) int {
		if lowA, lowB := strings.ToLower(a.Name), strings.ToLower(b.Name); lowA != lowB {
			return strings.Compare(lowA, lowB)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

// SubmoduleCheckout reads the local checkout of a known module path. The
// parent index and HEAD gitlinks are only needed when listing Submodules.
func (r *Repository) SubmoduleCheckout(ctx context.Context, path string) Submodule {
	m := Submodule{Path: path}
	pool := checkoutWorkers.acquire(r.Root)
	defer checkoutWorkers.release(r.Root, pool)
	select {
	case pool.slots <- struct{}{}:
		defer func() { <-pool.slots }()
	case <-ctx.Done():
		m.Error = ctx.Err().Error()
		return m
	}
	if err := r.path(path); err != nil {
		m.Error = err.Error()
		return m
	}
	r.readSubmoduleCheckout(ctx, &m)
	m.CheckoutKnown = m.Error == ""
	return m
}

func (r *Repository) readSubmoduleCheckout(ctx context.Context, m *Submodule) {
	full := filepath.Join(r.Root, m.Path)
	if _, err := os.Lstat(filepath.Join(full, ".git")); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			m.Error = err.Error()
		}
		return
	}
	child, err := Open(ctx, full)
	if err != nil {
		m.Error = err.Error()
		return
	}
	actual, _ := filepath.EvalSymlinks(full)
	root, _ := filepath.EvalSymlinks(child.Root)
	if actual == "" || actual != root {
		return
	}
	m.Initialized = true
	status, err := child.Status(ctx)
	if err != nil {
		m.Error = err.Error()
		return
	}
	m.Current, m.Branch = status.Head, status.Branch
	m.Dirty = len(status.Entries) > 0
	m.ChangedFiles = status.WorkspaceFileCount()
}

// Reopening a checkout replaces its Repository while cancelled processes may
// still be exiting. Share the bound until the last caller leaves that root.
type childWorkerPool struct {
	slots chan struct{}
	users int
}
type childWorkerPools struct {
	sync.Mutex
	roots map[string]*childWorkerPool
}

var checkoutWorkers = childWorkerPools{roots: make(map[string]*childWorkerPool)}

func (p *childWorkerPools) acquire(root string) *childWorkerPool {
	p.Lock()
	defer p.Unlock()
	pool := p.roots[root]
	if pool == nil {
		pool = &childWorkerPool{slots: make(chan struct{}, childWorkerLimit)}
		p.roots[root] = pool
	}
	pool.users++
	return pool
}
func (p *childWorkerPools) release(root string, pool *childWorkerPool) {
	p.Lock()
	defer p.Unlock()
	pool.users--
	if pool.users == 0 {
		delete(p.roots, root)
	}
}
