package lostcities

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// Model is the Lost Cities hierarchy as the site shows it:
// profile → world style → city styles (by biome) → buildings → parts →
// containers → loot conditions → loot tables. It follows Lost Cities 1.20
// (cityassets/*.java): city styles inherit their parent's selectors, names
// without a namespace belong to "lostcities".
type Model struct {
	Profiles    []ActiveProfile
	WorldStyles []*WorldStyle
	CityStyles  map[domain.ResourceID]*CityStyle
	Buildings   map[domain.ResourceID]*Building
	Conditions  map[domain.ResourceID]*Condition
}

// ActiveProfile is a dimension with cities.
type ActiveProfile struct {
	Dimension  string
	Profile    string
	WorldStyle domain.ResourceID
	Origin     string
}

// BiomeRule is Lost Cities' biome matcher: if_any, if_all, excluding.
type BiomeRule struct {
	IfAny, IfAll, Excluding []string
}

// Empty reports whether the rule matches every biome.
func (b BiomeRule) Empty() bool { return len(b.IfAny)+len(b.IfAll)+len(b.Excluding) == 0 }

// WorldStyle chooses city styles by biome and places scattered buildings.
type WorldStyle struct {
	ID         domain.ResourceID
	CityStyles []StyleRule
	Scattered  []ScatteredRule
}

// StyleRule is one city style of a world style.
type StyleRule struct {
	Style  *CityStyle
	Factor float64
	Biomes BiomeRule
}

// ScatteredRule is a building placed outside cities.
type ScatteredRule struct {
	Name      domain.ResourceID
	Weight    float64
	Biomes    BiomeRule
	Buildings []*Building
}

// CityStyle lists the buildings a city of that style can have.
type CityStyle struct {
	ID        domain.ResourceID
	Inherits  []domain.ResourceID
	Buildings []Weighted
	Multi     []Weighted
}

// Weighted is a building picked with a factor.
type Weighted struct {
	Building *Building
	Factor   float64
	Share    float64 // factor / total of its list
}

// Building is a building or a multi-building (a grid of buildings).
type Building struct {
	ID       domain.ResourceID
	Multi    bool
	Children []*Building // for multi-buildings
	Parts    []*Part
}

// Part is a floor (or section) of a building.
type Part struct {
	ID         domain.ResourceID
	Containers []PartContainer
}

// PartContainer is one kind of loot container in a part.
type PartContainer struct {
	Block     string
	Condition *Condition
	Count     int
}

// Condition picks a loot table by weight for each container placed.
type Condition struct {
	ID     domain.ResourceID
	Values []conditionValue
	Total  float64
}

// ValuesFor returns the values that apply to a part and building, with
// their share among those values.
func (c *Condition) ValuesFor(part, building string) []ConditionChoice {
	var applicable []conditionValue
	total := 0.0
	for _, v := range c.Values {
		if v.InPart != "" && !sameName(v.InPart, part) {
			continue
		}
		if v.InBuilding != "" && !sameName(v.InBuilding, building) {
			continue
		}
		applicable = append(applicable, v)
		total += v.Factor
	}
	var out []ConditionChoice
	for _, v := range applicable {
		id, err := domain.ParseResourceID(v.Value)
		if err != nil || total <= 0 {
			continue
		}
		out = append(out, ConditionChoice{Table: id, Share: v.Factor / total, Floors: v.Range, Biome: v.InBiome})
	}
	return out
}

// ConditionChoice is a loot table a container can get.
type ConditionChoice struct {
	Table  domain.ResourceID
	Share  float64
	Floors string // "min,max" floor range, "" for any
	Biome  string // biome restriction, "" for any
}

// sameName compares asset names with or without namespace.
func sameName(a, b string) bool {
	strip := func(s string) string {
		_, rest, ok := strings.Cut(s, ":")
		if ok {
			return rest
		}
		return s
	}
	return strip(a) == strip(b)
}

// HasContent reports whether there is anything to show.
func (m *Model) HasContent() bool { return m != nil && len(m.WorldStyles) > 0 }

// ref resolves an asset name: "name" → lostcities:name, "ns:name" as is.
func ref(name, defaultNS string) (domain.ResourceID, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ResourceID{}, false
	}
	if !strings.Contains(name, ":") {
		return domain.ResourceID{Namespace: defaultNS, Path: name}, true
	}
	id, err := domain.ParseResourceID(name)
	return id, err == nil
}

type reader struct {
	in        discovery.Input
	model     *Model
	parts     map[domain.ResourceID]*Part
	palettes  map[domain.ResourceID]map[string]paletteEntry
	globalPal map[string]paletteEntry
}

type paletteEntry struct {
	Block string
	Loot  string
}

