package puppet_test

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-workflow/puppet"
	"github.com/lyraproj/servicesdk/grpc"
	"github.com/lyraproj/servicesdk/serviceapi"
	"os"
	"os/exec"
	"time"

	//   Ensure Pcore and lookup are initialized
	_ "github.com/lyraproj/puppet-evaluator/pcore"
	_ "github.com/lyraproj/puppet-workflow/puppet/functions"
	_ "github.com/lyraproj/servicesdk/wf"
)

func withSampleService(sf func(eval.Context, serviceapi.Service)) {
	eval.Puppet.Do(func(ctx eval.Context) {
		// Command to start plug-in and read a given manifest
		cmd := exec.Command("go", "run", "../main.go", "--debug")

		// Logger that prints JSON on Stderr
		logger := hclog.New(&hclog.LoggerOptions{
			Level:      hclog.Debug,
			Output:     os.Stderr,
			JSONFormat: false,
			IncludeLocation: false,
		})

		server, err := grpc.Load(cmd, logger)

		// Ensure that plug-ins die when we're done.
		defer	func() {
			wait := make(chan bool)
			go func() {
				plugin.CleanupClients()
				wait <- true
			}()

			select {
			case <- wait:
			case <- time.After(2 * time.Second):
			}
		}()

		if err == nil {
			sf(ctx, server)
		} else {
			fmt.Println(err)
		}
	})
}

func withSampleLocalService(sf func(eval.Context, serviceapi.Service)) {
	puppet.WithService(`Puppet`, sf)
}

func ExampleActivity() {
	withSampleService(func(ctx eval.Context, s serviceapi.Service) {
		s.Metadata(ctx)
		rs := s.Invoke(ctx, puppet.ManifestLoaderID, "load_manifest", types.WrapString("testdata/attach.pp")).(serviceapi.Definition)
		v := s.Invoke(ctx, rs.Identifier().Name(), "metadata").(eval.List)
		dl := v.At(1).(eval.List)
		dl.Each(func(def eval.Value) {
			fmt.Println(eval.ToPrettyString(def))
		})
	})

	// Output:
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Lyra::Aws::InstanceHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
	//   ),
	//   'properties' => {
	//     'interface' => Lyra::Aws::InstanceHandler,
	//     'style' => 'callable',
	//     'handler_for' => Lyra::Aws::Instance
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Lyra::Aws::InternetGatewayHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
	//   ),
	//   'properties' => {
	//     'interface' => Lyra::Aws::InternetGatewayHandler,
	//     'style' => 'callable',
	//     'handler_for' => Lyra::Aws::InternetGateway
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Lyra::Aws::SubnetHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
	//   ),
	//   'properties' => {
	//     'interface' => Lyra::Aws::SubnetHandler,
	//     'style' => 'callable',
	//     'handler_for' => Lyra::Aws::Subnet
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Lyra::Aws::VpcHandler'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
	//   ),
	//   'properties' => {
	//     'interface' => Lyra::Aws::VpcHandler,
	//     'style' => 'callable',
	//     'handler_for' => Lyra::Aws::Vpc
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'attach'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
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
	//           'name' => 'Testdata::AttachPp'
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
	//           'name' => 'Testdata::AttachPp'
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
	//           'name' => 'Testdata::AttachPp'
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
	//               'name' => 'Testdata::AttachPp'
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
	//           'name' => 'attach::internetgateway'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Testdata::AttachPp'
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
	//           'resource_type' => Lyra::Aws::InternetGateway,
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
