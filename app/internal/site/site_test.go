package site_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

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
	res, err := plugins.Analyzer().
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
	// Items include the vanilla trades (built-in knowledge), so only a minimum.
	if stats.Items < 3 || stats.Owners != 1 || stats.Biomes != 2 {
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
	// Mod tabs only exist when the mod is present.
	if strings.Contains(read("index.html"), "Lost Cities") || fileExists(filepath.Join(out, "lostcities")) {
		t.Error("sin Lost Cities no debe haber pestaña de Lost Cities")
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func TestLostCitiesTab(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/lostcities.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("lostcities", "The Lost Cities", "[1.20.1,1.21)"),
		"data/lostcities/lostcities/worldstyles/standard.json": `{"citystyles":[
			{"factor":1,"citystyle":"citystyle_common"},
			{"factor":4,"citystyle":"citystyle_desert","biomes":{"if_any":["minecraft:desert","#minecraft:is_badlands"]}}]}`,
		"data/lostcities/lostcities/citystyles/citystyle_common.json": `{"selectors":{"buildings":[{"factor":3,"value":"shop"},{"factor":1,"value":"house"}]}}`,
		"data/lostcities/lostcities/citystyles/citystyle_desert.json": `{"inherit":"citystyle_common","selectors":{"buildings":[{"factor":4,"value":"house"}]}}`,
		"data/lostcities/lostcities/buildings/shop.json":              `{"palette":{"palette":[{"char":"C","block":"minecraft:chest[facing=north]","loot":"shop_loot"}]},"parts":[{"part":"shop_floor"},{"part":"shop_roof"}]}`,
		"data/lostcities/lostcities/buildings/house.json":             `{"parts":[{"part":"house_floor"}]}`,
		"data/lostcities/lostcities/parts/shop_floor.json":            `{"slices":[["CC#","#C#"]]}`,
		"data/lostcities/lostcities/parts/shop_roof.json":             `{"slices":[["###"]]}`,
		"data/lostcities/lostcities/parts/house_floor.json":           `{"refpalette":"house_pal","slices":[["B#"]]}`,
		"data/lostcities/lostcities/palettes/house_pal.json":          `{"palette":[{"char":"B","block":"minecraft:barrel","loot":"house_loot"}]}`,
		"data/lostcities/lostcities/conditions/shop_loot.json":        `{"values":[{"factor":3,"value":"lostcities:chests/shop"},{"factor":1,"value":"lostcities:chests/rare","range":"2,5"}]}`,
		"data/lostcities/lostcities/conditions/house_loot.json":       `{"values":[{"factor":1,"value":"lostcities:chests/shop"}]}`,
		"data/lostcities/loot_tables/chests/shop.json":                table,
		"data/lostcities/loot_tables/chests/rare.json":                table,
		"assets/lostcities/lang/es_es.json":                           `{"lostcities.citystyle.citystyle_desert":"Ciudad del desierto","lostcities.advancement.title.shop":"Tienda de la esquina"}`,
	})
	res, err := plugins.Analyzer().
		Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	out := filepath.Join(t.TempDir(), "site")
	if _, err := site.Build(res, site.Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	home := read("index.html")
	if !strings.Contains(home, `href="lostcities/index.html"`) {
		t.Error("falta la pestaña Lost Cities en la navegación")
	}
	index := read("lostcities/index.html")
	for _, want := range []string{"Ciudad del desierto", "Common", "Desierto", "Grupo «Badlands»"} {
		if want == "Desierto" {
			continue // without vanilla data the biome keeps its readable id
		}
		if !strings.Contains(index, want) {
			t.Errorf("la portada de Lost Cities no contiene %q", want)
		}
	}
	desert := read("lostcities/estilos/lostcities/citystyle_desert/index.html")
	for _, want := range []string{"Tienda de la esquina", "House", "Hereda de", "Common"} {
		if !strings.Contains(desert, want) {
			t.Errorf("el estilo desierto no contiene %q", want)
		}
	}
	shop := read("lostcities/edificios/lostcities/shop/index.html")
	for _, want := range []string{"3 contenedores con loot", "Shop Floor", "75 %", "25 %", "pisos 2 a 5", "1 parte sin contenedores", "../../../../tablas/lostcities/chests/shop/index.html"} {
		if !strings.Contains(shop, want) {
			t.Errorf("la página de la tienda no contiene %q", want)
		}
	}
	house := read("lostcities/edificios/lostcities/house/index.html")
	if !strings.Contains(house, "1 contenedor con loot") || !strings.Contains(house, "100 %") {
		t.Errorf("la casa debería tener un barril con su paleta de referencia")
	}
	if !strings.Contains(read("assets/search-index.js"), `"t":"l"`) {
		t.Error("los edificios deben estar en la búsqueda")
	}
}

func TestFishingTab(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/starcatcher.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("starcatcher", "Starcatcher", "[1.20.1,1.21)"),
		"data/minecraft/starcatcher/fish/nether_star.json": `{"base_chance":0,"rarity":"legendary",
			"catch_info":{"item":"minecraft:nether_star","entity":"minecraft:wither","always_spawn_entity":true},
			"restrictions":[{"type":"starcatcher:bait","baits":{"minecraft:wither_skeleton_skull":200}}]}`,
		"data/starcatcher/starcatcher/fish/trout.json": `{"base_chance":30,"rarity":"common","catch_info":{"item":"starcatcher:trout"},
			"restrictions":[{"type":"starcatcher:biome","biomes":["minecraft:river"],"biomes_tags":[],"biomes_blacklist":[],"biomes_blacklist_tags":[]},
			{"type":"starcatcher:fluid","fluids":["minecraft:water"]},
			{"type":"starcatcher:daytime_restriction","ranges":[{"first":13000,"second":23000}]}]}`,
		"data/starcatcher/starcatcher/fish/missing_mod.json": `{"base_chance":10,"catch_info":{"item":"othermod:fish"},"restrictions":[],
			"forge:conditions":[{"type":"forge:mod_loaded","modid":"othermod"}]}`,
		"data/minecraft/worldgen/biome/river.json":             `{"temperature":0.5}`,
		"data/minecraft/worldgen/biome/desert.json":            `{"temperature":2.0}`,
		"data/minecraft/tags/worldgen/biome/is_overworld.json": `{"values":["minecraft:river","minecraft:desert"]}`,
		"assets/starcatcher/lang/es_es.json":                   `{"item.starcatcher.trout":"Trucha"}`,
	})
	in.Jar("mods/tide.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("tide", "Tide", "[1.20.1,1.21)"),
		"data/tide/fishing/fish/freshwater/carp.json": `{"fish":"tide:carp","selection_weight":20,
			"conditions":[{"type":"tide:dimension","dimensions":["minecraft:overworld"]},{"type":"tide:fluid","fluid":"water"}],
			"modifiers":[{"type":"tide:temperature","preferred_temperature":0.5,"temperature_tolerance":1.0}]}`,
		"data/tide/fishing/fish/freshwater/perch.json": `{"fish":"tide:perch","selection_weight":20,
			"conditions":[{"type":"tide:fluid","fluid":"water"}]}`,
		"data/tide/fishing/fish/freshwater/compat.json":    `{"fish":"other:fish","associated_mods":["othermod"],"selection_weight":20}`,
		"data/tide/fishing/loot/junk.json":                 `{"loot_table":"tide:gameplay/fishing/junk","weight":10,"conditions":[{"type":"tide:above","y":40}]}`,
		"data/tide/loot_tables/gameplay/fishing/junk.json": `{"type":"minecraft:fishing","pools":[{"rolls":1,"entries":[{"type":"minecraft:item","name":"minecraft:stick"}]}]}`,
	})
	in.Jar("mods/vanilla-data.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("vanilladata", "Vanilla data", "[1.20.1,1.21)"),
		"data/minecraft/loot_tables/gameplay/fishing.json": `{"type":"minecraft:fishing","pools":[{"rolls":1,"entries":[
			{"type":"minecraft:loot_table","name":"minecraft:gameplay/fishing/junk","weight":10,"quality":-2},
			{"type":"minecraft:loot_table","name":"minecraft:gameplay/fishing/treasure","weight":5,"quality":2,
			 "conditions":[{"condition":"minecraft:entity_properties","entity":"this","predicate":{"type_specific":{"type":"fishing_hook","in_open_water":true}}}]},
			{"type":"minecraft:loot_table","name":"minecraft:gameplay/fishing/fish","weight":85,"quality":-1}]}]}`,
	})
	res, err := plugins.Analyzer().
		Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	out := filepath.Join(t.TempDir(), "site")
	if _, err := site.Build(res, site.Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if !strings.Contains(read("index.html"), `href="pesca/index.html"`) {
		t.Error("falta la pestaña Pesca")
	}
	star := read("objetos/minecraft/nether_star/index.html")
	for _, want := range []string{"Pescando", "La forma más probable", "Starcatcher", "solo con cebo:", "Wither Skeleton Skull", "aparece Wither al pescarlo", "en cualquier bioma"} {
		if !strings.Contains(star, want) {
			t.Errorf("la Estrella del Nether no contiene %q", want)
		}
	}
	river := read("pesca/starcatcher/minecraft/river/index.html")
	for _, want := range []string{"Trucha", "100 %", "solo de 19:00 a 05:00", "Con cebo", "con Wither Skeleton Skull"} {
		if !strings.Contains(river, want) {
			t.Errorf("Starcatcher en el río no contiene %q", want)
		}
	}
	if strings.Contains(read("pesca/starcatcher/index.html"), "othermod") || fileExists(filepath.Join(out, "objetos/othermod")) {
		t.Error("los peces de mods ausentes no deben aparecer")
	}
	if fileExists(filepath.Join(out, "pesca/starcatcher/minecraft/desert/index.html")) && strings.Contains(read("pesca/starcatcher/minecraft/desert/index.html"), "Trucha") {
		t.Error("la trucha solo vive en el río")
	}
	// Tide: carp prefers 0.5 °, so in the desert (2.0) it loses all weight.
	tideRiver := read("pesca/tide/minecraft/river/index.html")
	tideDesert := read("pesca/tide/minecraft/desert/index.html")
	if !strings.Contains(tideRiver, "Carp") || !strings.Contains(tideRiver, "50 %") {
		t.Errorf("Tide en el río debería repartir carpa y perca al 50 %%")
	}
	if strings.Contains(tideDesert, "Carp") || !strings.Contains(tideDesert, "100 %") {
		t.Errorf("Tide en el desierto: la carpa no debería salir")
	}
	if !strings.Contains(tideRiver, "por encima de Y=40") || !strings.Contains(tideRiver, "peso 10") {
		t.Error("falta el botín de Tide con su condición")
	}
	vanilla := read("pesca/minecraft/index.html")
	for _, want := range []string{"En aguas abiertas", "Fuera de aguas abiertas", "85 %", "89 %", "Tide reemplaza la caña vanilla"} {
		if !strings.Contains(vanilla, want) {
			t.Errorf("la pesca vanilla no contiene %q", want)
		}
	}
}

