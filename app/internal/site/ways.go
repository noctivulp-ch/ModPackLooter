package site

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
)

// Way is one method of getting an item (structures, mobs, fishing…), for
// the summary at the top of the item page: places first, details below.
type Way struct {
	Key    string // anchor of its detail section
	Vague  bool   // no known place: shown after the others
	Title  string // "En estructuras"
	Unit   string // what the chance is per: "por contenedor"
	Best   float64
	Places []*WayPlace
	// Sources are the loot rows behind the way, for the details.
	Sources []*ItemSource
}

// WayPlace is a place of a way, with its best chance.
type WayPlace struct {
	Name   string
	URL    string
	Chance float64 // 0 when the chance is not known
	Detail string
	Way    *Way
	// Line is the chain from a biome to the item, for mobs and merchants.
	Line *Line
}

// More returns how many places are beyond the first n.
func (w *Way) More(n int) int {
	if len(w.Places) > n {
		return len(w.Places) - n
	}
	return 0
}

// Head returns the first n places.
func (w *Way) Head(n int) []*WayPlace {
	if len(w.Places) > n {
		return w.Places[:n]
	}
	return w.Places
}

// wayOrder lists the ways; vague ones (no known place) never lead.
var wayOrder = []struct {
	key, title, unit string
	vague            bool
}{
	{"estructuras", "En estructuras", "por contenedor", false},
	{"lostcities", "En edificios de Lost Cities", "por contenedor", false},
	{"arqueologia", "Arqueología", "por bloque sospechoso", false},
	{"criaturas", "Lo sueltan criaturas", "por criatura", false},
	{"pesca", "Pescando", "por captura", false},
	{"tradeo", "Comerciando", "de que te lo ofrezcan", false},
	{"npc", "Lo sueltan NPCs", "por NPC", false},
	{"jugabilidad", "Jugabilidad", "por vez", false},
	{"contenedores", "Contenedores sin estructura conocida", "por contenedor", true},
	{"otros", "Tablas de loot sin origen conocido", "por tirada", true},
}

