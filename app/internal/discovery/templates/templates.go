// Package templates discovers loot placed by structure templates: containers
// saved with a LootTable in the .nbt files and processor lists that append
// loot while the structure generates.
package templates

import (
	"context"
	"fmt"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// ID of the discoverer.
const ID = "structure-templates"

// Discoverer walks every jigsaw structure and every template with loot.
type Discoverer struct{}

func (Discoverer) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ID, Phase: discovery.PhaseSpecific, Priority: 100}
}

func (Discoverer) Discover(ctx context.Context, in discovery.Input, out *discovery.Claims) error {
	reached := map[domain.ResourceID]bool{}
	for _, s := range in.World.Structures() {
		if err := ctx.Err(); err != nil {
			return err
		}
		found, templates := in.World.StructureLoot(s)
		for _, t := range templates {
			reached[t] = true
		}
		for _, f := range found {
			detail := fmt.Sprintf("plantilla %s", f.Template)
			if f.Via == "processor" {
				detail = fmt.Sprintf("processor list %s en la plantilla %s", f.Detail, f.Template)
			}
			out.Add(domain.LootSource{
				LootTable:  f.LootTable,
				Kind:       discovery.KindForContainer(f.Block),
				Owner:      domain.Owner{Kind: domain.OwnerStructure, ID: s.ID},
				Confidence: domain.ConfidenceExact,
				Container:  f.Block,
				Evidence:   []domain.Evidence{{DiscoveredBy: ID, Detail: detail}},
			})
		}
	}
	// Templates no structure reaches are often placed by code (features,
	// modded structures). The table is still tied to that template.
	for _, t := range in.World.TemplatesWithLoot() {
		if reached[t.ID] {
			continue
		}
		for _, c := range t.Containers {
			out.Add(domain.LootSource{
				LootTable:  c.LootTable,
				Kind:       discovery.KindForContainer(c.Block),
				Owner:      domain.Owner{Kind: domain.OwnerTemplate, ID: t.ID},
				Confidence: domain.ConfidenceExact,
				Container:  c.Block,
				Evidence:   []domain.Evidence{{DiscoveredBy: ID, Detail: "plantilla sin estructura conocida"}},
			})
		}
	}
	return nil
}
