// Package loot parses loot tables and computes, for each item, how likely it
// is to appear when the table is rolled once (e.g. per chest opened).
package loot

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// Drop is an item a table can produce.
type Drop struct {
	Item domain.ResourceID `json:"item"`
	// Chance of getting at least one stack when the table is rolled once.
	Chance float64 `json:"chance"`
	// CountMin/CountMax is the stack size range of a single occurrence.
	CountMin float64 `json:"countMin"`
	CountMax float64 `json:"countMax"`
	// Expected is the expected number of items per roll of the table.
	Expected float64 `json:"expected"`
	// Notes are loot function/condition names worth showing (enchanted…).
	Notes []string `json:"notes,omitempty"`
	// Approximate is true when some mechanic could not be modelled exactly.
	Approximate bool `json:"approximate,omitempty"`
}

// Table is a resolved loot table.
type Table struct {
	ID          domain.ResourceID   `json:"id"`
	Type        string              `json:"type,omitempty"`
	Drops       []Drop              `json:"drops"`
	Nested      []domain.ResourceID `json:"nested,omitempty"`
	Approximate bool                `json:"approximate,omitempty"`
	// Missing lists referenced tables that do not exist.
	Missing []domain.ResourceID `json:"missing,omitempty"`
}

// Source provides raw loot table JSON and item tags.
type Source interface {
	LootTableJSON(id domain.ResourceID) ([]byte, bool, error)
	ItemTag(id domain.ResourceID) []domain.ResourceID
}

// Resolver parses and resolves tables, caching results.
type Resolver struct {
	src   Source
	cache map[domain.ResourceID]*Table
	busy  map[domain.ResourceID]bool
}

// NewResolver creates a resolver over a source.
func NewResolver(src Source) *Resolver {
	return &Resolver{src: src, cache: map[domain.ResourceID]*Table{}, busy: map[domain.ResourceID]bool{}}
}

type rawTable struct {
	Type  string    `json:"type"`
	Pools []rawPool `json:"pools"`
}

type rawPool struct {
	Rolls      json.RawMessage `json:"rolls"`
	Entries    []rawEntry      `json:"entries"`
	Conditions []rawCondition  `json:"conditions"`
	Functions  []rawFunction   `json:"functions"`
}

type rawEntry struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Value      json.RawMessage `json:"value"` // inline table (1.20.5+)
	Weight     *float64        `json:"weight"`
	Expand     bool            `json:"expand"`
	Children   []rawEntry      `json:"children"`
	Conditions []rawCondition  `json:"conditions"`
	Functions  []rawFunction   `json:"functions"`
}

type rawCondition map[string]json.RawMessage

type rawFunction map[string]json.RawMessage

func short(s string) string { return strings.TrimPrefix(s, "minecraft:") }

func (c rawCondition) name() string {
	var s string
	_ = json.Unmarshal(c["condition"], &s)
	return short(s)
}

func (f rawFunction) name() string {
	var s string
	_ = json.Unmarshal(f["function"], &s)
	return short(s)
}

// conditionChance returns the probability that the conditions pass, and
// whether that value is exact.
func conditionChance(conds []rawCondition) (float64, []string, bool) {
	p, exact := 1.0, true
	var notes []string
	for _, c := range conds {
		switch c.name() {
		case "random_chance", "random_chance_with_looting":
			var chance float64
			if err := json.Unmarshal(c["chance"], &chance); err == nil {
				p *= chance
			} else {
				exact = false
			}
		case "killed_by_player":
			notes = append(notes, "killed_by_player")
		case "survives_explosion":
			// Always passes for containers, chests are not explosions.
		default:
			notes = append(notes, c.name())
			exact = false
		}
	}
	return p, notes, exact
}

// Resolve returns the resolved table, or (nil, false) if it does not exist.
func (r *Resolver) Resolve(id domain.ResourceID) (*Table, bool, error) {
	if t, ok := r.cache[id]; ok {
		return t, true, nil
	}
	data, ok, err := r.src.LootTableJSON(id)
	if err != nil || !ok {
		return nil, ok, err
	}
	var raw rawTable
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, true, fmt.Errorf("loot table %s: %w", id, err)
	}
	r.busy[id] = true
	t := r.resolveRaw(id, raw)
	delete(r.busy, id)
	r.cache[id] = t
	return t, true, nil
}

