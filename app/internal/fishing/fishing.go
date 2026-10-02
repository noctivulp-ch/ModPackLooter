// Package fishing is the common model of the fishing tab. Each fishing mod
// (Starcatcher, Tide…) has its own discoverer that reads its data, decides
// the chances with its own rules and attaches a Source; the site only
// renders Sources: pesca › fuente › bioma › condiciones › captura.
package fishing

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// ExtraPrefix is the prefix of the keys fishing discoverers attach with.
const ExtraPrefix = "fishing/"

// Kind is what a catch entry gives.
type Kind string

const (
	KindFish  Kind = "fish"  // a fish (an item)
	KindLoot  Kind = "loot"  // a loot table (treasure, junk)
	KindCrate Kind = "crate" // a crate block filled with a loot table
)

// Fluids, normalised.
const (
	FluidWater = "agua"
	FluidLava  = "lava"
	FluidVoid  = "vacío"
)

// Source is one fishing system.
type Source struct {
	Mod   string // mod id; the page lives at pesca/<Mod>/
	Name  string
	Intro []string // how the mod picks a catch, in plain words
	// Entries is every catch the system knows, valid for this modpack.
	Entries []*Entry
	// Biomes holds the chances per biome, computed with the mod's rules.
	Biomes []*BiomeCatches
	// Unplaced are entries no known biome can give (e.g. modded dimensions
	// without a model world); they are still listed.
	Unplaced []*Entry
	// Everywhere are catches that do not depend on the biome.
	Everywhere []*Group
	// ReplacesVanilla is true when this mod takes over the vanilla rod.
	ReplacesVanilla bool
	// Order sorts sources in the tab (vanilla last).
	Order int
}

// Places returns the biome catches that can give the entry.
func (s *Source) Places(e *Entry) []*BiomeCatches {
	var out []*BiomeCatches
	for _, b := range s.Biomes {
	groups:
		for _, g := range b.Groups {
			for _, c := range g.Catches {
				if c.Entry == e {
					out = append(out, b)
					break groups
				}
			}
		}
	}
	return out
}

// Entry is one thing that can be caught.
type Entry struct {
	ID     domain.ResourceID // data file
	Kind   Kind
	Item   domain.ResourceID // fish item (zero for tables)
	Table  domain.ResourceID // loot table (loot and crates)
	Block  domain.ResourceID // crate block
	Rarity string
	Weight float64
	// Fluids where it can be caught; empty = any.
	Fluids []string
	// Dimensions (empty = any) and excluded dimensions.
	Dimensions    []domain.ResourceID
	NotDimensions []domain.ResourceID
	// Biomes: every rule must accept the biome.
	Biomes []BiomeRule
	// Conditions are the rest, for reading.
	Conditions []Cond
	// Temperature makes the weight depend on the biome temperature (Tide).
	Temperature *Temperature
	// Baits that make it possible (Starcatcher fish with base chance 0) or
	// more likely; Bonus is added to the chance.
	Baits    []Bait
	BaitOnly bool
	Spawns   domain.ResourceID // entity that appears with the catch
}

// BiomeRule is a whitelist and blacklist of biomes and tags ("#ns:path").
type BiomeRule struct {
	Any, None []string
}

// Empty reports whether the rule accepts every biome.
func (r BiomeRule) Empty() bool { return len(r.Any)+len(r.None) == 0 }

// Cond is a condition shown to the reader. Key is a lang key that may say it
// better (Starcatcher's translation_override); Refs are ids the site names
// (items, structures, biomes) and appends to Text.
type Cond struct {
	Text    string
	Key     string
	Refs    []domain.ResourceID
	RefKind string // "item", "structure", "biome"
	// Inactive marks conditions without effect in this modpack (e.g.
	// seasons without a seasons mod).
	Inactive bool
}

// Temperature is Tide's preferred temperature: the weight is multiplied by
// max(0, 1 - ((t - Preferred) / Tolerance)²).
type Temperature struct {
	Preferred, Tolerance float64
}

