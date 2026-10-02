package site

import (
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/fishing"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
)

// Fishing is the fishing tab: pesca › fuente › bioma › condiciones › captura.
type Fishing struct {
	Sources []*FishSource
}

// FishSource is a fishing system (a mod, or the vanilla rod).
type FishSource struct {
	Ref
	Intro      []string
	Warning    string
	Everywhere []FishGroup
	Biomes     []*FishBiome
	Entries    []*FishEntry
	Unplaced   []*FishEntry
}

// Everywhere reports whether the entry is possible in every biome of its source.
func (e *FishEntry) Everywhere() bool {
	return len(e.Source.Biomes) > 1 && len(e.Places) == len(e.Source.Biomes)
}

// FishBiome is what a source gives in a biome.
type FishBiome struct {
	Ref
	Source    *FishSource
	Biome     *Biome // page of the biome, if any
	Dimension string
	Groups    []FishGroup
	Count     int
	Top       []FishCatch
}

// FishGroup is a list of catches that compete with each other.
type FishGroup struct {
	Title, Note string
	Catches     []FishCatch
}

// FishCatch is an entry with its chance in a group.
type FishCatch struct {
	Entry  *FishEntry
	Chance float64
	Weight float64
	Bait   string
	Note   string
}

// FishEntry is a catch, ready to read.
type FishEntry struct {
	Name       string
	ID         domain.ResourceID
	Item       *Item
	Table      *Table
	Kind       string
	Rarity     string
	Fluids     string
	Dimensions string
	Biomes     []string
	Conditions []string
	Inactive   []string
	Baits      []string
	BaitOnly   bool
	Spawns     string
	Source     *FishSource
	Places     []*FishBiome
}

var kindNames = map[fishing.Kind]string{fishing.KindFish: "Pez", fishing.KindLoot: "Botín", fishing.KindCrate: "Caja"}

var rarityNames = map[string]string{"common": "Común", "uncommon": "Poco común", "rare": "Rara", "very_rare": "Muy rara", "epic": "Épica", "legendary": "Legendaria", "mythical": "Mítica", "golden": "Dorada", "special": "Especial"}