func TestNoFishingTabWithoutFishing(t *testing.T) {
	out := buildSite(t)
	if fileExists(filepath.Join(out, "pesca")) {
		t.Error("sin datos de pesca no debe haber pestaña")
	}
}

func TestEnchantNotes(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/a-vanilla.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("avanilla", "Vanilla data", "[1.20.1,1.21)"),
		"data/minecraft/loot_tables/gameplay/fishing/treasure.json": `{"type":"minecraft:fishing","pools":[{"rolls":1,"entries":[
			{"type":"minecraft:item","name":"minecraft:book","functions":[{"function":"minecraft:enchant_with_levels","levels":30.0,"treasure":true}]}]}]}`,
	})
	in.Jar("mods/dctweaks.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("deceasedcraft", "DeceasedCraft", "[1.20.1,1.21)"),
		// DCTweaks overrides the vanilla fishing treasure without treasure enchantments.
		"data/minecraft/loot_tables/gameplay/fishing/treasure.json": `{"type":"minecraft:fishing","pools":[{"rolls":1,"entries":[
			{"type":"minecraft:item","name":"minecraft:book","functions":[{"function":"minecraft:enchant_with_levels","levels":15.0,"treasure":false}]},
			{"type":"minecraft:item","name":"minecraft:bow","functions":[{"function":"minecraft:enchant_with_levels","levels":{"type":"minecraft:uniform","min":20,"max":39},"treasure":true}]},
			{"type":"minecraft:item","name":"minecraft:fishing_rod","functions":[{"function":"minecraft:enchant_randomly","enchantments":["minecraft:mending","minecraft:lure"]}]},
			{"type":"minecraft:item","name":"minecraft:enchanted_book","functions":[{"function":"minecraft:set_nbt","tag":"{StoredEnchantments:[{id:\"minecraft:mending\",lvl:1s}]}"}]},
			{"type":"minecraft:item","name":"minecraft:potion","functions":[{"function":"minecraft:set_nbt","tag":"{Potion:\"minecraft:strong_healing\"}"}]}]}]}`,
		"assets/minecraft/lang/es_es.json": `{"enchantment.minecraft.mending":"Reparación","enchantment.minecraft.frost_walker":"Paso helado","enchantment.minecraft.lure":"Atracción",
			"item.minecraft.enchanted_book":"Libro encantado","item.minecraft.fishing_rod":"Caña de pescar","item.minecraft.potion.effect.healing":"Poción de curación","enchantment.level.1":"I"}`,
		"assets/minecraft/lang/en_us.json": `{"enchantment.minecraft.mending":"Mending","item.minecraft.enchanted_book":"Enchanted Book"}`,
	})
	res, err := plugins.Analyzer().
		Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	out := filepath.Join(t.TempDir(), "site")
	if _, err := site.Build(res, site.Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "tablas/minecraft/gameplay/fishing/treasure/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	for _, want := range []string{
		"Encantado (nivel 15, sin encantamientos de tesoro: no da Reparación, Paso helado ni maldiciones)",
		"Encantado (nivel 20–39, puede dar encantamientos de tesoro como Reparación)",
		// Each useful variant is its own item.
		"Caña de pescar: Reparación", "Caña de pescar: Atracción",
		"Libro encantado: Reparación", "Nivel I", "Poción de curación II",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("la tabla de tesoro no contiene %q", want)
		}
	}
	book, err := os.ReadFile(filepath.Join(out, "objetos/minecraft/enchanted_book/enchantment-minecraft_mending/index.html"))
	if err != nil || !strings.Contains(string(book), "Libro encantado: Reparación") {
		t.Errorf("falta la página de la variante: %v", err)
	}
	index, _ := os.ReadFile(filepath.Join(out, "assets/search-index.js"))
	if !strings.Contains(string(index), "Enchanted Book: Mending") {
		t.Error("el índice debe permitir buscar por el nombre en inglés")
	}
	// Overriding a table is a change: listed in "Cambios de mods".
	changes, err := os.ReadFile(filepath.Join(out, "cambios/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changes), "DeceasedCraft") {
		t.Error("la página de cambios debería nombrar al mod que reemplaza la tabla")
	}
}

