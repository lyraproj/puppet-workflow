package puppetwf

import (
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/evaluator"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/servicesdk/wf"
)

type PuppetActivity interface {
	Activity() wf.Activity

	Name() string
}

type puppetActivity struct {
	name       string
	parent     *puppetActivity
	expression parser.Expression
	properties px.OrderedMap
	activity   wf.Activity
}

func init() {
	evaluator.NewPuppetActivity = func(c pdsl.EvaluationContext, expression *parser.ActivityExpression) evaluator.Resolvable {
		return newActivity(c, nil, expression)
	}
}

func (a *puppetActivity) Activity() wf.Activity {
	return a.activity
}

func (a *puppetActivity) Name() string {
	return a.name
}

func (a *puppetActivity) Resolve(c px.Context) {
	if a.activity == nil {
		switch a.Style() {
		case `stateHandler`:
			a.activity = wf.NewStateHandler(c, a.buildStateHandler)
		case `workflow`:
			a.activity = wf.NewWorkflow(c, a.buildWorkflow)
		case `resource`:
			a.activity = wf.NewResource(c, a.buildResource)
		case `action`:
			a.activity = wf.NewAction(c, a.buildAction)
		}
	}
}

func (a *puppetActivity) buildActivity(builder wf.Builder) {
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Input(a.extractParameters(a.properties, `input`, a.inferInput)...)
	builder.Output(a.extractParameters(a.properties, `output`, func() []px.Parameter { return []px.Parameter{} })...)
}

func newActivity(c pdsl.EvaluationContext, parent *puppetActivity, ex *parser.ActivityExpression) *puppetActivity {
	ca := &puppetActivity{parent: parent, expression: ex}
	if props := ex.Properties(); props != nil {
		v := pdsl.Evaluate(c, props)
		dh, ok := v.(*types.Hash)
		if !ok {
			panic(px.Error(FieldTypeMismatch, issue.H{`field`: `properties`, `expected`: `Hash`, `actual`: v.PType()}))
		}
		ca.properties = dh
	} else {
		ca.properties = px.EmptyMap
	}
	sgs := strings.Split(ex.Name(), `::`)
	ca.name = sgs[len(sgs)-1]
	return ca
}

func (a *puppetActivity) buildStateHandler(builder wf.StateHandlerBuilder) {
	a.buildActivity(builder)
	builder.API(a.getAPI(builder.Context(), builder.GetInput()))
}

func (a *puppetActivity) buildResource(builder wf.ResourceBuilder) {
	a.buildActivity(builder)
	c := builder.Context().(pdsl.EvaluationContext)
	builder.State(&state{ctx: c, stateType: a.getResourceType(c), unresolvedState: a.getState(c)})
	if extId, ok := a.getStringProperty(`externalId`); ok {
		builder.ExternalId(extId)
	}
}

func (a *puppetActivity) buildAction(builder wf.ActionBuilder) {
	if fd, ok := a.expression.(*parser.FunctionDefinition); ok {
		fn := evaluator.NewPuppetFunction(fd)
		fn.Resolve(builder.Context())
		builder.Name(fn.Name())
		builder.Input(fn.Parameters()...)
		builder.Doer(&do{name: fn.Name(), body: fd.Body(), parameters: fn.Parameters()})
		s := fn.Signature()
		rt := s.ReturnType()
		if rt != nil {
			if st, ok := rt.(*types.StructType); ok {
				es := st.Elements()
				ps := make([]px.Parameter, len(es))
				for i, e := range es {
					ps[i] = px.NewParameter(e.Name(), e.Value(), nil, false)
				}
				builder.Output(ps...)
			}
		}
		return
	}
	if ae, ok := a.expression.(*parser.ActivityExpression); ok {
		a.buildActivity(builder)
		builder.Doer(&do{name: builder.GetName(), body: ae.Definition(), parameters: builder.GetInput()})
	}
}