// outcome is what one weighted candidate produces when chosen.
type outcome struct {
	item     domain.ResourceID
	prob     float64 // chance the candidate yields the item once chosen
	countMin float64
	countMax float64
	mean     float64 // expected items when chosen
	notes    []string
	approx   bool
}

type candidate struct {
	weight   float64
	outcomes []outcome
}

type accum struct {
	notMiss  float64 // probability of never getting the item, across pools
	expected float64
	countMin float64
	countMax float64
	notes    map[string]bool
	approx   bool
}

func (r *Resolver) resolveRaw(id domain.ResourceID, raw rawTable) *Table {
	t := &Table{ID: id, Type: short(raw.Type)}
	acc := map[domain.ResourceID]*accum{}
	for _, pool := range raw.Pools {
		poolChance, poolNotes, exact := conditionChance(pool.Conditions)
		if !exact {
			t.Approximate = true
		}
		rolls := parseNumber(pool.Rolls, 1)
		if rolls.Approximate {
			t.Approximate = true
		}
		var cands []candidate
		for _, e := range pool.Entries {
			cands = append(cands, r.candidates(t, e)...)
		}
		total := 0.0
		for _, c := range cands {
			total += c.weight
		}
		if total <= 0 {
			continue
		}
		// Per-roll chance and expected amount of each item in this pool.
		perRoll := map[domain.ResourceID]float64{}
		for _, c := range cands {
			pc := c.weight / total
			for _, o := range c.outcomes {
				perRoll[o.item] = 1 - (1-perRoll[o.item])*(1-pc*o.prob)
				a := acc[o.item]
				if a == nil {
					a = &accum{notMiss: 1, countMin: math.Inf(1), notes: map[string]bool{}}
					acc[o.item] = a
				}
				a.expected += poolChance * rolls.Mean * pc * o.mean
				a.countMin = math.Min(a.countMin, o.countMin)
				a.countMax = math.Max(a.countMax, o.countMax)
				a.approx = a.approx || o.approx
				for _, n := range append(o.notes, poolNotes...) {
					a.notes[n] = true
				}
			}
		}
		dist := rolls.distribution()
		for item, q := range perRoll {
			hit := 0.0
			for k, pk := range dist {
				hit += pk * (1 - math.Pow(1-q, float64(k)))
			}
			acc[item].notMiss *= 1 - poolChance*hit
		}
	}
	for item, a := range acc {
		d := Drop{Item: item, Chance: 1 - a.notMiss, CountMin: a.countMin, CountMax: a.countMax, Expected: a.expected, Approximate: a.approx}
		for n := range a.notes {
			d.Notes = append(d.Notes, n)
		}
		sort.Strings(d.Notes)
		t.Drops = append(t.Drops, d)
	}
	sort.Slice(t.Drops, func(i, j int) bool {
		if t.Drops[i].Chance != t.Drops[j].Chance {
			return t.Drops[i].Chance > t.Drops[j].Chance
		}
		return t.Drops[i].Item.String() < t.Drops[j].Item.String()
	})
	return t
}

