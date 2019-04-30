package yaml_test

import (
	"fmt"
	"io/ioutil"
	"os"

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
	//       Parameter(
	//         'name' => 'tags',
	//         'type' => Hash[String, String],
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.tags']
	//         )
	//       )],
	//     'returns' => [
	//       Parameter(
	//         'name' => 'vpcId',
	//         'type' => String
	//       ),
	//       Parameter(
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
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'returns' => [
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
	//           'parameters' => [
	//             Parameter(
	//               'name' => 'vpcId',
	//               'type' => String
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'returns' => [
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
