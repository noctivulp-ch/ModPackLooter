package disablers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// StructurifyID is the ID of the Structurify disabler.
const StructurifyID = "structurify"

// Structurify reads config/structurify.json (format of Structurify 2.0.x):
// general.disable_all_structures, and "is_disabled" on structures, structure
// sets and namespaces.
type Structurify struct{}

func (Structurify) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: StructurifyID, Phase: discovery.PhaseSpecific, Priority: 70,
		Applies: discovery.Applicability{RequiresMods: []string{"structurify"}}}
}

type structurifyEntry struct {
	Name       string `json:"name"`
	IsDisabled bool   `json:"is_disabled"`
}

func (Structurify) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	data, err := in.Files.ReadFile("config/structurify.json")
	if err != nil {
		return nil
	}
	var cfg struct {
		General struct {
			DisableAll bool `json:"disable_all_structures"`
		} `json:"general"`
		Namespaces []structurifyEntry `json:"structure_namespaces"`
		Structures []structurifyEntry `json:"structures"`
		Sets       []structurifyEntry `json:"structure_sets"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		in.Diagnostics.Add(domain.LevelWarning, "desactivadores", "config/structurify.json", "configuración de Structurify ilegible: %v", err)
		return nil
	}
	add := func(id domain.ResourceID, reason string) {
		out.Add(domain.Disablement{Target: structureTarget(id), Certainty: domain.Certainly, By: StructurifyID, Reason: reason})
	}
	structures := in.World.Structures()
	if cfg.General.DisableAll {
		for _, s := range structures {
			add(s.ID, "Structurify desactiva todas las estructuras")
		}
		return nil
	}
	disabledNS := map[string]bool{}
	for _, n := range cfg.Namespaces {
		if n.IsDisabled {
			disabledNS[n.Name] = true
		}
	}
	for _, s := range structures {
		if disabledNS[s.ID.Namespace] {
			add(s.ID, "Structurify desactiva todas las estructuras de "+s.ID.Namespace)
		}
	}
	for _, e := range cfg.Structures {
		if id, err := domain.ParseResourceID(e.Name); err == nil && e.IsDisabled {
			add(id, "desactivada en Structurify")
		}
	}
	sets := in.World.StructureSets()
	for _, e := range cfg.Sets {
		if !e.IsDisabled {
			continue
		}
		for structure, owners := range sets {
			for _, set := range owners {
				if set.String() == e.Name {
					add(structure, "Structurify desactiva su structure_set "+e.Name)
				}
			}
		}
	}
	return nil
}

// InControlID is the ID of the InControl disabler.
const InControlID = "incontrol-spawns"

// InControl reads config/incontrol/spawn.json: a rule that denies a mob with
// no other condition stops it for sure; with conditions (dimension, biome,
// phase…) it is only possibly gone.
type InControl struct{}

func (InControl) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: InControlID, Phase: discovery.PhaseSpecific, Priority: 60,
		Applies: discovery.Applicability{RequiresMods: []string{"incontrol"}}}
}

func (InControl) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	data, err := in.Files.ReadFile("config/incontrol/spawn.json")
	if err != nil {
		return nil
	}
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(data, &rules); err != nil {
		in.Diagnostics.Add(domain.LevelWarning, "desactivadores", "config/incontrol/spawn.json", "reglas de InControl ilegibles: %v", err)
		return nil
	}
	for i, r := range rules {
		var result string
		_ = json.Unmarshal(r["result"], &result)
		if result != "deny" || r["mob"] == nil {
			continue
		}
		var mobs []string
		if err := json.Unmarshal(r["mob"], &mobs); err != nil {
			var one string
			if json.Unmarshal(r["mob"], &one) == nil {
				mobs = []string{one}
			}
		}
		var conditions []string
		for k := range r {
			if k != "mob" && k != "result" {
				conditions = append(conditions, k)
			}
		}
		sort.Strings(conditions)
		certainty, reason := domain.Certainly, fmt.Sprintf("InControl impide que aparezca (regla %d de spawn.json, sin condiciones)", i+1)
		if len(conditions) > 0 {
			certainty = domain.Possibly
			reason = fmt.Sprintf("InControl lo bloquea bajo condiciones: %s (regla %d de spawn.json)", strings.Join(conditions, ", "), i+1)
		}
		for _, m := range mobs {
			if id, err := domain.ParseResourceID(m); err == nil {
				out.Add(domain.Disablement{Target: domain.Target{Kind: domain.TargetEntity, ID: id}, Certainty: certainty, By: InControlID, Reason: reason})
			}
		}
	}
	return nil
}

// Words that, near a structure id, suggest the config or script turns it
// off. They also match inside keys such as "disabledStructures".
var disableWords = regexp.MustCompile(`(?i)[a-z_]*(disabl|blacklist|blocklist|deny|denied|exclu|remov|prevent|cancel)[a-z_]*|\bban(ned|s)?\b`)

// Files handled by a specific plugin or whose lists do not disable
// generation (Lootr's blacklists only stop the per-player conversion; Lost
// Cities' avoidStructures only keeps cities away).
var skipFiles = regexp.MustCompile(`(?i)(^|/)(lootr[^/]*|structurify[^/]*|biome_replacer\.properties|lostcities[^/]*|incontrol)(/|$)`)

var textExt = map[string]bool{".toml": true, ".json": true, ".json5": true, ".properties": true, ".cfg": true, ".txt": true, ".yaml": true, ".yml": true, ".snbt": true, ".js": true}

const maxScanBytes = 2 << 20

// mention scans text files for structure ids with disabling words nearby.
type mention struct {
	id     string
	label  string
	dirs   func(in discovery.Input) [][2]string // (dir, label)
	window int                                  // > 0: look at that many previous lines (scripts); 0: use the governing key (configs)
}

func (m mention) detect(in discovery.Input, out *discovery.Disablements) {
	structures := in.World.Structures()
	if len(structures) == 0 {
		return
	}
	known := map[string]domain.ResourceID{}
	var alts []string
	for _, s := range structures {
		known[s.ID.String()] = s.ID
		alts = append(alts, regexp.QuoteMeta(s.ID.String()))
	}
	sort.Slice(alts, func(i, j int) bool { return len(alts[i]) > len(alts[j]) })
	re := regexp.MustCompile(`(?:^|[^a-z0-9_:/.-])(` + strings.Join(alts, "|") + `)(?:$|[^a-z0-9_/.-])`)
	seen := map[string]bool{}
	for _, d := range m.dirs(in) {
		_ = filepath.WalkDir(d[0], func(path string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() || !textExt[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			rel, _ := filepath.Rel(d[0], path)
			rel = filepath.ToSlash(rel)
			if skipFiles.MatchString(rel) {
				return nil
			}
			info, err := e.Info()
			if err != nil || info.Size() > maxScanBytes {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				for _, match := range re.FindAllStringSubmatch(line, -1) {
					var context string
					if m.window > 0 {
						context = strings.Join(lines[max(0, i-m.window):i+1], "\n")
					} else {
						context = governingKey(lines, i)
					}
					word := disableWords.FindString(context)
					if word == "" {
						continue
					}
					id := known[match[1]]
					key := id.String() + "|" + rel
					if seen[key] {
						continue
					}
					seen[key] = true
					out.Add(domain.Disablement{
						Target: structureTarget(id), Certainty: domain.Possibly, By: m.id,
						Reason: fmt.Sprintf("aparece en %s/%s (línea %d) junto a «%s»", d[1], rel, i+1, strings.ToLower(word)),
					})
				}
			}
			return nil
		})
	}
}

var (
	reTomlKey = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-"' ]+?)\s*=`)
	reJSONKey = regexp.MustCompile(`"([^"]+)"\s*:`)
	reYAMLKey = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-]+)\s*:(\s|$)`)
)

// keyOf returns the key a config line assigns, if any.
func keyOf(line string) string {
	for _, re := range []*regexp.Regexp{reTomlKey, reJSONKey, reYAMLKey} {
		if m := re.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// governingKey returns the key that owns line i: its own key, or the key of
// the list it belongs to (walking up until the list opens), plus a comment
// right above that key.
func governingKey(lines []string, i int) string {
	j := i
	key := keyOf(lines[i])
	for key == "" && j > 0 && i-j < 200 {
		j--
		trimmed := strings.TrimSpace(lines[j])
		if strings.HasPrefix(trimmed, "]") || strings.HasPrefix(trimmed, "}") {
			return "" // the value is not inside the block above
		}
		key = keyOf(lines[j])
	}
	if key == "" {
		return ""
	}
	if j > 0 {
		prev := strings.TrimSpace(lines[j-1])
		if strings.HasPrefix(prev, "#") || strings.HasPrefix(prev, "//") {
			return prev + "\n" + key
		}
	}
	return key
}

// ConfigMentionsID is the ID of the ConfigMentions disabler.
const ConfigMentionsID = "config-mentions"

// ConfigMentions flags structures named in mod configs (config/,
// defaultconfigs/ and the world's serverconfig/) next to words such as
// "disable", "blacklist" or "exclude". It cannot know the mod's semantics, so
// it only says "possibly".
type ConfigMentions struct{}

func (ConfigMentions) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: ConfigMentionsID, Phase: discovery.PhaseHeuristic, Priority: 10}
}

func (ConfigMentions) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	mention{id: ConfigMentionsID, dirs: func(in discovery.Input) [][2]string {
		root := in.Files.RootDir()
		dirs := [][2]string{{filepath.Join(root, "config"), "config"}, {filepath.Join(root, "defaultconfigs"), "defaultconfigs"}}
		if w := in.Files.Level(); w != nil {
			dirs = append(dirs, [2]string{filepath.Join(w.Path, "serverconfig"), "serverconfig"})
		}
		return dirs
	}}.detect(in, out)
	return nil
}

// KubeJSID is the ID of the KubeJS disabler.
const KubeJSID = "kubejs-scripts"

// KubeJS flags structures named in KubeJS scripts next to words such as
// "remove" or "disable". Scripts are code, so the result is only "possibly".
type KubeJS struct{}

func (KubeJS) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: KubeJSID, Phase: discovery.PhaseHeuristic, Priority: 5,
		Applies: discovery.Applicability{RequiresMods: []string{"kubejs"}}}
}

func (KubeJS) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	mention{id: KubeJSID, window: 2, dirs: func(in discovery.Input) [][2]string {
		root := filepath.Join(in.Files.RootDir(), "kubejs")
		return [][2]string{{filepath.Join(root, "startup_scripts"), "kubejs/startup_scripts"}, {filepath.Join(root, "server_scripts"), "kubejs/server_scripts"}}
	}}.detect(in, out)
	return nil
}
