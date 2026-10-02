package site

import (
	"encoding/json"
	"fmt"
	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"hash/fnv"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
)

// itemKey tells items apart: each useful variant has its own page.
type itemKey struct {
	id domain.ResourceID
	v  domain.Variant
}

// variantSlug is the folder of a variant page under its item.
func variantSlug(v domain.Variant) string {
	s := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, strings.ToLower(strings.TrimPrefix(v.String(), v.Kind+":")))
	if len(s) > 48 || v.Kind == domain.VariantTitle {
		h := fnv.New32a()
		h.Write([]byte(v.String()))
		if len(s) > 40 {
			s = s[:40]
		}
		s = fmt.Sprintf("%s-%08x", s, h.Sum32())
	}
	return v.Kind + "-" + s
}

var romans = []string{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"}

func levelName(namer *names.Namer, level int) string {
	if v, ok := namer.Text(fmt.Sprintf("enchantment.level.%d", level)); ok {
		return v
	}
	if level < len(romans) {
		return romans[level]
	}
	return fmt.Sprint(level)
}

// effectName names a mob effect.
func effectName(namer *names.Namer, id string) string {
	rid, err := domain.ParseResourceID(id)
	if err != nil {
		return id
	}
	if v, ok := namer.Text("effect." + rid.Namespace + "." + rid.Path); ok {
		return v
	}
	return names.Humanize(rid.Path)
}

// Explorer map destinations and the lang key of the map they make.
var mapNames = map[string]string{
	"on_treasure_maps":          "filled_map.buried_treasure",
	"on_ocean_explorer_maps":    "filled_map.monument",
	"on_woodland_explorer_maps": "filled_map.mansion",
	"on_trial_chambers_maps":    "filled_map.trial_chambers",
	"on_jungle_explorer_maps":   "filled_map.explorer_jungle",
	"on_swamp_explorer_maps":    "filled_map.explorer_swamp",
	"on_desert_village_maps":    "filled_map.village_desert",
	"on_plains_village_maps":    "filled_map.village_plains",
	"on_savanna_village_maps":   "filled_map.village_savanna",
	"on_snowy_village_maps":     "filled_map.village_snowy",
	"on_taiga_village_maps":     "filled_map.village_taiga",
}

// variantLabel names what makes the variant special ("Reparación",
// "Curación instantánea II", "UMP45"…), without the item name.
func variantLabel(namer *names.Namer, item domain.ResourceID, v domain.Variant) (label string, whole bool) {
	switch v.Kind {
	case domain.VariantEnchantment:
		var parts []string
		for _, id := range strings.Split(v.Value, ",") {
			parts = append(parts, enchantName(namer, id))
		}
		s := strings.Join(parts, " + ")
		if v.Level > 0 && len(parts) == 1 {
			s += " " + levelName(namer, v.Level)
		}
		return s, false
	case domain.VariantPotion:
		rid, err := domain.ParseResourceID(v.Value)
		if err != nil {
			return v.Value, false
		}
		base, suffix := rid.Path, ""
		if b, ok := strings.CutPrefix(base, "strong_"); ok {
			base, suffix = b, " II"
		} else if b, ok := strings.CutPrefix(base, "long_"); ok {
			base, suffix = b, " (prolongada)"
		}
		// The game names potions "item.minecraft.<item>.effect.<potion>".
		if s, ok := namer.Text("item.minecraft." + item.Path + ".effect." + base); ok && item.Namespace == "minecraft" {
			return s + suffix, true
		}
		return effectName(namer, rid.Namespace+":"+base) + suffix, false
	case domain.VariantEffect:
		return effectName(namer, v.Value), false
	case domain.VariantInstrument:
		rid, err := domain.ParseResourceID(v.Value)
		if err != nil {
			return v.Value, false
		}
		if s, ok := namer.Text("instrument." + rid.Namespace + "." + rid.Path); ok {
			return s, false
		}
		return names.Humanize(strings.TrimSuffix(rid.Path, "_goat_horn")), false
	case domain.VariantNBT:
		word := strings.ToLower(strings.TrimSuffix(v.Key, "Id"))
		rid, err := domain.ParseResourceID(v.Value)
		if err != nil {
			return v.Value, false
		}
		for _, k := range []string{
			rid.Namespace + "." + word + "." + rid.Path + ".name",
			rid.Namespace + "." + word + "." + rid.Path,
			word + "." + rid.Namespace + "." + rid.Path,
			"item." + rid.Namespace + "." + rid.Path,
		} {
			if s, ok := namer.Text(k); ok {
				return s, false
			}
		}
		return namer.Asset(word, rid), false
	case domain.VariantTitle:
		return textOf(namer, v.Value), false
	case domain.VariantMap:
		rid, err := domain.ParseResourceID(v.Value)
		if err != nil {
			return v.Value, false
		}
		if key, ok := mapNames[rid.Path]; ok {
			if s, ok := namer.Text(key); ok {
				return s, true
			}
		}
		return names.Humanize(strings.TrimSuffix(strings.TrimPrefix(rid.Path, "on_"), "_maps")), false
	}
	return v.String(), false
}

// textOf reads a lang key, a JSON text component or plain text.
func textOf(namer *names.Namer, s string) string {
	if strings.HasPrefix(s, "{") {
		var c struct {
			Translate string `json:"translate"`
			Text      string `json:"text"`
		}
		if json.Unmarshal([]byte(s), &c) == nil {
			if c.Translate != "" {
				s = c.Translate
			} else if c.Text != "" {
				return c.Text
			}
		}
	}
	if v, ok := namer.Text(s); ok {
		return v
	}
	return s
}

// variantName is the full name of a variant page: "Libro encantado:
// Reparación", or the game's own name when it has one ("Poción de
// curación").
func variantName(namer *names.Namer, base string, item domain.ResourceID, v domain.Variant) (name, label string) {
	label, whole := variantLabel(namer, item, v)
	if whole {
		return label, label
	}
	return base + ": " + label, label
}

// englishNamer names things in English, so players can also search by the
// names of the wikis and videos. Nil when the site is already in English.
func englishNamer(res *analysis.Result, opts Options) *names.Namer {
	if strings.HasPrefix(opts.Lang, "en_") {
		return nil
	}
	l := res.Resources.Lang("en_us")
	if len(l) == 0 {
		return nil
	}
	return names.New(l)
}
