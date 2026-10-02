package disablers

import (
	"strings"
	"testing"
)

func TestGoverningKey(t *testing.T) {
	cases := []struct {
		text string
		line int
		want string
	}{
		{"disabled = [\"a:b\"]", 0, "disabled"},
		{"# no spawn\nblacklist = [\n  \"a:b\",\n]", 2, "# no spawn\nblacklist"},
		{"{\n  \"excluded_structures\": [\n    \"a:b\"\n  ],\n  \"other\": \"a:c\"\n}", 2, "excluded_structures"},
		{"{\n  \"excluded\": [\n    \"x\"\n  ],\n  \"a:c\"\n}", 4, ""},
		{"structures:\n  disabled:\n    - a:b", 2, "disabled"},
		{"[a]\nfavourite = \"a:b\"", 1, "favourite"},
	}
	for _, c := range cases {
		lines := strings.Split(c.text, "\n")
		if got := GoverningKey(lines, c.line); got != c.want {
			t.Errorf("GoverningKey(%q, %d) = %q, se esperaba %q", c.text, c.line, got, c.want)
		}
	}
}

func TestDisableWords(t *testing.T) {
	for _, s := range []string{"disabledStructures", "structure_blacklist", "excludeList", "REMOVE", "bans"} {
		if !disableWords.MatchString(s) {
			t.Errorf("%q debería coincidir", s)
		}
	}
	for _, s := range []string{"banner", "favourite", "spawn_weight"} {
		if disableWords.MatchString(s) {
			t.Errorf("%q no debería coincidir", s)
		}
	}
}
