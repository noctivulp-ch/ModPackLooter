// Package discovery is the hook point where one Discoverer per supported loot
// source is plugged in. Discoverers run in a deterministic order (phase, then
// priority, then ID) and publish what they find as claims; the generic
// discoverer runs last and only handles loot tables nobody claimed. A second
// hook point runs Enrichers, which annotate sources without creating new ones.
//
// See docs/12-arquitectura-de-descubrimiento.md.
package discovery

import (
	"context"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/worldgen"
)

// Phase groups plugins by how trustworthy their findings are.
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

// Applicability declares when a plugin variant can run. Empty fields match
// everything.
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

// Descriptor identifies a plugin and where it goes in the chain. Version
// variants of the same plugin share the ID and must have non-overlapping
// Applicability.
type Descriptor struct {
	ID       string
	Phase    Phase
	Priority int // higher runs earlier within the phase
	Applies  Applicability
}

// Plugin is anything that can be registered at a hook point.
type Plugin interface {
	Descriptor() Descriptor
}

// ResourceIndex is the read-only view of the merged data discoverers query.
type ResourceIndex interface {
	// LootTables lists every effective loot table after overrides.
	LootTables() []domain.ResourceID
	// ReadJSON decodes the effective file of a resource type and id.
	ReadJSON(typ string, id domain.ResourceID, v any) (bool, error)
	// IDs lists the ids of a resource type.
	IDs(typ string) []domain.ResourceID
	// Tag resolves a tag of a tag type (e.g. "tags/block"), merged across packs.
	Tag(typ string, id domain.ResourceID) []domain.ResourceID
}

// Worldgen is the structure data discoverers need.
type Worldgen interface {
	Structures() []worldgen.Structure
	StructureLoot(s worldgen.Structure) ([]worldgen.Found, []domain.ResourceID)
	TemplatesWithLoot() []*worldgen.Template
	BiomeTag(id domain.ResourceID) []domain.ResourceID
}

// Files reads files of the game folder, such as mod configs.
type Files interface {
	ReadFile(rel string) ([]byte, error)
}

// Input is everything a plugin may read.
type Input struct {
	Target      Target
	Resources   ResourceIndex
	World       Worldgen
	Files       Files
	Diagnostics *domain.Diagnostics
}

// Discoverer finds loot sources of one kind and publishes them as claims.
type Discoverer interface {
	Plugin
	Discover(ctx context.Context, in Input, out *Claims) error
}

// Enricher annotates sources already discovered (Lootr, …).
type Enricher interface {
	Plugin
	Enrich(ctx context.Context, in Input, sources *Claims) error
}
