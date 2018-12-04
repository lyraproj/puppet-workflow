package puppet

import (
	"bytes"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-workflow/puppet/functions"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/servicesdk/grpc"
	"github.com/lyraproj/servicesdk/service"
	"github.com/lyraproj/servicesdk/serviceapi"
	"io/ioutil"
	"unicode"

	// Ensure initialization of needed packages
	_ "github.com/lyraproj/servicesdk/wf"
)

const ManifestLoaderID = `Puppet::ManifestLoader`

type manifestLoader struct {
	ctx eval.Context
	serviceName string
}

type manifestService struct {
	ctx eval.Context
	service serviceapi.Service
}

func (m *manifestService) Invoke(identifier, name string, arguments ...eval.Value) eval.Value {
	return m.service.Invoke(m.ctx.Fork(), identifier, name, arguments...)
}

func (m *manifestService) Metadata() (eval.TypeSet, []serviceapi.Definition) {
	return m.service.Metadata(m.ctx.Fork())
}

func (m *manifestService) State(name string, input eval.OrderedMap) eval.PuppetObject {
	return m.service.State(m.ctx.Fork(), name, input)
}

func WithService(serviceName string, sf func(c eval.Context, s serviceapi.Service)) {
	eval.Puppet.Set(`tasks`, types.Boolean_TRUE)
	eval.Puppet.Set(`workflow`, types.Boolean_TRUE)
	eval.Puppet.Do(func(c eval.Context) {
		sb := service.NewServerBuilder(c, serviceName)
		sb.RegisterApiType(`Puppet::Service`, &manifestService{})
		sb.RegisterAPI(`Puppet::ManifestLoader`, &manifestLoader{c, serviceName})
		s := sb.Server()
		c.Set(`Puppet::ServiceLoader`, s)
		sf(c, s)
	})
}

func Start(serviceName string) {
	WithService(serviceName, func(c eval.Context, s serviceapi.Service) {
		grpc.Serve(c, s)
	})
}

func (m *manifestLoader) LoadManifest(fileName string) serviceapi.Definition {
	c := m.ctx // TODO: Concurrency issue here. Need common loader for all threads, but not common context
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(eval.Error(eval.EVAL_UNABLE_TO_READ_FILE, issue.H{`path`: fileName, `detail`: err.Error()}))
	}
	ast := c.ParseAndValidate(fileName, string(content), false)
	c.AddDefinitions(ast)

	mf := munge(fileName)
	sb := service.NewServerBuilder(c, mf)
	sb.RegisterStateConverter(ResolveState)
	for _, def := range c.ResolveDefinitions() {
		switch def.(type) {
		case PuppetActivity:
			sb.RegisterActivity(def.(PuppetActivity).Activity())
		case eval.Type:
			sb.RegisterType(def.(eval.Type))
		}
	}
	c.Set(functions.ServerBuilderKey, sb)
	_, e := eval.TopEvaluate(c, ast)
	if e != nil {
		panic(e)
	}
	s, _ := m.ctx.Get(`Puppet::ServiceLoader`)
	return s.(*service.Server).AddApi(mf, &manifestService{c, sb.Server()})
}

func munge(path string) string {
	b := bytes.NewBufferString(``)
	pu := true
	ps := true
	for _, c := range path {
		if c == '/' {
			if !ps {
				b.WriteString(`::`)
				ps = true
			}
		} else if c == '_' || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			if ps || pu {
				c = unicode.ToUpper(c)
			}
			b.WriteRune(c)
			ps = false
			pu = false
		} else {
			pu = true
		}
 	}
	if ps {
		b.Truncate(b.Len() - 2)
	}
	return b.String()
}
