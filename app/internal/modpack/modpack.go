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

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/names"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
	"github.com/EnierAragon/ModPackLooter/app/internal/world"
)

const stage = "cargador"

// Options controls how a modpack is opened. Empty fields are auto-detected.
type Options struct {
	Path         string
	MinecraftJar string // vanilla client/server jar, or an unpacked data folder
	MCVersion    string
	Loader       string
	Datapacks    []string // extra datapack folders or zips
	AssetsDir    string   // launcher assets folder (for vanilla translations)
	Lang         string   // language for names; empty = the game's (options.txt), else es_es
	World        string   // model world: a folder with level.dat, or a name under saves/
	Diagnostics  *domain.Diagnostics
}

// Modpack is an opened modpack, ready to be analysed.
type Modpack struct {
	Root          string // the game directory (the one holding mods/)
	MCVersion     mcversion.Version
	Loader        domain.Loader
	VersionSource string       // how the version was detected, for the report
	Lang          string       // language used for names, e.g. "es_ar"
	LangSource    string       // "--lang", "options.txt" or "por defecto"
	World         *world.World // model world, nil when none was given
	WorldAuto     bool         // the world was detected (server.properties), not given
	Mods          []Mod
	Index         *resources.Index
	HasVanilla    bool

	closers []func() error
	jars    []discovery.Jar
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

// Jars lists the mod jars of the mods folder.
func (m *Modpack) Jars() []discovery.Jar { return m.jars }

// ModName names the mod shipped in a pack (a jar file, or a nested jar
// "outer.jar!/inner.jar"); other packs keep their name.
func (m *Modpack) ModName(pack string) string {
	for _, md := range m.Mods {
		if md.File == pack && md.Name != "" {
			return md.Name
		}
	}
	for _, md := range m.Mods {
		if md.File == pack {
			return md.ID
		}
	}
	return pack
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
	mp.Lang, mp.LangSource = detectLang(root, opts.Lang)
	worldPath := ""
	if opts.World != "" {
		worldPath = resolveWorld(root, opts.World)
	} else if sw := serverWorld(root); sw != "" {
		// A dedicated server keeps its world next to mods/: use it as the
		// model world automatically.
		worldPath = sw
		mp.WorldAuto = true
	}
	if worldPath != "" {
		w, err := world.Load(worldPath)
		switch {
		case err == nil:
			mp.World = w
			if mp.WorldAuto {
				diags.Add(domain.LevelInfo, stage, worldPath, "mundo del servidor detectado (server.properties); se usa como mundo modelo")
			}
		case mp.WorldAuto:
			diags.Add(domain.LevelWarning, stage, worldPath, "no se pudo leer el mundo del servidor: %v", err)
			mp.WorldAuto = false
		default:
			return nil, err
		}
	}

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
	if langs := mp.loadAssetLangs(opts.AssetsDir, mp.Lang, diags); langs != nil {
		packs = append(packs, langs)
	}
	packs = append(packs, modPacks...)
	packs = append(packs, mp.loadDatapacks(opts.Datapacks, diags)...)
	packs = append(packs, mp.loadWorldDatapacks(diags)...)
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
		mp.jars = append(mp.jars, discovery.Jar{File: name, Pack: zp})
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
		candidates = append(serverJars(mp.Root, v),
			// Client installs.
			filepath.Join(mp.Root, "versions", v, v+".jar"),
			// CurseForge: <cf>/minecraft/Instances/<name> -> <cf>/minecraft/Install/versions
			filepath.Join(mp.Root, "..", "..", "Install", "versions", v, v+".jar"),
			// Prism/MultiMC: <prism>/instances/<name>/.minecraft -> <prism>/libraries
			filepath.Join(mp.Root, "..", "..", "..", "libraries", "com", "mojang", "minecraft", v, "minecraft-"+v+"-client.jar"),
			// Official launcher.
			filepath.Join(userHome(), ".minecraft", "versions", v, v+".jar"),
		)
	}
	// Jars without data (a loader's launcher renamed server.jar) are only
	// reported when no candidate has the data.
	var skipped []string
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
		pack := unwrapBundler(zp, diags)
		if !hasVanillaData(pack) {
			skipped = append(skipped, c)
			continue
		}
		diags.Add(domain.LevelInfo, stage, c, "datos vanilla de Minecraft %s cargados", v)
		return pack
	}
	for _, c := range skipped {
		diags.Add(domain.LevelWarning, stage, c, "el jar no contiene los datos de Minecraft (data/minecraft/); se ignora")
	}
	return nil
}

