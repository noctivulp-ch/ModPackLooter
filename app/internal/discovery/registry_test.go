package discovery_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/generic"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
)

// fake is a configurable discoverer for tests.
type fake struct {
	desc    discovery.Descriptor
	sources []domain.LootSource
	err     error
	calls   *[]string
}

func (f fake) Descriptor() discovery.Descriptor { return f.desc }

func (f fake) Discover(_ context.Context, _ discovery.Input, out *discovery.Claims) error {
	if f.calls != nil {
		*f.calls = append(*f.calls, f.desc.ID)
	}
	for _, s := range f.sources {
		out.Add(s)
	}
	return f.err
}

type memIndex []domain.ResourceID

func (m memIndex) LootTables() []domain.ResourceID                       { return m }
func (m memIndex) ReadJSON(string, domain.ResourceID, any) (bool, error) { return false, nil }
func (m memIndex) IDs(string) []domain.ResourceID                        { return nil }
func (m memIndex) Tag(string, domain.ResourceID) []domain.ResourceID     { return nil }

func target(version string, loader domain.Loader, mods ...string) discovery.Target {
	t := discovery.Target{Version: mcversion.MustParse(version), Loader: loader, Mods: map[string]bool{}}
	for _, m := range mods {
		t.Mods[m] = true
	}
	return t
}

func ids(p discovery.Plan[discovery.Discoverer]) []string {
	var out []string
	for _, d := range p.Steps {
		out = append(out, d.Descriptor().ID)
	}
	return out
}

func TestPlanOrdersByPhasePriorityAndID(t *testing.T) {
	reg := discovery.NewRegistry[discovery.Discoverer](
		generic.ByPath{},
		fake{desc: discovery.Descriptor{ID: "heuristic", Phase: discovery.PhaseHeuristic}},
		fake{desc: discovery.Descriptor{ID: "b-specific", Phase: discovery.PhaseSpecific}},
		fake{desc: discovery.Descriptor{ID: "a-specific", Phase: discovery.PhaseSpecific}},
		fake{desc: discovery.Descriptor{ID: "urgent", Phase: discovery.PhaseSpecific, Priority: 10}},
		fake{desc: discovery.Descriptor{ID: "spawns", Phase: discovery.PhaseRelational}},
	)
	plan, err := reg.Plan(target("1.20.1", domain.LoaderForge))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"urgent", "a-specific", "b-specific", "spawns", "heuristic", generic.ID}
	if got := ids(plan); !reflect.DeepEqual(got, want) {
		t.Errorf("orden = %v, se esperaba %v", got, want)
	}
}

func TestPlanSelectsVersionLoaderAndModVariants(t *testing.T) {
	v120 := fake{desc: discovery.Descriptor{ID: "lootr-extras", Phase: discovery.PhaseSpecific, Applies: discovery.Applicability{
		Versions: mcversion.MustParseRange(">=1.20.1 <1.21"), RequiresMods: []string{"lootr"},
	}}}
	v26 := fake{desc: discovery.Descriptor{ID: "lootr-extras", Phase: discovery.PhaseSpecific, Applies: discovery.Applicability{
		Versions: mcversion.MustParseRange(">=26.1"), RequiresMods: []string{"lootr"},
	}}}
	neoOnly := fake{desc: discovery.Descriptor{ID: "neoforge-glm", Phase: discovery.PhaseSpecific, Applies: discovery.Applicability{
		Loaders: []domain.Loader{domain.LoaderNeoForge},
	}}}
	reg := discovery.NewRegistry[discovery.Discoverer](v120, v26, neoOnly)

	cases := []struct {
		name string
		t    discovery.Target
		want []string
	}{
		{"forge 1.20.1 con lootr", target("1.20.1", domain.LoaderForge, "lootr"), []string{"lootr-extras"}},
		{"forge 1.20.1 sin lootr", target("1.20.1", domain.LoaderForge), nil},
		{"neoforge 26.3 con lootr", target("26.3", domain.LoaderNeoForge, "lootr"), []string{"lootr-extras", "neoforge-glm"}},
		{"neoforge 1.21.1 con lootr", target("1.21.1", domain.LoaderNeoForge, "lootr"), []string{"neoforge-glm"}},
	}
	for _, c := range cases {
		plan, err := reg.Plan(c.t)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := ids(plan); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: plan = %v, se esperaba %v", c.name, got, c.want)
		}
	}
	plan, _ := reg.Plan(target("26.3", domain.LoaderNeoForge, "lootr"))
	if !reflect.DeepEqual(plan.Steps[0].Descriptor().Applies.Versions, v26.desc.Applies.Versions) {
		t.Error("para 26.3 debe elegirse la variante de 26.x")
	}
}

