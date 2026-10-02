package resources

import (
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
)

// Canonical resource types. The index always uses these names (the 1.21+
// singular spelling), whatever version the files come from, so discoverers
// never deal with per-version folder names.
const (
	TypeLootTable       = "loot_table"
	TypeStructureNBT    = "structure"
	TypeStructure       = "worldgen/structure"
	TypeStructureSet    = "worldgen/structure_set"
	TypeTemplatePool    = "worldgen/template_pool"
	TypeProcessorList   = "worldgen/processor_list"
	TypeBiome           = "worldgen/biome"
	TypeBiomeTag        = "tags/worldgen/biome"
	TypeStructureTag    = "tags/worldgen/structure"
	TypeItemTag         = "tags/item"
	TypeBlockTag        = "tags/block"
	TypeEntityTypeTag   = "tags/entity_type"
	TypeLootModifier    = "loot_modifiers"
	TypeForgeBiomeMod   = "forge/biome_modifier"
	TypeNeoBiomeMod     = "neoforge/biome_modifier"
	TypeLostCitiesCond  = "lostcities/conditions"
	TypeLostCitiesPal   = "lostcities/palettes"
	TypeLostCitiesParts = "lostcities/parts"
	TypeStarcatcherFish = "starcatcher/fish"
	TypeTideFish        = "fishing/fish"
	TypeTideLoot        = "fishing/loot"
	TypeTideCrate       = "fishing/crates"
	TypeEnchantment     = "enchantment" // data-driven since 1.21
	TypeEnchantmentTag  = "tags/enchantment"
	TypeInstrumentTag   = "tags/instrument"
)

// Layout maps the folder names of one range of Minecraft versions to the
// canonical types. A new version with renamed folders only needs a new Layout.
type Layout struct {
	Name     string
	Versions mcversion.Range
	// Folders maps a raw folder (relative to data/<namespace>/) to a canonical type.
	Folders map[string]string
	// sorted raw folders, longest first, for prefix matching.
	sorted []string
}

func newLayout(name, versions string, folders map[string]string) *Layout {
	l := &Layout{Name: name, Versions: mcversion.MustParseRange(versions), Folders: folders}
	for raw := range folders {
		l.sorted = append(l.sorted, raw)
	}
	sort.Slice(l.sorted, func(i, j int) bool {
		if len(l.sorted[i]) != len(l.sorted[j]) {
			return len(l.sorted[i]) > len(l.sorted[j])
		}
		return l.sorted[i] < l.sorted[j]
	})
	return l
}

// common folders whose name did not change across the supported range.
func commonFolders() map[string]string {
	return map[string]string{
		"worldgen/structure":        TypeStructure,
		"worldgen/structure_set":    TypeStructureSet,
		"worldgen/template_pool":    TypeTemplatePool,
		"worldgen/processor_list":   TypeProcessorList,
		"worldgen/biome":            TypeBiome,
		"tags/worldgen/biome":       TypeBiomeTag,
		"tags/worldgen/structure":   TypeStructureTag,
		"loot_modifiers":            TypeLootModifier,
		"forge/biome_modifier":      TypeForgeBiomeMod,
		"neoforge/biome_modifier":   TypeNeoBiomeMod,
		"lostcities/conditions":     TypeLostCitiesCond,
		"lostcities/palettes":       TypeLostCitiesPal,
		"lostcities/parts":          TypeLostCitiesParts,
		"lostcities/worldstyles":    "lostcities/worldstyles",
		"lostcities/citystyles":     "lostcities/citystyles",
		"lostcities/buildings":      "lostcities/buildings",
		"lostcities/multibuildings": "lostcities/multibuildings",
		"lostcities/scattered":      "lostcities/scattered",
		"lostcities/styles":         "lostcities/styles",
		"starcatcher/fish":          TypeStarcatcherFish,
		"fishing/fish":              TypeTideFish,
		"fishing/loot":              TypeTideLoot,
		"fishing/crates":            TypeTideCrate,
		"enchantment":               TypeEnchantment,
		"tags/enchantment":          TypeEnchantmentTag,
		"tags/instrument":           TypeInstrumentTag,
	}
}

func with(base map[string]string, extra map[string]string) map[string]string {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

// Layouts known by the app, one per range of versions.
var layouts = []*Layout{
	newLayout("1.20.x (carpetas en plural)", ">=1.20 <1.21", with(commonFolders(), map[string]string{
		"loot_tables":       TypeLootTable,
		"structures":        TypeStructureNBT,
		"tags/items":        TypeItemTag,
		"tags/blocks":       TypeBlockTag,
		"tags/entity_types": TypeEntityTypeTag,
	})),
	newLayout("1.21+ (carpetas en singular)", ">=1.21", with(commonFolders(), map[string]string{
		"loot_table":       TypeLootTable,
		"structure":        TypeStructureNBT,
		"tags/item":        TypeItemTag,
		"tags/block":       TypeBlockTag,
		"tags/entity_type": TypeEntityTypeTag,
	})),
}

// LayoutFor returns the layout for a Minecraft version, or nil if unsupported.
func LayoutFor(v mcversion.Version) *Layout {
	for _, l := range layouts {
		if l.Versions.Contains(v) {
			return l
		}
	}
	return nil
}

// classify splits "data/<ns>/<folder>/<path>.<ext>" into canonical type,
// namespace and id path. ok is false for files the app does not use.
func (l *Layout) classify(path string) (typ, ns, id string, ok bool) {
	rest, found := strings.CutPrefix(path, "data/")
	if !found {
		return "", "", "", false
	}
	ns, rest, found = strings.Cut(rest, "/")
	if !found || ns == "" {
		return "", "", "", false
	}
	for _, raw := range l.sorted {
		if after, ok := strings.CutPrefix(rest, raw+"/"); ok {
			ext := ".json"
			if l.Folders[raw] == TypeStructureNBT {
				ext = ".nbt"
			}
			id, ok := strings.CutSuffix(after, ext)
			if !ok || id == "" {
				return "", "", "", false
			}
			return l.Folders[raw], ns, id, true
		}
	}
	return "", "", "", false
}
