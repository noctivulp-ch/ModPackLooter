package site

// NavGroup is a labelled group of the side navigation.
type NavGroup struct {
	Label string
	Links []Section
}

// Nav groups the tabs by the question they answer: find something, places,
// other ways of getting things, and the modpack itself. Mod tabs only
// appear when their data exists.
func (m *Model) Nav() []NavGroup {
	has := map[string]Section{}
	for _, s := range m.Sections {
		has[s.Nav] = s
	}
	pick := func(keys ...string) []Section {
		var out []Section
		for _, k := range keys {
			if s, ok := has[k]; ok {
				out = append(out, s)
			}
		}
		return out
	}
	groups := []NavGroup{
		{Label: "Buscar", Links: []Section{{Nav: "inicio", Label: "Inicio", URL: ""}, {Nav: "objetos", Label: "Objetos", URL: "objetos/"}}},
		{Label: "Lugares", Links: append([]Section{{Nav: "estructuras", Label: "Estructuras", URL: "estructuras/"}, {Nav: "biomas", Label: "Biomas", URL: "biomas/"}}, pick("lostcities")...)},
		{Label: "Otras formas", Links: append(append([]Section{{Nav: "criaturas", Label: "Criaturas", URL: "criaturas/"}}, pick("pesca", "tradeos")...), Section{Nav: "fuentes", Label: "Otras fuentes", URL: "fuentes/"})},
		{Label: "El modpack", Links: append(pick("cambios"), Section{Nav: "mods", Label: "Mods", URL: "mods/"}, Section{Nav: "acerca", Label: "Acerca", URL: "acerca/"})},
	}
	return groups
}

// Examples are sample searches for the home page, taken from the pack.
func (m *Model) Examples() []string {
	byID := map[string]*Item{}
	for _, it := range m.Items {
		if it.Variant.IsZero() {
			byID[it.ID.String()] = it
		} else if it.Variant.String() == "enchantment:minecraft:mending" {
			byID["mending"] = it
		}
	}
	var out []string
	for _, id := range []string{"mending", "minecraft:diamond", "minecraft:name_tag", "minecraft:netherite_scrap", "minecraft:totem_of_undying"} {
		if it, ok := byID[id]; ok && it.Top != nil {
			name := it.Name
			if it.Label != "" {
				name = it.Label
			}
			out = append(out, name)
		}
		if len(out) == 3 {
			break
		}
	}
	var big *Mod
	for _, md := range m.Mods {
		if md.ID.Namespace != "minecraft" && (big == nil || len(md.Items) > len(big.Items)) {
			big = md
		}
	}
	if big != nil {
		out = append(out, "@"+big.ID.Namespace)
	}
	return out
}

// Has reports whether a mod tab exists.
func (m *Model) Has(nav string) bool {
	for _, s := range m.Sections {
		if s.Nav == nav {
			return true
		}
	}
	return false
}
