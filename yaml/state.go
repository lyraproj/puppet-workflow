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

func ResolveState(ctx px.Context, state wf.State, input px.OrderedMap) px.PuppetObject {
	resolvedState := types.ResolveDeferred(ctx, state.State().(px.OrderedMap), input).(px.OrderedMap)
	resolvedState = convertState(ctx, state.Type(), resolvedState)
	return px.New(ctx, state.Type(), resolvedState).(px.PuppetObject)
}

// convertState coerces each state property into the type given by its corresponding attribute
func convertState(c px.Context, t px.ObjectType, st px.OrderedMap) px.OrderedMap {
	el := make([]*types.HashEntry, 0, st.Len())
	for _, a := range t.AttributesInfo().Attributes() {
		if v, ok := st.Get4(a.Name()); ok {
			el = append(el, types.WrapHashEntry2(a.Name(), coerceTo(c, a, a.Type(), v)))
		}
	}
	nst := types.WrapHash(el)
	if len(el) < st.Len() {
		// State contains attributes that are not known to the type. Merge them in to
		// force error
		st = st.Merge(nst)
	} else {
		st = nst
	}
	return st
}

func coerceTo(c px.Context, a px.Attribute, t px.Type, o px.Value) (v px.Value) {
	if t.IsInstance(o, nil) {
		return o
	}

	if opt, ok := t.(*types.OptionalType); ok {
		t = opt.ContainedType()
	}

	switch t := t.(type) {
	case *types.ArrayType:
		et := t.ElementType()
		if oa, ok := o.(*types.Array); ok {
			o = oa.Map(func(e px.Value) px.Value { return coerceTo(c, a, et, e) })
		} else {
			o = types.WrapValues([]px.Value{coerceTo(c, a, et, o)})
		}
		v = px.AssertInstance(a.Label(), t, o)
	case *types.HashType:
		kt := t.KeyType()
		vt := t.ValueType()
		if oh, ok := o.(*types.Hash); ok {
			o = oh.MapEntries(func(e px.MapEntry) px.MapEntry {
				return types.WrapHashEntry(coerceTo(c, a, kt, e.Key()), coerceTo(c, a, vt, e.Value()))
			})
		}
		v = px.AssertInstance(a.Label(), t, o)
	case px.ObjectType:
		ai := t.AttributesInfo()
		switch o := o.(type) {
		case *types.Array:
			el := make([]px.Value, o.Len())
			for i, ca := range ai.Attributes() {
				if i >= o.Len() {
					break
				}
				el[i] = coerceTo(c, ca, ca.Type(), o.At(i))
			}
			v = px.New(c, t, el...)
		case *types.Hash:
			el := make([]*types.HashEntry, 0, o.Len())
			for _, ca := range ai.Attributes() {
				if v, ok := o.Get4(ca.Name()); ok {
					el = append(el, types.WrapHashEntry2(ca.Name(), coerceTo(c, ca, ca.Type(), v)))
				}
			}
			v = px.New(c, t, o.Merge(types.WrapHash(el)))
		default:
			v = px.New(c, t, o)
		}
	default:
		// Create using single argument. This takes care of coercions from String to Version,
		// Number to Timespan, etc.
		v = px.New(c, t, o)
	}
	return
}
