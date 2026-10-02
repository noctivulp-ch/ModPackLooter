package site

import (
	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
	"math"
	"strconv"
	"strings"
)

// Dimension names, in the order the site lists them.
var dimensionOrder = []string{"Mundo normal", "Nether", "End", "Otras dimensiones"}

// biomeDimensions tells the dimension of a biome by the usual tags.
func biomeDimensions(res *analysis.Result) func(domain.ResourceID) string {
	in := map[domain.ResourceID]string{}
	for _, d := range []struct{ tag, name string }{
		{"minecraft:is_end", "End"}, {"forge:is_end", "End"}, {"c:is_end", "End"},
		{"minecraft:is_nether", "Nether"}, {"forge:is_nether", "Nether"}, {"c:is_nether", "Nether"},
		{"minecraft:is_overworld", "Mundo normal"}, {"forge:is_overworld", "Mundo normal"}, {"c:is_overworld", "Mundo normal"},
	} {
		for _, b := range res.Resources.Tag(resources.TypeBiomeTag, domain.MustParseResourceID(d.tag)) {
			if _, ok := in[b]; !ok {
				in[b] = d.name
			}
		}
	}
	return func(id domain.ResourceID) string {
		if d, ok := in[id]; ok {
			return d
		}
		return "Otras dimensiones"
	}
}

// Step is a link of the chain that leads to an item: a biome, a creature,
// a merchant… with the chance of that step, when known.
type Step struct {
	Label  string
	URL    string
	Chance float64
	Detail string
}

// Line is the whole chain from where you are to the item, and the real
// chance: the product of every step, per Unit.
type Line struct {
	Steps []Step
	Total float64
	Unit  string
}

// bestSpawn is the biome where the creature is most likely, among the ones
// that still exist.
func (c *Creature) bestSpawn() (CreatureBiome, bool) {
	var best CreatureBiome
	ok := false
	for _, n := range c.Natural {
		if !ok || n.Share > best.Share {
			best, ok = n, true
		}
	}
	return best, ok
}

func spawnSteps(c *Creature, sp CreatureBiome) []Step {
	var steps []Step
	if sp.Biome != nil {
		steps = append(steps, Step{Label: sp.Biome.Dimension})
	}
	steps = append(steps, Step{Label: sp.Name, URL: sp.URL})
	steps = append(steps, Step{Label: c.Name, URL: c.URL, Chance: sp.Share, Detail: "de las apariciones de tipo " + sp.Category})
	return steps
}

// mobLine is biome → creature → drop.
func mobLine(c *Creature, drop float64) *Line {
	if c == nil {
		return nil
	}
	sp, ok := c.bestSpawn()
	if !ok {
		return nil
	}
	l := &Line{Steps: spawnSteps(c, sp)}
	l.Steps = append(l.Steps, Step{Label: "lo suelta", Chance: drop})
	l.Total = sp.Share * drop
	l.Unit = "por cada criatura " + sp.Category + " que aparece en " + sp.Name
	return l
}

// tradeLine is biome → creature (→ the creature it turns into) → offer.
func tradeLine(t *TradeRef) *Line {
	c := t.Merchant.Creature
	if c == nil {
		return nil
	}
	offer := Step{Label: t.Merchant.Name + " lo ofrece", URL: t.Merchant.URL, Chance: t.Offer.Chance}
	if t.Level != "" {
		offer.Detail = "nivel " + t.Level
	}
	if sp, ok := c.bestSpawn(); ok {
		l := &Line{Steps: append(spawnSteps(c, sp), offer)}
		l.Total = sp.Share * t.Offer.Chance
		l.Unit = "por cada criatura " + sp.Category + " que aparece en " + sp.Name
		return l
	}
	for _, f := range c.From {
		if sp, ok := f.bestSpawn(); ok {
			l := &Line{Steps: spawnSteps(f, sp)}
			l.Steps = append(l.Steps, Step{Label: c.Name, URL: c.URL, Detail: "al convertirlo"}, offer)
			l.Total = sp.Share * t.Offer.Chance
			l.Unit = "por cada criatura " + sp.Category + " que aparece en " + sp.Name + ", si la conviertes"
			return l
		}
	}
	return nil
}

// Line is the chain from a biome to this trade, when known.
func (t *TradeRef) Line() *Line { return tradeLine(t) }

// Line is the chain from a biome to this drop, when known.
func (s *ItemSource) Line() *Line {
	if s.Table == nil || s.Table.Creature == nil {
		return nil
	}
	return mobLine(s.Table.Creature, s.Chance)
}

// fine formats small chances with two significant digits and "1 de cada N".
func fine(p float64) string {
	if p <= 0 || p*100 >= 0.1 {
		return pct(p)
	}
	v := p * 100
	digits := 1
	for x := v; x < 1; x *= 10 {
		digits++
	}
	s := strings.Replace(strconv.FormatFloat(v, 'f', digits, 64), ".", ",", 1)
	n := math.Round(1 / p)
	return s + " % · 1 de cada " + thousands(int64(n))
}

func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}
