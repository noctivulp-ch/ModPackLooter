package discovery

import (
	"context"
	"fmt"
	"sort"
)

// Registry holds every available plugin of one hook point, including
// version variants. It is built explicitly in the composition root; there is
// no global state.
type Registry[T Plugin] struct {
	plugins []T
}

// NewRegistry registers the given plugins.
func NewRegistry[T Plugin](ps ...T) *Registry[T] {
	return &Registry[T]{plugins: append([]T(nil), ps...)}
}

// Plan is the ordered list of plugins that apply to one target.
type Plan[T Plugin] struct {
	Steps []T
}

// Plan selects the variants that apply to the target and orders them.
// The generic phase sorts last by construction. It fails if two variants of
// the same ID apply to the target at once.
func (r *Registry[T]) Plan(t Target) (Plan[T], error) {
	var steps []T
	seen := map[string]Descriptor{}
	for _, p := range r.plugins {
		desc := p.Descriptor()
		if desc.ID == "" {
			return Plan[T]{}, fmt.Errorf("plugin sin ID: %T", p)
		}
		if desc.Phase < PhaseSpecific || desc.Phase > PhaseGeneric {
			return Plan[T]{}, fmt.Errorf("plugin %q con fase no válida", desc.ID)
		}
		if !desc.Applies.AppliesTo(t) {
			continue
		}
		if prev, dup := seen[desc.ID]; dup {
			return Plan[T]{}, fmt.Errorf("variantes solapadas de %q para %s: %s y %s",
				desc.ID, t.Version, prev.Applies.Versions, desc.Applies.Versions)
		}
		seen[desc.ID] = desc
		steps = append(steps, p)
	}
	sort.SliceStable(steps, func(i, j int) bool {
		a, b := steps[i].Descriptor(), steps[j].Descriptor()
		if a.Phase != b.Phase {
			return a.Phase < b.Phase
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		return a.ID < b.ID
	})
	return Plan[T]{Steps: steps}, nil
}

// Failure records a plugin that returned an error. A failing plugin does not
// stop the others.
type Failure struct {
	PluginID string
	Err      error
}

func run[T Plugin](ctx context.Context, steps []T, call func(T) error) []Failure {
	var failures []Failure
	for _, p := range steps {
		if err := ctx.Err(); err != nil {
			failures = append(failures, Failure{PluginID: p.Descriptor().ID, Err: err})
			break
		}
		if err := call(p); err != nil {
			failures = append(failures, Failure{PluginID: p.Descriptor().ID, Err: err})
		}
	}
	return failures
}

// Discover runs every discoverer of the plan and returns the claims.
func Discover(ctx context.Context, plan Plan[Discoverer], in Input) (*Claims, []Failure) {
	claims := NewClaims()
	failures := run(ctx, plan.Steps, func(d Discoverer) error { return d.Discover(ctx, in, claims) })
	return claims, failures
}

// Enrich runs every enricher of the plan over the claims.
func Enrich(ctx context.Context, plan Plan[Enricher], in Input, claims *Claims) []Failure {
	return run(ctx, plan.Steps, func(e Enricher) error { return e.Enrich(ctx, in, claims) })
}
