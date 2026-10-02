// Package starcatcher reads Starcatcher's fish (data/<ns>/starcatcher/fish)
// for the fishing tab. Rules follow Starcatcher 2.3 for 1.20.1
// (FishingBobEntity.reel, FishProperties.calculateChance): each fish has a
// base chance; restrictions add to it (or subtract 9999 when they fail) and
// the fish is drawn with a probability proportional to the result. A bait
// adds its bonus, so a fish with base chance 0 can only be caught with it.
package starcatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/fishing"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// ID of the discoverer, also the key of the attached source.
const ID = "starcatcher"

// Discoverer reads Starcatcher's fish.
type Discoverer struct{}

func (Discoverer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ID, Phase: discovery.PhaseSpecific, Priority: 60,
		Applies: discovery.Applicability{RequiresMods: []string{"starcatcher"}}}
}

type restriction struct {
	Type string `json:"type"`
	// dimension
	Dimensions          []string `json:"dimensions"`
	DimensionsBlacklist []string `json:"dimensions_blacklist"`
	// biome / biome bias
	Biomes              []string `json:"biomes"`
	BiomesTags          []string `json:"biomes_tags"`
	BiomesBlacklist     []string `json:"biomes_blacklist"`
	BiomesBlacklistTags []string `json:"biomes_blacklist_tags"`
	ExtraChance         int      `json:"extra_chance"`
	// fluid
	Fluids []string `json:"fluids"`
	// elevation, daytime
	MinY   *int `json:"min_y"`
	MaxY   *int `json:"max_y"`
	BestY  *int `json:"best_y"`
	Best   *int `json:"best_daytime"`
	Range  int  `json:"range"`
	AtBest int  `json:"extra_chance_at_best"`
	Ranges []struct {
		First  int64 `json:"first"`
		Second int64 `json:"second"`
	} `json:"ranges"`
	Weather string             `json:"weather"`
	Seasons map[string]int     `json:"season_extra_chance"`
	Baits   map[string]float64 `json:"baits"`
	Limit   int                `json:"limit"`
	Chance  float64            `json:"chance"`
	Rarity  []struct {
		Count     int    `json:"count"`
		CountType string `json:"count_type"`
		Rarity    string `json:"rarity"`
	} `json:"rarities"`
	Override string `json:"translation_override"`
}

type fishFile struct {
	BaseChance float64 `json:"base_chance"`
	CatchInfo  struct {
		Item              string `json:"item"`
		Entity            string `json:"entity"`
		AlwaysSpawnEntity bool   `json:"always_spawn_entity"`
	} `json:"catch_info"`
	Rarity       string        `json:"rarity"`
	Restrictions []restriction `json:"restrictions"`
}

// entry keeps what the chance rules need beyond the common model.
type entry struct {
	*fishing.Entry
	base      float64
	biasRules []bias
	pct       float64 // percentage_chance, 0 = none
}

type bias struct {
	rule  fishing.BiomeRule
	extra float64
}