// BuildModel reads every Lost Cities asset reachable from the active
// profiles. It returns nil when the modpack has no Lost Cities world style.
func BuildModel(in discovery.Input) *Model {
	if len(in.Resources.IDs(typeWorldStyle)) == 0 {
		return nil
	}
	r := &reader{
		in: in,
		model: &Model{
			CityStyles: map[domain.ResourceID]*CityStyle{},
			Buildings:  map[domain.ResourceID]*Building{},
			Conditions: map[domain.ResourceID]*Condition{},
		},
		parts:    map[domain.ResourceID]*Part{},
		palettes: map[domain.ResourceID]map[string]paletteEntry{},
	}
	r.globalPal = r.globalPalette()

	profiles := ReadProfiles(in)
	seen := map[domain.ResourceID]bool{}
	dims := make([]string, 0)
	for d := range profiles.Active() {
		dims = append(dims, d)
	}
	sort.Strings(dims)
	for _, dim := range dims {
		prof := profiles.Active()[dim]
		ws := r.profileWorldStyle(prof)
		origin := profiles.Origin
		if dim != "minecraft:overworld" {
			origin = "configuración común de Lost Cities"
		}
		r.model.Profiles = append(r.model.Profiles, ActiveProfile{Dimension: dim, Profile: prof, WorldStyle: ws, Origin: origin})
		if !seen[ws] {
			seen[ws] = true
			if w := r.worldStyle(ws); w != nil {
				r.model.WorldStyles = append(r.model.WorldStyles, w)
			}
		}
	}
	// Without an active profile (or before the world exists), show every
	// world style the modpack defines.
	if len(r.model.WorldStyles) == 0 {
		for _, id := range in.Resources.IDs(typeWorldStyle) {
			if w := r.worldStyle(id); w != nil {
				r.model.WorldStyles = append(r.model.WorldStyles, w)
			}
		}
	}
	return r.model
}

const (
	typeWorldStyle = "lostcities/worldstyles"
	typeCityStyle  = "lostcities/citystyles"
	typeBuilding   = "lostcities/buildings"
	typeMulti      = "lostcities/multibuildings"
	typeScattered  = "lostcities/scattered"
	typeStyle      = "lostcities/styles"
)

// profileWorldStyle reads the profile's worldStyle (config/lostcities/
// profiles/<name>.json); built-in profiles use "standard".
func (r *reader) profileWorldStyle(profile string) domain.ResourceID {
	data, err := r.in.Files.ReadFile("config/lostcities/profiles/" + profile + ".json")
	if err == nil {
		var doc map[string]any
		if json.Unmarshal(data, &doc) == nil {
			flat := map[string]any{}
			flatten(doc, flat) // the section changed between versions
			if s, ok := flat["worldStyle"].(string); ok {
				if id, ok := ref(s, "lostcities"); ok {
					return id
				}
			}
		}
	}
	return domain.ResourceID{Namespace: "lostcities", Path: "standard"}
}

func (r *reader) read(typ string, id domain.ResourceID, v any) bool {
	ok, err := r.in.Resources.ReadJSON(typ, id, v)
	if err != nil {
		r.in.Diagnostics.Add(domain.LevelWarning, "discovery", id.String(), "Lost Cities: %v", err)
		return false
	}
	return ok
}

type rawBiomes struct {
	IfAny     []string `json:"if_any"`
	IfAll     []string `json:"if_all"`
	Excluding []string `json:"excluding"`
}

func (b *rawBiomes) rule() BiomeRule {
	if b == nil {
		return BiomeRule{}
	}
	return BiomeRule{IfAny: b.IfAny, IfAll: b.IfAll, Excluding: b.Excluding}
}

func (r *reader) worldStyle(id domain.ResourceID) *WorldStyle {
	var raw struct {
		CityStyles []struct {
			Factor    float64    `json:"factor"`
			CityStyle string     `json:"citystyle"`
			Biomes    *rawBiomes `json:"biomes"`
		} `json:"citystyles"`
		Scattered struct {
			List []struct {
				Name   string     `json:"name"`
				Weight float64    `json:"weight"`
				Biomes *rawBiomes `json:"biomes"`
			} `json:"list"`
		} `json:"scattered"`
	}
	if !r.read(typeWorldStyle, id, &raw) {
		return nil
	}
	w := &WorldStyle{ID: id}
	for _, c := range raw.CityStyles {
		cid, ok := ref(c.CityStyle, id.Namespace)
		if !ok {
			continue
		}
		if cs := r.cityStyle(cid, 0); cs != nil {
			w.CityStyles = append(w.CityStyles, StyleRule{Style: cs, Factor: c.Factor, Biomes: c.Biomes.rule()})
		}
	}
	for _, s := range raw.Scattered.List {
		sid, ok := ref(s.Name, id.Namespace)
		if !ok {
			continue
		}
		rule := ScatteredRule{Name: sid, Weight: s.Weight, Biomes: s.Biomes.rule()}
		var sc struct {
			Buildings     []string `json:"buildings"`
			Multibuilding string   `json:"multibuilding"`
		}
		if r.read(typeScattered, sid, &sc) {
			for _, b := range sc.Buildings {
				if bid, ok := ref(b, sid.Namespace); ok {
					if bl := r.building(bid); bl != nil {
						rule.Buildings = append(rule.Buildings, bl)
					}
				}
			}
			if mid, ok := ref(sc.Multibuilding, sid.Namespace); ok {
				if bl := r.multi(mid); bl != nil {
					rule.Buildings = append(rule.Buildings, bl)
				}
			}
		}
		w.Scattered = append(w.Scattered, rule)
	}
	return w
}

