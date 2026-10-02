package modpack

import (
	"os"
	"path/filepath"

	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
	"github.com/EnierAragon/ModPackLooter/app/internal/world"
	"strings"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
	"github.com/EnierAragon/ModPackLooter/app/internal/testkit"
)

func TestOpenMergesPacksInGameOrder(t *testing.T) {
	in := testkit.NewInstance(t)
	vanilla := in.Jar("vanilla/1.20.1.jar", testkit.Files{
		"data/minecraft/loot_tables/chests/desert_pyramid.json": `{"pools":[]}`,
		"data/minecraft/tags/worldgen/biome/is_ocean.json":      `{"values":["minecraft:ocean"]}`,
		"assets/minecraft/lang/en_us.json":                      `{"item.minecraft.diamond":"Diamond"}`,
	})
	nested := testkit.ZipBytes(t, testkit.Files{
		"META-INF/mods.toml":                          testkit.ModsToml("innerlib", "Inner Lib", "[1.20.1,1.21)"),
		"data/innerlib/loot_tables/chests/inner.json": `{"pools":[]}`,
	})
	in.Jar("mods/towers-1.0.jar", testkit.Files{
		"META-INF/mods.toml":                                    testkit.ModsToml("towers", "Towers", "[1.20.1,1.21)"),
		"META-INF/jarjar/innerlib.jar":                          string(nested),
		"data/towers/loot_tables/chests/tower.json":             `{"pools":[]}`,
		"data/minecraft/loot_tables/chests/desert_pyramid.json": `{"pools":[],"from":"towers"}`,
		"data/towers/worldgen/structure/tower.json":             `{}`,
		"data/towers/structures/tower/top.nbt":                  "nbt",
		"data/minecraft/tags/worldgen/biome/is_ocean.json":      `{"values":["towers:deep"]}`,
		"assets/towers/lang/es_es.json":                         `{"item.towers.key":"Llave"}`,
	})
	in.Jar("mods/broken.jar.disabled", testkit.Files{"META-INF/mods.toml": testkit.ModsToml("off", "Off", "[1.20.1]")})
	in.Dir("datapacks/tweaks", testkit.Files{
		"pack.mcmeta": `{}`,
		"data/minecraft/loot_tables/chests/desert_pyramid.json": `{"pools":[],"from":"datapack"}`,
	})
	in.Dir("kubejs", testkit.Files{"data/kjs/loot_tables/chests/scripted.json": `{"pools":[]}`})

	diags := &domain.Diagnostics{}
	mp, err := Open(Options{Path: in.Root, MinecraftJar: vanilla, Diagnostics: diags})
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()

	if mp.MCVersion.String() != "1.20.1" || mp.Loader != domain.LoaderForge {
		t.Errorf("objetivo detectado = %s %s (%s)", mp.MCVersion, mp.Loader, mp.VersionSource)
	}
	ids := mp.ModIDs()
	for _, want := range []string{"towers", "innerlib"} {
		if !ids[want] {
			t.Errorf("falta el mod %q; mods = %v", want, ids)
		}
	}
	if ids["off"] {
		t.Error("los .jar.disabled no deben cargarse")
	}

	ix := mp.Index
	gotTables := map[string]bool{}
	for _, id := range ix.IDs(resources.TypeLootTable) {
		gotTables[id.String()] = true
	}
	for _, want := range []string{"minecraft:chests/desert_pyramid", "towers:chests/tower", "innerlib:chests/inner", "kjs:chests/scripted"} {
		if !gotTables[want] {
			t.Errorf("falta la loot table %s; hay %v", want, gotTables)
		}
	}
	pyramid, _ := ix.Lookup(resources.TypeLootTable, domain.MustParseResourceID("minecraft:chests/desert_pyramid"))
	var body struct{ From string }
	if err := pyramid.ReadJSON(&body); err != nil || body.From != "datapack" {
		t.Errorf("el datapack debe ganar: from=%q err=%v (%s)", body.From, err, pyramid)
	}
	if n := len(ix.All(resources.TypeBiomeTag, domain.MustParseResourceID("minecraft:is_ocean"))); n != 2 {
		t.Errorf("los tags deben conservar todas sus fuentes, hay %d", n)
	}
	if ix.Count(resources.TypeStructureNBT) != 1 || ix.Count(resources.TypeStructure) != 1 {
		t.Errorf("plantillas NBT = %d, estructuras = %d", ix.Count(resources.TypeStructureNBT), ix.Count(resources.TypeStructure))
	}
	if lang := ix.Lang("es_es", nil); lang["item.towers.key"] != "Llave" {
		t.Errorf("lang es_es = %v", lang)
	}
	if !mp.HasVanilla {
		t.Error("debe cargar el jar vanilla indicado")
	}
}

