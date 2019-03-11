package yaml_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/lyraproj/pcore/pcore"

	"github.com/lyraproj/puppet-evaluator/pdsl"

	"github.com/lyraproj/puppet-evaluator/puppet"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-workflow/yaml"
	"github.com/lyraproj/servicesdk/service"
	"github.com/lyraproj/servicesdk/serviceapi"

	// Ensure servicesdk initializaion
	_ "github.com/lyraproj/servicesdk/wf"
)

func ExampleNestedObject() {
	puppet.Do(func(ctx pdsl.EvaluationContext) {
		ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
		workflowFile := "testdata/tf-k8s-sample.yaml"
		content, err := ioutil.ReadFile(workflowFile)
		if err != nil {
			panic(err.Error())
		}
		a := yaml.CreateActivity(ctx, workflowFile, content)

		sb := service.NewServerBuilder(ctx, `Yaml::Test`)
		sb.RegisterStateConverter(yaml.ResolveState)
		sb.RegisterActivity(a)
		sv := sb.Server()
		_, defs := sv.Metadata(ctx)

		wf := defs[0]
		ac, _ := wf.Properties().Get4(`activities`)
		rs := ac.(px.List).At(0).(serviceapi.Definition)

		st := sv.State(ctx, rs.Identifier().Name(), px.EmptyMap)
		st.ToString(os.Stdout, px.Pretty, nil)
		fmt.Println()
	})

	// Output:
	// TerraformKubernetes::Kubernetes_namespace(
	//   'metadata' => TerraformKubernetes::Kubernetes_namespace_metadata_721(
	//     'name' => 'terraform-lyra',
	//     'resource_version' => 'hi',
	//     'self_link' => 'me'
	//   ),
	//   'kubernetes_namespace_id' => 'ignore'
	// )
}

func ExampleActivity() {
	pcore.Do(func(ctx px.Context) {
		ctx.SetLoader(px.NewFileBasedLoader(ctx.Loader(), "../puppetwf/testdata", ``, px.PuppetDataTypePath))
		workflowFile := "testdata/aws_vpc.yaml"
		content, err := ioutil.ReadFile(workflowFile)
		if err != nil {
			panic(err.Error())
		}
		a := yaml.CreateActivity(ctx, workflowFile, content)

		sb := service.NewServerBuilder(ctx, `Yaml::Test`)
		sb.RegisterStateConverter(yaml.ResolveState)
		sb.RegisterActivity(a)
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
	//     'input' => [
	//       Parameter(
	//         'name' => 'tags',
	//         'type' => Hash[String, String],
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.tags']
	//         )
	//       )],
	//     'output' => [
	//       Parameter(
	//         'name' => 'vpcId',
	//         'type' => String
	//       ),
	//       Parameter(
	//         'name' => 'subnetId',
	//         'type' => String
	//       )],
	//     'activities' => [
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
	//           'input' => [
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'vpcId',
	//               'type' => Optional[String]
	//             )],
	//           'resourceType' => Aws::Vpc,
	//           'style' => 'resource'
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
	//           'input' => [
	//             Parameter(
	//               'name' => 'vpcId',
	//               'type' => String
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'subnetId',
	//               'type' => Optional[String]
	//             )],
	//           'resourceType' => Aws::Subnet,
	//           'style' => 'resource'
	//         }
	//       )],
	//     'style' => 'workflow'
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
