package worldgen

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/nbt"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

const stage = "parser"

// Structure is a worldgen structure definition.
type Structure struct {
	ID        domain.ResourceID
	Type      string
	Biomes    []domain.ResourceID
	StartPool *domain.ResourceID
}

// Container is a block or entity in a template that rolls a loot table.
type Container struct {
	Block     string // block or entity id, e.g. "minecraft:chest"
	Entity    bool
	LootTable domain.ResourceID
}

// Template is what the app needs from a structure template (.nbt).
type Template struct {
	ID          domain.ResourceID
	Containers  []Container
	JigsawPools []domain.ResourceID
	DataMarkers []string
}

// Found is a loot table placed by a structure.
type Found struct {
	LootTable domain.ResourceID
	Block     string
	Template  domain.ResourceID
	Via       string // "nbt" or "processor"
	Detail    string // processor list id, when Via == "processor"
}

// World gives cached access to worldgen data.
type World struct {
	ix        *resources.Index
	diags     *domain.Diagnostics
	Biomes    *Tags
	templates map[domain.ResourceID]*Template
	pools     map[domain.ResourceID]*pool
	procs     map[domain.ResourceID][]procLoot
}

// New creates a World over the index.
func New(ix *resources.Index, diags *domain.Diagnostics) *World {
	return &World{
		ix: ix, diags: diags,
		Biomes:    NewTags(ix, resources.TypeBiomeTag, diags),
		templates: map[domain.ResourceID]*Template{},
		pools:     map[domain.ResourceID]*pool{},
		procs:     map[domain.ResourceID][]procLoot{},
	}
}

func (w *World) warn(e resources.Entry, format string, args ...any) {
	if w.diags != nil {
		w.diags.Add(domain.LevelWarning, stage, e.String(), format, args...)
	}
}

// Structures lists every structure definition.
func (w *World) Structures() []Structure {
	var out []Structure
	for _, id := range w.ix.IDs(resources.TypeStructure) {
		e, _ := w.ix.Lookup(resources.TypeStructure, id)
		var raw struct {
			Type      string          `json:"type"`
			Biomes    json.RawMessage `json:"biomes"`
			StartPool string          `json:"start_pool"`
		}
		if err := e.ReadJSON(&raw); err != nil {
			w.warn(e, "estructura ilegible: %v", err)
			continue
		}
		s := Structure{ID: id, Type: strings.TrimPrefix(raw.Type, "minecraft:"), Biomes: w.Biomes.HolderSet(raw.Biomes)}
		if raw.StartPool != "" {
			if p, err := domain.ParseResourceID(raw.StartPool); err == nil {
				s.StartPool = &p
			}
		}
		out = append(out, s)
	}
	return out
}

// Template reads and caches a structure template. ok is false if missing.
func (w *World) Template(id domain.ResourceID) (*Template, bool) {
	if t, ok := w.templates[id]; ok {
		return t, t != nil
	}
	e, ok := w.ix.Lookup(resources.TypeStructureNBT, id)
	if !ok {
		w.templates[id] = nil
		return nil, false
	}
	data, err := e.Read()
	if err != nil {
		w.warn(e, "plantilla ilegible: %v", err)
		w.templates[id] = nil
		return nil, false
	}
	doc, err := nbt.Decode(data)
	if err != nil {
		w.warn(e, "plantilla NBT ilegible: %v", err)
		w.templates[id] = nil
		return nil, false
	}
	t := parseTemplate(id, doc)
	w.templates[id] = t
	return t, true
}

