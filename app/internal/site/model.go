package site

import (
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
)

// Ref is a link to a page of the site. URL is relative to the site root.
type Ref struct {
	ID   domain.ResourceID
	Name string
	URL  string
}

// Mod groups everything a namespace contributes.
type Mod struct {
	Ref
	Version    string
	Items      []*Item
	Structures []*Owner
	Tables     []*Table
}

// Item is an item that can be obtained from at least one loot source.
type Item struct {
	Ref
	Mod     *Mod
	Sources []*ItemSource // best first
	Best    float64
	Biomes  []*Biome // biomes of the owners that can give the item
	// Unobtainable is true when every source is certainly disabled.
	Unobtainable bool
	// Fishing lists the fishing systems that can give the item.
	Fishing []*FishEntry
	// Changes are mods adding or removing the item somewhere.
	Changes []*Change
	// Trades sell or buy the item; NPCDrops are NPCs that drop it.
	Trades   []*TradeRef
	NPCDrops []*NPCDrop
	// Variant is set on variant pages (an enchanted book's enchantment…);
	// Base is then the plain item. Variants are the plain item's variants.
	Variant  domain.Variant
	Label    string // what makes the variant special, e.g. "Reparación"
	Base     *Item
	Variants []*Item
	// Aka are other names to search by (English name…).
	Aka []string
	// LCBuildings are Lost Cities buildings that can hold the item.
	LCBuildings []LCHaulRef
	// Ways summarise how to get it, most likely first; Top is the best.
	Ways []*Way
	Top  *WayPlace
}

// LCHaulRef links an item to a Lost Cities building that can hold it.
type LCHaulRef struct {
	Building *LCBuilding
	Haul     HaulItem
}

// ItemSource is one way to get an item.
type ItemSource struct {
	Table      *Table
	Kind       domain.SourceKind
	Owner      *Owner // nil when the table has no known owner
	Chance     float64
	Share      float64
	Effective  float64 // Chance weighted by Share, used to rank sources
	CountMin   float64
	CountMax   float64
	Confidence domain.Confidence
	Notes      []string
	Approx     bool
	Status     domain.Status
}

// Owner is a structure, feature or template that holds loot.
type Owner struct {
	Ref
	Kind       domain.OwnerKind
	Mod        *Mod
	Biomes     []*Biome
	BiomesNote string
	Uses       []*TableUse
	Template   bool
	Status     domain.Status
	// Haul is "¿Qué hay aquí?": every item of its containers.
	Haul []HaulItem
}

// TableUse is a loot table used by an owner.
type TableUse struct {
	Table      *Table
	Owner      *Owner
	Kind       domain.SourceKind
	Container  string
	Share      float64
	Confidence domain.Confidence
	Notes      []string
	Evidence   []string
	Status     domain.Status
}

// Table is a loot table page.
type Table struct {
	Ref
	Mod     *Mod
	Kind    domain.SourceKind
	Drops   []Drop
	Uses    []*TableUse
	Approx  bool
	Changes []*Change // modifications by mods, scripts or configs
}

// Drop is a row of a loot table.
type Drop struct {
	Item     *Item
	Chance   float64
	CountMin float64
	CountMax float64
	Notes    []string
	Approx   bool
}

// Biome lists the owners that can generate in it.
type Biome struct {
	Ref
	Mod    *Mod
	Owners []*Owner
	Top    []BiomeItem // best items obtainable in the biome
	Status domain.Status
}

// BiomeItem is an item and its best source within a biome.
type BiomeItem struct {
	Item   *Item
	Source *ItemSource
}

// PackInfo summarises the analysed modpack.
type PackInfo struct {
	MCVersion     string
	Loader        string
	VersionSource string
	Mods          int
	HasVanilla    bool
	Sources       int
	Tables        int
	BlockTables   int
}