func (a *puppetActivity) buildWorkflow(builder wf.WorkflowBuilder) {
	a.buildActivity(builder)
	de := a.expression.(*parser.ActivityExpression).Definition()
	if de == nil {
		return
	}

	block, ok := de.(*parser.BlockExpression)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
	}

	// Block should only contain activity expressions or something is wrong.
	for _, stmt := range block.Statements() {
		if as, ok := stmt.(*parser.ActivityExpression); ok {
			a.workflowActivity(builder, as)
		} else if fn, ok := stmt.(*parser.FunctionDefinition); ok {
			ac := &puppetActivity{parent: a, expression: fn}
			builder.Action(ac.buildAction)
		} else {
			panic(px.Error(NotActivity, issue.H{`actual`: stmt}))
		}
	}
}

func (a *puppetActivity) workflowActivity(builder wf.WorkflowBuilder, as *parser.ActivityExpression) {
	ac := newActivity(builder.Context().(pdsl.EvaluationContext), a, as)
	if _, ok := ac.properties.Get4(`iteration`); ok {
		builder.Iterator(ac.buildIterator)
	} else {
		switch as.Style() {
		case parser.ActivityStyleStateHandler:
			builder.StateHandler(ac.buildStateHandler)
		case parser.ActivityStyleWorkflow:
			builder.Workflow(ac.buildWorkflow)
		case parser.ActivityStyleResource:
			builder.Resource(ac.buildResource)
		case parser.ActivityStyleAction:
			builder.Action(ac.buildAction)
		}
	}
}

func (a *puppetActivity) Style() string {
	if _, ok := a.expression.(*parser.FunctionDefinition); ok {
		return `action`
	}
	return string(a.expression.(*parser.ActivityExpression).Style())
}

func (a *puppetActivity) inferInput() []px.Parameter {
	// TODO:
	return []px.Parameter{}
}

func noParamsFunc() []px.Parameter {
	return []px.Parameter{}
}

func (a *puppetActivity) buildIterator(builder wf.IteratorBuilder) {
	v, _ := a.properties.Get4(`iteration`)
	iteratorDef, ok := v.(*types.Hash)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`field`: `iteration`, `expected`: `Hash`, `actual`: v.PType()}))
	}

	v = iteratorDef.Get5(`function`, px.Undef)
	style, ok := v.(px.StringValue)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`field`: `iteration.style`, `expected`: `String`, `actual`: v}))
	}
	if name, ok := iteratorDef.Get4(`name`); ok {
		builder.Name(name.String())
	}
	builder.Style(wf.NewIterationStyle(style.String()))
	builder.Over(a.extractParameters(iteratorDef, `params`, noParamsFunc)...)
	builder.Variables(a.extractParameters(iteratorDef, `vars`, noParamsFunc)...)

	switch a.Style() {
	case `stateHandler`:
		builder.StateHandler(a.buildStateHandler)
	case `workflow`:
		builder.Workflow(a.buildWorkflow)
	case `resource`:
		builder.Resource(a.buildResource)
	case `action`:
		builder.Action(a.buildAction)
	}
}

