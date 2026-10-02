package site

import (
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"sort"
	"strings"
)

// Card is one way of getting an item with every detail, for the item page:
// nothing hides behind a link, links only lead to the master pages.
type Card struct {
	Key   string
	Title string
	Unit  string
	Best  float64
	Rows  int // amount of data, to show the richest cards first

	Structures []StructGroup
	LC         []LCGroup
	Mobs       []*ItemSource
	Fish       []FishPlace
	FishTables []*ItemSource
	Trades     []*TradeRef
	NPCs       []*NPCDrop
	Plain      []*ItemSource
}

// StructGroup is the structures of one mod (or vanilla).
type StructGroup struct {
	Label      string
	Structures []StructCard
}

// StructCard is a structure and every container that can give the item.
type StructCard struct {
	Owner *Owner
	Best  float64
	Rows  []*ItemSource
}

// LCGroup is the Lost Cities buildings of the same city styles.
type LCGroup struct {
	Label     string
	Buildings []LCCard
}

// LCCard is a building and every container that can give the item.
type LCCard struct {
	Building *LCBuilding
	Best     float64
	Rows     []LCItemRow
}

// LCItemRow is a container of a building part that can give the item.
type LCItemRow struct {
	Part     string
	Block    string
	Count    int
	Chance   float64 // per container: table share × drop chance
	Share    float64
	Table    *Table
	Detail   string
	CountMin float64
	CountMax float64
}

// FishPlace is a biome (or "any biome") and the catches of the item there.
type FishPlace struct {
	Name string
	URL  string
	Rows []FishRow
}

// FishRow is a catch of the item in a fishing system.
type FishRow struct {
	Source *FishSource
	Group  string
	Catch  FishCatch
}

func (it *Item) addLC(b *LCBuilding, row LCItemRow) {
	if it.lcRows == nil {
		it.lcRows = map[*LCBuilding][]LCItemRow{}
	}
	it.lcRows[b] = append(it.lcRows[b], row)
}

