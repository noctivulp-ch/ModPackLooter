// Package names turns resource ids into human names using the lang files of
// the modpack, falling back to English and then to a readable form of the id.
package names

import (
	"strings"
	"unicode"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// Namer resolves display names.
type Namer struct {
	langs []map[string]string   // preferred language first
	byEnd []map[string][]string // per lang: last key segment -> keys, built on first Asset call
}

// New creates a namer from lang maps, most preferred first.
func New(langs ...map[string]string) *Namer { return &Namer{langs: langs} }

func (n *Namer) lookup(keys ...string) (string, bool) {
	for _, lang := range n.langs {
		for _, k := range keys {
			if v, ok := lang[k]; ok && v != "" {
				return v, true
			}
		}
	}
	return "", false
}

// key builds "<prefix>.<namespace>.<path with dots>".
func key(prefix string, id domain.ResourceID) string {
	return prefix + "." + id.Namespace + "." + strings.ReplaceAll(id.Path, "/", ".")
}

// Item returns the name of an item or block.
func (n *Namer) Item(id domain.ResourceID) string {
	if v, ok := n.lookup(key("item", id), key("block", id)); ok {
		return v
	}
	return Humanize(id.Path)
}

// Biome returns the name of a biome.
func (n *Namer) Biome(id domain.ResourceID) string {
	if v, ok := n.lookup(key("biome", id)); ok {
		return v
	}
	return Humanize(id.Path)
}

// Entity returns the name of an entity.
func (n *Namer) Entity(id domain.ResourceID) string {
	if v, ok := n.lookup(key("entity", id)); ok {
		return v
	}
	return Humanize(id.Path)
}

// Structure returns the name of a structure. Few mods translate them, so
// the readable id is the usual result.
func (n *Namer) Structure(id domain.ResourceID) string {
	if v, ok := n.lookup(key("structure", id), key("structures", id), key("filled_map", id)); ok {
		return v
	}
	return Humanize(id.Path)
}

// Humanize turns "chests/village/village_weaponsmith" into
// "Village Weaponsmith" (last segment, words capitalised).
func Humanize(path string) string {
	last := path[strings.LastIndex(path, "/")+1:]
	words := strings.FieldsFunc(last, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	if len(words) == 0 {
		return path
	}
	return strings.Join(words, " ")
}

// Asset names a mod asset that has no standard translation key, such as a
// Lost Cities city style or building. The reader of the site is not assumed
// to be technical, so every lang key that may name it is tried before the
// readable id:
//  1. explicit keys: "<kind>.<ns>.<path>", "<ns>.<kind>.<path>",
//     "<ns>.<kind>s.<path>", "lostcities.<kind>.<path>";
//  2. any key of the asset's namespace ending in ".<path>" that looks like a
//     name ("title", "name"), e.g. the advancement titles some modpacks give
//     each district ("deceasedcraft.advancement.title.retail_district");
//  3. any other key of the namespace ending in ".<path>" that is not a
//     description or tooltip.
//
// Languages are tried in the order of the chain (requested, its family,
// en_us): any key in a preferred language wins over a better key in a
// fallback one. Empty values count as missing.
func (n *Namer) Asset(kind string, id domain.ResourceID) string {
	path := strings.ReplaceAll(id.Path, "/", ".")
	explicit := []string{
		kind + "." + id.Namespace + "." + path, id.Namespace + "." + kind + "." + path,
		id.Namespace + "." + kind + "s." + path, "lostcities." + kind + "." + path,
	}
	suffix := "." + path
	prefix := id.Namespace + "."
	if n.byEnd == nil {
		for _, lang := range n.langs {
			idx := map[string][]string{}
			for k := range lang {
				idx[k[strings.LastIndex(k, ".")+1:]] = append(idx[k[strings.LastIndex(k, ".")+1:]], k)
			}
			n.byEnd = append(n.byEnd, idx)
		}
	}
	last := path[strings.LastIndex(path, ".")+1:]
	for i, lang := range n.langs {
		for _, k := range explicit {
			if v := lang[k]; v != "" {
				return v
			}
		}
		var named, other string
		for _, k := range n.byEnd[i][last] {
			v := lang[k]
			if v == "" || !strings.HasPrefix(k, prefix) || !strings.HasSuffix(k, suffix) {
				continue
			}
			low := strings.ToLower(k)
			if strings.Contains(low, "desc") || strings.Contains(low, "tooltip") || strings.Contains(low, "lore") {
				continue
			}
			// Deterministic choice: the shortest key, then alphabetical.
			better := func(cur string) bool { return cur == "" || len(k) < len(cur) || len(k) == len(cur) && k < cur }
			if strings.Contains(low, "title") || strings.Contains(low, "name") {
				if better(named) {
					named = k
				}
			} else if better(other) {
				other = k
			}
		}
		if named != "" {
			return lang[named]
		}
		if other != "" {
			return lang[other]
		}
	}
	// "citystyle_desert" reads better as "Desert" under "Estilos de ciudad".
	base := id.Path[strings.LastIndex(id.Path, "/")+1:]
	if rest, ok := strings.CutPrefix(base, kind+"_"); ok && rest != "" {
		return Humanize(rest)
	}
	return Humanize(id.Path)
}

// BiomeTag names a biome tag ("minecraft:is_ocean" → "Ocean" unless a
// lang file names it).
func (n *Namer) BiomeTag(id domain.ResourceID) string {
	if v, ok := n.lookup(key("tag.worldgen.biome", id), key("biome_tag", id)); ok {
		return v
	}
	base := id.Path[strings.LastIndex(id.Path, "/")+1:]
	if rest, ok := strings.CutPrefix(base, "is_"); ok && rest != "" {
		return Humanize(rest)
	}
	return Humanize(id.Path)
}

// Text returns the translation of a lang key, following the language chain.
func (n *Namer) Text(key string) (string, bool) { return n.lookup(key) }
