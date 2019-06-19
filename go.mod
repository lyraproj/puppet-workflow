module github.com/lyraproj/puppet-workflow

require (
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-plugin v0.0.0-20190220160451-3f118e8ee104
	github.com/lyraproj/issue v0.0.0-20190606092846-e082d6813d15
	github.com/lyraproj/pcore v0.0.0-20190618142417-30605b6ee043
	github.com/lyraproj/puppet-evaluator v0.0.0-20190606112643-077894dcf8b7
	github.com/lyraproj/puppet-parser v0.0.0-20190606112603-21687f912799
	github.com/lyraproj/servicesdk v0.0.0-20190618142858-870593a059dc
	github.com/stretchr/testify v1.3.0
	gopkg.in/check.v1 v1.0.0-20161208181325-20d25e280405 // indirect
)

replace github.com/lyraproj/pcore => github.com/thallgren/pcore v0.0.0-20190619151240-bebc8c351bb4

replace github.com/lyraproj/servicesdk => github.com/thallgren/servicesdk v0.0.0-20190619152445-7481da553aae
