package changes

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/disablers"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// --- Generic: config keys -----------------------------------------------------

// ConfigKeysID is the ID of the ConfigKeys detector.
const ConfigKeysID = "config-keys"

// ConfigKeys is the catch-all for mod configs (config/, defaultconfigs/, the
// model world's serverconfig/): keys about loot, drops, trades, fishing or
// bartering that switch something off, change a chance or list items are
// reported as possible changes, with the comment above the key, which mods
// use to explain it ("Makes Sheep not drop Wool when killed").
type ConfigKeys struct{}

func (ConfigKeys) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ConfigKeysID, Phase: discovery.PhaseGeneric, Priority: 10}
}

var (
	configExt   = map[string]bool{".toml": true, ".json": true, ".json5": true, ".cfg": true, ".properties": true, ".yaml": true, ".yml": true}
	skipConfigs = regexp.MustCompile(`(?i)(^|/)(lootr[^/]*|ftbquests|ftbteams|jei|emi|rei|xaero[^/]*|journeymap|kubejs|tide|starcatcher|lostcities)(/|$)|client|_client|-client`)
	changeWords = regexp.MustCompile(`(?i)disabl|enabl|remov|blacklist|blocklist|whitelist|allow|chance|multipl|weight|nerf|prevent|replace|\badd|max|min|amount|count|rate|bonus|override|custom|items|table|reward|loot`)
	offWords    = regexp.MustCompile(`(?i)disabl|remov|blacklist|blocklist|prevent|nerf|deny|ban`)
	onWords     = regexp.MustCompile(`(?i)enabl|allow`)
)

func (ConfigKeys) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	dirs := []string{"config", "defaultconfigs"}
	if w := in.Files.Level(); w != nil {
		if rel, err := filepath.Rel(in.Files.RootDir(), filepath.Join(w.Path, "serverconfig")); err == nil {
			dirs = append(dirs, filepath.ToSlash(rel))
		}
	}
	for _, dir := range dirs {
		scriptFiles(in, dir, configExt, func(file string, lines []string) {
			if skipConfigs.MatchString(strings.TrimPrefix(file, dir+"/")) {
				return
			}
			mod := configMod(in, file)
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
					continue
				}
				key := disablers.KeyOf(line)
				if key == "" || strings.Contains(key, ":") {
					continue // ids used as keys name a thing, not a setting
				}
				value := valueOf(lines, i)
				kind, ok := kindOfKey(key)
				if !ok && offWords.MatchString(key) && strings.EqualFold(strings.Trim(value, ` ",`), "true") {
					// "Disable Iron Farms = true" says what in its comment:
					// "Makes Iron Golems not drop Iron Ingots".
					kind, ok = kindOfKey(commentAbove(lines, i))
				}
				if !ok || out.Covered(origin(file, i+1)) {
					continue
				}
				lowVal := strings.ToLower(strings.Trim(value, ` ",`))
				isList := strings.HasPrefix(lowVal, "[")
				if !isList && !changeWords.MatchString(key) {
					continue
				}
				switch {
				case lowVal == "" || lowVal == "[]" || lowVal == "{}" || lowVal == `""`:
					continue
				case lowVal == "false" && offWords.MatchString(key):
					continue
				case lowVal == "true" && onWords.MatchString(key) && !offWords.MatchString(key):
					continue
				case strings.HasPrefix(lowVal, "{"):
					continue // a section header, not a value
				}
				comment := commentAbove(lines, i)
				detail := strings.Trim(key, `"' `) + " = " + shorten(strings.TrimSpace(value), 120)
				if comment != "" {
					detail += " — " + comment
				}
				target := domain.ChangeTarget{Kind: kind}
				if kind == domain.ChangeLootType {
					target.ID = domain.ResourceID{Namespace: "loot", Path: "all"}
				}
				out.Add(domain.Change{Target: target, Effect: configEffect(key, lowVal), Certainty: domain.Possibly, Mod: mod,
					Origin: origin(file, i+1), By: ConfigKeysID, Items: k.itemsOf(idsIn(value)), Detail: detail})
			}
		})
	}
	return nil
}

var (
	reCamel = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	reWords = regexp.MustCompile(`[a-z0-9]+`)
)

// keyWords splits a config key into lowercase words ("dropChance",
// "drop_chance", "Drop Chance" → drop, chance).
func keyWords(key string) map[string]bool {
	out := map[string]bool{}
	for _, w := range reWords.FindAllString(strings.ToLower(reCamel.ReplaceAllString(key, "$1 $2")), -1) {
		out[w] = true
	}
	return out
}

