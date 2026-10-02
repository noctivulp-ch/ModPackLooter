package worldgen

import (
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
	"github.com/EnierAragon/ModPackLooter/app/internal/testkit"
)

func template(blocks nbt.List, entities nbt.List, palette ...string) string {
	pal := nbt.List{}
	for _, p := range palette {
		pal = append(pal, nbt.Compound{"Name": p})
	}
	return string(nbt.Encode(nbt.Compound{"palette": pal, "blocks": blocks, "entities": entities}))
}

func block(state int32, data nbt.Compound) nbt.Compound {
	return nbt.Compound{"pos": nbt.List{int32(0), int32(0), int32(0)}, "state": state, "nbt": data}
}

func index(t *testing.T, files testkit.Files) *resources.Index {
	t.Helper()
	in := testkit.NewInstance(t)
	p, err := resources.OpenZipPack(in.Jar("mods/towers.jar", files), resources.KindMod)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return resources.NewIndex(resources.LayoutFor(mcversion.MustParse("1.20.1")), []resources.Pack{p})
}

func TestStructureLootFollowsJigsawPoolsAndProcessors(t *testing.T) {
	ix := index(t, testkit.Files{
		"data/towers/worldgen/structure/tower.json":         `{"type":"minecraft:jigsaw","start_pool":"towers:base","biomes":"#towers:has_tower"}`,
		"data/towers/tags/worldgen/biome/has_tower.json":    `{"values":["minecraft:plains","#minecraft:is_forest",{"id":"other:biome","required":false}]}`,
		"data/minecraft/tags/worldgen/biome/is_forest.json": `{"values":["minecraft:forest"]}`,
		"data/towers/worldgen/template_pool/base.json": `{"fallback":"minecraft:empty","elements":[
			{"weight":1,"element":{"element_type":"minecraft:single_pool_element","location":"towers:base","processors":"towers:loot","projection":"rigid"}}]}`,
		"data/towers/worldgen/template_pool/top.json": `{"elements":[
			{"weight":1,"element":{"element_type":"minecraft:list_pool_element","elements":[
				{"element_type":"minecraft:single_pool_element","location":"towers:top","processors":{"processors":[]}}]}}]}`,
		"data/towers/worldgen/processor_list/loot.json": `{"processors":[{"processor_type":"minecraft:capped","limit":2,"delegate":{
			"processor_type":"minecraft:rule","rules":[{"output_state":{"Name":"minecraft:suspicious_sand"},
			"block_entity_modifier":{"type":"minecraft:append_loot","loot_table":"towers:archaeology/base"}}]}}]}`,
		"data/towers/structures/base.nbt": template(nbt.List{
			block(0, nbt.Compound{"id": "minecraft:jigsaw", "pool": "towers:top"}),
			block(1, nbt.Compound{"id": "minecraft:structure_block", "mode": "DATA", "metadata": "ChestWest"}),
		}, nbt.List{}, "minecraft:jigsaw", "minecraft:structure_block"),
		"data/towers/structures/top.nbt": template(nbt.List{
			block(0, nbt.Compound{"id": "minecraft:chest", "LootTable": "towers:chests/top"}),
		}, nbt.List{nbt.Compound{"nbt": nbt.Compound{"id": "minecraft:chest_minecart", "LootTable": "towers:chests/cart"}}}, "minecraft:chest"),
	})
	w := New(ix, &domain.Diagnostics{})
	structures := w.Structures()
	if len(structures) != 1 {
		t.Fatalf("estructuras = %+v", structures)
	}
	s := structures[0]
	var biomes []string
	for _, b := range s.Biomes {
		biomes = append(biomes, b.String())
	}
	if want := "[minecraft:forest minecraft:plains other:biome]"; fmtList(biomes) != want {
		t.Errorf("biomas = %v, se esperaba %s", biomes, want)
	}

	found, reached := w.StructureLoot(s)
	if len(reached) != 2 {
		t.Errorf("plantillas alcanzadas = %v", reached)
	}
	got := map[string]Found{}
	for _, f := range found {
		got[f.LootTable.String()] = f
	}
	if f := got["towers:chests/top"]; f.Via != "nbt" || f.Block != "minecraft:chest" || f.Template.String() != "towers:top" {
		t.Errorf("cofre de la plantilla = %+v", f)
	}
	if f := got["towers:chests/cart"]; f.Block != "minecraft:chest_minecart" {
		t.Errorf("vagoneta = %+v", f)
	}
	if f := got["towers:archaeology/base"]; f.Via != "processor" || f.Block != "minecraft:suspicious_sand" || f.Detail != "towers:loot" {
		t.Errorf("processor append_loot = %+v", f)
	}
	base, _ := w.Template(domain.MustParseResourceID("towers:base"))
	if len(base.DataMarkers) != 1 || base.DataMarkers[0] != "ChestWest" {
		t.Errorf("data markers = %v", base.DataMarkers)
	}
}

func TestTagReplaceDropsEarlierValues(t *testing.T) {
	in := testkit.NewInstance(t)
	a, _ := resources.OpenZipPack(in.Jar("mods/a.jar", testkit.Files{"data/x/tags/worldgen/biome/t.json": `{"values":["x:a"]}`}), resources.KindMod)
	b, _ := resources.OpenZipPack(in.Jar("mods/b.jar", testkit.Files{"data/x/tags/worldgen/biome/t.json": `{"replace":true,"values":["x:b"]}`}), resources.KindMod)
	defer a.Close()
	defer b.Close()
	ix := resources.NewIndex(resources.LayoutFor(mcversion.MustParse("1.20.1")), []resources.Pack{a, b})
	got := NewTags(ix, resources.TypeBiomeTag, nil).Resolve(domain.MustParseResourceID("x:t"))
	if len(got) != 1 || got[0].String() != "x:b" {
		t.Errorf("tag con replace = %v", got)
	}
}

func fmtList(s []string) string {
	out := "["
	for i, x := range s {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out + "]"
}
