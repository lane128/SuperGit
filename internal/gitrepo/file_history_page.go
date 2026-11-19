package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FileHistoryPage follows one raw path from a pinned HEAD. Nil tips starts a
// traversal; a non-nil empty slice pins an unborn repository. Each request
// replays rename following from that same tip before skipping prior results.
// This bounds output size, but not Git's traversal cost as the offset grows.
func (r *Repository) FileHistoryPage(ctx context.Context, path string, tips []string, offset, limit int) (HistoryPage, error) {
	if err := r.path(path); err != nil {
		return HistoryPage{}, err
	}
	if offset < 0 || limit < 1 || limit > 2000 || tips == nil && offset != 0 || len(tips) > 1 {
		return HistoryPage{}, fmt.Errorf("无效的文件历史分页范围")
	}
	if err := ctx.Err(); err != nil {
		return HistoryPage{}, err
	}
	if tips == nil {
		sha, err := r.ResolveRef(ctx, "HEAD")
		if err != nil {
			if ctx.Err() != nil {
				return HistoryPage{}, ctx.Err()
			}
			// Only a valid unborn branch is empty. A missing repository or a broken
			// detached HEAD is a read failure, not a completed empty traversal.
			ref, refErr := run(ctx, r.Root, "", "symbolic-ref", "--quiet", "HEAD")
			if refErr != nil {
				return HistoryPage{}, err
			}
			_, refErr = run(ctx, r.Root, "", "show-ref", "--verify", strings.TrimSpace(ref))
			if refErr == nil {
				return HistoryPage{}, err
			}
			// rev-parse still verifies that the repository is readable.
			if _, probeErr := run(ctx, r.Root, "", "rev-parse", "--git-dir"); probeErr != nil {
				return HistoryPage{}, probeErr
			}
			tips = []string{}
		} else {
			tips = []string{sha}
		}
	}
	page := HistoryPage{Tips: append([]string{}, tips...)}
	if len(tips) == 0 {
		return page, nil
	}
	if !oidRE.MatchString(tips[0]) {
		return HistoryPage{}, fmt.Errorf("无效的文件历史起始 SHA")
	}
	if offset > int(^uint(0)>>1)-limit-1 || offset > int(^uint(0)>>1)/historyFieldCount {
		return HistoryPage{}, fmt.Errorf("无效的文件历史分页范围")
	}
	output := fileHistoryPageOutput{skipFields: offset * historyFieldCount, page: limitedBuffer{limit: outputLimit}}
	// Git --skip omits rename processing for skipped records. Replay them and
	// discard their NUL-terminated record fields without retaining their bytes.
	err := runOutput(ctx, r.Root, "", &output, "log", "--follow", "-z", historyFormat, "-n", strconv.FormatInt(int64(offset+limit+1), 10), tips[0], "--", path)
	if err != nil {
		return HistoryPage{}, err
	}
	if output.page.truncated {
		return HistoryPage{}, errors.New("文件历史页超过 4 MiB 显示上限")
	}
	page.Commits, err = parseHistory(output.page.String())
	if err != nil {
		return HistoryPage{}, err
	}
	page.HasMore = len(page.Commits) > limit
	if page.HasMore {
		page.Commits = page.Commits[:limit]
	}
	return page, nil
}

// fileHistoryPageOutput never buffers skipped records, even if one field spans
// multiple pipe writes. The requested page retains the ordinary output cap.
type fileHistoryPageOutput struct {
	skipFields int
	page       limitedBuffer
}

func (w *fileHistoryPageOutput) Write(p []byte) (int, error) {
	n := len(p)
	for w.skipFields > 0 {
		i := bytes.IndexByte(p, 0)
		if i < 0 {
			return n, nil
		}
		w.skipFields--
		p = p[i+1:]
	}
	_, err := w.page.Write(p)
	return n, err
}
