package gitrepo

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Check the source tree even when no parent revisions need pushing. Git's
// recurse-submodules=check skips that case and children without remotes.
func (r *Repository) checkPublishedSubmodules(ctx context.Context, source string) error {
	out, err := run(ctx, r.Root, "", "ls-tree", "-r", "-z", source)
	if err != nil {
		return err
	}
	for _, entry := range strings.Split(out, "\x00") {
		meta, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[0] != "160000" {
			continue
		}
		if err := r.path(path); err != nil {
			return err
		}
		full := filepath.Join(r.Root, path)
		child, err := Open(ctx, full)
		if err != nil || canonical(full) != canonical(child.Root) {
			return fmt.Errorf("请先初始化子模块 %q，以检查提交 %s 是否已发布", path, fields[2])
		}
		remotes, err := run(ctx, child.Root, "", "remote")
		if err != nil {
			return err
		}
		refs, err := run(ctx, child.Root, "", "for-each-ref", "--contains="+fields[2], "--format=%(refname)", "refs/remotes")
		if err != nil {
			return fmt.Errorf("无法检查子模块 %q 的提交 %s：%w", path, fields[2], err)
		}
		published := false
		for _, remote := range strings.Split(strings.TrimSuffix(remotes, "\n"), "\n") {
			if remote == "" {
				continue
			}
			for _, ref := range strings.Split(strings.TrimSuffix(refs, "\n"), "\n") {
				if strings.HasPrefix(ref, "refs/remotes/"+remote+"/") {
					published = true
				}
			}
		}
		if !published {
			return fmt.Errorf("子模块 %q 的提交 %s 未被已配置远端的跟踪引用包含；请先进入子模块推送或 Fetch 核对", path, fields[2])
		}
	}
	return nil
}
