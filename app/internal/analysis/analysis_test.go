package analysis_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/modpack"
	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
	"github.com/EnierAragon/ModPackLooter/app/internal/plugins"
	"github.com/EnierAragon/ModPackLooter/app/internal/testkit"
)

const chestTable = `{"type":"minecraft:chest","pools":[{"rolls":1,"entries":[{"type":"minecraft:item","name":"minecraft:diamond"}]}]}`

func chestTemplate(table string) string {
	return string(nbt.Encode(nbt.Compound{
		"palette":  nbt.List{nbt.Compound{"Name": "minecraft:chest"}},
		"blocks":   nbt.List{nbt.Compound{"pos": nbt.List{int32(0), int32(0), int32(0)}, "state": int32(0), "nbt": nbt.Compound{"id": "minecraft:chest", "LootTable": table}}},
		"entities": nbt.List{},
	}))
}

func analyzer() analysis.Analyzer {
	return analysis.Analyzer{Discoverers: plugins.Discoverers(), Enrichers: plugins.Enrichers()}
}

func find(sources []domain.LootSource, table, owner string) (domain.LootSource, bool) {
	for _, s := range sources {
		ownerMatches := s.Owner.ID.String() == owner || (owner == "" && s.Owner.Kind == domain.OwnerNone)
		if s.LootTable.String() == table && ownerMatches {
			return s, true
		}
	}
	return domain.LootSource{}, false
}

func hasNote(s domain.LootSource, key string) bool {
	for _, n := range s.Notes {
		if n.Key == key {
			return true
		}
	}
	return false
}

