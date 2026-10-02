package loot

import (
	"encoding/json"
	"math"
	"strings"
)

// Number is a resolved number provider: its range, mean and, for integer
// providers used as rolls, its distribution.
type Number struct {
	Min, Max, Mean float64
	// dist maps an integer outcome to its probability. Nil when unknown.
	dist map[int]float64
	// Approximate is true when the provider type was not understood.
	Approximate bool
}

func constant(v float64) Number {
	return Number{Min: v, Max: v, Mean: v, dist: map[int]float64{int(math.Round(v)): 1}}
}

// parseNumber reads a number provider: a plain number, {"min","max"},
// {"type":"uniform"|"constant"|"binomial",…}. Missing input yields def.
func parseNumber(raw json.RawMessage, def float64) Number {
	if len(raw) == 0 || string(raw) == "null" {
		return constant(def)
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return constant(f)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return Number{Min: def, Max: def, Mean: def, Approximate: true}
	}
	typ := ""
	if t, ok := obj["type"]; ok {
		_ = json.Unmarshal(t, &typ)
	}
	typ = strings.TrimPrefix(typ, "minecraft:")
	switch {
	case typ == "constant":
		return parseNumber(obj["value"], def)
	case typ == "uniform" || (typ == "" && obj["min"] != nil):
		lo, hi := parseNumber(obj["min"], 0), parseNumber(obj["max"], 0)
		n := Number{Min: lo.Min, Max: hi.Max, Mean: (lo.Mean + hi.Mean) / 2, Approximate: lo.Approximate || hi.Approximate}
		if lo.dist != nil && hi.dist != nil && lo.Min == lo.Max && hi.Min == hi.Max {
			// Rolls use Mth.nextInt(min, max): uniform over the inclusive range.
			a, b := int(math.Round(lo.Min)), int(math.Round(hi.Max))
			if b >= a {
				n.dist = map[int]float64{}
				for k := a; k <= b; k++ {
					n.dist[k] = 1 / float64(b-a+1)
				}
			}
		}
		return n
	case typ == "binomial":
		nn, p := parseNumber(obj["n"], 0), parseNumber(obj["p"], 0)
		n := Number{Min: 0, Max: nn.Max, Mean: nn.Mean * p.Mean, Approximate: nn.Approximate || p.Approximate}
		if nn.Min == nn.Max && p.Min == p.Max {
			trials := int(math.Round(nn.Min))
			n.dist = map[int]float64{}
			for k := 0; k <= trials; k++ {
				n.dist[k] = binomial(trials, k) * math.Pow(p.Min, float64(k)) * math.Pow(1-p.Min, float64(trials-k))
			}
		}
		return n
	default:
		// score, storage, enchantment level… depend on the game state.
		return Number{Min: def, Max: def, Mean: def, Approximate: true}
	}
}

func binomial(n, k int) float64 {
	r := 1.0
	for i := 1; i <= k; i++ {
		r = r * float64(n-k+i) / float64(i)
	}
	return r
}

// distribution returns the probability of each integer outcome. Providers
// without a known distribution are treated as uniform over their range.
func (n Number) distribution() map[int]float64 {
	if n.dist != nil {
		return n.dist
	}
	a, b := int(math.Round(n.Min)), int(math.Round(n.Max))
	if b < a {
		a, b = b, a
	}
	d := map[int]float64{}
	for k := a; k <= b; k++ {
		d[k] = 1 / float64(b-a+1)
	}
	return d
}
