package yaml

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
	"github.com/lyraproj/servicesdk/wf"
)

type activity struct {
	name     string
	parent   *activity
	hash     px.OrderedMap
	rt       px.ObjectType
	activity wf.Activity
}

const kindWorkflow = 1
const kindResource = 2

func CreateActivity(c px.Context, file string, content []byte) wf.Activity {
	c.StackPush(issue.NewLocation(file, 0, 0))
	defer c.StackPop()

	v := yaml.Unmarshal(c, content)
	h, ok := v.(px.OrderedMap)
	if !(ok && h.Len() == 1) {
		panic(px.Error(NotOneDefinition, issue.NoArgs))
	}

	var name string
	var def px.OrderedMap
	h.EachPair(func(k, v px.Value) {
		if n, ok := k.(px.StringValue); ok {
			name = n.String()
		}
		if m, ok := v.(px.OrderedMap); ok {
			def = m
		}
	})
	if name == `` || def == nil {
		panic(px.Error(NotActivity, issue.NoArgs))
	}

	a := newActivity(name, nil, def)
	switch a.activityKind() {
	case kindWorkflow:
		return wf.NewWorkflow(c, func(wb wf.WorkflowBuilder) {
			a.buildWorkflow(wb)
		})
	default:
		return wf.NewResource(c, func(wb wf.ResourceBuilder) {
			a.buildResource(wb)
		})
	}
}

func newActivity(name string, parent *activity, ex px.OrderedMap) *activity {
	ca := &activity{parent: parent, hash: ex}
	sgs := strings.Split(name, `::`)
	ca.name = sgs[len(sgs)-1]
	return ca
}

func (a *activity) activityKind() int {
	m := a.hash
	if m.IncludesKey2(`activities`) {
		return kindWorkflow
	}
	if m.IncludesKey2(`state`) {
		return kindResource
	}
	panic(px.Error(NotActivity, issue.NoArgs))
}

func (a *activity) Activity() wf.Activity {
	return a.activity
}

func (a *activity) Name() string {
	return a.name
}

func (a *activity) Label() string {
	return a.Style() + " " + a.Name()
}

func (a *activity) buildActivity(builder wf.Builder) {
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Input(a.extractParameters(builder.Context(), a.hash, `input`, false)...)
	builder.Output(a.extractParameters(builder.Context(), a.hash, `output`, true)...)
}

func (a *activity) buildResource(builder wf.ResourceBuilder) {
	c := builder.Context()

	builder.Name(a.Name())
	builder.When(a.getWhen())

	st, input := a.getState(c, a.extractParameters(builder.Context(), a.hash, `input`, false))
	builder.Input(input...)
	builder.Output(a.extractParameters(builder.Context(), a.hash, `output`, true)...)
	builder.State(&state{ctx: c, stateType: a.getResourceType(c), unresolvedState: st})

	if extId, ok := a.getStringProperty(a.hash, `external_id`); ok {
		builder.ExternalId(extId)
	}
}

func (a *activity) buildWorkflow(builder wf.WorkflowBuilder) {
	a.buildActivity(builder)
	de, ok := a.hash.Get4(`activities`)
	if !ok {
		return
	}

	block, ok := de.(px.OrderedMap)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
	}

	// Block should only contain activity expressions or something is wrong.
	block.EachPair(func(k, v px.Value) {
		if as, ok := v.(px.OrderedMap); ok {
			a.workflowActivity(builder, k.String(), as)
		} else {
			panic(px.Error(NotActivity, issue.H{`actual`: as}))
		}
	})
}

func (a *activity) workflowActivity(builder wf.WorkflowBuilder, name string, as px.OrderedMap) {
	ac := newActivity(name, a, as)
	if _, ok := ac.hash.Get4(`iteration`); ok {
		builder.Iterator(ac.buildIterator)
	} else {
		switch ac.activityKind() {
		case kindWorkflow:
			builder.Workflow(ac.buildWorkflow)
		default:
			builder.Resource(ac.buildResource)
		}
	}
}

