package functions

import (
	"bytes"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"os/exec"
)

func stringArgs(args []eval.Value) []string {
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
	eval.NewGoFunction(`exec`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Variant[String,Number,Boolean]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
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
