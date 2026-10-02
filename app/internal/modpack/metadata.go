package modpack

import (
	"bufio"
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// Mod describes one mod found in the modpack.
type Mod struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Version string        `json:"version,omitempty"`
	File    string        `json:"file"`
	Loader  domain.Loader `json:"loader"`
	// MinecraftRange is the raw Minecraft dependency declared by the mod.
	MinecraftRange string `json:"-"`
}

type forgeModsToml struct {
	Mods []struct {
		ModID       string `toml:"modId"`
		DisplayName string `toml:"displayName"`
		Version     string `toml:"version"`
	} `toml:"mods"`
	Dependencies map[string][]struct {
		ModID        string `toml:"modId"`
		VersionRange string `toml:"versionRange"`
	} `toml:"dependencies"`
}

type fabricModJSON struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Depends map[string]any `json:"depends"`
}

var (
	reModID       = regexp.MustCompile(`(?m)^\s*modId\s*=\s*"([^"]+)"`)
	reDisplayName = regexp.MustCompile(`(?m)^\s*displayName\s*=\s*"([^"]+)"`)
)

// readMods extracts the mods declared by a jar. A jar may declare several
// mods (or none, for libraries).
func readMods(p resources.Pack, file string) ([]Mod, error) {
	for _, candidate := range []struct {
		path   string
		loader domain.Loader
	}{
		{"META-INF/neoforge.mods.toml", domain.LoaderNeoForge},
		{"META-INF/mods.toml", domain.LoaderForge},
	} {
		data, err := resources.ReadFile(p, candidate.path)
		if err != nil {
			continue
		}
		return parseModsToml(data, file, candidate.loader, manifestVersion(p)), nil
	}
	if data, err := resources.ReadFile(p, "fabric.mod.json"); err == nil {
		var meta fabricModJSON
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, err
		}
		mod := Mod{ID: meta.ID, Name: meta.Name, Version: meta.Version, File: file, Loader: domain.LoaderFabric}
		if mc, ok := meta.Depends["minecraft"].(string); ok {
			mod.MinecraftRange = mc
		}
		return []Mod{mod}, nil
	}
	return nil, nil
}

func parseModsToml(data []byte, file string, loader domain.Loader, jarVersion string) []Mod {
	var meta forgeModsToml
	if _, err := toml.Decode(string(data), &meta); err != nil || len(meta.Mods) == 0 {
		// Some mods ship slightly invalid TOML; fall back to the essentials.
		var mods []Mod
		names := reDisplayName.FindAllSubmatch(data, -1)
		for i, m := range reModID.FindAllSubmatch(data, -1) {
			mod := Mod{ID: string(m[1]), File: file, Loader: loader, Version: jarVersion}
			if i < len(names) {
				mod.Name = string(names[i][1])
			}
			mods = append(mods, mod)
		}
		return mods
	}
	var mods []Mod
	for _, m := range meta.Mods {
		version := m.Version
		if version == "" || strings.Contains(version, "${") {
			version = jarVersion
		}
		mod := Mod{ID: m.ModID, Name: m.DisplayName, Version: version, File: file, Loader: loader}
		for _, dep := range meta.Dependencies[m.ModID] {
			if dep.ModID == "minecraft" {
				mod.MinecraftRange = dep.VersionRange
			}
		}
		mods = append(mods, mod)
	}
	return mods
}

func manifestVersion(p resources.Pack) string {
	data, err := resources.ReadFile(p, "META-INF/MANIFEST.MF")
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "Implementation-Version:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
