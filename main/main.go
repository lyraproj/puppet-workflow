package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/puppet-workflow/puppetwf"
)

func init() {
	// Configuring hclog like this allows Lyra to handle log levels automatically
	hclog.DefaultOptions = &hclog.LoggerOptions{
		Name:            "Puppet",
		Level:           hclog.LevelFromString(os.Getenv("LYRA_LOG_LEVEL")),
		JSONFormat:      true,
		IncludeLocation: false,
		Output:          os.Stderr,
	}
}

func main() {
	puppetwf.Start(`Puppet`)
}
