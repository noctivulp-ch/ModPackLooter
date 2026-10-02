package changes

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

var jsExt = map[string]bool{".js": true}

// source is a script file joined into one string with line offsets, so
// multi-line call chains can be read.
type source struct {
	file   string
	text   string
	starts []int // offset of each line
}

func newSource(file string, lines []string) *source {
	s := &source{file: file}
	var b strings.Builder
	for _, l := range lines {
		s.starts = append(s.starts, b.Len())
		b.WriteString(stripComment(l))
		b.WriteByte('\n')
	}
	s.text = b.String()
	return s
}

// line returns the 1-based line of an offset.
func (s *source) line(off int) int {
	return sort.Search(len(s.starts), func(i int) bool { return s.starts[i] > off })
}

// cover marks every line of [from, to) as explained.
func (s *source) cover(out *discovery.Changes, from, to int) {
	for l := s.line(from); l <= s.line(max(from, to-1)); l++ {
		out.Cover(origin(s.file, l))
	}
}

// statementEnd returns where the statement that contains text[from] ends:
// the first ';' outside brackets, or the bracket that closes the enclosing
// block (a callback), whichever comes first.
func statementEnd(text string, from int) int {
	depth := 0
	inStr := byte(0)
	for i := from; i < len(text); i++ {
		c := text[i]
		if inStr != 0 {
			if c == '\\' {
				i++
			} else if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			inStr = c
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth < 0 {
				return i
			}
		case ';':
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(text)
}

// coverHeaders marks the lines that open script events (they say nothing
// by themselves) as explained.
func (s *source) coverHeaders(out *discovery.Changes, re *regexp.Regexp) {
	for _, m := range re.FindAllStringIndex(s.text, -1) {
		out.Cover(origin(s.file, s.line(m[0])))
	}
}

var reEventHeader = regexp.MustCompile(`(LootJS\.(modifiers|lootTables)|MoreJSEvents\.\w+|ServerEvents\.\w+LootTables)\s*\(`)

// args returns the text inside the parenthesis that opens at text[open].
func args(text string, open int) string {
	depth := 0
	for i := open; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[open+1 : i]
			}
		}
	}
	return text[open+1:]
}

// --- LootJS -----------------------------------------------------------------

// LootJSID is the ID of the LootJS detector.
const LootJSID = "lootjs"

// LootJS reads LootJS modifiers in KubeJS server scripts (LootJS 2 for
// 1.20.1 and LootJS 3 for 1.21): which tables, loot types, blocks or mobs
// they target and whether they add, remove, replace or modify loot.
type LootJS struct{}

func (LootJS) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: LootJSID, Phase: discovery.PhaseSpecific, Priority: 80}
}

var (
	reLootJSStart  = regexp.MustCompile(`\.(addLootTableModifier|addTableModifier|addLootTypeModifier|addTypeModifier|addBlockLootModifier|addBlockModifier|addEntityLootModifier|addEntityModifier)\s*\(`)
	reLootJSAction = regexp.MustCompile(`\.(addLoot|addWeightedLoot|addAlternativesLoot|addSequenceLoot|removeLoot|replaceLoot|modifyLoot|modifyItem|addCustomTrigger|triggerLootTable|apply|dropExperience)\s*\(`)
	reLootJSCond   = regexp.MustCompile(`\.(randomChance|randomChanceWithLooting|randomChanceWithEnchantment|playerPredicate|entityPredicate|matchMainHand|matchOffHand|killedByPlayer|biome|anyBiome|dimension|anyDimension|matchEntity|matchPlayer|matchEquip|hasAnyStage|weather|isRaining|timeCheck|survivesExplosion|blockEntityPredicate|lightLevel|not|or|and)\s*\(`)
	reLootType     = regexp.MustCompile(`LootType\.([A-Z_]+)`)
	reRegexArg     = regexp.MustCompile(`/((?:[^/\\\n]|\\.)+)/`)
)

