package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-puppet-dsl-workflow/puppet"
	"github.com/puppetlabs/go-servicesdk/service"
)

func init() {
	eval.NewGoFunction(`register_handler`,
		func(d eval.Dispatch) {
			d.Param(`Type[Object]`)
			d.Param(`Object`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				if v, ok := c.Get(puppet.ServerBuilderKey); ok {
					if sb, ok := v.(*service.ServerBuilder); ok {
						handler := args[1]
						sb.RegisterHandler(handler.PType().Name(), handler, args[0].(eval.Type))
						return eval.UNDEF
					}
				}
				panic(eval.Error(puppet.WF_NO_SERVER_BUILDER_IN_CONTEXT, issue.NO_ARGS))
			})
		},
	)
}
