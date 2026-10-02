package analysis_test

import (
	"context"
	"strings"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/modpack"
	"github.com/EnierAragon/ModPackLooter/app/internal/plugins"
	"github.com/EnierAragon/ModPackLooter/app/internal/testkit"
)

func TestChangeDetectors(t *testing.T) {
	in := testkit.NewInstance(t)
	item := func(name string) string { return `{"type":"minecraft:item","name":"` + name + `"}` }
	in.Jar("mods/base.jar", testkit.Files{
		"META-INF/mods.toml":                                    testkit.ModsToml("basemod", "Base Mod", "[1.20.1,1.21)"),
		"data/minecraft/loot_tables/entities/zombie.json":       `{"type":"minecraft:entity","pools":[{"rolls":1,"entries":[` + item("minecraft:rotten_flesh") + `,` + item("minecraft:iron_ingot") + `]}]}`,
		"data/minecraft/loot_tables/chests/simple_dungeon.json": `{"type":"minecraft:chest","pools":[{"rolls":1,"entries":[` + item("minecraft:diamond") + `]}]}`,
		"assets/minecraft/lang/en_us.json":                      `{"item.minecraft.rotten_flesh":"Rotten Flesh","item.minecraft.iron_ingot":"Iron Ingot","item.minecraft.diamond":"Diamond"}`,
	})
	in.Jar("mods/tweaks.jar", testkit.Files{
		"META-INF/mods.toml":                                   testkit.ModsToml("tweaks", "Tweaks", "[1.20.1,1.21)"),
		"data/minecraft/loot_tables/entities/zombie.json":      `{"type":"minecraft:entity","pools":[{"rolls":1,"entries":[` + item("minecraft:rotten_flesh") + `]}]}`,
		"data/forge/loot_modifiers/global_loot_modifiers.json": `{"replace":false,"entries":["tweaks:add_gem","tweaks:off","tweaks:inject"]}`,
		"data/tweaks/loot_modifiers/add_gem.json":              `{"type":"tweaks:add_item","item":"tweaks:gem","chance":0.2,"conditions":[{"condition":"forge:loot_table_id","loot_table_id":"minecraft:chests/simple_dungeon"}]}`,
		"data/tweaks/loot_modifiers/off.json":                  `{"type":"tweaks:add_item","item":"tweaks:gem","chance":0.0,"conditions":[{"condition":"forge:loot_table_id","loot_table_id":"minecraft:entities/zombie"}]}`,
		"data/tweaks/loot_modifiers/inject.json":               `{"type":"other:roll_loot_table","lootTable":"tweaks:bonus","conditions":[{"condition":"minecraft:any_of","terms":[{"condition":"forge:loot_table_id","loot_table_id":"minecraft:chests/simple_dungeon"}]}]}`,
		"data/tweaks/loot_tables/bonus.json":                   `{"type":"minecraft:chest","pools":[{"rolls":1,"entries":[` + item("tweaks:gem") + `]}]}`,
		"assets/tweaks/lang/en_us.json":                        `{"item.tweaks.gem":"Gem"}`,
	})
	in.Dir("kubejs/server_scripts", testkit.Files{
		"loot.js": `LootJS.modifiers((event) => {
    event
        .addLootTableModifier("minecraft:chests/simple_dungeon")
        .removeLoot("minecraft:diamond");
    event.addLootTypeModifier(LootType.CHEST).randomChance(0.5).addLoot("tweaks:gem");
});
MoreJSEvents.villagerTrades((event) => {
    event.addTrade("minecraft:farmer", 1, ["minecraft:emerald"], "tweaks:gem");
});
// fishing is fine
SomeMod.changeFishing("whatever");
`,
	})
	in.Dir("config", testkit.Files{
		"sheepmod-common.toml": "[nerfs]\n\t#Makes Sheep not drop Wool when killed\n\t\"Disable Wool Drops\" = true\n\t#Makes Iron Golems not drop Iron Ingots\n\t\"Disable Iron Farms\" = true\n\t\"Enable Wool Drops\" = true\n\tlootSounds = true\n",
	})

	res, err := plugins.Analyzer().Run(context.Background(), modpack.Options{Path: in.Root, MCVersion: "1.20.1", Loader: "forge"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()

	find := func(by string, pred func(c domain.Change) bool) *domain.Change {
		for i, c := range res.Changes {
			if c.By == by && pred(c) {
				return &res.Changes[i]
			}
		}
		return nil
	}
	has := func(ids []domain.ResourceID, id string) bool {
		for _, x := range ids {
			if x.String() == id {
				return true
			}
		}
		return false
	}
	zombie := domain.MustParseResourceID("minecraft:entities/zombie")
	dungeon := domain.MustParseResourceID("minecraft:chests/simple_dungeon")

	if c := find("table-overrides", func(c domain.Change) bool { return c.Target.ID == zombie }); c == nil ||
		c.Effect != domain.EffectRemove || !has(c.Items, "minecraft:iron_ingot") || c.Mod != "Tweaks" || c.Certainty != domain.Certainly {
		t.Errorf("reemplazo de la tabla del zombi: %+v", c)
	}
	if c := find("global-loot-modifiers", func(c domain.Change) bool { return c.Origin == "loot_modifier:tweaks:add_gem" }); c == nil ||
		c.Target.ID != dungeon || c.Effect != domain.EffectAdd || !has(c.Items, "tweaks:gem") || c.Chance != 0.2 {
		t.Errorf("modificador que añade la gema: %+v", c)
	}
	if c := find("global-loot-modifiers", func(c domain.Change) bool { return c.Origin == "loot_modifier:tweaks:off" }); c == nil || c.Effect != domain.EffectDisable {
		t.Errorf("modificador con probabilidad 0: %+v", c)
	}
	if c := find("global-loot-modifiers", func(c domain.Change) bool { return c.Origin == "loot_modifier:tweaks:inject" }); c == nil ||
		c.Target.ID != dungeon || c.Table.String() != "tweaks:bonus" {
		t.Errorf("modificador que inyecta una tabla: %+v", c)
	}
	if c := find("lootjs", func(c domain.Change) bool { return c.Effect == domain.EffectRemove }); c == nil || c.Target.ID != dungeon || !has(c.Items, "minecraft:diamond") {
		t.Errorf("LootJS removeLoot: %+v", c)
	}
	if c := find("lootjs", func(c domain.Change) bool { return c.Effect == domain.EffectAdd }); c == nil ||
		c.Target.Kind != domain.ChangeLootType || c.Target.ID.Path != "chest" || !strings.Contains(c.Detail, "randomChance") {
		t.Errorf("LootJS por tipo de loot: %+v", c)
	}
	if c := find("trade-scripts", func(c domain.Change) bool { return true }); c == nil ||
		c.Target.ID.String() != "minecraft:farmer" || c.Effect != domain.EffectAdd || !has(c.Items, "tweaks:gem") {
		t.Errorf("tradeo de MoreJS: %+v", c)
	}
	if c := find("script-mentions", func(c domain.Change) bool { return c.Target.Kind == domain.ChangeFishing }); c == nil || !strings.Contains(c.Detail, "11") {
		t.Errorf("el genérico de scripts debería atrapar la línea de pesca: %+v", c)
	}
	if c := find("script-mentions", func(c domain.Change) bool { return c.Target.Kind != domain.ChangeFishing }); c != nil {
		t.Errorf("el genérico no debe repetir lo que explicaron los específicos: %+v", c)
	}
	if c := find("config-keys", func(c domain.Change) bool { return strings.Contains(c.Detail, "Wool Drops = true") }); c == nil ||
		c.Target.Kind != domain.ChangeDrops || c.Effect != domain.EffectRemove || c.Certainty != domain.Possibly || !strings.Contains(c.Detail, "Makes Sheep") {
		t.Errorf("clave de config de Quark: %+v", c)
	}
	if c := find("config-keys", func(c domain.Change) bool { return strings.Contains(c.Detail, "Iron Farms") }); c == nil || c.Target.Kind != domain.ChangeDrops {
		t.Errorf("la clave sin palabra de fuente usa su comentario: %+v", c)
	}
	for _, bad := range []string{"Enable Wool Drops", "lootSounds"} {
		if c := find("config-keys", func(c domain.Change) bool { return strings.Contains(c.Detail, bad) }); c != nil {
			t.Errorf("%s no es un cambio: %+v", bad, c)
		}
	}
}
