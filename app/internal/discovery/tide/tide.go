// Package tide reads Tide 2's fishing data (data/<ns>/fishing/{fish,loot,
// crates}) for the fishing tab. Rules follow Tide 2.1.1
// (TideFishingManager, FishingRandomSelector): every cast picks, by weight,
// among the loot entries whose conditions pass, the fish selector (weight
// 85) and the crate selector (crateWeight, 4 by default, if a crate fits);
// the fish selector then picks a fish by its selection_weight, multiplied by
// its temperature modifier.
package tide

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/fishing"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// ID of the discoverer, also the key of the attached source.
const ID = "tide"

// FishWeight is the weight of Tide's fish selector.
const FishWeight = 85.0

// Discoverer reads Tide's fishing data.
type Discoverer struct{}

func (Discoverer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ID, Phase: discovery.PhaseSpecific, Priority: 60,
		Applies: discovery.Applicability{RequiresMods: []string{"tide"}}}
}

type rawCond struct {
	Type       string          `json:"type"`
	Biomes     []string        `json:"biomes"`
	Dimensions []string        `json:"dimensions"`
	Fluid      string          `json:"fluid"`
	Y          *int            `json:"y"`
	MinY       *int            `json:"min_y"`
	MaxY       *int            `json:"max_y"`
	Structures []string        `json:"structures"`
	Seasons    []string        `json:"seasons"`
	AnyOf      []int           `json:"any_of"`
	Types      []string        `json:"types"`
	Min        *int            `json:"min"`
	Max        *int            `json:"max"`
	Open       *bool           `json:"in_open_water"`
	Blocks     string          `json:"blocks"`
	Radius     int             `json:"radius"`
	A          json.RawMessage `json:"a"`
	B          json.RawMessage `json:"b"`
	Ranges     []struct {
		Min int64 `json:"min"`
		Max int64 `json:"max"`
	} `json:"ranges"`
}

type rawModifier struct {
	Type       string          `json:"type"`
	Preferred  float64         `json:"preferred_temperature"`
	Tolerance  float64         `json:"temperature_tolerance"`
	If         json.RawMessage `json:"if"`
	Multiplier float64         `json:"multiplier"`
}

type config struct {
	crateWeight     float64
	overrideVanilla bool
	origin          string
}

// readConfig reads Tide's server config (cloth-config JSON).
func readConfig(in discovery.Input) config {
	c := config{crateWeight: 4, overrideVanilla: true, origin: "valores por defecto de Tide"}
	for _, p := range []string{"config/tide/tide_server.json", "config/tide_server.json", "config/tide/server.json"} {
		data, err := in.Files.ReadFile(p)
		if err != nil {
			continue
		}
		var doc map[string]any
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		general, _ := doc["general"].(map[string]any)
		if general == nil {
			general = doc
		}
		if v, ok := general["crateWeight"].(float64); ok {
			c.crateWeight = v
		}
		if v, ok := general["overrideVanillaRod"].(bool); ok {
			c.overrideVanilla = v
		}
		c.origin = p
		break
	}
	return c
}

