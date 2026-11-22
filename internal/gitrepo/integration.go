package gitrepo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Operation struct{ Kind, Head, Token string }

func (r *Repository) gitPath(ctx context.Context, name string) (string, error) {
	out, err := run(ctx, r.Root, "", "rev-parse", "--path-format=absolute", "--git-path", name)
	return strings.TrimSuffix(out, "\n"), err
}
func (r *Repository) Operation(ctx context.Context) (Operation, error) {
	op := Operation{}
	gitDir, err := r.gitPath(ctx, ".")
	if err != nil {
		return op, err
	}
	for _, item := range []struct{ path, kind string }{{"rebase-merge", "rebase"}, {"rebase-apply", "rebase"}, {"MERGE_HEAD", "merge"}, {"CHERRY_PICK_HEAD", "cherry-pick"}, {"REVERT_HEAD", "revert"}} {
		path := filepath.Join(gitDir, item.path)
		info, err := os.Stat(path)
		if err == nil {
			op.Kind = item.kind
			h := sha256.New()
			fmt.Fprintf(h, "%s:%d", path, info.ModTime().UnixNano())
			files := []string{path}
			if info.IsDir() {
				files = nil
				for _, name := range []string{"head-name", "onto", "orig-head", "stopped-sha", "original-commit", "msgnum", "next", "last", "applying"} {
					files = append(files, filepath.Join(path, name))
				}
				if item.path == "rebase-apply" {
					if _, e := os.Stat(filepath.Join(path, "applying")); e == nil {
						op.Kind = "am"
					}
				}
			}
			for _, file := range files {
				data, e := os.ReadFile(file)
				if errors.Is(e, os.ErrNotExist) {
					continue
				}
				if e != nil {
					return Operation{}, e
				}
				fmt.Fprintf(h, "%s:%x;", file, sha256.Sum256(data))
			}
			op.Token = fmt.Sprintf("%x", h.Sum(nil))
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return op, err
		}
	}
	if op.Kind != "" {
		head, err := r.resolve(ctx, "HEAD")
		if err != nil {
			return op, err
		}
		op.Head = head
	}
	return op, nil
}

