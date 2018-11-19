package main

import (
	"flag"
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-puppet-dsl-workflow/puppet"
	"github.com/puppetlabs/go-servicesdk/grpc"
	"os"

	// Ensure that Pcore is initialized
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-puppet-dsl-workflow/puppet/functions"
	_ "github.com/puppetlabs/go-servicesdk/wf"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "missing file name")
		os.Exit(1)
	}

	eval.Puppet.Set(`tasks`, types.Boolean_TRUE)
	eval.Puppet.Set(`workflow`, types.Boolean_TRUE)
	eval.Puppet.Do(func(c eval.Context) {
		// TODO: Service name should probably be a command line option
		grpc.Serve(c, puppet.CreateService(c, `Puppet`, args[0]))
	})
}
