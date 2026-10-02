// Command modpacklooter is the composition root: it wires the concrete
// plugins into the core and hands control to the CLI.
package main

import (
	"os"

	"github.com/EnierAragon/ModPackLooter/app/internal/analysis"
	"github.com/EnierAragon/ModPackLooter/app/internal/cli"
	"github.com/EnierAragon/ModPackLooter/app/internal/plugins"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

func main() {
	deps := cli.Deps{
		Version:  version,
		Analyzer: analysis.Analyzer{Discoverers: plugins.Discoverers(), Enrichers: plugins.Enrichers()},
	}
	os.Exit(cli.Main(deps, os.Args[1:]))
}