// buildWays summarises how to get an item, most likely way first.
func buildWays(it *Item, namer *names.Namer) {
	ways := map[string]*Way{}
	var current *ItemSource
	place := func(key, name, url string, chance float64, detail string) {
		w := ways[key]
		if w == nil {
			for _, o := range wayOrder {
				if o.key == key {
					w = &Way{Key: o.key, Title: o.title, Unit: o.unit, Vague: o.vague}
				}
			}
			ways[key] = w
		}
		if current != nil {
			w.Sources = append(w.Sources, current)
		}
		for _, p := range w.Places {
			if p.URL == url && p.Name == name {
				if chance > p.Chance {
					p.Chance = chance
				}
				return
			}
		}
		w.Places = append(w.Places, &WayPlace{Name: name, URL: url, Chance: chance, Detail: detail, Way: w})
		if chance > w.Best {
			w.Best = chance
		}
	}
	for _, s := range it.Sources {
		current = s
		if s.Status.Disabled() {
			continue
		}
		switch {
		case s.Owner != nil && s.Kind == domain.KindArchaeology:
			place("arqueologia", s.Owner.Name, s.Owner.URL, s.Effective, "")
		case s.Owner != nil:
			detail := ""
			if n := len(s.Owner.Biomes); n > 0 {
				detail = fmt.Sprintf("%d %s", n, plural(n, "bioma", "biomas"))
			}
			place("estructuras", s.Owner.Name, s.Owner.URL, s.Effective, detail)
		case s.Kind == domain.KindEntity && s.Table.Creature != nil:
			place("criaturas", s.Table.Creature.Name, s.Table.Creature.URL, s.Effective, s.Table.Creature.Where())
		case s.Kind == domain.KindEntity:
			place("criaturas", s.Table.Name, s.Table.URL, s.Effective, "")
		case s.Kind == domain.KindFishing:
			place("pesca", s.Table.Name, s.Table.URL, s.Effective, "")
		case s.Kind == domain.KindArchaeology:
			place("arqueologia", s.Table.Name, s.Table.URL, s.Effective, "")
		case s.Kind == domain.KindContainer:
			place("contenedores", s.Table.Name, s.Table.URL, s.Effective, "")
		case s.Kind == domain.KindGameplay:
			place("jugabilidad", s.Table.Name, s.Table.URL, s.Effective, "")
		default:
			place("otros", s.Table.Name, s.Table.URL, s.Effective, "")
		}
	}
	current = nil
	for _, r := range it.LCBuildings {
		detail := ""
		if r.Haul.Containers > 0 {
			detail = fmt.Sprintf("%d %s", r.Haul.Containers, plural(r.Haul.Containers, "contenedor posible", "contenedores posibles"))
		}
		place("lostcities", r.Building.Name, r.Building.URL, r.Haul.Best, detail)
	}
	for _, f := range it.Fishing {
		detail := f.Rarity
		if f.Everywhere() {
			detail = joinNonEmpty(detail, "en cualquier bioma")
		} else if n := len(f.Places); n > 0 {
			detail = joinNonEmpty(detail, fmt.Sprintf("%d %s", n, plural(n, "bioma", "biomas")))
		}
		place("pesca", f.Source.Name, f.Source.URL, 0, detail)
	}
	for _, t := range it.Trades {
		if t.Sells {
			detail := t.Level
			if t.Merchant.Creature != nil {
				detail = joinNonEmpty(detail, t.Merchant.Creature.Where())
			} else if t.Merchant.Location != "" {
				detail = joinNonEmpty(detail, t.Merchant.Location)
			}
			place("tradeo", t.Merchant.Name, t.Merchant.URL, t.Offer.Chance, detail)
		}
	}
	for _, d := range it.NPCDrops {
		place("npc", d.Merchant.Name, d.Merchant.URL, d.Chance, "")
	}
	it.Ways = nil
	for _, o := range wayOrder {
		if w := ways[o.key]; w != nil {
			sort.SliceStable(w.Places, func(i, j int) bool { return w.Places[i].Chance > w.Places[j].Chance })
			it.Ways = append(it.Ways, w)
		}
	}
	// Most likely way first; ways without a known chance keep their order.
	sort.SliceStable(it.Ways, func(i, j int) bool {
		if it.Ways[i].Vague != it.Ways[j].Vague {
			return !it.Ways[i].Vague
		}
		return it.Ways[i].Best > it.Ways[j].Best
	})
	it.Top = nil
	if len(it.Ways) > 0 {
		it.Top = it.Ways[0].Places[0]
		switch it.Top.Way.Key {
		case "criaturas":
			for _, s := range it.Sources {
				if s.Table.Creature != nil && s.Table.Creature.URL == it.Top.URL && s.Effective == it.Top.Chance {
					it.Top.Line = s.Line()
				}
			}
		case "tradeo":
			for _, t := range it.Sells() {
				if t.Merchant.URL == it.Top.URL && t.Offer.Chance == it.Top.Chance && it.Top.Line == nil {
					it.Top.Line = t.Line()
				}
			}
		}
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func joinNonEmpty(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + " · " + b
}

// DisabledSources are loot sources a mod or config turns off.
func (it *Item) DisabledSources() []*ItemSource {
	var out []*ItemSource
	for _, s := range it.Sources {
		if s.Status.Disabled() {
			out = append(out, s)
		}
	}
	return out
}

// Buys are the trades where a merchant takes the item from the player.
func (it *Item) Buys() []*TradeRef {
	var out []*TradeRef
	for _, t := range it.Trades {
		if !t.Sells {
			out = append(out, t)
		}
	}
	return out
}

// Sells are the trades where a merchant gives the item.
func (it *Item) Sells() []*TradeRef {
	var out []*TradeRef
	for _, t := range it.Trades {
		if t.Sells {
			out = append(out, t)
		}
	}
	return out
}

// WayKeys lists the keys of the ways, for list filters.
func (it *Item) WayKeys() string {
	var keys []string
	for _, w := range it.Ways {
		keys = append(keys, w.Key)
	}
	if len(it.Buys()) > 0 {
		keys = append(keys, "compra")
	}
	return strings.Join(keys, " ")
}

// listRows orders the item list: each item followed by its variants.
func listRows(items []*Item) []*Item {
	out := make([]*Item, 0, len(items))
	for _, it := range items {
		if it.Base != nil {
			continue
		}
		out = append(out, it)
		out = append(out, it.Variants...)
	}
	return out
}

// SearchKeys are the extra words a list filter matches: English name,
// id and variant.
func (it *Item) SearchKeys() string {
	keys := append([]string{it.ID.String(), it.Variant.Value}, it.Aka...)
	return strings.Join(keys, " ")
}
