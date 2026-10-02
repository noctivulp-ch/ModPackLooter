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
	owners  map[domain.Owner]domain.OwnerInfo
}

// NewClaims returns an empty claim set.
func NewClaims() *Claims {
	return &Claims{
		byKey:   map[claimKey]*domain.LootSource{},
		byTable: map[domain.ResourceID]domain.Confidence{},
		owners:  map[domain.Owner]domain.OwnerInfo{},
	}
}

// Add publishes a source, deduplicating against earlier claims.
func (c *Claims) Add(s domain.LootSource) {
	key := claimKey{table: s.LootTable, kind: s.Kind, owner: s.Owner}
	if existing, ok := c.byKey[key]; ok {
		if s.Confidence > existing.Confidence {
			existing.Confidence = s.Confidence
		}
		if existing.Container == "" {
			existing.Container = s.Container
		}
		existing.Share = max(existing.Share, s.Share)
		existing.Evidence = append(existing.Evidence, s.Evidence...)
		existing.Notes = append(existing.Notes, s.Notes...)
	} else {
		stored := s
		stored.Evidence = append([]domain.Evidence(nil), s.Evidence...)
		stored.Notes = append([]domain.Note(nil), s.Notes...)
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

// DeclareOwner registers an owner that is not described by game data.
func (c *Claims) DeclareOwner(info domain.OwnerInfo) {
	c.owners[info.Owner] = info
}

// Owners returns the declared owners, sorted by id.
func (c *Claims) Owners() []domain.OwnerInfo {
	out := make([]domain.OwnerInfo, 0, len(c.owners))
	for _, o := range c.owners {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Owner.ID.String() < out[j].Owner.ID.String() })
	return out
}

// Annotate lets an enricher add notes to every source. fn may modify the
// source's Notes only.
func (c *Claims) Annotate(fn func(s domain.LootSource) []domain.Note) {
	for _, s := range c.byKey {
		s.Notes = append(s.Notes, fn(*s)...)
	}
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
