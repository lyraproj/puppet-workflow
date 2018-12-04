package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/puppet-workflow/puppet"
	"os"

	// Ensure that Pcore is initialized
	_ "github.com/lyraproj/puppet-evaluator/pcore"
	_ "github.com/lyraproj/puppet-workflow/puppet/functions"
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
	puppet.Start(`Puppet`)
}
