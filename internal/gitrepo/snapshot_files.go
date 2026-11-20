package gitrepo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type SnapshotFile struct {
	Path, Kind, OID string
	Size            int64
}

// SnapshotFiles lists the complete immutable tree, including Git links and
// entries unchanged in this commit. NUL framing preserves raw pathnames.
func (r *Repository) SnapshotFiles(ctx context.Context, commit string) ([]SnapshotFile, error) {
	return r.snapshotEntries(ctx, commit, true)
}

// SnapshotDirectory reads a single directory without enumerating its descendants.
func (r *Repository) SnapshotDirectory(ctx context.Context, oid string) ([]SnapshotFile, error) {
	return r.snapshotEntries(ctx, oid, false)
}

func (r *Repository) snapshotEntries(ctx context.Context, commit string, recursive bool) ([]SnapshotFile, error) {
	if !oidRE.MatchString(commit) {
		return nil, fmt.Errorf("invalid commit ID %q", commit)
	}
	args := []string{"ls-tree", "-l", "-z"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, commit, "--")
	out, err := run(ctx, r.Root, "", args...)
	if err != nil {
		return nil, err
	}
	var files []SnapshotFile
	for _, record := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		if record == "" {
			continue
		}
		info, path, ok := strings.Cut(record, "\t")
		fields := strings.Fields(info)
		if !ok || len(fields) != 4 || !oidRE.MatchString(fields[2]) {
			return nil, fmt.Errorf("invalid ls-tree output")
		}
		size := int64(0)
		if fields[3] != "-" {
			size, err = strconv.ParseInt(fields[3], 10, 64)
			if err != nil {
				return nil, err
			}
		}
		files = append(files, SnapshotFile{Path: path, Kind: fields[1], OID: fields[2], Size: size})
	}
	return files, nil
}

func (r *Repository) SnapshotBlob(ctx context.Context, oid string) (string, error) {
	if !oidRE.MatchString(oid) {
		return "", fmt.Errorf("invalid blob ID %q", oid)
	}
	return run(ctx, r.Root, "", "cat-file", "blob", oid)
}
