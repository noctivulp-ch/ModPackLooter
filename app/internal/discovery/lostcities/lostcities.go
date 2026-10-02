// Package lostcities discovers the loot of Lost Cities buildings. Palette
// entries with a "loot" field name a condition; the condition picks one loot
// table by weight ("factor") each time a container is placed.
package lostcities

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// ID of the discoverer.
const ID = "lostcities"

// Owner is the synthetic structure that groups every Lost Cities building.
var Owner = domain.Owner{Kind: domain.OwnerStructure, ID: domain.MustParseResourceID("lostcities:city")}

// Discoverer reads palettes, parts and conditions of Lost Cities.
type Discoverer struct{}

func (Discoverer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{
		ID: ID, Phase: discovery.PhaseSpecific, Priority: 80,
		Applies: discovery.Applicability{RequiresMods: []string{"lostcities"}},
	}
}

type conditionValue struct {
	Factor float64 `json:"factor"`
	Value  string  `json:"value"`
	// Tests that restrict where the value applies (shown as evidence).
	Range      string `json:"range"`
	InPart     string `json:"inpart"`
	InBuilding string `json:"inbuilding"`
	InBiome    string `json:"inbiome"`
	Level      string `json:"level"`
}

func (Discoverer) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	used := map[domain.ResourceID]string{} // condition -> container block
	for _, typ := range []string{resources.TypeLostCitiesPal, resources.TypeLostCitiesParts} {
		for _, id := range in.Resources.IDs(typ) {
			var doc any
			if ok, err := in.Resources.ReadJSON(typ, id, &doc); !ok || err != nil {
				if err != nil {
					in.Diagnostics.Add(domain.LevelWarning, "discovery", id.String(), "Lost Cities: %v", err)
				}
				continue
			}
			for cond, block := range lootRefs(doc, id.Namespace) {
				if _, seen := used[cond]; !seen {
					used[cond] = block
				}
			}
		}
	}
	if m := BuildModel(in); m.HasContent() {
		out.Attach(ID, m)
	}
	if len(used) == 0 {
		return nil
	}
	info := domain.OwnerInfo{
		Owner:      Owner,
		Name:       "Lost Cities (edificios de la ciudad)",
		Biomes:     in.World.BiomeTag(domain.MustParseResourceID("minecraft:is_overworld")),
		BiomesNote: "según el perfil de Lost Cities; las ciudades pueden cubrir casi cualquier bioma del Overworld",
	}
	profiles := ReadProfiles(in)
	if desc := profiles.Describe(); desc != "" {
		info.BiomesNote = "Ciudades activas en " + desc + " (" + profiles.Origin + ")."
		// With a model world, the biomes of the city dimensions are known exactly.
		if w := in.Files.Level(); w != nil {
			set := map[domain.ResourceID]bool{}
			exact := true
			for dim := range profiles.Active() {
				d, ok := w.Dimension(dim)
				if !ok || d.Biomes == nil {
					exact = false
					continue
				}
				for _, b := range d.Biomes {
					set[b] = true
				}
			}
			if exact && len(set) > 0 {
				info.Biomes = info.Biomes[:0]
				for b := range set {
					info.Biomes = append(info.Biomes, b)
				}
				sort.Slice(info.Biomes, func(i, j int) bool { return info.Biomes[i].String() < info.Biomes[j].String() })
				info.BiomesNote += " Biomas tomados del mundo " + w.Name + "."
			}
		}
	}
	out.DeclareOwner(info)

	conds := make([]domain.ResourceID, 0, len(used))
	for c := range used {
		conds = append(conds, c)
	}
	sort.Slice(conds, func(i, j int) bool { return conds[i].String() < conds[j].String() })
	for _, cond := range conds {
		var raw struct {
			Values []conditionValue `json:"values"`
		}
		ok, err := in.Resources.ReadJSON(resources.TypeLostCitiesCond, cond, &raw)
		if err != nil || !ok {
			in.Diagnostics.Add(domain.LevelWarning, "discovery", cond.String(), "Lost Cities: condición de loot no encontrada o ilegible")
			continue
		}
		total := 0.0
		for _, v := range raw.Values {
			total += v.Factor
		}
		// A table may appear in several values (e.g. different floors): its
		// share is the sum of their weights.
		type agg struct {
			factor  float64
			details []string
		}
		byTable := map[domain.ResourceID]*agg{}
		var order []domain.ResourceID
		for _, v := range raw.Values {
			table, err := domain.ParseResourceID(v.Value)
			if err != nil || total <= 0 {
				continue
			}
			a := byTable[table]
			if a == nil {
				a = &agg{}
				byTable[table] = a
				order = append(order, table)
			}
			a.factor += v.Factor
			a.details = append(a.details, describe(cond, v, total))
		}
		for _, table := range order {
			a := byTable[table]
			src := domain.LootSource{
				LootTable:  table,
				Kind:       discovery.KindForContainer(used[cond]),
				Owner:      Owner,
				Confidence: domain.ConfidenceExact,
				Container:  used[cond],
				Share:      a.factor / total,
			}
			for _, d := range a.details {
				src.Evidence = append(src.Evidence, domain.Evidence{DiscoveredBy: ID, Detail: d})
			}
			out.Add(src)
		}
	}
	return nil
}

func describe(cond domain.ResourceID, v conditionValue, total float64) string {
	parts := []string{fmt.Sprintf("condición %s, peso %g de %g", cond, v.Factor, total)}
	if v.Range != "" {
		parts = append(parts, "pisos "+strings.ReplaceAll(v.Range, ",", " a "))
	}
	if v.InPart != "" {
		parts = append(parts, "en la pieza "+v.InPart)
	}
	if v.InBuilding != "" {
		parts = append(parts, "en el edificio "+v.InBuilding)
	}
	if v.InBiome != "" {
		parts = append(parts, "en el bioma "+v.InBiome)
	}
	return strings.Join(parts, "; ")
}

// lootRefs finds every {"loot": "<condition>", "block": "…"} object.
func lootRefs(v any, ns string) map[domain.ResourceID]string {
	out := map[domain.ResourceID]string{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if loot, ok := x["loot"].(string); ok && loot != "" {
				id := domain.ResourceID{Namespace: ns, Path: loot}
				if strings.Contains(loot, ":") {
					id, _ = domain.ParseResourceID(loot)
				}
				block, _ := x["block"].(string)
				block, _, _ = strings.Cut(block, "[")
				out[id] = block
			}
			for _, e := range x {
				walk(e)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(v)
	return out
}
