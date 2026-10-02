package variants

import (
	"encoding/json"
	"testing"
)

func TestFromNBT(t *testing.T) {
	cases := map[string]string{
		`{StoredEnchantments: [{id:"minecraft:mending",lvl:1s}]}`:                     "enchantment:minecraft:mending@1",
		`{StoredEnchantments:[{lvl:3s,id:"tacz_eo:firepower"}]}`:                       "enchantment:tacz_eo:firepower@3",
		`{StoredEnchantments:[{id:"a:x",lvl:1s},{id:"a:b",lvl:2s}]}`:                   "enchantment:a:b,a:x",
		`{Potion:"minecraft:healing"}`:                                                 "potion:minecraft:healing",
		`{Potion: "strong_healing"}`:                                                   "potion:minecraft:strong_healing",
		`{GunFireMode:"AUTO",GunId:"tacz:ump45",HasBulletInBarrel:0b}`:                 "nbt:GunId=tacz:ump45",
		`{AmmoId:"tacz:308"}`:                                                          "nbt:AmmoId=tacz:308",
		`{pages:['{"text":"x"}'],title:"lore.deceasedcraft.note_7.title",author:"Ana"}`: "title:lore.deceasedcraft.note_7.title",
		`{blueprint: "molds"}`:                                                         "nbt:blueprint=molds",
		`{"AttachmentId":"bf1:marksman_scope"}`:                                        "nbt:AttachmentId=bf1:marksman_scope",
		`{Enchantments:[{id:"minecraft:sharpness",lvl:5s}]}`:                           "enchantment:minecraft:sharpness@5",
	}
	for in, want := range cases {
		v, ok := FromNBT(in)
		if !ok || v.String() != want {
			t.Errorf("FromNBT(%s) = %q, %v; quiero %q", in, v.String(), ok, want)
		}
	}
	if v, ok := FromNBT(`{Damage:3}`); ok {
		t.Errorf("sin variante: %v", v)
	}
}

func TestFromComponents(t *testing.T) {
	cases := map[string]string{
		`{"minecraft:stored_enchantments":{"levels":{"minecraft:mending":1}}}`: "enchantment:minecraft:mending@1",
		`{"minecraft:potion_contents":{"potion":"minecraft:swiftness"}}`:       "potion:minecraft:swiftness",
		`{"potion_contents":"long_swiftness"}`:                                 "potion:minecraft:long_swiftness",
		`{"minecraft:custom_data":"{GunId:\"tacz:ak47\"}"}`:                    "nbt:GunId=tacz:ak47",
		`{"minecraft:instrument":"minecraft:ponder_goat_horn"}`:                "instrument:minecraft:ponder_goat_horn",
	}
	for in, want := range cases {
		var c map[string]json.RawMessage
		_ = json.Unmarshal([]byte(in), &c)
		v, ok := FromComponents(c)
		if !ok || v.String() != want {
			t.Errorf("FromComponents(%s) = %q; quiero %q", in, v.String(), want)
		}
	}
}