// Model is everything the templates render.
type Model struct {
	Title       string
	Lang        string // names language, e.g. "es_ar"
	HTMLLang    string // BCP 47 tag of the page; the interface is in Spanish
	LangSource  string
	Metal       string
	AppVersion  string
	Pack        PackInfo
	Items       []*Item
	Owners      []*Owner
	Biomes      []*Biome
	Tables      []*Table
	Mods        []*Mod
	Unowned     map[domain.SourceKind][]*TableUse
	Diagnostics []domain.Diagnostic
	Plan        []string
	Enrichments []string
	Disablers   []string
	// Disabled lists the disabled targets for the about page.
	Disabled  []DisabledRow
	WorldName string
	// Sections are the tabs of mods present in the pack (Lost Cities…).
	Sections   []Section
	LostCities *LostCities
	Fishing    *Fishing
	Changes    []*ChangeSection
	Trades     *Trades

	tradeChanges []*Change
}

// DisabledRow is a disabled structure, biome or mob, for the about page.
type DisabledRow struct {
	Kind   domain.TargetKind
	Ref    *Ref
	ID     domain.ResourceID
	Status domain.Status
}

func idPath(id domain.ResourceID) string {
	clean := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == '/' {
				return r
			}
			return '_'
		}, strings.ToLower(s))
	}
	return clean(id.Namespace) + "/" + clean(id.Path)
}

// noteText translates loot function and condition names.
var noteText = map[string]string{
	"enchant_randomly":           "Encantado al azar",
	"enchant_with_levels":        "Encantado",
	"set_enchantments":           "Encantamientos fijos",
	"set_potion":                 "Poción",
	"set_stew_effect":            "Efecto de estofado",
	"exploration_map":            "Mapa de exploración",
	"set_instrument":             "Instrumento",
	"set_nbt":                    "Datos especiales",
	"set_components":             "Datos especiales",
	"set_custom_data":            "Datos especiales",
	"set_name":                   "Con nombre propio",
	"set_damage":                 "Dañado",
	"set_attributes":             "Atributos",
	"set_contents":               "Con contenido",
	"set_banner_pattern":         "Estandarte",
	"looting_enchant":            "Más con Botín",
	"enchanted_count_increase":   "Más con Botín",
	"furnace_smelt":              "Cocinado si arde",
	"killed_by_player":           "Solo si lo mata un jugador",
	"random_chance_with_looting": "Más con Botín",
}

// detailedNote explains loot functions that carry parameters.
func detailedNote(n string, namer *names.Namer) (string, bool) {
	parts := strings.Split(n, "|")
	switch parts[0] {
	case "enchant_with_levels":
		if len(parts) != 4 {
			return "", false
		}
		lv := parts[1]
		if parts[2] != parts[1] {
			lv = parts[1] + "–" + parts[2]
		}
		mending := enchantName(namer, "minecraft:mending")
		if parts[3] == "true" {
			return "Encantado (nivel " + lv + ", puede dar encantamientos de tesoro como " + mending + ")", true
		}
		return "Encantado (nivel " + lv + ", sin encantamientos de tesoro: no da " + mending + ", " + enchantName(namer, "minecraft:frost_walker") + " ni maldiciones)", true
	case "enchant_randomly":
		if len(parts) != 2 {
			return "", false
		}
		var list []string
		for _, e := range strings.Split(parts[1], ",") {
			list = append(list, enchantName(namer, e))
		}
		return "Encantado al azar con: " + strings.Join(list, ", "), true
	}
	return "", false
}

func enchantName(namer *names.Namer, id string) string {
	if rid, err := domain.ParseResourceID(strings.TrimPrefix(id, "#")); err == nil {
		if v, ok := namer.Text("enchantment." + rid.Namespace + "." + rid.Path); ok {
			return v
		}
		return names.Humanize(rid.Path)
	}
	return id
}