type rawSelector struct {
	Factor float64 `json:"factor"`
	Value  string  `json:"value"`
}

func (r *reader) cityStyle(id domain.ResourceID, depth int) *CityStyle {
	if cs, ok := r.model.CityStyles[id]; ok {
		return cs
	}
	if depth > 10 {
		return nil
	}
	var raw struct {
		Inherit   string `json:"inherit"`
		Selectors struct {
			Buildings      []rawSelector `json:"buildings"`
			Multibuildings []rawSelector `json:"multibuildings"`
		} `json:"selectors"`
	}
	if !r.read(typeCityStyle, id, &raw) {
		return nil
	}
	cs := &CityStyle{ID: id}
	r.model.CityStyles[id] = cs
	add := func(list []rawSelector, multi bool) {
		for _, s := range list {
			bid, ok := ref(s.Value, id.Namespace)
			if !ok {
				continue
			}
			var b *Building
			if multi {
				b = r.multi(bid)
			} else {
				b = r.building(bid)
			}
			if b == nil {
				continue
			}
			w := Weighted{Building: b, Factor: s.Factor}
			if multi {
				cs.Multi = append(cs.Multi, w)
			} else {
				cs.Buildings = append(cs.Buildings, w)
			}
		}
	}
	add(raw.Selectors.Buildings, false)
	add(raw.Selectors.Multibuildings, true)
	if pid, ok := ref(raw.Inherit, id.Namespace); ok {
		if parent := r.cityStyle(pid, depth+1); parent != nil {
			cs.Inherits = append([]domain.ResourceID{pid}, parent.Inherits...)
			cs.Buildings = append(cs.Buildings, parent.Buildings...)
			cs.Multi = append(cs.Multi, parent.Multi...)
		}
	}
	shares(cs.Buildings)
	shares(cs.Multi)
	return cs
}

func shares(list []Weighted) {
	total := 0.0
	for _, w := range list {
		total += w.Factor
	}
	for i := range list {
		if total > 0 {
			list[i].Share = list[i].Factor / total
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Factor > list[j].Factor })
}

func (r *reader) multi(id domain.ResourceID) *Building {
	if b, ok := r.model.Buildings[id]; ok {
		return b
	}
	var raw struct {
		Buildings [][]string `json:"buildings"`
	}
	if !r.read(typeMulti, id, &raw) {
		return nil
	}
	b := &Building{ID: id, Multi: true}
	r.model.Buildings[id] = b
	seen := map[domain.ResourceID]bool{}
	for _, row := range raw.Buildings {
		for _, name := range row {
			if cid, ok := ref(name, id.Namespace); ok && !seen[cid] {
				seen[cid] = true
				if child := r.building(cid); child != nil {
					b.Children = append(b.Children, child)
				}
			}
		}
	}
	return b
}

func (r *reader) building(id domain.ResourceID) *Building {
	if b, ok := r.model.Buildings[id]; ok {
		return b
	}
	var raw struct {
		Palette json.RawMessage `json:"palette"`
		Parts   []struct {
			Part string `json:"part"`
		} `json:"parts"`
		Parts2 []struct {
			Part string `json:"part"`
		} `json:"parts2"`
	}
	if !r.read(typeBuilding, id, &raw) {
		return nil
	}
	b := &Building{ID: id}
	r.model.Buildings[id] = b
	pal := r.paletteRef(raw.Palette, id.Namespace)
	seen := map[domain.ResourceID]bool{}
	for _, p := range append(raw.Parts, raw.Parts2...) {
		pid, ok := ref(p.Part, id.Namespace)
		if !ok || seen[pid] {
			continue
		}
		seen[pid] = true
		if part := r.part(pid, pal); part != nil {
			b.Parts = append(b.Parts, part)
		}
	}
	return b
}

