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
	origin string
	parent *step
	hash   *yaml.Value
	rt     px.ObjectType
	step   wf.Step
}

const kindWorkflow = 1
const kindResource = 2
const kindCollect = 3
const kindCall = 4

func CreateStep(c px.Context, file string, content []byte) wf.Step {
	loc := issue.NewLocation(file, 0, 0)
	c.StackPush(loc)
	defer c.StackPop()

	v := yaml.UnmarshalWithPositions(c, content)
	_, ok := v.Value.(px.OrderedMap)
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
	a := newStep(name, file, nil, v)
	switch a.stepKind() {
	case kindWorkflow:
		return wf.NewWorkflow(c, func(wb wf.WorkflowBuilder) {
			a.buildWorkflow(wb)
		})
	case kindCollect:
		return wf.NewIterator(c, func(ib wf.IteratorBuilder) {
			a.buildIterator(ib)
		})
	case kindCall:
		return wf.NewCall(c, func(rb wf.CallBuilder) {
			a.buildCall(rb)
		})
	default:
		return wf.NewResource(c, func(wb wf.ResourceBuilder) {
			a.buildResource(wb)
		})
	}
}

func newStep(name, origin string, parent *step, ex *yaml.Value) *step {
	ca := &step{origin: origin, parent: parent, hash: ex}
	sgs := strings.Split(name, `::`)
	ca.name = sgs[len(sgs)-1]
	return ca
}

func (a *step) amendError(v *yaml.Value) {
	if r := recover(); r != nil {
		if re, ok := r.(issue.Reported); ok {
			// Avoid nesting of errors that stem from the same file
			if loc := re.Location(); loc != nil && loc.File() == a.origin {
				panic(re)
			}
		}
		panic(issue.NewNested(wf.StepBuildError, issue.H{`step`: a.Label()}, a.location(v), wf.ToError(r)))
	}
}

func (a *step) location(v *yaml.Value) issue.Location {
	return issue.NewLocation(a.origin, v.Line, v.Column)
}

func (a *step) Error(v *yaml.Value, code issue.Code, args issue.H) issue.Reported {
	return px.Error2(a.location(v), code, args)
}

func (a *step) stepKind() int {
	m, ok := a.hash.Value.(px.OrderedMap)
	if ok {
		if m.IncludesKey2(`steps`) {
			return kindWorkflow
		}
		if m.IncludesKey2(`each`) || m.IncludesKey2(`eachPair`) || m.IncludesKey2(`times`) || m.IncludesKey2(`range`) {
			return kindCollect
		}
		if m.IncludesKey2(`call`) {
			return kindCall
		}
		// hash must include exactly one key which is a type name
		if m.Keys().Select(func(key px.Value) bool { return types.TypeNamePattern.MatchString(key.String()) }).Len() == 1 {
			return kindResource
		}
	}
	panic(a.Error(a.hash, wf.NotStep, issue.NoArgs))
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
	defer a.amendError(a.hash)
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Parameters(a.extractParameters(builder.Context(), `parameters`, false)...)
	builder.Returns(a.extractParameters(builder.Context(), `returns`, true)...)
}

func (a *step) buildResource(builder wf.ResourceBuilder) {
	defer a.amendError(a.hash)

	c := builder.Context()

	builder.Name(a.Name())
	builder.When(a.getWhen())

	st, parameters := a.getState(c, a.extractParameters(builder.Context(), `parameters`, false))
	builder.Parameters(parameters...)
	builder.Returns(a.extractParameters(builder.Context(), `returns`, true)...)
	builder.State(&state{ctx: c, stateType: a.getResourceType(c), unresolvedState: st})

	if extId, ok := a.getStringProperty(a.hash, `external_id`); ok {
		builder.ExternalId(extId)
	}
}

func (a *step) buildCall(builder wf.CallBuilder) {
	defer a.amendError(a.hash)

	c := builder.Context()

	builder.Name(a.Name())
	builder.When(a.getWhen())

	builder.Parameters(a.extractParameters(c, `parameters`, true)...)
	builder.Returns(a.extractParameters(c, `returns`, true)...)

	// Step will call a step with the same name by default.
	call := a.Name()
	if ra, ok := a.getStringProperty(a.hash, `call`); ok && ra != `` {
		call = ra
	}
	builder.CallTo(call)
}