// buildCards gathers every way of getting the item with its details.
func buildCards(it *Item, m *Model) {
	cards := map[string]*Card{}
	card := func(key string) *Card {
		if c := cards[key]; c != nil {
			return c
		}
		c := &Card{Key: key}
		for _, o := range wayOrder {
			if o.key == key {
				c.Title, c.Unit = o.title, o.unit
			}
		}
		cards[key] = c
		return c
	}
	best := func(c *Card, p float64) {
		if p > c.Best {
			c.Best = p
		}
	}

	// Structures, grouped by mod, then by structure.
	structs := map[*Owner]*StructCard{}
	var order []*Owner
	for _, s := range it.Sources {
		if s.Status.Disabled() {
			continue
		}
		switch {
		case s.Owner != nil && s.Kind == domain.KindArchaeology:
			c := card("arqueologia")
			c.Plain = append(c.Plain, s)
			best(c, s.Effective)
		case s.Owner != nil:
			sc := structs[s.Owner]
			if sc == nil {
				sc = &StructCard{Owner: s.Owner}
				structs[s.Owner] = sc
				order = append(order, s.Owner)
			}
			sc.Rows = append(sc.Rows, s)
			if s.Effective > sc.Best {
				sc.Best = s.Effective
			}
			best(card("estructuras"), s.Effective)
		case s.Kind == domain.KindEntity:
			c := card("criaturas")
			c.Mobs = append(c.Mobs, s)
			best(c, s.Effective)
		case s.Kind == domain.KindFishing:
			c := card("pesca")
			c.FishTables = append(c.FishTables, s)
			best(c, s.Effective)
		case s.Kind == domain.KindArchaeology:
			c := card("arqueologia")
			c.Plain = append(c.Plain, s)
			best(c, s.Effective)
		case s.Kind == domain.KindContainer:
			c := card("contenedores")
			c.Plain = append(c.Plain, s)
			best(c, s.Effective)
		case s.Kind == domain.KindGameplay:
			c := card("jugabilidad")
			c.Plain = append(c.Plain, s)
			best(c, s.Effective)
		default:
			c := card("otros")
			c.Plain = append(c.Plain, s)
			best(c, s.Effective)
		}
	}
	if len(order) > 0 {
		c := card("estructuras")
		groups := map[string]*StructGroup{}
		var labels []string
		for _, o := range order {
			label := o.Mod.Name
			if o.Mod.ID.Namespace == "minecraft" {
				label = "Minecraft (vanilla)"
			}
			g := groups[label]
			if g == nil {
				g = &StructGroup{Label: label}
				groups[label] = g
				labels = append(labels, label)
			}
			g.Structures = append(g.Structures, *structs[o])
		}
		sort.SliceStable(labels, func(i, j int) bool {
			vi, vj := strings.HasSuffix(labels[i], "(vanilla)"), strings.HasSuffix(labels[j], "(vanilla)")
			if vi != vj {
				return vi
			}
			return labels[i] < labels[j]
		})
		for _, l := range labels {
			g := groups[l]
			sort.SliceStable(g.Structures, func(i, j int) bool { return g.Structures[i].Best > g.Structures[j].Best })
			c.Structures = append(c.Structures, *g)
			for _, sc := range g.Structures {
				c.Rows += len(sc.Rows)
			}
		}
	}

	// Lost Cities: buildings grouped by their city styles.
	if len(it.lcRows) > 0 {
		c := card("lostcities")
		groups := map[string]*LCGroup{}
		var labels []string
		for b, rows := range it.lcRows {
			var styles []string
			for _, s := range b.Styles {
				styles = append(styles, s.Name)
			}
			sort.Strings(styles)
			label := "Fuera de las ciudades o dentro de edificios múltiples"
			if len(styles) > 0 {
				label = "Ciudades de estilo " + strings.Join(styles, ", ")
			}
			g := groups[label]
			if g == nil {
				g = &LCGroup{Label: label}
				groups[label] = g
				labels = append(labels, label)
			}
			lc := LCCard{Building: b, Rows: rows}
			for _, r := range rows {
				if r.Chance > lc.Best {
					lc.Best = r.Chance
				}
			}
			best(c, lc.Best)
			g.Buildings = append(g.Buildings, lc)
			c.Rows += len(rows)
		}
		sort.Strings(labels)
		for _, l := range labels {
			g := groups[l]
			sort.Slice(g.Buildings, func(i, j int) bool {
				if g.Buildings[i].Best != g.Buildings[j].Best {
					return g.Buildings[i].Best > g.Buildings[j].Best
				}
				return g.Buildings[i].Building.Name < g.Buildings[j].Building.Name
			})
			c.LC = append(c.LC, *g)
		}
	}

	// Fishing, by biome.
	if len(it.Fishing) > 0 {
		c := card("pesca")
		places := map[string]*FishPlace{}
		var keys []string
		add := func(name, url string, row FishRow) {
			p := places[name]
			if p == nil {
				p = &FishPlace{Name: name, URL: url}
				places[name] = p
				keys = append(keys, name)
			}
			p.Rows = append(p.Rows, row)
			best(c, row.Catch.Chance)
			c.Rows++
		}
		sources := map[*FishSource]bool{}
		for _, f := range it.Fishing {
			sources[f.Source] = true
		}
		for _, src := range m.fishSources() {
			if !sources[src] {
				continue
			}
			for _, g := range src.Everywhere {
				for _, ct := range g.Catches {
					if ct.Entry.Item == it {
						add("Cualquier bioma", "", FishRow{Source: src, Group: g.Title, Catch: ct})
					}
				}
			}
			for _, b := range src.Biomes {
				for _, g := range b.Groups {
					for _, ct := range g.Catches {
						if ct.Entry.Item == it {
							add(b.Name, b.URL, FishRow{Source: src, Group: g.Title, Catch: ct})
						}
					}
				}
			}
		}
		sort.SliceStable(keys, func(i, j int) bool {
			if keys[i] == "Cualquier bioma" || keys[j] == "Cualquier bioma" {
				return keys[i] == "Cualquier bioma"
			}
			return keys[i] < keys[j]
		})
		for _, k := range keys {
			c.Fish = append(c.Fish, *places[k])
		}
	}
	if c := cards["pesca"]; c != nil {
		c.Rows += len(c.FishTables)
	}

	for _, t := range it.Trades {
		if t.Sells {
			c := card("tradeo")
			c.Trades = append(c.Trades, t)
			best(c, t.Offer.Chance)
			c.Rows++
		}
	}
	for _, d := range it.NPCDrops {
		c := card("npc")
		c.NPCs = append(c.NPCs, d)
		best(c, d.Chance)
		c.Rows++
	}
	for _, key := range []string{"criaturas", "arqueologia", "contenedores", "jugabilidad", "otros"} {
		if c := cards[key]; c != nil {
			c.Rows += len(c.Mobs) + len(c.Plain)
		}
	}

	it.Cards = nil
	for _, o := range wayOrder {
		if c := cards[o.key]; c != nil {
			it.Cards = append(it.Cards, c)
		}
	}
	vague := func(c *Card) bool { return c.Key == "contenedores" || c.Key == "otros" }
	sort.SliceStable(it.Cards, func(i, j int) bool {
		a, b := it.Cards[i], it.Cards[j]
		if vague(a) != vague(b) {
			return !vague(a)
		}
		return a.Rows > b.Rows
	})
}

// fishSources lists the fishing systems of the site.
func (m *Model) fishSources() []*FishSource {
	if m.Fishing == nil {
		return nil
	}
	return m.Fishing.Sources
}