func TestTradesTab(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/customnpcs.jar", testkit.Files{
		"META-INF/mods.toml": testkit.ModsToml("customnpcs", "CustomNPCs", "[1.20.1,1.21)"),
		"data/minecraft/loot_tables/gameplay/piglin_bartering.json": `{"type":"minecraft:barter","pools":[{"rolls":1,"entries":[{"type":"minecraft:item","name":"minecraft:ender_pearl","weight":1},{"type":"minecraft:item","name":"minecraft:gravel","weight":3}]}]}`,
		"assets/mymod/lang/es_es.json":                              `{"npc.mymod.bandit.name":"Bandido","entity.minecraft.villager.farmer":"Granjero"}`,
	})
	in.Dir("customnpcs/clones/1", testkit.Files{
		"Bandit.json": "{\n  \"Name\": \"npc.mymod.bandit.name\",\n  \"Role\": 0,\n  \"Health\": 40.0f,\n  \"ReturnToStart\": 0b,\n" +
			"  \"NpcInv\": [ { \"Slot\": 0b, \"id\": \"mymod:money\", \"Count\": 5b }, { \"Slot\": 1b, \"id\": \"minecraft:bread\", \"Count\": 2b } ],\n" +
			"  \"DropChance\": [ { \"Integer\": 100.0f, \"Slot\": 0 }, { \"Integer\": 25.0f, \"Slot\": 1 } ],\n  \"KilledTime\": 0L\n}\n",
		"Shopkeeper.json": `{"Name": "Tendero", "Role": 0, "ForgeData": {"somemod.shop.json": "{\"type\":\"npc_shop\",\"title\":\"Tienda del tendero\",\"currencyItem\":\"mymod:money\",\"buyEnabled\":true,\"sellEnabled\":false,\"items\":[{\"item\":\"minecraft:apple\",\"count\":3,\"price\":7,\"stock\":-1,\"currencyType\":\"inherit\"},{\"item\":\"minecraft:diamond\",\"count\":1,\"price\":2,\"currencyItem\":\"minecraft:emerald\",\"stock\":4}]}"}}`,
		"Merchant.json": "{\n  \"Name\": \"Mercader\",\n  \"Role\": 1,\n" +
			"  \"TraderSold\": [ { \"Slot\": 0b, \"id\": \"minecraft:diamond\", \"Count\": 1b } ],\n" +
			"  \"TraderCurrency\": [ { \"Slot\": 0b, \"id\": \"mymod:money\", \"Count\": 20b }, { \"Slot\": 18b, \"id\": \"minecraft:bread\", \"Count\": 2b } ]\n}\n",
	})
	res, err := plugins.Analyzer().Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	out := filepath.Join(t.TempDir(), "site")
	if _, err := site.Build(res, site.Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	index := read("tradeos/index.html")
	for _, want := range []string{"Aldeanos y comerciante errante", "Trueque con piglins", "NPCs (CustomNPCs)", "Granjero"} {
		if !strings.Contains(index, want) {
			t.Errorf("la portada de tradeos no contiene %q", want)
		}
	}
	farmer := read("tradeos/aldeanos/minecraft/farmer/index.html")
	for _, want := range []string{"Novato", "Se eligen 2 de estas 5 ofertas", "20 × ", "40 %", "Composter"} {
		if !strings.Contains(farmer, want) {
			t.Errorf("el granjero no contiene %q", want)
		}
	}
	bread := read("objetos/minecraft/bread/index.html")
	for _, want := range []string{"Comerciando", "Granjero", "Lo sueltan NPCs", "Bandido", "25 %"} {
		if !strings.Contains(bread, want) {
			t.Errorf("el pan no contiene %q", want)
		}
	}
	merchant := read("tradeos/npcs/npc/merchant/index.html")
	for _, want := range []string{"20 × ", "Money", "2 × ", "Bread", "1 × ", "Diamond"} {
		if !strings.Contains(merchant, want) {
			t.Errorf("el mercader no contiene %q", want)
		}
	}
	shop := read("tradeos/npcs/npc/shopkeeper/index.html")
	for _, want := range []string{"Tienda: Tienda del tendero", "7 × ", "Apple", "existencias: 4"} {
		if !strings.Contains(shop, want) {
			t.Errorf("la tienda no contiene %q", want)
		}
	}
	npc := read("tradeos/npcs/npc/bandit/index.html")
	if !strings.Contains(npc, "Lo que suelta al morir") || !strings.Contains(npc, "100 %") {
		t.Error("el bandido debería listar sus drops")
	}
}

