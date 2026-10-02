package discovery

import (
	"context"
	"fmt"
	"sort"
)

// Registry holds every available discoverer, including version variants.
// It is built explicitly in the composition root; there is no global state.
type Registry struct {
	discoverers []Discoverer
}

// NewRegistry registers the given discoverers.
func NewRegistry(ds ...Discoverer) *Registry {
	return &Registry{discoverers: append([]Discoverer(nil), ds...)}
}

// Plan is the ordered list of discoverers that apply to one target.
type Plan struct {
	Steps []Discoverer
}

// Plan selects the variants that apply to the target and orders them.
// The generic phase sorts last by construction. It fails if two variants of
// the same ID apply to the target at once.
func (r *Registry) Plan(t Target) (Plan, error) {
	var steps []Discoverer
	seen := map[string]Descriptor{}
	for _, d := range r.discoverers {
		desc := d.Descriptor()
		if desc.ID == "" {
			return Plan{}, fmt.Errorf("descubridor sin ID: %T", d)
		}
		if desc.Phase < PhaseSpecific || desc.Phase > PhaseGeneric {
			return Plan{}, fmt.Errorf("descubridor %q con fase no válida", desc.ID)
		}
		if !desc.Applies.AppliesTo(t) {
			continue
		}
		if prev, dup := seen[desc.ID]; dup {
			return Plan{}, fmt.Errorf("variantes solapadas de %q para %s: %s y %s",
				desc.ID, t.Version, prev.Applies.Versions, desc.Applies.Versions)
		}
		seen[desc.ID] = desc
		steps = append(steps, d)
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
	return Plan{Steps: steps}, nil
}

// Failure records a discoverer that returned an error. A failing discoverer
// does not stop the others.
type Failure struct {
	DiscovererID string
	Err          error
}

// Run executes the plan in order and returns the accumulated claims.
func (p Plan) Run(ctx context.Context, in Input) (*Claims, []Failure) {
	claims := NewClaims()
	var failures []Failure
	for _, d := range p.Steps {
		if err := ctx.Err(); err != nil {
			failures = append(failures, Failure{DiscovererID: d.Descriptor().ID, Err: err})
			break
		}
		if err := d.Discover(ctx, in, claims); err != nil {
			failures = append(failures, Failure{DiscovererID: d.Descriptor().ID, Err: err})
		}
	}
	return claims, failures
}