func parseTemplate(id domain.ResourceID, doc nbt.Compound) *Template {
	t := &Template{ID: id}
	palette := doc.List("palette")
	if palette == nil {
		// Templates with several palettes (e.g. shipwrecks) list them in "palettes".
		if ps := doc.List("palettes"); len(ps) > 0 {
			palette, _ = ps[0].(nbt.List)
		}
	}
	blockName := func(state int) string {
		if state >= 0 && state < len(palette) {
			if c, ok := palette[state].(nbt.Compound); ok {
				return c.String("Name")
			}
		}
		return ""
	}
	pools := map[domain.ResourceID]bool{}
	for _, b := range doc.List("blocks") {
		block, ok := b.(nbt.Compound)
		if !ok {
			continue
		}
		data := block.Compound("nbt")
		if data == nil {
			continue
		}
		state, _ := block.Int("state")
		name := blockName(state)
		if lt := data.String("LootTable"); lt != "" {
			if rid, err := domain.ParseResourceID(lt); err == nil {
				t.Containers = append(t.Containers, Container{Block: name, LootTable: rid})
			}
		}
		if p := data.String("pool"); p != "" && strings.HasSuffix(name, "jigsaw") {
			if rid, err := domain.ParseResourceID(p); err == nil && rid.Path != "empty" {
				pools[rid] = true
			}
		}
		if data.String("mode") == "DATA" {
			if m := data.String("metadata"); m != "" {
				t.DataMarkers = append(t.DataMarkers, m)
			}
		}
	}
	for _, en := range doc.List("entities") {
		entity, ok := en.(nbt.Compound)
		if !ok {
			continue
		}
		data := entity.Compound("nbt")
		if data == nil {
			continue
		}
		if lt := data.String("LootTable"); lt != "" {
			if rid, err := domain.ParseResourceID(lt); err == nil {
				t.Containers = append(t.Containers, Container{Block: data.String("id"), Entity: true, LootTable: rid})
			}
		}
	}
	for p := range pools {
		t.JigsawPools = append(t.JigsawPools, p)
	}
	sort.Slice(t.JigsawPools, func(i, j int) bool { return t.JigsawPools[i].String() < t.JigsawPools[j].String() })
	return t
}

type poolElement struct {
	template domain.ResourceID
	procs    []procLoot
	procID   string
}

type pool struct {
	elements []poolElement
	fallback *domain.ResourceID
}

type procLoot struct {
	lootTable domain.ResourceID
	block     string
}

func (w *World) pool(id domain.ResourceID) *pool {
	if p, ok := w.pools[id]; ok {
		return p
	}
	w.pools[id] = nil
	e, ok := w.ix.Lookup(resources.TypeTemplatePool, id)
	if !ok {
		return nil
	}
	var raw struct {
		Fallback string `json:"fallback"`
		Elements []struct {
			Element json.RawMessage `json:"element"`
		} `json:"elements"`
	}
	if err := e.ReadJSON(&raw); err != nil {
		w.warn(e, "template pool ilegible: %v", err)
		return nil
	}
	p := &pool{}
	if raw.Fallback != "" {
		if f, err := domain.ParseResourceID(raw.Fallback); err == nil && f.Path != "empty" {
			p.fallback = &f
		}
	}
	for _, el := range raw.Elements {
		p.elements = append(p.elements, w.poolElements(el.Element)...)
	}
	w.pools[id] = p
	return p
}

func (w *World) poolElements(raw json.RawMessage) []poolElement {
	var el struct {
		Type       string            `json:"element_type"`
		Location   string            `json:"location"`
		Processors json.RawMessage   `json:"processors"`
		Elements   []json.RawMessage `json:"elements"`
	}
	if err := json.Unmarshal(raw, &el); err != nil {
		return nil
	}
	switch strings.TrimPrefix(el.Type, "minecraft:") {
	case "single_pool_element", "legacy_single_pool_element":
		loc, err := domain.ParseResourceID(el.Location)
		if err != nil {
			return nil
		}
		procs, procID := w.processors(el.Processors)
		return []poolElement{{template: loc, procs: procs, procID: procID}}
	case "list_pool_element":
		var out []poolElement
		for _, child := range el.Elements {
			out = append(out, w.poolElements(child)...)
		}
		return out
	default:
		// Mods add their own element types; most keep "location".
		if loc, err := domain.ParseResourceID(el.Location); err == nil && el.Location != "" {
			procs, procID := w.processors(el.Processors)
			return []poolElement{{template: loc, procs: procs, procID: procID}}
		}
		return nil
	}
}

// processors resolves a processor list reference (id or inline) and returns
// the loot it appends.
func (w *World) processors(raw json.RawMessage) ([]procLoot, string) {
	if len(raw) == 0 {
		return nil, ""
	}
	var ref string
	if err := json.Unmarshal(raw, &ref); err == nil {
		id, err := domain.ParseResourceID(ref)
		if err != nil {
			return nil, ""
		}
		if cached, ok := w.procs[id]; ok {
			return cached, id.String()
		}
		var found []procLoot
		if e, ok := w.ix.Lookup(resources.TypeProcessorList, id); ok {
			var doc any
			if err := e.ReadJSON(&doc); err != nil {
				w.warn(e, "processor list ilegible: %v", err)
			} else {
				found = findAppendLoot(doc, "")
			}
		}
		w.procs[id] = found
		return found, id.String()
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, ""
	}
	return findAppendLoot(doc, ""), "inline"
}

