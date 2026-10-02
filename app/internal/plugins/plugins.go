// Package plugins lists every discoverer and enricher shipped with the app.
// It is the single place where a new source or version variant is registered.
package plugins

import (
	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/changes"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/disablers"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/generic"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/heuristics"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/lootr"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/lostcities"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/starcatcher"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/templates"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/tide"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/vanilla"
)

// Discoverers returns the discovery hook point with every known plugin.
func Discoverers() *discovery.Registry[discovery.Discoverer] {
	return discovery.NewRegistry[discovery.Discoverer](
		templates.Discoverer{},
		vanilla.Discoverer1_20{},
		lostcities.Discoverer{},
		starcatcher.Discoverer{},
		tide.Discoverer{},
		vanilla.Fishing{},
		heuristics.NameMatching{},
		generic.ByPath{},
	)
}

// Disablers returns the disabler hook point with every known plugin.
func Disablers() *discovery.Registry[discovery.Disabler] {
	return discovery.NewRegistry[discovery.Disabler](
		disablers.StructureSets{},
		disablers.StructureBiomes{},
		disablers.BiomeReplacer{},
		disablers.Structurify{},
		disablers.InControl{},
		lostcities.Disabler{},
		disablers.ConfigMentions{},
		disablers.KubeJS{},
	)
}

// Enrichers returns the enrichment hook point with every known plugin.
func Enrichers() *discovery.Registry[discovery.Enricher] {
	return discovery.NewRegistry[discovery.Enricher](
		lootr.Enricher1_20{},
	)
}

// Changes returns the change-detector hook point: specific plugins first,
// generic ones (any mod, script or config) last.
func Changes() *discovery.Registry[discovery.ChangeDetector] {
	return discovery.NewRegistry[discovery.ChangeDetector](
		changes.TableOverrides{},
		changes.GlobalLootModifiers{},
		changes.LootJS{},
		changes.KubeJSLootEvents{},
		changes.TradeScripts{},
		changes.CraftTweaker{},
		changes.ScriptMentions{},
		changes.ConfigKeys{},
		changes.CodeHooks{},
	)
}

// Analyzer returns the analysis use case with every hook point filled.
func Analyzer() analysis.Analyzer {
	return analysis.Analyzer{Discoverers: Discoverers(), Enrichers: Enrichers(), Disablers: Disablers(), Changes: Changes()}
}
