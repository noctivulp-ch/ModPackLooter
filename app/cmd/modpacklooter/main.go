// Command modpacklooter is the composition root: it wires the concrete
// adapters and discoverers into the core and hands control to the CLI.
package main

import (
	"fmt"
	"os"

	"github.com/EnierAragon/ModPackLooter/app/internal/cli"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/discovery/generic"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

func main() {
	registry := discovery.NewRegistry(
		// Specific, relational and heuristic discoverers are registered here
		// as they are implemented. The generic fallback always runs last.
		generic.ByPath{},
	)
	root := cli.NewRootCommand(cli.Deps{Version: version, Discovery: registry})
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
