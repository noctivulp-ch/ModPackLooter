// Package lootr annotates container sources with how Lootr changes them:
// loot per player, refresh and decay. Lootr does not change loot tables, so
// it is an enricher, not a discoverer. See docs/11-lootr.md.
package lootr

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// ID of the enricher.
const ID = "lootr"

// Note keys.
const (
	NotePerPlayer   = "lootr.per_player"
	NoteRefresh     = "lootr.refresh"
	NoteDecay       = "lootr.decay"
	NoteBlacklisted = "lootr.blacklisted"
)

// config holds the Lootr common options the site cares about, with the
// mod's defaults (ConfigManager in Lootr 0.7.x for 1.20.1).
type config struct {
	ConvertMineshafts   bool
	ConvertQuark        bool
	ConvertWoodenChests bool
	ConvertTrapped      bool
	AdditionalChests    []string
	AdditionalTrapped   []string
	LootTableBlacklist  []string
	LootModidBlacklist  []string
	StructureBlacklist  []string
	DecayValue          int
	DecayAll            bool
	DecayModids         []string
	DecayLootTables     []string
	RefreshValue        int
	RefreshAll          bool
	RefreshModids       []string
	RefreshLootTables   []string
}

func defaults() config {
	return config{
		ConvertMineshafts: true, ConvertQuark: true, ConvertWoodenChests: true, ConvertTrapped: true,
		DecayValue: 5 * 60 * 20, RefreshValue: 20 * 60 * 20,
	}
}

// parseConfig reads lootr-common.toml. Lootr 0.7.x (1.20.1) keeps every key
// at the root while later versions group them in sections, so keys are
// looked up regardless of the section they are in.
func parseConfig(data string) (config, error) {
	var doc map[string]any
	if _, err := toml.Decode(data, &doc); err != nil {
		return defaults(), err
	}
	flat := map[string]any{}
	var walk func(map[string]any)
	walk = func(m map[string]any) {
		for k, v := range m {
			if sub, ok := v.(map[string]any); ok {
				walk(sub)
				continue
			}
			flat[k] = v
		}
	}
	walk(doc)
	cfg := defaults()
	boolean := func(key string, dst *bool) {
		if v, ok := flat[key].(bool); ok {
			*dst = v
		}
	}
	integer := func(key string, dst *int) {
		if v, ok := flat[key].(int64); ok {
			*dst = int(v)
		}
	}
	list := func(key string, dst *[]string) {
		if v, ok := flat[key].([]any); ok {
			for _, e := range v {
				if s, ok := e.(string); ok {
					*dst = append(*dst, s)
				}
			}
		}
	}
	boolean("convert_mineshafts", &cfg.ConvertMineshafts)
	boolean("convert_quark", &cfg.ConvertQuark)
	boolean("convert_wooden_chests", &cfg.ConvertWoodenChests)
	boolean("convert_trapped_chests", &cfg.ConvertTrapped)
	list("additional_chests", &cfg.AdditionalChests)
	list("additional_trapped_chests", &cfg.AdditionalTrapped)
	list("loot_table_blacklist", &cfg.LootTableBlacklist)
	list("loot_modid_blacklist", &cfg.LootModidBlacklist)
	list("loot_table_modid_blacklist", &cfg.LootModidBlacklist)
	list("loot_structure_blacklist", &cfg.StructureBlacklist)
	integer("decay_value", &cfg.DecayValue)
	boolean("decay_all", &cfg.DecayAll)
	list("decay_modids", &cfg.DecayModids)
	list("decay_loot_table_modids", &cfg.DecayModids)
	list("decay_loot_tables", &cfg.DecayLootTables)
	integer("refresh_value", &cfg.RefreshValue)
	boolean("refresh_all", &cfg.RefreshAll)
	list("refresh_modids", &cfg.RefreshModids)
	list("refresh_loot_table_modids", &cfg.RefreshModids)
	list("refresh_loot_tables", &cfg.RefreshLootTables)
	return cfg, nil
}

// Enricher1_20 implements Lootr 0.7.x for Minecraft 1.20.x (Forge).
type Enricher1_20 struct{}

func (Enricher1_20) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{
		ID: ID, Phase: discovery.PhaseSpecific,
		Applies: discovery.Applicability{Versions: mcversion.MustParseRange(">=1.20 <1.21"), RequiresMods: []string{"lootr"}},
	}
}

