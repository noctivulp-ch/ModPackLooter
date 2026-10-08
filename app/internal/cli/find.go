package cli

import (
	"fmt"
	
	"sort"
	"strings"

	"github.com/noctivulp-ch/ModPackLooter/app/internal/domain"
	"github.com/noctivulp-ch/ModPackLooter/app/internal/site"
	"github.com/spf13/cobra"
)

func newFindCommand(deps Deps) *cobra.Command {
	var f packFlags
	cmd := &cobra.Command{
		Use:   "find <carpeta-del-modpack> <query>...",
		Short: "Busca los mejores contenedores (biomas, estructuras, mobs) para obtener un conjunto de objetos",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			diags := &domain.Diagnostics{}
			opts := f.options(args[0], diags)
			res, err := deps.Analyzer.Run(cmd.Context(), opts, nil)
			if err != nil {
				return err
			}
			defer res.Close()
			m := site.BuildModel(res, site.Options{Lang: f.lang})

			var targetItems []*site.Item
			for _, it := range m.Items {
				if it.Unobtainable {
					continue
				}
				match := false
				for _, q := range args[1:] {
					if strings.HasPrefix(q, "@") {
						if it.ID.Namespace == q[1:] {
							match = true
							break
						}
					} else if strings.Contains(q, "*") {
						prefix := strings.Split(q, "*")[0]
						if strings.HasPrefix(it.ID.String(), prefix) {
							match = true
							break
						}
					} else {
						if it.ID.String() == q || it.ID.Path == q {
							match = true
							break
						}
					}
				}
				if match {
					targetItems = append(targetItems, it)
				}
			}

			if len(targetItems) == 0 {
				fmt.Println("No se encontraron objetos obtenibles que coincidan con la búsqueda.")
				return nil
			}

			fmt.Printf("Buscando fuentes para %d objetos...\n\n", len(targetItems))

			// source name -> items it provides
			sourceItems := map[string]map[*site.Item]bool{}
			itemSources := map[*site.Item]map[string]bool{}

			for _, it := range targetItems {
				itemSources[it] = map[string]bool{}
				for _, card := range it.Cards {
					for _, g := range card.Structures {
						for _, st := range g.Structures {
							if st.Owner == nil { continue }; src := st.Owner.Name
							if sourceItems[src] == nil {
								sourceItems[src] = map[*site.Item]bool{}
							}
							sourceItems[src][it] = true
							itemSources[it][src] = true
						}
					}
					for _, g := range card.LC {
						for _, b := range g.Buildings {
							if b.Building == nil { continue }; src := b.Building.Name + " (Lost Cities)"
							if sourceItems[src] == nil {
								sourceItems[src] = map[*site.Item]bool{}
							}
							sourceItems[src][it] = true
							itemSources[it][src] = true
						}
					}
					for _, mob := range card.Mobs {
						if mob == nil || mob.Owner == nil { continue }; src := mob.Owner.Name + " (Mob)"
						if sourceItems[src] == nil {
							sourceItems[src] = map[*site.Item]bool{}
						}
						sourceItems[src][it] = true
						itemSources[it][src] = true
					}
					for _, f := range card.Fish {
						src := f.Name + " (Pesca)"
						if sourceItems[src] == nil {
							sourceItems[src] = map[*site.Item]bool{}
						}
						sourceItems[src][it] = true
						itemSources[it][src] = true
					}
				}
			}

			// Greedy Set Cover
			uncovered := map[*site.Item]bool{}
			for _, it := range targetItems {
				uncovered[it] = true
			}

			var selected []string
			for len(uncovered) > 0 {
				bestSrc := ""
				bestCount := 0
				for src, items := range sourceItems {
					count := 0
					for it := range items {
						if uncovered[it] {
							count++
						}
					}
					if count > bestCount {
						bestCount = count
						bestSrc = src
					}
				}

				if bestCount == 0 {
					break // No more coverage possible
				}

				selected = append(selected, bestSrc)
				for it := range sourceItems[bestSrc] {
					delete(uncovered, it)
				}
			}

			if len(selected) > 0 {
				fmt.Println("Mejores contenedores/fuentes para obtener estos objetos (de mayor a menor cobertura):")
				for i, src := range selected {
					fmt.Printf("%d. %s\n", i+1, src)
					// List which target items this provides
					var provided []string
					for it := range sourceItems[src] {
						provided = append(provided, it.Name)
					}
					sort.Strings(provided)
					fmt.Printf("   -> Provee %d objetos buscados: %s\n", len(provided), strings.Join(provided, ", "))
				}
			}

			if len(uncovered) > 0 {
				fmt.Printf("\nAtención: %d objetos no se pueden obtener de ninguna fuente conocida.\n", len(uncovered))
				for it := range uncovered {
					fmt.Printf(" - %s (%s)\n", it.Name, it.ID)
				}
			}

			return nil
		},
	}
	f.register(cmd)
	return cmd
}
