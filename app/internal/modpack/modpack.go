// Package modpack opens a modpack instance folder: it finds the mods,
// datapacks, scripts and vanilla data, detects the Minecraft version and
// loader, and builds the merged resource index.
package modpack

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

const stage = "cargador"

// Options controls how a modpack is opened. Empty fields are auto-detected.
type Options struct {
	Path         string
	MinecraftJar string // vanilla client/server jar, or an unpacked data folder
	MCVersion    string
	Loader       string
	Datapacks    []string // extra datapack folders or zips
	Diagnostics  *domain.Diagnostics
}

// Modpack is an opened modpack, ready to be analysed.
type Modpack struct {
	Root          string // the game directory (the one holding mods/)
	MCVersion     mcversion.Version
	Loader        domain.Loader
	VersionSource string // how the version was detected, for the report
	Mods          []Mod
	Index         *resources.Index
	HasVanilla    bool

	closers []func() error
}

// Close releases every open archive.
func (m *Modpack) Close() error {
	var errs []error
	for _, c := range m.closers {
		errs = append(errs, c())
	}
	return errors.Join(errs...)
}

// ModIDs returns the set of mod IDs present.
func (m *Modpack) ModIDs() map[string]bool {
	out := map[string]bool{"minecraft": true}
	for _, mod := range m.Mods {
		out[mod.ID] = true
	}
	return out
}

// ReadFile reads a file relative to the game directory (e.g. config files).
func (m *Modpack) ReadFile(rel string) ([]byte, error) {
	if strings.Contains(rel, "..") {
		return nil, fmt.Errorf("ruta no válida: %s", rel)
	}
	return os.ReadFile(filepath.Join(m.Root, filepath.FromSlash(rel)))
}

// Open loads the modpack described by opts.
func Open(opts Options) (*Modpack, error) {
	diags := opts.Diagnostics
	if diags == nil {
		diags = &domain.Diagnostics{}
	}
	root, err := findGameDir(opts.Path)
	if err != nil {
		return nil, err
	}
	mp := &Modpack{Root: root}

	modPacks, err := mp.loadMods(diags)
	if err != nil {
		return nil, err
	}
	if err := mp.detectTarget(opts); err != nil {
		mp.Close()
		return nil, err
	}
	layout := resources.LayoutFor(mp.MCVersion)
	if layout == nil {
		mp.Close()
		return nil, fmt.Errorf("Minecraft %s no está soportado todavía (soportado: 1.20 en adelante)", mp.MCVersion)
	}

	var packs []resources.Pack
	if vanilla := mp.loadVanilla(opts.MinecraftJar, diags); vanilla != nil {
		packs = append(packs, vanilla)
		mp.HasVanilla = true
	} else {
		diags.Add(domain.LevelWarning, stage, "", "no se encontró el jar de Minecraft %s: el loot vanilla no se incluirá. Usa --minecraft-jar <ruta>", mp.MCVersion)
	}
	packs = append(packs, modPacks...)
	packs = append(packs, mp.loadDatapacks(opts.Datapacks, diags)...)
	if p := mp.loadScripts(diags); p != nil {
		packs = append(packs, p)
	}
	mp.Index = resources.NewIndex(layout, packs)
	return mp, nil
}

// findGameDir accepts the instance folder or its game subfolder.
func findGameDir(path string) (string, error) {
	if path == "" {
		return "", errors.New("falta la ruta del modpack")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("no se puede leer %s: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s no es una carpeta; los .zip/.mrpack aún no están soportados, descomprímelo o usa la carpeta de la instancia", path)
	}
	for _, candidate := range []string{path, filepath.Join(path, ".minecraft"), filepath.Join(path, "minecraft")} {
		if st, err := os.Stat(filepath.Join(candidate, "mods")); err == nil && st.IsDir() {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("no se encontró la carpeta mods/ en %s; indica la carpeta de la instancia del modpack", path)
}

func (mp *Modpack) loadMods(diags *domain.Diagnostics) ([]resources.Pack, error) {
	dir := filepath.Join(mp.Root, "mods")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %s: %w", dir, err)
	}
	var packs []resources.Pack
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.EqualFold(filepath.Ext(name), ".jar") {
			continue
		}
		zp, err := resources.OpenZipPack(filepath.Join(dir, name), resources.KindMod)
		if err != nil {
			diags.Add(domain.LevelWarning, stage, name, "jar ilegible: %v", err)
			continue
		}
		mp.closers = append(mp.closers, zp.Close)
		packs = append(packs, mp.registerJar(zp, name, diags, 0)...)
	}
	return packs, nil
}

