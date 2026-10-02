package changes

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// TableOverridesID is the ID of the TableOverrides detector.
const TableOverridesID = "table-overrides"

// TableOverrides reports loot tables that a mod or datapack replaces (same
// id in a later pack), with what changed: items added or removed and
// functions changed (counts, enchantment level, treasure enchantments…).
// It works for any mod: it only compares data files.
type TableOverrides struct{}

func (TableOverrides) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: TableOverridesID, Phase: discovery.PhaseSpecific, Priority: 100}
}

func (TableOverrides) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	for _, id := range in.Resources.IDs(resources.TypeLootTable) {
		provs := in.Resources.Providers(resources.TypeLootTable, id)
		if len(provs) < 2 {
			continue
		}
		base, eff := provs[0], provs[len(provs)-1]
		a, errA := base.Read()
		b, errB := eff.Read()
		if errA != nil || errB != nil {
			continue
		}
		before, after := summarize(a), summarize(b)
		mod := packName(in, eff)
		prefix := fmt.Sprintf("Reemplaza la tabla de %s", packName(in, base))
		org := eff.Pack + "!/" + id.String()
		added, removed, changed := diffSummaries(before, after)
		if len(added)+len(removed)+len(changed) == 0 {
			if before.signature() != after.signature() {
				out.Add(domain.Change{Target: tableTarget(id), Effect: domain.EffectChange, Certainty: domain.Certainly,
					Mod: mod, Origin: org, By: TableOverridesID, Detail: prefix + ": cambian las tiradas o las condiciones"})
			}
			continue
		}
		if len(removed) > 0 {
			out.Add(domain.Change{Target: tableTarget(id), Effect: domain.EffectRemove, Items: ids(removed), Certainty: domain.Certainly,
				Mod: mod, Origin: org, By: TableOverridesID, Detail: prefix + ": quita " + strings.Join(removed, ", ")})
		}
		if len(added) > 0 {
			out.Add(domain.Change{Target: tableTarget(id), Effect: domain.EffectAdd, Items: ids(added), Certainty: domain.Certainly,
				Mod: mod, Origin: org, By: TableOverridesID, Detail: prefix + ": añade " + strings.Join(added, ", ")})
		}
		for _, c := range changed {
			out.Add(domain.Change{Target: tableTarget(id), Effect: domain.EffectChange, Items: ids([]string{c.item}), Certainty: domain.Certainly,
				Mod: mod, Origin: org, By: TableOverridesID, Detail: prefix + ": " + c.text})
		}
	}
	return nil
}

