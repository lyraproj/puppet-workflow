package puppetwf

import (
	"bytes"
	"os/exec"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/service"
)

const ServerBuilderKey = `WF::ServerBuilder`

func stringArgs(args []px.Value) []string {
	l := len(args)
	if l == 0 {
		return []string{}
	}
	se := make([]string, l)
	for i, a := range args {
		se[i] = a.String()
	}
	return se
}

func init() {
	px.NewGoFunction(`registerHandler`,
		func(d px.Dispatch) {
			d.Param(`Type[Object]`)
			d.Param(`Object`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				if v, ok := c.Get(ServerBuilderKey); ok {
					if sb, ok := v.(*service.Builder); ok {
						handler := args[1]
						sb.RegisterHandler(handler.PType().Name(), handler, args[0].(px.Type))
						return px.Undef
					}
				}
				panic(px.Error(NoServerBuilderInContext, issue.NO_ARGS))
			})
		},
	)

	px.NewGoFunction(`exec`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Variant[String,Number,Boolean]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				cmd := exec.Command(args[0].String(), stringArgs(args[1:])...)
				var out bytes.Buffer
				cmd.Stdout = &out
				err := cmd.Run()
				if err != nil {
					panic(err)
				}
				return types.WrapString(out.String())
			})
		},
	)
}
