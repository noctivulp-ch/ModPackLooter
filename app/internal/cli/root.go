// Package cli is the command-line adapter. It only parses input, calls the
// core and presents results; no business logic lives here.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
)

// Deps are the core services the CLI needs, injected by the composition root.
type Deps struct {
	Version   string
	Discovery *discovery.Registry[discovery.Discoverer]
}

// NewRootCommand builds the modpacklooter command tree.
func NewRootCommand(deps Deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "modpacklooter",
		Short:         "Genera un sitio web estático con todas las fuentes de loot de un modpack",
		Version:       deps.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newPlanCommand(deps))
	return root
}

func newPlanCommand(deps Deps) *cobra.Command {
	var (
		version string
		loader  string
		mods    []string
		asJSON  bool
	)
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Muestra qué descubridores se ejecutarán y en qué orden para un modpack",
		Long: "Muestra el plan de la puerta de descubrimiento: los descubridores que aplican\n" +
			"a la versión, cargador y mods indicados, en el orden en que se ejecutarán.",
		Example: "  modpacklooter plan --mc-version 1.20.1 --loader forge --mod lootr",
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, err := mcversion.Parse(version)
			if err != nil {
				return err
			}
			target := discovery.Target{Version: v, Loader: domain.Loader(strings.ToLower(loader)), Mods: map[string]bool{}}
			for _, m := range mods {
				target.Mods[m] = true
			}
			plan, err := deps.Discovery.Plan(target)
			if err != nil {
				return err
			}
			if asJSON {
				return writePlanJSON(cmd.OutOrStdout(), plan)
			}
			return writePlanText(cmd.OutOrStdout(), target, plan)
		},
	}
	cmd.Flags().StringVar(&version, "mc-version", "1.20.1", "versión de Minecraft (p. ej. 1.20.1, 1.21.1, 26.3)")
	cmd.Flags().StringVar(&loader, "loader", string(domain.LoaderForge), "cargador: forge, neoforge o fabric")
	cmd.Flags().StringSliceVar(&mods, "mod", nil, "ID de un mod presente en el modpack (repetible)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "salida en JSON")
	return cmd
}

type planStep struct {
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Priority int    `json:"priority"`
	Versions string `json:"versions"`
}

func steps(plan discovery.Plan[discovery.Discoverer]) []planStep {
	out := make([]planStep, 0, len(plan.Steps))
	for _, d := range plan.Steps {
		desc := d.Descriptor()
		out = append(out, planStep{ID: desc.ID, Phase: desc.Phase.String(), Priority: desc.Priority, Versions: desc.Applies.Versions.String()})
	}
	return out
}

func writePlanJSON(w io.Writer, plan discovery.Plan[discovery.Discoverer]) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(steps(plan))
}

func writePlanText(w io.Writer, t discovery.Target, plan discovery.Plan[discovery.Discoverer]) error {
	if _, err := fmt.Fprintf(w, "Plan de descubrimiento para Minecraft %s (%s)\n\n", t.Version, t.Loader); err != nil {
		return err
	}
	for i, s := range steps(plan) {
		if _, err := fmt.Fprintf(w, "%2d. %-24s fase %-11s versiones %s\n", i+1, s.ID, s.Phase, s.Versions); err != nil {
			return err
		}
	}
	return nil
}
