// Package disablers holds the plugins of the disabler gate: one per way a
// modpack can keep structures, biomes or mobs from generating. Each one says
// how sure it is: domain.Certainly (data or a config it fully understands) or
// domain.Possibly (a mention it cannot interpret for sure).
package disablers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/domain"
)

func structureTarget(id domain.ResourceID) domain.Target {
	return domain.Target{Kind: domain.TargetStructure, ID: id}
}

// StructureSetsID is the ID of the StructureSets disabler.
const StructureSetsID = "structure-sets"

// StructureSets: a structure only generates if a structure set places it. A
// datapack that removes the structure from its set (or empties the set)
// disables it for sure.
type StructureSets struct{}

func (StructureSets) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: StructureSetsID, Phase: discovery.PhaseSpecific, Priority: 100}
}

func (StructureSets) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	if !in.World.HasStructureSets() {
		return nil // without sets (e.g. no vanilla jar) the absence proves nothing
	}
	sets := in.World.StructureSets()
	for _, s := range in.World.Structures() {
		if len(sets[s.ID]) == 0 {
			out.Add(domain.Disablement{
				Target: structureTarget(s.ID), Certainty: domain.Certainly, By: StructureSetsID,
				Reason: "ningún structure_set la coloca: un datapack o mod la quitó de la generación",
			})
		}
	}
	return nil
}

// StructureBiomesID is the ID of the StructureBiomes disabler.
const StructureBiomesID = "structure-biomes"

// StructureBiomes: a structure with no biome to generate in, or whose biomes
// do not exist in the model world, cannot appear.
type StructureBiomes struct{}

func (StructureBiomes) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: StructureBiomesID, Phase: discovery.PhaseSpecific, Priority: 90}
}

func (StructureBiomes) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	var worldBiomes map[domain.ResourceID]bool
	complete := false
	worldName := ""
	if w := in.Files.Level(); w != nil {
		worldBiomes, complete = w.AllBiomes()
		worldName = w.Name
	}
	reported := map[domain.ResourceID]bool{}
	for _, s := range in.World.Structures() {
		if complete {
			for _, b := range s.Biomes {
				if !worldBiomes[b] && !reported[b] {
					reported[b] = true
					out.Add(domain.Disablement{
						Target: domain.Target{Kind: domain.TargetBiome, ID: b}, Certainty: domain.Certainly, By: StructureBiomesID,
						Reason: fmt.Sprintf("no existe en el mundo %q", worldName),
					})
				}
			}
		}
		if len(s.Biomes) == 0 {
			out.Add(domain.Disablement{
				Target: structureTarget(s.ID), Certainty: domain.Certainly, By: StructureBiomesID,
				Reason: "su lista de biomas está vacía: no puede generarse en ningún bioma",
			})
			continue
		}
		if worldBiomes == nil {
			continue
		}
		present := false
		for _, b := range s.Biomes {
			if worldBiomes[b] {
				present = true
				break
			}
		}
		if present {
			continue
		}
		certainty := domain.Certainly
		reason := fmt.Sprintf("ninguno de sus biomas existe en el mundo %q", worldName)
		if !complete {
			certainty = domain.Possibly
			reason = fmt.Sprintf("ninguno de sus biomas aparece en el mundo %q (algunas dimensiones no guardan su lista de biomas)", worldName)
		}
		out.Add(domain.Disablement{Target: structureTarget(s.ID), Certainty: certainty, By: StructureBiomesID, Reason: reason})
	}
	return nil
}

// BiomeReplacerID is the ID of the BiomeReplacer disabler.
const BiomeReplacerID = "biome-replacer"

// BiomeReplacer reads config/biome_replacer.properties ("old > new", "old >
// null", "#tag > …"): the old biomes no longer generate.
type BiomeReplacer struct{}

func (BiomeReplacer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: BiomeReplacerID, Phase: discovery.PhaseSpecific, Priority: 80,
		Applies: discovery.Applicability{RequiresMods: []string{"biome_replacer"}}}
}

func (BiomeReplacer) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	data, err := in.Files.ReadFile("config/biome_replacer.properties")
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "#") && !strings.Contains(line, ">") {
			continue
		}
		from, to, ok := strings.Cut(line, ">")
		if !ok {
			continue
		}
		from, to = strings.TrimSpace(from), strings.TrimSpace(to)
		var biomes []domain.ResourceID
		if tag, isTag := strings.CutPrefix(from, "#"); isTag {
			if tid, err := domain.ParseResourceID(tag); err == nil {
				biomes = in.World.BiomeTag(tid)
			}
		} else if id, err := domain.ParseResourceID(from); err == nil {
			biomes = []domain.ResourceID{id}
		}
		reason := "Biome Replacer lo reemplaza por " + to
		if to == "null" {
			reason = "Biome Replacer lo elimina"
		}
		for _, b := range biomes {
			out.Add(domain.Disablement{Target: domain.Target{Kind: domain.TargetBiome, ID: b}, Certainty: domain.Certainly, By: BiomeReplacerID, Reason: reason})
		}
	}
	return nil
}

