package vanilla

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/fishing"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// FishingID is the ID of the vanilla fishing discoverer.
const FishingID = "vanilla-fishing"

// fishingTable is the table the vanilla rod rolls.
var fishingTable = domain.MustParseResourceID("minecraft:gameplay/fishing")

// Fishing reads the vanilla fishing table for the fishing tab: its pool
// picks fish, junk or treasure by weight; treasure needs open water.
type Fishing struct{}

func (Fishing) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: FishingID, Phase: discovery.PhaseSpecific, Priority: 50}
}

func (Fishing) Discover(_ context.Context, in discovery.Input, out *discovery.Claims) error {
	var raw struct {
		Pools []struct {
			Entries []struct {
				Type       string            `json:"type"`
				Name       string            `json:"name"`
				Value      string            `json:"value"`
				Weight     *float64          `json:"weight"`
				Quality    float64           `json:"quality"`
				Conditions []json.RawMessage `json:"conditions"`
			} `json:"entries"`
		} `json:"pools"`
	}
	ok, err := in.Resources.ReadJSON(resources.TypeLootTable, fishingTable, &raw)
	if err != nil || !ok || len(raw.Pools) == 0 {
		return err
	}
	src := &fishing.Source{
		Mod: "minecraft", Name: "Minecraft (caña vanilla)", Order: 100,
		Intro: []string{"Con la caña normal, cada captura elige entre peces, basura y tesoro según su peso. El tesoro solo sale en aguas abiertas (un hueco de agua de 5×5 alrededor del corcho, sin bloques). Suerte del mar sube el tesoro y baja la basura."},
	}
	open := &fishing.Group{Title: "En aguas abiertas", Fluid: fishing.FluidWater}
	closed := &fishing.Group{Title: "Fuera de aguas abiertas", Fluid: fishing.FluidWater}
	var totalOpen, totalClosed float64
	for _, e := range raw.Pools[0].Entries {
		if !strings.HasSuffix(e.Type, "loot_table") {
			continue
		}
		name := e.Name
		if name == "" {
			name = e.Value
		}
		table, err := domain.ParseResourceID(name)
		if err != nil {
			continue
		}
		w := 1.0
		if e.Weight != nil {
			w = *e.Weight
		}
		entry := &fishing.Entry{ID: table, Kind: fishing.KindLoot, Table: table, Weight: w, Fluids: []string{fishing.FluidWater}}
		needsOpen := false
		for _, c := range e.Conditions {
			if strings.Contains(string(c), "in_open_water") {
				needsOpen = true
			}
		}
		if needsOpen {
			entry.Conditions = append(entry.Conditions, fishing.Cond{Text: "en aguas abiertas"})
		}
		if e.Quality != 0 {
			entry.Conditions = append(entry.Conditions, fishing.Cond{Text: fmt.Sprintf("Suerte del mar cambia su peso (%+g por nivel)", e.Quality)})
		}
		src.Entries = append(src.Entries, entry)
		open.Catches = append(open.Catches, fishing.Catch{Entry: entry, Weight: w})
		totalOpen += w
		if !needsOpen {
			closed.Catches = append(closed.Catches, fishing.Catch{Entry: entry, Weight: w})
			totalClosed += w
		}
	}
	if len(src.Entries) == 0 {
		return nil
	}
	for _, g := range []struct {
		g     *fishing.Group
		total float64
	}{{open, totalOpen}, {closed, totalClosed}} {
		for i := range g.g.Catches {
			g.g.Catches[i].Chance = g.g.Catches[i].Weight / g.total
		}
		fishing.SortCatches(g.g.Catches)
		src.Everywhere = append(src.Everywhere, g.g)
	}
	out.Attach(fishing.ExtraPrefix+"minecraft", src)
	return nil
}
