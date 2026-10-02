package discovery

import (
	"sort"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

type claimKey struct {
	table domain.ResourceID
	kind  domain.SourceKind
	owner domain.Owner
}

// Claims collects the loot sources published by discoverers. The same
// (loot table, kind, owner) found twice keeps the highest confidence and
// merges the evidence of both.
type Claims struct {
	byKey   map[claimKey]*domain.LootSource
	byTable map[domain.ResourceID]domain.Confidence
}

// NewClaims returns an empty claim set.
func NewClaims() *Claims {
	return &Claims{
		byKey:   map[claimKey]*domain.LootSource{},
		byTable: map[domain.ResourceID]domain.Confidence{},
	}
}

// Add publishes a source, deduplicating against earlier claims.
func (c *Claims) Add(s domain.LootSource) {
	key := claimKey{table: s.LootTable, kind: s.Kind, owner: s.Owner}
	if existing, ok := c.byKey[key]; ok {
		if s.Confidence > existing.Confidence {
			existing.Confidence = s.Confidence
		}
		existing.Evidence = append(existing.Evidence, s.Evidence...)
	} else {
		stored := s
		stored.Evidence = append([]domain.Evidence(nil), s.Evidence...)
		c.byKey[key] = &stored
	}
	if prev, ok := c.byTable[s.LootTable]; !ok || s.Confidence > prev {
		c.byTable[s.LootTable] = s.Confidence
	}
}

// IsClaimed reports whether any discoverer already published the table.
func (c *Claims) IsClaimed(table domain.ResourceID) bool {
	_, ok := c.byTable[table]
	return ok
}

// BestConfidence returns the highest confidence claimed for the table.
func (c *Claims) BestConfidence(table domain.ResourceID) (domain.Confidence, bool) {
	conf, ok := c.byTable[table]
	return conf, ok
}

// Sources returns every claim in a deterministic order.
func (c *Claims) Sources() []domain.LootSource {
	out := make([]domain.LootSource, 0, len(c.byKey))
	for _, s := range c.byKey {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.LootTable != b.LootTable {
			return a.LootTable.String() < b.LootTable.String()
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Owner.Kind != b.Owner.Kind {
			return a.Owner.Kind < b.Owner.Kind
		}
		return a.Owner.ID.String() < b.Owner.ID.String()
	})
	return out
}