func (Discoverer) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	cfg := readConfig(in)
	seasons := in.Target.Mods["sereneseasons"] || in.Target.Mods["eclipticseasons"] || in.Target.Mods["seasons"]
	r := reader{in: in, seasons: seasons}

	var fish, loot, crates []*fishing.Entry
	for _, id := range in.Resources.IDs(resources.TypeTideFish) {
		var f struct {
			Fish       string            `json:"fish"`
			Associated []string          `json:"associated_mods"`
			Conditions []json.RawMessage `json:"conditions"`
			Modifiers  []rawModifier     `json:"modifiers"`
			Weight     float64           `json:"selection_weight"`
			Profile    struct {
				Rarity string `json:"rarity"`
			} `json:"journal_profile"`
		}
		raw, ok := r.read(resources.TypeTideFish, id, &f)
		if !ok || !r.mods(f.Associated) || !fishing.ConditionsMet(raw, in.Target.Mods) {
			continue
		}
		item, err := domain.ParseResourceID(f.Fish)
		if err != nil {
			continue
		}
		e := &fishing.Entry{ID: id, Kind: fishing.KindFish, Item: item, Weight: f.Weight, Rarity: f.Profile.Rarity}
		r.conditions(e, f.Conditions)
		for _, m := range f.Modifiers {
			switch strings.TrimPrefix(m.Type, "tide:") {
			case "temperature":
				e.Temperature = &fishing.Temperature{Preferred: m.Preferred, Tolerance: m.Tolerance}
			case "conditional":
				tmp := &fishing.Entry{}
				r.conditions(tmp, []json.RawMessage{m.If})
				for _, c := range tmp.Conditions {
					c.Text = fmt.Sprintf("peso ×%g si %s", m.Multiplier, c.Text)
					e.Conditions = append(e.Conditions, c)
				}
			}
		}
		fish = append(fish, e)
	}
	for _, id := range in.Resources.IDs(resources.TypeTideLoot) {
		var f struct {
			Associated []string          `json:"associated_mods"`
			Conditions []json.RawMessage `json:"conditions"`
			Table      string            `json:"loot_table"`
			Weight     float64           `json:"weight"`
		}
		raw, ok := r.read(resources.TypeTideLoot, id, &f)
		if !ok || !r.mods(f.Associated) || !fishing.ConditionsMet(raw, in.Target.Mods) {
			continue
		}
		table, err := domain.ParseResourceID(f.Table)
		if err != nil {
			continue
		}
		e := &fishing.Entry{ID: id, Kind: fishing.KindLoot, Table: table, Weight: f.Weight}
		r.conditions(e, f.Conditions)
		loot = append(loot, e)
	}
	for _, id := range in.Resources.IDs(resources.TypeTideCrate) {
		var f struct {
			Associated []string          `json:"associated_mods"`
			Conditions []json.RawMessage `json:"conditions"`
			Table      string            `json:"loot_table"`
			Weight     *float64          `json:"weight"`
			Block      struct {
				State struct {
					Name string `json:"Name"`
				} `json:"state"`
			} `json:"block"`
		}
		raw, ok := r.read(resources.TypeTideCrate, id, &f)
		if !ok || !r.mods(f.Associated) || !fishing.ConditionsMet(raw, in.Target.Mods) {
			continue
		}
		table, err := domain.ParseResourceID(f.Table)
		if err != nil {
			continue
		}
		e := &fishing.Entry{ID: id, Kind: fishing.KindCrate, Table: table, Weight: 1}
		if f.Weight != nil {
			e.Weight = *f.Weight
		}
		if b, err := domain.ParseResourceID(f.Block.State.Name); err == nil {
			e.Block = b
		}
		r.conditions(e, f.Conditions)
		crates = append(crates, e)
	}
	if len(fish)+len(loot)+len(crates) == 0 {
		return nil
	}

	// Their loot tables are fishing loot, read straight from Tide's data.
	for _, e := range append(append([]*fishing.Entry(nil), loot...), crates...) {
		src := domain.LootSource{LootTable: e.Table, Kind: domain.KindFishing, Confidence: domain.ConfidenceExact,
			Evidence: []domain.Evidence{{DiscoveredBy: ID, Detail: "pesca de Tide: " + e.ID.String()}}}
		if e.Kind == fishing.KindCrate {
			src.Container = e.Block.String()
		}
		out.Add(src)
	}

	src := &fishing.Source{
		Mod: ID, Name: "Tide", Order: 20, ReplacesVanilla: cfg.overrideVanilla,
		Intro: []string{
			fmt.Sprintf("Se pesca con las cañas de Tide%s. En cada captura se elige, según su peso, entre los peces (peso %g), las cajas (peso %g, si alguna cabe) y los botines cuyas condiciones se cumplen.",
				map[bool]string{true: " (también reemplaza a la caña vanilla)", false: ""}[cfg.overrideVanilla], FishWeight, cfg.crateWeight),
			"Si sale «peces», se elige un pez por su peso; muchos prefieren una temperatura y pierden peso en biomas más fríos o más cálidos. La suerte (Suerte del mar, cebos) sube cajas y botines.",
			"Configuración: " + cfg.origin + ".",
		},
	}
	src.Entries = append(append(append(src.Entries, fish...), loot...), crates...)
	for _, b := range fishing.Biomes(in) {
		if bc := evaluate(fish, loot, crates, cfg, b); bc != nil {
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
	for _, e := range src.Entries {
		if !placed[e] {
			src.Unplaced = append(src.Unplaced, e)
		}
	}
	out.Attach(fishing.ExtraPrefix+ID, src)
	return nil
}

type reader struct {
	in      discovery.Input
	seasons bool
}

func (r reader) read(typ string, id domain.ResourceID, v any) (json.RawMessage, bool) {
	var raw json.RawMessage
	ok, err := r.in.Resources.ReadJSON(typ, id, &raw)
	if err == nil && ok {
		err = json.Unmarshal(raw, v)
	}
	if err != nil {
		r.in.Diagnostics.Add(domain.LevelWarning, "discovery", id.String(), "Tide: %v", err)
		return nil, false
	}
	return raw, ok
}

// mods checks associated_mods (Tide also accepts "-" spelled as "_").
func (r reader) mods(list []string) bool {
	for _, m := range list {
		if !r.in.Target.Mods[m] && !r.in.Target.Mods[strings.ReplaceAll(m, "-", "_")] {
			return false
		}
	}
	return true
}

func ids(list []string) []domain.ResourceID {
	var out []domain.ResourceID
	for _, s := range list {
		if id, err := domain.ParseResourceID(strings.TrimPrefix(s, "#")); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// conditions turns Tide conditions into the common model. Biome, dimension
// and fluid conditions decide where the entry applies; the rest is text.
func (r reader) conditions(e *fishing.Entry, list []json.RawMessage) {
	for _, raw := range list {
		var c rawCond
		if json.Unmarshal(raw, &c) != nil {
			continue
		}
		switch strings.TrimPrefix(c.Type, "tide:") {
		case "found_in":
			e.Biomes = append(e.Biomes, fishing.BiomeRule{Any: c.Biomes})
		case "freshwater":
			e.Biomes = append(e.Biomes, fishing.BiomeRule{None: []string{"#tide:is_saltwater"}})
			e.Conditions = append(e.Conditions, fishing.Cond{Text: "agua dulce"})
		case "saltwater":
			e.Biomes = append(e.Biomes, fishing.BiomeRule{Any: []string{"#tide:is_saltwater"}})
			e.Conditions = append(e.Conditions, fishing.Cond{Text: "agua salada"})
		case "dimension":
			e.Dimensions = append(e.Dimensions, ids(c.Dimensions)...)
		case "fluid":
			e.Fluids = append(e.Fluids, fishing.Fluid(c.Fluid))
		case "not":
			var inner rawCond
			if json.Unmarshal(c.A, &inner) == nil && strings.TrimPrefix(inner.Type, "tide:") == "found_in" {
				e.Biomes = append(e.Biomes, fishing.BiomeRule{None: inner.Biomes})
				continue
			}
			tmp := &fishing.Entry{}
			r.conditions(tmp, []json.RawMessage{c.A})
			for _, x := range tmp.Conditions {
				x.Text = "no " + x.Text
				e.Conditions = append(e.Conditions, x)
			}
		case "either":
			a, b := &fishing.Entry{}, &fishing.Entry{}
			r.conditions(a, []json.RawMessage{c.A})
			r.conditions(b, []json.RawMessage{c.B})
			e.Conditions = append(e.Conditions, fishing.Cond{Text: describe(a) + " o " + describe(b)})
		default:
			if cond, ok := r.text(c); ok {
				e.Conditions = append(e.Conditions, cond)
			}
		}
	}
}

// describe summarises an entry's conditions for "either".
func describe(e *fishing.Entry) string {
	var parts []string
	readable := func(s string) string {
		_, path, _ := strings.Cut(strings.TrimPrefix(s, "#"), ":")
		return names.Humanize(path)
	}
	for _, rule := range e.Biomes {
		var bs []string
		for _, b := range rule.Any {
			bs = append(bs, readable(b))
		}
		if len(bs) > 0 {
			parts = append(parts, "en "+strings.Join(bs, ", "))
		}
	}
	for _, c := range e.Conditions {
		t := c.Text
		var refs []string
		for _, ref := range c.Refs {
			refs = append(refs, readable(ref.String()))
		}
		if len(refs) > 0 {
			t += " " + strings.Join(refs, ", ")
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, " y ")
}

func (r reader) text(c rawCond) (fishing.Cond, bool) {
	switch strings.TrimPrefix(c.Type, "tide:") {
	case "above":
		if c.Y != nil {
			return fishing.Cond{Text: fmt.Sprintf("por encima de Y=%d", *c.Y)}, true
		}
	case "below":
		if c.Y != nil {
			return fishing.Cond{Text: fmt.Sprintf("por debajo de Y=%d", *c.Y)}, true
		}
	case "depth_range":
		if c.MinY != nil && c.MaxY != nil {
			return fishing.Cond{Text: fmt.Sprintf("altura entre Y=%d y Y=%d", *c.MinY, *c.MaxY)}, true
		}
	case "found_in_structures":
		return fishing.Cond{Text: "cerca de", Refs: ids(c.Structures), RefKind: "structure"}, true
	case "time_of_day":
		var parts []string
		for _, rg := range c.Ranges {
			parts = append(parts, "de "+fishing.Clock(rg.Min)+" a "+fishing.Clock(rg.Max))
		}
		return fishing.Cond{Text: "solo " + strings.Join(parts, " o ")}, true
	case "moon_phase":
		var parts []string
		for _, p := range c.AnyOf {
			parts = append(parts, fishing.MoonPhase(p))
		}
		return fishing.Cond{Text: "solo con " + strings.Join(parts, " o ")}, true
	case "weather":
		var parts []string
		for _, w := range c.Types {
			parts = append(parts, fishing.Weather(w))
		}
		return fishing.Cond{Text: "solo con clima: " + strings.Join(parts, " o ")}, true
	case "seasons":
		var parts []string
		for _, s := range c.Seasons {
			parts = append(parts, fishing.Season(s))
		}
		return fishing.Cond{Text: "solo en: " + strings.Join(parts, ", "), Inactive: !r.seasons}, true
	case "luck":
		if c.Min != nil {
			return fishing.Cond{Text: fmt.Sprintf("con suerte de pesca %d o más", *c.Min)}, true
		}
	case "open_water":
		if c.Open != nil && *c.Open {
			return fishing.Cond{Text: "en aguas abiertas"}, true
		}
		return fishing.Cond{Text: "fuera de aguas abiertas"}, true
	case "block_nearby":
		return fishing.Cond{Text: fmt.Sprintf("a %d bloques o menos de", c.Radius), Refs: ids([]string{c.Blocks}), RefKind: "blocktag"}, true
	case "has_enchantments":
		return fishing.Cond{Text: "con la caña encantada"}, true
	}
	return fishing.Cond{}, false
}

// evaluate computes the chances in a biome, grouped by fluid.
func evaluate(fish, loot, crates []*fishing.Entry, cfg config, b fishing.Biome) *fishing.BiomeCatches {
	bc := &fishing.BiomeCatches{Biome: b.ID, Dimension: b.Dimension}
	fits := func(e *fishing.Entry, fluid string) bool {
		return e.InDimension(b) && e.MatchesBiome(b) && e.HasFluid(fluid)
	}
	for _, fluid := range []string{fishing.FluidWater, fishing.FluidLava, fishing.FluidVoid} {
		g := &fishing.Group{Title: "Peces, pescando en " + fluid, Fluid: fluid,
			Note: "Probabilidad entre los peces de este bioma, ya ajustada por la temperatura del bioma. La altura, la hora y el clima pueden descartar algunos, y entonces suben los demás."}
		total := 0.0
		for _, e := range fish {
			if !fits(e, fluid) {
				continue
			}
			w := e.Weight * e.Temperature.Scale(b.Temperature)
			if w <= 0 {
				continue
			}
			total += w
			g.Catches = append(g.Catches, fishing.Catch{Entry: e, Weight: w})
		}
		for i := range g.Catches {
			g.Catches[i].Chance = g.Catches[i].Weight / total
		}
		fishing.SortCatches(g.Catches)

		other := &fishing.Group{Title: "Cajas y botín, pescando en " + fluid, Fluid: fluid}
		crateTotal := 0.0
		for _, e := range crates {
			if fits(e, fluid) && e.Weight > 0 {
				crateTotal += e.Weight
			}
		}
		for _, e := range crates {
			if fits(e, fluid) && e.Weight > 0 {
				other.Catches = append(other.Catches, fishing.Catch{Entry: e, Weight: e.Weight,
					Note: fmt.Sprintf("el %s de las cajas", pct(e.Weight/crateTotal))})
			}
		}
		for _, e := range loot {
			if fits(e, fluid) && e.Weight > 0 {
				other.Catches = append(other.Catches, fishing.Catch{Entry: e, Weight: e.Weight, Note: fmt.Sprintf("peso %g", e.Weight)})
			}
		}
		sort.SliceStable(other.Catches, func(i, j int) bool { return other.Catches[i].Weight > other.Catches[j].Weight })
		if len(other.Catches) > 0 {
			other.Note = fmt.Sprintf("Compiten con los peces (peso %g)%s. Los botines dependen sobre todo de la altura: revisa sus condiciones.",
				FishWeight, map[bool]string{true: fmt.Sprintf(" y con las cajas (peso %g en total)", cfg.crateWeight), false: ""}[crateTotal > 0])
		}
		if len(g.Catches) > 0 {
			bc.Groups = append(bc.Groups, g)
		}
		if len(other.Catches) > 0 && len(g.Catches) > 0 {
			bc.Groups = append(bc.Groups, other)
		}
	}
	if len(bc.Groups) == 0 {
		return nil
	}
	return bc
}

func pct(p float64) string {
	return strings.Replace(fmt.Sprintf("%.0f %%", p*100), ".", ",", 1)
}
