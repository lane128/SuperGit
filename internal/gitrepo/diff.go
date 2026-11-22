package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var hunkRE = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)
var oidRE = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

func parseDiff(text string) Diff {
	d := Diff{Text: text}
	chunks := strings.SplitAfter(text, "\n")
	var header strings.Builder
	index := -1
	old, next := 0, 0
	var patches []strings.Builder
	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}
		line := strings.TrimSuffix(chunk, "\n")
		if m := hunkRE.FindStringSubmatch(line); m != nil {
			old, _ = strconv.Atoi(m[1])
			next, _ = strconv.Atoi(m[2])
			index++
			d.Hunks = append(d.Hunks, Hunk{Header: line})
			patches = append(patches, strings.Builder{})
			patches[index].WriteString(chunk)
			d.Lines = append(d.Lines, DiffLine{Kind: "hunk", Text: line, Hunk: index})
			continue
		}
		if index < 0 {
			if strings.HasPrefix(line, "Binary files ") || line == "GIT binary patch" {
				d.Binary = true
			}
			header.WriteString(chunk)
			d.Lines = append(d.Lines, DiffLine{Kind: "meta", Text: line, Hunk: -1})
			continue
		}
		patches[index].WriteString(chunk)
		row := DiffLine{Text: line, Hunk: index}
		switch {
		case strings.HasPrefix(line, "+"):
			row.Kind = "add"
			row.New = next
			next++
		case strings.HasPrefix(line, "-"):
			row.Kind = "del"
			row.Old = old
			old++
		case strings.HasPrefix(line, " "):
			row.Kind = "context"
			row.Old = old
			row.New = next
			old++
			next++
		default:
			row.Kind = "meta"
		}
		d.Lines = append(d.Lines, row)
	}
	for i := range d.Hunks {
		d.Hunks[i].Patch = header.String() + patches[i].String()
	}
	d.CanStageHunks = len(d.Hunks) > 0
	// An existing symlink target diff carries its type only in the index line,
	// without a mode-change marker. Partial operations are for regular files.
	for _, line := range strings.Split(header.String(), "\n") {
		if !strings.HasPrefix(line, "index ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[2] != "100644" && fields[2] != "100755" {
			d.CanStageHunks = false
		}
	}
	for _, line := range strings.Split(header.String(), "\n") {
		if strings.HasPrefix(line, "new file mode ") && line != "new file mode 100644" && line != "new file mode 100755" {
			d.CanStageHunks = false
		}
	}
	for _, marker := range []string{"deleted file mode ", "rename from ", "copy from ", "Binary files ", "GIT binary patch", "old mode ", "new mode "} {
		if strings.Contains(header.String(), marker) {
			d.CanStageHunks = false
		}
	}
	// A submodule whose working tree is dirty but whose recorded object did
	// not move prints only the pointer with "-dirty" appended, and no index
	// line at all: git apply accepts that body and stages nothing. A regular
	// file always carries an index line (a modified one with mode 100644 or
	// 100755, which the check above already rejects for gitlinks), so the
	// missing index is what tells the two apart.
	if !strings.Contains(header.String(), "index ") {
		for _, line := range d.Lines {
			if (line.Kind == "add" || line.Kind == "del") && strings.Contains(line.Text, "Subproject commit ") {
				d.CanStageHunks = false
				break
			}
		}
	}
	return d
}
func (r *Repository) Diff(ctx context.Context, path string, staged bool) (Diff, error) {
	return r.DisplayDiff(ctx, path, staged, false)
}