func ids(list []string) []domain.ResourceID {
	var out []domain.ResourceID
	for _, s := range list {
		if id, err := domain.ParseResourceID(s); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// summary is what a loot table gives: entry name → function descriptions.
type summary struct {
	entries map[string][]string // "ns:item", "#tag", "@table" → sorted functions
	pools   int
}

func (s summary) signature() string {
	keys := make([]string, 0, len(s.entries))
	for k, v := range s.entries {
		keys = append(keys, k+"="+strings.Join(v, ","))
	}
	sort.Strings(keys)
	return fmt.Sprint(s.pools, keys)
}

type rawEntry struct {
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Value     string            `json:"value"`
	Functions []json.RawMessage `json:"functions"`
	Children  []rawEntry        `json:"children"`
}

func summarize(raw []byte) summary {
	var t struct {
		Functions []json.RawMessage `json:"functions"`
		Pools     []struct {
			Functions []json.RawMessage `json:"functions"`
			Entries   []rawEntry        `json:"entries"`
		} `json:"pools"`
	}
	s := summary{entries: map[string][]string{}}
	if json.Unmarshal(raw, &t) != nil {
		return s
	}
	s.pools = len(t.Pools)
	var walk func(e rawEntry, inherited []string)
	walk = func(e rawEntry, inherited []string) {
		fns := append(append([]string(nil), inherited...), describeFunctions(e.Functions)...)
		name := e.Name
		if name == "" {
			name = e.Value
		}
		typ := strings.TrimPrefix(e.Type, "minecraft:")
		switch typ {
		case "item":
			if !strings.Contains(name, ":") {
				name = "minecraft:" + name
			}
			s.entries[name] = mergeFns(s.entries[name], fns)
		case "tag":
			s.entries["#"+name] = mergeFns(s.entries["#"+name], fns)
		case "loot_table":
			s.entries["@"+name] = mergeFns(s.entries["@"+name], fns)
		}
		for _, c := range e.Children {
			walk(c, fns)
		}
	}
	tableFns := describeFunctions(t.Functions)
	for _, p := range t.Pools {
		poolFns := append(append([]string(nil), tableFns...), describeFunctions(p.Functions)...)
		for _, e := range p.Entries {
			walk(e, poolFns)
		}
	}
	return s
}

func mergeFns(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range append(a, b...) {
		set[x] = true
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// describeFunctions summarises loot functions in Spanish.
func describeFunctions(raw []json.RawMessage) []string {
	var out []string
	for _, r := range raw {
		var f map[string]json.RawMessage
		if json.Unmarshal(r, &f) != nil {
			continue
		}
		var name string
		_ = json.Unmarshal(f["function"], &name)
		name = strings.TrimPrefix(name, "minecraft:")
		switch name {
		case "enchant_with_levels":
			var treasure bool
			_ = json.Unmarshal(f["treasure"], &treasure)
			t := "sin tesoro"
			if treasure {
				t = "con tesoro"
			}
			out = append(out, "encantado nivel "+number(f["levels"])+" "+t)
		case "set_count":
			out = append(out, "cantidad "+number(f["count"]))
		case "enchant_randomly":
			out = append(out, "encantado al azar")
		case "set_damage":
			out = append(out, "dañado")
		case "":
		default:
			out = append(out, name)
		}
	}
	return out
}

// number formats a number provider: 3, 1–4, ≈.
func number(raw json.RawMessage) string {
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return fmt.Sprintf("%g", f)
	}
	var o struct {
		Min json.RawMessage `json:"min"`
		Max json.RawMessage `json:"max"`
		N   json.RawMessage `json:"n"`
	}
	if json.Unmarshal(raw, &o) == nil && o.Min != nil && o.Max != nil {
		return number(o.Min) + "–" + number(o.Max)
	}
	if o.N != nil {
		return "hasta " + number(o.N)
	}
	return "variable"
}

type changedEntry struct {
	item string
	text string
}

func diffSummaries(a, b summary) (added, removed []string, changed []changedEntry) {
	for k, fa := range a.entries {
		fb, ok := b.entries[k]
		if !ok {
			removed = append(removed, k)
			continue
		}
		if strings.Join(fa, ",") != strings.Join(fb, ",") {
			changed = append(changed, changedEntry{item: k, text: fmt.Sprintf("%s: %s → %s", k, orNone(fa), orNone(fb))})
		}
	}
	for k := range b.entries {
		if _, ok := a.entries[k]; !ok {
			added = append(added, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Slice(changed, func(i, j int) bool { return changed[i].item < changed[j].item })
	return added, removed, changed
}

func orNone(fns []string) string {
	if len(fns) == 0 {
		return "sin cambios especiales"
	}
	return strings.Join(fns, ", ")
}

// GlobalLootModifiersID is the ID of the GlobalLootModifiers detector.
const GlobalLootModifiersID = "global-loot-modifiers"

// GlobalLootModifiers reads Forge/NeoForge Global Loot Modifiers of any mod:
// the active list (global_loot_modifiers.json of every pack), the target
// tables (loot_table_id conditions), the chance, the items and loot tables
// they mention. The type is mod specific, so the effect is guessed from its
// name ("remove", "replace", else "add").
type GlobalLootModifiers struct{}

func (GlobalLootModifiers) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: GlobalLootModifiersID, Phase: discovery.PhaseSpecific, Priority: 90}
}

func (GlobalLootModifiers) Detect(_ context.Context, in discovery.Input, out *discovery.Changes) error {
	k := newKnown(in)
	var active []domain.ResourceID
	seen := map[domain.ResourceID]bool{}
	for _, list := range []string{"forge:global_loot_modifiers", "neoforge:global_loot_modifiers"} {
		for _, p := range in.Resources.Providers(resources.TypeLootModifier, domain.MustParseResourceID(list)) {
			data, err := p.Read()
			if err != nil {
				continue
			}
			var doc struct {
				Replace bool     `json:"replace"`
				Entries []string `json:"entries"`
			}
			if json.Unmarshal(data, &doc) != nil {
				continue
			}
			if doc.Replace {
				active, seen = nil, map[domain.ResourceID]bool{}
			}
			for _, e := range doc.Entries {
				if id, err := domain.ParseResourceID(e); err == nil && !seen[id] {
					seen[id] = true
					active = append(active, id)
				}
			}
		}
	}
	for _, id := range active {
		provs := in.Resources.Providers(resources.TypeLootModifier, id)
		if len(provs) == 0 {
			continue
		}
		eff := provs[len(provs)-1]
		data, err := eff.Read()
		if err != nil {
			continue
		}
		var doc map[string]json.RawMessage
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		var typ string
		_ = json.Unmarshal(doc["type"], &typ)
		targets, condChance, notes := lootConditions(doc["conditions"])
		chance := condChance
		if raw, ok := doc["chance"]; ok {
			var c float64
			if json.Unmarshal(raw, &c) == nil {
				if chance > 0 {
					chance *= c
				} else {
					chance = c
				}
				if c == 0 {
					chance = -1 // explicit zero
				}
			}
		}
		rest := map[string]json.RawMessage{}
		for k, v := range doc {
			if k != "type" && k != "conditions" {
				rest[k] = v
			}
		}
		var strs []string
		collectStrings(rest, &strs)
		var items []domain.ResourceID
		var table domain.ResourceID
		for _, s := range strs {
			idv, err := domain.ParseResourceID(strings.TrimPrefix(s, "#"))
			if err != nil {
				continue
			}
			switch {
			case k.isItem(idv):
				items = append(items, idv)
			case table.IsZero() && k.isTable(idv):
				table = idv
			}
		}
		effect := domain.EffectUnknown
		low := strings.ToLower(typ)
		switch {
		case chance < 0:
			effect = domain.EffectDisable
		case strings.Contains(low, "remove"):
			effect = domain.EffectRemove
		case strings.Contains(low, "replace"):
			effect = domain.EffectReplace
		case len(items) > 0 || !table.IsZero():
			effect = domain.EffectAdd
		}
		certainty := domain.Certainly
		if effect == domain.EffectUnknown {
			certainty = domain.Possibly
		}
		detail := "Modificador de loot " + id.String() + " (" + typ + ")"
		if effect == domain.EffectDisable {
			detail += ": probabilidad 0, no tiene efecto"
		}
		if len(provs) > 1 {
			detail += "; reemplazado por " + packName(in, eff) + " sobre el de " + packName(in, provs[0])
		}
		if len(notes) > 0 {
			detail += "; condiciones: " + strings.Join(notes, ", ")
		}
		c := domain.Change{Effect: effect, Items: items, Table: table, Certainty: certainty,
			Mod: packName(in, eff), Origin: "loot_modifier:" + id.String(), By: GlobalLootModifiersID, Detail: detail}
		if chance > 0 {
			c.Chance = chance
		}
		if len(targets) == 0 {
			c.Target = domain.ChangeTarget{Kind: domain.ChangeLootType, ID: domain.ResourceID{Namespace: "loot", Path: "all"}}
			c.Certainty = domain.Possibly
			out.Add(c)
			continue
		}
		for _, t := range targets {
			c.Target = tableTarget(t)
			out.Add(c)
		}
	}
	return nil
}

// lootConditions reads loot_table_id targets (also inside any_of /
// alternative), random_chance and other condition names.
func lootConditions(raw json.RawMessage) (targets []domain.ResourceID, chance float64, notes []string) {
	var conds []map[string]json.RawMessage
	if json.Unmarshal(raw, &conds) != nil {
		return nil, 0, nil
	}
	chance = 0
	for _, c := range conds {
		var name string
		_ = json.Unmarshal(c["condition"], &name)
		_, short, _ := strings.Cut(name, ":")
		switch short {
		case "loot_table_id":
			var s string
			if json.Unmarshal(c["loot_table_id"], &s) == nil {
				if id, err := domain.ParseResourceID(s); err == nil {
					targets = append(targets, id)
				}
			}
		case "any_of", "alternative":
			key := "terms"
			if _, ok := c[key]; !ok {
				key = "conditions"
			}
			t, _, n := lootConditions(c[key])
			targets = append(targets, t...)
			notes = append(notes, n...)
		case "random_chance":
			var p float64
			if json.Unmarshal(c["chance"], &p) == nil {
				chance = p
			}
		case "killed_by_player":
			notes = append(notes, "solo si lo mata un jugador")
		case "":
		default:
			notes = append(notes, short)
		}
	}
	return targets, chance, notes
}

func collectStrings(v any, out *[]string) {
	switch x := v.(type) {
	case map[string]json.RawMessage:
		for _, r := range x {
			collectStrings(r, out)
		}
	case json.RawMessage:
		var anyv any
		if json.Unmarshal(x, &anyv) == nil {
			collectStrings(anyv, out)
		}
	case map[string]any:
		for _, r := range x {
			collectStrings(r, out)
		}
	case []any:
		for _, r := range x {
			collectStrings(r, out)
		}
	case string:
		*out = append(*out, x)
	}
}

// packName names who ships a resource: the mod of a jar, "Minecraft" for
// vanilla data, or the datapack name.
func packName(in discovery.Input, p discovery.Provider) string {
	switch p.Kind {
	case resources.KindVanilla:
		return "Minecraft"
	case resources.KindDatapack:
		return "el datapack " + p.Pack
	case resources.KindScript:
		return "KubeJS (" + p.Pack + ")"
	}
	return in.Files.ModName(p.Pack)
}