func (LootJS) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	scriptFiles(in, "kubejs/server_scripts", jsExt, func(file string, lines []string) {
		s := newSource(file, lines)
		s.coverHeaders(out, reEventHeader)
		starts := reLootJSStart.FindAllStringSubmatchIndex(s.text, -1)
		for i, m := range starts {
			end := statementEnd(s.text, m[1]-1)
			if i+1 < len(starts) && starts[i+1][0] < end {
				end = starts[i+1][0]
			}
			chunk := s.text[m[0]:end]
			method := s.text[m[2]:m[3]]
			targetArgs := args(s.text, m[1]-1)
			targets := lootJSTargets(method, targetArgs)
			var conds []string
			for _, c := range reLootJSCond.FindAllStringSubmatch(chunk, -1) {
				conds = append(conds, c[1])
			}
			org := origin(file, s.line(m[0]))
			var changes []domain.Change
			for _, a := range reLootJSAction.FindAllStringSubmatchIndex(chunk, -1) {
				action := chunk[a[2]:a[3]]
				body := args(chunk, a[1]-1)
				found := idsIn(body)
				c := domain.Change{Certainty: domain.Certainly, Mod: "LootJS (KubeJS)", Origin: org, By: LootJSID}
				switch action {
				case "removeLoot":
					c.Effect, c.Items = domain.EffectRemove, k.itemsOf(found)
					if strings.Contains(body, "Ingredient.all") || strings.Contains(body, `"*"`) {
						c.Detail = "quita todo el loot"
					}
				case "replaceLoot":
					parts := splitArgs(body)
					c.Effect = domain.EffectReplace
					if len(parts) >= 2 {
						c.Items, c.With = k.itemsOf(idsIn(parts[0])), k.itemsOf(idsIn(parts[1]))
					}
				case "modifyLoot", "modifyItem", "apply":
					c.Effect, c.Items = domain.EffectChange, k.itemsOf(found)
					if strings.Contains(body, "Ingredient.all") {
						c.Detail = "modifica todo el loot (cantidades, datos…)"
					} else {
						c.Detail = "modifica cantidades o datos"
					}
				case "triggerLootTable":
					c.Effect = domain.EffectAdd
					for _, id := range found {
						if k.isTable(id) {
							c.Table = id
							break
						}
					}
				case "dropExperience", "addCustomTrigger":
					c.Effect, c.Detail = domain.EffectChange, action
				default:
					c.Effect, c.Items = domain.EffectAdd, k.itemsOf(found)
				}
				if len(conds) > 0 {
					c.Detail = strings.TrimPrefix(c.Detail+"; con condiciones: "+strings.Join(conds, ", "), "; ")
				}
				changes = append(changes, c)
			}
			if len(changes) == 0 {
				changes = append(changes, domain.Change{Effect: domain.EffectUnknown, Certainty: domain.Possibly,
					Mod: "LootJS (KubeJS)", Origin: org, By: LootJSID, Detail: "modificador de LootJS sin acciones reconocidas"})
			}
			for _, t := range targets {
				for _, c := range changes {
					c.Target = t.target
					if t.note != "" {
						c.Detail = strings.TrimPrefix(c.Detail+"; "+t.note, "; ")
					}
					out.Add(c)
				}
			}
			s.cover(out, m[0], end)
		}
	})
	return nil
}

type lootTarget struct {
	target domain.ChangeTarget
	note   string
}

func lootJSTargets(method, body string) []lootTarget {
	var out []lootTarget
	switch {
	case strings.Contains(method, "Type"):
		for _, m := range reLootType.FindAllStringSubmatch(body, -1) {
			out = append(out, lootTarget{target: lootTypeTarget(m[1])})
		}
	case strings.Contains(method, "Block"):
		for _, id := range idsIn(body) {
			out = append(out, lootTarget{target: tableTarget(domain.ResourceID{Namespace: id.Namespace, Path: "blocks/" + id.Path})})
		}
	case strings.Contains(method, "Entity"):
		for _, id := range idsIn(body) {
			out = append(out, lootTarget{target: domain.ChangeTarget{Kind: domain.ChangeDrops, ID: id}})
		}
	default:
		for _, id := range idsIn(body) {
			out = append(out, lootTarget{target: tableTarget(id)})
		}
	}
	for _, m := range reRegexArg.FindAllStringSubmatch(body, -1) {
		out = append(out, lootTarget{target: domain.ChangeTarget{Kind: domain.ChangeLootType, ID: domain.ResourceID{Namespace: "loot", Path: "all"}},
			note: "tablas que coinciden con /" + m[1] + "/"})
	}
	if len(out) == 0 {
		out = append(out, lootTarget{target: domain.ChangeTarget{Kind: domain.ChangeLootType, ID: domain.ResourceID{Namespace: "loot", Path: "all"}},
			note: "objetivo no reconocido: " + strings.TrimSpace(body)})
	}
	return out
}

