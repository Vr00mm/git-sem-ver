// Package semver provides minimal semantic versioning (semver.org) support.
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version (major.minor.patch[-pre-release]).
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
}

// Zero is the implicit base version when no tag exists yet.
var Zero = Version{}

// Parse parses a semver string, accepting an optional leading "v".
// Pre-release and build metadata suffixes are stripped.
func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	// strip pre-release / build metadata.
	s = strings.SplitN(s, "-", 2)[0]
	s = strings.SplitN(s, "+", 2)[0]

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semver %q: expected major.minor.patch", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major in %q: %w", s, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor in %q: %w", s, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch in %q: %w", s, err)
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// BumpMajor increments the major component and resets minor and patch to 0.
func (v Version) BumpMajor() Version {
	return Version{Major: v.Major + 1}
}

// BumpMinor increments the minor component and resets patch to 0.
func (v Version) BumpMinor() Version {
	return Version{Major: v.Major, Minor: v.Minor + 1}
}

// BumpPatch increments the patch component.
func (v Version) BumpPatch() Version {
	return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
}

// WithPreRelease returns a copy of v with the given pre-release string.
func (v Version) WithPreRelease(pre string) Version {
	v.PreRelease = pre
	return v
}

// String returns the semver string representation.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	return s
}