func TestOpenRequiresModsFolder(t *testing.T) {
	_, err := Open(Options{Path: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "mods/") {
		t.Errorf("se esperaba un error que mencione mods/, fue %v", err)
	}
}

func TestOpenFindsGameDirInsidePrismInstance(t *testing.T) {
	in := testkit.NewInstance(t)
	// A Prism instance keeps mods/ inside .minecraft/, not at its root.
	if err := os.Remove(filepath.Join(in.Root, "mods")); err != nil {
		t.Fatal(err)
	}
	in.Jar(".minecraft/mods/a.jar", testkit.Files{"META-INF/mods.toml": testkit.ModsToml("a", "A", "[1.20.1,1.21)")})
	in.Dir("", testkit.Files{"mmc-pack.json": `{"components":[{"uid":"net.minecraft","version":"1.20.1"},{"uid":"net.neoforged"}]}`})
	mp, err := Open(Options{Path: in.Root, Diagnostics: &domain.Diagnostics{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()
	if filepath.Base(mp.Root) != ".minecraft" || mp.Loader != domain.LoaderNeoForge || mp.VersionSource != "mmc-pack.json" {
		t.Errorf("root=%s loader=%s fuente=%s", mp.Root, mp.Loader, mp.VersionSource)
	}
	if mp.HasVanilla {
		t.Error("no debería encontrar vanilla en una instancia sintética")
	}
}

func TestVanillaTranslationsFromLauncherAssets(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/a.jar", testkit.Files{"META-INF/mods.toml": testkit.ModsToml("a", "A", "[1.20.1,1.21)")})
	in.Dir("assets", testkit.Files{
		"indexes/5.json":    `{"objects":{"minecraft/lang/es_es.json":{"hash":"ab12cd"},"minecraft/lang/es_ar.json":{"hash":"ef34"},"minecraft/lang/fr_fr.json":{"hash":"aa99"}}}`,
		"objects/ab/ab12cd": `{"item.minecraft.diamond":"Diamante"}`,
		"objects/ef/ef34":   `{"item.minecraft.potato":"Papa"}`,
		"objects/aa/aa99":   `{"item.minecraft.diamond":"Diamant"}`,
	})
	in.Dir("", testkit.Files{"options.txt": "version:3465\nlang:es_ar\n"})
	mp, err := Open(Options{Path: in.Root, Diagnostics: &domain.Diagnostics{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()
	if mp.Lang != "es_ar" || mp.LangSource != "options.txt" {
		t.Errorf("idioma = %s (%s)", mp.Lang, mp.LangSource)
	}
	if got := mp.Index.Lang("es_es", nil)["item.minecraft.diamond"]; got != "Diamante" {
		t.Errorf("traducción vanilla es_es = %q", got)
	}
	if got := mp.Index.Lang("es_ar", nil)["item.minecraft.potato"]; got != "Papa" {
		t.Errorf("traducción vanilla es_ar = %q", got)
	}
	if len(mp.Index.Lang("fr_fr", nil)) != 0 {
		t.Error("no deben cargarse idiomas de otra familia")
	}
}

func TestDedicatedServerLayout(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/a.jar", testkit.Files{"META-INF/mods.toml": testkit.ModsToml("a", "A", "[1.20.1,1.21)")})
	// server.jar is a bundler: the real server, with its data, is nested.
	inner := testkit.ZipBytes(t, testkit.Files{"data/minecraft/loot_tables/chests/igloo_chest.json": `{"pools":[]}`})
	in.Jar("server.jar", testkit.Files{"net/minecraft/bundler/Main.class": "x", "META-INF/versions/1.20.1/server-1.20.1.jar": string(inner)})
	in.Dir("", testkit.Files{"server.properties": "motd=hola\nlevel-name=mundo\n"})
	dir := filepath.Join(in.Root, "mundo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "level.dat"), nbt.Encode(nbt.Compound{"Data": world.Sample()}), 0o644); err != nil {
		t.Fatal(err)
	}
	mp, err := Open(Options{Path: in.Root, Diagnostics: &domain.Diagnostics{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()
	if !mp.HasVanilla {
		t.Error("debe cargar los datos del jar interno del server.jar")
	}
	if _, ok := mp.Index.Lookup(resources.TypeLootTable, domain.MustParseResourceID("minecraft:chests/igloo_chest")); !ok {
		t.Error("falta la loot table vanilla del bundler")
	}
	if mp.World == nil || !mp.WorldAuto || mp.World.Name != "Prueba" {
		t.Errorf("mundo del servidor = %+v auto=%v", mp.World, mp.WorldAuto)
	}
}

func TestServerUsesPrismAssets(t *testing.T) {
	// A dedicated server has no assets: the Prism default folder is used.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", "")
	assets := filepath.Join(home, ".local", "share", "PrismLauncher", "assets")
	for path, content := range map[string]string{
		"indexes/5.json":    `{"objects":{"minecraft/lang/es_es.json":{"hash":"ab12cd"}}}`,
		"objects/ab/ab12cd": `{"item.minecraft.diamond":"Diamante"}`,
	} {
		full := filepath.Join(assets, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	in := testkit.NewInstance(t)
	in.Jar("mods/a.jar", testkit.Files{"META-INF/mods.toml": testkit.ModsToml("a", "A", "[1.20.1,1.21)")})
	in.Dir("", testkit.Files{"server.properties": "level-name=world\n"})
	mp, err := Open(Options{Path: in.Root, Diagnostics: &domain.Diagnostics{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()
	if got := mp.Index.Lang("es_es", nil)["item.minecraft.diamond"]; got != "Diamante" {
		t.Errorf("traducción desde los assets de Prism = %q", got)
	}
}
