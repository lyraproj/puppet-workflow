package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/puppet-workflow/puppetwf"
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
