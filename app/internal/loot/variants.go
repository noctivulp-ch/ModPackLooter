package loot

import (
	"encoding/json"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/variants"
)

// Registry is optionally implemented by a Source so that random
// enchantments and goat horns can be expanded into one variant each.
type Registry interface {
	// Enchantments lists every enchantment of the pack.
	Enchantments() []domain.ResourceID
	// InstrumentTag lists the instruments of a tag.
	InstrumentTag(id domain.ResourceID) []domain.ResourceID
}

var (
	book          = domain.ResourceID{Namespace: "minecraft", Path: "book"}
	enchantedBook = domain.ResourceID{Namespace: "minecraft", Path: "enchanted_book"}
	emptyMap      = domain.ResourceID{Namespace: "minecraft", Path: "map"}
	filledMap     = domain.ResourceID{Namespace: "minecraft", Path: "filled_map"}
)

// Vanilla goat horn tags, used when the pack's tags are not readable.
var vanillaHorns = map[string][]string{
	"minecraft:regular_goat_horns":   {"ponder", "sing", "seek", "feel"},
	"minecraft:screaming_goat_horns": {"admire", "call", "yearn", "dream"},
	"minecraft:goat_horns":           {"ponder", "sing", "seek", "feel", "admire", "call", "yearn", "dream"},
}

// share is a variant and the fraction of the drops that are it.
type share struct {
	v domain.Variant
	p float64
}

// variantsOf applies the functions that turn an item into variants. It
// returns the final item (a book enchanted becomes an enchanted book), the
// variants with their share, and which notes the variants make redundant.
func (r *Resolver) variantsOf(item domain.ResourceID, fns []rawFunction) (domain.ResourceID, []share, map[string]bool) {
	out := []share{{p: 1}}
	used := map[string]bool{}
	single := func(v domain.Variant, fn string) {
		out = []share{{v: v, p: 1}}
		used[fn] = true
	}
	split := func(vs []domain.Variant, fn string) {
		if len(vs) == 0 {
			return
		}
		out = out[:0]
		for _, v := range vs {
			out = append(out, share{v: v, p: 1 / float64(len(vs))})
		}
		used[fn] = true
	}
	reg, _ := r.src.(Registry)
	for _, f := range fns {
		name := f.name()
		switch name {
		case "set_nbt":
			var tag string
			if json.Unmarshal(f["tag"], &tag) != nil {
				tag = string(f["tag"])
			}
			if v, ok := variants.FromNBT(tag); ok {
				single(v, name)
				if item == book && v.Kind == domain.VariantEnchantment {
					item = enchantedBook
				}
			}
		case "set_components":
			var c map[string]json.RawMessage
			if json.Unmarshal(f["components"], &c) == nil {
				if v, ok := variants.FromComponents(c); ok {
					single(v, name)
				}
			}
		case "set_potion":
			var id string
			if json.Unmarshal(f["id"], &id) == nil && id != "" {
				single(domain.Variant{Kind: domain.VariantPotion, Value: full(id)}, name)
			}
		case "set_enchantments":
			var m map[string]json.RawMessage
			_ = json.Unmarshal(f["enchantments"], &m)
			var ids []string
			level := 0
			for id, lv := range m {
				ids = append(ids, id)
				if n := parseNumber(lv, 0); !n.Approximate && n.Min == n.Max {
					level = int(n.Min)
				}
			}
			if v, ok := variants.Enchantments(ids, level); ok {
				single(v, name)
				if item == book {
					item = enchantedBook
				}
			}
		case "enchant_randomly":
			list := stringList(f["enchantments"])
			if len(list) == 0 {
				list = stringList(f["options"])
			}
			isBook := item == book || item == enchantedBook
			if isBook {
				item = enchantedBook
			}
			if len(list) == 1 && strings.HasPrefix(list[0], "#") {
				list = nil // a tag of enchantments: not expanded yet
			}
			if len(list) == 0 && isBook && reg != nil {
				for _, id := range reg.Enchantments() {
					list = append(list, id.String())
				}
			}
			var vs []domain.Variant
			for _, id := range list {
				vs = append(vs, domain.Variant{Kind: domain.VariantEnchantment, Value: full(id)})
			}
			split(vs, name)
		case "enchant_with_levels":
			if item == book {
				item = enchantedBook
			}
		case "set_stew_effect":
			var effects []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			_ = json.Unmarshal(f["effects"], &effects)
			var vs []domain.Variant
			for _, e := range effects {
				id := e.Type
				if id == "" {
					id = e.ID
				}
				if id != "" {
					vs = append(vs, domain.Variant{Kind: domain.VariantEffect, Value: full(id)})
				}
			}
			split(vs, name)
		case "set_instrument":
			var opt string
			_ = json.Unmarshal(f["options"], &opt)
			tag := full(strings.TrimPrefix(opt, "#"))
			var vs []domain.Variant
			if reg != nil {
				if id, err := domain.ParseResourceID(tag); err == nil {
					for _, x := range reg.InstrumentTag(id) {
						vs = append(vs, domain.Variant{Kind: domain.VariantInstrument, Value: x.String()})
					}
				}
			}
			if len(vs) == 0 {
				for _, h := range vanillaHorns[tag] {
					vs = append(vs, domain.Variant{Kind: domain.VariantInstrument, Value: "minecraft:" + h + "_goat_horn"})
				}
			}
			split(vs, name)
		case "exploration_map":
			dest := "minecraft:on_treasure_maps"
			var d string
			if json.Unmarshal(f["destination"], &d) == nil && d != "" {
				dest = full(d)
			}
			if item == emptyMap {
				item = filledMap
			}
			single(domain.Variant{Kind: domain.VariantMap, Value: dest}, name)
		}
	}
	return item, out, used
}

// stringList reads a string or a list of strings.
func stringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return []string{s}
	}
	return nil
}

func full(s string) string {
	if strings.HasPrefix(s, "#") {
		return "#" + full(s[1:])
	}
	if !strings.Contains(s, ":") {
		return "minecraft:" + s
	}
	return s
}
