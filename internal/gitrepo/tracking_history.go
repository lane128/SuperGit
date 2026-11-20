package gitrepo

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

var ErrTrackingChanged = errors.New("branch or upstream has changed")

// TrackingTip pins the configured upstream of the displayed checkout. The
// fully qualified ref from Git also handles local and custom tracking refs.
func (r *Repository) TrackingTip(ctx context.Context, expected Status) (string, error) {
	if expected.Branch == "" || expected.Upstream == "" || !oidRE.MatchString(expected.Head) {
		return "", ErrTrackingChanged
	}
	ref := "refs/heads/" + expected.Branch
	out, err := run(ctx, r.Root, "", "for-each-ref", "--count=1", "--format=%(HEAD)%00%(refname)%00%(objectname)%00%(upstream:short)%00%(upstream)", "--", ref)
	if err != nil {
		return "", err
	}
	fields := strings.Split(strings.TrimSuffix(out, "\n"), "\x00")
	if len(fields) != 5 || fields[0] != "*" || fields[1] != ref || fields[2] != expected.Head || fields[3] != expected.Upstream || fields[4] == "" {
		return "", ErrTrackingChanged
	}
	out, err = run(ctx, r.Root, "", "show-ref", "--verify", "--hash", "--", fields[4])
	if err != nil {
		return "", errors.Join(ErrTrackingChanged, err)
	}
	return strings.TrimSpace(out), nil
}

// TrackingCommits walks only the included side of a pinned comparison. One
// extra row determines whether the menu can page forward.
func (r *Repository) TrackingCommits(ctx context.Context, include, exclude string, offset, limit int) ([]Commit, bool, error) {
	if !oidRE.MatchString(include) || !oidRE.MatchString(exclude) || offset < 0 || limit < 1 || limit > 2000 {
		return nil, false, errors.New("invalid tracking range")
	}
	out, err := run(ctx, r.Root, "", "--no-lazy-fetch", "log", "-z", "--topo-order", "--no-decorate", historyFormat,
		"--skip="+strconv.Itoa(offset), "-n", strconv.Itoa(limit+1), exclude+".."+include, "--")
	if err != nil {
		return nil, false, err
	}
	commits, err := parseHistory(out)
	if len(commits) > limit {
		return commits[:limit], true, err
	}
	return commits, false, err
}
