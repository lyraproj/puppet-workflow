package yaml

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
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
	resolvedState := types.ResolveDeferred(ctx, state.State().(px.OrderedMap), parameters).(px.OrderedMap)
	return px.New(ctx, state.Type(), resolvedState).(px.PuppetObject)
}
