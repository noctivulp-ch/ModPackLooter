// Package cli is the command-line adapter. It only parses input, calls the
// core and presents results; no business logic lives here.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/modpack"
	"github.com/EnierAragon/ModPackLooter/app/internal/site"
)

// Deps are the core services the CLI needs, injected by the composition root.
type Deps struct {
	Version  string
	Analyzer analysis.Analyzer
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
	root.AddCommand(newBuildCommand(deps), newScanCommand(deps), newPlanCommand(deps))
	return root
}

// packFlags are the options shared by commands that open a modpack.
type packFlags struct {
	mcVersion    string
	loader       string
	minecraftJar string
	assetsDir    string
	datapacks    []string
	lang         string
	world        string
}

func (f *packFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.mcVersion, "mc-version", "", "versión de Minecraft si no se detecta (p. ej. 1.20.1)")
	cmd.Flags().StringVar(&f.loader, "loader", "", "cargador si no se detecta: forge, neoforge o fabric")
	cmd.Flags().StringVar(&f.minecraftJar, "minecraft-jar", "", "jar de Minecraft vanilla (o carpeta de datos) para incluir el loot vanilla")
	cmd.Flags().StringSliceVar(&f.datapacks, "datapack", nil, "datapack extra (carpeta o .zip); repetible")
	cmd.Flags().StringVar(&f.assetsDir, "assets-dir", "", "carpeta assets del launcher, para traducir los nombres vanilla")
	cmd.Flags().StringVar(&f.world, "world", "", "mundo modelo creado con el modpack (carpeta con level.dat o nombre dentro de saves/): usa sus opciones de generación")
	cmd.Flags().StringVar(&f.lang, "lang", "", "idioma de los nombres (es_es, es_ar, en_us…); por defecto el del juego (options.txt) o es_es")
}

func (f *packFlags) options(path string, diags *domain.Diagnostics) modpack.Options {
	return modpack.Options{
		Path: path, MCVersion: f.mcVersion, Loader: f.loader, MinecraftJar: f.minecraftJar,
		Datapacks: f.datapacks, AssetsDir: f.assetsDir, Lang: f.lang, World: f.world, Diagnostics: diags,
	}
}

// stderrProgress prints one line per stage on stderr, keeping stdout clean.
type stderrProgress struct{ w io.Writer }

func (p stderrProgress) Stage(name string) { fmt.Fprintf(p.w, "· %s…\n", name) }

func newBuildCommand(deps Deps) *cobra.Command {
	var (
		pf    packFlags
		out   string
		title string
		metal string
	)
	cmd := &cobra.Command{
		Use:   "build <carpeta-del-modpack>",
		Short: "Analiza el modpack y genera el sitio estático",
		Example: "  modpacklooter build ~/curseforge/minecraft/Instances/DeceasedCraft --out site\n" +
			"  modpacklooter build ./instancia --minecraft-jar ~/.minecraft/versions/1.20.1/1.20.1.jar --metal jade",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				title = "Guía de loot · " + filepath.Base(filepath.Clean(args[0]))
			}
			diags := &domain.Diagnostics{}
			res, err := deps.Analyzer.Run(context.Background(), pf.options(args[0], diags), stderrProgress{cmd.ErrOrStderr()})
			if err != nil {
				return err
			}
			defer res.Close()
			fmt.Fprintln(cmd.ErrOrStderr(), "· Generando el sitio…")
			stats, err := site.Build(res, site.Options{OutDir: out, Title: title, Metal: metal, AppVersion: deps.Version})
			if err != nil {
				return fmt.Errorf("no se pudo generar el sitio: %w", err)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "✓ Sitio generado en %s\n", out)
			fmt.Fprintf(w, "  %d páginas · %d objetos · %d estructuras · %d biomas · %d tablas\n", stats.Pages, stats.Items, stats.Owners, stats.Biomes, stats.Tables)
			printDiagnosticSummary(w, diags)
			fmt.Fprintf(w, "  Ábrelo con: %s\n", filepath.Join(out, "index.html"))
			return nil
		},
	}
	pf.register(cmd)
	cmd.Flags().StringVarP(&out, "out", "o", "site", "carpeta de salida")
	cmd.Flags().StringVar(&title, "title", "", "título del sitio (por defecto, el nombre de la carpeta)")
	cmd.Flags().StringVar(&metal, "metal", "oro", "metal de Wulfenite UI: oro, jade o peltre")
	return cmd
}

