// Package variants reads the data that makes an item a distinct, useful
// variant (an enchanted book's enchantment, a potion, a modded gun's model…)
// from NBT text or 1.21 item components. Loot functions, NPC shops and any
// other source share it, so a variant found anywhere gets the same key.
package variants

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

const idChars = `[a-z0-9_.-]+:[a-z0-9_./-]+`

var (
	// A key starts the text or follows "{", "," or a space; keys may be
	// quoted (JSON) or bare (SNBT).
	keyStart   = `(?:^|[{,\s])"?`
	reStored   = regexp.MustCompile(keyStart + `StoredEnchantments"?\s*:\s*\[([^\]]*)\]`)
	reEnchants = regexp.MustCompile(keyStart + `Enchantments"?\s*:\s*\[([^\]]*)\]`)
	reEntryID  = regexp.MustCompile(`"?id"?\s*:\s*"?(` + idChars + `)"?`)
	reEntryLvl = regexp.MustCompile(`"?lvl"?\s*:\s*(\d+)`)
	rePotion   = regexp.MustCompile(keyStart + `Potion"?\s*:\s*"((?:` + idChars + `)|[a-z0-9_./-]+)"`)
	reModID    = regexp.MustCompile(keyStart + `([A-Z][A-Za-z]*Id)"?\s*:\s*"(` + idChars + `)"`)
	reTitle    = regexp.MustCompile(keyStart + `title"?\s*:\s*"((?:[^"\\]|\\.)+)"`)
	reKnown    = regexp.MustCompile(keyStart + `(blueprint)"?\s*:\s*"([a-z0-9_]+)"`)
)

// FromNBT finds the variant described by NBT written as SNBT or JSON.
func FromNBT(text string) (domain.Variant, bool) {
	if m := reStored.FindStringSubmatch(text); m != nil {
		if v, ok := enchantList(m[1]); ok {
			return v, true
		}
	}
	if m := rePotion.FindStringSubmatch(text); m != nil {
		return domain.Variant{Kind: domain.VariantPotion, Value: fullID(m[1])}, true
	}
	if m := reModID.FindStringSubmatch(text); m != nil {
		return domain.Variant{Kind: domain.VariantNBT, Key: m[1], Value: m[2]}, true
	}
	if m := reTitle.FindStringSubmatch(text); m != nil {
		return domain.Variant{Kind: domain.VariantTitle, Value: strings.ReplaceAll(m[1], `\"`, `"`)}, true
	}
	if m := reKnown.FindStringSubmatch(text); m != nil {
		return domain.Variant{Kind: domain.VariantNBT, Key: m[1], Value: m[2]}, true
	}
	if m := reEnchants.FindStringSubmatch(text); m != nil {
		if v, ok := enchantList(m[1]); ok {
			return v, true
		}
	}
	return domain.Variant{}, false
}

// enchantList reads "{id:"x",lvl:1s},{…}".
func enchantList(list string) (domain.Variant, bool) {
	var ids []string
	level := 0
	for _, entry := range strings.Split(list, "}") {
		m := reEntryID.FindStringSubmatch(entry)
		if m == nil {
			continue
		}
		ids = append(ids, m[1])
		if l := reEntryLvl.FindStringSubmatch(entry); l != nil {
			level, _ = strconv.Atoi(l[1])
		}
	}
	return Enchantments(ids, level)
}

// Enchantments builds an enchantment variant; level only counts for a
// single enchantment.
func Enchantments(ids []string, level int) (domain.Variant, bool) {
	if len(ids) == 0 {
		return domain.Variant{}, false
	}
	if len(ids) > 1 {
		level = 0
		ids = append([]string(nil), ids...)
		sort.Strings(ids)
	}
	for i := range ids {
		ids[i] = fullID(ids[i])
	}
	return domain.Variant{Kind: domain.VariantEnchantment, Value: strings.Join(ids, ","), Level: level}, true
}

// FromComponents finds the variant in 1.20.5+ item components.
func FromComponents(c map[string]json.RawMessage) (domain.Variant, bool) {
	get := func(k string) json.RawMessage {
		if v, ok := c["minecraft:"+k]; ok {
			return v
		}
		return c[k]
	}
	if raw := get("stored_enchantments"); raw != nil {
		var wrapped struct {
			Levels map[string]int `json:"levels"`
		}
		levels := map[string]int{}
		if json.Unmarshal(raw, &wrapped) == nil && len(wrapped.Levels) > 0 {
			levels = wrapped.Levels
		} else {
			_ = json.Unmarshal(raw, &levels)
		}
		var ids []string
		level := 0
		for id, l := range levels {
			ids = append(ids, id)
			level = l
		}
		if v, ok := Enchantments(ids, level); ok {
			return v, true
		}
	}
	if raw := get("potion_contents"); raw != nil {
		var id string
		if json.Unmarshal(raw, &id) != nil {
			var obj struct {
				Potion string `json:"potion"`
			}
			_ = json.Unmarshal(raw, &obj)
			id = obj.Potion
		}
		if id != "" {
			return domain.Variant{Kind: domain.VariantPotion, Value: fullID(id)}, true
		}
	}
	if raw := get("instrument"); raw != nil {
		var id string
		if json.Unmarshal(raw, &id) == nil && id != "" {
			return domain.Variant{Kind: domain.VariantInstrument, Value: fullID(id)}, true
		}
	}
	if raw := get("custom_data"); raw != nil {
		var text string
		if json.Unmarshal(raw, &text) != nil {
			text = string(raw)
		}
		return FromNBT(text)
	}
	return domain.Variant{}, false
}

func fullID(s string) string {
	s = strings.TrimPrefix(s, "#")
	if !strings.Contains(s, ":") {
		return "minecraft:" + s
	}
	return s
}