func (a *puppetActivity) getAPI(c px.Context, input []px.Parameter) px.PuppetObject {
	var de parser.Expression
	if ae, ok := a.expression.(*parser.ActivityExpression); ok {
		de = ae.Definition()
	} else {
		// The block is the function
		return NewDo(a.Name(), input, a.expression)
	}
	if de == nil {
		panic(c.Error(a.expression, NoDefinition, issue.NO_ARGS))
	}

	block, ok := de.(*parser.BlockExpression)
	if !ok {
		panic(c.Error(de, FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
	}

	// Block must only consist of functions the functions create, read, update, and delete.
	var create, read, update, remove px.InvokableValue
	for _, e := range block.Statements() {
		if fd, ok := e.(*parser.FunctionDefinition); ok {
			switch fd.Name() {
			case `create`:
				create = createFunction(c, fd)
				continue
			case `read`:
				read = createFunction(c, fd)
				continue
			case `update`:
				update = createFunction(c, fd)
				continue
			case `delete`:
				remove = createFunction(c, fd)
				continue
			default:
				panic(c.Error(e, InvalidFunction, issue.H{`name`: fd.Name()}))
			}
		}
		panic(c.Error(e, FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `function`, `actual`: e}))
	}

	missing := ``
	if create == nil {
		missing = `create`
	} else if read == nil {
		missing = `read`
	} else if remove == nil {
		missing = `delete`
	}
	if missing != `` {
		panic(c.Error(block, MissingRequiredFunction, issue.H{`function`: missing}))
	}
	if update == nil {
		return NewCRD(a.Name(), create, read, remove)
	}
	return NewCRUD(a.Name(), create, read, update, remove)
}

func createFunction(c px.Context, fd *parser.FunctionDefinition) evaluator.PuppetFunction {
	f := evaluator.NewPuppetFunction(fd)
	f.Resolve(c)
	return f
}

func (a *puppetActivity) getWhen() string {
	if when, ok := a.getStringProperty(`when`); ok {
		return when
	}
	return ``
}

func (a *puppetActivity) extractParameters(props px.OrderedMap, field string, dflt func() []px.Parameter) []px.Parameter {
	if props == nil {
		return dflt()
	}

	v, ok := props.Get4(field)
	if !ok {
		return dflt()
	}

	ia, ok := v.(*types.Array)
	if !ok {
		panic(px.Error(FieldTypeMismatch, issue.H{`field`: field, `expected`: `Array`, `actual`: v.PType()}))
	}

	params := make([]px.Parameter, ia.Len())
	ia.EachWithIndex(func(v px.Value, i int) {
		if p, ok := v.(px.Parameter); ok {
			params[i] = p
		} else {
			panic(px.Error(ElementNotParameter, issue.H{`type`: p.PType(), `field`: field}))
		}
	})
	return params
}

func (a *puppetActivity) getState(c pdsl.EvaluationContext) px.OrderedMap {
	ae, ok := a.expression.(*parser.ActivityExpression)
	if !ok {
		return px.EmptyMap
	}
	de := ae.Definition()
	if de == nil {
		return px.EmptyMap
	}

	if hash, ok := de.(*parser.LiteralHash); ok {
		// Transform all variable references to Deferred expressions
		return pdsl.Evaluate(c, hash).(px.OrderedMap)
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `Hash`, `actual`: de}))
}

func (a *puppetActivity) getResourceType(c px.Context) px.ObjectType {
	n := a.Name()
	if a.properties != nil {
		if tv, ok := a.properties.Get4(`type`); ok {
			if t, ok := tv.(px.ObjectType); ok {
				return t
			}
			if s, ok := tv.(px.StringValue); ok {
				n = s.String()
			} else {
				panic(px.Error(FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `Variant[String,ObjectType]`, `actual`: tv}))
			}
		} else {
			ts := a.getTypeSpace()
			if ts != `` {
				n = ts + `::` + wf.LeafName(n)
			}
		}
	}
	tn := px.NewTypedName(px.NsType, n)
	if t, ok := px.Load(c, tn); ok {
		if pt, ok := t.(px.ObjectType); ok {
			return pt
		}
		panic(px.Error(FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `ObjectType`, `actual`: t}))
	}
	panic(px.Error(px.UnresolvedType, issue.H{`typeString`: tn.Name()}))
}

func (a *puppetActivity) getTypeSpace() string {
	if ts, ok := a.getStringProperty(`typespace`); ok {
		return ts
	}
	if a.parent != nil {
		return a.parent.getTypeSpace()
	}
	return ``
}

func (a *puppetActivity) getStringProperty(field string) (string, bool) {
	if a.properties == nil {
		return ``, false
	}

	v, ok := a.properties.Get4(field)
	if !ok {
		return ``, false
	}

	if s, ok := v.(px.StringValue); ok {
		return s.String(), true
	}
	panic(px.Error(FieldTypeMismatch, issue.H{`field`: field, `expected`: `String`, `actual`: v.PType()}))
}
