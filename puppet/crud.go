package puppet

import (
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/impl"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
	"io"
)

var doType eval.Type
var crdType eval.Type
var crudType eval.Type

func init() {
	doType = eval.NewObjectType(`Puppet::Do`, `{
		attributes => {
      name => String
    },
    functions => {
      do => Callable[[RichData,1], RichData]
    }
  }`)

	crdType = eval.NewObjectType(`Puppet::CRD`, `{
		attributes => {
      name => String
    },
    functions => {
      create => Callable[[Object], Tuple[Object,String]],
      read   => Callable[[String], Object],
      delete => Callable[[String], Boolean]
    }
  }`)

	crudType = eval.NewObjectType(`Puppet::CRUD`, `Puppet::CRD{
    functions => {
      update => Callable[[String, Object], Object]
    }
  }`)
}

type do struct {
	name       string
	parameters []eval.Parameter
	body       parser.Expression
}

func (c *do) Name() string {
	return c.name
}

func (c *do) String() string {
	return eval.ToString(c)
}

func (c *do) Equals(other interface{}, guard eval.Guard) bool {
	return c == other
}

func (c *do) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(c, format, bld, g)
}

func (c *do) PType() eval.Type {
	return doType
}

func (c *do) Get(key string) (eval.Value, bool) {
	if key == `name` {
		return types.WrapString(c.name), true
	}
	return nil, false
}

func (c *do) InitHash() eval.OrderedMap {
	return types.SingletonHash2(`name`, types.WrapString(c.name))
}

func (c *do) Call(ctx eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
	if method.Name() != `do` {
		return nil, false
	}
	if block != nil {
		panic(errors.NewArgumentsError(c.name, `nested lambdas are not supported`))
	}

	ok = true
	defer func() {
		if err := recover(); err != nil {
			switch err.(type) {
			case *errors.NextIteration:
				result = err.(*errors.NextIteration).Value()
			case *errors.Return:
				result = err.(*errors.Return).Value()
			default:
				panic(err)
			}
		}
	}()
	result = impl.CallBlock(ctx, c.name, c.parameters, method.Type().(*types.CallableType), c.body, args)
	return
}

type crd struct {
	name   string
	create eval.InvocableValue
	read   eval.InvocableValue
	delete eval.InvocableValue
}

func (c *crd) Name() string {
	return c.name
}

func (c *crd) String() string {
	return eval.ToString(c)
}

func (c *crd) Equals(other interface{}, guard eval.Guard) bool {
	return c == other
}

func (c *crd) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(c, format, bld, g)
}

func (c *crd) PType() eval.Type {
	return crdType
}

func (c *crd) Get(key string) (eval.Value, bool) {
	if key == `name` {
		return types.WrapString(c.name), true
	}
	return nil, false
}

func (c *crd) InitHash() eval.OrderedMap {
	return types.SingletonHash2(`name`, types.WrapString(c.name))
}

func (c *crd) Call(ctx eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
	var f eval.InvocableValue
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
	update eval.InvocableValue
}

func (c *crud) String() string {
	return eval.ToString(c)
}

func (c *crud) PType() eval.Type {
	return crudType
}

func (c *crud) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(c, format, bld, g)
}

func (c *crud) Call(ctx eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
	if method.Name() == `update` {
		return c.update.Call(ctx, block, args...), true
	}
	return c.crd.Call(ctx, method, args, block)
}

func NewDo(name string, parameters []eval.Parameter, block parser.Expression) eval.PuppetObject {
	return &do{name, parameters, block}
}

func NewCRD(name string, create, read, delete eval.InvocableValue) eval.PuppetObject {
	return &crd{name, create, read, delete}
}

func NewCRUD(name string, create, read, update, delete eval.InvocableValue) eval.PuppetObject {
	return &crud{crd{name, create, read, delete}, update}
}
