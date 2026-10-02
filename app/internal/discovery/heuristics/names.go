// Package heuristics links loot tables to structures when no data does,
// using naming conventions. Results are marked as heuristic.
package heuristics

import (
	"context"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/generic"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// NameMatchingID is the ID of the name matching discoverer.
const NameMatchingID = "name-matching"

// NameMatching links an unclaimed container table such as
// "towers:chests/ruined_tower_top" to the structure "towers:ruined_tower"
// of the same namespace, choosing the longest structure name contained in
// the table path.
type NameMatching struct{}

func (NameMatching) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: NameMatchingID, Phase: discovery.PhaseHeuristic}
}

func (NameMatching) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	byNamespace := map[string][]domain.ResourceID{}
	for _, s := range in.World.Structures() {
		byNamespace[s.ID.Namespace] = append(byNamespace[s.ID.Namespace], s.ID)
	}
	for _, table := range in.Resources.LootTables() {
		if out.IsClaimed(table) {
			continue
		}
		kind := generic.KindFromPath(table.Path)
		if kind != domain.KindContainer && kind != domain.KindArchaeology {
			continue
		}
		if best, ok := bestMatch(table, byNamespace[table.Namespace]); ok {
			out.Add(domain.LootSource{
				LootTable:  table,
				Kind:       kind,
				Owner:      domain.Owner{Kind: domain.OwnerStructure, ID: best},
				Confidence: domain.ConfidenceHeuristic,
				Evidence:   []domain.Evidence{{DiscoveredBy: NameMatchingID, Detail: "el nombre de la tabla contiene el de la estructura"}},
			})
		}
	}
	return nil
}

// bestMatch returns the structure whose last path segment appears, as whole
// words, in the table path. Short names (< 4 chars) are ignored to avoid
// false positives.
func bestMatch(table domain.ResourceID, structures []domain.ResourceID) (domain.ResourceID, bool) {
	path := "/" + strings.ReplaceAll(table.Path, "/", "_/") + "_"
	var best domain.ResourceID
	bestLen := 0
	for _, s := range structures {
		name := s.Path[strings.LastIndex(s.Path, "/")+1:]
		if len(name) < 4 || len(name) <= bestLen {
			continue
		}
		if containsWord(path, name) {
			best, bestLen = s, len(name)
		}
	}
	return best, bestLen > 0
}

func containsWord(path, word string) bool {
	for i := 0; ; {
		j := strings.Index(path[i:], word)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(word)
		before, after := path[start-1], byte('_')
		if end < len(path) {
			after = path[end]
		}
		if (before == '/' || before == '_') && (after == '_' || after == '/') {
			return true
		}
		i = start + 1
	}
}
