// Package domain holds the core concepts of ModPackLooter. It must not
// import anything from infrastructure or user interfaces.
package domain

import (
	"fmt"
	"strings"
)

// ResourceID is a namespaced identifier such as "minecraft:chests/desert_pyramid".
type ResourceID struct {
	Namespace string
	Path      string
}

// ParseResourceID reads "namespace:path"; a missing namespace means "minecraft".
func ParseResourceID(s string) (ResourceID, error) {
	ns, path, found := strings.Cut(strings.TrimSpace(s), ":")
	if !found {
		ns, path = "minecraft", ns
	}
	if ns == "" || path == "" {
		return ResourceID{}, fmt.Errorf("identificador no válido: %q", s)
	}
	return ResourceID{Namespace: ns, Path: path}, nil
}

// MustParseResourceID is ParseResourceID for constants known to be valid.
func MustParseResourceID(s string) ResourceID {
	id, err := ParseResourceID(s)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ResourceID) String() string { return id.Namespace + ":" + id.Path }

// Loader is the mod loader of a modpack.
type Loader string

const (
	LoaderForge    Loader = "forge"
	LoaderNeoForge Loader = "neoforge"
	LoaderFabric   Loader = "fabric"
)

// SourceKind says how a loot table is obtained in game.
type SourceKind string

const (
	KindContainer   SourceKind = "container"   // chests, barrels, vaults, chest minecarts…
	KindArchaeology SourceKind = "archaeology" // suspicious sand/gravel
	KindEntity      SourceKind = "entity"      // mob drops
	KindFishing     SourceKind = "fishing"
	KindGameplay    SourceKind = "gameplay" // villager gifts, cat gifts, piglin barter…
	KindBlock       SourceKind = "block"
	KindUnknown     SourceKind = "unknown"
)

// Confidence ranks how trustworthy an association is. Higher is better.
type Confidence int

const (
	ConfidenceUnknown   Confidence = iota // only the loot table path is known
	ConfidenceHeuristic                   // inferred, e.g. by matching names
	ConfidenceKnown                       // built-in knowledge for code-generated content
	ConfidenceManual                      // user override in the project config
	ConfidenceExact                       // read directly from game data
)

func (c Confidence) String() string {
	switch c {
	case ConfidenceExact:
		return "exacta"
	case ConfidenceManual:
		return "manual"
	case ConfidenceKnown:
		return "conocida"
	case ConfidenceHeuristic:
		return "heurística"
	default:
		return "desconocida"
	}
}

// OwnerKind is what a loot source belongs to.
type OwnerKind string

const (
	OwnerNone      OwnerKind = ""
	OwnerStructure OwnerKind = "structure"
	OwnerEntity    OwnerKind = "entity"
	OwnerBiome     OwnerKind = "biome"
	OwnerTemplate  OwnerKind = "template" // a template no known structure uses
)

// Owner is the structure, mob or biome a loot source is attached to.
type Owner struct {
	Kind OwnerKind
	ID   ResourceID
}

// Evidence records where an association was found, for diagnostics and the site.
type Evidence struct {
	DiscoveredBy string // discoverer ID
	Detail       string // file, template piece, config key…
}

// Note is extra information about a source, e.g. "loot per player (Lootr)".
type Note struct {
	Key  string // stable identifier, e.g. "lootr.refresh"
	Text string // human readable, already localised
}

// LootSource links a loot table to the place it is obtained from.
type LootSource struct {
	LootTable  ResourceID
	Kind       SourceKind
	Owner      Owner
	Confidence Confidence
	// Container is the block or entity that holds the loot, when known
	// (e.g. "minecraft:chest", "minecraft:chest_minecart").
	Container string
	// Share is the fraction of this owner's containers that use the table,
	// when the owner picks among several tables (0 means not applicable).
	Share    float64
	Evidence []Evidence
	Notes    []Note
}

// OwnerInfo describes an owner a discoverer knows about, such as a structure
// that is not declared in data (e.g. a Lost Cities city).
type OwnerInfo struct {
	Owner Owner
	Name  string
	// Biomes where the owner can appear; BiomesNote explains when the list is
	// not exhaustive or not known.
	Biomes     []ResourceID
	BiomesNote string
}
