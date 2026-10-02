package mcversion

import "testing"

func TestCompareAcrossVersionSchemes(t *testing.T) {
	ordered := []string{"1.20.1", "1.20.4", "1.21", "1.21.1", "1.21.11", "26.1", "26.1.2", "26.3"}
	for i := 1; i < len(ordered); i++ {
		lo, hi := MustParse(ordered[i-1]), MustParse(ordered[i])
		if lo.Compare(hi) != -1 || hi.Compare(lo) != 1 {
			t.Errorf("se esperaba %s < %s", lo, hi)
		}
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, s := range []string{"", "1", "1.x", "1.2.3.4", "-1.2"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) debería fallar", s)
		}
	}
}

func TestRangeContains(t *testing.T) {
	r := MustParseRange(">=1.20.1 <1.21")
	cases := map[string]bool{"1.20.1": true, "1.20.6": true, "1.21": false, "1.19.4": false, "26.1": false}
	for s, want := range cases {
		if got := r.Contains(MustParse(s)); got != want {
			t.Errorf("%s en %s = %v, se esperaba %v", s, r, got, want)
		}
	}
	if !(Range{}).Contains(MustParse("26.3")) {
		t.Error("el rango vacío debe aceptar cualquier versión")
	}
}