// candidates expands an entry into the weighted candidates of a pool, the
// way the game expands composite entries.
func (r *Resolver) candidates(t *Table, e rawEntry) []candidate {
	condP, condNotes, exact := conditionChance(e.Conditions)
	if !exact {
		t.Approximate = true
	}
	weight := 1.0
	if e.Weight != nil {
		weight = *e.Weight
	}
	if weight <= 0 {
		return nil
	}
	count, fnNotes, fnApprox := countAndNotes(e.Functions)
	if fnApprox {
		t.Approximate = true
	}
	notes := append(condNotes, fnNotes...)
	mk := func(item domain.ResourceID) outcome {
		return outcome{item: item, prob: condP, countMin: count.Min, countMax: count.Max, mean: condP * count.Mean, notes: notes, approx: !exact || fnApprox}
	}

	switch short(e.Type) {
	case "item":
		id, err := domain.ParseResourceID(e.Name)
		if err != nil {
			return nil
		}
		return []candidate{{weight: weight, outcomes: []outcome{mk(id)}}}
	case "tag":
		tagID, err := domain.ParseResourceID(e.Name)
		if err != nil {
			return nil
		}
		items := r.src.ItemTag(tagID)
		if len(items) == 0 {
			t.Approximate = true
			return nil
		}
		if e.Expand {
			out := make([]candidate, 0, len(items))
			for _, it := range items {
				out = append(out, candidate{weight: weight, outcomes: []outcome{mk(it)}})
			}
			return out
		}
		c := candidate{weight: weight}
		for _, it := range items {
			c.outcomes = append(c.outcomes, mk(it))
		}
		return []candidate{c}
	case "loot_table":
		nested := r.nested(t, e)
		if nested == nil {
			return []candidate{{weight: weight}}
		}
		c := candidate{weight: weight}
		for _, d := range nested.Drops {
			c.outcomes = append(c.outcomes, outcome{
				item: d.Item, prob: condP * d.Chance, countMin: d.CountMin, countMax: d.CountMax,
				mean: condP * d.Expected, notes: append(append([]string(nil), notes...), d.Notes...),
				approx: d.Approximate || nested.Approximate || !exact,
			})
		}
		return []candidate{c}
	case "group":
		var out []candidate
		for _, child := range e.Children {
			out = append(out, r.candidates(t, child)...)
		}
		return out
	case "alternatives", "sequence":
		// Children are tried in order; modelling the first one is exact only
		// when it has no conditions.
		if len(e.Children) == 0 {
			return nil
		}
		if len(e.Children) > 1 {
			t.Approximate = true
		}
		return r.candidates(t, e.Children[0])
	case "empty":
		return []candidate{{weight: weight}}
	default: // dynamic and modded entry types
		t.Approximate = true
		return []candidate{{weight: weight}}
	}
}

func (r *Resolver) nested(t *Table, e rawEntry) *Table {
	if len(e.Value) > 0 && e.Value[0] == '{' {
		var inline rawTable
		if err := json.Unmarshal(e.Value, &inline); err == nil {
			return r.resolveRaw(t.ID, inline)
		}
	}
	name := e.Name
	if name == "" && len(e.Value) > 0 {
		_ = json.Unmarshal(e.Value, &name)
	}
	id, err := domain.ParseResourceID(name)
	if err != nil {
		t.Approximate = true
		return nil
	}
	t.Nested = append(t.Nested, id)
	if r.busy[id] {
		t.Approximate = true // recursive tables
		return nil
	}
	nested, ok, err := r.Resolve(id)
	if err != nil || !ok {
		t.Missing = append(t.Missing, id)
		t.Approximate = true
		return nil
	}
	return nested
}

// Functions that only decorate the item; shown as notes on the site.
var notableFunctions = map[string]bool{
	"enchant_randomly": true, "enchant_with_levels": true, "set_potion": true,
	"exploration_map": true, "set_stew_effect": true, "set_instrument": true,
	"set_enchantments": true, "set_nbt": true, "set_components": true,
	"set_custom_data": true, "set_name": true, "set_damage": true,
	"set_attributes": true, "set_contents": true, "set_banner_pattern": true,
	"looting_enchant": true, "enchanted_count_increase": true, "furnace_smelt": true,
}

func countAndNotes(fns []rawFunction) (Number, []string, bool) {
	count := constant(1)
	var notes []string
	approx := false
	for _, f := range fns {
		name := f.name()
		if name == "set_count" {
			var add bool
			_ = json.Unmarshal(f["add"], &add)
			n := parseNumber(f["count"], 1)
			if add {
				n = Number{Min: count.Min + n.Min, Max: count.Max + n.Max, Mean: count.Mean + n.Mean, Approximate: n.Approximate}
			}
			if _, conditional := f["conditions"]; conditional {
				approx = true
			}
			count = n
			approx = approx || n.Approximate
			continue
		}
		if notableFunctions[name] {
			notes = append(notes, name)
			continue
		}
		if name == "limit_count" || name == "explosion_decay" || name == "copy_name" || name == "copy_nbt" || name == "copy_state" {
			continue
		}
		notes = append(notes, name)
	}
	return count, notes, approx
}
