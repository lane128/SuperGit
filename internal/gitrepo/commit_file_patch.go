package gitrepo

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// WriteCommitFilePatch streams one changed file relative to the commit's first
// parent (or the empty tree for a root). It never uses the capped diff preview.
func (r *Repository) WriteCommitFilePatch(ctx context.Context, sha, path string, dst io.Writer) error {
	if !oidRE.MatchString(sha) {
		return fmt.Errorf("无效的提交 SHA")
	}
	if err := r.path(path); err != nil {
		return err
	}
	if dst == nil {
		return fmt.Errorf("补丁输出不能为空")
	}
	files, err := r.CommitFiles(ctx, sha)
	if err != nil {
		return err
	}
	paths := []string{path}
	rename := false
	found := false
	for _, file := range files {
		if file.Path == path {
			found = true
			if file.OriginalPath != "" && file.OriginalPath != path {
				paths = append(paths, file.OriginalPath)
				rename = strings.HasPrefix(file.Status, "R")
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("文件不在此提交的更改中：%s", path)
	}
	parents, err := run(ctx, r.Root, "", "show", "-s", "--format=%P", sha)
	if err != nil {
		return err
	}
	options := []string{"--binary", "--full-index", "--no-ext-diff", "--no-textconv", "--no-color", "--src-prefix=a/", "--dst-prefix=b/", "--submodule=short", "-M"}
	// Both names are needed for rename detection, but an unrelated addition at
	// the vacated name must not become a second file in this export.
	if rename {
		options = append(options, "--diff-filter=R")
	}
	var args []string
	if p := strings.Fields(parents); len(p) > 0 {
		args = append([]string{"diff"}, options...)
		args = append(args, p[0], sha)
	} else {
		args = append([]string{"show", "--format=", "--root"}, options...)
		args = append(args, sha)
	}
	args = append(args, "--")
	args = append(args, paths...)
	return runOutput(ctx, r.Root, "", dst, args...)
}