// registerJar records the mods of a jar and returns it plus any nested jars
// (jar-in-jar) that carry game data.
func (mp *Modpack) registerJar(zp *resources.ZipPack, file string, diags *domain.Diagnostics, depth int) []resources.Pack {
	mods, err := readMods(zp, file)
	if err != nil {
		diags.Add(domain.LevelWarning, stage, file, "metadatos del mod ilegibles: %v", err)
	}
	mp.Mods = append(mp.Mods, mods...)
	packs := []resources.Pack{zp}
	if depth > 2 {
		return packs
	}
	for _, path := range zp.Files() {
		if !strings.HasSuffix(path, ".jar") || !(strings.HasPrefix(path, "META-INF/jarjar/") || strings.HasPrefix(path, "META-INF/jars/")) {
			continue
		}
		data, err := resources.ReadFile(zp, path)
		if err != nil {
			diags.Add(domain.LevelWarning, stage, file, "jar anidado ilegible %s: %v", path, err)
			continue
		}
		nestedName := file + "!/" + filepath.Base(path)
		nested, err := resources.NewZipPackFromBytes(nestedName, resources.KindMod, data)
		if err != nil {
			diags.Add(domain.LevelWarning, stage, file, "jar anidado ilegible %s: %v", path, err)
			continue
		}
		packs = append(packs, mp.registerJar(nested, nestedName, diags, depth+1)...)
	}
	return packs
}

var reVersion = regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

func (mp *Modpack) detectTarget(opts Options) error {
	if opts.MCVersion != "" {
		v, err := mcversion.Parse(opts.MCVersion)
		if err != nil {
			return err
		}
		mp.MCVersion, mp.VersionSource = v, "--mc-version"
	}
	if opts.Loader != "" {
		mp.Loader = domain.Loader(strings.ToLower(opts.Loader))
	}

	// CurseForge app instance.
	var cf struct {
		GameVersion   string `json:"gameVersion"`
		BaseModLoader struct {
			Name string `json:"name"`
		} `json:"baseModLoader"`
	}
	if data, err := os.ReadFile(filepath.Join(mp.Root, "minecraftinstance.json")); err == nil && json.Unmarshal(data, &cf) == nil {
		mp.setVersion(cf.GameVersion, "minecraftinstance.json")
		mp.setLoader(loaderFromName(cf.BaseModLoader.Name))
	}

	// Prism Launcher / MultiMC instance.
	var mmc struct {
		Components []struct {
			UID     string `json:"uid"`
			Version string `json:"version"`
		} `json:"components"`
	}
	if data, err := os.ReadFile(filepath.Join(filepath.Dir(mp.Root), "mmc-pack.json")); err == nil && json.Unmarshal(data, &mmc) == nil {
		for _, c := range mmc.Components {
			switch c.UID {
			case "net.minecraft":
				mp.setVersion(c.Version, "mmc-pack.json")
			case "net.minecraftforge":
				mp.setLoader(domain.LoaderForge)
			case "net.neoforged":
				mp.setLoader(domain.LoaderNeoForge)
			case "net.fabricmc.fabric-loader":
				mp.setLoader(domain.LoaderFabric)
			}
		}
	}

	// Fall back to what the mods declare.
	votes := map[string]int{}
	loaders := map[domain.Loader]int{}
	for _, m := range mp.Mods {
		loaders[m.Loader]++
		if v := reVersion.FindString(m.MinecraftRange); v != "" {
			votes[v]++
		}
	}
	if best := mostVoted(votes); best != "" {
		mp.setVersion(best, "dependencias de los mods")
	}
	var bestLoader domain.Loader
	for l, n := range loaders {
		if bestLoader == "" || n > loaders[bestLoader] || (n == loaders[bestLoader] && l < bestLoader) {
			bestLoader = l
		}
	}
	mp.setLoader(bestLoader)

	if mp.VersionSource == "" {
		return errors.New("no se pudo detectar la versión de Minecraft; indícala con --mc-version")
	}
	if mp.Loader == "" {
		mp.Loader = domain.LoaderForge
	}
	return nil
}

func (mp *Modpack) setVersion(s, source string) {
	if mp.VersionSource != "" || s == "" {
		return
	}
	if v, err := mcversion.Parse(s); err == nil {
		mp.MCVersion, mp.VersionSource = v, source
	}
}

func (mp *Modpack) setLoader(l domain.Loader) {
	if mp.Loader == "" && l != "" {
		mp.Loader = l
	}
}

