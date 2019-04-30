package puppetwf

import (
	"bytes"
	"io/ioutil"
	"strings"
	"unicode"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/evaluator"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-evaluator/puppet"
	"github.com/lyraproj/puppet-workflow/yaml"
	"github.com/lyraproj/servicesdk/grpc"
	"github.com/lyraproj/servicesdk/service"
	"github.com/lyraproj/servicesdk/serviceapi"
)

const ManifestLoaderID = `Puppet::ManifestLoader`

type manifestLoader struct {
	ctx         pdsl.EvaluationContext
	serviceName string
}

type manifestService struct {
	ctx     pdsl.EvaluationContext
	service serviceapi.Service
}

func (m *manifestService) Invoke(identifier, name string, arguments ...px.Value) px.Value {
	return m.service.Invoke(m.ctx.Fork(), identifier, name, arguments...)
}

func (m *manifestService) Metadata() (px.TypeSet, []serviceapi.Definition) {
	return m.service.Metadata(m.ctx.Fork())
}

func (m *manifestService) State(name string, parameters px.OrderedMap) px.PuppetObject {
	return m.service.State(m.ctx.Fork(), name, parameters)
}

func WithService(serviceName string, sf func(c pdsl.EvaluationContext, s serviceapi.Service)) {
	pcore.Set(`tasks`, types.BooleanTrue)
	pcore.Set(`workflow`, types.BooleanTrue)
	puppet.Do(func(c pdsl.EvaluationContext) {
		sb := service.NewServiceBuilder(c, serviceName)
		sb.RegisterApiType(`Puppet::Service`, &manifestService{})
		sb.RegisterAPI(`Puppet::ManifestLoader`, &manifestLoader{c, serviceName})
		s := sb.Server()
		c.Set(`Puppet::ServiceLoader`, s)
		sf(c, s)
	})
}

func Start(serviceName string) {
	WithService(serviceName, func(c pdsl.EvaluationContext, s serviceapi.Service) {
		grpc.Serve(c, s)
	})
}

func (m *manifestLoader) LoadManifest(moduleDir string, fileName string) serviceapi.Definition {
	ec := evaluator.WithParent(m.ctx, evaluator.NewEvaluator)
	ec.SetLoader(px.NewFileBasedLoader(ec.Loader(), moduleDir, ``, px.PuppetDataTypePath))
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(px.Error(px.UnableToReadFile, issue.H{`path`: fileName, `detail`: err.Error()}))
	}

	mf := munged(fileName)
	sb := service.NewServiceBuilder(ec, mf)
	ec.Set(ServerBuilderKey, sb)

	if strings.HasSuffix(fileName, `.yaml`) {
		// Assume YAML content instead of Puppet DSL
		sb.RegisterStateConverter(yaml.ResolveState)
		sb.RegisterStep(yaml.CreateStep(ec, fileName, content))
	} else {
		ast := ec.ParseAndValidate(fileName, string(content), false)
		ec.AddDefinitions(ast)

		sb.RegisterStateConverter(ResolveState)
		for _, def := range ec.ResolveDefinitions() {
			switch def := def.(type) {
			case PuppetStep:
				sb.RegisterStep(def.Step())
			case px.Type:
				sb.RegisterType(def)
			}
		}
		pdsl.TopEvaluate(ec, ast)
	}
	s, _ := m.ctx.Get(`Puppet::ServiceLoader`)
	return s.(*service.Server).AddApi(mf, &manifestService{ec, sb.Server()})
}

func munged(path string) string {
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
				// First character of the name must be an upper case letter
				if ps && (c == '_' || c >= '0' && c <= '9') {
					// Must insert extra character
					b.WriteRune('X')
				} else {
					c = unicode.ToUpper(c)
				}
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
