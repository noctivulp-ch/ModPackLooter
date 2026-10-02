package modpack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// The vanilla jar only ships en_us. Other languages live in the launcher's
// asset store: assets/indexes/<id>.json maps "minecraft/lang/es_es.json" to a
// hash stored at assets/objects/<first two chars>/<hash>.
// Every regional variant of the chosen language is loaded (es_ar, es_es,
// es_mx…) so the site can fall back between them.
func (mp *Modpack) loadAssetLangs(explicit, lang string, diags *domain.Diagnostics) resources.Pack {
	family, _, _ := strings.Cut(lang, "_")
	if family == "" || lang == "en_us" {
		return nil
	}
	candidates := []string{explicit}
	if explicit == "" {
		candidates = []string{
			filepath.Join(mp.Root, "assets"),
			filepath.Join(mp.Root, "..", "..", "Install", "assets"), // CurseForge
			filepath.Join(mp.Root, "..", "..", "..", "assets"),      // Prism / MultiMC
			filepath.Join(userHome(), ".minecraft", "assets"),       // launcher oficial
		}
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		indexes, _ := filepath.Glob(filepath.Join(dir, "indexes", "*.json"))
		// Newer indexes first: their numeric ids grow with the game version.
		sort.Sort(sort.Reverse(sort.StringSlice(indexes)))
		files := map[string]string{}
		for _, idx := range indexes {
			data, err := os.ReadFile(idx)
			if err != nil {
				continue
			}
			var doc struct {
				Objects map[string]struct {
					Hash string `json:"hash"`
				} `json:"objects"`
			}
			if json.Unmarshal(data, &doc) != nil {
				continue
			}
			for key, obj := range doc.Objects {
				code, ok := strings.CutPrefix(key, "minecraft/lang/")
				if !ok || !strings.HasPrefix(code, family+"_") {
					continue
				}
				packPath := "assets/" + key
				if _, done := files[packPath]; done || len(obj.Hash) < 2 {
					continue
				}
				real := filepath.Join(dir, "objects", obj.Hash[:2], obj.Hash)
				if _, err := os.Stat(real); err == nil {
					files[packPath] = real
				}
			}
		}
		if len(files) > 0 {
			diags.Add(domain.LevelInfo, stage, dir, "traducciones vanilla cargadas de los assets del launcher")
			return resources.NewMappedPack("assets del launcher", resources.KindVanilla, files)
		}
	}
	diags.Add(domain.LevelInfo, stage, "", "no se encontraron las traducciones vanilla (%s) en los assets del launcher; los objetos vanilla saldrán en inglés. Usa --assets-dir <carpeta assets>", lang)
	return nil
}
