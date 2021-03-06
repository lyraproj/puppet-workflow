package puppetwf

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/servicesdk/wf"
)

type state struct {
	ctx             px.Context
	stateType       px.ObjectType
	unresolvedState px.OrderedMap
}

func (r *state) Type() px.ObjectType {
	return r.stateType
}

func (r *state) State() interface{} {
	return r.unresolvedState
}

func ResolveState(ctx px.Context, state wf.State, parameters px.OrderedMap) px.PuppetObject {
	scope := ctx.Scope().(pdsl.Scope)
	return scope.WithLocalScope(func() (v px.Value) {
		parameters.EachPair(func(k, v px.Value) {
			scope.Set(k.String(), v)
		})
		st := types.ResolveDeferred(ctx, state.State().(px.OrderedMap), scope).(px.OrderedMap)
		return px.New(ctx, state.Type(), st).(px.PuppetObject)
	}).(px.PuppetObject)
}
