package yaml_test

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-workflow/yaml"
	"github.com/lyraproj/servicesdk/service"
	"io/ioutil"
	"os"

	// Ensure Pcore and lookup are initialized
	_ "github.com/lyraproj/puppet-evaluator/pcore"
	_ "github.com/lyraproj/servicesdk/wf"
)

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
	//         'name' => 'key_name',
	//         'type' => String,
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.keyname']
	//         )
	//       ),
	//       Parameter(
	//         'name' => 'ec2_cnt',
	//         'type' => Integer,
	//         'value' => Deferred(
	//           'name' => 'lookup',
	//           'arguments' => ['aws.instance.count']
	//         )
	//       )],
	//     'output' => [
	//       Parameter(
	//         'name' => 'vpc_id',
	//         'type' => String
	//       ),
	//       Parameter(
	//         'name' => 'subnet_id',
	//         'type' => String
	//       ),
	//       Parameter(
	//         'name' => 'internet_gateway_id',
	//         'type' => String
	//       ),
	//       Parameter(
	//         'name' => 'nodes',
	//         'type' => Hash[String, Struct[{'public_ip' => String, 'private_ip' => String}]]
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
	//               'name' => 'vpc_id',
	//               'type' => Optional[String]
	//             )],
	//           'resource_type' => Lyra::Aws::Vpc,
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
	//               'name' => 'vpc_id',
	//               'type' => String
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Hash[String, String]
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'subnet_id',
	//               'type' => Optional[String]
	//             )],
	//           'resource_type' => Lyra::Aws::Subnet,
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
	//           'iteration_style' => 'times',
	//           'over' => [
	//             Parameter(
	//               'name' => 'ec2_cnt',
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
	//                   'name' => 'key_name',
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
	//                   'value' => 'instance_id'
	//                 ),
	//                 Parameter(
	//                   'name' => 'value',
	//                   'type' => Tuple[Optional[String], Optional[String]],
	//                   'value' => ['public_ip', 'private_ip']
	//                 )],
	//               'resource_type' => Lyra::Aws::Instance,
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
	//               'name' => 'internet_gateway_id',
	//               'type' => Optional[String]
	//             )],
	//           'resource_type' => Lyra::Aws::InternetGateway,
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
	//   'cidr_block' => '192.168.0.0/16',
	//   'enable_dns_hostnames' => true,
	//   'enable_dns_support' => true
	// )
}