// OperationMessage returns the commit message Git prepared for the pending
// merge, cherry-pick or revert. Git writes it before it stops on a conflict,
// so the commit field can start with Git's own text instead of making the user
// retype what `git commit` would already use.
func (r *Repository) OperationMessage(ctx context.Context) (string, error) {
	gitDir, err := r.gitPath(ctx, ".")
	if err != nil {
		return "", err
	}
	for _, name := range []string{"MERGE_MSG", filepath.Join("rebase-merge", "message")} {
		data, err := os.ReadFile(filepath.Join(gitDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if len(data) > 1<<20 {
			return "", fmt.Errorf("待提交说明超过 1 MiB 上限")
		}
		message := strings.TrimSpace(string(data))
		if !utf8.ValidString(message) || strings.ContainsRune(message, '\x00') {
			return "", fmt.Errorf("待提交说明不是有效的 UTF-8 文本")
		}
		// Git's prepared message carries guidance comments such as
		// "# Conflicts:". `git commit -m` strips them, so drop them here too
		// and the field shows exactly what the commit will record.
		lines := strings.Split(message, "\n")
		kept := lines[:0]
		for _, line := range lines {
			if strings.HasPrefix(line, "#") {
				continue
			}
			kept = append(kept, line)
		}
		return strings.TrimSpace(strings.Join(kept, "\n")), nil
	}
	return "", nil
}

func (r *Repository) idle(ctx context.Context) error {
	op, err := r.Operation(ctx)
	if err != nil {
		return err
	}
	if op.Kind != "" {
		return errors.New("已有进行中的 " + op.Kind + "，请先继续或中止")
	}
	return nil
}

var ErrIntegrationTargetChanged = errors.New("current HEAD or branch changed before integration")

// IntegrateChecked uses an immutable incoming commit and checks which current
// branch the confirmation authorized. External Git does not share writeMu.
func (r *Repository) IntegrateChecked(ctx context.Context, kind, sha string, expected Status) error {
	if !oidRE.MatchString(sha) {
		return errors.New("无效的提交 SHA")
	}
	return r.integrate(ctx, kind, sha, &expected)
}

func (r *Repository) Integrate(ctx context.Context, kind, ref string) error {
	return r.integrate(ctx, kind, ref, nil)
}

func (r *Repository) integrate(ctx context.Context, kind, ref string, expected *Status) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if expected != nil {
		current, err := r.Status(ctx)
		if err != nil {
			return err
		}
		if current.Head != expected.Head || current.Branch != expected.Branch {
			return ErrIntegrationTargetChanged
		}
	}
	if err := r.operationReady(ctx); err != nil {
		return err
	}
	sha, err := r.resolve(ctx, ref)
	if err != nil {
		return err
	}
	var args []string
	switch kind {
	case "merge":
		args = []string{"merge", "--no-edit", "--no-autostash", "--no-overwrite-ignore", "--", sha}
	case "rebase":
		// Rebase can check out intermediate replay trees, not only its final tree.
		if err := r.noIgnored(ctx); err != nil {
			return err
		}
		args = []string{"rebase", "--no-autostash", "--", sha}
	case "cherry-pick", "revert":
		target := sha
		if kind == "revert" {
			parents, e := run(ctx, r.Root, "", "show", "-s", "--format=%P", sha)
			if e != nil {
				return e
			}
			fields := strings.Fields(parents)
			if len(fields) == 0 {
				target = ""
			} else {
				target = fields[0]
			}
		}
		if target != "" {
			if err := r.protectIgnored(ctx, target); err != nil {
				return err
			}
		}
		args = []string{kind, "--no-edit", sha}
	default:
		return errors.New("不支持的整合操作")
	}
	_, err = run(ctx, r.Root, "", args...)
	return wrapDirtyTreeError(err)
}
func (r *Repository) ContinueChecked(ctx context.Context, expected Operation) error {
	return r.continueOperation(ctx, expected.Kind, &expected)
}

func (r *Repository) Continue(ctx context.Context, kind string) error {
	return r.continueOperation(ctx, kind, nil)
}

func (r *Repository) continueOperation(ctx context.Context, kind string, expected *Operation) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	op, err := r.operationForControl(ctx, expected)
	if err != nil {
		return err
	}
	if op.Kind != kind || kind == "" {
		return errors.New("进行中的操作已变化，请刷新")
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	for _, e := range s.Entries {
		if e.Conflict {
			return errors.New("仍有未解决冲突")
		}
	}
	if kind == "merge" {
		_, err = run(ctx, r.Root, "", "commit", "--no-edit")
	} else {
		_, err = run(ctx, r.Root, "", "-c", "core.editor=true", kind, "--continue")
	}
	return err
}

// Skip drops the commit an operation stopped on and continues. A merge has
// no commit to skip, so it can only continue or abort.
var ErrOperationTargetChanged = errors.New("pending operation changed before control action")

// operationForControl compares the operation displayed before confirmation.
// Callers hold writeMu; external Git processes do not share that lock.
func (r *Repository) operationForControl(ctx context.Context, expected *Operation) (Operation, error) {
	op, err := r.Operation(ctx)
	if err != nil {
		return op, err
	}
	if expected != nil && (expected.Token == "" || op != *expected) {
		return op, ErrOperationTargetChanged
	}
	return op, nil
}

func (r *Repository) SkipChecked(ctx context.Context, expected Operation) error {
	return r.skip(ctx, expected.Kind, &expected)
}

func (r *Repository) AbortChecked(ctx context.Context, expected Operation) error {
	return r.abort(ctx, expected.Kind, &expected)
}

func (r *Repository) Skip(ctx context.Context, kind string) error {
	return r.skip(ctx, kind, nil)
}

func (r *Repository) skip(ctx context.Context, kind string, expected *Operation) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	op, err := r.operationForControl(ctx, expected)
	if err != nil {
		return err
	}
	if op.Kind != kind || kind == "" || kind == "merge" {
		return errors.New("进行中的操作不支持跳过")
	}
	_, err = run(ctx, r.Root, "", "-c", "core.editor=true", kind, "--skip")
	return err
}

func (r *Repository) Abort(ctx context.Context, kind string) error {
	return r.abort(ctx, kind, nil)
}

func (r *Repository) abort(ctx context.Context, kind string, expected *Operation) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	op, err := r.operationForControl(ctx, expected)
	if err != nil {
		return err
	}
	if op.Kind != kind || kind == "" {
		return errors.New("进行中的操作已变化，请刷新")
	}
	_, err = run(ctx, r.Root, "", kind, "--abort")
	return err
}
