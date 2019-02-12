package yaml

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/impl"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-evaluator/yaml"
	"github.com/lyraproj/servicesdk/wfapi"
	"regexp"
	"strings"
	"unicode"
)

type activity struct {
	name     string
	parent   *activity
	hash     eval.OrderedMap
	rt       eval.ObjectType
	activity wfapi.Activity
}

const kindWorkflow = 1
const kindResource = 2

func CreateActivity(c eval.Context, file string, content []byte) wfapi.Activity {
	c.StackPush(issue.NewLocation(file, 0, 0))
	defer c.StackPop()

	v := yaml.Unmarshal(c, content)
	h, ok := v.(eval.OrderedMap)
	if !(ok && h.Len() == 1) {
		panic(eval.Error(WF_NOT_ONE_DEFINITION, issue.NO_ARGS))
	}

	var name string
	var def eval.OrderedMap
	h.EachPair(func(k, v eval.Value) {
		if n, ok := k.(eval.StringValue); ok {
			name = n.String()
		}
		if m, ok := v.(eval.OrderedMap); ok {
			def = m
		}
	})
	if name == `` || def == nil {
		panic(eval.Error(WF_NOT_ACTIVITY, issue.NO_ARGS))
	}

	a := newActivity(name, nil, def)
	switch a.activityKind() {
	case kindWorkflow:
		return wfapi.NewWorkflow(c, func(wb wfapi.WorkflowBuilder) {
			a.buildWorkflow(wb)
		})
	default:
		return wfapi.NewResource(c, func(wb wfapi.ResourceBuilder) {
			a.buildResource(wb)
		})
	}
}

func newActivity(name string, parent *activity, ex eval.OrderedMap) *activity {
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
	panic(eval.Error(WF_NOT_ACTIVITY, issue.NO_ARGS))
}

func (a *activity) Activity() wfapi.Activity {
	return a.activity
}

func (a *activity) Name() string {
	return a.name
}

func (a *activity) Label() string {
	return a.Style() + " " + a.Name()
}

func (a *activity) buildActivity(builder wfapi.Builder) {
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Input(a.extractParameters(builder.Context(), a.hash, `input`, false)...)
	builder.Output(a.extractParameters(builder.Context(), a.hash, `output`, true)...)
}

func (a *activity) buildResource(builder wfapi.ResourceBuilder) {
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

func (a *activity) buildStateless(builder wfapi.StatelessBuilder) {
	a.buildActivity(builder)
}

func (a *activity) buildWorkflow(builder wfapi.WorkflowBuilder) {
	a.buildActivity(builder)
	de, ok := a.hash.Get4(`activities`)
	if !ok {
		return
	}

	block, ok := de.(eval.OrderedMap)
	if !ok {
		panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
	}

	// Block should only contain activity expressions or something is wrong.
	block.EachPair(func(k, v eval.Value) {
		if as, ok := v.(eval.OrderedMap); ok {
			a.workflowActivity(builder, k.String(), as)
		} else {
			panic(eval.Error(WF_NOT_ACTIVITY, issue.H{`actual`: as}))
		}
	})
}

func (a *activity) workflowActivity(builder wfapi.WorkflowBuilder, name string, as eval.OrderedMap) {
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

func (a *activity) inferInput() []eval.Parameter {
	// TODO:
	return eval.NoParameters
}

func (a *activity) inferOutput() []eval.Parameter {
	// TODO:
	return eval.NoParameters
}

func (a *activity) buildIterator(builder wfapi.IteratorBuilder) {
	v, _ := a.hash.Get4(`iteration`)
	iteratorDef, ok := v.(*types.HashValue)
	if !ok {
		panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: `iteration`, `expected`: `Hash`, `actual`: v.PType()}))
	}

	v = iteratorDef.Get5(`function`, eval.UNDEF)
	style, ok := v.(eval.StringValue)
	if !ok {
		panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: `iteration.style`, `expected`: `String`, `actual`: v}))
	}
	if name, ok := iteratorDef.Get4(`name`); ok {
		builder.Name(name.String())
	}
	builder.Style(wfapi.NewIterationStyle(style.String()))
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