func (a *step) buildWorkflow(builder wf.WorkflowBuilder) {
	a.buildStep(builder)
	de, ok := getProperty(a.hash, `steps`)
	if !ok {
		return
	}

	block, ok := de.Value.(px.OrderedMap)
	if !ok {
		panic(a.Error(de, wf.FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `CodeBlock`, `actual`: de.Value}))
	}

	// Block should only contain step expressions or something is wrong.
	block.EachPair(func(k, v px.Value) {
		vl := v.(*yaml.Value)
		if _, ok := vl.Value.(px.OrderedMap); ok {
			a.workflowStep(builder, k.String(), vl)
		} else {
			panic(a.Error(vl, wf.NotStep, issue.H{`actual`: vl.Value}))
		}
	})
}

func (a *step) workflowStep(builder wf.ChildBuilder, name string, as *yaml.Value) {
	ac := newStep(name, a.origin, a, as)
	switch ac.stepKind() {
	case kindCollect:
		builder.Iterator(ac.buildIterator)
	case kindWorkflow:
		builder.Workflow(ac.buildWorkflow)
	case kindCall:
		builder.Call(ac.buildCall)
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
	case kindCall:
		return `call`
	default:
		return `resource`
	}
}

func (a *step) buildIterator(builder wf.IteratorBuilder) {
	a.buildStep(builder)

	var vl, ol *yaml.Value
	var ok bool
	if vl, ok = getProperty(a.hash, `each`); ok {
		builder.Style(wf.IterationStyleEach)
		ol = vl
	} else if vl, ok = getProperty(a.hash, `eachPair`); ok {
		builder.Style(wf.IterationStyleEachPair)
		ol = vl
	} else if vl, ok = getProperty(a.hash, `times`); ok {
		builder.Style(wf.IterationStyleTimes)
		ol = vl
	} else if vl, ok = getProperty(a.hash, `range`); ok {
		builder.Style(wf.IterationStyleRange)
		ol = vl
	}
	if vl, ok = getProperty(a.hash, `into`); ok {
		builder.Into(vl.Value.String())
	}
	over, parameters := a.resolveParameters(ol, types.DefaultAnyType(), []serviceapi.Parameter{})
	builder.Over(over)
	builder.Parameters(parameters...)
	builder.Variables(a.extractParameters(builder.Context(), `as`, false)...)

	if vl, ok = getProperty(a.hash, `step`); ok {
		if _, ok = vl.Value.(px.OrderedMap); ok {
			a.workflowStep(builder, a.Name(), vl)
		} else {
			panic(a.Error(vl, wf.NotStep, issue.H{`actual`: vl.Value.PType()}))
		}
	}
}

func (a *step) getWhen() string {
	if when, ok := a.getStringProperty(a.hash, `when`); ok {
		return when
	}
	return ``
}

func (a *step) extractParameters(c px.Context, field string, aliased bool) []serviceapi.Parameter {
	vl, ok := getProperty(a.hash, field)
	if !ok {
		return []serviceapi.Parameter{}
	}
	v := vl.Value

	if ph, ok := v.(px.OrderedMap); ok {
		params := make([]serviceapi.Parameter, 0, ph.Len())
		ph.EachPair(func(k, v px.Value) {
			params = append(params, a.makeParameter(c, field, k.(*yaml.Value), v.(*yaml.Value), aliased))
		})
		return params
	}

	if _, ok := v.(px.StringValue); ok {
		// Allow single name as a convenience
		v = types.WrapValues([]px.Value{vl})
	}

	if pa, ok := v.(*types.Array); ok {
		// List of names.
		params := make([]serviceapi.Parameter, pa.Len())
		pa.EachWithIndex(func(e px.Value, i int) {
			el := e.(*yaml.Value)
			if ne, ok := el.Value.(px.StringValue); ok {
				n := ne.String()
				if aliased && a.stepKind() == kindResource {
					// Names must match attribute names
					params[i] = serviceapi.NewParameter(n, ``, a.attributeType(c, n, el), nil)
				} else {
					params[i] = serviceapi.NewParameter(n, ``, types.DefaultAnyType(), nil)
				}
			} else {
				panic(a.Error(el, wf.BadParameter, issue.H{`step`: a, `name`: e, `parameterType`: field}))
			}
		})
		return params
	}
	panic(a.Error(vl, wf.FieldTypeMismatch, issue.H{`step`: a, `field`: field, `expected`: `Hash`, `actual`: v.PType()}))
}

var varNamePattern = regexp.MustCompile(`\A[a-z]\w*(?:\.[a-z]\w*)*\z`)