func translateNotes(in []string, namer *names.Namer) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range in {
		t, ok := noteText[n]
		if !ok {
			t, ok = detailedNote(n, namer)
		}
		if !ok {
			t = names.Humanize(n)
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// buildModel turns an analysis result into the site model.
func buildModel(res *analysis.Result, opts Options) *Model {
	ix := res.Resources.Index()
	var langs []map[string]string
	for _, code := range names.LangChain(opts.Lang, ix.Languages()) {
		langs = append(langs, ix.Lang(code, nil))
	}
	namer := names.New(langs...)

	m := &Model{
		Title: opts.Title, Metal: opts.Metal, AppVersion: opts.AppVersion,
		Lang: opts.Lang, HTMLLang: "es", LangSource: res.Modpack.LangSource,
		Unowned:     map[domain.SourceKind][]*TableUse{},
		Diagnostics: res.Diagnostics.Items(), Plan: res.Plan, Enrichments: res.Enrichments,
		Disablers: res.Disablers,
	}
	if res.Modpack.World != nil {
		m.WorldName = res.Modpack.World.Name
	}
	mp := res.Modpack
	m.Pack = PackInfo{
		MCVersion: mp.MCVersion.String(), Loader: string(mp.Loader), VersionSource: mp.VersionSource,
		Mods: len(mp.Mods), HasVanilla: mp.HasVanilla, Tables: len(res.Tables),
	}

	mods := map[string]*Mod{}
	modOf := func(ns string) *Mod {
		if md, ok := mods[ns]; ok {
			return md
		}
		id := domain.ResourceID{Namespace: ns, Path: ns}
		md := &Mod{Ref: Ref{ID: id, Name: names.Humanize(ns), URL: "mods/" + strings.SplitN(idPath(id), "/", 2)[0] + "/"}}
		if ns == "minecraft" {
			md.Name = "Minecraft"
		}
		for _, x := range mp.Mods {
			if x.ID == ns {
				if x.Name != "" {
					md.Name = x.Name
				}
				md.Version = x.Version
				break
			}
		}
		mods[ns] = md
		return md
	}

	items := map[itemKey]*Item{}
	english := englishNamer(res, opts)
	itemOf := func(id domain.ResourceID) *Item {
		if it, ok := items[itemKey{id: id}]; ok {
			return it
		}
		it := &Item{Ref: Ref{ID: id, Name: namer.Item(id), URL: "objetos/" + idPath(id) + "/"}, Mod: modOf(id.Namespace)}
		if english != nil {
			if en := english.Item(id); en != it.Name {
				it.Aka = append(it.Aka, en)
			}
		}
		items[itemKey{id: id}] = it
		return it
	}
	// variantOf returns the page of a useful variant of an item.
	variantOf := func(id domain.ResourceID, v domain.Variant) *Item {
		if v.IsZero() {
			return itemOf(id)
		}
		if it, ok := items[itemKey{id, v}]; ok {
			return it
		}
		base := itemOf(id)
		it := &Item{Ref: Ref{ID: id, URL: base.URL + variantSlug(v) + "/"}, Mod: base.Mod, Variant: v, Base: base}
		it.Name, it.Label = variantName(namer, base.Name, id, v)
		if english != nil {
			if en, _ := variantName(english, english.Item(id), id, v); en != it.Name {
				it.Aka = append(it.Aka, en)
			}
		}
		base.Variants = append(base.Variants, it)
		items[itemKey{id, v}] = it
		return it
	}

	tables := map[domain.ResourceID]*Table{}
	tableOf := func(id domain.ResourceID, kind domain.SourceKind) *Table {
		if t, ok := tables[id]; ok {
			return t
		}
		t := &Table{Ref: Ref{ID: id, Name: names.Humanize(id.Path), URL: "tablas/" + idPath(id) + "/"}, Mod: modOf(id.Namespace), Kind: kind}
		if kind == domain.KindEntity {
			t.Name = namer.Entity(entityOf(id))
		}
		if lt := res.Tables[id]; lt != nil {
			t.Approx = lt.Approximate
			for _, d := range lt.Drops {
				t.Drops = append(t.Drops, Drop{Item: variantOf(d.Item, d.Variant), Chance: d.Chance, CountMin: d.CountMin, CountMax: d.CountMax, Notes: translateNotes(d.Notes, namer), Approx: d.Approximate})
			}
		}
		tables[id] = t
		return t
	}

	// Owners: structures from data, owners declared by discoverers, templates.
	owners := map[domain.Owner]*Owner{}
	biomes := map[domain.ResourceID]*Biome{}
	biomeOf := func(id domain.ResourceID) *Biome {
		if b, ok := biomes[id]; ok {
			return b
		}
		b := &Biome{Ref: Ref{ID: id, Name: namer.Biome(id), URL: "biomas/" + idPath(id) + "/"}, Mod: modOf(id.Namespace)}
		b.Status = res.Status(domain.Target{Kind: domain.TargetBiome, ID: id})
		biomes[id] = b
		return b
	}
	structureBiomes := map[domain.ResourceID][]domain.ResourceID{}
	for _, s := range res.Structures {
		structureBiomes[s.ID] = s.Biomes
	}
	declared := map[domain.Owner]domain.OwnerInfo{}
	for _, o := range res.Owners {
		declared[o.Owner] = o
	}
	ownerOf := func(o domain.Owner) *Owner {
		if x, ok := owners[o]; ok {
			return x
		}
		x := &Owner{Kind: o.Kind, Mod: modOf(o.ID.Namespace)}
		x.Ref = Ref{ID: o.ID, Name: namer.Structure(o.ID), URL: "estructuras/" + idPath(o.ID) + "/"}
		var biomeIDs []domain.ResourceID
		if info, ok := declared[o]; ok {
			if info.Name != "" {
				x.Name = info.Name
			}
			biomeIDs, x.BiomesNote = info.Biomes, info.BiomesNote
		} else {
			biomeIDs = structureBiomes[o.ID]
		}
		x.Status = res.Status(domain.Target{Kind: domain.TargetStructure, ID: o.ID})
		if o.Kind == domain.OwnerTemplate {
			x.Template = true
			x.Name = "Plantilla " + names.Humanize(o.ID.Path)
			x.URL = "estructuras/plantillas/" + idPath(o.ID) + "/"
			x.BiomesNote = "plantilla colocada por código; estructura y biomas desconocidos"
		}
		for _, b := range biomeIDs {
			bm := biomeOf(b)
			x.Biomes = append(x.Biomes, bm)
			bm.Owners = append(bm.Owners, x)
		}
		owners[o] = x
		return x
	}

	for _, s := range res.Sources {
		if s.Kind == domain.KindBlock {
			m.Pack.BlockTables++
			continue
		}
		m.Pack.Sources++
		t := tableOf(s.LootTable, s.Kind)
		use := &TableUse{Table: t, Kind: s.Kind, Container: s.Container, Share: s.Share, Confidence: s.Confidence}
		for _, n := range s.Notes {
			use.Notes = append(use.Notes, n.Text)
		}
		for _, e := range s.Evidence {
			use.Evidence = append(use.Evidence, e.Detail)
		}
		t.Uses = append(t.Uses, use)
		var owner *Owner
		if s.Owner.Kind != domain.OwnerNone {
			owner = ownerOf(s.Owner)
			use.Owner = owner
			use.Status = owner.Status
			owner.Uses = append(owner.Uses, use)
		} else {
			if s.Kind == domain.KindEntity {
				use.Status = res.Status(domain.Target{Kind: domain.TargetEntity, ID: entityOf(s.LootTable)})
			}
			m.Unowned[s.Kind] = append(m.Unowned[s.Kind], use)
		}
		for _, d := range t.Drops {
			eff := d.Chance
			if s.Share > 0 {
				eff *= s.Share
			}
			if use.Status.Disabled() {
				eff = 0
			}
			src := &ItemSource{
				Table: t, Kind: s.Kind, Owner: owner, Chance: d.Chance, Share: s.Share, Effective: eff,
				CountMin: d.CountMin, CountMax: d.CountMax, Confidence: s.Confidence,
				Notes: append(append([]string(nil), d.Notes...), use.Notes...), Approx: d.Approx,
				Status: use.Status,
			}
			d.Item.Sources = append(d.Item.Sources, src)
			if eff > d.Item.Best {
				d.Item.Best = eff
			}
		}
	}

	// Collect and sort everything for deterministic output. listItem adds an
	// item (and the plain item of a variant) to the lists once.
	listed := map[*Item]bool{}
	var listItem func(it *Item)
	listItem = func(it *Item) {
		if listed[it] {
			return
		}
		listed[it] = true
		m.Items = append(m.Items, it)
		it.Mod.Items = append(it.Mod.Items, it)
		if it.Base != nil {
			listItem(it.Base)
		}
	}
	for _, it := range items {
		if len(it.Sources) == 0 {
			continue
		}
		it.Unobtainable = true
		for _, src := range it.Sources {
			if !src.Status.Disabled() {
				it.Unobtainable = false
				break
			}
		}
		sort.SliceStable(it.Sources, func(i, j int) bool {
			a, b := it.Sources[i], it.Sources[j]
			if a.Status.Disabled() != b.Status.Disabled() {
				return b.Status.Disabled()
			}
			if a.Effective != b.Effective {
				return a.Effective > b.Effective
			}
			return a.Table.ID.String() < b.Table.ID.String()
		})
		seen := map[*Biome]bool{}
		for _, src := range it.Sources {
			if src.Owner == nil {
				continue
			}
			for _, b := range src.Owner.Biomes {
				if !seen[b] {
					seen[b] = true
					it.Biomes = append(it.Biomes, b)
				}
			}
		}
		sortRefs(it.Biomes, func(b *Biome) Ref { return b.Ref })
		listItem(it)
	}
	sortRefs(m.Items, func(i *Item) Ref { return i.Ref })
	for _, o := range owners {
		h := newHaul()
		for _, u := range o.Uses {
			h.add(u.Table, u.Share, 0)
		}
		o.Haul = h.list()
		m.Owners = append(m.Owners, o)
		o.Mod.Structures = append(o.Mod.Structures, o)
		sortUses(o.Uses)
	}
	sortRefs(m.Owners, func(o *Owner) Ref { return o.Ref })
	for _, t := range tables {
		m.Tables = append(m.Tables, t)
		t.Mod.Tables = append(t.Mod.Tables, t)
	}
	sortRefs(m.Tables, func(t *Table) Ref { return t.Ref })
	for _, b := range biomes {
		sortRefs(b.Owners, func(o *Owner) Ref { return o.Ref })
		b.Top = topItems(b.Owners, 24)
		m.Biomes = append(m.Biomes, b)
	}
	sortRefs(m.Biomes, func(b *Biome) Ref { return b.Ref })
	for _, md := range mods {
		if len(md.Items)+len(md.Structures)+len(md.Tables) == 0 {
			continue
		}
		sortRefs(md.Items, func(i *Item) Ref { return i.Ref })
		sortRefs(md.Structures, func(o *Owner) Ref { return o.Ref })
		sortRefs(md.Tables, func(t *Table) Ref { return t.Ref })
		m.Mods = append(m.Mods, md)
	}
	sortRefs(m.Mods, func(md *Mod) Ref { return md.Ref })
	for k := range m.Unowned {
		sort.Slice(m.Unowned[k], func(i, j int) bool { return m.Unowned[k][i].Table.ID.String() < m.Unowned[k][j].Table.ID.String() })
	}
	for t, st := range res.Statuses {
		if st.Certainty == 0 {
			continue
		}
		row := DisabledRow{Kind: t.Kind, ID: t.ID, Status: st}
		switch t.Kind {
		case domain.TargetStructure:
			if o, ok := owners[domain.Owner{Kind: domain.OwnerStructure, ID: t.ID}]; ok {
				row.Ref = &o.Ref
			}
		case domain.TargetBiome:
			if b, ok := biomes[t.ID]; ok {
				row.Ref = &b.Ref
			}
		case domain.TargetEntity:
			if tb, ok := tables[domain.ResourceID{Namespace: t.ID.Namespace, Path: "entities/" + t.ID.Path}]; ok {
				row.Ref = &tb.Ref
			}
		}
		m.Disabled = append(m.Disabled, row)
	}
	sort.Slice(m.Disabled, func(i, j int) bool {
		a, b := m.Disabled[i], m.Disabled[j]
		if a.Status.Certainty != b.Status.Certainty {
			return a.Status.Certainty > b.Status.Certainty
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID.String() < b.ID.String()
	})
	buildLostCities(res, m, namer)
	ensureVariant := func(id domain.ResourceID, v domain.Variant) *Item {
		// Fished or traded items may have no loot source: list them too.
		it := variantOf(id, v)
		listItem(it)
		for _, md := range []*Mod{it.Mod} {
			known := false
			for _, x := range m.Mods {
				known = known || x == md
			}
			if !known {
				m.Mods = append(m.Mods, md)
			}
		}
		return it
	}
	ensureItem := func(id domain.ResourceID) *Item { return ensureVariant(id, domain.Variant{}) }
	buildFishing(res, m, namer, ensureItem)
	buildChanges(res, m, namer, ensureItem)
	buildTrades(res, m, namer, ensureVariant)
	for _, it := range m.Items {
		sortRefs(it.Variants, func(v *Item) Ref { return v.Ref })
	}
	if m.LostCities != nil {
		for _, b := range m.LostCities.Buildings {
			for _, h := range b.Haul {
				h.Item.LCBuildings = append(h.Item.LCBuildings, LCHaulRef{Building: b, Haul: h})
			}
		}
	}
	for _, it := range m.Items {
		buildWays(it, namer)
	}
	sortRefs(m.Items, func(i *Item) Ref { return i.Ref })
	sortRefs(m.Mods, func(md *Mod) Ref { return md.Ref })
	for _, md := range m.Mods {
		sortRefs(md.Items, func(i *Item) Ref { return i.Ref })
	}
	return m
}

// entityOf maps "ns:entities/zombie" to the entity "ns:zombie".
func entityOf(table domain.ResourceID) domain.ResourceID {
	path := strings.TrimPrefix(strings.TrimPrefix(table.Path, "entities/"), "entity/")
	return domain.ResourceID{Namespace: table.Namespace, Path: path}
}

func sortRefs[T any](s []T, ref func(T) Ref) {
	sort.SliceStable(s, func(i, j int) bool {
		a, b := ref(s[i]), ref(s[j])
		if !strings.EqualFold(a.Name, b.Name) {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return a.ID.String() < b.ID.String()
	})
}

func sortUses(uses []*TableUse) {
	sort.SliceStable(uses, func(i, j int) bool {
		if uses[i].Confidence != uses[j].Confidence {
			return uses[i].Confidence > uses[j].Confidence
		}
		return uses[i].Table.ID.String() < uses[j].Table.ID.String()
	})
}

// topItems returns the items with the best chance across the owners.
func topItems(owners []*Owner, limit int) []BiomeItem {
	best := map[*Item]*ItemSource{}
	for _, o := range owners {
		if o.Status.Disabled() {
			continue
		}
		for _, u := range o.Uses {
			for _, d := range u.Table.Drops {
				eff := d.Chance
				if u.Share > 0 {
					eff *= u.Share
				}
				if cur, ok := best[d.Item]; !ok || eff > cur.Effective {
					best[d.Item] = &ItemSource{Table: u.Table, Owner: o, Chance: d.Chance, Share: u.Share, Effective: eff, CountMin: d.CountMin, CountMax: d.CountMax, Confidence: u.Confidence}
				}
			}
		}
	}
	out := make([]BiomeItem, 0, len(best))
	for it, s := range best {
		out = append(out, BiomeItem{Item: it, Source: s})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source.Effective != out[j].Source.Effective {
			return out[i].Source.Effective > out[j].Source.Effective
		}
		return out[i].Item.ID.String() < out[j].Item.ID.String()
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