func loaderFromName(name string) domain.Loader {
	switch {
	case strings.HasPrefix(name, "neoforge"):
		return domain.LoaderNeoForge
	case strings.HasPrefix(name, "forge"):
		return domain.LoaderForge
	case strings.HasPrefix(name, "fabric"):
		return domain.LoaderFabric
	}
	return ""
}

func mostVoted(votes map[string]int) string {
	keys := make([]string, 0, len(votes))
	for k := range votes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best := ""
	for _, k := range keys {
		if best == "" || votes[k] > votes[best] {
			best = k
		}
	}
	return best
}

func (mp *Modpack) loadVanilla(explicit string, diags *domain.Diagnostics) resources.Pack {
	v := mp.MCVersion.String()
	candidates := []string{explicit}
	if explicit == "" {
		candidates = []string{
			filepath.Join(mp.Root, "versions", v, v+".jar"),
			// CurseForge: <cf>/minecraft/Instances/<name> -> <cf>/minecraft/Install/versions
			filepath.Join(mp.Root, "..", "..", "Install", "versions", v, v+".jar"),
			// Prism/MultiMC: <prism>/instances/<name>/.minecraft -> <prism>/libraries
			filepath.Join(mp.Root, "..", "..", "..", "libraries", "com", "mojang", "minecraft", v, "minecraft-"+v+"-client.jar"),
			// Official launcher.
			filepath.Join(userHome(), ".minecraft", "versions", v, v+".jar"),
		}
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		info, err := os.Stat(c)
		if err != nil {
			if explicit != "" {
				diags.Add(domain.LevelError, stage, c, "no se puede leer el jar de Minecraft: %v", err)
			}
			continue
		}
		if info.IsDir() {
			dp, err := resources.OpenDirPack(c, resources.KindVanilla)
			if err != nil {
				diags.Add(domain.LevelError, stage, c, "%v", err)
				continue
			}
			return dp.WithName("minecraft " + v)
		}
		zp, err := resources.OpenZipPack(c, resources.KindVanilla)
		if err != nil {
			diags.Add(domain.LevelError, stage, c, "%v", err)
			continue
		}
		mp.closers = append(mp.closers, zp.Close)
		diags.Add(domain.LevelInfo, stage, c, "datos vanilla de Minecraft %s cargados", v)
		return zp
	}
	return nil
}

func userHome() string {
	h, _ := os.UserHomeDir()
	return h
}

// Folders that mods such as Paxi, Open Loader or Global Packs load datapacks from.
var datapackDirs = []string{
	"datapacks",
	"global_packs/required_data",
	"global_packs/optional_data",
	"config/paxi/datapacks",
	"config/openloader/data",
}

func (mp *Modpack) loadDatapacks(extra []string, diags *domain.Diagnostics) []resources.Pack {
	var candidates []string
	for _, d := range datapackDirs {
		entries, err := os.ReadDir(filepath.Join(mp.Root, filepath.FromSlash(d)))
		if err != nil {
			continue
		}
		for _, e := range entries {
			candidates = append(candidates, filepath.Join(mp.Root, filepath.FromSlash(d), e.Name()))
		}
	}
	candidates = append(candidates, extra...)

	var packs []resources.Pack
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err != nil {
			diags.Add(domain.LevelWarning, stage, c, "datapack ilegible: %v", err)
			continue
		}
		switch {
		case info.IsDir():
			if _, err := os.Stat(filepath.Join(c, "data")); err != nil {
				continue
			}
			dp, err := resources.OpenDirPack(c, resources.KindDatapack)
			if err != nil {
				diags.Add(domain.LevelWarning, stage, c, "%v", err)
				continue
			}
			packs = append(packs, dp)
		case strings.EqualFold(filepath.Ext(c), ".zip"):
			zp, err := resources.OpenZipPack(c, resources.KindDatapack)
			if err != nil {
				diags.Add(domain.LevelWarning, stage, c, "%v", err)
				continue
			}
			mp.closers = append(mp.closers, zp.Close)
			packs = append(packs, zp)
		}
	}
	return packs
}

// loadScripts adds the data and assets that KubeJS loads from kubejs/.
func (mp *Modpack) loadScripts(diags *domain.Diagnostics) resources.Pack {
	dir := filepath.Join(mp.Root, "kubejs")
	if _, err := os.Stat(filepath.Join(dir, "data")); err != nil {
		if _, err := os.Stat(filepath.Join(dir, "assets")); err != nil {
			return nil
		}
	}
	dp, err := resources.OpenDirPack(dir, resources.KindScript)
	if err != nil {
		diags.Add(domain.LevelWarning, stage, dir, "%v", err)
		return nil
	}
	return dp
}
