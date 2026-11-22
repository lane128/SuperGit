package updater

import (
	"fmt"
	"strconv"
	"strings"
)

// Version holds a parsed semantic version (major.minor.patch) with an optional
// pre-release tag. Build numbers are tracked separately because local builds
// share the same semantic version as a release tag.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ParseVersion parses strings like "0.1.0", "v1.2.3", or "v1.2.3-rc.1".
func ParseVersion(raw string) (Version, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	if s == "" {
		return Version{}, fmt.Errorf("empty version")
	}
	if plus := strings.IndexByte(s, '+'); plus >= 0 {
		s = s[:plus]
	}
	var pre string
	if dash := strings.IndexByte(s, '-'); dash >= 0 {
		pre = strings.TrimSpace(s[dash+1:])
		s = s[:dash]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	nums := [3]int{}
	for i, part := range parts {
		if part == "" {
			return Version{}, fmt.Errorf("invalid version %q", raw)
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("invalid version %q", raw)
		}
		nums[i] = n
	}
	return Version{
		Major:      nums[0],
		Minor:      nums[1],
		Patch:      nums[2],
		Prerelease: pre,
	}, nil
}

// String formats the canonical version without a leading "v".
func (v Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		return base + "-" + v.Prerelease
	}
	return base
}

// Compare returns -1 if v < other, 0 if equal, or +1 if v > other following
// SemVer precedence (a stable release is newer than a pre-release of the same
// major.minor.patch).
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	if v.Prerelease == other.Prerelease {
		return 0
	}
	if v.Prerelease == "" {
		return 1
	}
	if other.Prerelease == "" {
		return -1
	}
	return comparePrerelease(v.Prerelease, other.Prerelease)
}

func comparePrerelease(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	limit := min(len(ap), len(bp))
	for i := range limit {
		if ap[i] == bp[i] {
			continue
		}
		an, aErr := strconv.Atoi(ap[i])
		bn, bErr := strconv.Atoi(bp[i])
		switch {
		case aErr == nil && bErr == nil:
			if an < bn {
				return -1
			}
			return 1
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		default:
			if ap[i] < bp[i] {
				return -1
			}
			return 1
		}
	}
	if len(ap) < len(bp) {
		return -1
	}
	if len(ap) > len(bp) {
		return 1
	}
	return 0
}

// NormalizeVersion strips a leading "v" and returns the canonical form when valid.
func NormalizeVersion(raw string) string {
	v, err := ParseVersion(raw)
	if err != nil {
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "v"))
	}
	return v.String()
}

// IsNewer reports whether candidateVersion is strictly newer than currentVersion.
// Equal semantic versions return false so local builds on the same release line
// do not repeatedly prompt to update to themselves.
func IsNewer(candidateVersion, currentVersion string) (bool, error) {
	cand, err := ParseVersion(candidateVersion)
	if err != nil {
		return false, fmt.Errorf("candidate version: %w", err)
	}
	curr, err := ParseVersion(currentVersion)
	if err != nil {
		return false, fmt.Errorf("current version: %w", err)
	}
	return cand.Compare(curr) > 0, nil
}
