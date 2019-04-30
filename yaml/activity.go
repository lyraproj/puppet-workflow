package yaml

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
	"github.com/lyraproj/servicesdk/wf"
)

type step struct {
	name   string
	parent *step
	hash   px.OrderedMap
	rt     px.ObjectType
	step   wf.Step
}

const kindWorkflow = 1
const kindResource = 2
const kindCollect = 3

func CreateStep(c px.Context, file string, content []byte) wf.Step {
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
		panic(px.Error(NotStep, issue.NoArgs))
	}

	a := newStep(name, nil, def)
	switch a.stepKind() {
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

func newStep(name string, parent *step, ex px.OrderedMap) *step {
	ca := &step{parent: parent, hash: ex}
	sgs := strings.Split(name, `::`)
	ca.name = sgs[len(sgs)-1]
	return ca
}

func (a *step) stepKind() int {
	m := a.hash
	if m.IncludesKey2(`steps`) {
		return kindWorkflow
	}
	if m.IncludesKey2(`state`) {
		return kindResource
	}
	if m.IncludesKey2(`each`) || m.IncludesKey2(`eachPair`) || m.IncludesKey2(`times`) || m.IncludesKey2(`range`) {
		return kindCollect
	}
	panic(px.Error(NotStep, issue.NoArgs))
}

func (a *step) Step() wf.Step {
	return a.step
}

func (a *step) Name() string {
	return a.name
}

func (a *step) Label() string {
	return a.Style() + " " + a.Name()
}

func (a *step) buildStep(builder wf.Builder) {
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Parameters(a.extractParameters(builder.Context(), a.hash, `parameters`, false)...)
	builder.Returns(a.extractParameters(builder.Context(), a.hash, `returns`, true)...)
}

func (a *step) buildResource(builder wf.ResourceBuilder) {
	c := builder.Context()

	builder.Name(a.Name())
	builder.When(a.getWhen())

	st, parameters := a.getState(c, a.extractParameters(builder.Context(), a.hash, `parameters`, false))
	builder.Parameters(parameters...)
	builder.Returns(a.extractParameters(builder.Context(), a.hash, `returns`, true)...)
	builder.State(&state{ctx: c, stateType: a.getResourceType(c), unresolvedState: st})

	if extId, ok := a.getStringProperty(a.hash, `external_id`); ok {
		builder.ExternalId(extId)
	}
}

func (a *step) buildWorkflow(builder wf.WorkflowBuilder) {
	a.buildStep(builder)
	de, ok := a.hash.Get4(`steps`)
	if !ok {
		return
	}

	block, ok := de.(px.OrderedMap)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
	}

	// Block should only contain step expressions or something is wrong.
	block.EachPair(func(k, v px.Value) {
		if as, ok := v.(px.OrderedMap); ok {
			a.workflowStep(builder, k.String(), as)
		} else {
			panic(px.Error(NotStep, issue.H{`actual`: as}))
		}
	})
}

func (a *step) workflowStep(builder wf.ChildBuilder, name string, as px.OrderedMap) {
	ac := newStep(name, a, as)
	switch ac.stepKind() {
	case kindCollect:
		builder.Iterator(ac.buildIterator)
	case kindWorkflow:
		builder.Workflow(ac.buildWorkflow)
	default:
		builder.Resource(ac.buildResource)
	}
}

func (a *step) Style() string {
	switch a.stepKind() {
	case kindWorkflow:
		return `workflow`
	case kindCollect:
		return `collect`
	default:
		return `resource`
	}
}

func (a *step) buildIterator(builder wf.IteratorBuilder) {
	a.buildStep(builder)

	var v, over px.Value
	var ok bool
	if v, ok = a.hash.Get4(`each`); ok {
		builder.Style(wf.IterationStyleEach)
		over = v
	} else if v, ok = a.hash.Get4(`eachPair`); ok {
		builder.Style(wf.IterationStyleEachPair)
		over = v
	} else if v, ok = a.hash.Get4(`times`); ok {
		builder.Style(wf.IterationStyleTimes)
		over = v
	} else if v, ok = a.hash.Get4(`range`); ok {
		builder.Style(wf.IterationStyleRange)
		over = v
	}
	if v, ok = a.hash.Get4(`into`); ok {
		builder.Into(v.String())
	}
	over, parameters := a.resolveParameters(over, types.DefaultAnyType(), []px.Parameter{})
	builder.Over(over)
	builder.Parameters(parameters...)
	builder.Variables(a.extractParameters(builder.Context(), a.hash, `as`, false)...)

	if v, ok = a.hash.Get4(`step`); ok {
		var step px.OrderedMap
		if step, ok = v.(px.OrderedMap); ok {
			a.workflowStep(builder, a.Name(), step)
		} else {
			panic(px.Error(NotStep, issue.H{`actual`: v.PType()}))
		}
	}
}

