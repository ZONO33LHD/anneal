package model

import "testing"

func TestCleanVersion(t *testing.T) {
	for in, want := range map[string]string{"^1.6.2": "1.6.2", "~v2.0.0": "2.0.0", "1.2.3-beta.1": "1.2.3"} {
		if got := CleanVersion(in); got != want {
			t.Errorf("CleanVersion(%q)=%q want %q", in, got, want)
		}
	}
}

func TestClassifyUpdate(t *testing.T) {
	cases := []struct {
		cur, tgt string
		want     UpdateType
	}{
		{"1.6.2", "1.6.3", Patch},
		{"1.6.2", "1.7.0", Minor},
		{"4.1.2", "5.3.0", Major},
	}
	for _, c := range cases {
		if got := ClassifyUpdate(c.cur, c.tgt); got != c.want {
			t.Errorf("ClassifyUpdate(%s,%s)=%s want %s", c.cur, c.tgt, got, c.want)
		}
	}
}

func TestCompareAndUpgrade(t *testing.T) {
	if CompareVersions("1.0.0", "1.0.1") != -1 {
		t.Error("1.0.0 < 1.0.1")
	}
	if CompareVersions("2.0.0", "1.9.9") != 1 {
		t.Error("2.0.0 > 1.9.9")
	}
	if !IsUpgrade("1.6.2", "1.7.0") {
		t.Error("expected upgrade")
	}
	if IsUpgrade("1.7.0", "1.7.0") {
		t.Error("same version is not an upgrade")
	}
}
