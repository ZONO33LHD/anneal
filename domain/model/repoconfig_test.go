package model

import "testing"

func TestIsIgnored(t *testing.T) {
	c := RepoConfig{Ignore: []string{"react", "eslint-*"}}
	cases := map[string]bool{
		"react":           true,  // 完全一致
		"reactive-lib":    false, // "react" は前方一致しない
		"eslint-plugin-x": true,  // "eslint-*" の前方一致
		"eslintx":         false, // "eslint-" で始まらない
		"axios":           false,
	}
	for pkg, want := range cases {
		if got := c.IsIgnored(pkg); got != want {
			t.Errorf("IsIgnored(%q)=%v want %v", pkg, got, want)
		}
	}
}