// serverJars lists where dedicated servers keep the vanilla server jar (or
// a jar with its data), most specific first:
//   - official server: server.jar (a bundler since 1.18) and, once run,
//     versions/<v>/server-<v>.jar;
//   - Forge 1.17+, NeoForge 1.20.1–1.21.1 and hybrids (Mohist, Arclight…):
//     libraries/net/minecraft/server/<v>-<mcp>/server-<v>-<mcp>-extra.jar
//     (the server's data and assets, without classes);
//   - NeoForge (newer installers): libraries/net/minecraft/server/<v>/
//     server-<v>.jar and the patched libraries/net/neoforged/
//     minecraft-server-patched/<ver>/*.jar;
//   - Fabric and Quilt: .fabric/server/<v>-server.jar, .quilt/server/… and
//     their remapped jars;
//   - Paper, Purpur, Folia: cache/mojang_<v>.jar and versions/<v>/*.jar;
//   - any other jar in the server folder (renamed launchers are skipped
//     because they have no data).
func serverJars(root, v string) []string {
	out := []string{
		filepath.Join(root, "server.jar"),
		filepath.Join(root, "minecraft_server."+v+".jar"),
	}
	globs := []string{
		filepath.Join(root, "libraries", "net", "minecraft", "server", v+"*", "server-"+v+"*-extra.jar"),
		filepath.Join(root, "libraries", "net", "minecraft", "server", v, "server-"+v+".jar"),
		filepath.Join(root, "libraries", "net", "neoforged", "minecraft-server-patched", "*", "minecraft-server-patched-*.jar"),
		filepath.Join(root, "versions", v, "server-"+v+".jar"),
		filepath.Join(root, "versions", v, "*.jar"),
		filepath.Join(root, "cache", "mojang_"+v+".jar"),
		filepath.Join(root, ".fabric", "server", v+"-server.jar"),
		filepath.Join(root, ".fabric", "server", "*.jar"),
		filepath.Join(root, ".fabric", "remappedJars", "minecraft-"+v+"*", "*.jar"),
		filepath.Join(root, ".quilt", "server", "*.jar"),
		filepath.Join(root, ".quilt", "remappedJars", "minecraft-"+v+"*", "*.jar"),
		filepath.Join(root, "*.jar"),
	}
	seen := map[string]bool{}
	for _, o := range out {
		seen[o] = true
	}
	for _, g := range globs {
		matches, _ := filepath.Glob(g)
		sort.Strings(matches)
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	return out
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

// detectLang picks the language for names: the explicit one, the language the
// player uses in this instance (options.txt), or the default.
func detectLang(root, explicit string) (string, string) {
	if explicit != "" {
		return names.NormalizeLang(explicit), "--lang"
	}
	if data, err := os.ReadFile(filepath.Join(root, "options.txt")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "lang:"); ok && strings.TrimSpace(v) != "" {
				return names.NormalizeLang(v), "options.txt"
			}
		}
	}
	return names.DefaultLang, "por defecto"
}

// resolveWorld accepts a world folder or the name of one under saves/.
func resolveWorld(root, name string) string {
	if st, err := os.Stat(filepath.Join(name, "level.dat")); err == nil && !st.IsDir() {
		return name
	}
	// A server keeps worlds next to mods/; a client, under saves/.
	if _, err := os.Stat(filepath.Join(root, name, "level.dat")); err == nil {
		return filepath.Join(root, name)
	}
	return filepath.Join(root, "saves", name)
}

