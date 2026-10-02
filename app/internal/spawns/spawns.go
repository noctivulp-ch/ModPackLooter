// Package spawns is the common model of where creatures appear: natural
// spawns per biome and creatures placed by structures. The discoverer
// attaches a *Model with the key Extra.
package spawns

import "github.com/EnierAragon/ModPackLooter/app/internal/domain"

// Extra is the key of the attached model.
const Extra = "spawns"

// Model maps each entity to where it appears.
type Model struct {
	Entities map[domain.ResourceID]*Entity
}

// Entity is where a creature appears.
type Entity struct {
	ID         domain.ResourceID
	Natural    []BiomeSpawn
	Structures []StructureSpawn
}

// BiomeSpawn is a natural spawn in a biome. Share is the weight over the
// total weight of its category in that biome: how often a spawn attempt of
// the category picks this creature.
type BiomeSpawn struct {
	Biome    domain.ResourceID
	Category string // monster, creature, ambient, water_creature…
	Weight   int
	Min, Max int
	Share    float64
	Origin   string // "bioma" or the biome modifier that adds it
}

// StructureSpawn is a creature placed by a structure: saved in its
// templates, made by its spawners, or spawned inside it (spawn_overrides).
type StructureSpawn struct {
	Structure domain.ResourceID
	Count     int // creatures saved in the templates reached
	Spawners  int // spawner blocks
	Override  bool
}

// Of returns the entity, creating it.
func (m *Model) Of(id domain.ResourceID) *Entity {
	if m.Entities == nil {
		m.Entities = map[domain.ResourceID]*Entity{}
	}
	e := m.Entities[id]
	if e == nil {
		e = &Entity{ID: id}
		m.Entities[id] = e
	}
	return e
}