func newScanCommand(deps Deps) *cobra.Command {
	var (
		pf     packFlags
		asJSON bool
	)
	cmd := &cobra.Command{
		Use:   "scan <carpeta-del-modpack>",
		Short: "Analiza el modpack y muestra un resumen y los diagnósticos, sin generar el sitio",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			diags := &domain.Diagnostics{}
			res, err := deps.Analyzer.Run(context.Background(), pf.options(args[0], diags), stderrProgress{cmd.ErrOrStderr()})
			if err != nil {
				return err
			}
			defer res.Close()
			summary := summarize(res)
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(summary)
			}
			writeSummary(cmd.OutOrStdout(), summary)
			printDiagnosticSummary(cmd.OutOrStdout(), diags)
			for _, d := range diags.Items() {
				if d.Level >= domain.LevelWarning {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s [%s] %s: %s\n", symbol(d.Level), d.Stage, d.Source, d.Message)
				}
			}
			return nil
		},
	}
	pf.register(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "salida en JSON")
	return cmd
}

type scanSummary struct {
	Root          string               `json:"root"`
	MCVersion     string               `json:"mcVersion"`
	VersionSource string               `json:"versionSource"`
	Lang          string               `json:"lang"`
	World         string               `json:"world,omitempty"`
	Disablers     []string             `json:"disablers"`
	Disabled      int                  `json:"disabled"`
	Possibly      int                  `json:"possiblyDisabled"`
	Disablements  []domain.Disablement `json:"disablements"`
	LangSource    string               `json:"langSource"`
	Loader        string               `json:"loader"`
	Mods          int                  `json:"mods"`
	HasVanilla    bool                 `json:"vanilla"`
	LootTables    int                  `json:"lootTables"`
	Structures    int                  `json:"structures"`
	Sources       int                  `json:"sources"`
	ByConfidence  map[string]int       `json:"byConfidence"`
	ByDiscoverer  map[string]int       `json:"byDiscoverer"`
	Plan          []string             `json:"plan"`
	Enrichments   []string             `json:"enrichments"`
	Diagnostics   []domain.Diagnostic  `json:"diagnostics"`
}

func summarize(res *analysis.Result) scanSummary {
	s := scanSummary{
		Root: res.Modpack.Root, MCVersion: res.Modpack.MCVersion.String(), VersionSource: res.Modpack.VersionSource,
		Lang: res.Modpack.Lang, LangSource: res.Modpack.LangSource,
		Disablers: res.Disablers, Disablements: res.Disablements,
		Loader: string(res.Modpack.Loader), Mods: len(res.Modpack.Mods), HasVanilla: res.Modpack.HasVanilla,
		LootTables: len(res.Tables), Structures: len(res.Structures), Sources: len(res.Sources),
		ByConfidence: map[string]int{}, ByDiscoverer: map[string]int{},
		Plan: res.Plan, Enrichments: res.Enrichments, Diagnostics: res.Diagnostics.Items(),
	}
	if res.Modpack.World != nil {
		s.World = res.Modpack.World.Name
	}
	for _, st := range res.Statuses {
		switch st.Certainty {
		case domain.Certainly:
			s.Disabled++
		case domain.Possibly:
			s.Possibly++
		}
	}
	for _, src := range res.Sources {
		s.ByConfidence[src.Confidence.String()]++
		for _, e := range src.Evidence {
			s.ByDiscoverer[e.DiscoveredBy]++
		}
	}
	return s
}