func TestAnalyzeSyntheticModpack(t *testing.T) {
	in := testkit.NewInstance(t)
	vanilla := in.Jar("vanilla.jar", testkit.Files{
		"data/minecraft/loot_tables/chests/desert_pyramid.json":                chestTable,
		"data/minecraft/loot_tables/chests/simple_dungeon.json":                chestTable,
		"data/minecraft/loot_tables/entities/zombie.json":                      `{"type":"minecraft:entity","pools":[]}`,
		"data/minecraft/worldgen/structure/desert_pyramid.json":                `{"type":"minecraft:desert_pyramid","biomes":"#minecraft:has_structure/desert_pyramid"}`,
		"data/minecraft/tags/worldgen/biome/has_structure/desert_pyramid.json": `{"values":["minecraft:desert"]}`,
		"data/minecraft/tags/worldgen/biome/is_overworld.json":                 `{"values":["minecraft:desert","minecraft:plains"]}`,
	})
	in.Jar("mods/towers.jar", testkit.Files{
		"META-INF/mods.toml":                                  testkit.ModsToml("towers", "Towers", "[1.20.1,1.21)"),
		"data/towers/worldgen/structure/tower.json":           `{"type":"minecraft:jigsaw","start_pool":"towers:start","biomes":["minecraft:plains"]}`,
		"data/towers/worldgen/template_pool/start.json":       `{"elements":[{"weight":1,"element":{"element_type":"minecraft:single_pool_element","location":"towers:base","processors":"minecraft:empty"}}]}`,
		"data/towers/structures/base.nbt":                     chestTemplate("towers:chests/tower_base"),
		"data/towers/loot_tables/chests/tower_base.json":      chestTable,
		"data/towers/worldgen/structure/ruined_keep.json":     `{"type":"towers:custom","biomes":"minecraft:plains"}`,
		"data/towers/loot_tables/chests/ruined_keep_top.json": chestTable,
		"data/towers/loot_tables/chests/orphan.json":          chestTable,
	})
	in.Jar("mods/lostcities.jar", testkit.Files{
		"META-INF/mods.toml":                              testkit.ModsToml("lostcities", "Lost Cities", "[1.20.1,1.21)"),
		"data/lostcities/lostcities/palettes/common.json": `{"palette":[{"char":"C","block":"minecraft:chest[facing=north]","loot":"chestloot"}]}`,
		"data/lostcities/lostcities/conditions/chestloot.json": `{"values":[
			{"factor":3,"value":"lostcities:chests/lostcitychest","range":"4,100"},
			{"factor":1,"value":"minecraft:chests/simple_dungeon"}]}`,
		"data/lostcities/loot_tables/chests/lostcitychest.json": chestTable,
	})
	in.Jar("mods/lootr.jar", testkit.Files{"META-INF/mods.toml": testkit.ModsToml("lootr", "Lootr", "[1.20,1.21)")})
	in.Dir("config", testkit.Files{"lootr-common.toml": "refresh_loot_tables = [\"towers:chests/tower_base\"]\nrefresh_value = 12000\nloot_modid_blacklist = [\"lostcities\"]\n"})

	res, err := analyzer().Run(context.Background(), modpack.Options{Path: in.Root, MinecraftJar: vanilla}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()

	tower, ok := find(res.Sources, "towers:chests/tower_base", "towers:tower")
	if !ok || tower.Confidence != domain.ConfidenceExact || tower.Container != "minecraft:chest" {
		t.Errorf("cofre de la torre = %+v (ok=%v)", tower, ok)
	}
	if !hasNote(tower, "lootr.per_player") || !hasNote(tower, "lootr.refresh") {
		t.Errorf("la torre debe ser por jugador y rellenarse: %+v", tower.Notes)
	}
	if !strings.Contains(tower.Notes[len(tower.Notes)-1].Text, "10 minutos") {
		t.Errorf("tiempo de refresh = %+v", tower.Notes)
	}

	pyramid, ok := find(res.Sources, "minecraft:chests/desert_pyramid", "minecraft:desert_pyramid")
	if !ok || pyramid.Confidence != domain.ConfidenceKnown {
		t.Errorf("pirámide (conocimiento vanilla) = %+v", pyramid)
	}
	if _, ok := find(res.Sources, "minecraft:chests/simple_dungeon", "minecraft:monster_room"); !ok {
		t.Error("la mazmorra debe venir del conocimiento vanilla")
	}

	city, ok := find(res.Sources, "lostcities:chests/lostcitychest", "lostcities:city")
	if !ok || city.Share != 0.75 || !strings.Contains(city.Evidence[0].Detail, "pisos 4 a 100") {
		t.Errorf("Lost Cities = %+v", city)
	}
	if !hasNote(city, "lootr.blacklisted") {
		t.Errorf("las tablas de lostcities están en la blacklist de Lootr: %+v", city.Notes)
	}

	keep, ok := find(res.Sources, "towers:chests/ruined_keep_top", "towers:ruined_keep")
	if !ok || keep.Confidence != domain.ConfidenceHeuristic {
		t.Errorf("heurística por nombre = %+v", keep)
	}
	orphan, ok := find(res.Sources, "towers:chests/orphan", "")
	if !ok || orphan.Confidence != domain.ConfidenceUnknown || orphan.Kind != domain.KindContainer {
		t.Errorf("genérico = %+v", orphan)
	}
	if _, ok := find(res.Sources, "minecraft:entities/zombie", ""); !ok {
		t.Error("las tablas sin referencia deben aparecer gracias al genérico")
	}

	if len(res.Tables) != 7 {
		t.Errorf("tablas resueltas = %d", len(res.Tables))
	}
	if want := "structure-templates vanilla-knowledge lostcities name-matching generic-by-path"; strings.Join(res.Plan, " ") != want {
		t.Errorf("plan = %v", res.Plan)
	}
	for _, d := range res.Diagnostics.Items() {
		if d.Level == domain.LevelError {
			t.Errorf("diagnóstico de error inesperado: %+v", d)
		}
	}
}

// TestAnalyzeRealVanilla runs the whole analysis over real vanilla data when
// MPL_MCMETA_DATA points at a misode/mcmeta data checkout.
func TestAnalyzeRealVanilla(t *testing.T) {
	data := os.Getenv("MPL_MCMETA_DATA")
	if data == "" {
		t.Skip("MPL_MCMETA_DATA no definido")
	}
	in := testkit.NewInstance(t)
	res, err := analyzer().Run(context.Background(), modpack.Options{Path: in.Root, MinecraftJar: data, MCVersion: "1.20.1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	byConf := map[domain.Confidence]int{}
	for _, s := range res.Sources {
		byConf[s.Confidence]++
	}
	t.Logf("%d fuentes, %d tablas; por confianza: %v", len(res.Sources), len(res.Tables), byConf)
	for _, want := range [][2]string{
		{"minecraft:chests/village/village_weaponsmith", "minecraft:village_plains"},
		{"minecraft:chests/ancient_city", "minecraft:ancient_city"},
		{"minecraft:archaeology/trail_ruins_rare", "minecraft:trail_ruins"},
		{"minecraft:chests/abandoned_mineshaft", "minecraft:mineshaft"},
		{"minecraft:chests/bastion_treasure", "minecraft:bastion_remnant"},
	} {
		if _, ok := find(res.Sources, want[0], want[1]); !ok {
			t.Errorf("falta %s en %s", want[0], want[1])
		}
	}
	for _, d := range res.Diagnostics.Items() {
		if d.Level >= domain.LevelWarning {
			t.Logf("diagnóstico: %+v", d)
		}
	}
}
