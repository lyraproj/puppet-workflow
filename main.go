package main

import (
	"github.com/puppetlabs/go-puppet-dsl-workflow/puppet"

	// Ensure that Pcore is initialized
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-puppet-dsl-workflow/puppet/functions"
	_ "github.com/puppetlabs/go-servicesdk/wf"
)

func main() {
	puppet.Start(`Puppet`)
}
