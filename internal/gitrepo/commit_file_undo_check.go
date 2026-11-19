package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrUnsupportedCommitFileUndo = errors.New("仅支持当前 HEAD 祖先中的单父提交及同路径普通文本修改")

// CheckCommitFileUndo is a read-only eligibility check. A later write must
// repeat these checks against the confirmation's captured target; this result
// does not authorize a write against a changed repository.
func (r *Repository) CheckCommitFileUndo(ctx context.Context, sha, path string, expected Status) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return r.checkCommitFileUndo(ctx, sha, path, expected, "")
}

// ignoredTemp is exclusively the caller's newly created, untracked adjacent
// file. Tracked changes at that name must still reject publication.
func (r *Repository) checkCommitFileUndo(ctx context.Context, sha, path string, expected Status, ignoredTemp string) error {
	if !oidRE.MatchString(sha) {
		return errors.New("无效的提交 SHA")
	}
	if err := r.path(path); err != nil {
		return err
	}
	checkCurrent := func() error {
		current, err := r.Status(ctx)
		if err != nil {
			return err
		}
		if current.Head != expected.Head || current.Branch != expected.Branch || !oidRE.MatchString(expected.Head) {
			return ErrIntegrationTargetChanged
		}
		for _, entry := range current.Entries {
			if ignoredTemp != "" && entry.Path == ignoredTemp && entry.Index == '?' && entry.Worktree == '?' {
				continue
			}
			return ErrDirtyWorkingTree
		}
		return r.idle(ctx)
	}
	if err := checkCurrent(); err != nil {
		return err
	}
	parents, err := run(ctx, r.Root, "", "show", "-s", "--format=%P", sha)
	if err != nil {
		return err
	}
	parent := strings.Fields(parents)
	if len(parent) != 1 {
		return ErrUnsupportedCommitFileUndo
	}
	if _, err := run(ctx, r.Root, "", "merge-base", "--is-ancestor", sha, expected.Head); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w：来源提交不在当前历史中", ErrUnsupportedCommitFileUndo)
	}
	files, err := r.CommitFiles(ctx, sha)
	if err != nil {
		return err
	}
	found := false
	for _, file := range files {
		if file.Path == path {
			found = file.Status == "M" && file.OriginalPath == ""
			break
		}
	}
	if !found {
		return ErrUnsupportedCommitFileUndo
	}
	oldMode, oldBlob, err := r.undoRegularBlob(ctx, parent[0], path)
	if err != nil {
		return err
	}
	newMode, newBlob, err := r.undoRegularBlob(ctx, sha, path)
	if err != nil {
		return err
	}
	if oldMode != newMode || oldBlob == newBlob {
		return ErrUnsupportedCommitFileUndo
	}
	currentMode, currentBlob, err := r.undoRegularBlob(ctx, expected.Head, path)
	if err != nil {
		return err
	}
	if currentMode != newMode {
		return ErrUnsupportedCommitFileUndo
	}
	for _, blob := range []string{oldBlob, newBlob, currentBlob} {
		text := &undoTextCheck{}
		err := runOutput(ctx, r.Root, "", text, "cat-file", "blob", blob)
		if text.binary {
			return ErrUnsupportedCommitFileUndo
		}
		if err != nil {
			return err
		}
	}
	// Respect Git's binary classification as well as rejecting NUL bytes. A
	// user attribute may mark an otherwise ASCII blob as binary.
	stats, err := run(ctx, r.Root, "", "diff", "--numstat", "-z", "--no-ext-diff", "--no-textconv", parent[0], sha, "--", path)
	if err != nil {
		return err
	}
	counts := strings.SplitN(stats, "\t", 3)
	if len(counts) != 3 || counts[0] == "-" || counts[1] == "-" {
		return ErrUnsupportedCommitFileUndo
	}
	autocrlf, err := run(ctx, r.Root, "", "config", "--type=bool-or-str", "--default=false", "--get", "core.autocrlf")
	if err != nil {
		return err
	}
	if strings.TrimSpace(autocrlf) != "false" {
		return fmt.Errorf("%w：暂不支持工作区内容转换", ErrUnsupportedCommitFileUndo)
	}
	// Scratch applies canonical blob patches. Until worktree conversions are
	// supported, reject conversion attributes rather than publish blob text.
	attrs, err := run(ctx, r.Root, "", "check-attr", "-z", "filter", "working-tree-encoding", "ident", "text", "eol", "--", path)
	if err != nil {
		return err
	}
	values := strings.Split(strings.TrimSuffix(attrs, "\x00"), "\x00")
	if len(values) != 15 {
		return ErrUnsupportedCommitFileUndo
	}
	for i := 2; i < len(values); i += 3 {
		if values[i] != "unspecified" && values[i] != "unset" {
			return fmt.Errorf("%w：暂不支持工作区内容转换", ErrUnsupportedCommitFileUndo)
		}
	}
	// Status can hide real worktree edits behind assume-unchanged/skip-worktree.
	// Hash through Git's path filters without writing an object or index entry.
	workingBlob, err := run(ctx, r.Root, "", "hash-object", "--path="+path, "--", path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(workingBlob) != currentBlob {
		return ErrDirtyWorkingTree
	}
	rawBlob, err := run(ctx, r.Root, "", "hash-object", "--no-filters", "--", path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(rawBlob) != currentBlob {
		return fmt.Errorf("%w：暂不支持工作区内容转换", ErrUnsupportedCommitFileUndo)
	}
	return checkCurrent()
}

func (r *Repository) undoRegularBlob(ctx context.Context, sha, path string) (string, string, error) {
	out, err := run(ctx, r.Root, "", "ls-tree", "--full-tree", "-z", sha, "--", path)
	if err != nil {
		return "", "", err
	}
	header, name, ok := strings.Cut(out, "\t")
	fields := strings.Fields(header)
	if !ok || strings.TrimSuffix(name, "\x00") != path || len(fields) != 3 || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
		return "", "", ErrUnsupportedCommitFileUndo
	}
	return fields[0], fields[2], nil
}

type undoTextCheck struct{ binary bool }

func (w *undoTextCheck) Write(p []byte) (int, error) {
	if bytes.IndexByte(p, 0) >= 0 {
		w.binary = true
		return 0, ErrUnsupportedCommitFileUndo
	}
	return len(p), nil
}
