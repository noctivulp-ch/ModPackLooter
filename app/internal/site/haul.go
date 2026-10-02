package site

import "sort"

// HaulItem is an item a place can give, for "¿Qué hay aquí?": the best
// chance per container and how many containers (or tables) can give it.
type HaulItem struct {
	Item       *Item
	Best       float64 // best chance in one container
	Containers int     // containers that can hold it (0 when unknown)
	Tables     []*Table
	CountMax   float64
}

// haul accumulates the items of a place.
type haul struct {
	byItem map[*Item]*HaulItem
}

func newHaul() *haul { return &haul{byItem: map[*Item]*HaulItem{}} }

// add records that n containers (0 = unknown) hold table t with the given
// share (probability that a container uses this table; 0 means always).
func (h *haul) add(t *Table, share float64, n int) {
	if t == nil {
		return
	}
	if share <= 0 {
		share = 1
	}
	for _, d := range t.Drops {
		x := h.byItem[d.Item]
		if x == nil {
			x = &HaulItem{Item: d.Item}
			h.byItem[d.Item] = x
		}
		if p := d.Chance * share; p > x.Best {
			x.Best = p
		}
		if d.CountMax > x.CountMax {
			x.CountMax = d.CountMax
		}
		x.Containers += n
		seen := false
		for _, o := range x.Tables {
			seen = seen || o == t
		}
		if !seen {
			x.Tables = append(x.Tables, t)
		}
	}
}

func (h *haul) merge(o []HaulItem) {
	for _, it := range o {
		x := h.byItem[it.Item]
		if x == nil {
			cp := it
			cp.Tables = append([]*Table(nil), it.Tables...)
			h.byItem[it.Item] = &cp
			continue
		}
		if it.Best > x.Best {
			x.Best = it.Best
		}
		if it.CountMax > x.CountMax {
			x.CountMax = it.CountMax
		}
		x.Containers += it.Containers
		for _, t := range it.Tables {
			seen := false
			for _, o := range x.Tables {
				seen = seen || o == t
			}
			if !seen {
				x.Tables = append(x.Tables, t)
			}
		}
	}
}

// list returns the items, most likely first.
func (h *haul) list() []HaulItem {
	out := make([]HaulItem, 0, len(h.byItem))
	for _, x := range h.byItem {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Best != out[j].Best {
			return out[i].Best > out[j].Best
		}
		if out[i].Item.Name != out[j].Item.Name {
			return out[i].Item.Name < out[j].Item.Name
		}
		return out[i].Item.URL < out[j].Item.URL
	})
	return out
}
