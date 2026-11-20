package gitrepo

import (
	"context"
	"errors"
	"strings"
)

// Both ends come from displayed commits (or peeled commit tags). A full
// object ID fixes each end even if HEAD or a tag moves between reads.
func comparisonCommits(base, target string) ([]string, error) {
	for _, id := range []string{base, target} {
		if (len(id) != 40 && len(id) != 64) || !oidRE.MatchString(id) {
			return nil, errors.New("invalid comparison commit ID")
		}
	}
	return []string{base + "^{commit}", target + "^{commit}", "--"}, nil
}

func (r *Repository) CompareRevisionFiles(ctx context.Context, base, target string) ([]ChangedFile, error) {
	ends, err := comparisonCommits(base, target)
	if err != nil {
		return nil, err
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", "--name-status", "-z", "-M"}
	out, err := run(ctx, r.Root, "", append(args, ends...)...)
	if err != nil {
		return nil, err
	}
	return parseChangedFiles(out)
}

// CompareRevisionFile reads a patch lazily so one oversized file cannot
// prevent browsing the other changed paths. Both rename paths are literal.
func (r *Repository) CompareRevisionFile(ctx context.Context, base, target string, file ChangedFile) (Diff, error) {
	ends, err := comparisonCommits(base, target)
	if err != nil {
		return Diff{}, err
	}
	if err := r.path(file.Path); err != nil {
		return Diff{}, err
	}
	paths := []string{file.Path}
	if file.OriginalPath != "" {
		if err := r.path(file.OriginalPath); err != nil {
			return Diff{}, err
		}
		paths = append(paths, file.OriginalPath)
	}
	args := []string{"diff", "--patch", "--no-ext-diff", "--no-textconv", "--no-color", "--submodule=short", "--ignore-submodules=none", "-M"}
	// A rename can vacate a name reused by an unrelated addition. A former
	// file can also be a directory on the other side of the comparison.
	if file.Status != "" && strings.Contains("ACDMRT", file.Status[:1]) {
		args = append(args, "--diff-filter="+file.Status[:1])
	}
	args = append(args, ends...)
	out, err := run(ctx, r.Root, "", append(args, paths...)...)
	if err != nil {
		return Diff{}, err
	}
	return parseDiff(out), nil
}