// splitArgs splits top-level comma separated arguments.
func splitArgs(body string) []string {
	var out []string
	depth, last := 0, 0
	for i, r := range body {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, body[last:i])
				last = i + 1
			}
		}
	}
	return append(out, body[last:])
}

// --- KubeJS loot table events ----------------------------------------------

// KubeJSLootEventsID is the ID of the KubeJSLootEvents detector.
const KubeJSLootEventsID = "kubejs-loot-events"

// KubeJSLootEvents reads KubeJS' own loot table events
// (ServerEvents.chestLootTables, entityLootTables, blockLootTables,
// fishingLootTables, giftLootTables, genericLootTables): addX replaces a
// whole table, modify changes it.
type KubeJSLootEvents struct{}

func (KubeJSLootEvents) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: KubeJSLootEventsID, Phase: discovery.PhaseSpecific, Priority: 75}
}

var (
	reKJSLootEvent = regexp.MustCompile(`ServerEvents\.(chest|entity|block|fishing|gift|generic)LootTables\s*\(`)
	reKJSLootCall  = regexp.MustCompile(`\.(addChest|addEntity|addBlock|addFishing|addGift|addGeneric|addSimpleBlock|modify|modifyChest|modifyEntity|modifyBlock)\s*\(`)
)

func (KubeJSLootEvents) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	scriptFiles(in, "kubejs/server_scripts", jsExt, func(file string, lines []string) {
		s := newSource(file, lines)
		events := reKJSLootEvent.FindAllStringSubmatchIndex(s.text, -1)
		for i, ev := range events {
			end := len(s.text)
			if i+1 < len(events) {
				end = events[i+1][0]
			}
			kind := s.text[ev[2]:ev[3]]
			region := s.text[ev[1]:end]
			calls := reKJSLootCall.FindAllStringSubmatchIndex(region, -1)
			for j, c := range calls {
				cend := len(region)
				if j+1 < len(calls) {
					cend = calls[j+1][0]
				}
				method := region[c[2]:c[3]]
				body := region[c[0]:cend]
				parts := splitArgs(args(region, c[1]-1))
				if len(parts) == 0 {
					continue
				}
				first := idsIn(parts[0])
				if len(first) == 0 {
					continue
				}
				table := kubeJSTable(kind, method, first[0])
				ch := domain.Change{Target: tableTarget(table), Certainty: domain.Certainly, Mod: "KubeJS",
					Origin: origin(file, s.line(ev[1]+c[0])), By: KubeJSLootEventsID, Items: k.itemsOf(idsIn(body))}
				if strings.HasPrefix(method, "modify") {
					ch.Effect, ch.Detail = domain.EffectChange, "KubeJS modifica la tabla"
				} else {
					ch.Effect, ch.Detail = domain.EffectReplace, "KubeJS redefine la tabla entera"
				}
				out.Add(ch)
			}
			s.cover(out, ev[0], end)
		}
	})
	return nil
}

// kubeJSTable maps the id KubeJS receives to the loot table it writes.
func kubeJSTable(kind, method string, id domain.ResourceID) domain.ResourceID {
	prefix := map[string]string{"chest": "chests/", "entity": "entities/", "block": "blocks/", "fishing": "gameplay/fishing/", "gift": "gameplay/hero_of_the_village/"}[kind]
	if strings.HasPrefix(id.Path, strings.TrimSuffix(prefix, "/")) || prefix == "" {
		return id
	}
	return domain.ResourceID{Namespace: id.Namespace, Path: prefix + id.Path}
}

// --- Trades in KubeJS scripts (MoreJS and similar) -------------------------

// TradeScriptsID is the ID of the TradeScripts detector.
const TradeScriptsID = "trade-scripts"

// TradeScripts reads trade changes in KubeJS scripts: MoreJS
// (MoreJSEvents.villagerTrades / wandererTrades) and the common method
// names other addons use (addTrade, removeTrades, removeVanillaTrades…).
type TradeScripts struct{}

func (TradeScripts) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: TradeScriptsID, Phase: discovery.PhaseSpecific, Priority: 70}
}

var reTradeCall = regexp.MustCompile(`\b(addTrade|addCustomTrade|addRandomTrade|removeTrades|removeVanillaTrades|removeModdedTrades|removeVanillaTypedTrades|removeModdedTypedTrades|removeTrade)\s*\(`)