// findAppendLoot walks any processor JSON (rule, capped, modded wrappers)
// looking for block entity modifiers of type append_loot.
func findAppendLoot(v any, block string) []procLoot {
	var out []procLoot
	switch x := v.(type) {
	case map[string]any:
		if os, ok := x["output_state"].(map[string]any); ok {
			if name, ok := os["Name"].(string); ok {
				block = name
			}
		}
		if t, _ := x["type"].(string); strings.TrimPrefix(t, "minecraft:") == "append_loot" {
			if lt, _ := x["loot_table"].(string); lt != "" {
				if id, err := domain.ParseResourceID(lt); err == nil {
					out = append(out, procLoot{lootTable: id, block: block})
				}
			}
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, findAppendLoot(x[k], block)...)
		}
	case []any:
		for _, e := range x {
			out = append(out, findAppendLoot(e, block)...)
		}
	}
	return out
}

// maxTemplates bounds the jigsaw walk of a single structure.
const maxTemplates = 5000

// StructureLoot walks the jigsaw pools of a structure and returns every loot
// table placed by its templates and processors, plus the templates reached.
func (w *World) StructureLoot(s Structure) ([]Found, []domain.ResourceID) {
	if s.StartPool == nil {
		return nil, nil
	}
	seenPools := map[domain.ResourceID]bool{}
	seenTemplates := map[domain.ResourceID]bool{}
	seenFound := map[Found]bool{}
	var found []Found
	var reached []domain.ResourceID
	queue := []domain.ResourceID{*s.StartPool}
	for len(queue) > 0 && len(reached) < maxTemplates {
		pid := queue[0]
		queue = queue[1:]
		if seenPools[pid] {
			continue
		}
		seenPools[pid] = true
		p := w.pool(pid)
		if p == nil {
			continue
		}
		if p.fallback != nil {
			queue = append(queue, *p.fallback)
		}
		for _, el := range p.elements {
			for _, pl := range el.procs {
				f := Found{LootTable: pl.lootTable, Block: pl.block, Template: el.template, Via: "processor", Detail: el.procID}
				if !seenFound[f] {
					seenFound[f] = true
					found = append(found, f)
				}
			}
			if seenTemplates[el.template] {
				continue
			}
			seenTemplates[el.template] = true
			t, ok := w.Template(el.template)
			if !ok {
				continue
			}
			reached = append(reached, el.template)
			for _, c := range t.Containers {
				f := Found{LootTable: c.LootTable, Block: c.Block, Template: t.ID, Via: "nbt"}
				if !seenFound[f] {
					seenFound[f] = true
					found = append(found, f)
				}
			}
			queue = append(queue, t.JigsawPools...)
		}
	}
	return found, reached
}

// TemplatesWithLoot lists every template that contains loot containers,
// whether or not a structure reaches it.
func (w *World) TemplatesWithLoot() []*Template {
	var out []*Template
	for _, id := range w.ix.IDs(resources.TypeStructureNBT) {
		if t, ok := w.Template(id); ok && len(t.Containers) > 0 {
			out = append(out, t)
		}
	}
	return out
}

// BiomeTag resolves a biome tag.
func (w *World) BiomeTag(id domain.ResourceID) []domain.ResourceID { return w.Biomes.Resolve(id) }

// StructureSets maps each structure to the structure sets that place it.
// Structures absent from the map are not placed by any set.
func (w *World) StructureSets() map[domain.ResourceID][]domain.ResourceID {
	out := map[domain.ResourceID][]domain.ResourceID{}
	for _, id := range w.ix.IDs(resources.TypeStructureSet) {
		e, _ := w.ix.Lookup(resources.TypeStructureSet, id)
		var raw struct {
			Structures []struct {
				Structure string  `json:"structure"`
				Weight    float64 `json:"weight"`
			} `json:"structures"`
		}
		if err := e.ReadJSON(&raw); err != nil {
			w.warn(e, "structure set ilegible: %v", err)
			continue
		}
		for _, s := range raw.Structures {
			if sid, err := domain.ParseResourceID(s.Structure); err == nil {
				out[sid] = append(out[sid], id)
			}
		}
	}
	return out
}

// HasStructureSets reports whether any structure set was loaded at all.
func (w *World) HasStructureSets() bool { return w.ix.Count(resources.TypeStructureSet) > 0 }