func (a *activity) extractParameters(c eval.Context, props eval.OrderedMap, field string, isOutput bool) []eval.Parameter {
	if props == nil {
		return []eval.Parameter{}
	}

	v, ok := props.Get4(field)
	if !ok {
		return []eval.Parameter{}
	}

	if ph, ok := v.(eval.OrderedMap); ok {
		params := make([]eval.Parameter, 0, ph.Len())
		ph.EachPair(func(k, v eval.Value) {
			var p eval.Parameter
			if isOutput {
				p = a.makeOutputParameter(c, field, k, v)
			} else {
				p = a.makeInputParameter(c, field, k, v)
			}
			params = append(params, p)
		})
		return params
	}

	if _, ok := v.(eval.StringValue); ok {
		// Allow single name as a convenience
		v = types.WrapValues([]eval.Value{v})
	}

	if pa, ok := v.(*types.ArrayValue); ok {
		// List of names.
		params := make([]eval.Parameter, pa.Len())
		pa.EachWithIndex(func(e eval.Value, i int) {
			if ne, ok := e.(eval.StringValue); ok {
				n := ne.String()
				if isOutput && a.activityKind() == kindResource {
					// Names must match attribute names
					params[i] = impl.NewParameter(n, a.attributeType(c, n), nil, false)
				} else {
					params[i] = impl.NewParameter(n, types.DefaultAnyType(), nil, false)
				}
			} else {
				panic(eval.Error(WF_BAD_PARAMETER, issue.H{`activity`: a, `name`: e, `parameterType`: field}))
			}
		})
		return params
	}
	panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: field, `expected`: `Hash`, `actual`: v.PType()}))
}

func (a *activity) makeInputParameter(c eval.Context, field string, k, v eval.Value) (param eval.Parameter) {
	if n, ok := k.(eval.StringValue); ok {
		name := n.String()
		switch v.(type) {
		case eval.Parameter:
			param = v.(eval.Parameter)
		case eval.StringValue:
			param = impl.NewParameter(name, c.ParseType2(v.String()), nil, false)
		case eval.OrderedMap:
			m := v.(eval.OrderedMap)
			tn, ok := a.getStringProperty(m, `type`)
			if !ok {
				break
			}
			var val eval.Value
			tp := c.ParseType2(tn)
			if lu, ok := m.Get4(`lookup`); ok {
				var args []eval.Value
				if a, ok := lu.(*types.ArrayValue); ok {
					args = a.AppendTo(make([]eval.Value, 0, a.Len()))
				} else {
					args = []eval.Value{lu}
				}
				val = types.NewDeferred(`lookup`, args...)
			} else {
				val = m.Get5(`value`, nil)
			}
			param = impl.NewParameter(name, tp, val, false)
		}
	}
	if param == nil {
		panic(eval.Error(WF_BAD_PARAMETER, issue.H{
			`activity`: a, `name`: k, `parameterType`: `input`}))
	}
	return
}

var varNamePattern = regexp.MustCompile(`\A[a-z]\w*(?:\.[a-z]\w*)*\z`)

func (a *activity) makeOutputParameter(c eval.Context, field string, k, v eval.Value) (param eval.Parameter) {
	// TODO: Iterator output etc.
	if n, ok := k.(eval.StringValue); ok {
		name := n.String()
		switch v.(type) {
		case eval.Parameter:
			param = v.(eval.Parameter)
		case eval.StringValue:
			s := v.String()
			if len(s) > 0 && unicode.IsUpper(rune(s[0])) {
				if a.activityKind() == kindWorkflow {
					param = impl.NewParameter(name, c.ParseType2(s), nil, false)
				}
			} else if varNamePattern.MatchString(s) {
				if a.activityKind() == kindResource {
					// Alias declaration
					param = impl.NewParameter(name, a.attributeType(c, s), v, false)
				}
			}
		case eval.List:
			if a.activityKind() == kindResource {
				vl := v.(eval.List)
				ts := make([]eval.Type, 0, vl.Len())
				if v.(eval.List).All(func(e eval.Value) bool {
					if sv, ok := e.(eval.StringValue); ok {
						s := sv.String()
						if varNamePattern.MatchString(s) {
							ts = append(ts, a.attributeType(c, s))
							return true
						}
					}
					return false
				}) {
					param = impl.NewParameter(name, types.NewTupleType(ts, nil), v, false)
				}
			}
		}
	}
	if param == nil {
		panic(eval.Error(WF_BAD_PARAMETER, issue.H{
			`activity`: a, `name`: k, `parameterType`: `output`}))
	}
	return
}