func buildFishing(res *analysis.Result, m *Model, namer *names.Namer, itemOf func(domain.ResourceID) *Item) {
	var sources []*fishing.Source
	for key, v := range res.Extras {
		if s, ok := v.(*fishing.Source); ok && strings.HasPrefix(key, fishing.ExtraPrefix) {
			sources = append(sources, s)
		}
	}
	if len(sources) == 0 {
		return
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Order < sources[j].Order })
	replaced := ""
	for _, s := range sources {
		if s.ReplacesVanilla {
			replaced = s.Name
		}
	}

	biomes := map[domain.ResourceID]*Biome{}
	for _, b := range m.Biomes {
		biomes[b.ID] = b
	}
	tables := map[domain.ResourceID]*Table{}
	for _, t := range m.Tables {
		tables[t.ID] = t
	}
	idName := func(kind string, id domain.ResourceID) string {
		switch kind {
		case "item":
			return namer.Item(id)
		case "biome":
			return namer.Biome(id)
		case "structure":
			return namer.Structure(id)
		case "dimension":
			return namer.Asset("dimension", id)
		}
		return names.Humanize(id.Path)
	}
	ruleText := func(r fishing.BiomeRule) []string {
		term := func(s string) string {
			if tag, ok := strings.CutPrefix(s, "#"); ok {
				if id, err := domain.ParseResourceID(tag); err == nil {
					return "grupo «" + namer.BiomeTag(id) + "»"
				}
				return s
			}
			if id, err := domain.ParseResourceID(s); err == nil {
				return namer.Biome(id)
			}
			return s
		}
		var out []string
		if len(r.Any) > 0 {
			var parts []string
			for _, s := range r.Any {
				parts = append(parts, term(s))
			}
			out = append(out, "en "+strings.Join(parts, ", "))
		}
		if len(r.None) > 0 {
			var parts []string
			for _, s := range r.None {
				parts = append(parts, term(s))
			}
			out = append(out, "excepto "+strings.Join(parts, ", "))
		}
		return out
	}

	tab := &Fishing{}
	for _, s := range sources {
		fs := &FishSource{Ref: Ref{ID: domain.ResourceID{Namespace: s.Mod, Path: s.Mod}, Name: s.Name, URL: "pesca/" + strings.SplitN(idPath(domain.ResourceID{Namespace: s.Mod, Path: s.Mod}), "/", 2)[0] + "/"}, Intro: s.Intro}
		if s.Mod == "minecraft" && replaced != "" {
			fs.Warning = replaced + " reemplaza la caña vanilla en este modpack: esta tabla solo se usa con cañas de otros mods que la respeten."
		}
		entries := map[*fishing.Entry]*FishEntry{}
		for _, e := range s.Entries {
			fe := &FishEntry{ID: e.ID, Kind: kindNames[e.Kind], Rarity: rarityNames[e.Rarity], BaitOnly: e.BaitOnly, Source: fs}
			if r := strings.ToLower(strings.ReplaceAll(e.Rarity, " ", "_")); rarityNames[r] != "" {
				fe.Rarity = rarityNames[r]
			} else if e.Rarity != "" {
				fe.Rarity = names.Humanize(e.Rarity)
			}
			switch e.Kind {
			case fishing.KindFish:
				fe.Item = itemOf(e.Item)
				fe.Name = fe.Item.Name
				fe.Item.Fishing = append(fe.Item.Fishing, fe)
			default:
				fe.Table = tables[e.Table]
				fe.Name = names.Humanize(e.Table.Path)
				if fe.Table != nil {
					fe.Name = fe.Table.Name
				}
				if e.Kind == fishing.KindCrate && !e.Block.IsZero() {
					fe.Name = namer.Item(e.Block)
				}
			}
			fe.Fluids = strings.Join(e.Fluids, ", ")
			var dims []string
			for _, d := range e.Dimensions {
				dims = append(dims, idName("dimension", d))
			}
			for _, d := range e.NotDimensions {
				dims = append(dims, "no en "+idName("dimension", d))
			}
			fe.Dimensions = strings.Join(dims, ", ")
			for _, r := range e.Biomes {
				fe.Biomes = append(fe.Biomes, ruleText(r)...)
			}
			for _, c := range e.Conditions {
				text := c.Text
				if c.Key != "" {
					if v, ok := namer.Text(c.Key); ok {
						text = v
					}
				}
				if len(c.Refs) > 0 {
					var refs []string
					for _, r := range c.Refs {
						refs = append(refs, idName(c.RefKind, r))
					}
					text += " " + strings.Join(refs, ", ")
				}
				if c.Inactive {
					fe.Inactive = append(fe.Inactive, text)
				} else {
					fe.Conditions = append(fe.Conditions, text)
				}
			}
			for _, b := range e.Baits {
				fe.Baits = append(fe.Baits, namer.Item(b.Item))
			}
			if !e.Spawns.IsZero() {
				fe.Spawns = namer.Entity(e.Spawns)
			}
			entries[e] = fe
			fs.Entries = append(fs.Entries, fe)
		}
		group := func(g *fishing.Group) FishGroup {
			out := FishGroup{Title: g.Title, Note: g.Note}
			for _, c := range g.Catches {
				fc := FishCatch{Entry: entries[c.Entry], Chance: c.Chance, Weight: c.Weight, Note: c.Note}
				if c.Bait != nil {
					fc.Bait = namer.Item(c.Bait.Item)
				}
				out.Catches = append(out.Catches, fc)
			}
			return out
		}
		for _, g := range s.Everywhere {
			fs.Everywhere = append(fs.Everywhere, group(g))
		}
		for _, bc := range s.Biomes {
			fb := &FishBiome{Source: fs, Biome: biomes[bc.Biome], Dimension: bc.Dimension}
			fb.Ref = Ref{ID: bc.Biome, Name: namer.Biome(bc.Biome), URL: fs.URL + idPath(bc.Biome) + "/"}
			seen := map[*FishEntry]bool{}
			for _, g := range bc.Groups {
				fg := group(g)
				fb.Groups = append(fb.Groups, fg)
				for _, c := range fg.Catches {
					if !seen[c.Entry] {
						seen[c.Entry] = true
						c.Entry.Places = append(c.Entry.Places, fb)
						fb.Count++
					}
				}
			}
			if len(fb.Groups) > 0 {
				for _, c := range fb.Groups[0].Catches {
					if len(fb.Top) == 3 {
						break
					}
					fb.Top = append(fb.Top, c)
				}
			}
			fs.Biomes = append(fs.Biomes, fb)
		}
		sortRefs(fs.Biomes, func(b *FishBiome) Ref { return b.Ref })
		for _, e := range s.Unplaced {
			fs.Unplaced = append(fs.Unplaced, entries[e])
		}
		sort.SliceStable(fs.Entries, func(i, j int) bool {
			a, b := fs.Entries[i], fs.Entries[j]
			if a.Kind != b.Kind {
				return a.Kind > b.Kind // Pez, Caja, Botín
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		})
		tab.Sources = append(tab.Sources, fs)
	}
	m.Fishing = tab
	m.Sections = append(m.Sections, Section{Nav: "pesca", Label: "Pesca", URL: "pesca/"})
}

func (r *renderer) renderFishing(m *Model) {
	if m.Fishing == nil {
		return
	}
	r.render("fish_index", "pesca/", "pesca", "Pesca", m, m.Fishing)
	for _, s := range m.Fishing.Sources {
		r.render("fish_source", s.URL, "pesca", "Pesca · "+s.Name, m, s)
		for _, b := range s.Biomes {
			r.render("fish_biome", b.URL, "pesca", b.Name+" · "+s.Name, m, b)
		}
	}
}