func (a *activity) Style() string {
	switch a.activityKind() {
	case kindWorkflow:
		return `workflow`
	default:
		return `resource`
	}
}

func (a *activity) buildIterator(builder wf.IteratorBuilder) {
	v, _ := a.hash.Get4(`iteration`)
	iteratorDef, ok := v.(*types.Hash)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: `iteration`, `expected`: `Hash`, `actual`: v.PType()}))
	}

	v = iteratorDef.Get5(`function`, px.Undef)
	style, ok := v.(px.StringValue)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: `iteration.style`, `expected`: `String`, `actual`: v}))
	}
	if name, ok := iteratorDef.Get4(`name`); ok {
		builder.Name(name.String())
	}
	builder.Style(wf.NewIterationStyle(style.String()))
	builder.Over(a.extractParameters(builder.Context(), iteratorDef, `over`, false)...)
	builder.Variables(a.extractParameters(builder.Context(), iteratorDef, `vars`, false)...)

	switch a.activityKind() {
	case kindWorkflow:
		builder.Workflow(a.buildWorkflow)
	default:
		builder.Resource(a.buildResource)
	}
}

func (a *activity) getWhen() string {
	if when, ok := a.getStringProperty(a.hash, `when`); ok {
		return when
	}
	return ``
}

func (a *activity) extractParameters(c px.Context, props px.OrderedMap, field string, isOutput bool) []px.Parameter {
	if props == nil {
		return []px.Parameter{}
	}

	v, ok := props.Get4(field)
	if !ok {
		return []px.Parameter{}
	}

	if ph, ok := v.(px.OrderedMap); ok {
		params := make([]px.Parameter, 0, ph.Len())
		ph.EachPair(func(k, v px.Value) {
			var p px.Parameter
			if isOutput {
				p = a.makeOutputParameter(c, field, k, v)
			} else {
				p = a.makeInputParameter(c, field, k, v)
			}
			params = append(params, p)
		})
		return params
	}

	if _, ok := v.(px.StringValue); ok {
		// Allow single name as a convenience
		v = types.WrapValues([]px.Value{v})
	}

	if pa, ok := v.(*types.Array); ok {
		// List of names.
		params := make([]px.Parameter, pa.Len())
		pa.EachWithIndex(func(e px.Value, i int) {
			if ne, ok := e.(px.StringValue); ok {
				n := ne.String()
				if isOutput && a.activityKind() == kindResource {
					// Names must match attribute names
					params[i] = px.NewParameter(n, a.attributeType(c, n), nil, false)
				} else {
					params[i] = px.NewParameter(n, types.DefaultAnyType(), nil, false)
				}
			} else {
				panic(px.Error(BadParameter, issue.H{`activity`: a, `name`: e, `parameterType`: field}))
			}
		})
		return params
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: field, `expected`: `Hash`, `actual`: v.PType()}))
}

func (a *activity) makeInputParameter(c px.Context, field string, k, v px.Value) (param px.Parameter) {
	if n, ok := k.(px.StringValue); ok {
		name := n.String()
		switch v := v.(type) {
		case px.Parameter:
			param = v
		case px.StringValue:
			param = px.NewParameter(name, c.ParseType(v.String()), nil, false)
		case px.OrderedMap:
			tn, ok := a.getStringProperty(v, `type`)
			if !ok {
				break
			}
			var val px.Value
			tp := c.ParseType(tn)
			if lu, ok := v.Get4(`lookup`); ok {
				var args []px.Value
				if a, ok := lu.(*types.Array); ok {
					args = a.AppendTo(make([]px.Value, 0, a.Len()))
				} else {
					args = []px.Value{lu}
				}
				val = types.NewDeferred(`lookup`, args...)
			} else {
				val = v.Get5(`value`, nil)
			}
			param = px.NewParameter(name, tp, val, false)
		}
	}
	if param == nil {
		panic(px.Error(BadParameter, issue.H{
			`activity`: a, `name`: k, `parameterType`: `input`}))
	}
	return
}

var varNamePattern = regexp.MustCompile(`\A[a-z]\w*(?:\.[a-z]\w*)*\z`)

