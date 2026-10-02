package domain

// ChangeKind is the kind of source a mod changes.
type ChangeKind string

const (
	ChangeTable    ChangeKind = "table"    // one loot table (ID = table id)
	ChangeLootType ChangeKind = "loottype" // every table of a type (ID path = chest, entity, block, fishing, gameplay, archaeology, all)
	ChangeDrops    ChangeKind = "drops"    // a mob's drops (ID = entity id; empty = every mob)
	ChangeTrades   ChangeKind = "trades"   // villager or wandering trader trades (ID = profession, empty = all)
	ChangeFishing  ChangeKind = "fishing"  // fishing in general
	ChangeBarter   ChangeKind = "barter"   // piglin bartering
)

// ChangeTarget says what a change touches.
type ChangeTarget struct {
	Kind ChangeKind
	ID   ResourceID // zero = every source of the kind
}

// Effect is what a change does.
type Effect string

const (
	EffectAdd     Effect = "add"     // adds items or a whole table
	EffectRemove  Effect = "remove"  // removes items
	EffectReplace Effect = "replace" // replaces items or the whole table
	EffectChange  Effect = "change"  // changes counts, chances, enchantments…
	EffectDisable Effect = "disable" // turns the source (or a modifier) off
	EffectUnknown Effect = "unknown" // something changes, the app cannot tell what
)

// Change is a modification of a loot source made by a mod, a datapack, a
// script or a config. Certainly: read from data or a script statement the
// app understands. Possibly: a hint (a config key, code that listens to an
// event) whose exact effect is unknown.
type Change struct {
	Target    ChangeTarget
	Effect    Effect
	Items     []ResourceID // items added, removed or replaced
	With      []ResourceID // replacements
	Table     ResourceID   // injected loot table, if any
	Chance    float64      // 0 = unknown
	Certainty Certainty
	// Mod is who makes the change, for reading ("DCTweaks", "KubeJS"…).
	Mod string
	// Origin is the file (and line) the change comes from; generic
	// detectors skip origins a specific plugin already explained.
	Origin string
	Detail string
	By     string // detector ID
}
