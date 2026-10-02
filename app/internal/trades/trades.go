// Package trades is the common model of the trades tab: villager
// professions, the wandering trader and NPCs that sell or buy items.
// Discoverers build Catalogs and attach them with the key prefix "trades/".
package trades

import (
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// ExtraPrefix is the prefix of the keys trade discoverers attach with.
const ExtraPrefix = "trades/"

// Catalog is one trading system (vanilla villagers, an NPC mod…).
type Catalog struct {
	Key   string // page under tradeos/
	Name  string
	Intro []string
	// Confidence of the data: Known for built-in knowledge, Exact when read
	// from files.
	Confidence domain.Confidence
	Merchants  []*Merchant
	Order      int
}

// Merchant is a profession, the wandering trader or an NPC.
type Merchant struct {
	ID       domain.ResourceID // profession or entity id; NPCs use npc:<name>
	Name     string            // fixed name (NPCs); empty = from the lang
	Entity   domain.ResourceID // entity to name it after, if no Name
	Station  domain.ResourceID // job site block
	Levels   []Level
	Drops    []Drop // NPCs: items they drop
	Note     string
	Location string // where to find it (NPC files)
}

// Level is a villager level (or a tier of the wandering trader). Each time
// the merchant reaches it, Picks offers are chosen at random from Offers.
type Level struct {
	Level  int
	Name   string
	Picks  int // 0 = every offer is always available
	Offers []Offer
}

// Chance returns the probability that a given offer of the level appears.
func (l Level) Chance() float64 {
	if l.Picks <= 0 || l.Picks >= len(l.Offers) {
		return 1
	}
	return float64(l.Picks) / float64(len(l.Offers))
}

// Offer is one trade: the player gives Buy and receives Sell.
type Offer struct {
	Buy     []Stack
	Sell    Stack
	MaxUses int
	XP      int
	Note    string
}

// Stack is an amount of an item. Max > Count marks a range.
type Stack struct {
	Item      domain.ResourceID
	Count     int
	Max       int
	Enchanted bool
	Note      string
}

// Drop is an item an NPC drops when killed.
type Drop struct {
	Item   domain.ResourceID
	Count  int
	Chance float64
}

// Emerald is the vanilla currency.
var Emerald = domain.MustParseResourceID("minecraft:emerald")

// Stack helpers.
func Of(item string, count int) Stack {
	return Stack{Item: domain.MustParseResourceID(item), Count: count}
}
