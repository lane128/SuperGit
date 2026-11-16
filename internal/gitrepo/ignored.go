package gitrepo

import (
	"context"
	"fmt"
	"strings"
)

// IgnoredFileError identifies a refusal that protects ignored local content.
// Error preserves the legacy diagnostic for callers that render it directly.
// Path is the raw repository-relative path, not a pathspec or display label.
type IgnoredFileError struct {
	Path           string
	WouldOverwrite bool
}

func (e *IgnoredFileError) Error() string {
	if e.WouldOverwrite {
		return fmt.Sprintf("目标会覆盖被忽略文件 %q，请先移出或备份", e.Path)
	}
	return fmt.Sprintf("存在被忽略文件 %q；为保留内容，此操作已停止，请先移出或备份", e.Path)
}

func (r *Repository) ignoredFiles(ctx context.Context) ([]string, error) {
	out, err := run(ctx, r.Root, "", "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(out, "\x00"), "\x00"), nil
}
func (r *Repository) noIgnored(ctx context.Context) error {
	files, err := r.ignoredFiles(ctx)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		return &IgnoredFileError{Path: files[0]}
	}
	return nil
}
func (r *Repository) protectIgnored(ctx context.Context, ref string) error {
	files, err := r.ignoredFiles(ctx)
	if err != nil || len(files) == 0 {
		return err
	}
	out, err := run(ctx, r.Root, "", "ls-tree", "-r", "--name-only", "-z", ref)
	if err != nil {
		return err
	}
	for _, incoming := range strings.Split(out, "\x00") {
		if incoming == "" {
			continue
		}
		for _, local := range files {
			if local == incoming || strings.HasPrefix(local, incoming+"/") || strings.HasPrefix(incoming, local+"/") {
				return &IgnoredFileError{Path: local, WouldOverwrite: true}
			}
		}
	}
	return nil
}
