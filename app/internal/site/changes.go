package site

import (
	"regexp"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
)

// Change is a modification of a source by a mod, ready to read.
type Change struct {
	Kind      string
	Target    string // what it touches, in words
	Table     *Table // target table page, if any
	Effect    string
	Items     []*Item
	With      []*Item
	Inject    *Table // loot table it adds
	Chance    float64
	Sure      bool
	Mod       string
	Origin    string
	Detail    string
	Disables  bool
	targetKey string
	targetID  domain.ResourceID
}

// ChangeGroup is a set of changes from the same origin (one Global Loot
// Modifier can touch hundreds of tables).
type ChangeGroup struct {
	First   *Change
	Targets []*Change
}

// ChangeSection groups changes by kind of source.
type ChangeSection struct {
	Title  string
	Note   string
	Groups []*ChangeGroup
	Sure   int
	Maybe  int
}

var effectNames = map[domain.Effect]string{
	domain.EffectAdd: "Añade", domain.EffectRemove: "Quita", domain.EffectReplace: "Reemplaza",
	domain.EffectChange: "Modifica", domain.EffectDisable: "Desactivado", domain.EffectUnknown: "Puede cambiar",
}

var changeKinds = []struct {
	kind  domain.ChangeKind
	title string
	note  string
}{
	{domain.ChangeTable, "Tablas de loot concretas", "Cambios a una tabla con nombre: reemplazos, modificadores de loot, scripts."},
	{domain.ChangeLootType, "Grupos de tablas o todo el loot", "Cambios a todas las tablas de un tipo (cofres, bloques…) o a cualquier tabla."},
	{domain.ChangeDrops, "Drops de criaturas", "Lo que sueltan los mobs al morir."},
	{domain.ChangeTrades, "Tradeos", "Aldeanos y comerciante errante."},
	{domain.ChangeFishing, "Pesca", ""},
	{domain.ChangeBarter, "Trueque con piglins", ""},
}

var lootTypeNames = map[string]string{"chest": "todos los cofres", "block": "todos los bloques", "gameplay": "loot de jugabilidad", "archaeology": "arqueología", "all": "cualquier tabla de loot"}

var reAnyID = regexp.MustCompile(`#?[a-z0-9_.-]+:[a-z0-9_./-]+`)

