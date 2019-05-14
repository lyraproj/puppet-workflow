module github.com/lyraproj/puppet-workflow

require (
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-plugin v0.0.0-20190220160451-3f118e8ee104
	github.com/lyraproj/issue v0.0.0-20190513084509-faf9b542f594
	github.com/lyraproj/pcore v0.0.0-20190514082225-61649d71c936
	github.com/lyraproj/puppet-evaluator v0.0.0-20190514124651-1338675d7039
	github.com/lyraproj/puppet-parser v0.0.0-20190513085601-0448059066e7
	github.com/lyraproj/servicesdk v0.0.0-20190513142839-bd1660ef5e23
	github.com/stretchr/testify v1.3.0
	gopkg.in/check.v1 v1.0.0-20161208181325-20d25e280405 // indirect
)

replace github.com/lyraproj/servicesdk => github.com/thallgren/servicesdk v0.0.0-20190514125406-2af500baae27