func (a *step) getWhen() string {
	if when, ok := a.getStringProperty(a.hash, `when`); ok {
		return when
	}
	return ``
}

func (a *step) extractParameters(c px.Context, props px.OrderedMap, field string, isReturns bool) []px.Parameter {
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
			if isReturns {
				p = a.makeReturnsParameter(c, field, k, v)
			} else {
				p = a.makeParametersParameter(c, field, k, v)
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
				if isReturns && a.stepKind() == kindResource {
					// Names must match attribute names
					params[i] = px.NewParameter(n, a.attributeType(c, n), nil, false)
				} else {
					params[i] = px.NewParameter(n, types.DefaultAnyType(), nil, false)
				}
			} else {
				panic(px.Error(BadParameter, issue.H{`step`: a, `name`: e, `parameterType`: field}))
			}
		})
		return params
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`step`: a, `field`: field, `expected`: `Hash`, `actual`: v.PType()}))
}

func (a *step) makeParametersParameter(c px.Context, field string, k, v px.Value) (param px.Parameter) {
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
			`step`: a, `name`: k, `parameterType`: `parameters`}))
	}
	return
}

var varNamePattern = regexp.MustCompile(`\A[a-z]\w*(?:\.[a-z]\w*)*\z`)

func (a *step) makeReturnsParameter(c px.Context, field string, k, v px.Value) (param px.Parameter) {
	// TODO: Iterator returns etc.
	if n, ok := k.(px.StringValue); ok {
		name := n.String()
		switch v := v.(type) {
		case px.Parameter:
			param = v
		case px.StringValue:
			s := v.String()
			if len(s) > 0 && unicode.IsUpper(rune(s[0])) {
				if a.stepKind() == kindWorkflow {
					param = px.NewParameter(name, c.ParseType(s), nil, false)
				}
			} else if varNamePattern.MatchString(s) {
				if a.stepKind() == kindResource {
					// Alias declaration
					param = px.NewParameter(name, a.attributeType(c, s), v, false)
				}
			}
		case px.List:
			if a.stepKind() == kindResource {
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
			`step`: a, `name`: k, `parameterType`: `returns`}))
	}
	return
}

func getAttributeType(tp px.TypeWithCallableMembers, name string) (px.Type, bool) {
	names := strings.Split(name, `.`)
	for i, n := range names {
		t := tp.(px.Type)
		m, ok := tp.Member(n)
		if !ok {
			hclog.Default().Debug(`no such attribute`, `type`, t.Name(), `name`, n)
			break
		}
		var a px.Attribute
		a, ok = m.(px.Attribute)
		if !ok {
			hclog.Default().Debug(`not an attribute`, `type`, t.Name(), `name`, n)
			break
		}
		at := a.Type()
		if i+1 == len(names) {
			return at, true
		}
		if ot, ok := at.(*types.OptionalType); ok {
			at = ot.ContainedType()
		}
		tp, ok = at.(px.ObjectType)
		if !ok {
			hclog.Default().Debug(`not an Object attribute`, `type`, t.Name(), `name`, n, `actual`, at.Name())
			break
		}
	}
	return nil, false
}

func (a *step) attributeType(c px.Context, name string) px.Type {
	tp := a.getResourceType(c)
	if at, ok := getAttributeType(tp, name); ok {
		return at
	}
	panic(px.Error(px.AttributeNotFound, issue.H{`type`: tp, `name`: name}))
}