// Scale returns the weight multiplier for a biome temperature.
func (t *Temperature) Scale(temp float64) float64 {
	if t == nil || t.Tolerance == 0 {
		return 1
	}
	x := (temp - t.Preferred) / t.Tolerance
	return max(0, 1-x*x)
}

// MatchesBiome reports whether every biome rule accepts the biome.
func (e *Entry) MatchesBiome(b Biome) bool {
	for _, r := range e.Biomes {
		if !r.Matches(b) {
			return false
		}
	}
	return true
}

// Bait is an item that adds chance to a catch.
type Bait struct {
	Item  domain.ResourceID
	Bonus float64
}

// BiomeCatches are the catches of a biome, grouped by fluid.
type BiomeCatches struct {
	Biome     domain.ResourceID
	Dimension string
	Groups    []*Group
}

// Group is a list of catches competing with each other.
type Group struct {
	Title   string
	Fluid   string
	Note    string
	Catches []Catch
}

// Catch is an entry with its chance within its group.
type Catch struct {
	Entry  *Entry
	Chance float64 // within the group; 0 when only a weight can be given
	Weight float64 // the weight used, for reading
	Bait   *Bait   // the chance applies when fishing with this bait
	Note   string
}

// Biome is what fishing rules need to know about a biome.
type Biome struct {
	ID          domain.ResourceID
	Dimension   string // "" if unknown
	Temperature float64
	Tags        map[domain.ResourceID]bool
}

// Matches reports whether the rule accepts the biome.
func (r BiomeRule) Matches(b Biome) bool {
	in := func(list []string) bool {
		for _, s := range list {
			if tag, ok := strings.CutPrefix(s, "#"); ok {
				if id, err := domain.ParseResourceID(tag); err == nil && b.Tags[id] {
					return true
				}
				continue
			}
			if id, err := domain.ParseResourceID(s); err == nil && id == b.ID {
				return true
			}
		}
		return false
	}
	if len(r.Any) > 0 && !in(r.Any) {
		return false
	}
	return !in(r.None)
}

// InDimension reports whether the entry is possible in the biome's dimension.
func (e *Entry) InDimension(b Biome) bool {
	for _, d := range e.NotDimensions {
		if d.String() == b.Dimension {
			return false
		}
	}
	if len(e.Dimensions) == 0 {
		return true
	}
	for _, d := range e.Dimensions {
		if d.String() == b.Dimension {
			return true
		}
	}
	return false
}

// HasFluid reports whether the entry can be caught in the fluid.
func (e *Entry) HasFluid(f string) bool {
	if len(e.Fluids) == 0 {
		return true
	}
	for _, x := range e.Fluids {
		if x == f {
			return true
		}
	}
	return false
}

// Fluid normalises a fluid id ("minecraft:water", "water", "lava"…).
func Fluid(s string) string {
	s = strings.TrimPrefix(strings.ToLower(s), "minecraft:")
	switch {
	case strings.Contains(s, "lava"):
		return FluidLava
	case strings.Contains(s, "void"):
		return FluidVoid
	case strings.Contains(s, "water"):
		return FluidWater
	}
	return s
}