func TestCreaturePages(t *testing.T) {
	in := testkit.NewInstance(t)
	in.Jar("mods/a-vanilla.jar", testkit.Files{
		"META-INF/mods.toml":                                         testkit.ModsToml("avanilla", "Vanilla data", "[1.20.1,1.21)"),
		"data/minecraft/loot_tables/entities/zombie.json":            `{"type":"minecraft:entity","pools":[{"rolls":1,"entries":[{"type":"minecraft:item","name":"minecraft:rotten_flesh"}]}]}`,
		"data/minecraft/worldgen/biome/plains.json":                  `{"spawners":{"monster":[{"type":"minecraft:zombie","weight":95,"minCount":4,"maxCount":4},{"type":"minecraft:spider","weight":5,"minCount":1,"maxCount":1}]}}`,
		"data/minecraft/worldgen/biome/desert.json":                  `{"spawners":{"monster":[{"type":"minecraft:zombie","weight":10,"minCount":1,"maxCount":2}]}}`,
		"data/minecraft/forge/biome_modifier/no_desert_zombies.json": `{"type":"forge:remove_spawns","biomes":"minecraft:desert","entity_types":"minecraft:zombie"}`,
		"assets/minecraft/lang/es_es.json":                           `{"entity.minecraft.zombie":"Zombi","biome.minecraft.plains":"Llanura","item.minecraft.rotten_flesh":"Carne podrida"}`,
	})
	res, err := plugins.Analyzer().Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	out := filepath.Join(t.TempDir(), "site")
	if _, err := site.Build(res, site.Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		data, err := os.ReadFile(filepath.Join(out, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	zombie := read("criaturas/minecraft/zombie/index.html")
	for _, want := range []string{"Zombi", "Llanura", "95 %", "hostil", "Carne podrida", "aparece en 1 bioma"} {
		if !strings.Contains(zombie, want) {
			t.Errorf("la ficha del zombi no contiene %q", want)
		}
	}
	if strings.Contains(zombie, "Desert") {
		t.Error("remove_spawns debe quitar el desierto")
	}
	flesh := read("objetos/minecraft/rotten_flesh/index.html")
	for _, want := range []string{"La forma más probable", "Lo sueltan criaturas", "Zombi", "aparece en 1 bioma"} {
		if !strings.Contains(flesh, want) {
			t.Errorf("la carne podrida no contiene %q", want)
		}
	}
}
