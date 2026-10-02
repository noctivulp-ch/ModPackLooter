package world

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
)

// Write creates a level.dat for tests in other packages too.
func writeLevel(t *testing.T, dir string, data nbt.Compound) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "level.dat"), nbt.Encode(nbt.Compound{"Data": data}), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	writeLevel(t, dir, Sample())
	w, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.Name != "Prueba" || w.Version != "1.20.1" || len(w.Dimensions) != 3 {
		t.Fatalf("mundo = %+v", w)
	}
	over, _ := w.Dimension("minecraft:overworld")
	if len(over.Biomes) != 2 || over.Biomes[0].String() != "minecraft:desert" {
		t.Errorf("biomas del overworld = %v", over.Biomes)
	}
	end, _ := w.Dimension("minecraft:the_end")
	if len(end.Biomes) != 5 {
		t.Errorf("biomas del end = %v", end.Biomes)
	}
	if all, complete := w.AllBiomes(); complete || !all[over.Biomes[1]] {
		t.Errorf("AllBiomes = %v completo=%v (la dimensión con preset no es completa)", all, complete)
	}
	if !w.PackDisabled("extra.zip") || w.PackDisabled("otro.zip") {
		t.Error("PackDisabled")
	}
}
