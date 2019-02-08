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
		resolvedState := types.ResolveDeferred(ctx, state.State().(eval.OrderedMap)).(eval.OrderedMap)
		resolvedState = convertState(ctx, state.Type(), resolvedState)
		return eval.New(ctx, state.Type(), resolvedState).(eval.PuppetObject)
	}).(eval.PuppetObject)
}

// convertState coerces each state property into the type given by its corresponding attribute
func convertState(c eval.Context, t eval.ObjectType, st eval.OrderedMap) eval.OrderedMap {
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

func coerceTo(c eval.Context, a eval.Attribute, t eval.Type, o eval.Value) eval.Value {
	if t.IsInstance(o, nil) {
		return o
	}

	if opt, ok := t.(*types.OptionalType); ok {
		t = opt.ContainedType()
	}

	switch t.(type) {
	case *types.ArrayType:
		et := t.(*types.ArrayType).ElementType()
		if oa, ok := o.(*types.ArrayValue); ok {
			o = oa.Map(func(e eval.Value) eval.Value { return coerceTo(c, a, et, e) })
		} else {
			o = types.WrapValues([]eval.Value{coerceTo(c, a, et, o)})
		}
		o = eval.AssertInstance(a.Label(), t, o)
	case *types.HashType:
		ht := t.(*types.HashType)
		kt := ht.KeyType()
		vt := ht.ValueType()
		if oh, ok := o.(*types.HashValue); ok {
			o = oh.MapEntries(func(e eval.MapEntry) eval.MapEntry {
				return types.WrapHashEntry(coerceTo(c, a, kt, e.Key()), coerceTo(c, a, vt, e.Value()))
			})
		}
		o = eval.AssertInstance(a.Label(), t, o)
	case eval.ObjectType:
		ot := t.(eval.ObjectType)
		ai := ot.AttributesInfo()
		switch o.(type) {
		case *types.ArrayValue:
			av := o.(*types.ArrayValue)
			el := make([]eval.Value, av.Len())
			for i, ca := range ai.Attributes() {
				if i >= av.Len() {
					break
				}
				el[i] = coerceTo(c, ca, ca.Type(), av.At(i))
			}
			o = eval.New(c, ot, el...)
		case *types.HashValue:
			hv := o.(*types.HashValue)
			el := make([]*types.HashEntry, 0, hv.Len())
			for _, ca := range ai.Attributes() {
				if v, ok := hv.Get4(ca.Name()); ok {
					el = append(el, types.WrapHashEntry2(ca.Name(), coerceTo(c, ca, ca.Type(), v)))
				}
			}
			o = eval.New(c, ot, hv.Merge(types.WrapHash(el)))
		default:
			o = eval.New(c, ot, o)
		}
	default:
		// Create using single argument. This takes care of coercions from String to Version,
		// Number to Timespan, etc.
		o = eval.New(c, t, o)
	}
	return o
}