// LevelDatFeaturesID is the ID of the LevelDatFeatures disabler.
const LevelDatFeaturesID = "leveldat-features"

// LevelDatFeatures: if generate_features is false in level.dat, no structures generate.
type LevelDatFeatures struct{}

func (LevelDatFeatures) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: LevelDatFeaturesID, Phase: discovery.PhaseSpecific, Priority: 110}
}

func (LevelDatFeatures) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	w := in.Files.Level()
	if w == nil || w.GenerateFeatures {
		return nil
	}
	for _, s := range in.World.Structures() {
		if s.ID.Namespace == "lostcities" {
			continue // Lost Cities ignore generate_features
		}
		out.Add(domain.Disablement{
			Target: structureTarget(s.ID), Certainty: domain.Certainly, By: LevelDatFeaturesID,
			Reason: "generación de estructuras desactivada en el mundo (generate_features = false en level.dat)",
		})
	}
	return nil
}

// DimensionAccessID is the ID of the DimensionAccess disabler.
const DimensionAccessID = "dimension-access"

// DimensionAccess disables biomes of dimensions that cannot be reached.
type DimensionAccess struct{}

func (DimensionAccess) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: DimensionAccessID, Phase: discovery.PhaseSpecific, Priority: 120}
}

func (DimensionAccess) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	hasPortalMods := in.Target.Mods["wormhole"] || in.Target.Mods["dimension_portal_api"] || in.Target.Mods["dimdoors"]
	var portalModNames []string
	if in.Target.Mods["wormhole"] { portalModNames = append(portalModNames, "wormhole") }
	if in.Target.Mods["dimension_portal_api"] { portalModNames = append(portalModNames, "dimension_portal_api") }
	if in.Target.Mods["dimdoors"] { portalModNames = append(portalModNames, "dimdoors") }
	portalModsStr := strings.Join(portalModNames, ", ")

	netherDisabled := false
	serverData, err := in.Files.ReadFile("server.properties")
	if err == nil {
		for _, line := range strings.Split(string(serverData), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "allow-nether=false") {
				netherDisabled = true
				break
			}
		}
	}
	
	if netherDisabled {
		certainty := domain.Certainly
		reason := "Dimensión inacesible: allow-nether=false en server.properties y no hay mods de portales alternativos"
		if hasPortalMods {
			certainty = domain.Possibly
			reason = "El acceso estándar está bloqueado (allow-nether=false), pero podría ser accesible vía mods: " + portalModsStr
		}
		
		if w := in.Files.Level(); w != nil {
			if d, ok := w.Dimension("minecraft:the_nether"); ok && d.Biomes != nil {
				for _, b := range d.Biomes {
					out.Add(domain.Disablement{
						Target: domain.Target{Kind: domain.TargetBiome, ID: b}, Certainty: certainty, By: DimensionAccessID,
						Reason: reason,
					})
				}
			}
		}
	}

	strongholdDisabled := false
	w := in.Files.Level()
	if w != nil && !w.GenerateFeatures {
		strongholdDisabled = true
	}
	
	if !strongholdDisabled {
		if cfgData, err := in.Files.ReadFile("config/structurify.json"); err == nil {
			var cfg struct {
				Structures []struct {
					Name       string `json:"name"`
					IsDisabled bool   `json:"is_disabled"`
				} `json:"structures"`
			}
			if json.Unmarshal(cfgData, &cfg) == nil {
				for _, e := range cfg.Structures {
					if e.Name == "minecraft:stronghold" && e.IsDisabled {
						strongholdDisabled = true
						break
					}
				}
			}
		}
	}

	if strongholdDisabled {
		certainty := domain.Certainly
		reason := "Dimensión inacesible: los Strongholds están desactivados y no hay mods de portales alternativos"
		if hasPortalMods {
			certainty = domain.Possibly
			reason = "El acceso estándar está bloqueado (sin Strongholds), pero podría ser accesible vía mods: " + portalModsStr
		}
		
		if w := in.Files.Level(); w != nil {
			if d, ok := w.Dimension("minecraft:the_end"); ok && d.Biomes != nil {
				for _, b := range d.Biomes {
					out.Add(domain.Disablement{
						Target: domain.Target{Kind: domain.TargetBiome, ID: b}, Certainty: certainty, By: DimensionAccessID,
						Reason: reason,
					})
				}
			}
		}
	}

	return nil
}
