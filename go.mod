module github.com/lyraproj/puppet-workflow

require (
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-plugin v0.0.0-20190220160451-3f118e8ee104
	github.com/lyraproj/issue v0.0.0-20190606092846-e082d6813d15
	github.com/lyraproj/pcore v0.0.0-20190619162937-645af37a80ad
	github.com/lyraproj/puppet-evaluator v0.0.0-20190620124608-a575c423de1a
	github.com/lyraproj/puppet-parser v0.0.0-20190606112603-21687f912799
	github.com/lyraproj/servicesdk v0.0.0-20190620124349-11383d404381
	github.com/stretchr/testify v1.3.0
	gopkg.in/check.v1 v1.0.0-20161208181325-20d25e280405 // indirect
)

replace github.com/lyraproj/pcore => github.com/thallgren/pcore v0.0.0-20190619151240-bebc8c351bb4

replace github.com/lyraproj/servicesdk => github.com/thallgren/servicesdk v0.0.0-20190619152445-7481da553aae
