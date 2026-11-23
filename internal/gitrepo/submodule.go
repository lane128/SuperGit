package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (r *Repository) Submodules(ctx context.Context) ([]Submodule, error) {
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
		index, e := run(ctx, r.Root, "", "ls-files", "--stage", "-z", "--", path)
		if e != nil {
			return nil, e
		}
		for _, entry := range strings.Split(index, "\x00") {
			meta, name, ok := strings.Cut(entry, "\t")
			fields := strings.Fields(meta)
			if ok && name == path && len(fields) == 3 && fields[0] == "160000" && fields[2] == "0" {
				m.Index = fields[1]
			}
		}
		if r.hasHead(ctx) {
			tree, e := run(ctx, r.Root, "", "ls-tree", "-z", "HEAD", "--", path)
			if e != nil {
				return nil, e
			}
			for _, entry := range strings.Split(tree, "\x00") {
				meta, name, ok := strings.Cut(entry, "\t")
				f := strings.Fields(meta)
				if ok && name == path && len(f) == 3 && f[0] == "160000" {
					m.Recorded = f[2]
				}
			}
		}
		r.readSubmoduleCheckout(ctx, &m)
		result = append(result, m)
	}
	return result, nil
}

// SubmoduleCheckout reads the local checkout of a known module path. The
// parent index and HEAD gitlinks are only needed when listing Submodules.
func (r *Repository) SubmoduleCheckout(ctx context.Context, path string) Submodule {
	m := Submodule{Path: path}
	if err := r.path(path); err != nil {
		m.Error = err.Error()
		return m
	}
	r.readSubmoduleCheckout(ctx, &m)
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
