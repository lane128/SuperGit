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
		full := filepath.Join(r.Root, path)
		if _, e := os.Lstat(filepath.Join(full, ".git")); e == nil {
			child, e := Open(ctx, full)
			if e != nil {
				m.Error = e.Error()
			} else {
				actual, _ := filepath.EvalSymlinks(full)
				root, _ := filepath.EvalSymlinks(child.Root)
				if actual != "" && actual == root {
					m.Initialized = true
					s, e := child.Status(ctx)
					if e != nil {
						m.Error = e.Error()
					} else {
						m.Current = s.Head
						m.Branch = s.Branch
						m.Dirty = len(s.Entries) > 0
					}
				}
			}
		}
		result = append(result, m)
	}
	return result, nil
}