func (a *step) makeParameter(c px.Context, field string, kl, vl *yaml.Value, aliased bool) (param serviceapi.Parameter) {
	k := kl.Value
	if n, ok := k.(px.StringValue); ok {
		name := n.String()
		switch v := vl.Value.(type) {
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
						param = serviceapi.NewParameter(name, s, a.attributeType(c, s, vl), nil)
					} else if a.stepKind() == kindCall {
						param = serviceapi.NewParameter(name, s, types.DefaultAnyType(), nil)
					}
				}
			}
		case px.OrderedMap:
			var tp px.Type
			if tul, ok := getProperty(vl, `type`); ok {
				if ts, ok := tul.Value.(px.StringValue); ok {
					defer func() {
						// Catch pcore type parse error and offset it with the types location in YAML.
						if r := recover(); r != nil {
							if re, ok := r.(issue.Reported); ok && re.Code() == types.ParseError {
								panic(re.OffsetByLocation(a.location(tul)))
							}
							panic(r)
						}
					}()

					str := ts.String()
					t := types.ParseFile(a.origin, str)
					if rt, ok := t.(px.ResolvableType); ok {
						tp = rt.Resolve(c)
					} else {
						panic(fmt.Errorf(`expression "%s" does no resolve to a Type`, str))
					}
				} else {
					panic(a.Error(tul, wf.FieldTypeMismatch, issue.H{`step`: a, `field`: `type`, `expected`: `String`, `actual`: tul.Value.PType()}))
				}
			}

			var val px.Value
			if lul, ok := getProperty(vl, `lookup`); ok {
				var args []px.Value
				lu := lul.Value
				if a, ok := lu.(*types.Array); ok {
					args = a.AppendTo(make([]px.Value, 0, a.Len()))
				} else {
					args = []px.Value{lu}
				}
				val = types.NewDeferred(`lookup`, args...)
			} else {
				if vl, ok = getProperty(vl, `value`); ok {
					val = vl.Unwrap()
					if tp != nil {
						defer a.amendError(vl)
						val = types.CoerceTo(c, fmt.Sprintf(`%s:%s parameter value`, a.Label(), name), tp, val)
					}
				}
			}

			if tp == nil {
				if val != nil && !val.Equals(px.Undef, nil) {
					tp = val.PType()
				} else {
					tp = types.DefaultAnyType()
				}
			}

			alias := ``
			if al, ok := v.Get4(`alias`); ok {
				alias = al.String()
			}
			param = serviceapi.NewParameter(name, alias, tp, val)
		}
	}
	if param == nil {
		panic(a.Error(vl, wf.BadParameter, issue.H{`step`: a, `name`: k, `parameterType`: field}))
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

func (a *step) attributeType(c px.Context, name string, k *yaml.Value) px.Type {
	tp := a.getResourceType(c)
	if at, ok := getAttributeType(tp, name); ok {
		return at
	}
	panic(a.Error(k, px.AttributeNotFound, issue.H{`type`: tp, `name`: name}))
}

func (a *step) getTypeName() *yaml.Value {
	keys := a.hash.Value.(px.OrderedMap).Keys().Select(func(key px.Value) bool {
		return types.TypeNamePattern.MatchString(key.(*yaml.Value).Value.String())
	})
	if keys.Len() == 1 {
		return keys.At(0).(*yaml.Value)
	}
	// This should never happen since the step is identified by the presence of a unique type key.
	panic(fmt.Errorf(`step has no type name key`))
}

func (a *step) getState(c px.Context, parameters []serviceapi.Parameter) (px.OrderedMap, []serviceapi.Parameter) {
	tn := a.getTypeName()
	de, ok := getProperty(a.hash, tn.Value.String())
	if !ok {
		return px.EmptyMap, []serviceapi.Parameter{}
	}

	if hash, ok := de.Value.(px.OrderedMap); ok {
		// Ensure that hash conforms to init of type with respect to attribute names
		// and transform all variable references to Deferred expressions
		es := make([]*types.HashEntry, 0, hash.Len())
		if hash.AllPairs(func(k, v px.Value) bool {
			kl := k.(*yaml.Value)
			if sv, ok := kl.Unwrap().(px.StringValue); ok {
				s := sv.String()
				if varNamePattern.MatchString(s) {
					at := a.attributeType(c, s, kl)
					v, parameters = a.resolveParameters(v.(*yaml.Value), at, parameters)
					es = append(es, types.WrapHashEntry(sv, v))
					return true
				}
			}
			return false
		}) {
			return types.WrapHash(es), parameters
		}
	}
	panic(a.Error(de, wf.FieldTypeMismatch, issue.H{`step`: a, `field`: `definition`, `expected`: `Hash`, `actual`: de.Value}))
}

func stripOptional(t px.Type) px.Type {
	if ot, ok := t.(*types.OptionalType); ok {
		return stripOptional(ot.ContainedType())
	}
	return t
}

func (a *step) resolveParameters(vl *yaml.Value, at px.Type, parameters []serviceapi.Parameter) (px.Value, []serviceapi.Parameter) {
	var v px.Value
	switch vr := vl.Value.(type) {
	case px.StringValue:
		v = vr
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
			vr.EachPair(func(kv, av px.Value) {
				kv = kv.(*yaml.Value).Unwrap()
				al := av.(*yaml.Value)
				if at, ok := getAttributeType(ot, kv.String()); ok {
					av, parameters = a.resolveParameters(al, at, parameters)
				} else {
					av, parameters = a.resolveParameters(al, types.DefaultAnyType(), parameters)
				}
				es = append(es, types.WrapHashEntry(kv, av))
			})
		} else if ht, ok := nta.(*types.HashType); ok {
			et := ht.ValueType()
			vr.EachPair(func(kv, av px.Value) {
				kv = kv.(*yaml.Value).Unwrap()
				al := av.(*yaml.Value)
				av, parameters = a.resolveParameters(al, et, parameters)
				es = append(es, types.WrapHashEntry(kv, av))
			})
		} else if st, ok := nta.(*types.StructType); ok {
			hm := st.HashedMembers()
			vr.EachPair(func(kv, av px.Value) {
				kv = kv.(*yaml.Value).Unwrap()
				al := av.(*yaml.Value)
				if m, ok := hm[kv.String()]; ok {
					av, parameters = a.resolveParameters(al, m.Value(), parameters)
				} else {
					av, parameters = a.resolveParameters(al, types.DefaultAnyType(), parameters)
				}
				es = append(es, types.WrapHashEntry(kv, av))
			})
		} else {
			et := types.DefaultAnyType()
			vr.EachPair(func(kv, av px.Value) {
				kv = kv.(*yaml.Value).Unwrap()
				al := av.(*yaml.Value)
				av, parameters = a.resolveParameters(al, et, parameters)
				es = append(es, types.WrapHashEntry(kv, av))
			})
		}
		v = types.WrapHash(es)
	case px.List:
		es := make([]px.Value, vr.Len())
		nta := stripOptional(at)
		if st, ok := nta.(*types.ArrayType); ok {
			et := st.ElementType()
			vr.EachWithIndex(func(ev px.Value, i int) {
				es[i], parameters = a.resolveParameters(ev.(*yaml.Value), et, parameters)
			})
		} else if tt, ok := nta.(*types.TupleType); ok {
			ts := tt.Types()
			vr.EachWithIndex(func(ev px.Value, i int) {
				el := ev.(*yaml.Value)
				if i < len(ts) {
					ev, parameters = a.resolveParameters(el, ts[i], parameters)
				} else {
					ev, parameters = a.resolveParameters(el, types.DefaultAnyType(), parameters)
				}
				es[i] = ev
			})
		} else {
			et := types.DefaultAnyType()
			vr.EachWithIndex(func(ev px.Value, i int) {
				ev, parameters = a.resolveParameters(ev.(*yaml.Value), et, parameters)
				es[i] = ev
			})
		}
		v = types.WrapValues(es)
	default:
		v = vr
	}
	return v, parameters
}