// Biomes returns the biomes of the modpack with their dimension, tags and
// temperature. With a model world, only its biomes are returned.
func Biomes(in discovery.Input) []Biome {
	ids := in.Resources.IDs(resources.TypeBiome)
	dimOf := map[domain.ResourceID]string{}
	if w := in.Files.Level(); w != nil {
		if all, complete := w.AllBiomes(); complete {
			ids = ids[:0:0]
			for id := range all {
				ids = append(ids, id)
			}
		}
		for _, d := range w.Dimensions {
			for _, b := range d.Biomes {
				dimOf[b] = d.ID.String()
			}
		}
	}
	tags := map[domain.ResourceID]map[domain.ResourceID]bool{}
	for _, tag := range in.Resources.IDs(resources.TypeBiomeTag) {
		for _, b := range in.Resources.Tag(resources.TypeBiomeTag, tag) {
			if tags[b] == nil {
				tags[b] = map[domain.ResourceID]bool{}
			}
			tags[b][tag] = true
		}
	}
	byTag := map[string]string{"minecraft:is_overworld": "minecraft:overworld", "minecraft:is_nether": "minecraft:the_nether", "minecraft:is_end": "minecraft:the_end"}
	var out []Biome
	for _, id := range ids {
		b := Biome{ID: id, Dimension: dimOf[id], Tags: tags[id]}
		if b.Tags == nil {
			b.Tags = map[domain.ResourceID]bool{}
		}
		if b.Dimension == "" {
			for t, d := range byTag {
				if b.Tags[domain.MustParseResourceID(t)] {
					b.Dimension = d
				}
			}
		}
		var raw struct {
			Temperature *float64 `json:"temperature"`
		}
		if ok, _ := in.Resources.ReadJSON(resources.TypeBiome, id, &raw); ok && raw.Temperature != nil {
			b.Temperature = *raw.Temperature
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() < out[j].ID.String() })
	return out
}

// ConditionsMet evaluates Forge/NeoForge load conditions of a data file
// ("forge:conditions", "neoforge:conditions"); unknown conditions pass.
func ConditionsMet(raw json.RawMessage, mods map[string]bool) bool {
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return true
	}
	for _, key := range []string{"forge:conditions", "neoforge:conditions"} {
		var conds []json.RawMessage
		if json.Unmarshal(doc[key], &conds) != nil {
			continue
		}
		for _, c := range conds {
			if !condition(c, mods) {
				return false
			}
		}
	}
	return true
}

func condition(raw json.RawMessage, mods map[string]bool) bool {
	var c struct {
		Type   string            `json:"type"`
		ModID  string            `json:"modid"`
		Value  json.RawMessage   `json:"value"`
		Values []json.RawMessage `json:"values"`
	}
	if json.Unmarshal(raw, &c) != nil {
		return true
	}
	_, typ, _ := strings.Cut(c.Type, ":")
	switch typ {
	case "mod_loaded":
		return mods[c.ModID]
	case "not":
		return !condition(c.Value, mods)
	case "and":
		for _, v := range c.Values {
			if !condition(v, mods) {
				return false
			}
		}
		return true
	case "or":
		for _, v := range c.Values {
			if condition(v, mods) {
				return true
			}
		}
		return false
	case "false":
		return false
	}
	return true
}

// Clock turns game ticks into a time of day ("06:00" is tick 0).
func Clock(ticks int64) string {
	t := ((ticks%24000)+24000)%24000 + 6000
	h := (t / 1000) % 24
	m := (t % 1000) * 60 / 1000
	return fmt.Sprintf("%02d:%02d", h, m)
}

// MoonPhase names a moon phase (0 = full moon).
func MoonPhase(n int) string {
	names := []string{"luna llena", "menguante gibosa", "cuarto menguante", "luna menguante", "luna nueva", "luna creciente", "cuarto creciente", "creciente gibosa"}
	if n >= 0 && n < len(names) {
		return names[n]
	}
	return fmt.Sprint(n)
}

// Season names a season ("early_autumn" → "principio de otoño").
func Season(s string) string {
	parts := map[string]string{"spring": "primavera", "summer": "verano", "autumn": "otoño", "fall": "otoño", "winter": "invierno"}
	when, base, ok := strings.Cut(s, "_")
	if !ok {
		if v, ok := parts[s]; ok {
			return v
		}
		return s
	}
	pre := map[string]string{"early": "principio de ", "mid": "mitad de ", "late": "final de "}
	return pre[when] + parts[base]
}

// Weather names a weather type.
func Weather(s string) string {
	switch strings.ToLower(s) {
	case "clear":
		return "despejado"
	case "rain":
		return "lluvia"
	case "storm", "thunder":
		return "tormenta"
	}
	return s
}

// SortCatches orders catches by chance, then by id.
func SortCatches(list []Catch) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Chance != list[j].Chance {
			return list[i].Chance > list[j].Chance
		}
		return list[i].Entry.ID.String() < list[j].Entry.ID.String()
	})
}
