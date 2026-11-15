package gitrepo

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// CommitMessage reads only immutable metadata, independent of file Diff. Format
// framing preserves the original trailing newline instead of trimming message
// whitespace. Rendering remains plain UTF-8 text, with an explicit size bound.
func (r *Repository) CommitMessage(ctx context.Context, sha string) (string, error) {
	if !oidRE.MatchString(sha) {
		return "", fmt.Errorf("无效的提交 SHA")
	}
	out, err := run(ctx, r.Root, "", "show", "--no-patch", "--no-show-signature", "--encoding=UTF-8", "--format=format:%B%x00", sha, "--")
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(out, "\x00") {
		return "", fmt.Errorf("无效的提交说明记录")
	}
	message := strings.TrimSuffix(out, "\x00")
	if len(message) > 1<<20 {
		return "", fmt.Errorf("提交说明超过 1 MiB 显示上限")
	}
	if strings.ContainsRune(message, '\x00') || !utf8.ValidString(message) {
		return "", fmt.Errorf("提交说明不是有效的 UTF-8 文本")
	}
	return message, nil
}
