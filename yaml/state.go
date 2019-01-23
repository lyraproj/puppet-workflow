package yaml

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/servicesdk/wfapi"
)

type state struct {
	ctx             eval.Context
	stateType       eval.ObjectType
	unresolvedState eval.OrderedMap
}

func (r *state) Type() eval.ObjectType {
	return r.stateType
}

func (r *state) State() interface{} {
	return r.unresolvedState
}

func ResolveState(ctx eval.Context, state wfapi.State, input eval.OrderedMap) eval.PuppetObject {
	return ctx.Scope().WithLocalScope(func() (v eval.Value) {
		scope := ctx.Scope()
		input.EachPair(func(k, v eval.Value) {
			scope.Set(k.String(), v)
		})
		resolvedState := types.ResolveDeferred(ctx, state.State().(eval.OrderedMap))
		return eval.New(ctx, state.Type(), resolvedState).(eval.PuppetObject)
	}).(eval.PuppetObject)
}
