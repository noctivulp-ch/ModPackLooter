// Package vanilla holds built-in knowledge about vanilla structures and
// features whose loot is assigned by code, not by data files.
package vanilla

import (
	"context"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
)

// ID of the discoverer.
const ID = "vanilla-knowledge"

type known struct {
	owner     string
	table     string
	container string
}

// feature is a vanilla feature (not a structure) that places loot.
type feature struct {
	id         string
	name       string
	biomeTag   string
	biomesNote string
}

// Knowledge for Minecraft 1.20.x, checked against the game's structure
// piece classes (StructurePiece subclasses set these tables in code).
var knowledge1_20 = []known{
	{"minecraft:desert_pyramid", "minecraft:chests/desert_pyramid", "minecraft:chest"},
	{"minecraft:desert_pyramid", "minecraft:archaeology/desert_pyramid", "minecraft:suspicious_sand"},
	{"minecraft:jungle_pyramid", "minecraft:chests/jungle_temple", "minecraft:chest"},
	{"minecraft:jungle_pyramid", "minecraft:chests/jungle_temple_dispenser", "minecraft:dispenser"},
	{"minecraft:igloo", "minecraft:chests/igloo_chest", "minecraft:chest"},
	{"minecraft:shipwreck", "minecraft:chests/shipwreck_map", "minecraft:chest"},
	{"minecraft:shipwreck", "minecraft:chests/shipwreck_supply", "minecraft:chest"},
	{"minecraft:shipwreck", "minecraft:chests/shipwreck_treasure", "minecraft:chest"},
	{"minecraft:shipwreck_beached", "minecraft:chests/shipwreck_map", "minecraft:chest"},
	{"minecraft:shipwreck_beached", "minecraft:chests/shipwreck_supply", "minecraft:chest"},
	{"minecraft:shipwreck_beached", "minecraft:chests/shipwreck_treasure", "minecraft:chest"},
	{"minecraft:ocean_ruin_cold", "minecraft:chests/underwater_ruin_big", "minecraft:chest"},
	{"minecraft:ocean_ruin_cold", "minecraft:chests/underwater_ruin_small", "minecraft:chest"},
	{"minecraft:ocean_ruin_cold", "minecraft:archaeology/ocean_ruin_cold", "minecraft:suspicious_gravel"},
	{"minecraft:ocean_ruin_warm", "minecraft:chests/underwater_ruin_big", "minecraft:chest"},
	{"minecraft:ocean_ruin_warm", "minecraft:chests/underwater_ruin_small", "minecraft:chest"},
	{"minecraft:ocean_ruin_warm", "minecraft:archaeology/ocean_ruin_warm", "minecraft:suspicious_sand"},
	{"minecraft:buried_treasure", "minecraft:chests/buried_treasure", "minecraft:chest"},
	{"minecraft:mansion", "minecraft:chests/woodland_mansion", "minecraft:chest"},
	{"minecraft:end_city", "minecraft:chests/end_city_treasure", "minecraft:chest"},
	{"minecraft:fortress", "minecraft:chests/nether_bridge", "minecraft:chest"},
	{"minecraft:stronghold", "minecraft:chests/stronghold_corridor", "minecraft:chest"},
	{"minecraft:stronghold", "minecraft:chests/stronghold_crossing", "minecraft:chest"},
	{"minecraft:stronghold", "minecraft:chests/stronghold_library", "minecraft:chest"},
	{"minecraft:mineshaft", "minecraft:chests/abandoned_mineshaft", "minecraft:chest_minecart"},
	{"minecraft:mineshaft_mesa", "minecraft:chests/abandoned_mineshaft", "minecraft:chest_minecart"},
	{"minecraft:ruined_portal", "minecraft:chests/ruined_portal", "minecraft:chest"},
	{"minecraft:ruined_portal_desert", "minecraft:chests/ruined_portal", "minecraft:chest"},
	{"minecraft:ruined_portal_jungle", "minecraft:chests/ruined_portal", "minecraft:chest"},
	{"minecraft:ruined_portal_mountain", "minecraft:chests/ruined_portal", "minecraft:chest"},
	{"minecraft:ruined_portal_nether", "minecraft:chests/ruined_portal", "minecraft:chest"},
	{"minecraft:ruined_portal_ocean", "minecraft:chests/ruined_portal", "minecraft:chest"},
	{"minecraft:ruined_portal_swamp", "minecraft:chests/ruined_portal", "minecraft:chest"},
	// Features: not structures, declared as owners below.
	{"minecraft:monster_room", "minecraft:chests/simple_dungeon", "minecraft:chest"},
	{"minecraft:desert_well", "minecraft:archaeology/desert_well", "minecraft:suspicious_sand"},
}

var features1_20 = []feature{
	{"minecraft:monster_room", "Mazmorra (monster room)", "minecraft:is_overworld", "subterránea, en casi cualquier bioma del Overworld"},
	{"minecraft:desert_well", "Pozo del desierto", "", "desierto"},
}

// Discoverer1_20 applies the 1.20.x knowledge.
type Discoverer1_20 struct{}

func (Discoverer1_20) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{
		ID: ID, Phase: discovery.PhaseSpecific, Priority: 90,
		Applies: discovery.Applicability{Versions: mcversion.MustParseRange(">=1.20 <1.21")},
	}
}

func (Discoverer1_20) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	return apply(in, out, knowledge1_20, features1_20)
}

func apply(in discovery.Input, out *discovery.Claims, entries []known, features []feature) error {
	tables := map[domain.ResourceID]bool{}
	for _, t := range in.Resources.LootTables() {
		tables[t] = true
	}
	structures := map[domain.ResourceID]bool{}
	for _, s := range in.World.Structures() {
		structures[s.ID] = true
	}
	featureIDs := map[domain.ResourceID]bool{}
	for _, f := range features {
		id := domain.MustParseResourceID(f.id)
		featureIDs[id] = true
		info := domain.OwnerInfo{Owner: domain.Owner{Kind: domain.OwnerStructure, ID: id}, Name: f.name, BiomesNote: f.biomesNote}
		if f.biomeTag != "" {
			info.Biomes = in.World.BiomeTag(domain.MustParseResourceID(f.biomeTag))
		} else if id.Path == "desert_well" {
			info.Biomes = []domain.ResourceID{domain.MustParseResourceID("minecraft:desert")}
		}
		out.DeclareOwner(info)
	}
	for _, k := range entries {
		owner := domain.MustParseResourceID(k.owner)
		table := domain.MustParseResourceID(k.table)
		// Only when the data exists: a datapack may remove a structure, and
		// without the vanilla jar there are no vanilla tables to show.
		if !tables[table] || (!structures[owner] && !featureIDs[owner]) {
			continue
		}
		out.Add(domain.LootSource{
			LootTable:  table,
			Kind:       discovery.KindForContainer(k.container),
			Owner:      domain.Owner{Kind: domain.OwnerStructure, ID: owner},
			Confidence: domain.ConfidenceKnown,
			Container:  k.container,
			Evidence:   []domain.Evidence{{DiscoveredBy: ID, Detail: "asignada por el código del juego"}},
		})
	}
	return nil
}
