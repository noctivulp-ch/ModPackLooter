package site_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/modpack"
	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
	"github.com/EnierAragon/ModPackLooter/app/internal/plugins"
	"github.com/EnierAragon/ModPackLooter/app/internal/site"
	"github.com/EnierAragon/ModPackLooter/app/internal/testkit"
)

const table = `{"type":"minecraft:chest","pools":[{"rolls":{"min":2,"max":3},"entries":[
	{"type":"minecraft:item","name":"minecraft:diamond","weight":1},
	{"type":"minecraft:item","name":"towers:key","weight":3,"functions":[{"function":"minecraft:set_count","count":{"min":1,"max":4}}]}]}]}`

func buildSite(t *testing.T) string {
	t.Helper()
	in := testkit.NewInstance(t)
	tpl := string(nbt.Encode(nbt.Compound{
		"palette":  nbt.List{nbt.Compound{"Name": "minecraft:barrel"}},
		"blocks":   nbt.List{nbt.Compound{"pos": nbt.List{int32(0), int32(0), int32(0)}, "state": int32(0), "nbt": nbt.Compound{"LootTable": "towers:chests/tower"}}},
		"entities": nbt.List{},
	}))
	in.Jar("mods/towers.jar", testkit.Files{
		"META-INF/mods.toml":                            testkit.ModsToml("towers", "Towers & Ruins", "[1.20.1,1.21)"),
		"data/towers/worldgen/structure/tower.json":     `{"type":"minecraft:jigsaw","start_pool":"towers:start","biomes":["minecraft:plains","towers:ash_fields"]}`,
		"data/towers/worldgen/template_pool/start.json": `{"elements":[{"weight":1,"element":{"element_type":"minecraft:single_pool_element","location":"towers:top"}}]}`,
		"data/towers/structures/top.nbt":                tpl,
		"data/towers/loot_tables/chests/tower.json":     table,
		"data/towers/loot_tables/entities/ghoul.json":   `{"type":"minecraft:entity","pools":[{"rolls":1,"entries":[{"type":"minecraft:item","name":"minecraft:bone"}]}]}`,
		"data/towers/loot_tables/blocks/brick.json":     `{"type":"minecraft:block","pools":[{"rolls":1,"entries":[{"type":"minecraft:item","name":"towers:brick"}]}]}`,
		"assets/towers/lang/es_es.json":                 `{"item.towers.key":"Llave <rúnica>","biome.towers.ash_fields":"Campos de ceniza"}`,
	})
	res, err := analysis.Analyzer{Discoverers: plugins.Discoverers(), Enrichers: plugins.Enrichers(), Disablers: plugins.Disablers()}.
		Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	out := filepath.Join(t.TempDir(), "site")
	stats, err := site.Build(res, site.Options{OutDir: out, Title: "Prueba", Metal: "jade", AppVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Items != 3 || stats.Owners != 1 || stats.Biomes != 2 {
		t.Errorf("estadísticas = %+v", stats)
	}
	return out
}

var reLink = regexp.MustCompile(`(?:href|src)="([^"#]*)(?:#[^"]*)?"`)

func TestSiteLinksAreRelativeAndResolve(t *testing.T) {
	out := buildSite(t)
	pages := 0
	err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil || filepath.Ext(path) != ".html" {
			return err
		}
		pages++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range reLink.FindAllStringSubmatch(string(data), -1) {
			link := m[1]
			if link == "" {
				continue
			}
			if strings.HasPrefix(link, "/") || strings.Contains(link, "://") {
				t.Errorf("%s: enlace no relativo %q", path, link)
				continue
			}
			target := filepath.Join(filepath.Dir(path), filepath.FromSlash(link))
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: enlace roto %q", path, link)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if pages < 10 {
		t.Errorf("solo se generaron %d páginas", pages)
	}
}

func TestSiteContent(t *testing.T) {
	out := buildSite(t)
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, f := range []string{".nojekyll", "assets/style.css", "assets/app.js", "acerca/diagnostics.json"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("falta %s", f)
		}
	}
	idx := read("assets/search-index.js")
	if !strings.HasPrefix(idx, "window.MPL_INDEX=[") || !strings.Contains(idx, `"n":"Llave `+"\\"+`u003crúnica`+"\\"+`u003e"`) {
		t.Errorf("índice de búsqueda = %.200s", idx)
	}
	key := read("objetos/towers/key/index.html")
	for _, want := range []string{"Llave &lt;rúnica&gt;", "Tower", "1–4", "Campos de ceniza", `data-metal="jade"`} {
		if !strings.Contains(key, want) {
			t.Errorf("la ficha de la llave no contiene %q", want)
		}
	}
	if strings.Contains(read("objetos/index.html"), "towers:brick") || fileExists(filepath.Join(out, "objetos/towers/brick")) {
		t.Error("los drops de bloques no deben aparecer como objetos")
	}
	if !strings.Contains(read("fuentes/index.html"), "Criatura") {
		t.Error("las tablas de criaturas deben aparecer en Otras fuentes")
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
