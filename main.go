package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/puppetlabs/go-puppet-dsl-workflow/puppet"
	"os"

	// Ensure that Pcore is initialized
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-puppet-dsl-workflow/puppet/functions"
	_ "github.com/puppetlabs/go-servicesdk/wf"
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
