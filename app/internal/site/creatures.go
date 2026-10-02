package site

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
	"github.com/EnierAragon/ModPackLooter/app/internal/spawns"
)

// Creature is the master page of a mob, a villager or a trader: where it
// appears, what it drops and what it trades.
type Creature struct {
	Ref
	Mod        *Mod
	Table      *Table // its drops
	Natural    []CreatureBiome
	Structures []CreatureStructure
	Merchants  []*TradeMerchant
	Other      []string // other ways it appears (curing, summoning…)
	// From are creatures it comes from (a villager from a cured zombie
	// villager): their spawns are its spawns too.
	From   []*Creature
	Status domain.Status
}

// CreatureBiome is a biome where it spawns naturally.
type CreatureBiome struct {
	Biome    *Biome
	Name     string
	URL      string
	Category string
	Share    float64
	Group    string // "1–4"
	Origin   string
}

// CreatureStructure is a structure that holds or spawns it.
type CreatureStructure struct {
	Name     string
	URL      string
	Count    int
	Spawners int
	Override bool
	Status   domain.Status
}

// Known reports whether anything says where it appears.
func (c *Creature) Known() bool {
	return len(c.Natural)+len(c.Structures)+len(c.Other) > 0
}

// LiveStructures are the structures that still generate.
func (c *Creature) LiveStructures() []CreatureStructure {
	var out []CreatureStructure
	for _, s := range c.Structures {
		if !s.Status.Disabled() {
			out = append(out, s)
		}
	}
	return out
}

// Where sums up where it appears, in a few words.
func (c *Creature) Where() string {
	var parts []string
	if n := len(c.LiveStructures()); n > 0 {
		parts = append(parts, fmt.Sprintf("en %d %s", n, plural(n, "estructura", "estructuras")))
	}
	if n := c.biomeCount(); n > 0 {
		parts = append(parts, fmt.Sprintf("aparece en %d %s", n, plural(n, "bioma", "biomas")))
	}
	if len(c.Other) > 0 {
		parts = append(parts, strings.ToLower(c.Other[0][:1])+c.Other[0][1:])
	}
	if len(parts) == 0 && len(c.Structures) > 0 {
		return "solo en estructuras desactivadas"
	}
	if len(parts) == 0 {
		return "no se sabe dónde aparece"
	}
	return strings.Join(parts, " · ")
}

func (c *Creature) biomeCount() int {
	seen := map[string]bool{}
	for _, b := range c.Natural {
		seen[b.Name] = true
	}
	return len(seen)
}

var categoryNames = map[string]string{
	"monster": "hostil", "creature": "pasiva", "ambient": "ambiente",
	"water_creature": "acuática", "water_ambient": "pez", "underground_water_creature": "acuática de cueva",
	"axolotls": "ajolote", "misc": "otra", "añadido": "añadida por un mod",
}

// Other ways vanilla creatures appear, set by the game's code.
var otherOrigins = map[string][]string{
	"minecraft:villager":         {"Curando a un aldeano zombi (Debilidad y manzana de oro)", "Reproduciendo aldeanos"},
	"minecraft:zombie_villager":  {"Un zombi que mata a un aldeano (en dificultad normal o difícil)"},
	"minecraft:iron_golem":       {"Lo crean los aldeanos de una aldea", "Construyéndolo con bloques de hierro y una calabaza"},
	"minecraft:snow_golem":       {"Construyéndolo con nieve y una calabaza"},
	"minecraft:witch":            {"Un rayo que cae cerca de un aldeano"},
	"minecraft:zombified_piglin": {"Desde portales del Nether en el mundo normal", "Un rayo que cae cerca de un cerdo"},
	"minecraft:wither":           {"Se invoca con cráneos de esqueleto Wither y arena de almas"},
	"minecraft:ender_dragon":     {"En el End, al llegar por primera vez"},
	"minecraft:wandering_trader": {"Aparece de vez en cuando cerca de los jugadores (cerca de una campana de aldea si la hay)"},
	"minecraft:warden":           {"Lo invocan los chilladores de sculk (en la ciudad antigua)"},
	"minecraft:phantom":          {"Aparece de noche si llevas más de 3 días sin dormir"},
	"minecraft:skeleton_horse":   {"Trampa de rayo durante una tormenta"},
	"minecraft:silverfish":       {"Bloques infestados en montañas y fortalezas"},
	"minecraft:pillager":         {"Patrullas y asaltos (malos presagios)"},
	"minecraft:ravager":          {"Asaltos (malos presagios)"},
	"minecraft:vindicator":       {"Asaltos (malos presagios)"},
	"minecraft:evoker":           {"Asaltos (malos presagios)"},
	"minecraft:cat":              {"Callejeros en aldeas; negros en cabañas de bruja"},
}

