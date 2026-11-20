package gitrepo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type LFSPattern struct {
	File, Pattern string
	line          int
	hash          [32]byte
}

func attributeRule(line string) (string, string, bool) {
	s := strings.TrimLeft(line, " \t")
	if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "[attr]") {
		return "", "", false
	}
	var token, pattern string
	if strings.HasPrefix(s, "\"") {
		var err error
		token, err = strconv.QuotedPrefix(s)
		if err != nil {
			return "", "", false
		}
		pattern, err = strconv.Unquote(token)
		if err != nil {
			return "", "", false
		}
	} else {
		end := strings.IndexAny(s, " \t\r\n")
		if end < 0 {
			return "", "", false
		}
		token, pattern = s[:end], s[:end]
	}
	tail := s[len(token):]
	isLFS := slices.Contains(strings.Fields(tail), "filter=lfs")
	return pattern, tail, isLFS
}

// Do not follow a version-controlled symlink to read or update attributes.
// Git itself ignores symlinked .gitattributes files in the working tree.
func readAttributes(parent *os.Root) (string, os.FileMode, error) {
	info, err := parent.Lstat(".gitattributes")
	if errors.Is(err, os.ErrNotExist) {
		return "", 0644, nil
	}
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, errors.New("The attributes file must be a regular file")
	}
	f, err := parent.Open(".gitattributes")
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	buf, err := io.ReadAll(io.LimitReader(f, outputLimit+1))
	if len(buf) > outputLimit {
		return "", 0, errors.New("The attributes file exceeds 4 MiB")
	}
	return string(buf), info.Mode().Perm(), err
}