func (a *activity) attributeType(c eval.Context, name string) eval.Type {
	tp := a.getResourceType(c)
	if m, ok := tp.Member(name); ok {
		if a, ok := m.(eval.Attribute); ok {
			return a.Type()
		}
	}
	panic(eval.Error(eval.EVAL_ATTRIBUTE_NOT_FOUND, issue.H{`type`: tp, `name`: name}))
}

func (a *activity) getState(c eval.Context, input []eval.Parameter) (eval.OrderedMap, []eval.Parameter) {
	de, ok := a.hash.Get4(`state`)
	if !ok {
		return eval.EMPTY_MAP, []eval.Parameter{}
	}

	if hash, ok := de.(eval.OrderedMap); ok {
		// Ensure that hash conforms to init of type with respect to attribute names
		// and transform all variable references to Deferred expressions
		es := make([]*types.HashEntry, 0, hash.Len())
		if hash.AllPairs(func(k, v eval.Value) bool {
			if sv, ok := k.(eval.StringValue); ok {
				s := sv.String()
				if varNamePattern.MatchString(s) {
					at := a.attributeType(c, s)
					if vs, ok := v.(eval.StringValue); ok {
						s := vs.String()
						if len(s) > 1 && s[0] == '$' {
							vn := s[1:]
							if varNamePattern.MatchString(vn) {
								// Add to input unless it's there already
								found := false
								for _, ip := range input {
									if ip.Name() == vn {
										found = true
										break
									}
								}
								if !found {
									input = append(input, impl.NewParameter(vn, at, nil, false))
								}
								v = types.NewDeferred(s)
							}
						}
					}
					es = append(es, types.WrapHashEntry(k, v))
					return true
				}
			}
			return false
		}) {
			return types.WrapHash(es), input
		}
	}
	panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: `definition`, `expected`: `Hash`, `actual`: de}))
}

func (a *activity) getResourceType(c eval.Context) eval.ObjectType {
	if a.rt != nil {
		return a.rt
	}
	n := a.Name()
	if tv, ok := a.hash.Get4(`type`); ok {
		if t, ok := tv.(eval.ObjectType); ok {
			a.rt = t
			return t
		}
		if s, ok := tv.(eval.StringValue); ok {
			n = s.String()
		} else {
			panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: `definition`, `expected`: `Variant[String,ObjectType]`, `actual`: tv}))
		}
	} else {
		ts := a.getTypespace()
		if ts != `` {
			n = ts + `::` + wfapi.LeafName(n)
		}
	}

	tn := eval.NewTypedName(eval.NsType, n)
	if t, ok := eval.Load(c, tn); ok {
		if pt, ok := t.(eval.ObjectType); ok {
			a.rt = pt
			return pt
		}
		panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: `definition`, `expected`: `ObjectType`, `actual`: t}))
	}
	panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: tn.Name()}))
}

func (a *activity) getTypespace() string {
	if ts, ok := a.getStringProperty(a.hash, `typespace`); ok {
		return ts
	}
	if a.parent != nil {
		return a.parent.getTypespace()
	}
	return ``
}

func (a *activity) getStringProperty(properties eval.OrderedMap, field string) (string, bool) {
	v, ok := properties.Get4(field)
	if !ok {
		return ``, false
	}

	if s, ok := v.(eval.StringValue); ok {
		return s.String(), true
	}
	panic(eval.Error(WF_FIELD_TYPE_MISMATCH, issue.H{`activity`: a, `field`: field, `expected`: `String`, `actual`: v.PType()}))
}
