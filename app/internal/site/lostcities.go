package site

import (
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/lostcities"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// Section is a dedicated tab of a mod. It only exists when the mod and its
// data are present, so the navigation never shows empty tabs.
type Section struct {
	Nav   string
	Label string
	URL   string
}

// LostCities is the Lost Cities tab, layer by layer:
// profiles → world styles → city styles → buildings → parts → containers → loot.
type LostCities struct {
	Status    domain.Status
	Profiles  []LCProfile
	Worlds    []*LCWorld
	Styles    []*LCStyle
	Buildings []*LCBuilding
}

// LCProfile is a dimension with cities.
type LCProfile struct {
	Dimension string
	Profile   string
	ProfileID string
	Origin    string
	World     *LCWorld
}

// LCWorld is a world style.
type LCWorld struct {
	Ref
	Rules     []LCRule
	Scattered []LCScattered
}

// LCRule is a city style allowed by a world style in some biomes.
type LCRule struct {
	Style  *LCStyle
	Factor float64
	Biomes LCBiomes
}

// LCScattered is a building placed outside cities.
type LCScattered struct {
	Name      string
	Weight    float64
	Biomes    LCBiomes
	Buildings []*LCBuilding
}

// LCBiomes is a biome rule ready to read.
type LCBiomes struct {
	Any, All, Excluding []LCBiomeTerm
}

// Everywhere reports whether the rule accepts every biome.
func (b LCBiomes) Everywhere() bool { return len(b.Any)+len(b.All)+len(b.Excluding) == 0 }

// LCBiomeTerm is a biome or a biome tag of a rule.
type LCBiomeTerm struct {
	Name    string
	ID      string
	Biome   *Biome   // page of the biome, if any
	Members []*Biome // biomes of a tag that have a page
	Tag     bool
}

// LCStyle is a city style.
type LCStyle struct {
	Ref
	Inherits  []*LCStyle
	Buildings []LCWeighted
	Multi     []LCWeighted
	Worlds    []*LCWorld
	// WithLoot counts the buildings (and multi-buildings) holding loot.
	WithLoot int
}

// LCWeighted is a building of a city style with its share.
type LCWeighted struct {
	Building *LCBuilding
	Share    float64
}

// LCBuilding is a building or a multi-building.
type LCBuilding struct {
	Ref
	Multi      bool
	Children   []*LCBuilding
	Parts      []LCPart
	Empty      int // parts without loot containers
	Styles     []*LCStyle
	Containers int
	Tables     []*Table // every loot table the building can hold
	Haul       []HaulItem
}

// LCPart is a floor or section of a building with loot containers.
type LCPart struct {
	Name       string
	ID         domain.ResourceID
	Containers []LCContainer
}

// LCContainer is a kind of container placed in a part.
type LCContainer struct {
	Block   string
	Count   int
	Choices []LCChoice
}

// LCChoice is a loot table a container can get.
type LCChoice struct {
	Table  *Table // nil if the table has no page
	Name   string
	Share  float64
	Detail string
}

// HasLoot reports whether the building or a child holds loot.
func (b *LCBuilding) HasLoot() bool { return b.Containers > 0 }

func buildLostCities(res *analysis.Result, m *Model, namer *names.Namer) {
	lm, ok := res.Extras[lostcities.ID].(*lostcities.Model)
	if !ok || !lm.HasContent() {
		return
	}
	lc := &LostCities{Status: res.Status(domain.Target{Kind: domain.TargetStructure, ID: lostcities.Owner.ID})}
	biomes := map[domain.ResourceID]*Biome{}
	for _, b := range m.Biomes {
		biomes[b.ID] = b
	}
	tables := map[domain.ResourceID]*Table{}
	for _, t := range m.Tables {
		tables[t.ID] = t
	}

	term := func(s string) LCBiomeTerm {
		if tag, isTag := strings.CutPrefix(s, "#"); isTag {
			id, err := domain.ParseResourceID(tag)
			if err != nil {
				return LCBiomeTerm{Name: s, ID: s, Tag: true}
			}
			t := LCBiomeTerm{Name: namer.BiomeTag(id), ID: s, Tag: true}
			for _, b := range res.Resources.Tag(resources.TypeBiomeTag, id) {
				if bm := biomes[b]; bm != nil {
					t.Members = append(t.Members, bm)
				}
			}
			sortRefs(t.Members, func(b *Biome) Ref { return b.Ref })
			return t
		}
		id, err := domain.ParseResourceID(s)
		if err != nil {
			return LCBiomeTerm{Name: s, ID: s}
		}
		return LCBiomeTerm{Name: namer.Biome(id), ID: s, Biome: biomes[id]}
	}
	rule := func(r lostcities.BiomeRule) LCBiomes {
		var out LCBiomes
		for _, s := range r.IfAny {
			out.Any = append(out.Any, term(s))
		}
		for _, s := range r.IfAll {
			out.All = append(out.All, term(s))
		}
		for _, s := range r.Excluding {
			out.Excluding = append(out.Excluding, term(s))
		}
		return out
	}

	buildings := map[*lostcities.Building]*LCBuilding{}
	var building func(b *lostcities.Building) *LCBuilding
	building = func(b *lostcities.Building) *LCBuilding {
		if x, ok := buildings[b]; ok {
			return x
		}
		x := &LCBuilding{Multi: b.Multi}
		x.Ref = Ref{ID: b.ID, Name: namer.Asset("building", b.ID), URL: "lostcities/edificios/" + idPath(b.ID) + "/"}
		buildings[b] = x
		h := newHaul()
		seenTable := map[*Table]bool{}
		addTable := func(t *Table) {
			if t != nil && !seenTable[t] {
				seenTable[t] = true
				x.Tables = append(x.Tables, t)
			}
		}
		for _, c := range b.Children {
			child := building(c)
			x.Children = append(x.Children, child)
			x.Containers += child.Containers
			h.merge(child.Haul)
			for _, t := range child.Tables {
				addTable(t)
			}
		}
		for _, p := range b.Parts {
			if len(p.Containers) == 0 {
				x.Empty++
				continue
			}
			part := LCPart{Name: namer.Asset("part", p.ID), ID: p.ID}
			for _, c := range p.Containers {
				box := LCContainer{Count: c.Count, Block: c.Block}
				if id, err := domain.ParseResourceID(c.Block); err == nil {
					box.Block = namer.Item(id)
				}
				for _, ch := range c.Condition.ValuesFor(p.ID.Path, b.ID.Path) {
					choice := LCChoice{Table: tables[ch.Table], Name: names.Humanize(ch.Table.Path), Share: ch.Share}
					if choice.Table != nil {
						choice.Name = choice.Table.Name
					}
					var detail []string
					if ch.Floors != "" {
						lo, hi, _ := strings.Cut(ch.Floors, ",")
						detail = append(detail, "pisos "+strings.TrimSpace(lo)+" a "+strings.TrimSpace(hi))
					}
					if ch.Biome != "" {
						detail = append(detail, "solo en "+term(ch.Biome).Name)
					}
					choice.Detail = strings.Join(detail, " · ")
					addTable(choice.Table)
					h.add(choice.Table, ch.Share, c.Count)
					if choice.Table != nil {
						share := ch.Share
						if share <= 0 {
							share = 1
						}
						for _, d := range choice.Table.Drops {
							d.Item.addLC(x, LCItemRow{Part: part.Name, Block: box.Block, Count: c.Count, Chance: d.Chance * share, Share: ch.Share, Table: choice.Table, Detail: choice.Detail, CountMin: d.CountMin, CountMax: d.CountMax})
						}
					}
					box.Choices = append(box.Choices, choice)
				}
				x.Containers += c.Count
				part.Containers = append(part.Containers, box)
			}
			x.Parts = append(x.Parts, part)
		}
		sortRefs(x.Tables, func(t *Table) Ref { return t.Ref })
		x.Haul = h.list()
		return x
	}

	styles := map[*lostcities.CityStyle]*LCStyle{}
	var style func(cs *lostcities.CityStyle) *LCStyle
	style = func(cs *lostcities.CityStyle) *LCStyle {
		if x, ok := styles[cs]; ok {
			return x
		}
		x := &LCStyle{Ref: Ref{ID: cs.ID, Name: namer.Asset("citystyle", cs.ID), URL: "lostcities/estilos/" + idPath(cs.ID) + "/"}}
		styles[cs] = x
		for _, id := range cs.Inherits {
			if parent := lm.CityStyles[id]; parent != nil {
				x.Inherits = append(x.Inherits, style(parent))
			}
		}
		add := func(list []lostcities.Weighted) []LCWeighted {
			var out []LCWeighted
			for _, w := range list {
				b := building(w.Building)
				out = append(out, LCWeighted{Building: b, Share: w.Share})
				if !containsStyle(b.Styles, x) {
					b.Styles = append(b.Styles, x)
				}
				if b.HasLoot() {
					x.WithLoot++
				}
			}
			return out
		}
		x.Buildings = add(cs.Buildings)
		x.Multi = add(cs.Multi)
		return x
	}

	worlds := map[domain.ResourceID]*LCWorld{}
	for _, w := range lm.WorldStyles {
		x := &LCWorld{Ref: Ref{ID: w.ID, Name: namer.Asset("worldstyle", w.ID), URL: "lostcities/"}}
		for _, r := range w.CityStyles {
			s := style(r.Style)
			if !containsWorld(s.Worlds, x) {
				s.Worlds = append(s.Worlds, x)
			}
			x.Rules = append(x.Rules, LCRule{Style: s, Factor: r.Factor, Biomes: rule(r.Biomes)})
		}
		for _, sc := range w.Scattered {
			out := LCScattered{Name: namer.Asset("scattered", sc.Name), Weight: sc.Weight, Biomes: rule(sc.Biomes)}
			for _, b := range sc.Buildings {
				out.Buildings = append(out.Buildings, building(b))
			}
			x.Scattered = append(x.Scattered, out)
		}
		worlds[w.ID] = x
		lc.Worlds = append(lc.Worlds, x)
	}
	for _, p := range lm.Profiles {
		pid := domain.ResourceID{Namespace: "lostcities", Path: p.Profile}
		dim := p.Dimension
		if id, err := domain.ParseResourceID(dim); err == nil {
			dim = namer.Asset("dimension", id)
		}
		lc.Profiles = append(lc.Profiles, LCProfile{
			Dimension: dim, Profile: namer.Asset("profile", pid), ProfileID: p.Profile,
			Origin: p.Origin, World: worlds[p.WorldStyle],
		})
	}

	for _, s := range styles {
		lc.Styles = append(lc.Styles, s)
	}
	sortRefs(lc.Styles, func(s *LCStyle) Ref { return s.Ref })
	for _, b := range buildings {
		sortRefs(b.Styles, func(s *LCStyle) Ref { return s.Ref })
		lc.Buildings = append(lc.Buildings, b)
	}
	sortRefs(lc.Buildings, func(b *LCBuilding) Ref { return b.Ref })

	m.LostCities = lc
	m.Sections = append(m.Sections, Section{Nav: "lostcities", Label: "Lost Cities", URL: "lostcities/"})
}

func containsStyle(list []*LCStyle, s *LCStyle) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func containsWorld(list []*LCWorld, w *LCWorld) bool {
	for _, x := range list {
		if x == w {
			return true
		}
	}
	return false
}

// renderLostCities writes the Lost Cities tab, if present.
func (r *renderer) renderLostCities(m *Model) {
	lc := m.LostCities
	if lc == nil {
		return
	}
	r.render("lc_index", "lostcities/", "lostcities", "Lost Cities", m, lc)
	r.render("lc_styles", "lostcities/estilos/", "lostcities", "Estilos de ciudad", m, lc)
	for _, s := range lc.Styles {
		r.render("lc_style", s.URL, "lostcities", s.Name, m, s)
	}
	r.render("lc_buildings", "lostcities/edificios/", "lostcities", "Edificios", m, lc)
	for _, b := range lc.Buildings {
		r.render("lc_building", b.URL, "lostcities", b.Name, m, b)
	}
}

// lcSearch adds city styles and buildings to the search index.
func lcSearch(m *Model) []searchEntry {
	if m.LostCities == nil {
		return nil
	}
	var out []searchEntry
	for _, s := range m.LostCities.Styles {
		out = append(out, searchEntry{Type: "l", Name: s.Name, ID: s.ID.String(), URL: s.URL + "index.html", Sub: "Estilo de ciudad", Keys: "Lost Cities", Mod: s.ID.Namespace})
	}
	for _, b := range m.LostCities.Buildings {
		sub := "Edificio"
		if b.Multi {
			sub = "Edificio múltiple"
		}
		out = append(out, searchEntry{Type: "l", Name: b.Name, ID: b.ID.String(), URL: b.URL + "index.html", Sub: sub, Keys: "Lost Cities", Mod: b.ID.Namespace})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
