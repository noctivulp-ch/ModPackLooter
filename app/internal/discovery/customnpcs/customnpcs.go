// Package customnpcs reads CustomNPCs' saved NPCs (clones) for the trades
// tab: what each NPC drops (NpcInv + DropChance) and, for traders, what it
// sells. Clones are saved as SNBT written like JSON ("5b", "1.0f"), so they
// are cleaned before decoding.
package customnpcs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
	"github.com/EnierAragon/ModPackLooter/app/internal/trades"
)

// ID of the discoverer.
const ID = "customnpcs"

// Discoverer reads CustomNPCs clones from the game folder and the model world.
type Discoverer struct{}

func (Discoverer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ID, Phase: discovery.PhaseSpecific, Priority: 30,
		Applies: discovery.Applicability{RequiresMods: []string{"customnpcs"}}}
}

type stack struct {
	Slot  int    `json:"Slot"`
	ID    string `json:"id"`
	Count int    `json:"Count"`
}

type npc struct {
	Name       string  `json:"Name"`
	Title      string  `json:"Title"`
	Role       int     `json:"Role"`
	NpcInv     []stack `json:"NpcInv"`
	DropChance []struct {
		Chance float64 `json:"Integer"`
		Slot   int     `json:"Slot"`
	} `json:"DropChance"`
	Sold     []stack `json:"TraderSold"`
	Currency []stack `json:"TraderCurrency"`
	Market   string  `json:"TraderMarket"`
}

// roleTrader is RoleType.TRADER in the CustomNPCs API: a trader has, per
// slot, two currencies and a sold item (IRoleTrader.getCurrency1/2,
// getSold), or uses a shared market (getMarket).
const roleTrader = 1

func (Discoverer) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	dirs := [][2]string{{filepath.Join(in.Files.RootDir(), "customnpcs", "clones"), "customnpcs/clones"}}
	if w := in.Files.Level(); w != nil {
		dirs = append(dirs, [2]string{filepath.Join(w.Path, "customnpcs", "clones"), w.Name + "/customnpcs/clones"})
	}
	cat := &trades.Catalog{Key: "npcs", Name: "NPCs (CustomNPCs)", Order: 50, Confidence: domain.ConfidenceExact,
		Intro: []string{
			"NPCs guardados en el modpack (clones de CustomNPCs): lo que sueltan al morir y, si son comerciantes, lo que venden.",
			"Los mercados que se crean dentro de un mundo solo se ven indicando ese mundo con --world.",
		}}
	// Shared markets live in the world (customnpcs/markets/<name>.json).
	markets := map[string]*npc{}
	if w := in.Files.Level(); w != nil {
		dir := filepath.Join(w.Path, "customnpcs", "markets")
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			var mk npc
			if json.Unmarshal(CleanSNBT(data), &mk) == nil {
				markets[strings.ToLower(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))] = &mk
			}
		}
	}
	seen := map[string]bool{}
	for _, d := range dirs {
		_ = filepath.WalkDir(d[0], func(path string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() || !strings.EqualFold(filepath.Ext(path), ".json") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var n npc
			if err := json.Unmarshal(CleanSNBT(data), &n); err != nil {
				rel, _ := filepath.Rel(d[0], path)
				in.Diagnostics.Add(domain.LevelWarning, "discovery", d[1]+"/"+filepath.ToSlash(rel), "CustomNPCs: %v", err)
				return nil
			}
			rel, _ := filepath.Rel(d[0], path)
			rel = filepath.ToSlash(rel)
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			key := strings.ToLower(base)
			if seen[key] {
				return nil
			}
			seen[key] = true
			m := &trades.Merchant{ID: domain.ResourceID{Namespace: "npc", Path: strings.ToLower(strings.ReplaceAll(base, " ", "_"))},
				Name: n.Name, Location: d[1] + "/" + rel}
			if m.Name == "" {
				m.Name = base
			}
			chance := map[int]float64{}
			for _, c := range n.DropChance {
				chance[c.Slot] = c.Chance / 100
			}
			for _, s := range n.NpcInv {
				id, err := domain.ParseResourceID(s.ID)
				if err != nil || s.Count == 0 {
					continue
				}
				p, ok := chance[s.Slot]
				if !ok {
					p = 1
				}
				m.Drops = append(m.Drops, trades.Drop{Item: id, Count: s.Count, Chance: p})
			}
			sort.SliceStable(m.Drops, func(i, j int) bool { return m.Drops[i].Chance > m.Drops[j].Chance })
			if offers := traderOffers(n); len(offers) > 0 {
				m.Levels = []trades.Level{{Level: 1, Name: "Vende", Offers: offers}}
			}
			if n.Role == roleTrader {
				switch {
				case n.Market != "" && markets[strings.ToLower(n.Market)] != nil:
					m.Levels = []trades.Level{{Level: 1, Name: "Vende (mercado " + n.Market + ")", Offers: traderOffers(*markets[strings.ToLower(n.Market)])}}
				case n.Market != "":
					m.Note = "Comerciante del mercado «" + n.Market + "»: sus tradeos se guardan en el mundo (indícalo con --world)."
				case len(m.Levels) == 0:
					m.Note = "Comerciante sin tradeos guardados en su archivo."
				}
			}
			if len(m.Drops) == 0 && len(m.Levels) == 0 && m.Note == "" {
				return nil
			}
			cat.Merchants = append(cat.Merchants, m)
			return nil
		})
	}
	if len(cat.Merchants) == 0 {
		return nil
	}
	sort.Slice(cat.Merchants, func(i, j int) bool {
		return names.Humanize(cat.Merchants[i].Name) < names.Humanize(cat.Merchants[j].Name)
	})
	out.Attach(trades.ExtraPrefix+ID, cat)
	return nil
}