func (Enricher1_20) Enrich(_ context.Context, in discovery.Input, claims *discovery.Claims) error {
	cfg := defaults()
	source := "valores por defecto"
	for _, path := range []string{"config/lootr-common.toml", "defaultconfigs/lootr-common.toml"} {
		data, err := in.Files.ReadFile(path)
		if err != nil {
			continue
		}
		parsed, err := parseConfig(string(data))
		if err != nil {
			in.Diagnostics.Add(domain.LevelWarning, "enrichment", path, "configuración de Lootr ilegible, se usan los valores por defecto: %v", err)
			break
		}
		cfg = parsed
		source = path
		break
	}
	in.Diagnostics.Add(domain.LevelInfo, "enrichment", source, "Lootr: configuración aplicada")

	convertible := convertibleContainers(in, cfg)
	tableBL, modBL, structBL := set(cfg.LootTableBlacklist), set(cfg.LootModidBlacklist), set(cfg.StructureBlacklist)
	decayTables, decayMods := set(cfg.DecayLootTables), set(cfg.DecayModids)
	refreshTables, refreshMods := set(cfg.RefreshLootTables), set(cfg.RefreshModids)

	claims.Annotate(func(s domain.LootSource) []domain.Note {
		if s.Kind != domain.KindContainer {
			return nil
		}
		if s.Container != "" && !convertible[s.Container] {
			return nil
		}
		table := s.LootTable.String()
		if tableBL[table] || modBL[s.LootTable.Namespace] || (s.Owner.Kind == domain.OwnerStructure && structBL[s.Owner.ID.String()]) {
			return []domain.Note{{Key: NoteBlacklisted, Text: "Compartido: excluido de Lootr por su configuración"}}
		}
		text := "Loot por jugador (Lootr)"
		if s.Container == "" {
			text = "Loot por jugador (Lootr) si es un cofre, barril o vagoneta"
		}
		notes := []domain.Note{{Key: NotePerPlayer, Text: text}}
		ns := s.LootTable.Namespace
		if cfg.RefreshAll || refreshTables[table] || refreshMods[ns] {
			notes = append(notes, domain.Note{Key: NoteRefresh, Text: "Se rellena cada " + minutes(cfg.RefreshValue)})
		}
		if cfg.DecayAll || decayTables[table] || decayMods[ns] {
			notes = append(notes, domain.Note{Key: NoteDecay, Text: "Desaparece " + minutes(cfg.DecayValue) + " después de abrirse"})
		}
		return notes
	})
	return nil
}

// convertibleContainers lists the blocks and entities Lootr 0.7.x replaces.
func convertibleContainers(in discovery.Input, cfg config) map[string]bool {
	out := set([]string{"minecraft:chest", "minecraft:barrel", "minecraft:trapped_chest", "minecraft:shulker_box"})
	if cfg.ConvertMineshafts {
		out["minecraft:chest_minecart"] = true
	}
	if cfg.ConvertQuark {
		for _, wood := range []string{"oak", "spruce", "birch", "jungle", "acacia", "dark_oak", "warped", "crimson"} {
			out["quark:"+wood+"_chest"] = true
			out["quark:"+wood+"_trapped_chest"] = true
		}
		out["quark:nether_brick_chest"], out["quark:purpur_chest"] = true, true
	}
	if cfg.ConvertWoodenChests {
		for _, b := range in.Resources.Tag(resources.TypeBlockTag, domain.MustParseResourceID("forge:chests/wooden")) {
			out[b.String()] = true
		}
	}
	if cfg.ConvertTrapped {
		for _, b := range in.Resources.Tag(resources.TypeBlockTag, domain.MustParseResourceID("forge:chests/trapped")) {
			out[b.String()] = true
		}
	}
	for _, b := range append(cfg.AdditionalChests, cfg.AdditionalTrapped...) {
		out[b] = true
	}
	return out
}

func set(values []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		out[strings.TrimSpace(v)] = true
	}
	return out
}

func minutes(ticks int) string {
	m := float64(ticks) / 20 / 60
	if m == float64(int(m)) {
		if m == 1 {
			return "1 minuto"
		}
		return fmt.Sprintf("%d minutos", int(m))
	}
	return fmt.Sprintf("%.1f minutos", m)
}
