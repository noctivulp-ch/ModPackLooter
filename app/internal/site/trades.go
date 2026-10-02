package site

import (
	"sort"
	"strconv"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
	"github.com/EnierAragon/ModPackLooter/app/internal/trades"
)

// Trades is the trades tab: catalogs (vanilla villagers, NPC mods…),
// piglin bartering and the mods that change trades.
type Trades struct {
	Catalogs []*TradeCatalog
	Barter   *Table
	Changes  []*Change
}

// TradeCatalog is a trading system.
type TradeCatalog struct {
	Ref
	Intro      []string
	Known      bool // built-in knowledge, not read from files
	Confidence domain.Confidence
	Merchants  []*TradeMerchant
}

// TradeMerchant is a profession, the wandering trader or an NPC.
type TradeMerchant struct {
	Ref
	Catalog  *TradeCatalog
	Station  *Item
	Levels   []TradeLevel
	Drops    []TradeDrop
	Note     string
	Location string
	Changes  []*Change
	Offers   int
}

// TradeLevel is a level of a merchant.
type TradeLevel struct {
	Level  int
	Name   string
	Picks  int
	Offers []TradeOffer
}

// TradeOffer is a trade: the player gives Buy and gets Sell.
type TradeOffer struct {
	Buy     []TradeStack
	Sell    TradeStack
	MaxUses int
	XP      int
	Note    string
	Chance  float64 // probability that a merchant offers it
}

// TradeStack is an amount of an item.
type TradeStack struct {
	Item      *Item
	Count     int
	Max       int
	Enchanted bool
}

// TradeDrop is an item an NPC drops.
type TradeDrop struct {
	Item   *Item
	Count  int
	Chance float64
}

// TradeRef links an item to a trade.
type TradeRef struct {
	Merchant *TradeMerchant
	Level    string
	Offer    TradeOffer
	Sells    bool // the merchant gives the item (else it buys it)
}

// NPCDrop links an item to an NPC that drops it.
type NPCDrop struct {
	Merchant *TradeMerchant
	Count    int
	Chance   float64
}

var enchantedBookID = domain.MustParseResourceID("minecraft:enchanted_book")

// Enchantments villagers never sell.
var untradeable = map[domain.ResourceID]bool{
	domain.MustParseResourceID("minecraft:soul_speed"):  true,
	domain.MustParseResourceID("minecraft:swift_sneak"): true,
}

var barterTable = domain.MustParseResourceID("minecraft:gameplay/piglin_bartering")