// traderOffers pairs CustomNPCs' trader slots: sold slot i costs
// currency 1 (slot i) and currency 2 (slot i+18).
func traderOffers(n npc) []trades.Offer {
	cost := map[int]stack{}
	for _, c := range n.Currency {
		cost[c.Slot] = c
	}
	var out []trades.Offer
	for _, s := range n.Sold {
		id, err := domain.ParseResourceID(s.ID)
		if err != nil {
			continue
		}
		o := trades.Offer{Sell: trades.Stack{Item: id, Count: max(1, s.Count)}}
		for _, slot := range []int{s.Slot, s.Slot + 18} {
			if c, ok := cost[slot]; ok {
				if cid, err := domain.ParseResourceID(c.ID); err == nil {
					o.Buy = append(o.Buy, trades.Stack{Item: cid, Count: max(1, c.Count)})
				}
			}
		}
		out = append(out, o)
	}
	return out
}

// CleanSNBT turns SNBT written like JSON into JSON: drops number suffixes
// (5b, 1.0f, 3L) and typed array prefixes ([I; …]).
func CleanSNBT(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inStr := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inStr {
			// Raw control characters (multi-line dialog text) are not
			// valid JSON inside strings.
			switch {
			case c == '\n':
				out = append(out, '\\', 'n')
				continue
			case c == '\r':
				out = append(out, '\\', 'r')
				continue
			case c == '\t':
				out = append(out, '\\', 't')
				continue
			case c < 0x20:
				out = append(out, ' ')
				continue
			}
			out = append(out, c)
			if c == '\\' && i+1 < len(data) {
				i++
				out = append(out, data[i])
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch {
		case c == '"':
			inStr = true
		case strings.ContainsRune("bBsSlLfFdD", rune(c)) && i > 0 && data[i-1] >= '0' && data[i-1] <= '9' &&
			(i+1 == len(data) || strings.ContainsRune(" \t\r\n,]}", rune(data[i+1]))):
			continue
		case c == '[' && i+2 < len(data) && strings.ContainsRune("BIL", rune(data[i+1])) && data[i+2] == ';':
			out = append(out, '[')
			i += 2
			continue
		}
		out = append(out, c)
	}
	return out
}