// loadWorldDatapacks adds <world>/datapacks, skipping the ones the world
// disabled. They load last, as in the game.
func (mp *Modpack) loadWorldDatapacks(diags *domain.Diagnostics) []resources.Pack {
	if mp.World == nil {
		return nil
	}
	dir := filepath.Join(mp.World.Path, "datapacks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var packs []resources.Pack
	for _, e := range entries {
		if mp.World.PackDisabled(e.Name()) {
			diags.Add(domain.LevelInfo, stage, e.Name(), "datapack desactivado en el mundo; no se usa")
			continue
		}
		path := filepath.Join(dir, e.Name())
		switch {
		case e.IsDir():
			if _, err := os.Stat(filepath.Join(path, "data")); err != nil {
				continue
			}
			if dp, err := resources.OpenDirPack(path, resources.KindDatapack); err == nil {
				packs = append(packs, dp)
			}
		case strings.EqualFold(filepath.Ext(path), ".zip"):
			if zp, err := resources.OpenZipPack(path, resources.KindDatapack); err == nil {
				mp.closers = append(mp.closers, zp.Close)
				packs = append(packs, zp)
			}
		}
	}
	return packs
}

// ServerConfig reads a Forge server config (e.g. "lostcities-server.toml").
// Server configs are per world: the model world's serverconfig/ wins; without
// a world, defaultconfigs/ holds what the pack assigns to new worlds.
// origin says where it was found ("mundo", "defaultconfigs", "config").
func (mp *Modpack) ServerConfig(name string) ([]byte, string, error) {
	if strings.Contains(name, "..") {
		return nil, "", fmt.Errorf("ruta no válida: %s", name)
	}
	var candidates [][2]string
	if mp.World != nil {
		candidates = append(candidates, [2]string{filepath.Join(mp.World.Path, "serverconfig", name), "mundo " + mp.World.Name})
	}
	candidates = append(candidates,
		[2]string{filepath.Join(mp.Root, "defaultconfigs", name), "defaultconfigs (mundos nuevos)"},
		[2]string{filepath.Join(mp.Root, "config", name), "config"},
	)
	for _, c := range candidates {
		if data, err := os.ReadFile(c[0]); err == nil {
			return data, c[1], nil
		}
	}
	return nil, "", os.ErrNotExist
}

// RootDir returns the game folder.
func (mp *Modpack) RootDir() string { return mp.Root }

// Level returns the model world, or nil.
func (mp *Modpack) Level() *world.World { return mp.World }

// unwrapBundler returns the real game jar inside a server "bundler" jar
// (Minecraft 1.18+ ships server.jar as a launcher that keeps the server in
// META-INF/versions/<version>/server-<version>.jar).
func unwrapBundler(zp *resources.ZipPack, diags *domain.Diagnostics) resources.Pack {
	if hasVanillaData(zp) {
		return zp
	}
	for _, path := range zp.Files() {
		if !strings.HasPrefix(path, "META-INF/versions/") || !strings.HasSuffix(path, ".jar") {
			continue
		}
		data, err := resources.ReadFile(zp, path)
		if err != nil {
			diags.Add(domain.LevelWarning, stage, zp.Name(), "jar interno ilegible %s: %v", path, err)
			continue
		}
		inner, err := resources.NewZipPackFromBytes(zp.Name()+"!/"+filepath.Base(path), resources.KindVanilla, data)
		if err != nil {
			diags.Add(domain.LevelWarning, stage, zp.Name(), "jar interno ilegible %s: %v", path, err)
			continue
		}
		if hasVanillaData(inner) {
			return inner
		}
	}
	return zp
}

func hasVanillaData(p resources.Pack) bool {
	for _, f := range p.Files() {
		if strings.HasPrefix(f, "data/minecraft/") {
			return true
		}
	}
	return false
}

// serverWorld returns the world of a dedicated server (server.properties
// level-name, "world" by default) when it exists next to mods/.
func serverWorld(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "server.properties"))
	if err != nil {
		return ""
	}
	name := "world"
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "level-name="); ok && strings.TrimSpace(v) != "" {
			name = strings.TrimSpace(v)
		}
	}
	if _, err := os.Stat(filepath.Join(root, name, "level.dat")); err != nil {
		return ""
	}
	return filepath.Join(root, name)
}
