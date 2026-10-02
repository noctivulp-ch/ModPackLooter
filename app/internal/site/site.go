// Package site renders the static website from an analysis result. Every
// link is relative and every page is a folder with index.html, so the site
// works from file://, GitHub Pages and GitLab Pages alike.
package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed assets/*
var assetFS embed.FS

// Metals offered by Wulfenite UI.
var Metals = []string{"oro", "jade", "peltre"}

// Options configures the generated site.
type Options struct {
	OutDir     string
	Title      string
	Lang       string // lang file code for names, e.g. "es_ar"; empty = the modpack's
	Metal      string // "oro", "jade" or "peltre"
	AppVersion string
}

// Stats reports what was written.
type Stats struct {
	Pages  int
	Items  int
	Owners int
	Biomes int
	Tables int
}

// page is what every template receives.
type page struct {
	M     *Model
	Root  string // relative path from the page to the site root
	Title string
	Nav   string
	Data  any
}

// Build writes the site to opts.OutDir.
func Build(res *analysis.Result, opts Options) (Stats, error) {
	if opts.Title == "" {
		opts.Title = "Guía de loot"
	}
	if opts.Lang == "" {
		opts.Lang = res.Modpack.Lang
	}
	if !validMetal(opts.Metal) {
		opts.Metal = "oro"
	}
	m := buildModel(res, opts)
	r, err := newRenderer(opts.OutDir)
	if err != nil {
		return Stats{}, err
	}

	r.render("home", "", "inicio", opts.Title, m, nil)
	r.render("items", "objetos/", "objetos", "Objetos", m, m.Items)
	for _, it := range m.Items {
		r.render("item", it.URL, "objetos", it.Name, m, it)
	}
	r.render("owners", "estructuras/", "estructuras", "Estructuras", m, groupOwners(m.Owners))
	for _, o := range m.Owners {
		r.render("owner", o.URL, "estructuras", o.Name, m, o)
	}
	r.render("biomes", "biomas/", "biomas", "Biomas", m, groupBiomes(m.Biomes))
	for _, b := range m.Biomes {
		r.render("biome", b.URL, "biomas", b.Name, m, b)
	}
	r.render("sources", "fuentes/", "fuentes", "Otras fuentes", m, unownedGroups(m))
	for _, t := range m.Tables {
		r.render("table", t.URL, "fuentes", t.Name, m, t)
	}
	r.renderLostCities(m)
	r.renderFishing(m)
	if len(m.Changes) > 0 {
		r.render("changes", "cambios/", "cambios", "Cambios de mods", m, m.Changes)
	}
	r.render("mods", "mods/", "mods", "Mods", m, m.Mods)
	for _, md := range m.Mods {
		r.render("mod", md.URL, "mods", md.Name, m, md)
	}
	r.render("about", "acerca/", "acerca", "Acerca de este sitio", m, nil)
	if r.err != nil {
		return Stats{}, r.err
	}

	if err := r.writeAssets(m); err != nil {
		return Stats{}, err
	}
	return Stats{Pages: r.pages, Items: len(m.Items), Owners: len(m.Owners), Biomes: len(m.Biomes), Tables: len(m.Tables)}, nil
}

func validMetal(m string) bool {
	for _, x := range Metals {
		if x == m {
			return true
		}
	}
	return false
}

type renderer struct {
	out   string
	sets  map[string]*template.Template
	pages int
	err   error
}

func newRenderer(out string) (*renderer, error) {
	if out == "" {
		return nil, fmt.Errorf("falta la carpeta de salida")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	base, err := template.New("layout.html").Funcs(funcs).ParseFS(templateFS, "templates/layout.html", "templates/parts.html")
	if err != nil {
		return nil, err
	}
	sets := map[string]*template.Template{}
	entries, err := fs.Glob(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := strings.TrimSuffix(path.Base(e), ".html")
		if name == "layout" || name == "parts" {
			continue
		}
		t, err := template.Must(base.Clone()).ParseFS(templateFS, e)
		if err != nil {
			return nil, err
		}
		sets[name] = t
	}
	return &renderer{out: out, sets: sets}, nil
}

func (r *renderer) render(name, url, nav, title string, m *Model, data any) {
	if r.err != nil {
		return
	}
	depth := strings.Count(url, "/")
	p := page{M: m, Root: strings.Repeat("../", depth), Title: title, Nav: nav, Data: data}
	var buf bytes.Buffer
	if err := r.sets[name].ExecuteTemplate(&buf, "layout.html", p); err != nil {
		r.err = fmt.Errorf("página %s (%s): %w", name, url, err)
		return
	}
	dir := filepath.Join(r.out, filepath.FromSlash(url))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.err = err
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), buf.Bytes(), 0o644); err != nil {
		r.err = err
		return
	}
	r.pages++
}

// searchEntry is one row of the client-side search index.
type searchEntry struct {
	Type string `json:"t"` // o: objeto, e: estructura, b: bioma, m: mod
	Name string `json:"n"`
	ID   string `json:"i"`
	URL  string `json:"u"`
	Sub  string `json:"s,omitempty"`
}

func (r *renderer) writeAssets(m *Model) error {
	dir := filepath.Join(r.out, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"style.css", "app.js"} {
		data, err := assetFS.ReadFile("assets/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	var idx []searchEntry
	for _, it := range m.Items {
		sub := it.Mod.Name
		if len(it.Sources) > 0 {
			sub += " · mejor " + pct(it.Best)
		}
		idx = append(idx, searchEntry{"o", it.Name, it.ID.String(), it.URL + "index.html", sub})
	}
	for _, o := range m.Owners {
		idx = append(idx, searchEntry{"e", o.Name, o.ID.String(), o.URL + "index.html", o.Mod.Name})
	}
	for _, b := range m.Biomes {
		idx = append(idx, searchEntry{"b", b.Name, b.ID.String(), b.URL + "index.html", b.Mod.Name})
	}
	idx = append(idx, lcSearch(m)...)
	for _, md := range m.Mods {
		idx = append(idx, searchEntry{"m", md.Name, md.ID.Namespace, md.URL + "index.html", ""})
	}
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	// A script, not a fetched JSON: fetch() does not work from file://.
	js := append([]byte("window.MPL_INDEX="), data...)
	js = append(js, []byte(";\n")...)
	if err := os.WriteFile(filepath.Join(dir, "search-index.js"), js, 0o644); err != nil {
		return err
	}
	diag, err := json.MarshalIndent(struct {
		App         string              `json:"app"`
		Generated   string              `json:"generated"`
		Pack        PackInfo            `json:"pack"`
		Plan        []string            `json:"plan"`
		Enrichments []string            `json:"enrichments"`
		Diagnostics []domain.Diagnostic `json:"diagnostics"`
	}{m.AppVersion, time.Now().UTC().Format(time.RFC3339), m.Pack, m.Plan, m.Enrichments, m.Diagnostics}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(r.out, "acerca", "diagnostics.json"), diag, 0o644); err != nil {
		return err
	}
	// GitHub Pages: serve folders starting with "_" too.
	return os.WriteFile(filepath.Join(r.out, ".nojekyll"), nil, 0o644)
}

// pct formats a probability the Spanish way: "23 %", "4,5 %", "0,3 %", "<0,1 %".
func pct(p float64) string {
	v := p * 100
	switch {
	case v <= 0:
		return "0 %"
	case v < 0.1:
		return "<0,1 %"
	case v >= 99.95:
		return "100 %"
	case v < 10:
		return strings.Replace(fmt.Sprintf("%.1f %%", v), ".", ",", 1)
	default:
		return fmt.Sprintf("%.0f %%", math.Round(v))
	}
}

func count(min, max float64) string {
	f := func(x float64) string {
		if x == math.Trunc(x) {
			return fmt.Sprintf("%.0f", x)
		}
		return strings.Replace(fmt.Sprintf("%.1f", x), ".", ",", 1)
	}
	if min == max {
		return f(min)
	}
	return f(min) + "–" + f(max)
}

var kindLabels = map[domain.SourceKind]string{
	domain.KindContainer:   "Contenedor",
	domain.KindArchaeology: "Arqueología",
	domain.KindEntity:      "Criatura",
	domain.KindFishing:     "Pesca",
	domain.KindGameplay:    "Jugabilidad",
	domain.KindBlock:       "Bloque",
	domain.KindUnknown:     "Otro",
}

var kindOrder = []domain.SourceKind{domain.KindContainer, domain.KindArchaeology, domain.KindEntity, domain.KindFishing, domain.KindGameplay, domain.KindUnknown}

var confInfo = map[domain.Confidence][2]string{
	domain.ConfidenceExact:     {"Exacta", "Leído directamente de los datos del juego o del mod."},
	domain.ConfidenceManual:    {"Manual", "Indicado a mano en la configuración del sitio."},
	domain.ConfidenceKnown:     {"Conocida", "Asignado por el código del juego; dato incluido en ModPackLooter."},
	domain.ConfidenceHeuristic: {"Heurística", "Deducido por el nombre de la tabla; puede no ser exacto."},
	domain.ConfidenceUnknown:   {"Sin origen", "Se sabe qué contiene la tabla, pero no dónde aparece."},
}

var funcs = template.FuncMap{
	"rel": func(root, url string) string {
		if strings.HasSuffix(url, "/") || url == "" {
			return root + url + "index.html"
		}
		return root + url
	},
	"pct":   pct,
	"count": count,
	"width": func(p float64) string { return fmt.Sprintf("%.1f%%", math.Max(1.5, math.Min(100, p*100))) },
	"kind":  func(k domain.SourceKind) string { return kindLabels[k] },
	"conf":  func(c domain.Confidence) string { return confInfo[c][0] },
	"confHelp": func(c domain.Confidence) string {
		return confInfo[c][1]
	},
	"confClass": func(c domain.Confidence) string {
		switch c {
		case domain.ConfidenceExact, domain.ConfidenceManual:
			return "ok"
		case domain.ConfidenceKnown:
			return "known"
		case domain.ConfidenceHeuristic:
			return "warn"
		default:
			return "none"
		}
	},
	"confs": func() []domain.Confidence {
		return []domain.Confidence{domain.ConfidenceExact, domain.ConfidenceKnown, domain.ConfidenceManual, domain.ConfidenceHeuristic, domain.ConfidenceUnknown}
	},
	"short": func(s string) string { return strings.TrimPrefix(s, "minecraft:") },
	"join":  func(s []string) string { return strings.Join(s, " · ") },
	"limit": func(n int, s any) any {
		switch v := s.(type) {
		case []*ItemSource:
			if len(v) > n {
				return v[:n]
			}
		case []*Biome:
			if len(v) > n {
				return v[:n]
			}
		case []*Change:
			if len(v) > n {
				return v[:n]
			}
		}
		return s
	},
	"plus": func(a, b int) int { return a + b },
	"dict": func(kv ...any) (map[string]any, error) {
		if len(kv)%2 != 0 {
			return nil, fmt.Errorf("dict: número impar de argumentos")
		}
		m := map[string]any{}
		for i := 0; i < len(kv); i += 2 {
			k, ok := kv[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict: clave no textual")
			}
			m[k] = kv[i+1]
		}
		return m, nil
	},
	"metals": func() []string { return Metals },
}

// ownerGroup groups owners by mod for the structures page.
type ownerGroup struct {
	Mod    *Mod
	Owners []*Owner
}

func groupOwners(owners []*Owner) []ownerGroup {
	var real, templates []*Owner
	for _, o := range owners {
		if o.Template {
			templates = append(templates, o)
		} else {
			real = append(real, o)
		}
	}
	var out []ownerGroup
	idx := map[*Mod]int{}
	for _, o := range real {
		i, ok := idx[o.Mod]
		if !ok {
			i = len(out)
			idx[o.Mod] = i
			out = append(out, ownerGroup{Mod: o.Mod})
		}
		out[i].Owners = append(out[i].Owners, o)
	}
	sortRefs(out, func(g ownerGroup) Ref { return g.Mod.Ref })
	if len(templates) > 0 {
		out = append(out, ownerGroup{Mod: &Mod{Ref: Ref{Name: "Plantillas sin estructura conocida"}}, Owners: templates})
	}
	return out
}

type biomeGroup struct {
	Mod    *Mod
	Biomes []*Biome
}

func groupBiomes(biomes []*Biome) []biomeGroup {
	var out []biomeGroup
	idx := map[*Mod]int{}
	for _, b := range biomes {
		i, ok := idx[b.Mod]
		if !ok {
			i = len(out)
			idx[b.Mod] = i
			out = append(out, biomeGroup{Mod: b.Mod})
		}
		out[i].Biomes = append(out[i].Biomes, b)
	}
	sortRefs(out, func(g biomeGroup) Ref { return g.Mod.Ref })
	return out
}

type kindGroup struct {
	Kind domain.SourceKind
	Uses []*TableUse
}

func unownedGroups(m *Model) []kindGroup {
	var out []kindGroup
	for _, k := range kindOrder {
		if uses := m.Unowned[k]; len(uses) > 0 {
			out = append(out, kindGroup{Kind: k, Uses: uses})
		}
	}
	return out
}