// Words that name each kind of source in a config key, in priority order,
// and words that show the key is about something else (spawning a mob that
// carries a rod, fluid "drops", sounds…).
var (
	kindKeyWords = []struct {
		kind  domain.ChangeKind
		words []string
	}{
		{domain.ChangeBarter, []string{"barter", "bartering", "barters"}},
		{domain.ChangeTrades, []string{"trade", "trades", "trading", "trader", "traders", "merchant", "merchants", "offers"}},
		{domain.ChangeFishing, []string{"fishing", "fish", "angler", "fished"}},
		{domain.ChangeDrops, []string{"drop", "drops", "dropped"}},
		{domain.ChangeLootType, []string{"loot", "loots", "treasure", "chest", "chests"}},
	}
	notSourceWords = []string{"equip", "spawn", "spawns", "spawning", "sound", "sounds", "voice", "voices", "particle", "particles",
		"render", "predictable", "evaporate", "fluid", "loops", "client", "gui", "screen", "tooltip", "hud", "color", "texture", "model", "animation",
		"effect", "effects", "amplifier", "thresholds", "slot", "access", "health", "snore", "juice", "bottle"}
)

// kindOfKey tells which kind of source a config key is about.
func kindOfKey(key string) (domain.ChangeKind, bool) {
	words := keyWords(key)
	for _, w := range notSourceWords {
		if words[w] {
			return "", false
		}
	}
	for _, kw := range kindKeyWords {
		for _, w := range kw.words {
			if words[w] {
				return kw.kind, true
			}
		}
	}
	return "", false
}

func configEffect(key, value string) domain.Effect {
	switch {
	case offWords.MatchString(key) && value == "true":
		return domain.EffectRemove
	case onWords.MatchString(key) && value == "false":
		return domain.EffectRemove
	}
	return domain.EffectChange
}

// valueOf returns the value of the key at line i, following multi-line
// lists up to their closing bracket.
func valueOf(lines []string, i int) string {
	line := lines[i]
	idx := strings.IndexAny(line, "=:")
	if idx < 0 {
		return ""
	}
	v := strings.TrimSpace(line[idx+1:])
	if strings.HasPrefix(v, "[") && !strings.Contains(v, "]") {
		var b strings.Builder
		b.WriteString(v)
		for j := i + 1; j < len(lines) && j < i+80; j++ {
			b.WriteString(" " + strings.TrimSpace(lines[j]))
			if strings.Contains(lines[j], "]") {
				break
			}
		}
		v = b.String()
	}
	return v
}

// commentAbove joins the comment lines right above line i.
func commentAbove(lines []string, i int) string {
	var parts []string
	for j := i - 1; j >= 0 && j >= i-6; j-- {
		t := strings.TrimSpace(lines[j])
		if !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "//") {
			break
		}
		t = strings.TrimSpace(strings.TrimLeft(t, "#/"))
		if strings.HasPrefix(strings.ToLower(t), "range:") || strings.HasPrefix(strings.ToLower(t), "default:") || strings.HasPrefix(strings.ToLower(t), "allowed values") {
			continue
		}
		if t != "" {
			parts = append([]string{t}, parts...)
		}
	}
	return shorten(strings.Join(parts, " "), 200)
}

func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

var reConfigSuffix = regexp.MustCompile(`(?i)[-_.](common|server|general|config|settings)$`)

// configMod guesses the mod of a config file from its folder or name.
func configMod(in discovery.Input, file string) string {
	parts := strings.Split(file, "/")
	cands := []string{}
	if len(parts) > 2 {
		cands = append(cands, parts[1])
	}
	base := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(file))
	for {
		trimmed := reConfigSuffix.ReplaceAllString(base, "")
		if trimmed == base {
			break
		}
		base = trimmed
	}
	cands = append(cands, base)
	for _, c := range cands {
		id := strings.ToLower(strings.ReplaceAll(c, "-", "_"))
		if in.Target.Mods[id] {
			return id
		}
	}
	return cands[0]
}

// --- Generic: code that hooks loot, trades or fishing ---------------------------

// CodeHooksID is the ID of the CodeHooks detector.
const CodeHooksID = "code-hooks"

// CodeHooks is the catch-all for mods that change sources in Java code: it
// looks inside each jar's classes for the events and mixin targets used to
// change loot tables, mob drops, trades, fishing and bartering (Forge,
// NeoForge and Fabric names). It cannot know what they do, so it only says
// "possibly", once per mod and kind.
type CodeHooks struct{}

