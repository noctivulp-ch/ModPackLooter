// Package world reads a saved Minecraft world ("mundo modelo"): its
// level.dat (dimensions, generators, biome lists, enabled datapacks) and the
// per-world folders (datapacks/, serverconfig/). A world created with the
// modpack captures the defaults the pack assigns, so the app does not need a
// world-creation menu of its own.
package world

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
)

// Dimension is one dimension as the world generates it.
type Dimension struct {
	ID          domain.ResourceID
	Type        string
	Generator   string // e.g. minecraft:noise, lostworlds:lostworlds
	Settings    string
	BiomeSource string // e.g. minecraft:multi_noise, minecraft:the_end
	// Biomes is the exact biome list when the world stores it (multi_noise
	// with a parameter list, fixed, checkerboard, the_end). Nil means
	// unknown (a preset or a modded biome source).
	Biomes []domain.ResourceID
}

// World is a loaded world folder.
type World struct {
	Path       string
	Name       string
	Version    string
	Dimensions []Dimension
	Enabled    []string // DataPacks.Enabled, e.g. "vanilla", "mod:lootr", "file/x.zip"
	Disabled   []string
}

// The End's biome source is not stored as a list but is fixed.
var endBiomes = []string{"minecraft:the_end", "minecraft:end_highlands", "minecraft:end_midlands", "minecraft:small_end_islands", "minecraft:end_barrens"}

// Load reads <dir>/level.dat.
func Load(dir string) (*World, error) {
	data, err := os.ReadFile(filepath.Join(dir, "level.dat"))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el mundo %s: %w", dir, err)
	}
	doc, err := nbt.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("level.dat ilegible en %s: %w", dir, err)
	}
	d := doc.Compound("Data")
	if d == nil {
		return nil, fmt.Errorf("level.dat sin datos en %s", dir)
	}
	w := &World{Path: dir, Name: d.String("LevelName"), Version: d.Compound("Version").String("Name")}
	for _, v := range d.Compound("DataPacks").List("Enabled") {
		if s, ok := v.(string); ok {
			w.Enabled = append(w.Enabled, s)
		}
	}
	for _, v := range d.Compound("DataPacks").List("Disabled") {
		if s, ok := v.(string); ok {
			w.Disabled = append(w.Disabled, s)
		}
	}
	for name, v := range d.Compound("WorldGenSettings").Compound("dimensions") {
		dim, ok := v.(nbt.Compound)
		if !ok {
			continue
		}
		id, err := domain.ParseResourceID(name)
		if err != nil {
			continue
		}
		gen := dim.Compound("generator")
		bs := gen.Compound("biome_source")
		out := Dimension{ID: id, Type: dim.String("type"), Generator: gen.String("type"), BiomeSource: bs.String("type")}
		if s, ok := gen["settings"].(string); ok {
			out.Settings = s
		}
		out.Biomes = biomesOf(bs)
		w.Dimensions = append(w.Dimensions, out)
	}
	sort.Slice(w.Dimensions, func(i, j int) bool { return w.Dimensions[i].ID.String() < w.Dimensions[j].ID.String() })
	return w, nil
}

func biomesOf(bs nbt.Compound) []domain.ResourceID {
	set := map[string]bool{}
	switch bs.String("type") {
	case "minecraft:the_end":
		for _, b := range endBiomes {
			set[b] = true
		}
	case "minecraft:fixed":
		set[bs.String("biome")] = true
	default:
		// multi_noise with an explicit list, checkerboard and most modded
		// sources keep a "biomes" list of ids or of {biome, parameters}.
		list := bs.List("biomes")
		if list == nil {
			return nil
		}
		for _, e := range list {
			switch x := e.(type) {
			case string:
				set[x] = true
			case nbt.Compound:
				set[x.String("biome")] = true
			}
		}
	}
	var out []domain.ResourceID
	for s := range set {
		if id, err := domain.ParseResourceID(s); err == nil && s != "" {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	if len(out) == 0 {
		return nil
	}
	return out
}

// Dimension returns a dimension by id.
func (w *World) Dimension(id string) (Dimension, bool) {
	for _, d := range w.Dimensions {
		if d.ID.String() == id {
			return d, true
		}
	}
	return Dimension{}, false
}

// AllBiomes returns every biome of every dimension and whether the list is
// complete (false if any dimension has an unknown biome source).
func (w *World) AllBiomes() (map[domain.ResourceID]bool, bool) {
	out := map[domain.ResourceID]bool{}
	complete := true
	for _, d := range w.Dimensions {
		if d.Biomes == nil {
			complete = false
			continue
		}
		for _, b := range d.Biomes {
			out[b] = true
		}
	}
	return out, complete
}

// PackDisabled reports whether a world datapack ("file/<name>") is disabled.
func (w *World) PackDisabled(fileName string) bool {
	for _, d := range w.Disabled {
		if d == "file/"+fileName {
			return true
		}
	}
	return false
}
