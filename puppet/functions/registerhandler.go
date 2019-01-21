package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/servicesdk/service"

	// Ensure initialization of needed packages
	_ "github.com/lyraproj/puppet-evaluator/pcore"
)

const ServerBuilderKey = `WF::ServerBuilder`

func init() {
	eval.NewGoFunction(`register_handler`,
		func(d eval.Dispatch) {
			d.Param(`Type[Object]`)
			d.Param(`Object`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				if v, ok := c.Get(ServerBuilderKey); ok {
					if sb, ok := v.(*service.ServerBuilder); ok {
						handler := args[1]
						sb.RegisterHandler(handler.PType().Name(), handler, args[0].(eval.Type))
						return eval.UNDEF
					}
				}
				panic(eval.Error(WF_NO_SERVER_BUILDER_IN_CONTEXT, issue.NO_ARGS))
			})
		},
	)
}