func (r *Repository) LFSPatterns(ctx context.Context) ([]LFSPattern, error) {
	out, err := run(ctx, r.Root, "", "--no-literal-pathspecs", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--deduplicate", "--", ":(glob)**/.gitattributes")
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	// The root row always exists so a new repository can add its first rule.
	files := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	if !slices.Contains(files, ".gitattributes") {
		files = append(files, ".gitattributes")
	}
	var result []LFSPattern
	for _, file := range files {
		if file == "" {
			continue
		}
		parent, err := root.OpenRoot(filepath.Dir(file))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		info, statErr := parent.Lstat(".gitattributes")
		if statErr == nil && !info.Mode().IsRegular() {
			parent.Close()
			continue
		}
		data, _, err := readAttributes(parent)
		parent.Close()
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256([]byte(data))
		if file == ".gitattributes" {
			result = append(result, LFSPattern{File: file, line: -1, hash: hash})
		}
		for i, line := range strings.Split(data, "\n") {
			if p, _, ok := attributeRule(line); ok {
				result = append(result, LFSPattern{File: file, Pattern: p, line: i, hash: hash})
			}
		}
	}
	return result, nil
}

// Edit the original rule in one replacement. This preserves other attributes,
// lockability, comments and line endings, including in nested directories.
func (r *Repository) EditLFSPattern(ctx context.Context, expected LFSPattern, next string, status Status) error {
	if !filepath.IsLocal(expected.File) || filepath.Base(expected.File) != ".gitattributes" || strings.ContainsAny(next, "\n\r\x00") || strings.HasPrefix(next, "!") {
		return errors.New("Invalid Git LFS pattern")
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkPaletteStatus(ctx, status); err != nil {
		return err
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	parent, err := root.OpenRoot(filepath.Dir(expected.File))
	if err != nil {
		return err
	}
	defer parent.Close()
	data, mode, err := readAttributes(parent)
	if err != nil {
		return err
	}
	if sha256.Sum256([]byte(data)) != expected.hash {
		return ErrIntegrationTargetChanged
	}
	token := next
	if strings.ContainsAny(next, " \t\"\\") || strings.HasPrefix(next, "#") {
		token = strconv.Quote(next)
	}
	var updated string
	if expected.line < 0 {
		if next == "" {
			return errors.New("Enter an LFS pattern")
		}
		eol := "\n"
		if strings.Contains(data, "\r\n") {
			eol = "\r\n"
		}
		if data != "" && !strings.HasSuffix(data, "\n") {
			data += eol
		}
		updated = data + token + " filter=lfs diff=lfs merge=lfs -text" + eol
	} else {
		lines := strings.SplitAfter(data, "\n")
		if expected.line >= len(lines) {
			return ErrIntegrationTargetChanged
		}
		line := lines[expected.line]
		old, tail, ok := attributeRule(line)
		if !ok || old != expected.Pattern {
			return ErrIntegrationTargetChanged
		}
		lines[expected.line] = ""
		if next != "" {
			lines[expected.line] = line[:len(line)-len(strings.TrimLeft(line, " \t"))] + token + tail
		}
		updated = strings.Join(lines, "")
	}
	tmpName := ".supergit-attributes-" + rand.Text()
	tmp, err := parent.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer parent.Remove(tmpName)
	_, writeErr := tmp.WriteString(updated)
	if err := errors.Join(writeErr, tmp.Chmod(mode), tmp.Close()); err != nil {
		return err
	}
	current, _, err := readAttributes(parent)
	if err != nil {
		return err
	}
	if sha256.Sum256([]byte(current)) != expected.hash {
		return ErrIntegrationTargetChanged
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return parent.Rename(tmpName, ".gitattributes")
}

type LFSLockFile struct{ Path, ID string }
type LFSLockList struct {
	Files    []LFSLockFile
	endpoint [32]byte
}

func (r *Repository) lfsEndpoint(ctx context.Context, remote Remote) ([32]byte, error) {
	if err := r.checkRemoteTarget(ctx, remote); err != nil {
		return [32]byte{}, err
	}
	if err := r.checkRemote(ctx, remote.Name, true); err != nil {
		return [32]byte{}, err
	}
	out, err := lfsBinary().run(ctx, r.Root, "", "lfs", "env")
	if err != nil {
		return [32]byte{}, err
	}
	var endpoints strings.Builder
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, "Endpoint") {
			endpoints.WriteString(line)
			endpoints.WriteByte('\n')
		}
	}
	if endpoints.Len() == 0 {
		return [32]byte{}, errors.New("Git LFS has no configured endpoint")
	}
	return sha256.Sum256([]byte(endpoints.String())), nil
}

func (r *Repository) LFSLockFiles(ctx context.Context, remote Remote, unlock bool) (LFSLockList, error) {
	endpoint, err := r.lfsEndpoint(ctx, remote)
	if err != nil {
		return LFSLockList{}, err
	}
	return r.lfsLockFilesAt(ctx, remote, unlock, endpoint)
}

func (r *Repository) lfsLockFilesAt(ctx context.Context, remote Remote, unlock bool, endpoint [32]byte) (LFSLockList, error) {
	list := LFSLockList{endpoint: endpoint}
	if unlock {
		out, err := lfsBinary().run(ctx, r.Root, "", "lfs", "locks", "--verify", "--json", "--remote", remote.Name)
		if err != nil {
			return list, err
		}
		var locks struct{ Ours []LFSLockFile }
		if err := json.Unmarshal([]byte(out), &locks); err != nil {
			return list, err
		}
		for _, f := range locks.Ours {
			if filepath.IsLocal(f.Path) && f.ID != "" {
				list.Files = append(list.Files, f)
			}
		}
		return list, nil
	}
	// check-attr includes files newly tracked by the index and paths marked
	// lockable even if their contents aren't stored in LFS.
	paths, err := run(ctx, r.Root, "", "ls-files", "--cached", "--deduplicate", "-z")
	if err != nil || paths == "" {
		return list, err
	}
	out, err := run(ctx, r.Root, paths, "check-attr", "-z", "--stdin", "filter", "lockable")
	if err != nil {
		return list, err
	}
	fields := strings.Split(out, "\x00")
	for i := 0; i+5 < len(fields); i += 6 {
		if fields[i+2] == "lfs" || fields[i+5] == "set" {
			list.Files = append(list.Files, LFSLockFile{Path: fields[i]})
		}
	}
	return list, nil
}

func (r *Repository) ChangeLFSLock(ctx context.Context, remote Remote, list LFSLockList, file LFSLockFile, status Status) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkPaletteStatus(ctx, status); err != nil {
		return err
	}
	if !filepath.IsLocal(file.Path) || !slices.Contains(list.Files, file) {
		return ErrIntegrationTargetChanged
	}
	endpoint, err := r.lfsEndpoint(ctx, remote)
	if err != nil {
		return err
	}
	if endpoint != list.endpoint {
		return ErrRemoteTargetChanged
	}
	current, err := r.lfsLockFilesAt(ctx, remote, file.ID != "", endpoint)
	if err != nil {
		return err
	}
	if !slices.Contains(current.Files, file) {
		return ErrRemoteTargetChanged
	}
	// LFS may change local permissions after a successful server operation.
	// Refuse paths traversing links, including a link at the locked filename.
	if info, err := os.Lstat(filepath.Join(r.Root, file.Path)); err == nil {
		full := filepath.Join(canonical(r.Root), file.Path)
		if !info.Mode().IsRegular() || canonical(full) != full {
			return errors.New("LFS requires a regular working tree file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := r.checkPaletteStatus(ctx, status); err != nil {
		return err
	}
	if err := r.checkRemoteTarget(ctx, remote); err != nil {
		return err
	}

	if file.ID != "" {
		_, err = lfsBinary().run(ctx, r.Root, "", "lfs", "unlock", "--remote", remote.Name, "--id", file.ID)
	} else {
		_, err = lfsBinary().run(ctx, r.Root, "", "lfs", "lock", "--remote", remote.Name, "--", file.Path)
	}
	return err
}
