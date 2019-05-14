package yaml

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/wf"
)

type step struct {
	name   string
	origin issue.Location
	parent *step
	hash   px.OrderedMap
	rt     px.ObjectType
	step   wf.Step
}

const kindWorkflow = 1
const kindResource = 2
const kindCollect = 3
const kindReference = 4

func CreateStep(c px.Context, file string, content []byte) wf.Step {
	loc := issue.NewLocation(file, 0, 0)
	c.StackPush(loc)
	defer c.StackPop()

	v := yaml.Unmarshal(c, content)
	def, ok := v.(px.OrderedMap)
	if !ok {
		panic(px.Error(wf.NotStepDefinition, issue.NoArgs))
	}

	// Use base name of path extending from 'workflows' directory without extension as the name of the workflow
	path := strings.Split(file, string([]byte{filepath.Separator}))
	wfi := 0
	for i, n := range path {
		if strings.EqualFold(n, `workflows`) {
			wfi = i + 1
			break
		}
	}
	var name string
	if wfi > 0 {
		name = strings.Join(path[wfi:], `::`)
	} else {
		// No 'workflows' directory found. Use base name.
		name = path[len(path)-1]
	}
	name = name[:len(name)-len(filepath.Ext(name))]
	a := newStep(name, loc, nil, def)
	switch a.stepKind() {
	case kindWorkflow:
		return wf.NewWorkflow(c, func(wb wf.WorkflowBuilder) {
			a.buildWorkflow(wb)
		})
	case kindCollect:
		return wf.NewIterator(c, func(ib wf.IteratorBuilder) {
			a.buildIterator(ib)
		})
	case kindReference:
		return wf.NewReference(c, func(rb wf.ReferenceBuilder) {
			a.buildReference(rb)
		})
	default:
		return wf.NewResource(c, func(wb wf.ResourceBuilder) {
			a.buildResource(wb)
		})
	}
}

func newStep(name string, origin issue.Location, parent *step, ex px.OrderedMap) *step {
	ca := &step{origin: origin, parent: parent, hash: ex}
	sgs := strings.Split(name, `::`)
	ca.name = sgs[len(sgs)-1]
	return ca
}

func (a *step) amendError() {
	if r := recover(); r != nil {
		if rx, ok := r.(issue.Reported); ok {
			// Location and stack included in nested error
			r = issue.ErrorWithoutStack(wf.StepBuildError, issue.H{`step`: a.Label()}, nil, rx)
		} else {
			r = issue.NewNested(wf.StepBuildError, issue.H{`step`: a.Label()}, a.origin, wf.ToError(r))
		}
		panic(r)
	}
}

func (a *step) Error(code issue.Code, args issue.H) issue.Reported {
	return px.Error2(a.origin, code, args)
}

func (a *step) stepKind() int {
	m := a.hash
	if m.IncludesKey2(`steps`) {
		return kindWorkflow
	}
	if m.IncludesKey2(`each`) || m.IncludesKey2(`eachPair`) || m.IncludesKey2(`times`) || m.IncludesKey2(`range`) {
		return kindCollect
	}
	if m.IncludesKey2(`reference`) {
		return kindReference
	}
	// hash must include exactly one key which is a type name
	if m.Keys().Select(func(key px.Value) bool { return types.TypeNamePattern.MatchString(key.String()) }).Len() == 1 {
		return kindResource
	}

	panic(a.Error(wf.NotStep, issue.NoArgs))
}

func (a *step) Step() wf.Step {
	return a.step
}

func (a *step) Name() string {
	return a.name
}

func (a *step) QName() string {
	if a.parent != nil {
		return a.parent.QName() + `/` + a.name
	}
	return a.name
}

func (a *step) Label() string {
	return a.Style() + " " + a.QName()
}

func (a *step) buildStep(builder wf.Builder) {
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Parameters(a.extractParameters(builder.Context(), a.hash, `parameters`, false)...)
	builder.Returns(a.extractParameters(builder.Context(), a.hash, `returns`, true)...)
}

