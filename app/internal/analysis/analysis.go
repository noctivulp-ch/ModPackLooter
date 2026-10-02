// Package analysis is the main use case: open a modpack, discover its loot
// sources, enrich them and resolve every loot table. Interfaces (CLI, future
// TUI) call it and only differ in how they report progress and results.
package analysis

import (
	"bytes"
	"context"
	"fmt"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/loot"
	"github.com/EnierAragon/ModPackLooter/app/internal/modpack"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
	"github.com/EnierAragon/ModPackLooter/app/internal/worldgen"
)

// Progress receives stage updates. Implementations must be cheap.
type Progress interface {
	Stage(name string)
}

// NoProgress discards progress updates.
type NoProgress struct{}

func (NoProgress) Stage(string) {}

// Analyzer holds the plugins registered at each hook point.
type Analyzer struct {
	Discoverers *discovery.Registry[discovery.Discoverer]
	Enrichers   *discovery.Registry[discovery.Enricher]
	Disablers   *discovery.Registry[discovery.Disabler]
	Changes     *discovery.Registry[discovery.ChangeDetector]
}

// Result is everything an output (site, report, query) needs.
type Result struct {
	Modpack     *modpack.Modpack
	Sources     []domain.LootSource
	Owners      []domain.OwnerInfo // owners declared by discoverers
	Structures  []worldgen.Structure
	Tables      map[domain.ResourceID]*loot.Table
	Plan        []string // discoverer IDs in execution order
	Enrichments []string
	Disablers   []string
	// Disablements lists every report of the disabler gate; Statuses combines
	// them per target (structures also inherit "all biomes disabled").
	Disablements []domain.Disablement
	Statuses     map[domain.Target]domain.Status
	Diagnostics  *domain.Diagnostics
	Resources    *Resources
	// Extras holds data attached by discoverers (see discovery.Claims.Attach).
	Extras map[string]any
	// Changes are modifications of loot sources by mods, scripts or configs.
	Changes         []domain.Change
	ChangeDetectors []string
}

// Close releases the modpack files.
func (r *Result) Close() error { return r.Modpack.Close() }

// Run analyses a modpack.
func (a Analyzer) Run(ctx context.Context, opts modpack.Options, progress Progress) (*Result, error) {
	if progress == nil {
		progress = NoProgress{}
	}
	diags := opts.Diagnostics
	if diags == nil {
		diags = &domain.Diagnostics{}
		opts.Diagnostics = diags
	}

	progress.Stage("Leyendo el modpack")
	mp, err := modpack.Open(opts)
	if err != nil {
		return nil, err
	}
	res := NewResources(mp.Index, diags)
	world := worldgen.New(mp.Index, diags)
	target := discovery.Target{Version: mp.MCVersion, Loader: mp.Loader, Mods: mp.ModIDs()}
	in := discovery.Input{Target: target, Resources: res, World: world, Files: mp, Diagnostics: diags}

	progress.Stage("Descubriendo fuentes de loot")
	plan, err := a.Discoverers.Plan(target)
	if err != nil {
		mp.Close()
		return nil, err
	}
	claims, failures := discovery.Discover(ctx, plan, in)
	report(diags, "discovery", failures)

	progress.Stage("Aplicando mods de comportamiento (Lootr…)")
	enrichPlan, err := a.Enrichers.Plan(target)
	if err != nil {
		mp.Close()
		return nil, err
	}
	report(diags, "enrichment", discovery.Enrich(ctx, enrichPlan, in, claims))

	progress.Stage("Buscando estructuras y criaturas desactivadas")
	var disPlan discovery.Plan[discovery.Disabler]
	if a.Disablers != nil {
		if disPlan, err = a.Disablers.Plan(target); err != nil {
			mp.Close()
			return nil, err
		}
	}
	dis, failures := discovery.Detect(ctx, disPlan, in)
	report(diags, "desactivadores", failures)

	progress.Stage("Buscando mods que cambian loot, tradeos y pesca")
	var chPlan discovery.Plan[discovery.ChangeDetector]
	if a.Changes != nil {
		if chPlan, err = a.Changes.Plan(target); err != nil {
			mp.Close()
			return nil, err
		}
	}
	changes, failures := discovery.DetectChanges(ctx, chPlan, in)
	report(diags, "cambios", failures)

	progress.Stage("Calculando probabilidades")
	resolver := loot.NewResolver(res)
	tables := map[domain.ResourceID]*loot.Table{}
	for _, id := range res.LootTables() {
		t, ok, err := resolver.Resolve(id)
		if err != nil {
			diags.Add(domain.LevelWarning, "parser", id.String(), "%v", err)
			continue
		}
		if ok {
			tables[id] = t
			for _, m := range t.Missing {
				diags.Add(domain.LevelWarning, "parser", id.String(), "referencia a una loot table inexistente: %s", m)
			}
		}
	}

	result := &Result{
		Modpack: mp, Sources: claims.Sources(), Owners: claims.Owners(),
		Structures: world.Structures(), Tables: tables, Diagnostics: diags, Resources: res,
		Disablements: dis.Items(), Extras: claims.Extras(), Changes: changes.Items(),
	}
	for _, d := range chPlan.Steps {
		result.ChangeDetectors = append(result.ChangeDetectors, d.Descriptor().ID)
	}
	result.Statuses = statuses(dis, result.Structures, result.Owners)
	for _, d := range disPlan.Steps {
		result.Disablers = append(result.Disablers, d.Descriptor().ID)
	}
	for _, d := range plan.Steps {
		result.Plan = append(result.Plan, d.Descriptor().ID)
	}
	for _, e := range enrichPlan.Steps {
		result.Enrichments = append(result.Enrichments, e.Descriptor().ID)
	}
	return result, nil
}