func (r *Repository) DisplayDiff(ctx context.Context, path string, staged, ignoreWhitespace bool) (Diff, error) {
	if err := r.path(path); err != nil {
		return Diff{}, err
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--unified=3", "--submodule=short", "--src-prefix=a/", "--dst-prefix=b/"}
	if ignoreWhitespace {
		args = append(args, "--ignore-all-space")
	}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	out, err := run(ctx, r.Root, "", args...)
	if err != nil {
		return Diff{}, err
	}
	if out == "" && !staged {
		status, e := r.Status(ctx)
		if e != nil {
			return Diff{}, e
		}
		untracked := false
		for _, entry := range status.Entries {
			if entry.Path == path && entry.Index == '?' {
				untracked = true
				break
			}
		}
		if untracked {
			info, e := os.Lstat(filepath.Join(r.Root, path))
			if e != nil {
				return Diff{}, e
			}
			if info.IsDir() {
				return parseDiff("目录中有未跟踪文件，请在列表中选择具体文件。\n"), nil
			}
			if info.Size() > 1<<20 {
				return parseDiff("新文件超过 1 MiB，首版仅支持文件级暂存。\n"), nil
			}
			out, err = run(ctx, r.Root, "", "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--no-color", "--src-prefix=a/", "--dst-prefix=b/", "--", os.DevNull, path)
			var ee *exec.ExitError
			if err != nil && !(errors.As(err, &ee) && ee.ExitCode() == 1) {
				return Diff{}, err
			}
		}
	}
	diff := parseDiff(out)
	// A whitespace-ignoring patch can have different hunk boundaries and old context.
	// Only canonical patches may be used for partial index or worktree writes.
	if ignoreWhitespace && diff.CanStageHunks {
		canonical, readErr := r.Diff(ctx, path, staged)
		diff.CanStageHunks = readErr == nil && canonical.CanStageHunks && canonical.Text == diff.Text
	}
	return diff, nil
}

// FileDiff is one file's part of a commit's diff. A commit's details show
// every file at once, so they need the paths as well as the patches.
type FileDiff struct {
	Path string
	Diff Diff
}

// ShowAll returns the diff of every file the commit touched. One git command
// answers for the whole commit; splitting it per file here is cheaper than one
// `git show` per path and keeps the sections in the order git printed them.
func (r *Repository) ShowAll(ctx context.Context, sha string) ([]FileDiff, error) {
	return r.DisplayCommitDiff(ctx, sha, false)
}

func (r *Repository) DisplayCommitDiff(ctx context.Context, sha string, ignoreWhitespace bool) ([]FileDiff, error) {
	return r.readCommitDiff(ctx, sha, ignoreWhitespace, nil)
}

// ErrDiffPreviewLimit means a successful Git read exceeded the preview budget.
var ErrDiffPreviewLimit = errors.New("差异超过 4 MiB 预览上限")

// DisplayCommitFileDiff uses the same baseline and display options as the
// combined patch, but applies the preview budget to just the selected file.
func (r *Repository) DisplayCommitFileDiff(ctx context.Context, sha string, file ChangedFile, ignoreWhitespace bool) ([]FileDiff, error) {
	if err := r.path(file.Path); err != nil {
		return nil, err
	}
	if file.OriginalPath != "" {
		if err := r.path(file.OriginalPath); err != nil {
			return nil, err
		}
	}
	return r.readCommitDiff(ctx, sha, ignoreWhitespace, &file)
}

func (r *Repository) readCommitDiff(ctx context.Context, sha string, ignoreWhitespace bool, file *ChangedFile) ([]FileDiff, error) {
	if !oidRE.MatchString(sha) {
		return nil, fmt.Errorf("无效的提交 SHA")
	}
	parents, err := run(ctx, r.Root, "", "show", "-s", "--format=%P", sha)
	if err != nil {
		return nil, err
	}
	var args []string
	if p := strings.Fields(parents); len(p) > 0 {
		args = []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--submodule=short", p[0], sha}
	} else {
		args = []string{"show", "--format=", "--no-ext-diff", "--no-textconv", "--no-color", "--submodule=short", sha}
	}
	if ignoreWhitespace {
		args = append(args, "--ignore-all-space")
	}
	args = append(args, "--src-prefix=a/", "--dst-prefix=b/")
	if file != nil {
		// Literal pathspecs still match descendants. For a file/directory
		// replacement keep the other side's children out of the byte budget.
		switch {
		case file.Status == "A", file.Status == "D":
			args = append(args, "--diff-filter="+file.Status)
		case strings.HasPrefix(file.Status, "R"):
			args = append(args, "-M", "--diff-filter=R")
		case strings.HasPrefix(file.Status, "C"):
			args = append(args, "-C", "--find-copies-harder", "--diff-filter=C")
		}
		args = append(args, "--", file.Path)
		if file.OriginalPath != "" && file.OriginalPath != file.Path {
			args = append(args, file.OriginalPath)
		}
	}
	out, err := run(ctx, r.Root, "", args...)
	if errors.Is(err, errOutputTruncated) {
		return nil, ErrDiffPreviewLimit
	}
	if err != nil {
		return nil, err
	}
	diffs := splitFileDiffs(out)
	if file != nil {
		diffs = slices.DeleteFunc(diffs, func(d FileDiff) bool { return d.Path != file.Path })
	}
	return diffs, nil
}

// splitFileDiffs cuts a multi-file patch into one Diff per file. Each section
// starts at a `diff --git` line; the path is read from the `+++` line, or the
// `---` line for a deletion, falling back to the header for a patch whose body
// git did not print (a binary change).
func splitFileDiffs(text string) []FileDiff {
	var files []FileDiff
	var body strings.Builder
	newPath, oldPath, headerPath := "", "", ""
	flush := func() {
		if body.Len() == 0 {
			return
		}
		path := firstNonEmpty(newPath, oldPath, headerPath)
		files = append(files, FileDiff{Path: path, Diff: parseDiff(body.String())})
		body.Reset()
		newPath, oldPath, headerPath = "", "", ""
	}
	for _, chunk := range strings.SplitAfter(text, "\n") {
		if strings.HasPrefix(chunk, "diff --git ") {
			flush()
			headerPath = diffHeaderNewPath(strings.TrimSuffix(chunk, "\n"))
		}
		body.WriteString(chunk)
		line := strings.TrimSuffix(chunk, "\n")
		switch {
		case newPath == "" && strings.HasPrefix(line, "+++ "):
			newPath = diffPathName(strings.TrimPrefix(line, "+++ "))
		case oldPath == "" && strings.HasPrefix(line, "--- "):
			oldPath = diffPathName(strings.TrimPrefix(line, "--- "))
		}
	}
	flush()
	return files
}

// patchPath reads the repository path a single-file patch applies to. The
// `+++` line wins: git quotes a name that contains a quote, a backslash or a
// control character, and a name can itself contain " b/", so the header's last
// " b/" can end inside the new name. A deletion has only the `---` side, and a
// binary patch with no body falls back to the header. Lines after the first
// `@@` are hunk content, where a line of text can begin with "+++ " or "--- ".
func patchPath(patch string) string {
	newPath, oldPath, headerPath := "", "", ""
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			return firstNonEmpty(newPath, oldPath, headerPath)
		case strings.HasPrefix(line, "diff --git "):
			if headerPath == "" {
				headerPath = diffHeaderNewPath(line)
			}
		case newPath == "" && strings.HasPrefix(line, "+++ "):
			newPath = diffPathName(strings.TrimPrefix(line, "+++ "))
		case oldPath == "" && strings.HasPrefix(line, "--- "):
			oldPath = diffPathName(strings.TrimPrefix(line, "--- "))
		}
	}
	return firstNonEmpty(newPath, oldPath, headerPath)
}

