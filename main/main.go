package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-workflow/puppetwf"
)

func main() {
	// Configuring hclog like this allows Lyra to handle log levels automatically
	hclog.DefaultOptions = &hclog.LoggerOptions{
		Name:            "Puppet",
		Level:           hclog.LevelFromString(os.Getenv("LYRA_LOG_LEVEL")),
		JSONFormat:      true,
		IncludeLocation: false,
		Output:          os.Stderr,
	}
	if hclog.DefaultOptions.Level <= hclog.Debug {
		// Tell issue reporting to amend all errors with a stack trace.
		issue.IncludeStacktrace(true)
	}
	puppetwf.Start(`Puppet`)
}