// statuses combines disablements per target. A structure whose biomes are
// all certainly disabled is certainly disabled too.
func statuses(dis *discovery.Disablements, structures []worldgen.Structure, owners []domain.OwnerInfo) map[domain.Target]domain.Status {
	out := map[domain.Target]domain.Status{}
	for _, d := range dis.Items() {
		if _, done := out[d.Target]; !done {
			out[d.Target] = dis.Status(d.Target)
		}
	}
	biomesOf := map[domain.ResourceID][]domain.ResourceID{}
	for _, s := range structures {
		biomesOf[s.ID] = s.Biomes
	}
	for _, o := range owners {
		biomesOf[o.Owner.ID] = o.Biomes
	}
	for id, biomes := range biomesOf {
		if len(biomes) == 0 {
			continue
		}
		all := true
		for _, b := range biomes {
			if !out[domain.Target{Kind: domain.TargetBiome, ID: b}].Disabled() {
				all = false
				break
			}
		}
		if all {
			t := domain.Target{Kind: domain.TargetStructure, ID: id}
			st := out[t]
			st.Certainty = domain.Certainly
			st.Reasons = append(st.Reasons, "todos sus biomas están desactivados")
			out[t] = st
		}
	}
	return out
}

// Status returns the combined disablement status of a target.
func (r *Result) Status(t domain.Target) domain.Status { return r.Statuses[t] }

func report(diags *domain.Diagnostics, stage string, failures []discovery.Failure) {
	for _, f := range failures {
		diags.Add(domain.LevelError, stage, f.PluginID, "%v", f.Err)
	}
}

// Resources adapts the resource index to the ports used by discoverers and
// the loot resolver.
type Resources struct {
	ix    *resources.Index
	diags *domain.Diagnostics
	tags  map[string]*worldgen.Tags
	langs map[string]map[string]string
}

// NewResources wraps an index.
func NewResources(ix *resources.Index, diags *domain.Diagnostics) *Resources {
	return &Resources{ix: ix, diags: diags, tags: map[string]*worldgen.Tags{}, langs: map[string]map[string]string{}}
}

// Index returns the underlying index.
func (r *Resources) Index() *resources.Index { return r.ix }

func (r *Resources) LootTables() []domain.ResourceID { return r.ix.IDs(resources.TypeLootTable) }

func (r *Resources) IDs(typ string) []domain.ResourceID { return r.ix.IDs(typ) }

func (r *Resources) ReadJSON(typ string, id domain.ResourceID, v any) (bool, error) {
	e, ok := r.ix.Lookup(typ, id)
	if !ok {
		return false, nil
	}
	return true, e.ReadJSON(v)
}

func (r *Resources) Tag(typ string, id domain.ResourceID) []domain.ResourceID {
	t := r.tags[typ]
	if t == nil {
		t = worldgen.NewTags(r.ix, typ, r.diags)
		r.tags[typ] = t
	}
	return t.Resolve(id)
}

func (r *Resources) Providers(typ string, id domain.ResourceID) []discovery.Provider {
	var out []discovery.Provider
	for _, e := range r.ix.All(typ, id) {
		e := e
		out = append(out, discovery.Provider{Pack: e.Pack.Name(), Kind: e.Pack.Kind(), Read: func() ([]byte, error) {
			data, err := e.Read()
			return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), err
		}})
	}
	return out
}

func (r *Resources) Lang(code string) map[string]string {
	if l, ok := r.langs[code]; ok {
		return l
	}
	l := r.ix.Lang(code, nil)
	r.langs[code] = l
	return l
}

func (r *Resources) LootTableJSON(id domain.ResourceID) ([]byte, bool, error) {
	e, ok := r.ix.Lookup(resources.TypeLootTable, id)
	if !ok {
		return nil, false, nil
	}
	data, err := e.Read()
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", e, err)
	}
	// Some mods save their JSON with a UTF-8 byte order mark.
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), true, nil
}

func (r *Resources) ItemTag(id domain.ResourceID) []domain.ResourceID {
	return r.Tag(resources.TypeItemTag, id)
}