// Creatures another one turns into, set by the game's code.
var comesFrom = map[string][]string{
	"minecraft:villager":         {"minecraft:zombie_villager"},
	"minecraft:zombie_villager":  {"minecraft:zombie"},
	"minecraft:witch":            {"minecraft:villager"},
	"minecraft:zombified_piglin": {"minecraft:piglin", "minecraft:pig"},
}

// buildCreatures makes a page for every mob with drops or trades.
func buildCreatures(res *analysis.Result, m *Model, namer *names.Namer, modOf func(string) *Mod) {
	model, _ := res.Extras[spawns.Extra].(*spawns.Model)
	owners := map[domain.ResourceID]*Owner{}
	for _, o := range m.Owners {
		if !o.Template {
			owners[o.ID] = o
		}
	}
	biomes := map[domain.ResourceID]*Biome{}
	for _, b := range m.Biomes {
		biomes[b.ID] = b
	}
	creatures := map[domain.ResourceID]*Creature{}
	of := func(id domain.ResourceID) *Creature {
		if c := creatures[id]; c != nil {
			return c
		}
		c := &Creature{Ref: Ref{ID: id, Name: namer.Entity(id), URL: "criaturas/" + idPath(id) + "/"}, Mod: modOf(id.Namespace)}
		c.Status = res.Status(domain.Target{Kind: domain.TargetEntity, ID: id})
		c.Other = otherOrigins[id.String()]
		if model != nil {
			if e := model.Entities[id]; e != nil {
				for _, n := range e.Natural {
					cb := CreatureBiome{Name: namer.Biome(n.Biome), Category: categoryNames[n.Category], Share: n.Share, Origin: n.Origin}
					if cb.Category == "" {
						cb.Category = names.Humanize(n.Category)
					}
					if n.Min > 0 {
						cb.Group = fmt.Sprint(n.Min)
						if n.Max > n.Min {
							cb.Group += "–" + fmt.Sprint(n.Max)
						}
					}
					if b := biomes[n.Biome]; b != nil {
						cb.Biome, cb.URL = b, b.URL
						if b.Status.Disabled() {
							continue
						}
					}
					c.Natural = append(c.Natural, cb)
				}
				for _, s := range e.Structures {
					cs := CreatureStructure{Name: namer.Structure(s.Structure), Count: s.Count, Spawners: s.Spawners, Override: s.Override}
					cs.Status = res.Status(domain.Target{Kind: domain.TargetStructure, ID: s.Structure})
					if o := owners[s.Structure]; o != nil {
						cs.Name, cs.URL, cs.Status = o.Name, o.URL, o.Status
					}
					c.Structures = append(c.Structures, cs)
				}
			}
		}
		sort.SliceStable(c.Natural, func(i, j int) bool { return c.Natural[i].Share > c.Natural[j].Share })
		creatures[id] = c
		return c
	}
	for _, t := range m.Tables {
		if t.Kind == domain.KindEntity {
			c := of(entityOf(t.ID))
			c.Table = t
			t.Creature = c
		}
	}
	if m.Trades != nil {
		for _, cat := range m.Trades.Catalogs {
			for _, mc := range cat.Merchants {
				if mc.entity.IsZero() {
					continue
				}
				c := of(mc.entity)
				c.Merchants = append(c.Merchants, mc)
				mc.Creature = c
			}
		}
	}
	for _, c := range creatures {
		for _, f := range comesFrom[c.ID.String()] {
			if src, ok := creatures[domain.MustParseResourceID(f)]; ok {
				c.From = append(c.From, src)
			} else if model != nil && model.Entities[domain.MustParseResourceID(f)] != nil {
				c.From = append(c.From, of(domain.MustParseResourceID(f)))
			}
		}
	}
	for _, c := range creatures {
		m.Creatures = append(m.Creatures, c)
		for _, n := range c.Natural {
			if n.Biome != nil {
				n.Biome.Creatures = append(n.Biome.Creatures, BiomeCreature{Creature: c, Spawn: n})
			}
		}
	}
	for _, b := range m.Biomes {
		sort.SliceStable(b.Creatures, func(i, j int) bool {
			x, y := b.Creatures[i].Spawn, b.Creatures[j].Spawn
			if x.Category != y.Category {
				return x.Category < y.Category
			}
			return x.Share > y.Share
		})
	}
	sortRefs(m.Creatures, func(c *Creature) Ref { return c.Ref })
}
