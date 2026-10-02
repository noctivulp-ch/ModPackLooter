// Package discovery is the hook point where one Discoverer per supported loot
// source is plugged in. Discoverers run in a deterministic order (phase, then
// priority, then ID) and publish what they find as claims; the generic
// discoverer runs last and only handles loot tables nobody claimed.
//
// See docs/12-arquitectura-de-descubrimiento.md.
package discovery

import (
	"context"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
)

// Phase groups discoverers by how trustworthy their findings are.
// Lower phases run first.
type Phase int

const (
	PhaseSpecific   Phase = iota + 1 // data read directly: NBT, processor lists, built-in knowledge
	PhaseRelational                  // indirect links: mob spawns, location_check, user overrides
	PhaseHeuristic                   // guesses, e.g. matching names
	PhaseGeneric                     // reserved: classify whatever is left
)

func (p Phase) String() string {
	switch p {
	case PhaseSpecific:
		return "específica"
	case PhaseRelational:
		return "relacional"
	case PhaseHeuristic:
		return "heurística"
	case PhaseGeneric:
		return "genérica"
	default:
		return "desconocida"
	}
}

// Target describes the modpack being analysed.
type Target struct {
	Version mcversion.Version
	Loader  domain.Loader
	Mods    map[string]bool // mod IDs present in the modpack
}

// Applicability declares when a discoverer variant can run. Empty fields
// match everything.
type Applicability struct {
	Versions     mcversion.Range
	Loaders      []domain.Loader
	RequiresMods []string
}

// AppliesTo reports whether the variant is valid for the target.
func (a Applicability) AppliesTo(t Target) bool {
	if !a.Versions.Contains(t.Version) {
		return false
	}
	if len(a.Loaders) > 0 {
		found := false
		for _, l := range a.Loaders {
			if l == t.Loader {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, mod := range a.RequiresMods {
		if !t.Mods[mod] {
			return false
		}
	}
	return true
}

// Descriptor identifies a discoverer and where it goes in the chain.
// Version variants of the same source share the ID and must have
// non-overlapping Applicability.
type Descriptor struct {
	ID       string
	Phase    Phase
	Priority int // higher runs earlier within the phase
	Applies  Applicability
}

// ResourceIndex is the read-only, version-normalised view of the merged
// modpack data that discoverers query. It grows as discoverers need more.
type ResourceIndex interface {
	// LootTables lists every effective loot table after overrides.
	LootTables() []domain.ResourceID
}

// Input is everything a discoverer may read.
type Input struct {
	Target    Target
	Resources ResourceIndex
}

// Discoverer finds loot sources of one kind and publishes them as claims.
type Discoverer interface {
	Descriptor() Descriptor
	Discover(ctx context.Context, in Input, out *Claims) error
}