func (a *step) getState(c px.Context, parameters []px.Parameter) (px.OrderedMap, []px.Parameter) {
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
					v, parameters = a.resolveParameters(v, at, parameters)
					es = append(es, types.WrapHashEntry(k, v))
					return true
				}
			}
			return false
		}) {
			return types.WrapHash(es), parameters
		}
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `Hash`, `actual`: de}))
}

func stripOptional(t px.Type) px.Type {
	if ot, ok := t.(*types.OptionalType); ok {
		return stripOptional(ot.ContainedType())
	}
	return t
}

func (a *step) resolveParameters(v px.Value, at px.Type, parameters []px.Parameter) (px.Value, []px.Parameter) {
	switch vr := v.(type) {
	case px.StringValue:
		s := vr.String()
		if len(s) > 1 && s[0] == '$' {
			vn := s[1:]
			if varNamePattern.MatchString(vn) {
				// Add to parameters unless it's there already
				found := false
				for i, ip := range parameters {
					if ip.Name() == vn {
						if ip.Type() == types.DefaultAnyType() {
							// Replace untyped with typed
							parameters[i] = px.NewParameter(vn, at, nil, false)
						}
						found = true
						break
					}
				}
				if !found {
					parameters = append(parameters, px.NewParameter(vn, at, nil, false))
				}
				v = types.NewDeferred(s)
			}
		}
	case px.OrderedMap:
		es := make([]*types.HashEntry, 0, vr.Len())
		nta := stripOptional(at)
		if ot, ok := nta.(px.TypeWithCallableMembers); ok {
			vr.EachPair(func(k, av px.Value) {
				if at, ok := getAttributeType(ot, k.String()); ok {
					av, parameters = a.resolveParameters(av, at, parameters)
				} else {
					av, parameters = a.resolveParameters(av, types.DefaultAnyType(), parameters)
				}
				es = append(es, types.WrapHashEntry(k, av))
			})
		} else if ht, ok := nta.(*types.HashType); ok {
			et := ht.ValueType()
			vr.EachPair(func(k, av px.Value) {
				av, parameters = a.resolveParameters(av, et, parameters)
				es = append(es, types.WrapHashEntry(k, av))
			})
		} else if st, ok := nta.(*types.StructType); ok {
			hm := st.HashedMembers()
			vr.EachPair(func(k, av px.Value) {
				if m, ok := hm[k.String()]; ok {
					av, parameters = a.resolveParameters(av, m.Value(), parameters)
				} else {
					av, parameters = a.resolveParameters(av, types.DefaultAnyType(), parameters)
				}
				es = append(es, types.WrapHashEntry(k, av))
			})
		} else {
			et := types.DefaultAnyType()
			vr.EachPair(func(k, av px.Value) {
				av, parameters = a.resolveParameters(av, et, parameters)
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
				ev, parameters = a.resolveParameters(ev, et, parameters)
				es[i] = ev
			})
		} else if tt, ok := nta.(*types.TupleType); ok {
			ts := tt.Types()
			vr.EachWithIndex(func(ev px.Value, i int) {
				if i < len(ts) {
					ev, parameters = a.resolveParameters(ev, ts[i], parameters)
				} else {
					ev, parameters = a.resolveParameters(ev, types.DefaultAnyType(), parameters)
				}
				es[i] = ev
			})
		} else {
			et := types.DefaultAnyType()
			vr.EachWithIndex(func(ev px.Value, i int) {
				ev, parameters = a.resolveParameters(ev, et, parameters)
				es[i] = ev
			})
		}
		v = types.WrapValues(es)
	}
	return v, parameters
}

func (a *step) getResourceType(c px.Context) px.ObjectType {
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
			panic(px.Error(FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `Variant[String,ObjectType]`, `actual`: tv}))
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
		panic(px.Error(FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `ObjectType`, `actual`: t}))
	}
	panic(px.Error(px.UnresolvedType, issue.H{`typeString`: tn.Name()}))
}

func (a *step) getTypespace() string {
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

func (a *step) getStringProperty(properties px.OrderedMap, field string) (string, bool) {
	v, ok := properties.Get4(field)
	if !ok {
		return ``, false
	}

	if s, ok := v.(px.StringValue); ok {
		return s.String(), true
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`step`: a, `field`: field, `expected`: `String`, `actual`: v.PType()}))
}