func (a *step) getResourceType(c px.Context) px.ObjectType {
	if a.rt != nil {
		return a.rt
	}
	nl := a.getTypeName()
	defer a.amendError(nl)
	n := nl.Value.String()
	if t, ok := px.Load(c, px.NewTypedName(px.NsType, n)); ok {
		if pt, ok := t.(px.ObjectType); ok {
			a.rt = pt
			return pt
		}
		panic(a.Error(nl, wf.FieldTypeMismatch, issue.H{`step`: a, `field`: n, `expected`: `ObjectType`, `actual`: t}))
	}
	panic(a.Error(nl, px.UnresolvedType, issue.H{`typeString`: n}))
}

func (a *step) getStringProperty(properties *yaml.Value, field string) (string, bool) {
	vl, ok := getProperty(properties, field)
	if !ok {
		return ``, false
	}
	v := vl.Unwrap()

	if s, ok := v.(px.StringValue); ok {
		return s.String(), true
	}
	panic(a.Error(vl, wf.FieldTypeMismatch, issue.H{`step`: a, `field`: field, `expected`: `String`, `actual`: v.PType()}))
}

func getProperty(properties *yaml.Value, key string) (*yaml.Value, bool) {
	if properties != nil {
		if hash, ok := properties.Value.(px.OrderedMap); ok {
			var v px.Value
			if v, ok = hash.Get4(key); ok {
				return v.(*yaml.Value), true
			}
		}
	}
	return nil, false
}