// diffHeaderNewPath reads the b/ path out of `diff --git a/X b/Y`. A change
// names the same file on both sides, and an unquoted name can itself contain
// " b/", so a split whose halves match wins over the last " b/"; a rename
// names two files, where the last " b/" is the new one as before.
func diffHeaderNewPath(line string) string {
	rest := strings.TrimPrefix(line, "diff --git ")
	if strings.HasPrefix(rest, "\"") {
		// Git quotes both sides when either name needs it; splitting the
		// quoted text on " b/" could land inside a name.
		if newPath, ok := quotedHeaderNewPath(rest); ok {
			return diffPathName(newPath)
		}
		return ""
	}
	fallback := ""
	for i := 2; i+2 < len(rest); i++ {
		if rest[i:i+3] != " b/" {
			continue
		}
		oldPath, newPath := rest[:i], rest[i+1:]
		fallback = newPath
		if strings.HasPrefix(oldPath, "a/") && strings.HasPrefix(newPath, "b/") && oldPath[2:] == newPath[2:] {
			return diffPathName(newPath)
		}
	}
	return diffPathName(fallback)
}

// quotedHeaderNewPath reads the new name of a `diff --git` line whose names
// git quoted.
func quotedHeaderNewPath(rest string) (string, bool) {
	first, err := strconv.QuotedPrefix(rest)
	if err != nil {
		return "", false
	}
	rest = rest[len(first):]
	if !strings.HasPrefix(rest, " ") {
		return "", false
	}
	second, err := strconv.QuotedPrefix(rest[1:])
	if err != nil {
		return "", false
	}
	if _, err := strconv.Unquote(second); err != nil {
		return "", false
	}
	return second, true
}

// diffPathName turns a patch's a/ or b/ path into the repository path,
// undoing git's C quoting and dropping /dev/null. A tab ends the path: git
// adds one when the name itself ends in whitespace, or as the separator
// before a timestamp, so only the tab and a CR are trimmed; spaces belong to
// the name.
func diffPathName(raw string) string {
	s := raw
	if strings.HasPrefix(s, "\"") {
		unquoted, err := strconv.Unquote(s)
		if err != nil {
			return ""
		}
		s = unquoted
	} else {
		if i := strings.IndexByte(s, '\t'); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSuffix(s, "\r")
	}
	if s == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(s, "a/") || strings.HasPrefix(s, "b/") {
		return s[2:]
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (r *Repository) Show(ctx context.Context, sha, path string) (Diff, error) {
	if !oidRE.MatchString(sha) {
		return Diff{}, fmt.Errorf("无效的提交 SHA")
	}
	if err := r.path(path); err != nil {
		return Diff{}, err
	}
	parents, err := run(ctx, r.Root, "", "show", "-s", "--format=%P", sha)
	if err != nil {
		return Diff{}, err
	}
	p := strings.Fields(parents)
	var args []string
	if len(p) > 0 {
		args = []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--submodule=short", p[0], sha, "--", path}
	} else {
		args = []string{"show", "--format=", "--no-ext-diff", "--no-textconv", "--no-color", "--submodule=short", sha, "--", path}
	}
	out, err := run(ctx, r.Root, "", args...)
	return parseDiff(out), err
}