func (Discoverer) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	seasons := in.Target.Mods["sereneseasons"] || in.Target.Mods["eclipticseasons"] || in.Target.Mods["tfc"]
	var entries []*entry
	for _, id := range in.Resources.IDs(resources.TypeStarcatcherFish) {
		var raw json.RawMessage
		ok, err := in.Resources.ReadJSON(resources.TypeStarcatcherFish, id, &raw)
		if err != nil {
			in.Diagnostics.Add(domain.LevelWarning, "discovery", id.String(), "Starcatcher: %v", err)
			continue
		}
		if !ok || !fishing.ConditionsMet(raw, in.Target.Mods) {
			continue
		}
		var f fishFile
		if err := json.Unmarshal(raw, &f); err != nil {
			in.Diagnostics.Add(domain.LevelWarning, "discovery", id.String(), "Starcatcher: %v", err)
			continue
		}
		item, err := domain.ParseResourceID(f.CatchInfo.Item)
		if err != nil {
			continue
		}
		e := &entry{Entry: &fishing.Entry{ID: id, Kind: fishing.KindFish, Item: item, Rarity: f.Rarity, Weight: f.BaseChance}, base: f.BaseChance}
		if f.CatchInfo.AlwaysSpawnEntity {
			if ent, err := domain.ParseResourceID(f.CatchInfo.Entity); err == nil {
				e.Spawns = ent
			}
		}
		for _, r := range f.Restrictions {
			e.restriction(r, seasons)
		}
		e.BaitOnly = e.base <= 0 && len(e.Baits) > 0
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		return nil
	}
	src := &fishing.Source{
		Mod: ID, Name: "Starcatcher", Order: 10,
		Intro: []string{
			"Se pesca con las cañas de Starcatcher. Cada pez tiene una probabilidad base; el bioma, la dimensión, el líquido, la altura, la hora, el clima y las estaciones deciden si puede salir.",
			"Entre los peces posibles se elige uno al azar en proporción a su probabilidad. Un cebo suma su bonificación: los peces con probabilidad base 0 solo salen con su cebo.",
		},
	}
	for _, e := range entries {
		src.Entries = append(src.Entries, e.Entry)
	}
	biomes := fishing.Biomes(in)
	for _, b := range biomes {
		if bc := evaluate(entries, b); bc != nil {
			src.Biomes = append(src.Biomes, bc)
		}
	}
	placed := map[*fishing.Entry]bool{}
	for _, bc := range src.Biomes {
		for _, g := range bc.Groups {
			for _, c := range g.Catches {
				placed[c.Entry] = true
			}
		}
	}
	for _, e := range entries {
		if !placed[e.Entry] {
			src.Unplaced = append(src.Unplaced, e.Entry)
		}
	}
	out.Attach(fishing.ExtraPrefix+ID, src)
	return nil
}

func ids(list []string) []domain.ResourceID {
	var out []domain.ResourceID
	for _, s := range list {
		if id, err := domain.ParseResourceID(s); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func tags(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, "#"+s)
	}
	return out
}

func (e *entry) restriction(r restriction, seasons bool) {
	cond := func(text string) {
		e.Conditions = append(e.Conditions, fishing.Cond{Text: text, Key: r.Override})
	}
	switch strings.TrimPrefix(r.Type, "starcatcher:") {
	case "dimension":
		e.Dimensions = append(e.Dimensions, ids(r.Dimensions)...)
		e.NotDimensions = append(e.NotDimensions, ids(r.DimensionsBlacklist)...)
	case "biome":
		e.Biomes = append(e.Biomes, fishing.BiomeRule{
			Any:  append(append([]string(nil), r.Biomes...), tags(r.BiomesTags)...),
			None: append(append([]string(nil), r.BiomesBlacklist...), tags(r.BiomesBlacklistTags)...),
		})
	case "biome_bias":
		rule := fishing.BiomeRule{Any: append(append([]string(nil), r.Biomes...), tags(r.BiomesTags)...)}
		e.biasRules = append(e.biasRules, bias{rule: rule, extra: float64(r.ExtraChance)})
		e.Conditions = append(e.Conditions, fishing.Cond{Text: fmt.Sprintf("más probable (+%d) en", r.ExtraChance), Refs: ids(r.Biomes), RefKind: "biome", Key: r.Override})
	case "fluid":
		for _, f := range r.Fluids {
			e.Fluids = append(e.Fluids, fishing.Fluid(f))
		}
	case "elevation_restriction":
		switch {
		case r.MinY != nil && r.MaxY != nil && *r.MaxY < math.MaxInt32:
			cond(fmt.Sprintf("altura entre Y=%d y Y=%d", *r.MinY, *r.MaxY))
		case r.MinY != nil:
			cond(fmt.Sprintf("altura desde Y=%d", *r.MinY))
		case r.MaxY != nil:
			cond(fmt.Sprintf("altura hasta Y=%d", *r.MaxY))
		}
	case "elevation_bias":
		if r.BestY != nil {
			cond(fmt.Sprintf("más probable cerca de Y=%d (±%d)", *r.BestY, r.Range))
		}
	case "weather_restriction":
		cond("solo con clima: " + fishing.Weather(r.Weather))
	case "daytime_restriction":
		var parts []string
		for _, rg := range r.Ranges {
			parts = append(parts, "de "+fishing.Clock(rg.First)+" a "+fishing.Clock(rg.Second))
		}
		cond("solo " + strings.Join(parts, " o "))
	case "daytime_bias":
		if r.Best != nil {
			cond("más probable hacia las " + fishing.Clock(int64(*r.Best)))
		}
	case "season":
		var allowed []string
		for s, v := range r.Seasons {
			if v > -1000 {
				allowed = append(allowed, fishing.Season(s))
			}
		}
		sort.Strings(allowed)
		c := fishing.Cond{Text: "solo en: " + strings.Join(allowed, ", "), Key: r.Override, Inactive: !seasons}
		e.Conditions = append(e.Conditions, c)
	case "bait":
		keys := make([]string, 0, len(r.Baits))
		for k := range r.Baits {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if id, err := domain.ParseResourceID(k); err == nil {
				e.Baits = append(e.Baits, fishing.Bait{Item: id, Bonus: r.Baits[k]})
			}
		}
	case "caught_limit":
		cond(fmt.Sprintf("cada jugador puede pescarlo como máximo %d veces", r.Limit))
	case "rarity_count":
		for _, rc := range r.Rarity {
			what := "peces"
			if rc.CountType == "unique" {
				what = "especies distintas"
			}
			cond(fmt.Sprintf("requiere haber pescado %d %s de rareza %s", rc.Count, what, rarity(rc.Rarity)))
		}
	case "percentage_chance":
		e.pct = r.Chance
		cond(fmt.Sprintf("solo aparece el %s de las veces", percent(r.Chance)))
	}
}

