package lostcities

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// Profiles says where Lost Cities generates cities, following Lost Cities
// 1.20 (setup/Config.java): the overworld uses the server config's
// selectedProfile, which is per world (serverconfig/, or defaultconfigs/ for
// new worlds); other dimensions come from the common config's
// dimensionsWithProfiles ("dimension=profile").
type Profiles struct {
	Overworld string // "" none, "<CHECK>" chosen when the world is created
	Origin    string // where Overworld was read from
	ByDim     map[string]string
	FromWorld bool
}

// Active returns the dimensions with cities and their profile.
func (p Profiles) Active() map[string]string {
	out := map[string]string{}
	for d, prof := range p.ByDim {
		out[d] = prof
	}
	if p.Overworld != "" && p.Overworld != "<CHECK>" {
		out["minecraft:overworld"] = p.Overworld
	}
	return out
}

func flatten(doc map[string]any, out map[string]any) {
	for k, v := range doc {
		if sub, ok := v.(map[string]any); ok {
			flatten(sub, out)
			continue
		}
		out[k] = v
	}
}

// ReadProfiles reads the Lost Cities configs of the modpack (and world).
func ReadProfiles(in discovery.Input) Profiles {
	p := Profiles{Overworld: "<CHECK>", Origin: "valor por defecto de Lost Cities", ByDim: map[string]string{}}
	if data, origin, err := in.Files.ServerConfig("lostcities-server.toml"); err == nil {
		var doc map[string]any
		if _, err := toml.Decode(string(data), &doc); err == nil {
			flat := map[string]any{}
			flatten(doc, flat)
			if s, ok := flat["selectedProfile"].(string); ok {
				p.Overworld, p.Origin = s, origin
				p.FromWorld = strings.HasPrefix(origin, "mundo")
			}
		}
	}
	for _, path := range []string{"config/lostcities/common.toml", "config/lostcities-common.toml"} {
		data, err := in.Files.ReadFile(path)
		if err != nil {
			continue
		}
		var doc map[string]any
		if _, err := toml.Decode(string(data), &doc); err != nil {
			continue
		}
		flat := map[string]any{}
		flatten(doc, flat)
		if list, ok := flat["dimensionsWithProfiles"].([]any); ok {
			for _, e := range list {
				if s, ok := e.(string); ok {
					if dim, prof, ok := strings.Cut(s, "="); ok {
						p.ByDim[strings.TrimSpace(dim)] = strings.TrimSpace(prof)
					}
				}
			}
		}
		break
	}
	return p
}

// Describe summarises the profiles for the site.
func (p Profiles) Describe() string {
	active := p.Active()
	if len(active) == 0 {
		return ""
	}
	var parts []string
	for d, prof := range active {
		parts = append(parts, fmt.Sprintf("%s: perfil %s", d, prof))
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

// DisablerID is the ID of the Lost Cities profile disabler.
const DisablerID = "lostcities-profile"

// Disabler reports Lost Cities buildings as disabled when no dimension has a
// city profile.
type Disabler struct{}

func (Disabler) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: DisablerID, Phase: discovery.PhaseSpecific, Priority: 50,
		Applies: discovery.Applicability{RequiresMods: []string{"lostcities"}}}
}

func (Disabler) Detect(_ context.Context, in discovery.Input, out *discovery.Disablements) error {
	p := ReadProfiles(in)
	target := domain.Target{Kind: domain.TargetStructure, ID: Owner.ID}
	switch {
	case len(p.Active()) > 0:
		return nil
	case p.Overworld == "<CHECK>":
		out.Add(domain.Disablement{Target: target, Certainty: domain.Possibly, By: DisablerID,
			Reason: "el perfil de Lost Cities se elige al crear el mundo; indica un mundo modelo con --world para saberlo"})
	case p.FromWorld:
		out.Add(domain.Disablement{Target: target, Certainty: domain.Certainly, By: DisablerID,
			Reason: "el mundo no tiene perfil de Lost Cities en ninguna dimensión (" + p.Origin + ")"})
	default:
		out.Add(domain.Disablement{Target: target, Certainty: domain.Possibly, By: DisablerID,
			Reason: "los mundos nuevos no tienen perfil de Lost Cities (" + p.Origin + ")"})
	}
	return nil
}