func writeSummary(w io.Writer, s scanSummary) {
	vanilla := "incluido"
	if !s.HasVanilla {
		vanilla = "NO incluido (usa --minecraft-jar)"
	}
	fmt.Fprintf(w, "Modpack     %s\n", s.Root)
	fmt.Fprintf(w, "Minecraft   %s (%s) · %s\n", s.MCVersion, s.VersionSource, s.Loader)
	fmt.Fprintf(w, "Idioma      %s (%s)\n", s.Lang, s.LangSource)
	world := "ninguno (valores por defecto del modpack)"
	if s.World != "" {
		world = s.World
	}
	fmt.Fprintf(w, "Mundo       %s\n", world)
	fmt.Fprintf(w, "Mods        %d\n", s.Mods)
	fmt.Fprintf(w, "Vanilla     %s\n", vanilla)
	fmt.Fprintf(w, "Loot tables %d\n", s.LootTables)
	fmt.Fprintf(w, "Estructuras %d\n", s.Structures)
	fmt.Fprintf(w, "Fuentes     %d\n", s.Sources)
	fmt.Fprintln(w, "\nPor confianza:")
	for _, k := range sortedKeys(s.ByConfidence) {
		fmt.Fprintf(w, "  %-12s %d\n", k, s.ByConfidence[k])
	}
	fmt.Fprintln(w, "\nPor descubridor:")
	for _, k := range sortedKeys(s.ByDiscoverer) {
		fmt.Fprintf(w, "  %-22s %d\n", k, s.ByDiscoverer[k])
	}
	if len(s.Enrichments) > 0 {
		fmt.Fprintf(w, "\nAjustes aplicados: %s\n", strings.Join(s.Enrichments, ", "))
	}
	fmt.Fprintf(w, "\nDesactivados: %d seguro · %d posible\n", s.Disabled, s.Possibly)
	for _, d := range s.Disablements {
		mark := "!"
		if d.Certainty == domain.Certainly {
			mark = "⊘"
		}
		fmt.Fprintf(w, "  %s %-9s %-40s %s\n", mark, d.Target.Kind, d.Target.ID, d.Reason)
	}
	fmt.Fprintln(w)
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func symbol(l domain.Level) string {
	switch l {
	case domain.LevelError:
		return "✗"
	case domain.LevelWarning:
		return "!"
	default:
		return "·"
	}
}

func printDiagnosticSummary(w io.Writer, diags *domain.Diagnostics) {
	errs, warns := diags.Count(domain.LevelError), diags.Count(domain.LevelWarning)
	if errs+warns == 0 {
		fmt.Fprintln(w, "  Sin avisos.")
		return
	}
	fmt.Fprintf(w, "  %d errores y %d avisos (detalle en el sitio: acerca/ o con 'scan')\n", errs, warns)
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
		Short: "Muestra qué descubridores y ajustes se ejecutarán y en qué orden",
		Long: "Muestra el plan de la puerta de descubrimiento y de la de enriquecimiento:\n" +
			"los plugins que aplican a la versión, cargador y mods indicados, en orden.",
		Example: "  modpacklooter plan --mc-version 1.20.1 --loader forge --mod lootr --mod lostcities",
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, err := mcversion.Parse(version)
			if err != nil {
				return err
			}
			target := discovery.Target{Version: v, Loader: domain.Loader(strings.ToLower(loader)), Mods: map[string]bool{}}
			for _, m := range mods {
				target.Mods[m] = true
			}
			dplan, err := deps.Analyzer.Discoverers.Plan(target)
			if err != nil {
				return err
			}
			eplan, err := deps.Analyzer.Enrichers.Plan(target)
			if err != nil {
				return err
			}
			xplan, err := deps.Analyzer.Disablers.Plan(target)
			if err != nil {
				return err
			}
			steps := describe(dplan.Steps, "descubrimiento")
			steps = append(steps, describe(eplan.Steps, "enriquecimiento")...)
			steps = append(steps, describe(xplan.Steps, "desactivadores")...)
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(steps)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Plan para Minecraft %s (%s)\n\n", target.Version, target.Loader)
			for i, s := range steps {
				fmt.Fprintf(w, "%2d. %-16s %-22s fase %-11s versiones %s\n", i+1, s.Gate, s.ID, s.Phase, s.Versions)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "mc-version", "1.20.1", "versión de Minecraft (p. ej. 1.20.1, 1.21.1, 26.3)")
	cmd.Flags().StringVar(&loader, "loader", string(domain.LoaderForge), "cargador: forge, neoforge o fabric")
	cmd.Flags().StringSliceVar(&mods, "mod", nil, "ID de un mod presente en el modpack (repetible)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "salida en JSON")
	return cmd
}

type planStep struct {
	Gate     string `json:"gate"`
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Priority int    `json:"priority"`
	Versions string `json:"versions"`
}

func describe[T discovery.Plugin](steps []T, gate string) []planStep {
	out := make([]planStep, 0, len(steps))
	for _, p := range steps {
		d := p.Descriptor()
		out = append(out, planStep{Gate: gate, ID: d.ID, Phase: d.Phase.String(), Priority: d.Priority, Versions: d.Applies.Versions.String()})
	}
	return out
}

// Exit codes keep a stable contract for scripts and CI.
const (
	ExitOK    = 0
	ExitError = 1
)

// Main runs the CLI and returns the process exit code.
func Main(deps Deps, args []string) int {
	root := NewRootCommand(deps)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return ExitError
	}
	return ExitOK
}
