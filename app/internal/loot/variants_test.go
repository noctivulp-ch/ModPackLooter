package loot

import (
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

type regSource struct{ memSource }

func (regSource) Enchantments() []domain.ResourceID {
	return []domain.ResourceID{domain.MustParseResourceID("minecraft:mending"), domain.MustParseResourceID("minecraft:unbreaking")}
}

func (regSource) InstrumentTag(domain.ResourceID) []domain.ResourceID { return nil }

func variantDrop(t *testing.T, tab *Table, item, variant string) Drop {
	t.Helper()
	for _, d := range tab.Drops {
		if d.Item.String() == item && d.Variant.String() == variant {
			return d
		}
	}
	t.Fatalf("%s no tiene %s [%s]: %+v", tab.ID, item, variant, tab.Drops)
	return Drop{}
}

func TestVariants(t *testing.T) {
	src := regSource{memSource{tables: map[string]string{"test:v": `{"pools": [
	  {"rolls": 1, "entries": [{"type": "item", "name": "minecraft:book", "functions": [{"function": "minecraft:enchant_randomly"}]}]},
	  {"rolls": 1, "entries": [
	    {"type": "item", "name": "minecraft:enchanted_book", "functions": [{"function": "set_nbt", "tag": "{StoredEnchantments:[{id:\"minecraft:mending\",lvl:1s}]}"}]},
	    {"type": "item", "name": "minecraft:potion", "functions": [{"function": "set_nbt", "tag": "{Potion:\"minecraft:healing\"}"}]},
	    {"type": "item", "name": "minecraft:potion", "functions": [{"function": "set_potion", "id": "minecraft:swiftness"}]},
	    {"type": "item", "name": "tacz:ammo", "functions": [{"function": "set_nbt", "tag": "{AmmoId:\"tacz:308\"}"}]}]},
	  {"rolls": 1, "entries": [{"type": "item", "name": "minecraft:goat_horn", "functions": [{"function": "set_instrument", "options": "#minecraft:regular_goat_horns"}]}]}
	]}`}}}
	tab := resolve(t, src, "test:v")
	// Random book: half mending, half unbreaking, plus the fixed mending book (1/4).
	if d := variantDrop(t, tab, "minecraft:enchanted_book", "enchantment:minecraft:unbreaking"); !near(d.Chance, 0.5) || len(d.Notes) != 0 {
		t.Errorf("irrompibilidad = %+v", d)
	}
	if d := variantDrop(t, tab, "minecraft:enchanted_book", "enchantment:minecraft:mending"); !near(d.Chance, 0.5) {
		t.Errorf("reparación al azar = %+v", d)
	}
	if d := variantDrop(t, tab, "minecraft:enchanted_book", "enchantment:minecraft:mending@1"); !near(d.Chance, 0.25) || len(d.Notes) != 0 {
		t.Errorf("reparación fija = %+v", d)
	}
	variantDrop(t, tab, "minecraft:potion", "potion:minecraft:healing")
	variantDrop(t, tab, "minecraft:potion", "potion:minecraft:swiftness")
	variantDrop(t, tab, "tacz:ammo", "nbt:AmmoId=tacz:308")
	if d := variantDrop(t, tab, "minecraft:goat_horn", "instrument:minecraft:seek_goat_horn"); !near(d.Chance, 0.25) {
		t.Errorf("cuerno = %+v", d)
	}
}