func (TradeScripts) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	for _, dir := range []string{"kubejs/server_scripts", "kubejs/startup_scripts"} {
		scriptFiles(in, dir, jsExt, func(file string, lines []string) {
			s := newSource(file, lines)
			s.coverHeaders(out, reEventHeader)
			for _, m := range reTradeCall.FindAllStringSubmatchIndex(s.text, -1) {
				method := s.text[m[2]:m[3]]
				body := args(s.text, m[1]-1)
				before := s.text[:m[0]]
				wanderer := strings.LastIndex(strings.ToLower(before), "wanderer") > strings.LastIndex(strings.ToLower(before), "villagertrades")
				target := domain.ChangeTarget{Kind: domain.ChangeTrades}
				found := idsIn(body)
				if wanderer {
					target.ID = domain.MustParseResourceID("minecraft:wandering_trader")
				} else if parts := splitArgs(body); len(parts) > 0 {
					if p := idsIn(parts[0]); len(p) > 0 && !k.isItem(p[0]) {
						target.ID = p[0]
					}
				}
				c := domain.Change{Target: target, Certainty: domain.Certainly, Mod: "KubeJS", Origin: origin(file, s.line(m[0])), By: TradeScriptsID,
					Items: k.itemsOf(found)}
				if strings.HasPrefix(method, "remove") {
					c.Effect = domain.EffectRemove
					switch method {
					case "removeVanillaTrades", "removeVanillaTypedTrades":
						c.Detail = "quita tradeos vanilla"
					case "removeModdedTrades", "removeModdedTypedTrades":
						c.Detail = "quita tradeos de mods"
					}
				} else {
					c.Effect, c.Detail = domain.EffectAdd, "añade un tradeo"
				}
				out.Add(c)
				s.cover(out, m[0], m[1]+len(body)+1)
			}
		})
	}
	return nil
}

// --- CraftTweaker -------------------------------------------------------------

// CraftTweakerID is the ID of the CraftTweaker detector.
const CraftTweakerID = "crafttweaker"

// CraftTweaker reads ZenScript statements that touch loot modifiers or
// trades (loot.modifiers.register, villagerTrades / wanderingTrades).
type CraftTweaker struct{}

func (CraftTweaker) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: CraftTweakerID, Phase: discovery.PhaseSpecific, Priority: 70}
}

func (CraftTweaker) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	scriptFiles(in, "scripts", map[string]bool{".zs": true}, func(file string, lines []string) {
		s := newSource(file, lines)
		start := 0
		for start < len(s.text) {
			end := strings.IndexByte(s.text[start:], ';')
			if end < 0 {
				end = len(s.text)
			} else {
				end += start + 1
			}
			stmt := s.text[start:end]
			low := strings.ToLower(stmt)
			org := origin(file, s.line(start+len(stmt)-len(strings.TrimLeft(stmt, " \t\n"))))
			found := idsIn(stmt)
			switch {
			case strings.Contains(low, "loot.modifiers") || strings.Contains(low, "lootmodifier"):
				c := domain.Change{Certainty: domain.Certainly, Mod: "CraftTweaker", Origin: org, By: CraftTweakerID, Items: k.itemsOf(found)}
				c.Effect = domain.EffectAdd
				if strings.Contains(low, "remove") {
					c.Effect = domain.EffectRemove
				} else if strings.Contains(low, "replace") {
					c.Effect = domain.EffectReplace
				}
				var targets []domain.ResourceID
				for _, id := range found {
					if k.isTable(id) {
						targets = append(targets, id)
					}
				}
				if len(targets) == 0 {
					c.Target = domain.ChangeTarget{Kind: domain.ChangeLootType, ID: domain.ResourceID{Namespace: "loot", Path: "all"}}
					c.Certainty = domain.Possibly
					out.Add(c)
				}
				for _, t := range targets {
					c.Target = tableTarget(t)
					out.Add(c)
				}
				s.cover(out, start, end)
			case strings.Contains(low, "villagertrades") || strings.Contains(low, "wanderingtrades"):
				c := domain.Change{Target: domain.ChangeTarget{Kind: domain.ChangeTrades}, Certainty: domain.Certainly, Mod: "CraftTweaker",
					Origin: org, By: CraftTweakerID, Items: k.itemsOf(found), Effect: domain.EffectAdd}
				if strings.Contains(low, "wanderingtrades") {
					c.Target.ID = domain.MustParseResourceID("minecraft:wandering_trader")
				}
				for _, id := range found {
					if !k.isItem(id) {
						c.Target.ID = id
						break
					}
				}
				if strings.Contains(low, "remove") {
					c.Effect = domain.EffectRemove
				}
				out.Add(c)
				s.cover(out, start, end)
			}
			start = end
		}
	})
	return nil
}

