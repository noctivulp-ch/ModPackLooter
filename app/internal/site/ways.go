package site

import (
	"fmt"
	"sort"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
)

// Way is one method of getting an item (structures, mobs, fishing…), for
// the summary at the top of the item page: places first, details below.
type Way struct {
	Key    string // anchor of its detail section
	Title  string // "En estructuras"
	Unit   string // what the chance is per: "por contenedor"
	Best   float64
	Places []*WayPlace
}

// WayPlace is a place of a way, with its best chance.
type WayPlace struct {
	Name   string
	URL    string
	Chance float64 // 0 when the chance is not known
	Detail string
	Way    *Way
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

var wayOrder = []struct{ key, title, unit string }{
	{"estructuras", "En estructuras", "por contenedor"},
	{"lostcities", "En edificios de Lost Cities", "por contenedor"},
	{"arqueologia", "Arqueología", "por bloque sospechoso"},
	{"criaturas", "Lo sueltan criaturas", "por criatura"},
	{"pesca", "Pescando", "por captura"},
	{"tradeo", "Comerciando", "de que te lo ofrezcan"},
	{"npc", "Lo sueltan NPCs", "por NPC"},
	{"contenedores", "En contenedores", "por contenedor"},
	{"jugabilidad", "Jugabilidad", "por vez"},
	{"otros", "Otras tablas de loot", "por tirada"},
}

// buildWays summarises how to get an item, most likely way first.
func buildWays(it *Item, namer *names.Namer) {
	ways := map[string]*Way{}
	place := func(key, name, url string, chance float64, detail string) {
		w := ways[key]
		if w == nil {
			for _, o := range wayOrder {
				if o.key == key {
					w = &Way{Key: o.key, Title: o.title, Unit: o.unit}
				}
			}
			ways[key] = w
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
			place("tradeo", t.Merchant.Name, t.Merchant.URL, t.Offer.Chance, t.Level)
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
	sort.SliceStable(it.Ways, func(i, j int) bool { return it.Ways[i].Best > it.Ways[j].Best })
	it.Top = nil
	if len(it.Ways) > 0 {
		it.Top = it.Ways[0].Places[0]
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
