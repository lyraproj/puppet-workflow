package main

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/lyraproj/puppet-workflow/puppetwf"

	// Ensure that Pcore is initialized
	_ "github.com/lyraproj/servicesdk/wf"
)

func main() {
	hclog.DefaultOptions = &hclog.LoggerOptions{
		Name:            "Puppet",
		Level:           hclog.Debug,
		JSONFormat:      false,
		IncludeLocation: false,
		Output:          os.Stderr,
	}
	puppetwf.Start(`Puppet`)
}