// --- Generic: script lines nobody explained ----------------------------------

// ScriptMentionsID is the ID of the ScriptMentions detector.
const ScriptMentionsID = "script-mentions"

// ScriptMentions is the catch-all for scripts (KubeJS and CraftTweaker):
// lines about loot, drops, trades, fishing or bartering that no specific
// detector explained are reported as possible changes, one per file and
// kind, with the lines.
type ScriptMentions struct{}

func (ScriptMentions) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ScriptMentionsID, Phase: discovery.PhaseGeneric, Priority: 20}
}

func (ScriptMentions) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	scan := func(dir string, exts map[string]bool, mod string) {
		scriptFiles(in, dir, exts, func(file string, lines []string) {
			type hit struct {
				lines []string
				items []domain.ResourceID
				first int
			}
			hits := map[domain.ChangeKind]*hit{}
			for i, raw := range lines {
				line := stripComment(raw)
				if strings.TrimSpace(line) == "" || out.Covered(origin(file, i+1)) {
					continue
				}
				// Keywords count in code only: ids such as
				// "mod:fishing_rod" in an item list say nothing.
				kind, ok := kindOfText(reStrings.ReplaceAllString(line, `""`))
				if !ok {
					continue
				}
				h := hits[kind]
				if h == nil {
					h = &hit{first: i + 1}
					hits[kind] = h
				}
				h.lines = append(h.lines, itoaLine(i+1))
				h.items = append(h.items, k.itemsOf(idsIn(line))...)
			}
			for kind, h := range hits {
				target := domain.ChangeTarget{Kind: kind}
				if kind == domain.ChangeLootType {
					target.ID = domain.ResourceID{Namespace: "loot", Path: "all"}
				}
				out.Add(domain.Change{Target: target, Effect: domain.EffectUnknown, Certainty: domain.Possibly, Mod: mod,
					Origin: origin(file, h.first), By: ScriptMentionsID, Items: h.items,
					Detail: "el script menciona " + kindWords[kind] + " (líneas " + strings.Join(limitStrings(h.lines, 12), ", ") + ")"})
			}
		})
	}
	scan("kubejs/server_scripts", jsExt, "KubeJS")
	scan("kubejs/startup_scripts", jsExt, "KubeJS")
	scan("scripts", map[string]bool{".zs": true}, "CraftTweaker")
	return nil
}

var kindWords = map[domain.ChangeKind]string{
	domain.ChangeLootType: "loot", domain.ChangeDrops: "drops de criaturas", domain.ChangeTrades: "tradeos",
	domain.ChangeFishing: "pesca", domain.ChangeBarter: "trueque con piglins",
}

// Keywords of each source kind, in priority order.
var kindPatterns = []struct {
	kind domain.ChangeKind
	re   *regexp.Regexp
}{
	{domain.ChangeBarter, regexp.MustCompile(`(?i)barter|piglin_barter`)},
	{domain.ChangeTrades, regexp.MustCompile(`(?i)trade|trading|wandering|wanderer|merchant|villager_?offer`)},
	{domain.ChangeFishing, regexp.MustCompile(`(?i)fishing|\bfish(ed)?\b|angl(er|ing)`)},
	{domain.ChangeDrops, regexp.MustCompile(`(?i)drops?(chance|rate|count)?\b|livingdrops|mob_?drop`)},
	{domain.ChangeLootType, regexp.MustCompile(`(?i)loot|treasure`)},
}

func kindOfText(s string) (domain.ChangeKind, bool) {
	for _, p := range kindPatterns {
		if p.re.MatchString(s) {
			return p.kind, true
		}
	}
	return "", false
}

var reStrings = regexp.MustCompile(`"[^"]*"|'[^']*'|` + "`[^`]*`")

func itoaLine(n int) string { return origin("", n)[1:] }

func limitStrings(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return append(s[:n:n], "…")
}
