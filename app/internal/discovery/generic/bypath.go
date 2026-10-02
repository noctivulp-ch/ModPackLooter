// Package generic holds the fallback discoverer that always runs last.
package generic

import (
	"context"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// ID of the fallback discoverer.
const ID = "generic-by-path"

// ByPath classifies every loot table nobody claimed using the conventional
// folder of its path (chests/, entities/, gameplay/…). It never touches
// tables already claimed by a more specific discoverer.
type ByPath struct{}

func (ByPath) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ID, Phase: discovery.PhaseGeneric}
}

func (ByPath) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	for _, table := range in.Resources.LootTables() {
		if out.IsClaimed(table) {
			continue
		}
		out.Add(domain.LootSource{
			LootTable:  table,
			Kind:       KindFromPath(table.Path),
			Confidence: domain.ConfidenceUnknown,
			Evidence:   []domain.Evidence{{DiscoveredBy: ID, Detail: "clasificada por la ruta de la tabla"}},
		})
	}
	return nil
}

// prefixes maps the first meaningful folder of a loot table path to a kind.
// Mods often nest their tables (e.g. "chests/village/house"), so only the
// first segment that matches is used.
var prefixes = map[string]domain.SourceKind{
	"chests":      domain.KindContainer,
	"chest":       domain.KindContainer,
	"entities":    domain.KindEntity,
	"entity":      domain.KindEntity,
	"gameplay":    domain.KindGameplay,
	"archaeology": domain.KindArchaeology,
	"blocks":      domain.KindBlock,
	"block":       domain.KindBlock,
}

// KindFromPath guesses the source kind from a loot table path.
func KindFromPath(path string) domain.SourceKind {
	// Vanilla keeps fishing under gameplay/fishing; fishing is more useful.
	if strings.Contains(path, "fishing") {
		return domain.KindFishing
	}
	for _, segment := range strings.Split(path, "/") {
		if kind, ok := prefixes[segment]; ok {
			return kind
		}
	}
	return domain.KindUnknown
}
