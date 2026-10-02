// Package plugins lists every discoverer and enricher shipped with the app.
// It is the single place where a new source or version variant is registered.
package plugins

import (
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery/generic"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery/heuristics"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery/lootr"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery/lostcities"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery/templates"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/discovery/vanilla"
)

// Discoverers returns the discovery hook point with every known plugin.
func Discoverers() *discovery.Registry[discovery.Discoverer] {
	return discovery.NewRegistry[discovery.Discoverer](
		templates.Discoverer{},
		vanilla.Discoverer1_20{},
		lostcities.Discoverer{},
		heuristics.NameMatching{},
		generic.ByPath{},
	)
}

// Enrichers returns the enrichment hook point with every known plugin.
func Enrichers() *discovery.Registry[discovery.Enricher] {
	return discovery.NewRegistry[discovery.Enricher](
		lootr.Enricher1_20{},
	)
}