func buildTrades(res *analysis.Result, m *Model, namer *names.Namer, variantOf func(domain.ResourceID, domain.Variant) *Item) {
	itemOf := func(id domain.ResourceID) *Item { return variantOf(id, domain.Variant{}) }
	var cats []*trades.Catalog
	for key, v := range res.Extras {
		if c, ok := v.(*trades.Catalog); ok && strings.HasPrefix(key, trades.ExtraPrefix) {
			cats = append(cats, c)
		}
	}
	tab := &Trades{}
	for _, t := range m.Tables {
		if t.ID == barterTable {
			tab.Barter = t
		}
	}
	tab.Changes = m.tradeChanges
	if len(cats) == 0 && tab.Barter == nil {
		return
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].Order < cats[j].Order })
	var tradeable []domain.ResourceID
	for _, e := range res.Resources.Enchantments() {
		if !untradeable[e] {
			tradeable = append(tradeable, e)
		}
	}
	stack := func(s trades.Stack) TradeStack {
		return TradeStack{Item: variantOf(s.Item, s.Variant), Count: s.Count, Max: s.Max, Enchanted: s.Enchanted}
	}
	for _, c := range cats {
		tc := &TradeCatalog{Ref: Ref{Name: c.Name, URL: "tradeos/" + c.Key + "/"}, Intro: c.Intro, Known: c.Confidence == domain.ConfidenceKnown, Confidence: c.Confidence}
		for _, mc := range c.Merchants {
			tm := &TradeMerchant{Catalog: tc, Note: mc.Note, Location: mc.Location}
			name := mc.Name
			if v, ok := namer.Text(name); ok {
				name = v
			}
			if name == "" {
				switch {
				case mc.ID == mc.Entity:
					name = namer.Entity(mc.Entity)
				default:
					if v, ok := namer.Text("entity." + mc.Entity.Namespace + "." + mc.Entity.Path + "." + mc.ID.Path); ok {
						name = v
					} else {
						name = names.Humanize(mc.ID.Path)
					}
				}
			}
			tm.Ref = Ref{ID: mc.ID, Name: name, URL: tc.URL + idPath(mc.ID) + "/"}
			if !mc.Station.IsZero() {
				tm.Station = itemOf(mc.Station)
			}
			for _, l := range mc.Levels {
				tl := TradeLevel{Level: l.Level, Name: l.Name, Picks: l.Picks}
				for _, o := range l.Offers {
					to := TradeOffer{Sell: stack(o.Sell), MaxUses: o.MaxUses, XP: o.XP, Note: o.Note, Chance: l.Chance()}
					for _, b := range o.Buy {
						to.Buy = append(to.Buy, stack(b))
					}
					tl.Offers = append(tl.Offers, to)
					tm.Offers++
					if to.Sell.Item.ID != trades.Emerald {
						to.Sell.Item.Trades = append(to.Sell.Item.Trades, &TradeRef{Merchant: tm, Level: l.Name, Offer: to, Sells: true})
					}
					// A random enchanted book is each tradeable enchantment
					// with an equal share.
					if o.Sell.Item == enchantedBookID && o.Sell.Enchanted && o.Sell.Variant.IsZero() {
						for _, e := range tradeable {
							v := to
							v.Chance = to.Chance / float64(len(tradeable))
							v.Sell.Item = variantOf(o.Sell.Item, domain.Variant{Kind: domain.VariantEnchantment, Value: e.String()})
							v.Note = "uno al azar entre " + strconv.Itoa(len(tradeable)) + " encantamientos comerciables; el nivel también es al azar"
							v.Sell.Item.Trades = append(v.Sell.Item.Trades, &TradeRef{Merchant: tm, Level: l.Name, Offer: v, Sells: true})
						}
					}
					for _, b := range to.Buy {
						if b.Item.ID != trades.Emerald {
							b.Item.Trades = append(b.Item.Trades, &TradeRef{Merchant: tm, Level: l.Name, Offer: to})
						}
					}
				}
				tm.Levels = append(tm.Levels, tl)
			}
			for _, d := range mc.Drops {
				it := itemOf(d.Item)
				tm.Drops = append(tm.Drops, TradeDrop{Item: it, Count: d.Count, Chance: d.Chance})
				it.NPCDrops = append(it.NPCDrops, &NPCDrop{Merchant: tm, Count: d.Count, Chance: d.Chance})
			}
			for _, ch := range m.tradeChanges {
				if ch.targetID == mc.ID {
					tm.Changes = append(tm.Changes, ch)
				}
			}
			tc.Merchants = append(tc.Merchants, tm)
		}
		tab.Catalogs = append(tab.Catalogs, tc)
	}
	m.Trades = tab
	m.Sections = append(m.Sections, Section{Nav: "tradeos", Label: "Tradeos", URL: "tradeos/"})
}

func (r *renderer) renderTrades(m *Model) {
	if m.Trades == nil {
		return
	}
	r.render("trades_index", "tradeos/", "tradeos", "Tradeos", m, m.Trades)
	for _, c := range m.Trades.Catalogs {
		r.render("trades_catalog", c.URL, "tradeos", c.Name, m, c)
		for _, mc := range c.Merchants {
			r.render("trades_merchant", mc.URL, "tradeos", mc.Name, m, mc)
		}
	}
}
