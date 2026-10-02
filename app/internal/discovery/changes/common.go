// Package changes detects modifications of loot sources (loot tables, mob
// drops, trades, fishing, bartering) made by mods, datapacks, scripts and
// configs. Detectors that understand a format (datapack overrides, Global
// Loot Modifiers, LootJS, KubeJS loot events, MoreJS trades, CraftTweaker)
// run first and report what they can read; generic detectors (config keys,
// script lines, code that listens to loot/trade/fishing events) run last and
// catch the rest, skipping origins already explained.
//
// See docs/17-cambios-de-fuentes.md.
package changes

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// known tells items and loot tables apart from other ids.
type known struct {
	in    discovery.Input
	items map[domain.ResourceID]bool
}

func newKnown(in discovery.Input) *known {
	k := &known{in: in, items: map[domain.ResourceID]bool{}}
	for _, code := range []string{"en_us", "es_es"} {
		for key := range in.Resources.Lang(code) {
			for _, prefix := range []string{"item.", "block."} {
				rest, ok := strings.CutPrefix(key, prefix)
				if !ok {
					continue
				}
				ns, path, ok := strings.Cut(rest, ".")
				if !ok || strings.Contains(path, ".") {
					continue
				}
				k.items[domain.ResourceID{Namespace: ns, Path: path}] = true
			}
		}
	}
	return k
}

func (k *known) isItem(id domain.ResourceID) bool { return k.items[id] }

func (k *known) isTable(id domain.ResourceID) bool {
	return len(k.in.Resources.Providers(resources.TypeLootTable, id)) > 0
}

// ids finds resource ids in text: quoted "ns:path" and CraftTweaker
// <item:ns:path> brackets.
var (
	reQuotedID  = regexp.MustCompile(`["'` + "`" + `]#?([a-z0-9_.-]+:[a-z0-9_./-]+)["'` + "`" + `]`)
	reBracketID = regexp.MustCompile(`<(?:item|block|resource|entitytype|profession|tag:items):([a-z0-9_.-]+:[a-z0-9_./-]+)>`)
)

func idsIn(text string) []domain.ResourceID {
	var out []domain.ResourceID
	seen := map[domain.ResourceID]bool{}
	for _, re := range []*regexp.Regexp{reQuotedID, reBracketID} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			id, err := domain.ParseResourceID(m[1])
			if err == nil && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

// split returns the items among ids.
func (k *known) itemsOf(ids []domain.ResourceID) []domain.ResourceID {
	var out []domain.ResourceID
	for _, id := range ids {
		if k.isItem(id) {
			out = append(out, id)
		}
	}
	return out
}

// lootTypes maps LootJS / KubeJS loot type names to the canonical ones.
var lootTypes = map[string]string{
	"chest": "chest", "chests": "chest", "entity": "entity", "entities": "entity",
	"block": "block", "blocks": "block", "fishing": "fishing", "gift": "gameplay",
	"gameplay": "gameplay", "archaeology": "archaeology", "piglin_barter": "barter",
	"advancement_reward": "gameplay", "unknown": "all",
}

// lootTypeTarget builds the target of "every table of a type".
func lootTypeTarget(t string) domain.ChangeTarget {
	t = strings.ToLower(t)
	if canon, ok := lootTypes[t]; ok {
		t = canon
	}
	switch t {
	case "fishing":
		return domain.ChangeTarget{Kind: domain.ChangeFishing}
	case "barter":
		return domain.ChangeTarget{Kind: domain.ChangeBarter}
	case "entity":
		return domain.ChangeTarget{Kind: domain.ChangeDrops}
	}
	return domain.ChangeTarget{Kind: domain.ChangeLootType, ID: domain.ResourceID{Namespace: "loot", Path: t}}
}

// tableTarget builds the target of a loot table, using the specific kind
// when the table is a fishing or barter table.
func tableTarget(id domain.ResourceID) domain.ChangeTarget {
	return domain.ChangeTarget{Kind: domain.ChangeTable, ID: id}
}

// scriptFiles walks text files under dir (relative to the game folder).
func scriptFiles(in discovery.Input, dir string, exts map[string]bool, fn func(rel string, lines []string)) {
	root := filepath.Join(in.Files.RootDir(), filepath.FromSlash(dir))
	_ = filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() || !exts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		info, err := e.Info()
		if err != nil || info.Size() > 2<<20 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(in.Files.RootDir(), path)
		fn(filepath.ToSlash(rel), strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"))
		return nil
	})
}

// stripComment removes // and # comments from a script line (naive: ignores
// comment markers inside strings only when the line has quotes before them).
func stripComment(line string) string {
	if i := strings.Index(line, "//"); i >= 0 && strings.Count(line[:i], `"`)%2 == 0 && strings.Count(line[:i], `'`)%2 == 0 {
		line = line[:i]
	}
	return line
}

func origin(file string, line int) string {
	return file + ":" + strconv.Itoa(line)
}