func buildChanges(res *analysis.Result, m *Model, namer *names.Namer, itemOf func(domain.ResourceID) *Item) {
	if len(res.Changes) == 0 {
		return
	}
	tables := map[domain.ResourceID]*Table{}
	for _, t := range m.Tables {
		tables[t.ID] = t
	}
	items := map[domain.ResourceID]*Item{}
	for _, it := range m.Items {
		items[it.ID] = it
	}
	item := func(id domain.ResourceID) *Item {
		if it, ok := items[id]; ok {
			return it
		}
		return itemOf(id)
	}
	// Ids in details become names when they are items.
	human := func(s string) string {
		return reAnyID.ReplaceAllStringFunc(s, func(x string) string {
			if strings.HasPrefix(x, "#") {
				return x
			}
			id, err := domain.ParseResourceID(x)
			if err != nil {
				return x
			}
			if it, ok := items[id]; ok {
				return it.Name
			}
			return x
		})
	}
	byType := map[string][]*Table{}
	for _, t := range m.Tables {
		key := map[domain.SourceKind]string{domain.KindContainer: "chest", domain.KindArchaeology: "archaeology", domain.KindGameplay: "gameplay"}[t.Kind]
		if strings.Contains(t.ID.Path, "blocks/") {
			key = "block"
		}
		if key != "" {
			byType[key] = append(byType[key], t)
		}
	}

	sections := map[domain.ChangeKind]*ChangeSection{}
	groups := map[string]*ChangeGroup{}
	for _, c := range res.Changes {
		v := &Change{Effect: effectNames[c.Effect], Chance: c.Chance, Sure: c.Certainty == domain.Certainly, Mod: c.Mod,
			Origin: c.Origin, Detail: human(c.Detail), Disables: c.Effect == domain.EffectDisable}
		for _, id := range c.Items {
			v.Items = append(v.Items, item(id))
		}
		for _, id := range c.With {
			v.With = append(v.With, item(id))
		}
		if !c.Table.IsZero() {
			v.Inject = tables[c.Table]
		}
		var attach []*Table
		switch c.Target.Kind {
		case domain.ChangeTable:
			v.Table = tables[c.Target.ID]
			v.Target = names.Humanize(c.Target.ID.Path)
			if v.Table != nil {
				v.Target = v.Table.Name
				attach = append(attach, v.Table)
			}
		case domain.ChangeDrops:
			if c.Target.ID.IsZero() {
				v.Target = "cualquier criatura"
			} else {
				v.Target = namer.Entity(c.Target.ID)
				id := domain.ResourceID{Namespace: c.Target.ID.Namespace, Path: "entities/" + c.Target.ID.Path}
				if t := tables[id]; t != nil {
					v.Table = t
					attach = append(attach, t)
				}
			}
		case domain.ChangeLootType:
			v.Target = lootTypeNames[c.Target.ID.Path]
			if v.Target == "" {
				v.Target = c.Target.ID.Path
			}
			if c.Target.ID.Path != "all" {
				attach = append(attach, byType[c.Target.ID.Path]...)
			}
		case domain.ChangeTrades:
			v.Target = "todos los tradeos"
			if !c.Target.ID.IsZero() {
				v.Target = namer.Asset("profession", c.Target.ID)
				if c.Target.ID.Path == "wandering_trader" {
					v.Target = namer.Entity(c.Target.ID)
				}
			}
		case domain.ChangeFishing:
			v.Target = "la pesca"
		case domain.ChangeBarter:
			v.Target = "el trueque con piglins"
		}
		v.targetKey = string(c.Target.Kind) + c.Target.ID.String()
		v.targetID = c.Target.ID
		if c.Target.Kind == domain.ChangeTrades || c.Target.Kind == domain.ChangeBarter {
			m.tradeChanges = append(m.tradeChanges, v)
		}
		if !v.Disables {
			for _, t := range attach {
				t.Changes = append(t.Changes, v)
			}
			// Items added or removed by a known change show it on their page.
			if c.Target.Kind == domain.ChangeTable || c.Target.Kind == domain.ChangeDrops {
				for _, it := range v.Items {
					it.Changes = append(it.Changes, v)
				}
			}
		}

		sec := sections[c.Target.Kind]
		if sec == nil {
			sec = &ChangeSection{}
			sections[c.Target.Kind] = sec
		}
		if v.Sure {
			sec.Sure++
		} else {
			sec.Maybe++
		}
		key := string(c.Target.Kind) + "|" + c.Origin + "|" + c.Detail + "|" + string(c.Effect)
		g := groups[key]
		if g == nil {
			g = &ChangeGroup{First: v}
			groups[key] = g
			sec.Groups = append(sec.Groups, g)
		}
		g.Targets = append(g.Targets, v)
	}
	for _, k := range changeKinds {
		sec := sections[k.kind]
		if sec == nil {
			continue
		}
		sec.Title, sec.Note = k.title, k.note
		sort.SliceStable(sec.Groups, func(i, j int) bool {
			a, b := sec.Groups[i].First, sec.Groups[j].First
			if a.Sure != b.Sure {
				return a.Sure
			}
			if a.Disables != b.Disables {
				return b.Disables
			}
			if !strings.EqualFold(a.Mod, b.Mod) {
				return strings.ToLower(a.Mod) < strings.ToLower(b.Mod)
			}
			return a.Origin < b.Origin
		})
		m.Changes = append(m.Changes, sec)
	}
	m.Sections = append(m.Sections, Section{Nav: "cambios", Label: "Cambios de mods", URL: "cambios/"})
}
