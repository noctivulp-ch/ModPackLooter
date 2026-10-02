package loot

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

type memSource struct {
	tables map[string]string
	tags   map[string][]string
}

func (m memSource) LootTableJSON(id domain.ResourceID) ([]byte, bool, error) {
	s, ok := m.tables[id.String()]
	return []byte(s), ok, nil
}

func (m memSource) ItemTag(id domain.ResourceID) []domain.ResourceID {
	var out []domain.ResourceID
	for _, s := range m.tags[id.String()] {
		out = append(out, domain.MustParseResourceID(s))
	}
	return out
}

func drop(t *testing.T, tab *Table, item string) Drop {
	t.Helper()
	for _, d := range tab.Drops {
		if d.Item.String() == item {
			return d
		}
	}
	t.Fatalf("%s no contiene %s: %+v", tab.ID, item, tab.Drops)
	return Drop{}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func resolve(t *testing.T, src Source, id string) *Table {
	t.Helper()
	tab, ok, err := NewResolver(src).Resolve(domain.MustParseResourceID(id))
	if err != nil || !ok {
		t.Fatalf("Resolve(%s): ok=%v err=%v", id, ok, err)
	}
	return tab
}

func TestChanceWithUniformRolls(t *testing.T) {
	src := memSource{tables: map[string]string{"test:chest": `{
	  "type": "minecraft:chest",
	  "pools": [{
	    "rolls": {"type": "minecraft:uniform", "min": 2, "max": 4},
	    "entries": [
	      {"type": "minecraft:item", "name": "minecraft:diamond", "weight": 1,
	       "functions": [{"function": "minecraft:set_count", "count": {"min": 1, "max": 3}}]},
	      {"type": "minecraft:item", "name": "minecraft:bone", "weight": 3}
	    ]
	  }]
	}`}}
	tab := resolve(t, src, "test:chest")
	d := drop(t, tab, "minecraft:diamond")
	want := (3 - math.Pow(.75, 2) - math.Pow(.75, 3) - math.Pow(.75, 4)) / 3
	if !near(d.Chance, want) {
		t.Errorf("probabilidad del diamante = %v, se esperaba %v", d.Chance, want)
	}
	if d.CountMin != 1 || d.CountMax != 3 || !near(d.Expected, 3*0.25*2) {
		t.Errorf("cantidades = %v–%v, esperado %v", d.CountMin, d.CountMax, d.Expected)
	}
	if tab.Approximate || tab.Type != "chest" {
		t.Errorf("tabla = %+v", tab)
	}
	if tab.Drops[0].Item.String() != "minecraft:bone" {
		t.Error("las drops deben ordenarse por probabilidad descendente")
	}
}

func TestNestedTablesTagsAndConditions(t *testing.T) {
	src := memSource{
		tables: map[string]string{
			"test:outer": `{"pools": [
			  {"rolls": 1, "entries": [{"type": "loot_table", "name": "test:inner"}]},
			  {"rolls": 1, "conditions": [{"condition": "minecraft:random_chance", "chance": 0.5}],
			   "entries": [{"type": "minecraft:tag", "name": "test:gems", "expand": true}]},
			  {"rolls": 1, "entries": [{"type": "minecraft:item", "name": "minecraft:book",
			    "functions": [{"function": "minecraft:enchant_randomly"}]}, {"type": "minecraft:empty"}]}
			]}`,
			"test:inner": `{"pools": [{"rolls": 1, "entries": [
			  {"type": "minecraft:item", "name": "minecraft:emerald"},
			  {"type": "minecraft:item", "name": "minecraft:stick"}]}]}`,
		},
		tags: map[string][]string{"test:gems": {"minecraft:ruby", "minecraft:emerald"}},
	}
	tab := resolve(t, src, "test:outer")
	// emerald: 0.5 from inner table, plus 0.5*0.5 from the gem pool.
	if d := drop(t, tab, "minecraft:emerald"); !near(d.Chance, 1-0.5*0.75) {
		t.Errorf("esmeralda = %v", d.Chance)
	}
	if d := drop(t, tab, "minecraft:ruby"); !near(d.Chance, 0.25) {
		t.Errorf("rubí = %v", d.Chance)
	}
	// A book enchanted by a loot function is an enchanted book.
	book := drop(t, tab, "minecraft:enchanted_book")
	if !near(book.Chance, 0.5) || len(book.Notes) != 1 || book.Notes[0] != "enchant_randomly" {
		t.Errorf("libro = %+v", book)
	}
	if len(tab.Nested) != 1 || tab.Nested[0].String() != "test:inner" {
		t.Errorf("tablas anidadas = %v", tab.Nested)
	}
}

func TestMissingAndRecursiveTablesAreApproximate(t *testing.T) {
	src := memSource{tables: map[string]string{
		"test:loop":    `{"pools": [{"rolls": 1, "entries": [{"type": "loot_table", "name": "test:loop"}, {"type": "item", "name": "minecraft:dirt"}]}]}`,
		"test:missing": `{"pools": [{"rolls": 1, "entries": [{"type": "loot_table", "name": "test:nope"}]}]}`,
	}}
	if tab := resolve(t, src, "test:loop"); !tab.Approximate || !near(drop(t, tab, "minecraft:dirt").Chance, 0.5) {
		t.Errorf("tabla recursiva = %+v", tab)
	}
	if tab := resolve(t, src, "test:missing"); !tab.Approximate || len(tab.Missing) != 1 {
		t.Errorf("tabla con referencia rota = %+v", tab)
	}
	if _, ok, _ := NewResolver(src).Resolve(domain.MustParseResourceID("test:none")); ok {
		t.Error("una tabla inexistente debe devolver ok=false")
	}
}

func TestBinomialRolls(t *testing.T) {
	n := parseNumber([]byte(`{"type":"minecraft:binomial","n":2,"p":0.5}`), 0)
	if !near(n.Mean, 1) || !near(n.dist[0], .25) || !near(n.dist[1], .5) || !near(n.dist[2], .25) {
		t.Errorf("binomial = %+v", n)
	}
}

// TestResolveRealVanillaTables resolves every vanilla table when
// MPL_MCMETA_DATA points at a misode/mcmeta data checkout.
func TestResolveRealVanillaTables(t *testing.T) {
	root := os.Getenv("MPL_MCMETA_DATA")
	if root == "" {
		t.Skip("MPL_MCMETA_DATA no definido")
	}
	base := filepath.Join(root, "data", "minecraft", "loot_tables")
	src := memSource{tables: map[string]string{}, tags: map[string][]string{}}
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || filepath.Ext(path) != ".json" {
			return err
		}
		rel, _ := filepath.Rel(base, path)
		data, err := os.ReadFile(path)
		src.tables["minecraft:"+filepath.ToSlash(rel[:len(rel)-5])] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver(src)
	for id := range src.tables {
		if _, _, err := r.Resolve(domain.MustParseResourceID(id)); err != nil {
			t.Error(err)
		}
	}
	tab, _, _ := r.Resolve(domain.MustParseResourceID("minecraft:chests/desert_pyramid"))
	d := drop(t, tab, "minecraft:diamond")
	t.Logf("%d tablas; pirámide del desierto: diamante %.1f%% por cofre (%v–%v)", len(src.tables), d.Chance*100, d.CountMin, d.CountMax)
}