func (a *activity) makeOutputParameter(c px.Context, field string, k, v px.Value) (param px.Parameter) {
	// TODO: Iterator output etc.
	if n, ok := k.(px.StringValue); ok {
		name := n.String()
		switch v := v.(type) {
		case px.Parameter:
			param = v
		case px.StringValue:
			s := v.String()
			if len(s) > 0 && unicode.IsUpper(rune(s[0])) {
				if a.activityKind() == kindWorkflow {
					param = px.NewParameter(name, c.ParseType(s), nil, false)
				}
			} else if varNamePattern.MatchString(s) {
				if a.activityKind() == kindResource {
					// Alias declaration
					param = px.NewParameter(name, a.attributeType(c, s), v, false)
				}
			}
		case px.List:
			if a.activityKind() == kindResource {
				ts := make([]px.Type, 0, v.Len())
				if v.All(func(e px.Value) bool {
					if sv, ok := e.(px.StringValue); ok {
						s := sv.String()
						if varNamePattern.MatchString(s) {
							ts = append(ts, a.attributeType(c, s))
							return true
						}
					}
					return false
				}) {
					param = px.NewParameter(name, types.NewTupleType(ts, nil), v, false)
				}
			}
		}
	}
	if param == nil {
		panic(px.Error(BadParameter, issue.H{
			`activity`: a, `name`: k, `parameterType`: `output`}))
	}
	return
}

func (a *activity) attributeType(c px.Context, name string) px.Type {
	tp := a.getResourceType(c)
	if m, ok := tp.Member(name); ok {
		if a, ok := m.(px.Attribute); ok {
			return a.Type()
		}
	}
	panic(px.Error(px.AttributeNotFound, issue.H{`type`: tp, `name`: name}))
}

func (a *activity) getState(c px.Context, input []px.Parameter) (px.OrderedMap, []px.Parameter) {
	de, ok := a.hash.Get4(`state`)
	if !ok {
		return px.EmptyMap, []px.Parameter{}
	}

	if hash, ok := de.(px.OrderedMap); ok {
		// Ensure that hash conforms to init of type with respect to attribute names
		// and transform all variable references to Deferred expressions
		es := make([]*types.HashEntry, 0, hash.Len())
		if hash.AllPairs(func(k, v px.Value) bool {
			if sv, ok := k.(px.StringValue); ok {
				s := sv.String()
				if varNamePattern.MatchString(s) {
					at := a.attributeType(c, s)
					v, input = a.resolveInputs(v, at, input)
					es = append(es, types.WrapHashEntry(k, v))
					return true
				}
			}
			return false
		}) {
			return types.WrapHash(es), input
		}
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: `definition`, `expected`: `Hash`, `actual`: de}))
}

func stripOptional(t px.Type) px.Type {
	if ot, ok := t.(*types.OptionalType); ok {
		return stripOptional(ot.ContainedType())
	}
	return t
}

