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

type PuppetStep interface {
	Step() wf.Step

	Name() string
}

type puppetStep struct {
	name       string
	parent     *puppetStep
	expression parser.Expression
	properties px.OrderedMap
	step       wf.Step
}

func init() {
	evaluator.NewPuppetStep = func(c pdsl.EvaluationContext, expression *parser.StepExpression) evaluator.Resolvable {
		return newStep(c, nil, expression)
	}
}

func (a *puppetStep) Step() wf.Step {
	return a.step
}

func (a *puppetStep) Name() string {
	return a.name
}

func (a *puppetStep) Resolve(c px.Context) {
	if a.step == nil {
		switch a.Style() {
		case `stateHandler`:
			a.step = wf.NewStateHandler(c, a.buildStateHandler)
		case `workflow`:
			a.step = wf.NewWorkflow(c, a.buildWorkflow)
		case `resource`:
			a.step = wf.NewResource(c, a.buildResource)
		case `action`:
			a.step = wf.NewAction(c, a.buildAction)
		}
	}
}

func (a *puppetStep) buildStep(builder wf.Builder) {
	builder.Name(a.Name())
	builder.When(a.getWhen())
	builder.Parameters(a.extractParameters(a.properties, `parameters`, a.inferParameters)...)
	builder.Returns(a.extractParameters(a.properties, `returns`, func() []px.Parameter { return []px.Parameter{} })...)
}

func newStep(c pdsl.EvaluationContext, parent *puppetStep, ex *parser.StepExpression) *puppetStep {
	ca := &puppetStep{parent: parent, expression: ex}
	if props := ex.Properties(); props != nil {
		v := pdsl.Evaluate(c, props)
		dh, ok := v.(*types.Hash)
		if !ok {
			panic(px.Error2(props, wf.FieldTypeMismatch, issue.H{`field`: `properties`, `expected`: `Hash`, `actual`: v.PType()}))
		}
		ca.properties = dh
	} else {
		ca.properties = px.EmptyMap
	}
	sgs := strings.Split(ex.Name(), `::`)
	ca.name = sgs[len(sgs)-1]
	return ca
}

func (a *puppetStep) buildStateHandler(builder wf.StateHandlerBuilder) {
	a.buildStep(builder)
	builder.API(a.getAPI(builder.Context(), builder.GetParameters()))
}

func (a *puppetStep) buildResource(builder wf.ResourceBuilder) {
	defer a.amendError()

	a.buildStep(builder)
	c := builder.Context().(pdsl.EvaluationContext)
	builder.State(&state{ctx: c, stateType: a.getResourceType(c), unresolvedState: a.getState(c)})
	if extId, ok := a.getStringProperty(`externalId`); ok {
		builder.ExternalId(extId)
	}
}

func (a *puppetStep) buildAction(builder wf.ActionBuilder) {
	defer a.amendError()

	if fd, ok := a.expression.(*parser.FunctionDefinition); ok {
		fn := evaluator.NewPuppetFunction(fd)
		fn.Resolve(builder.Context())
		builder.Name(fn.Name())
		builder.Parameters(fn.Parameters()...)
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
				builder.Returns(ps...)
			}
		}
		return
	}
	if ae, ok := a.expression.(*parser.StepExpression); ok {
		a.buildStep(builder)
		builder.Doer(&do{name: builder.GetName(), body: ae.Definition(), parameters: builder.GetParameters()})
	}
}

func (a *puppetStep) buildWorkflow(builder wf.WorkflowBuilder) {
	// Block should only contain step expressions or something is wrong.
	block := a.buildWorkflowInternals(builder)
	if block == nil {
		return
	}
	for _, stmt := range block.Statements() {
		if as, ok := stmt.(*parser.StepExpression); ok {
			a.workflowStep(builder, as)
		} else if fn, ok := stmt.(*parser.FunctionDefinition); ok {
			ac := &puppetStep{parent: a, expression: fn}
			builder.Action(ac.buildAction)
		} else {
			defer a.amendError()
			panic(a.Error(wf.NotStep, issue.H{`actual`: stmt}))
		}
	}
}

func (a *puppetStep) buildWorkflowInternals(builder wf.WorkflowBuilder) *parser.BlockExpression {
	defer a.amendError()

	a.buildStep(builder)
	de := a.expression.(*parser.StepExpression).Definition()
	if de == nil {
		return nil
	}

	if block, ok := de.(*parser.BlockExpression); ok {
		return block
	}
	panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
}

func (a *puppetStep) workflowStep(builder wf.WorkflowBuilder, as *parser.StepExpression) {
	ac := newStep(builder.Context().(pdsl.EvaluationContext), a, as)
	if _, ok := ac.properties.Get4(`iteration`); ok {
		builder.Iterator(ac.buildIterator)
	} else {
		switch as.Style() {
		case parser.StepStyleStateHandler:
			builder.StateHandler(ac.buildStateHandler)
		case parser.StepStyleWorkflow:
			builder.Workflow(ac.buildWorkflow)
		case parser.StepStyleResource:
			builder.Resource(ac.buildResource)
		case parser.StepStyleAction:
			builder.Action(ac.buildAction)
		}
	}
}

func (a *puppetStep) Style() string {
	if _, ok := a.expression.(*parser.FunctionDefinition); ok {
		return `action`
	}
	return string(a.expression.(*parser.StepExpression).Style())
}