func rarity(r string) string {
	m := map[string]string{"common": "común", "uncommon": "poco común", "rare": "rara", "epic": "épica", "legendary": "legendaria", "golden": "dorada"}
	if v, ok := m[r]; ok {
		return v
	}
	return r
}

func percent(p float64) string {
	return strings.Replace(fmt.Sprintf("%g %%", p*100), ".", ",", 1)
}

// evaluate computes the chances in a biome, grouped by fluid.
func evaluate(entries []*entry, b fishing.Biome) *fishing.BiomeCatches {
	bc := &fishing.BiomeCatches{Biome: b.ID, Dimension: b.Dimension}
	for _, fluid := range []string{fishing.FluidWater, fishing.FluidLava, fishing.FluidVoid} {
		type cand struct {
			e      *entry
			chance float64
		}
		var pool, baited []cand
		total := 0.0
		for _, e := range entries {
			if !e.InDimension(b) || !e.MatchesBiome(b) || !e.HasFluid(fluid) {
				continue
			}
			c := e.base
			for _, bs := range e.biasRules {
				if bs.rule.Matches(b) {
					c += bs.extra
				}
			}
			if c > 0 {
				pool = append(pool, cand{e, c})
				total += c
			}
			if len(e.Baits) > 0 {
				baited = append(baited, cand{e, c})
			}
		}
		if len(pool) == 0 && len(baited) == 0 {
			continue
		}
		g := &fishing.Group{Title: "Pescando en " + fluid, Fluid: fluid,
			Note: "Probabilidad entre los peces de este bioma sin cebo especial. La altura, la hora y el clima pueden descartar algunos, y entonces suben los demás."}
		for _, c := range pool {
			catch := fishing.Catch{Entry: c.e.Entry, Chance: c.chance / total, Weight: c.chance}
			if c.e.pct > 0 {
				catch.Chance *= c.e.pct
			}
			g.Catches = append(g.Catches, catch)
		}
		fishing.SortCatches(g.Catches)
		bc.Groups = append(bc.Groups, g)
		if len(baited) > 0 {
			bg := &fishing.Group{Title: "Con cebo, en " + fluid, Fluid: fluid,
				Note: "Probabilidad usando ese cebo: el cebo suma su bonificación a este pez."}
			for _, c := range baited {
				for i := range c.e.Baits {
					bait := c.e.Baits[i]
					w := c.chance + bait.Bonus
					catch := fishing.Catch{Entry: c.e.Entry, Weight: w, Bait: &bait}
					if w > 0 {
						catch.Chance = w / (total - max(c.chance, 0) + w)
					}
					if c.e.pct > 0 {
						catch.Chance *= c.e.pct
					}
					bg.Catches = append(bg.Catches, catch)
				}
			}
			fishing.SortCatches(bg.Catches)
			bc.Groups = append(bc.Groups, bg)
		}
	}
	if len(bc.Groups) == 0 {
		return nil
	}
	return bc
}
