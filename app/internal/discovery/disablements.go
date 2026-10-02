package discovery

import (
	"context"
	"sort"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// Disablements collects what disablers report.
type Disablements struct {
	items []domain.Disablement
}

// Add records a disablement.
func (d *Disablements) Add(x domain.Disablement) { d.items = append(d.items, x) }

// Items returns every disablement in a deterministic order.
func (d *Disablements) Items() []domain.Disablement {
	out := append([]domain.Disablement(nil), d.items...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Target != b.Target {
			if a.Target.Kind != b.Target.Kind {
				return a.Target.Kind < b.Target.Kind
			}
			return a.Target.ID.String() < b.Target.ID.String()
		}
		return a.Certainty > b.Certainty
	})
	return out
}

// Status combines every disablement of a target: the highest certainty wins
// and all reasons are kept.
func (d *Disablements) Status(t domain.Target) domain.Status {
	var s domain.Status
	for _, x := range d.items {
		if x.Target != t {
			continue
		}
		if x.Certainty > s.Certainty {
			s.Certainty = x.Certainty
		}
		s.Reasons = append(s.Reasons, x.Reason)
	}
	return s
}

// Detect runs every disabler of the plan.
func Detect(ctx context.Context, plan Plan[Disabler], in Input) (*Disablements, []Failure) {
	out := &Disablements{}
	failures := run(ctx, plan.Steps, func(d Disabler) error { return d.Detect(ctx, in, out) })
	return out, failures
}
