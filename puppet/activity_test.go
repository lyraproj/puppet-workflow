package puppet_test

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-puppet-dsl-workflow/puppet"
	"github.com/puppetlabs/go-servicesdk/grpc"
	"github.com/puppetlabs/go-servicesdk/serviceapi"
	"os"
	"os/exec"

	//   Ensure Pcore and lookup are initialized
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-puppet-dsl-workflow/puppet/functions"
	_ "github.com/puppetlabs/go-servicesdk/wf"
)

func withSampleService(sf func(eval.Context, serviceapi.Service)) {
	eval.Puppet.Set(`tasks`, types.Boolean_TRUE)
	eval.Puppet.Set(`workflow`, types.Boolean_TRUE)
	eval.Puppet.Do(func(ctx eval.Context) {
		// Command to start plug-in and read a given manifest
		cmd := exec.Command("go", "run", "../main/main.go", `testdata/attach.pp`)

		// Logger that prints JSON on Stderr
		logger := hclog.New(&hclog.LoggerOptions{
			Level:      hclog.Debug,
			Output:     os.Stderr,
			JSONFormat: true,
		})

		server, err := grpc.Load(cmd, logger)
		defer func() {
			// Ensure that plug-ins die when we're done.
			plugin.CleanupClients()
		}()

		if err == nil {
			sf(ctx, server)
		} else {
			fmt.Println(err)
		}
	})
}

func withSampleLocalService(sf func(eval.Context, serviceapi.Service)) {
	eval.Puppet.Set(`tasks`, types.Boolean_TRUE)
	eval.Puppet.Set(`workflow`, types.Boolean_TRUE)
	eval.Puppet.Do(func(ctx eval.Context) {
		workflowName := `attach`
		path := `testdata/` + workflowName + `.pp`
		sf(ctx, puppet.CreateService(ctx, `Puppet`, path))
	})
}

func ExampleActivity() {
	withSampleLocalService(func(ctx eval.Context, s serviceapi.Service) {
		_, defs := s.Metadata(ctx)
		for _, def := range defs {
			fmt.Println(eval.ToPrettyString(def))
		}
	})

	// Output:
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Genesis::Aws::InstanceHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Puppet'
	//   ),
	//   'properties' => {
	//     'interface' => Genesis::Aws::InstanceHandler,
	//     'style' => 'callable',
	//     'handler_for' => Genesis::Aws::Instance
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Genesis::Aws::InternetGatewayHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Puppet'
	//   ),
	//   'properties' => {
	//     'interface' => Genesis::Aws::InternetGatewayHandler,
	//     'style' => 'callable',
	//     'handler_for' => Genesis::Aws::InternetGateway
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Genesis::Aws::SubnetHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Puppet'
	//   ),
	//   'properties' => {
	//     'interface' => Genesis::Aws::SubnetHandler,
	//     'style' => 'callable',
	//     'handler_for' => Genesis::Aws::Subnet
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Genesis::Aws::VpcHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Puppet'
	//   ),
	//   'properties' => {
	//     'interface' => Genesis::Aws::VpcHandler,
	//     'style' => 'callable',
	//     'handler_for' => Genesis::Aws::Vpc
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'attach'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Puppet'
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
	//         'type' => Hash[String, Struct
	//           [{'public_ip' => String, 'private_ip' => String}]]
	//       )],
	//     'activities' => [
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::vpc'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Puppet'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'region',
	//               'type' => Any
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'vpc_id',
	//               'type' => Any
	//             )],
	//           'resource_type' => Genesis::Aws::Vpc,
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
	//           'name' => 'Puppet'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'region',
	//               'type' => Any
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Any
	//             ),
	//             Parameter(
	//               'name' => 'vpc_id',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'subnet_id',
	//               'type' => Any
	//             )],
	//           'resource_type' => Genesis::Aws::Subnet,
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
	//           'name' => 'Puppet'
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
	//               'name' => 'n',
	//               'type' => Any
	//             )],
	//           'producer' => Service::Definition(
	//             'identifier' => TypedName(
	//               'namespace' => 'definition',
	//               'name' => 'attach::instance'
	//             ),
	//             'serviceId' => TypedName(
	//               'namespace' => 'service',
	//               'name' => 'Puppet'
	//             ),
	//             'properties' => {
	//               'input' => [
	//                 Parameter(
	//                   'name' => 'n',
	//                   'type' => Any
	//                 ),
	//                 Parameter(
	//                   'name' => 'region',
	//                   'type' => Any
	//                 ),
	//                 Parameter(
	//                   'name' => 'key_name',
	//                   'type' => Any
	//                 ),
	//                 Parameter(
	//                   'name' => 'tags',
	//                   'type' => Any
	//                 )],
	//               'output' => [
	//                 Parameter(
	//                   'name' => 'key',
	//                   'type' => Any,
	//                   'value' => 'instance_id'
	//                 ),
	//                 Parameter(
	//                   'name' => 'value',
	//                   'type' => Any,
	//                   'value' => ['public_ip', 'private_ip']
	//                 )],
	//               'resource_type' => Genesis::Aws::Instance,
	//               'style' => 'resource'
	//             }
	//           ),
	//           'style' => 'iterator'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::internetgateway'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Puppet'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'region',
	//               'type' => Any
	//             ),
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'internet_gateway_id',
	//               'type' => Any
	//             )],
	//           'resource_type' => Genesis::Aws::InternetGateway,
	//           'style' => 'resource'
	//         }
	//       )],
	//     'style' => 'workflow'
	//   }
	// )
	//
}

/*
type allExists struct{}

func (allExists) Exists(identity string) bool {
	return true
}

func ExampleDelete() {
	eval.Puppet.Set(`tasks`, types.Boolean_TRUE)
	eval.Puppet.Set(`workflow`, types.Boolean_TRUE)
	err := lookup.DoWithParent(context.Background(), provider, func(ctx lookup.Context) error {
		wf, err := sampleWorkflow(ctx)
		if err != nil {
			return err
		}
		we := wfe.NewWorkflowEngine(wf)
		we.BuildInvertedGraph(&allExists{})
		//   return ioutil.WriteFile(os.Getenv("HOME") + "/tmp/wf.dot", we.GraphAsDot(), 0644)
		fmt.Println(string(we.GraphAsDot()))
		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}

	//   Output:
	//   strict digraph attach {
	//     //   Node definitions.
	//     vpc [label="vpc{
	//   input:[region,tags],
	//   output:[vpc_id]}"];
	//     subnet [label="subnet{
	//   input:[region,tags,vpc_id],
	//   output:[subnet_id]}"];
	//     instance [label="instance{
	//   input:[ec2_cnt,region,key_name,tags],
	//   output:[nodes]}"];
	//     internetgateway [label="internetgateway{
	//   input:[region,tags],
	//   output:[internet_gateway_id]}"];
	//
	//     //   Edge definitions.
	//     subnet -> vpc;
	//   }
}
*/
