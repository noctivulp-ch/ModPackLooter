package domain

// Certainty is how sure the app is that something does not generate.
type Certainty int

const (
	// Possibly disabled: a config or script mentions it next to words such as
	// "disable", or a rule with conditions denies it. Worth a warning only.
	Possibly Certainty = iota + 1
	// Certainly disabled: the data or a config the app understands removes it
	// (no structure set places it, none of its biomes exist, a mod's config
	// disables it explicitly…).
	Certainly
)

func (c Certainty) String() string {
	switch c {
	case Certainly:
		return "desactivado"
	case Possibly:
		return "posiblemente desactivado"
	default:
		return ""
	}
}

// MarshalText makes certainties readable in JSON reports.
func (c Certainty) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

// TargetKind is what a disablement applies to.
type TargetKind string

const (
	TargetStructure TargetKind = "structure" // also synthetic owners such as lostcities:city
	TargetBiome     TargetKind = "biome"
	TargetEntity    TargetKind = "entity" // a mob that no longer spawns
)

// Target identifies what is disabled.
type Target struct {
	Kind TargetKind
	ID   ResourceID
}

// Disablement says that something may not generate, and why.
type Disablement struct {
	Target    Target
	Certainty Certainty
	Reason    string
	By        string // disabler ID
}

// Status is the combined result for one target.
type Status struct {
	Certainty Certainty // 0 when enabled
	Reasons   []string
}

// Disabled reports whether the target is certainly disabled.
func (s Status) Disabled() bool { return s.Certainty == Certainly }
