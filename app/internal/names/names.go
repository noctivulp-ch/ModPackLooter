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
	langs []map[string]string // preferred language first
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