func (a *step) buildResource(builder wf.ResourceBuilder) {
	defer a.amendError()

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

func (a *step) buildReference(builder wf.ReferenceBuilder) {
	defer a.amendError()

	c := builder.Context()

	builder.Name(a.Name())
	builder.When(a.getWhen())

	builder.Parameters(a.extractParameters(c, a.hash, `parameters`, true)...)
	builder.Returns(a.extractParameters(c, a.hash, `returns`, true)...)

	// Step will reference a step with the same name by default.
	reference := a.Name()
	if ra, ok := a.getStringProperty(a.hash, `reference`); ok && ra != `` {
		reference = ra
	}
	builder.ReferenceTo(reference)
}

func (a *step) buildWorkflow(builder wf.WorkflowBuilder) {
	a.buildStep(builder)
	de, ok := a.hash.Get4(`steps`)
	if !ok {
		return
	}

	block, ok := de.(px.OrderedMap)
	if !ok {
		panic(a.Error(wf.FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
	}

	// Block should only contain step expressions or something is wrong.
	block.EachPair(func(k, v px.Value) {
		if as, ok := v.(px.OrderedMap); ok {
			a.workflowStep(builder, k.String(), as)
		} else {
			panic(a.Error(wf.NotStep, issue.H{`actual`: as}))
		}
	})
}

func (a *step) workflowStep(builder wf.ChildBuilder, name string, as px.OrderedMap) {
	ac := newStep(name, a.origin, a, as)
	switch ac.stepKind() {
	case kindCollect:
		builder.Iterator(ac.buildIterator)
	case kindWorkflow:
		builder.Workflow(ac.buildWorkflow)
	case kindReference:
		builder.Reference(ac.buildReference)
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
	case kindReference:
		return `reference`
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
	over, parameters := a.resolveParameters(over, types.DefaultAnyType(), []serviceapi.Parameter{})
	builder.Over(over)
	builder.Parameters(parameters...)
	builder.Variables(a.extractParameters(builder.Context(), a.hash, `as`, false)...)

	if v, ok = a.hash.Get4(`step`); ok {
		var step px.OrderedMap
		if step, ok = v.(px.OrderedMap); ok {
			a.workflowStep(builder, a.Name(), step)
		} else {
			panic(a.Error(wf.NotStep, issue.H{`actual`: v.PType()}))
		}
	}
}

func (a *step) getWhen() string {
	if when, ok := a.getStringProperty(a.hash, `when`); ok {
		return when
	}
	return ``
}

func (a *step) extractParameters(c px.Context, props px.OrderedMap, field string, aliased bool) []serviceapi.Parameter {
	if props == nil {
		return []serviceapi.Parameter{}
	}

	v, ok := props.Get4(field)
	if !ok {
		return []serviceapi.Parameter{}
	}

	if ph, ok := v.(px.OrderedMap); ok {
		params := make([]serviceapi.Parameter, 0, ph.Len())
		ph.EachPair(func(k, v px.Value) {
			params = append(params, a.makeParameter(c, field, k, v, aliased))
		})
		return params
	}

	if _, ok := v.(px.StringValue); ok {
		// Allow single name as a convenience
		v = types.WrapValues([]px.Value{v})
	}

	if pa, ok := v.(*types.Array); ok {
		// List of names.
		params := make([]serviceapi.Parameter, pa.Len())
		pa.EachWithIndex(func(e px.Value, i int) {
			if ne, ok := e.(px.StringValue); ok {
				n := ne.String()
				if aliased && a.stepKind() == kindResource {
					// Names must match attribute names
					params[i] = serviceapi.NewParameter(n, ``, a.attributeType(c, n), nil)
				} else {
					params[i] = serviceapi.NewParameter(n, ``, types.DefaultAnyType(), nil)
				}
			} else {
				panic(a.Error(wf.BadParameter, issue.H{`step`: a, `name`: e, `parameterType`: field}))
			}
		})
		return params
	}
	panic(a.Error(wf.FieldTypeMismatch, issue.H{`step`: a, `field`: field, `expected`: `Hash`, `actual`: v.PType()}))
}

var varNamePattern = regexp.MustCompile(`\A[a-z]\w*(?:\.[a-z]\w*)*\z`)

func (a *step) makeParameter(c px.Context, field string, k, v px.Value, aliased bool) (param serviceapi.Parameter) {
	if n, ok := k.(px.StringValue); ok {
		name := n.String()
		switch v := v.(type) {
		case px.Parameter:
			var val px.Value
			if v.HasValue() {
				val = v.Value()
			}
			param = serviceapi.NewParameter(v.Name(), ``, v.Type(), val)
		case serviceapi.Parameter:
			param = v
		case px.StringValue:
			s := v.String()
			if types.TypeNamePattern.MatchString(s) {
				param = serviceapi.NewParameter(name, ``, c.ParseType(v.String()), nil)
			} else {
				if aliased && len(s) > 0 && varNamePattern.MatchString(s) {
					if a.stepKind() == kindResource {
						// Alias declaration
						param = serviceapi.NewParameter(name, s, a.attributeType(c, s), nil)
					} else if a.stepKind() == kindReference {
						param = serviceapi.NewParameter(name, s, types.DefaultAnyType(), nil)
					}
				}
			}
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
			alias := ``
			if al, ok := v.Get4(`alias`); ok {
				alias = al.String()
			}
			param = serviceapi.NewParameter(name, alias, tp, val)
		}
	}
	if param == nil {
		panic(a.Error(wf.BadParameter, issue.H{`step`: a, `name`: k, `parameterType`: field}))
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
	panic(a.Error(px.AttributeNotFound, issue.H{`type`: tp, `name`: name}))
}

func (a *step) getTypeName() string {
	keys := a.hash.Keys().Select(func(key px.Value) bool { return types.TypeNamePattern.MatchString(key.String()) })
	if keys.Len() == 1 {
		return keys.At(0).String()
	}
	// This should never happen since the step is identified by the presence of a unique type key.
	panic(fmt.Errorf(`step has no type name key`))
}

func (a *step) getState(c px.Context, parameters []serviceapi.Parameter) (px.OrderedMap, []serviceapi.Parameter) {
	de, ok := a.hash.Get4(a.getTypeName())
	if !ok {
		return px.EmptyMap, []serviceapi.Parameter{}
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
	panic(a.Error(wf.FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `Hash`, `actual`: de}))
}

func stripOptional(t px.Type) px.Type {
	if ot, ok := t.(*types.OptionalType); ok {
		return stripOptional(ot.ContainedType())
	}
	return t
}

func (a *step) resolveParameters(v px.Value, at px.Type, parameters []serviceapi.Parameter) (px.Value, []serviceapi.Parameter) {
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
							parameters[i] = serviceapi.NewParameter(vn, ``, at, nil)
						}
						found = true
						break
					}
				}
				if !found {
					parameters = append(parameters, serviceapi.NewParameter(vn, ``, at, nil))
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
	n := a.getTypeName()
	if t, ok := px.Load(c, px.NewTypedName(px.NsType, n)); ok {
		if pt, ok := t.(px.ObjectType); ok {
			a.rt = pt
			return pt
		}
		panic(a.Error(wf.FieldTypeMismatch, issue.H{`step`: a, `field`: n, `expected`: `ObjectType`, `actual`: t}))
	}
	panic(a.Error(px.UnresolvedType, issue.H{`typeString`: n}))
}

func (a *step) getStringProperty(properties px.OrderedMap, field string) (string, bool) {
	v, ok := properties.Get4(field)
	if !ok {
		return ``, false
	}

	if s, ok := v.(px.StringValue); ok {
		return s.String(), true
	}
	panic(a.Error(wf.FieldTypeMismatch, issue.H{`step`: a, `field`: field, `expected`: `String`, `actual`: v.PType()}))
}