func (a *activity) resolveInputs(v px.Value, at px.Type, input []px.Parameter) (px.Value, []px.Parameter) {
	switch vr := v.(type) {
	case px.StringValue:
		s := vr.String()
		if len(s) > 1 && s[0] == '$' {
			vn := s[1:]
			if varNamePattern.MatchString(vn) {
				// Add to input unless it's there already
				found := false
				for i, ip := range input {
					if ip.Name() == vn {
						if ip.Type() == types.DefaultAnyType() {
							// Replace untyped with typed
							input[i] = px.NewParameter(vn, at, nil, false)
						}
						found = true
						break
					}
				}
				if !found {
					input = append(input, px.NewParameter(vn, at, nil, false))
				}
				v = types.NewDeferred(s)
			}
		}
	case px.OrderedMap:
		es := make([]*types.HashEntry, 0, vr.Len())
		nta := stripOptional(at)
		if ot, ok := nta.(px.TypeWithCallableMembers); ok {
			vr.EachPair(func(k, av px.Value) {
				if m, ok := ot.Member(k.String()); ok {
					av, input = a.resolveInputs(av, m.(px.AnnotatedMember).Type(), input)
				} else {
					av, input = a.resolveInputs(av, types.DefaultAnyType(), input)
				}
				es = append(es, types.WrapHashEntry(k, av))
			})
		} else if ht, ok := nta.(*types.HashType); ok {
			et := ht.ValueType()
			vr.EachPair(func(k, av px.Value) {
				av, input = a.resolveInputs(av, et, input)
				es = append(es, types.WrapHashEntry(k, av))
			})
		} else if st, ok := nta.(*types.StructType); ok {
			hm := st.HashedMembers()
			vr.EachPair(func(k, av px.Value) {
				if m, ok := hm[k.String()]; ok {
					av, input = a.resolveInputs(av, m.Value(), input)
				} else {
					av, input = a.resolveInputs(av, types.DefaultAnyType(), input)
				}
				es = append(es, types.WrapHashEntry(k, av))
			})
		} else {
			et := types.DefaultAnyType()
			vr.EachPair(func(k, av px.Value) {
				av, input = a.resolveInputs(av, et, input)
				es = append(es, types.WrapHashEntry(k, av))
			})
		}
		v = types.WrapHash(es)
	case px.List:
		es := make([]px.Value, vr.Len())
		nta := stripOptional(at)
		if st, ok := nta.(*types.ArrayType); ok {
			et := st.ElementType()
			vr.EachWithIndex(func(ev px.Value, i int) {
				ev, input = a.resolveInputs(ev, et, input)
				es[i] = ev
			})
		} else if tt, ok := nta.(*types.TupleType); ok {
			ts := tt.Types()
			vr.EachWithIndex(func(ev px.Value, i int) {
				if i < len(ts) {
					ev, input = a.resolveInputs(ev, ts[i], input)
				} else {
					ev, input = a.resolveInputs(ev, types.DefaultAnyType(), input)
				}
				es[i] = ev
			})
		} else {
			et := types.DefaultAnyType()
			vr.EachWithIndex(func(ev px.Value, i int) {
				ev, input = a.resolveInputs(ev, et, input)
				es[i] = ev
			})
		}
		v = types.WrapValues(es)
	}
	return v, input
}

func (a *activity) getResourceType(c px.Context) px.ObjectType {
	if a.rt != nil {
		return a.rt
	}
	n := a.Name()
	if tv, ok := a.hash.Get4(`type`); ok {
		if t, ok := tv.(px.ObjectType); ok {
			a.rt = t
			return t
		}
		if s, ok := tv.(px.StringValue); ok {
			n = s.String()
			if !types.TypeNamePattern.MatchString(n) {
				panic(px.Error(InvalidTypeName, issue.H{`name`: n}))
			}
		} else {
			panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: `definition`, `expected`: `Variant[String,ObjectType]`, `actual`: tv}))
		}
	} else {
		ts := a.getTypespace()
		if ts != `` {
			n = ts + `::` + wf.LeafName(n)
		}
	}

	tn := px.NewTypedName(px.NsType, n)
	if t, ok := px.Load(c, tn); ok {
		if pt, ok := t.(px.ObjectType); ok {
			a.rt = pt
			return pt
		}
		panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: `definition`, `expected`: `ObjectType`, `actual`: t}))
	}
	panic(px.Error(px.UnresolvedType, issue.H{`typeString`: tn.Name()}))
}

func (a *activity) getTypespace() string {
	if ts, ok := a.getStringProperty(a.hash, `typespace`); ok {
		if types.TypeNamePattern.MatchString(ts) {
			return ts
		}
		panic(px.Error(InvalidTypeName, issue.H{`name`: ts}))
	}
	if a.parent != nil {
		return a.parent.getTypespace()
	}
	return ``
}

func (a *activity) getStringProperty(properties px.OrderedMap, field string) (string, bool) {
	v, ok := properties.Get4(field)
	if !ok {
		return ``, false
	}

	if s, ok := v.(px.StringValue); ok {
		return s.String(), true
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`activity`: a, `field`: field, `expected`: `String`, `actual`: v.PType()}))
}
