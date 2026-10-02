package site

import "testing"

func TestFine(t *testing.T) {
	for p, want := range map[float64]string{0.25: "25 %", 0.00083: "0,083 % · 1 de cada 1.205", 0.0000412: "0,0041 % · 1 de cada 24.272"} {
		if got := fine(p); got != want {
			t.Errorf("fine(%v) = %q; quiero %q", p, got, want)
		}
	}
}
