package nbt

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDecodeRealTemplates decodes every vanilla template when MPL_MCMETA_DATA
// points at a checkout of misode/mcmeta (branch <version>-data).
func TestDecodeRealTemplates(t *testing.T) {
	root := os.Getenv("MPL_MCMETA_DATA")
	if root == "" {
		t.Skip("MPL_MCMETA_DATA no definido")
	}
	n, loot := 0, 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || filepath.Ext(path) != ".nbt" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := Decode(data)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		n++
		for _, b := range doc.List("blocks") {
			if c, ok := b.(Compound); ok && c.Compound("nbt").String("LootTable") != "" {
				loot++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d plantillas decodificadas, %d contenedores con LootTable", n, loot)
}
