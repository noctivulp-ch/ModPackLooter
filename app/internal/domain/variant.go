package domain

import "strconv"

// Variant is a useful variant of an item that the game tells apart only by
// its data: the enchantment of an enchanted book, the potion of a potion or
// tipped arrow, the effect of a suspicious stew, the sound of a goat horn,
// the model of a modded gun (GunId, AmmoId…), the title of a written book.
// The zero value is the plain item.
type Variant struct {
	// Kind is one of the Variant* constants.
	Kind string
	// Key is the NBT key for VariantNBT (GunId, AmmoId…).
	Key string
	// Value is an id ("minecraft:mending") or a lang key (book titles).
	// Several enchantments on one book are joined with ",".
	Value string
	// Level is the enchantment level, when fixed.
	Level int
}

// Variant kinds.
const (
	VariantEnchantment = "enchantment" // stored or applied enchantment
	VariantPotion      = "potion"
	VariantEffect      = "effect"     // suspicious stew
	VariantInstrument  = "instrument" // goat horns
	VariantNBT         = "nbt"        // a mod's *Id key
	VariantTitle       = "title"      // written books
	VariantMap         = "map"        // explorer maps: destination tag
)

// IsZero reports whether v is the plain item.
func (v Variant) IsZero() bool { return v.Kind == "" }

// String is a stable key, e.g. "enchantment:minecraft:mending@1".
func (v Variant) String() string {
	if v.IsZero() {
		return ""
	}
	s := v.Kind + ":"
	if v.Key != "" {
		s += v.Key + "="
	}
	s += v.Value
	if v.Level > 0 {
		s += "@" + strconv.Itoa(v.Level)
	}
	return s
}
