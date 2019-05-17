package yaml_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-evaluator/puppet"
	"github.com/lyraproj/puppet-workflow/yaml"
	"github.com/lyraproj/servicesdk/service"
	"github.com/lyraproj/servicesdk/serviceapi"
)

func ExampleCreateStep_nestedObject() {
	puppet.Do(func(ctx pdsl.EvaluationContext) {
		ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
		workflowFile := "testdata/tf-k8s-sample.yaml"
		content, err := ioutil.ReadFile(workflowFile)
		if err != nil {
			panic(err.Error())
		}
		a := yaml.CreateStep(ctx, workflowFile, content)

		sb := service.NewServiceBuilder(ctx, `Yaml::Test`)
		sb.RegisterStateConverter(yaml.ResolveState)
		sb.RegisterStep(a)
		sv := sb.Server()
		_, defs := sv.Metadata(ctx)

		wf := defs[0]
		ac, _ := wf.Properties().Get4(`steps`)
		rs := ac.(px.List).At(0).(serviceapi.Definition)

		st := sv.State(ctx, rs.Identifier().Name(), px.EmptyMap)
		st.ToString(os.Stdout, px.Pretty, nil)
		fmt.Println()
	})

	// Output:
	// Kubernetes::Namespace(
	//   'metadata' => {
	//     'name' => 'terraform-lyra',
	//     'resource_version' => 'hi',
	//     'self_link' => 'me'
	//   },
	//   'namespace_id' => 'ignore'
	// )
}

func ExampleCreateStep() {
	pcore.Do(func(ctx px.Context) {
		ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
		workflowFile := "testdata/aws_vpc.yaml"
		content, err := ioutil.ReadFile(workflowFile)
		if err != nil {
			panic(err.Error())
		}
		a := yaml.CreateStep(ctx, workflowFile, content)

		sb := service.NewServiceBuilder(ctx, `Yaml::Test`)
		sb.RegisterStateConverter(yaml.ResolveState)
		sb.RegisterStep(a)
		sv := sb.Server()
		_, defs := sv.Metadata(ctx)

		wf := defs[0]
		wf.ToString(os.Stdout, px.Pretty, nil)
		fmt.Println()

		st := sv.State(ctx, `aws_vpc::vpc`, px.Wrap(ctx, map[string]interface{}{
			`tags`: map[string]string{`a`: `av`, `b`: `bv`}}).(px.OrderedMap))
		st.ToString(os.Stdout, px.Pretty, nil)
		fmt.Println()
	})

	// Output:
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'aws_vpc'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Yaml::Test'
	//   ),
	//   'properties' => {
	//     'parameters' => [
	//       Lyra::Parameter(
	//         'name' => 'tags',
	//         'type' => Hash[String, String],
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.tags']
	//         )
	//       )],
	//     'returns' => [
	//       Lyra::Parameter(
	//         'name' => 'vpcId',
	//         'type' => String
	//       ),
	//       Lyra::Parameter(
	//         'name' => 'subnetId',
	//         'type' => String
	//       )],
	//     'steps' => [
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'aws_vpc::vpc'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Yaml::Test'
	//         ),
	//         'properties' => {
	//           'parameters' => [
	//             Lyra::Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'returns' => [
	//             Lyra::Parameter(
	//               'name' => 'vpcId',
	//               'type' => Optional[String]
	//             )],
	//           'resourceType' => Aws::Vpc,
	//           'style' => 'resource',
	//           'origin' => '(file: testdata/aws_vpc.yaml)'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'aws_vpc::subnet'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Yaml::Test'
	//         ),
	//         'properties' => {
	//           'parameters' => [
	//             Lyra::Parameter(
	//               'name' => 'vpcId',
	//               'type' => String
	//             ),
	//             Lyra::Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'returns' => [
	//             Lyra::Parameter(
	//               'name' => 'subnetId',
	//               'type' => Optional[String]
	//             )],
	//           'resourceType' => Aws::Subnet,
	//           'style' => 'resource',
	//           'origin' => '(file: testdata/aws_vpc.yaml)'
	//         }
	//       )],
	//     'style' => 'workflow',
	//     'origin' => '(file: testdata/aws_vpc.yaml)'
	//   }
	// )
	// Aws::Vpc(
	//   'amazonProvidedIpv6CidrBlock' => false,
	//   'cidrBlock' => '192.168.0.0/16',
	//   'enableDnsHostnames' => false,
	//   'enableDnsSupport' => false,
	//   'tags' => {
	//     'a' => 'av',
	//     'b' => 'bv'
	//   },
	//   'isDefault' => false,
	//   'state' => 'available'
	// )
}

func TestParse_unresolvedType(t *testing.T) {
	requireError(t, `Reference to unresolved type 'No::Such::Type' (file: testdata/typefail.yaml, line: 3, column: 5)`, func() {
		pcore.Do(func(ctx px.Context) {
			ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
			workflowFile := "testdata/typefail.yaml"
			content, err := ioutil.ReadFile(workflowFile)
			if err != nil {
				panic(err.Error())
			}
			yaml.CreateStep(ctx, workflowFile, content)
		})
	})
}

func TestParse_unparsableType(t *testing.T) {
	requireError(t, `expected one of ',' or '}', got '' (file: testdata/typeparsefail.yaml, line: 6, column: 11)`, func() {
		pcore.Do(func(ctx px.Context) {
			ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
			workflowFile := "testdata/typeparsefail.yaml"
			content, err := ioutil.ReadFile(workflowFile)
			if err != nil {
				panic(err.Error())
			}
			yaml.CreateStep(ctx, workflowFile, content)
		})
	})
}

func TestParse_mismatchedType(t *testing.T) {
	requireError(t,
		regexp.MustCompile(`(?m:/typemismatchfail.yaml, line: 11, column: 7\)\s*Caused by: invalid arguments for function Integer)`),
		func() {
			pcore.Do(func(ctx px.Context) {
				ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
				workflowFile := "testdata/typemismatchfail.yaml"
				content, err := ioutil.ReadFile(workflowFile)
				if err != nil {
					panic(err.Error())
				}
				yaml.CreateStep(ctx, workflowFile, content)
			})
		})
}

func TestParse_unresolvedAttr(t *testing.T) {
	requireError(t, `A Kubernetes::Namespace has no attribute named no_such_attribute (file: testdata/attrfail.yaml, line: 3, column: 14)`, func() {
		pcore.Do(func(ctx px.Context) {
			ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
			workflowFile := "testdata/attrfail.yaml"
			content, err := ioutil.ReadFile(workflowFile)
			if err != nil {
				panic(err.Error())
			}
			yaml.CreateStep(ctx, workflowFile, content)
		})
	})
}

func requireError(t *testing.T, msg interface{}, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				if s, ok := msg.(string); ok {
					require.Equal(t, s, err.Error())
				} else {
					require.True(t, msg.(*regexp.Regexp).FindString(err.Error()) != ``)
				}
			} else {
				panic(r)
			}
		}
	}()
	f()
	require.Fail(t, `expected panic didn't happen`)
}