func (r *reader) part(id domain.ResourceID, buildingPal map[string]paletteEntry) *Part {
	if p, ok := r.parts[id]; ok {
		return p
	}
	var raw struct {
		RefPalette string          `json:"refpalette"`
		Palette    json.RawMessage `json:"palette"`
		Slices     [][]string      `json:"slices"`
	}
	if !r.read(resources.TypeLostCitiesParts, id, &raw) {
		return nil
	}
	p := &Part{ID: id}
	r.parts[id] = p
	local := r.paletteRef(raw.Palette, id.Namespace)
	var refPal map[string]paletteEntry
	if pid, ok := ref(raw.RefPalette, id.Namespace); ok {
		refPal = r.palette(pid)
	}
	counts := map[string]int{}
	for _, slice := range raw.Slices {
		for _, row := range slice {
			for _, ch := range row {
				counts[string(ch)]++
			}
		}
	}
	chars := make([]string, 0, len(counts))
	for c := range counts {
		chars = append(chars, c)
	}
	sort.Strings(chars)
	for _, c := range chars {
		e, ok := local[c]
		if !ok {
			e, ok = refPal[c]
		}
		if !ok {
			e, ok = buildingPal[c]
		}
		if !ok {
			e, ok = r.globalPal[c]
		}
		if !ok || e.Loot == "" {
			continue
		}
		cid, ok := ref(e.Loot, id.Namespace)
		if !ok {
			continue
		}
		cond := r.condition(cid)
		if cond == nil {
			continue
		}
		p.Containers = append(p.Containers, PartContainer{Block: e.Block, Condition: cond, Count: counts[c]})
	}
	return p
}

// paletteRef reads a "palette" field: the name of a palette, an inline list
// of entries or an object {"palette": [...]}.
func (r *reader) paletteRef(raw json.RawMessage, ns string) map[string]paletteEntry {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil
	}
	switch raw[0] {
	case '"':
		var name string
		if json.Unmarshal(raw, &name) == nil {
			if pid, ok := ref(name, ns); ok {
				return r.palette(pid)
			}
		}
	case '[':
		return parsePalette(raw)
	case '{':
		var inner struct {
			Palette json.RawMessage `json:"palette"`
		}
		if json.Unmarshal(raw, &inner) == nil && len(inner.Palette) > 0 && inner.Palette[0] == '[' {
			return parsePalette(inner.Palette)
		}
	}
	return nil
}

func parsePalette(raw json.RawMessage) map[string]paletteEntry {
	var entries []struct {
		Char  string `json:"char"`
		Block string `json:"block"`
		Loot  string `json:"loot"`
	}
	if json.Unmarshal(raw, &entries) != nil {
		return nil
	}
	out := map[string]paletteEntry{}
	for _, e := range entries {
		block, _, _ := strings.Cut(e.Block, "[")
		out[e.Char] = paletteEntry{Block: block, Loot: e.Loot}
	}
	return out
}

func (r *reader) palette(id domain.ResourceID) map[string]paletteEntry {
	if p, ok := r.palettes[id]; ok {
		return p
	}
	var raw struct {
		Palette json.RawMessage `json:"palette"`
	}
	var p map[string]paletteEntry
	if r.read(resources.TypeLostCitiesPal, id, &raw) {
		p = parsePalette(raw.Palette)
	}
	r.palettes[id] = p
	return p
}

// globalPalette merges the palettes the city styles draw from (styles/*),
// which give chars such as "C" their meaning in parts without own palette.
func (r *reader) globalPalette() map[string]paletteEntry {
	out := map[string]paletteEntry{}
	for _, sid := range r.in.Resources.IDs(typeStyle) {
		var raw struct {
			RandomPalettes [][]struct {
				Palette string `json:"palette"`
			} `json:"randompalettes"`
		}
		if !r.read(typeStyle, sid, &raw) {
			continue
		}
		for _, group := range raw.RandomPalettes {
			for _, e := range group {
				if pid, ok := ref(e.Palette, sid.Namespace); ok {
					for c, entry := range r.palette(pid) {
						if _, set := out[c]; !set || (entry.Loot != "" && out[c].Loot == "") {
							out[c] = entry
						}
					}
				}
			}
		}
	}
	return out
}

func (r *reader) condition(id domain.ResourceID) *Condition {
	if c, ok := r.model.Conditions[id]; ok {
		return c
	}
	var raw struct {
		Values []conditionValue `json:"values"`
	}
	if !r.read(resources.TypeLostCitiesCond, id, &raw) {
		r.model.Conditions[id] = nil
		return nil
	}
	c := &Condition{ID: id, Values: raw.Values}
	for _, v := range raw.Values {
		c.Total += v.Factor
	}
	r.model.Conditions[id] = c
	return c
}

// HasLoot reports whether the building (or any child) holds loot containers.
func (b *Building) HasLoot() bool {
	for _, p := range b.Parts {
		if len(p.Containers) > 0 {
			return true
		}
	}
	for _, c := range b.Children {
		if c.HasLoot() {
			return true
		}
	}
	return false
}