func TestPlanRejectsOverlappingVariants(t *testing.T) {
	reg := discovery.NewRegistry[discovery.Discoverer](
		fake{desc: discovery.Descriptor{ID: "templates", Phase: discovery.PhaseSpecific, Applies: discovery.Applicability{Versions: mcversion.MustParseRange(">=1.20.1")}}},
		fake{desc: discovery.Descriptor{ID: "templates", Phase: discovery.PhaseSpecific, Applies: discovery.Applicability{Versions: mcversion.MustParseRange(">=1.21")}}},
	)
	if _, err := reg.Plan(target("1.20.1", domain.LoaderForge)); err != nil {
		t.Errorf("1.20.1 solo tiene una variante aplicable: %v", err)
	}
	if _, err := reg.Plan(target("1.21.1", domain.LoaderForge)); err == nil {
		t.Error("se esperaba error por variantes solapadas en 1.21.1")
	}
}

func TestPlanRejectsInvalidDescriptors(t *testing.T) {
	for _, d := range []discovery.Descriptor{{Phase: discovery.PhaseSpecific}, {ID: "x"}} {
		if _, err := discovery.NewRegistry[discovery.Discoverer](fake{desc: d}).Plan(target("1.20.1", domain.LoaderForge)); err == nil {
			t.Errorf("se esperaba error para %+v", d)
		}
	}
}

func TestRunKeepsGoingAfterFailureAndGenericSkipsClaimedTables(t *testing.T) {
	desert := domain.MustParseResourceID("minecraft:chests/desert_pyramid")
	zombie := domain.MustParseResourceID("minecraft:entities/zombie")
	custom := domain.MustParseResourceID("examplemod:loot/mystery")
	pyramid := domain.Owner{Kind: domain.OwnerStructure, ID: domain.MustParseResourceID("minecraft:desert_pyramid")}

	var calls []string
	reg := discovery.NewRegistry[discovery.Discoverer](
		generic.ByPath{},
		fake{calls: &calls, err: errors.New("boom"), desc: discovery.Descriptor{ID: "broken", Phase: discovery.PhaseSpecific, Priority: 5}},
		fake{calls: &calls, desc: discovery.Descriptor{ID: "templates", Phase: discovery.PhaseSpecific}, sources: []domain.LootSource{{
			LootTable: desert, Kind: domain.KindContainer, Owner: pyramid, Confidence: domain.ConfidenceExact,
			Evidence: []domain.Evidence{{DiscoveredBy: "templates"}},
		}}},
		fake{calls: &calls, desc: discovery.Descriptor{ID: "names", Phase: discovery.PhaseHeuristic}, sources: []domain.LootSource{{
			LootTable: desert, Kind: domain.KindContainer, Owner: pyramid, Confidence: domain.ConfidenceHeuristic,
			Evidence: []domain.Evidence{{DiscoveredBy: "names"}},
		}}},
	)
	plan, err := reg.Plan(target("1.20.1", domain.LoaderForge))
	if err != nil {
		t.Fatal(err)
	}
	claims, failures := discovery.Discover(context.Background(), plan, discovery.Input{Resources: memIndex{desert, zombie, custom}})

	if len(failures) != 1 || failures[0].PluginID != "broken" {
		t.Errorf("fallos = %+v, se esperaba solo 'broken'", failures)
	}
	if want := []string{"broken", "templates", "names"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("llamadas = %v, se esperaba %v", calls, want)
	}

	sources := claims.Sources()
	if len(sources) != 3 {
		t.Fatalf("se esperaban 3 fuentes, hay %d: %+v", len(sources), sources)
	}
	byTable := map[domain.ResourceID]domain.LootSource{}
	for _, s := range sources {
		byTable[s.LootTable] = s
	}
	got := byTable[desert]
	if got.Confidence != domain.ConfidenceExact || len(got.Evidence) != 2 {
		t.Errorf("la pirámide debe conservar confianza exacta y fusionar evidencia: %+v", got)
	}
	if byTable[zombie].Kind != domain.KindEntity || byTable[zombie].Confidence != domain.ConfidenceUnknown {
		t.Errorf("zombie debe clasificarse por ruta como entidad: %+v", byTable[zombie])
	}
	if byTable[custom].Kind != domain.KindUnknown {
		t.Errorf("tabla sin carpeta conocida debe ser 'unknown': %+v", byTable[custom])
	}
}
