package site

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type CompletenessLevel int

const (
	Level1 CompletenessLevel = 1 // Incompleto
	Level2 CompletenessLevel = 2 // Parcial
	Level3 CompletenessLevel = 3 // Completo
)

type CompletenessReport struct {
	Level   CompletenessLevel
	Name    string
	Missing []string
}

func EvaluateCompleteness(m *Model, outDir string) {
	var items, buildings, mobs, biomes []CompletenessReport
	
	// Evaluate Items
	for _, it := range m.Items {
		var missing []string
		if it.Unobtainable { continue }
		
		if len(it.Sources) == 0 { missing = append(missing, "Fuentes de obtención") }
		
		level := Level3
		if len(missing) == 1 {
			level = Level1
		}
		items = append(items, CompletenessReport{Level: level, Name: it.Name, Missing: missing})
	}
	
	// Evaluate Buildings (Lost Cities)
	if m.LostCities != nil {
		for _, b := range m.LostCities.Buildings {
			var missing []string
			if b.Containers == 0 && len(b.Children) == 0 { missing = append(missing, "Contenedores (vacío)") }
			if len(b.Haul) == 0 { missing = append(missing, "Loot (sin objetos)") }
			if len(b.Blocks) == 0 { missing = append(missing, "Bloques de construcción") }
			
			level := Level3
			if len(missing) >= 2 {
				level = Level1
			} else if len(missing) > 0 {
				level = Level2
			}
			buildings = append(buildings, CompletenessReport{Level: level, Name: b.Name, Missing: missing})
		}
	}
	
	// Evaluate Mobs
	for _, mob := range m.Creatures {
		var missing []string
		if len(mob.Natural) == 0 && len(mob.Structures) == 0 && len(mob.From) == 0 && len(mob.Other) == 0 { missing = append(missing, "Biomas/Estructuras de spawn") }
		if mob.Table == nil || len(mob.Table.Drops) == 0 { missing = append(missing, "Drops/Loot") }
		
		level := Level3
		if len(missing) == 2 {
			level = Level1
		} else if len(missing) > 0 {
			level = Level2
		}
		mobs = append(mobs, CompletenessReport{Level: level, Name: mob.Name, Missing: missing})
	}
	
	// Evaluate Biomes
	for _, b := range m.Biomes {
		var missing []string
		if b.Status.Disabled() { continue }
		
		hasStr := false
		for _, o := range m.Owners {
			if o.Kind == "structure" {
				for _, biome := range o.Biomes {
					if biome == b { hasStr = true; break }
				}
			}
		}
		
		if !hasStr { missing = append(missing, "Estructuras") }
		
		level := Level3
		if len(missing) > 0 {
			level = Level2
		}
		biomes = append(biomes, CompletenessReport{Level: level, Name: b.Name, Missing: missing})
	}
	
	// Generate output
	f, err := os.Create(filepath.Join(outDir, "reporte_completitud.txt"))
	if err != nil { return }
	defer f.Close()
	
	writeCat := func(title string, list []CompletenessReport) {
		counts := map[CompletenessLevel]int{Level1: 0, Level2: 0, Level3: 0}
		for _, r := range list { counts[r.Level]++ }
		
		fmt.Fprintf(f, "=== %s ===\n", title)
		fmt.Fprintf(f, "Total: %d | Nivel 3 (Completos): %d | Nivel 2 (Parciales): %d | Nivel 1 (Incompletos): %d\n\n", len(list), counts[Level3], counts[Level2], counts[Level1])
		
		// Sort to show Level 1 first, then 2
		sort.Slice(list, func(i, j int) bool { return list[i].Level < list[j].Level })
		
		for _, r := range list {
			if r.Level == Level3 { continue }
			fmt.Fprintf(f, "[Nivel %d] %s -> Falta: %v\n", r.Level, r.Name, r.Missing)
		}
		fmt.Fprintf(f, "\n")
		
		fmt.Printf("· %s: %d Completos (N3), %d Parciales (N2), %d Incompletos (N1)\n", title, counts[Level3], counts[Level2], counts[Level1])
	}
	
	fmt.Println("\n=== Reporte de Completitud de Datos ===")
	writeCat("Objetos", items)
	writeCat("Edificios (Lost Cities)", buildings)
	writeCat("Criaturas", mobs)
	writeCat("Biomas", biomes)
	fmt.Printf("El reporte detallado se ha guardado en %s\n\n", filepath.Join(outDir, "reporte_completitud.txt"))
}
