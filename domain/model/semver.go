package model

import (
	"regexp"
	"strconv"
	"strings"
)

// UpdateType classifies the size of a version jump.
type UpdateType string

const (
	Patch UpdateType = "patch"
	Minor UpdateType = "minor"
	Major UpdateType = "major"
)

// SemVer is a parsed major.minor.patch triple.
type SemVer struct {
	Major, Minor, Patch int
}

var (
	rangePrefix = regexp.MustCompile(`^[\^~>=<v\s]+`)
	verCore     = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
)

// CleanVersion strips range operators and a leading v, returning a bare version.
func CleanVersion(raw string) string {
	s := rangePrefix.ReplaceAllString(strings.TrimSpace(raw), "")
	s = strings.SplitN(s, "-", 2)[0]
	s = strings.SplitN(s, "+", 2)[0]
	return s
}

// ParseSemVer parses a version string, returning ok=false if unparseable.
func ParseSemVer(raw string) (SemVer, bool) {
	m := verCore.FindStringSubmatch(CleanVersion(raw))
	if m == nil {
		return SemVer{}, false
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return SemVer{maj, min, pat}, true
}

// CompareVersions returns -1/0/1 for a<b/a==b/a>b. Unparseable compare equal.
func CompareVersions(a, b string) int {
	pa, oka := ParseSemVer(a)
	pb, okb := ParseSemVer(b)
	if !oka || !okb {
		return 0
	}
	switch {
	case pa.Major != pb.Major:
		return sign(pa.Major - pb.Major)
	case pa.Minor != pb.Minor:
		return sign(pa.Minor - pb.Minor)
	default:
		return sign(pa.Patch - pb.Patch)
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// ClassifyUpdate classifies the jump from current to target.
func ClassifyUpdate(current, target string) UpdateType {
	c, okc := ParseSemVer(current)
	t, okt := ParseSemVer(target)
	if !okc || !okt {
		return Patch
	}
	switch {
	case t.Major > c.Major:
		return Major
	case t.Major == c.Major && t.Minor > c.Minor:
		return Minor
	default:
		return Patch
	}
}

// IsUpgrade reports whether target is strictly newer than current.
func IsUpgrade(current, target string) bool {
	return CompareVersions(target, current) > 0
}
