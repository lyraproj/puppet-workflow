package puppetwf

import (
	"io"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/puppet-evaluator/pdsl"

	"github.com/lyraproj/puppet-evaluator/evaluator"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-parser/parser"
)

var doType px.Type
var crdType px.Type
var crudType px.Type

func init() {
	doType = px.NewObjectType(`Puppet::Do`, `{
		attributes => {
      name => String
    },
    functions => {
      do => Callable[[RichData,1], RichData]
    }
  }`)

	crdType = px.NewObjectType(`Puppet::CRD`, `{
		attributes => {
      name => String
    },
    functions => {
      create => Callable[[Object], Tuple[Object,String]],
      read   => Callable[[String], Object],
      delete => Callable[[String], Boolean]
    }
  }`)

	crudType = px.NewObjectType(`Puppet::CRUD`, `Puppet::CRD{
    functions => {
      update => Callable[[String, Object], Object]
    }
  }`)
}

type do struct {
	name       string
	parameters []px.Parameter
	body       parser.Expression
}

func (c *do) Name() string {
	return c.name
}

func (c *do) String() string {
	return px.ToString(c)
}

func (c *do) Equals(other interface{}, guard px.Guard) bool {
	return c == other
}

func (c *do) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(c, format, bld, g)
}

func (c *do) PType() px.Type {
	return doType
}

func (c *do) Get(key string) (px.Value, bool) {
	if key == `name` {
		return types.WrapString(c.name), true
	}
	return nil, false
}

func (c *do) InitHash() px.OrderedMap {
	return px.SingletonMap(`name`, types.WrapString(c.name))
}

func (c *do) Call(ctx px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	if method.Name() != `do` {
		return nil, false
	}
	if block != nil {
		panic(px.Error(px.IllegalArguments, issue.H{`function`: c.name, `message`: `nested lambdas are not supported`}))
	}

	ok = true
	defer func() {
		if err := recover(); err != nil {
			switch err := err.(type) {
			case *errors.NextIteration:
				result = err.Value()
			case *errors.Return:
				result = err.Value()
			default:
				panic(err)
			}
		}
	}()
	result = evaluator.CallBlock(ctx.(pdsl.EvaluationContext), c.name, c.parameters, method.Type().(*types.CallableType), c.body, args)
	return
}

type crd struct {
	name   string
	create px.InvokableValue
	read   px.InvokableValue
	delete px.InvokableValue
}

func (c *crd) Name() string {
	return c.name
}

func (c *crd) String() string {
	return px.ToString(c)
}

func (c *crd) Equals(other interface{}, guard px.Guard) bool {
	return c == other
}

func (c *crd) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(c, format, bld, g)
}

func (c *crd) PType() px.Type {
	return crdType
}

func (c *crd) Get(key string) (px.Value, bool) {
	if key == `name` {
		return types.WrapString(c.name), true
	}
	return nil, false
}

func (c *crd) InitHash() px.OrderedMap {
	return px.SingletonMap(`name`, types.WrapString(c.name))
}

func (c *crd) Call(ctx px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	var f px.InvokableValue
	switch method.Name() {
	case `create`:
		f = c.create
	case `read`:
		f = c.read
	case `delete`:
		f = c.delete
	default:
		return nil, false
	}
	return f.Call(ctx, block, args...), true
}

type crud struct {
	crd
	update px.InvokableValue
}

func (c *crud) String() string {
	return px.ToString(c)
}

func (c *crud) PType() px.Type {
	return crudType
}

func (c *crud) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(c, format, bld, g)
}

func (c *crud) Call(ctx px.Context, method px.ObjFunc, args []px.Value, block px.Lambda) (result px.Value, ok bool) {
	if method.Name() == `update` {
		return c.update.Call(ctx, block, args...), true
	}
	return c.crd.Call(ctx, method, args, block)
}

func NewDo(name string, parameters []px.Parameter, block parser.Expression) px.PuppetObject {
	return &do{name, parameters, block}
}

func NewCRD(name string, create, read, delete px.InvokableValue) px.PuppetObject {
	return &crd{name, create, read, delete}
}

func NewCRUD(name string, create, read, update, delete px.InvokableValue) px.PuppetObject {
	return &crud{crd{name, create, read, delete}, update}
}
