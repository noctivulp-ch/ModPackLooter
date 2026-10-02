package discovery

import (
	"context"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// ChangeDetector finds modifications of loot sources (loot tables, drops,
// trades, fishing…) made by mods, datapacks, scripts or configs. Specific
// detectors (a mod whose format is known) run first; generic ones catch the
// rest and skip origins already explained (see Changes.Covered).
type ChangeDetector interface {
	Plugin
	Detect(ctx context.Context, in Input, out *Changes) error
}

// Changes collects what change detectors report.
type Changes struct {
	items   []domain.Change
	covered map[string]bool
}

// Add records a change and marks its origin as covered.
func (c *Changes) Add(x domain.Change) {
	if c.covered == nil {
		c.covered = map[string]bool{}
	}
	c.items = append(c.items, x)
	if x.Origin != "" {
		c.covered[x.Origin] = true
	}
}

// Cover marks an origin as handled without reporting a change (a specific
// plugin read it and found nothing worth showing). "file:*" covers a file.
func (c *Changes) Cover(origin string) {
	if c.covered == nil {
		c.covered = map[string]bool{}
	}
	c.covered[origin] = true
}

// Covered reports whether a specific detector already handled the origin
// ("file:line"), or the whole file.
func (c *Changes) Covered(origin string) bool {
	if c.covered[origin] {
		return true
	}
	file, _, _ := strings.Cut(origin, ":")
	return c.covered[file+":*"]
}

// Items returns the changes in a deterministic order.
func (c *Changes) Items() []domain.Change {
	out := append([]domain.Change(nil), c.items...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Target.Kind != b.Target.Kind {
			return a.Target.Kind < b.Target.Kind
		}
		if a.Target.ID != b.Target.ID {
			return a.Target.ID.String() < b.Target.ID.String()
		}
		return a.Origin < b.Origin
	})
	return out
}

// DetectChanges runs every change detector of the plan.
func DetectChanges(ctx context.Context, plan Plan[ChangeDetector], in Input) (*Changes, []Failure) {
	out := &Changes{}
	failures := run(ctx, plan.Steps, func(d ChangeDetector) error { return d.Detect(ctx, in, out) })
	return out, failures
}

// Provider is one pack that ships a resource; the last one wins.
type Provider struct {
	Pack string
	Kind resources.PackKind
	Read func() ([]byte, error)
}

// Jar is a mod jar, for detectors that look inside code.
type Jar struct {
	File string
	Pack resources.Pack
}