func (CodeHooks) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: CodeHooksID, Phase: discovery.PhaseGeneric, Priority: 0}
}

type hook struct {
	kind   domain.ChangeKind
	marker []byte
	what   string
	mixin  bool // only counts inside a @Mixin class
}

var hooks = []hook{
	{domain.ChangeLootType, []byte("event/LootTableLoadEvent"), "LootTableLoadEvent", false},
	{domain.ChangeLootType, []byte("fabric/api/loot/v2/LootTableEvents"), "LootTableEvents (Fabric)", false},
	{domain.ChangeLootType, []byte("fabric/api/loot/v3/LootTableEvents"), "LootTableEvents (Fabric)", false},
	{domain.ChangeLootType, []byte("net/minecraft/world/level/storage/loot/LootTable;"), "mixin en LootTable", true},
	{domain.ChangeDrops, []byte("event/entity/living/LivingDropsEvent"), "LivingDropsEvent", false},
	{domain.ChangeTrades, []byte("event/village/VillagerTradesEvent"), "VillagerTradesEvent", false},
	{domain.ChangeTrades, []byte("event/village/WandererTradesEvent"), "WandererTradesEvent", false},
	{domain.ChangeTrades, []byte("fabric/api/object/builder/v1/trade/TradeOfferHelper"), "TradeOfferHelper (Fabric)", false},
	{domain.ChangeTrades, []byte("net/minecraft/world/entity/npc/VillagerTrades"), "mixin en VillagerTrades", true},
	{domain.ChangeFishing, []byte("event/entity/player/ItemFishedEvent"), "ItemFishedEvent", false},
	{domain.ChangeFishing, []byte("net/minecraft/world/entity/projectile/FishingHook;"), "mixin en FishingHook", true},
	{domain.ChangeBarter, []byte("net/minecraft/world/entity/monster/piglin/PiglinAi;"), "mixin en PiglinAi", true},
}

var mixinMarker = []byte("Lorg/spongepowered/asm/mixin/Mixin;")

// API and loader jars define the hooks themselves.
var skipJars = regexp.MustCompile(`(?i)^(fabric-api|fabric_api|forge-|neoforge-|architectury|kotlinforforge|mixinextras)`)

func (CodeHooks) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	type result struct {
		file  string
		found map[domain.ChangeKind][]string
	}
	jars := in.Files.Jars()
	results := make([]result, len(jars))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, j := range jars {
		if skipJars.MatchString(j.File) {
			continue
		}
		wg.Add(1)
		go func(i int, j discovery.Jar) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			found := map[domain.ChangeKind]map[string]bool{}
			for _, path := range j.Pack.Files() {
				if !strings.HasSuffix(path, ".class") {
					continue
				}
				data, err := resources.ReadFile(j.Pack, path)
				if err != nil {
					continue
				}
				isMixin := bytes.Contains(data, mixinMarker)
				for _, h := range hooks {
					if h.mixin && !isMixin {
						continue
					}
					if bytes.Contains(data, h.marker) {
						if found[h.kind] == nil {
							found[h.kind] = map[string]bool{}
						}
						found[h.kind][h.what] = true
					}
				}
			}
			r := result{file: j.File, found: map[domain.ChangeKind][]string{}}
			for kind, set := range found {
				for w := range set {
					r.found[kind] = append(r.found[kind], w)
				}
				sort.Strings(r.found[kind])
			}
			results[i] = r
		}(i, j)
	}
	wg.Wait()
	for _, r := range results {
		if r.file == "" {
			continue
		}
		kinds := make([]string, 0, len(r.found))
		for kind := range r.found {
			kinds = append(kinds, string(kind))
		}
		sort.Strings(kinds)
		for _, ks := range kinds {
			kind := domain.ChangeKind(ks)
			target := domain.ChangeTarget{Kind: kind}
			if kind == domain.ChangeLootType {
				target.ID = domain.ResourceID{Namespace: "loot", Path: "all"}
			}
			out.Add(domain.Change{Target: target, Effect: domain.EffectUnknown, Certainty: domain.Possibly,
				Mod: in.Files.ModName(r.file), Origin: "jar:" + r.file + ":" + ks, By: CodeHooksID,
				Detail: fmt.Sprintf("su código usa %s: puede cambiar %s", strings.Join(r.found[kind], ", "), kindWords[kind])})
		}
	}
	return nil
}
