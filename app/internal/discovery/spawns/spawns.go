// Package spawns discovers where creatures appear: the spawners of each
// biome, the biome modifiers that add or remove spawns (Forge and
// NeoForge), the creatures and spawner blocks saved in structure templates
// and the spawn_overrides of structures. The model answers "where does this
// mob come from?" for drops and merchants.
package spawns

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
	"github.com/EnierAragon/ModPackLooter/app/internal/spawns"
)

// ID of the discoverer.
const ID = "spawns"

// Discoverer builds the spawns model.
type Discoverer struct{}

func (Discoverer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ID, Phase: discovery.PhaseSpecific, Priority: 50}
}

type spawner struct {
	Type     string `json:"type"`
	Weight   int    `json:"weight"`
	MinCount int    `json:"minCount"`
	MaxCount int    `json:"maxCount"`
}

// spawnerMap reads {"monster": [...]}.
func spawnerMap(raw json.RawMessage) map[string][]spawner {
	out := map[string][]spawner{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// Things saved in templates that are not creatures.
var notCreatures = map[string]bool{
	"item_frame": true, "glow_item_frame": true, "painting": true, "armor_stand": true,
	"minecart": true, "chest_minecart": true, "hopper_minecart": true, "tnt_minecart": true,
	"furnace_minecart": true, "spawner_minecart": true, "command_block_minecart": true,
	"boat": true, "chest_boat": true, "end_crystal": true, "falling_block": true, "item": true,
	"leash_knot": true, "marker": true, "block_display": true, "item_display": true,
	"text_display": true, "interaction": true, "experience_orb": true, "arrow": true,
}

func creature(id domain.ResourceID) bool {
	return !(id.Namespace == "minecraft" && notCreatures[id.Path])
}

// Vanilla structures built by code, not by jigsaw templates: their
// creatures and spawners come from the game's structure pieces.
var codeStructures = map[string][]spawns.StructureSpawn{
	"minecraft:mansion":      {{Structure: id("minecraft:mansion"), Count: 1}},
	"minecraft:mineshaft":    {{Structure: id("minecraft:mineshaft"), Spawners: 1}},
	"minecraft:stronghold":   {{Structure: id("minecraft:stronghold"), Spawners: 1}},
	"minecraft:monster_room": {{Structure: id("minecraft:monster_room"), Spawners: 1}},
}

var codeMobs = map[string][]string{
	"minecraft:mansion":      {"minecraft:vindicator", "minecraft:evoker"},
	"minecraft:mineshaft":    {"minecraft:cave_spider"},
	"minecraft:stronghold":   {"minecraft:silverfish"},
	"minecraft:monster_room": {"minecraft:zombie", "minecraft:skeleton", "minecraft:spider"},
	"minecraft:monument":     {"minecraft:elder_guardian"},
	"minecraft:igloo":        {"minecraft:villager", "minecraft:zombie_villager"},
}

func id(s string) domain.ResourceID { return domain.MustParseResourceID(s) }

func (Discoverer) Discover(ctx context.Context, in discovery.Input, out *discovery.Claims) error {
	m := &spawns.Model{}
	type key struct {
		biome    domain.ResourceID
		category string
		entity   domain.ResourceID
	}
	natural := map[key]*spawns.BiomeSpawn{}
	add := func(biome domain.ResourceID, category string, s spawner, origin string) {
		e, err := domain.ParseResourceID(s.Type)
		if err != nil || s.Weight <= 0 {
			return
		}
		k := key{biome, category, e}
		if natural[k] == nil {
			natural[k] = &spawns.BiomeSpawn{Biome: biome, Category: category, Weight: s.Weight, Min: s.MinCount, Max: s.MaxCount, Origin: origin}
		}
	}
	for _, b := range in.Resources.IDs(resources.TypeBiome) {
		var raw struct {
			Spawners json.RawMessage `json:"spawners"`
		}
		if ok, err := in.Resources.ReadJSON(resources.TypeBiome, b, &raw); !ok || err != nil {
			continue
		}
		for cat, list := range spawnerMap(raw.Spawners) {
			for _, s := range list {
				add(b, cat, s, "bioma")
			}
		}
	}

	// Biome modifiers: add_spawns and remove_spawns.
	biomes := func(raw json.RawMessage) []domain.ResourceID { return holderSet(in, resources.TypeBiomeTag, raw) }
	for _, typ := range []string{resources.TypeForgeBiomeMod, resources.TypeNeoBiomeMod} {
		var removals []struct {
			biomes   []domain.ResourceID
			entities map[domain.ResourceID]bool
		}
		for _, mid := range in.Resources.IDs(typ) {
			var raw struct {
				Type        string          `json:"type"`
				Biomes      json.RawMessage `json:"biomes"`
				Spawners    json.RawMessage `json:"spawners"`
				EntityTypes json.RawMessage `json:"entity_types"`
			}
			if ok, err := in.Resources.ReadJSON(typ, mid, &raw); !ok || err != nil {
				continue
			}
			kind := raw.Type[strings.Index(raw.Type, ":")+1:]
			switch kind {
			case "add_spawns":
				var list []spawner
				if json.Unmarshal(raw.Spawners, &list) != nil {
					var one spawner
					if json.Unmarshal(raw.Spawners, &one) == nil {
						list = []spawner{one}
					}
				}
				for _, b := range biomes(raw.Biomes) {
					for _, s := range list {
						add(b, categoryOf(s.Type), s, "modificador "+mid.String())
					}
				}
			case "remove_spawns":
				r := struct {
					biomes   []domain.ResourceID
					entities map[domain.ResourceID]bool
				}{biomes(raw.Biomes), map[domain.ResourceID]bool{}}
				for _, e := range holderSet(in, resources.TypeEntityTypeTag, raw.EntityTypes) {
					r.entities[e] = true
				}
				removals = append(removals, r)
			}
		}
		for _, r := range removals {
			for _, b := range r.biomes {
				for k := range natural {
					if k.biome == b && r.entities[k.entity] {
						delete(natural, k)
					}
				}
			}
		}
	}
	total := map[[2]string]int{}
	for k, s := range natural {
		total[[2]string{k.biome.String(), k.category}] += s.Weight
	}
	for k, s := range natural {
		if t := total[[2]string{k.biome.String(), k.category}]; t > 0 {
			s.Share = float64(s.Weight) / float64(t)
		}
		e := m.Of(k.entity)
		e.Natural = append(e.Natural, *s)
	}

	// Structures: templates, spawners and spawn_overrides.
	for _, s := range in.World.Structures() {
		if err := ctx.Err(); err != nil {
			return err
		}
		mobs := map[domain.ResourceID]*spawns.StructureSpawn{}
		at := func(e domain.ResourceID) *spawns.StructureSpawn {
			if mobs[e] == nil {
				mobs[e] = &spawns.StructureSpawn{Structure: s.ID}
			}
			return mobs[e]
		}
		_, reached := in.World.StructureLoot(s)
		for _, tid := range reached {
			t, ok := in.World.Template(tid)
			if !ok {
				continue
			}
			for e, n := range t.Mobs {
				if creature(e) {
					at(e).Count += n
				}
			}
			for e, n := range t.Spawners {
				at(e).Spawners += n
			}
		}
		var raw struct {
			Overrides map[string]struct {
				Spawns []spawner `json:"spawns"`
			} `json:"spawn_overrides"`
		}
		if ok, _ := in.Resources.ReadJSON(resources.TypeStructure, s.ID, &raw); ok {
			for _, o := range raw.Overrides {
				for _, sp := range o.Spawns {
					if e, err := domain.ParseResourceID(sp.Type); err == nil {
						at(e).Override = true
					}
				}
			}
		}
		for _, mob := range codeMobs[s.ID.String()] {
			x := at(id(mob))
			if base, ok := codeStructures[s.ID.String()]; ok && base[0].Spawners > 0 {
				x.Spawners++
			} else if x.Count == 0 && !x.Override {
				x.Count = 1
			}
		}
		for e, x := range mobs {
			m.Of(e).Structures = append(m.Of(e).Structures, *x)
		}
	}
	for _, e := range m.Entities {
		sort.Slice(e.Natural, func(i, j int) bool {
			if e.Natural[i].Biome != e.Natural[j].Biome {
				return e.Natural[i].Biome.String() < e.Natural[j].Biome.String()
			}
			return e.Natural[i].Category < e.Natural[j].Category
		})
		sort.Slice(e.Structures, func(i, j int) bool { return e.Structures[i].Structure.String() < e.Structures[j].Structure.String() })
	}
	if len(m.Entities) > 0 {
		out.Attach(spawns.Extra, m)
	}
	return nil
}

// categoryOf guesses the category of a spawn added by a modifier without
// one: the list form of add_spawns does not say it.
func categoryOf(string) string { return "añadido" }

// holderSet reads "#tag", "id" or a list of them.
func holderSet(in discovery.Input, tagType string, raw json.RawMessage) []domain.ResourceID {
	var list []string
	var one string
	if json.Unmarshal(raw, &one) == nil {
		list = []string{one}
	} else if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	var out []domain.ResourceID
	for _, s := range list {
		if t, ok := strings.CutPrefix(s, "#"); ok {
			if tid, err := domain.ParseResourceID(t); err == nil {
				if tagType == resources.TypeBiomeTag {
					out = append(out, in.World.BiomeTag(tid)...)
				} else {
					out = append(out, in.Resources.Tag(tagType, tid)...)
				}
			}
			continue
		}
		if rid, err := domain.ParseResourceID(s); err == nil {
			out = append(out, rid)
		}
	}
	return out
}