func (a *puppetStep) inferParameters() []px.Parameter {
	// TODO:
	return []px.Parameter{}
}

func noParamsFunc() []px.Parameter {
	return []px.Parameter{}
}

func (a *puppetStep) buildIterator(builder wf.IteratorBuilder) {
	a.buildIteratorInternals(builder)
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

func (a *puppetStep) buildIteratorInternals(builder wf.IteratorBuilder) {
	defer a.amendError()

	v, _ := a.properties.Get4(`iteration`)
	iteratorDef, ok := v.(*types.Hash)
	if !ok {
		panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: `iteration`, `expected`: `Hash`, `actual`: v.PType()}))
	}

	v = iteratorDef.Get5(`function`, px.Undef)
	style, ok := v.(px.StringValue)
	if !ok {
		panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: `iteration.style`, `expected`: `String`, `actual`: v}))
	}
	if name, ok := iteratorDef.Get4(`name`); ok {
		builder.Name(name.String())
	}
	builder.Style(wf.NewIterationStyle(style.String()))
	builder.Over(a.extractOver(iteratorDef))
	vars := a.extractParameters(iteratorDef, `variable`, noParamsFunc)
	if len(vars) == 0 {
		vars = a.extractParameters(iteratorDef, `variables`, noParamsFunc)
	}
	builder.Variables(vars...)
}

func (a *puppetStep) extractOver(props px.OrderedMap) px.Value {
	if props == nil {
		return px.Undef
	}
	return props.Get5(`over`, px.Undef)
}

func (a *puppetStep) getAPI(c px.Context, parameters []px.Parameter) px.PuppetObject {
	var de parser.Expression
	if ae, ok := a.expression.(*parser.StepExpression); ok {
		de = ae.Definition()
	} else {
		// The block is the function
		return NewDo(a.Name(), parameters, a.expression)
	}
	if de == nil {
		panic(c.Error(a.expression, wf.NoDefinition, issue.NoArgs))
	}

	block, ok := de.(*parser.BlockExpression)
	if !ok {
		panic(c.Error(de, wf.FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `CodeBlock`, `actual`: de}))
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
				panic(c.Error(e, wf.InvalidFunction, issue.H{`name`: fd.Name()}))
			}
		}
		panic(c.Error(e, wf.FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `function`, `actual`: e}))
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
		panic(c.Error(block, wf.MissingRequiredFunction, issue.H{`function`: missing}))
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

func (a *puppetStep) getWhen() string {
	if when, ok := a.getStringProperty(`when`); ok {
		return when
	}
	return ``
}

func (a *puppetStep) extractParameters(props px.OrderedMap, field string, dflt func() []px.Parameter) []px.Parameter {
	if props == nil {
		return dflt()
	}

	v, ok := props.Get4(field)
	if !ok {
		return dflt()
	}

	ia, ok := v.(*types.Array)
	if !ok {
		panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: field, `expected`: `Array`, `actual`: v.PType()}))
	}

	params := make([]px.Parameter, ia.Len())
	ia.EachWithIndex(func(v px.Value, i int) {
		if p, ok := v.(px.Parameter); ok {
			params[i] = p
		} else {
			panic(a.Error(wf.ElementNotParameter, issue.H{`type`: p.PType(), `field`: field}))
		}
	})
	return params
}

func (a *puppetStep) getState(c pdsl.EvaluationContext) px.OrderedMap {
	ae, ok := a.expression.(*parser.StepExpression)
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
	panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: `definition`, `expected`: `Hash`, `actual`: de}))
}

func (a *puppetStep) getResourceType(c px.Context) px.ObjectType {
	if a.properties != nil {
		if tv, ok := a.properties.Get4(`type`); ok {
			if t, ok := tv.(px.ObjectType); ok {
				return t
			}
			if s, ok := tv.(px.StringValue); ok {
				n := s.String()
				if !types.TypeNamePattern.MatchString(n) {
					panic(px.Error(wf.InvalidTypeName, issue.H{`name`: n}))
				}
				if t, ok := px.Load(c, px.NewTypedName(px.NsType, n)); ok {
					if pt, ok := t.(px.ObjectType); ok {
						return pt
					}
					panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: `type`, `expected`: `ObjectType`, `actual`: t}))
				}
				panic(a.Error(px.UnresolvedType, issue.H{`typeString`: n}))
			}
			panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: `type`, `expected`: `Variant[String,ObjectType]`, `actual`: tv}))
		}
	}
	panic(a.Error(wf.MissingRequiredField, issue.H{`field`: `type`}))
}

func (a *puppetStep) Error(code issue.Code, args issue.H) issue.Reported {
	return px.Error2(a.expression, code, args)
}

func (a *puppetStep) getStringProperty(field string) (string, bool) {
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
	panic(a.Error(wf.FieldTypeMismatch, issue.H{`field`: field, `expected`: `String`, `actual`: v.PType()}))
}

func (a *puppetStep) amendError() {
	if r := recover(); r != nil {
		if rx, ok := r.(issue.Reported); ok {
			// Location and stack included in nested error
			r = issue.ErrorWithoutStack(wf.StepBuildError, issue.H{`step`: a.Name()}, nil, rx)
		} else {
			r = issue.NewNested(wf.StepBuildError, issue.H{`step`: a.Name()}, nil, wf.ToError(r))
		}
		panic(r)
	}
}
