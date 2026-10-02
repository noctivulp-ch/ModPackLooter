package analysis_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/modpack"
	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
	"github.com/EnierAragon/ModPackLooter/app/internal/testkit"
	"github.com/EnierAragon/ModPackLooter/app/internal/world"
)

func structure(biomes string) string {
	return `{"type":"minecraft:jigsaw","start_pool":"minecraft:empty","biomes":` + biomes + `}`
}

func set(structures ...string) string {
	var parts []string
	for _, s := range structures {
		parts = append(parts, `{"structure":"`+s+`","weight":1}`)
	}
	return `{"structures":[` + strings.Join(parts, ",") + `],"placement":{"type":"minecraft:random_spread","spacing":32,"separation":8,"salt":1}}`
}

func disablerInstance(t *testing.T) *testkit.Instance {
	in := testkit.NewInstance(t)
	files := testkit.Files{"META-INF/mods.toml": testkit.ModsToml("towers", "Towers", "[1.20.1,1.21)")}
	for _, name := range []string{"kept", "unset", "ash", "struct_off", "ns_set_off", "mentioned", "scripted", "nowhere"} {
		files["data/towers/worldgen/structure/"+name+".json"] = structure(`["minecraft:plains"]`)
	}
	files["data/towers/worldgen/structure/ash.json"] = structure(`["towers:ash_fields"]`)
	files["data/towers/worldgen/structure/nowhere.json"] = structure(`"#towers:none"`)
	files["data/towers/worldgen/structure_set/main.json"] = set("towers:kept", "towers:ash", "towers:struct_off", "towers:mentioned", "towers:scripted", "towers:nowhere")
	files["data/towers/worldgen/structure_set/other.json"] = set("towers:ns_set_off")
	in.Jar("mods/towers.jar", files)
	for _, mod := range []string{"biome_replacer", "structurify", "incontrol", "kubejs", "lostcities"} {
		in.Jar("mods/"+mod+".jar", testkit.Files{"META-INF/mods.toml": testkit.ModsToml(mod, mod, "[1.20.1,1.21)")})
	}
	in.Dir("config", testkit.Files{
		"biome_replacer.properties": "! comment\ntowers:ash_fields > null\n",
		"structurify.json":          `{"general":{"disable_all_structures":false},"structures":[{"name":"towers:struct_off","is_disabled":true},{"name":"towers:kept","is_disabled":false}],"structure_sets":[{"name":"towers:other","is_disabled":true}]}`,
		"incontrol/spawn.json":      `[{"mob":"minecraft:creeper","result":"deny"},{"mob":["minecraft:witch"],"dimension":"minecraft:overworld","result":"deny"},{"mob":"minecraft:pig","result":"allow"}]`,
		"somemod-common.toml":       "[worldgen]\n# structures that will not spawn\ndisabledStructures = [\n  \"towers:mentioned\"\n]\nfavourite = \"towers:kept\"\n",
		"lootr-common.toml":         "loot_structure_blacklist = [\"towers:kept\"]\n",
	})
	in.Dir("kubejs/server_scripts", testkit.Files{"world.js": "// quitar la torre\nWorldgenEvents.remove(event => {\n  event.removeStructure('towers:scripted')\n})\n"})
	in.Dir("defaultconfigs", testkit.Files{"lostcities-server.toml": "[profiles]\nselectedProfile = \"\"\n"})
	return in
}

func status(t *testing.T, res interface {
	Status(domain.Target) domain.Status
}, kind domain.TargetKind, id string) domain.Status {
	t.Helper()
	return res.Status(domain.Target{Kind: kind, ID: domain.MustParseResourceID(id)})
}

func TestDisablersWithoutWorld(t *testing.T) {
	in := disablerInstance(t)
	res, err := analyzer().Run(context.Background(), modpack.Options{Path: in.Root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()

	cases := []struct {
		kind domain.TargetKind
		id   string
		want domain.Certainty
		why  string
	}{
		{domain.TargetStructure, "towers:kept", 0, ""},
		{domain.TargetStructure, "towers:unset", domain.Certainly, "structure_set"},
		{domain.TargetBiome, "towers:ash_fields", domain.Certainly, "Biome Replacer"},
		{domain.TargetStructure, "towers:ash", domain.Certainly, "todos sus biomas"},
		{domain.TargetStructure, "towers:struct_off", domain.Certainly, "Structurify"},
		{domain.TargetStructure, "towers:ns_set_off", domain.Certainly, "towers:other"},
		{domain.TargetStructure, "towers:nowhere", domain.Certainly, "vacía"},
		{domain.TargetStructure, "towers:mentioned", domain.Possibly, "somemod-common.toml"},
		{domain.TargetStructure, "towers:scripted", domain.Possibly, "kubejs/server_scripts/world.js"},
		{domain.TargetEntity, "minecraft:creeper", domain.Certainly, "sin condiciones"},
		{domain.TargetEntity, "minecraft:witch", domain.Possibly, "dimension"},
		{domain.TargetEntity, "minecraft:pig", 0, ""},
		{domain.TargetStructure, "lostcities:city", domain.Possibly, "mundos nuevos"},
	}
	for _, c := range cases {
		st := status(t, res, c.kind, c.id)
		if st.Certainty != c.want || !strings.Contains(strings.Join(st.Reasons, " | "), c.why) {
			t.Errorf("%s %s: %v %q, se esperaba %v con %q", c.kind, c.id, st.Certainty, st.Reasons, c.want, c.why)
		}
	}
}

func TestDisablersWithModelWorld(t *testing.T) {
	in := disablerInstance(t)
	data := world.Sample()
	dir := filepath.Join(in.Root, "saves", "Prueba")
	if err := os.MkdirAll(filepath.Join(dir, "serverconfig"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "level.dat"), nbt.Encode(nbt.Compound{"Data": data}), 0o644); err != nil {
		t.Fatal(err)
	}
	// The world chose a Lost Cities profile, unlike the pack defaults.
	if err := os.WriteFile(filepath.Join(dir, "serverconfig", "lostcities-server.toml"), []byte("[profiles]\nselectedProfile = \"onlycities\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in.Jar("mods/towers2.jar", testkit.Files{
		"META-INF/mods.toml":                            testkit.ModsToml("towers2", "Towers 2", "[1.20.1,1.21)"),
		"data/towers2/worldgen/structure/sandy.json":    structure(`["minecraft:desert"]`),
		"data/towers2/worldgen/structure/snowy.json":    structure(`["minecraft:snowy_plains"]`),
		"data/towers2/worldgen/structure_set/both.json": set("towers2:sandy", "towers2:snowy"),
	})
	res, err := analyzer().Run(context.Background(), modpack.Options{Path: in.Root, World: "Prueba"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	if res.Modpack.World == nil || res.Modpack.World.Name != "Prueba" {
		t.Fatalf("mundo = %+v", res.Modpack.World)
	}
	if st := status(t, res, domain.TargetStructure, "towers2:sandy"); st.Certainty != 0 {
		t.Errorf("el desierto existe en el mundo: %+v", st)
	}
	// The sample world has a dimension without a biome list (the nether
	// preset), so absence is only "possibly".
	if st := status(t, res, domain.TargetStructure, "towers2:snowy"); st.Certainty != domain.Possibly {
		t.Errorf("snowy_plains no está en el mundo: %+v", st)
	}
	if st := status(t, res, domain.TargetStructure, "lostcities:city"); st.Certainty != 0 {
		t.Errorf("el mundo tiene perfil de Lost Cities: %+v", st)
	}
}
