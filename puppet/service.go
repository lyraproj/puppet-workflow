package puppet

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-servicesdk/service"
	"github.com/puppetlabs/go-servicesdk/serviceapi"
	"io/ioutil"
)

const ServerBuilderKey = `WF::ServerBuilder`

func CreateService(c eval.Context, serviceName, fileName string) serviceapi.Service {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(eval.Error(eval.EVAL_UNABLE_TO_READ_FILE, issue.H{`path`: fileName, `detail`: err.Error()}))
	}
	ast := c.ParseAndValidate(fileName, string(content), false)
	c.AddDefinitions(ast)

	sb := service.NewServerBuilder(c, serviceName)
	sb.RegisterStateConverter(ResolveState)
	for _, def := range c.ResolveDefinitions() {
		switch def.(type) {
		case PuppetActivity:
			sb.RegisterActivity(def.(PuppetActivity).Activity())
		case eval.Type:
			sb.RegisterType(def.(eval.Type))
		}
	}
	c.Set(ServerBuilderKey, sb)
	_, e := c.Evaluator().Evaluate(c, ast)
	if e != nil {
		panic(e)
	}

	return sb.Server()
}
