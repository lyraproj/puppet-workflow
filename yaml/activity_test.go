package yaml_test

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-workflow/yaml"
	"github.com/lyraproj/servicesdk/service"
	"github.com/lyraproj/servicesdk/serviceapi"
	"io/ioutil"
	"os"
	// Ensure Pcore and lookup are initialized
	_ "github.com/lyraproj/puppet-evaluator/pcore"
	_ "github.com/lyraproj/servicesdk/wf"
)

func ExampleNestedObject() {
	eval.Puppet.Do(func(ctx eval.Context) {
		typesFile := "testdata/tf-k8s.pp"
		content, err := ioutil.ReadFile(typesFile)
		if err != nil {
			panic(err.Error())
		}
		ast := ctx.ParseAndValidate(typesFile, string(content), false)
		ctx.AddDefinitions(ast)
		_, err = eval.TopEvaluate(ctx, ast)
		if err != nil {
			panic(err.Error())
		}

		workflowFile := "testdata/tf-k8s-sample.yaml"
		content, err = ioutil.ReadFile(workflowFile)
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
		rs := ac.(eval.List).At(0).(serviceapi.Definition)

		st := sv.State(ctx, rs.Identifier().Name(), eval.EMPTY_MAP)
		st.ToString(os.Stdout, eval.PRETTY, nil)
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
	eval.Puppet.Do(func(ctx eval.Context) {
		typesFile := "testdata/types.pp"
		content, err := ioutil.ReadFile(typesFile)
		if err != nil {
			panic(err.Error())
		}
		ast := ctx.ParseAndValidate(typesFile, string(content), false)
		ctx.AddDefinitions(ast)
		_, err = eval.TopEvaluate(ctx, ast)
		if err != nil {
			panic(err.Error())
		}

		workflowFile := "testdata/attach.yaml"
		content, err = ioutil.ReadFile(workflowFile)
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
		wf.ToString(os.Stdout, eval.PRETTY, nil)
		fmt.Println()

		st := sv.State(ctx, `attach::vpc`, eval.Wrap(ctx, map[string]interface{}{
			`region`: `us-west`,
			`tags`:   map[string]string{`a`: `av`, `b`: `bv`}}).(eval.OrderedMap))
		st.ToString(os.Stdout, eval.PRETTY, nil)
		fmt.Println()
	})

	// Output:
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'attach'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Yaml::Test'
	//   ),
	//   'properties' => {
	//     'input' => [
	//       Parameter(
	//         'name' => 'region',
	//         'type' => String,
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.region']
	//         )
	//       ),
	//       Parameter(
	//         'name' => 'tags',
	//         'type' => Hash[String, String],
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.tags']
	//         )
	//       ),
	//       Parameter(
	//         'name' => 'keyName',
	//         'type' => String,
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.keyname']
	//         )
	//       ),
	//       Parameter(
	//         'name' => 'ec2Cnt',
	//         'type' => Integer,
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.instance.count']
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
	//       ),
	//       Parameter(
	//         'name' => 'internetGatewayId',
	//         'type' => String
	//       ),
	//       Parameter(
	//         'name' => 'nodes',
	//         'type' => Hash[String, Struct[{'publicIp' => String, 'privateIp' => String}]]
	//       )],
	//     'activities' => [
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::vpc'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Yaml::Test'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'region',
	//               'type' => String
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'vpcId',
	//               'type' => Optional[String]
	//             )],
	//           'resourceType' => Lyra::Aws::Vpc,
	//           'style' => 'resource'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::subnet'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Yaml::Test'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'region',
	//               'type' => String
	//             ),
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
	//           'resourceType' => Lyra::Aws::Subnet,
	//           'style' => 'resource'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::nodes'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Yaml::Test'
	//         ),
	//         'properties' => {
	//           'iterationStyle' => 'times',
	//           'over' => [
	//             Parameter(
	//               'name' => 'ec2Cnt',
	//               'type' => Any
	//             )],
	//           'variables' => [
	//             Parameter(
	//               'name' => 'i',
	//               'type' => Any
	//             )],
	//           'producer' => Service::Definition(
	//             'identifier' => TypedName(
	//               'namespace' => 'definition',
	//               'name' => 'attach::instance'
	//             ),
	//             'serviceId' => TypedName(
	//               'namespace' => 'service',
	//               'name' => 'Yaml::Test'
	//             ),
	//             'properties' => {
	//               'input' => [
	//                 Parameter(
	//                   'name' => 'region',
	//                   'type' => String
	//                 ),
	//                 Parameter(
	//                   'name' => 'i',
	//                   'type' => Optional[String]
	//                 ),
	//                 Parameter(
	//                   'name' => 'keyName',
	//                   'type' => String
	//                 ),
	//                 Parameter(
	//                   'name' => 'tags',
	//                   'type' => Hash[String, String]
	//                 )],
	//               'output' => [
	//                 Parameter(
	//                   'name' => 'key',
	//                   'type' => Optional[String],
	//                   'value' => 'instanceId'
	//                 ),
	//                 Parameter(
	//                   'name' => 'value',
	//                   'type' => Tuple[Optional[String], Optional[String]],
	//                   'value' => ['publicIp', 'privateIp']
	//                 )],
	//               'resourceType' => Lyra::Aws::Instance,
	//               'style' => 'resource'
	//             }
	//           ),
	//           'style' => 'iterator'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::gw'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Yaml::Test'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'region',
	//               'type' => String
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'internetGatewayId',
	//               'type' => Optional[String]
	//             )],
	//           'resourceType' => Lyra::Aws::InternetGateway,
	//           'style' => 'resource'
	//         }
	//       )],
	//     'style' => 'workflow'
	//   }
	// )
	// Lyra::Aws::Vpc(
	//   'ensure' => 'present',
	//   'region' => 'us-west',
	//   'tags' => {
	//     'a' => 'av',
	//     'b' => 'bv'
	//   },
	//   'cidrBlock' => '192.168.0.0/16',
	//   'enableDnsHostnames' => true,
	//   'enableDnsSupport' => true
	// )
}