package model

import (
	"regexp"
	"strconv"
	"strings"
)

// UpdateType はバージョン変化の大きさを分類する。
type UpdateType string

const (
	Patch UpdateType = "patch"
	Minor UpdateType = "minor"
	Major UpdateType = "major"
)

// SemVer はパース済みの major.minor.patch の三つ組である。
type SemVer struct {
	Major, Minor, Patch int
}

var (
	rangePrefix = regexp.MustCompile(`^[\^~>=<v\s]+`)
	verCore     = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
)

// CleanVersion は範囲演算子と先頭の v を取り除き、素のバージョンを返す。
func CleanVersion(raw string) string {
	s := rangePrefix.ReplaceAllString(strings.TrimSpace(raw), "")
	s = strings.SplitN(s, "-", 2)[0]
	s = strings.SplitN(s, "+", 2)[0]
	return s
}

// ParseSemVer はバージョン文字列をパースし、パースできない場合は ok=false を返す。
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

// CompareVersions は a<b/a==b/a>b に対して -1/0/1 を返す。パース不能なものは等しいとみなす。
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

// ClassifyUpdate は current から target への変化を分類する。
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

// IsUpgrade は target が current より厳密に新しいかどうかを返す。
func IsUpgrade(current, target string) bool {
	return CompareVersions(target, current) > 0
}
