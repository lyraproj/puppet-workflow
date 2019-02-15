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
			Level:           hclog.Debug,
			Output:          os.Stderr,
			JSONFormat:      false,
			IncludeLocation: false,
		})

		server, err := grpc.Load(cmd, logger)

		// Ensure that plug-ins die when we're done.
		defer func() {
			wait := make(chan bool)
			go func() {
				plugin.CleanupClients()
				wait <- true
			}()

			select {
			case <-wait:
			case <-time.After(2 * time.Second):
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
	withSampleLocalService(func(ctx eval.Context, s serviceapi.Service) {
		s.Metadata(ctx)
		rs := s.Invoke(ctx, puppet.ManifestLoaderID, "loadManifest", types.WrapString("testdata/attach.pp")).(serviceapi.Definition)
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
	//     'name' => 'Attach::Notice'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
	//   ),
	//   'properties' => {
	//     'interface' => Puppet::Do,
	//     'style' => 'callable'
	//   }
	// )
	// Service::Definition(
	//   'identifier' => TypedName(
	//     'namespace' => 'definition',
	//     'name' => 'Attach::Notice2'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::AttachPp'
	//   ),
	//   'properties' => {
	//     'interface' => Puppet::Do,
	//     'style' => 'callable'
	//   }
	// )
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
	//     'handlerFor' => Lyra::Aws::Instance
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
	//     'handlerFor' => Lyra::Aws::InternetGateway
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
	//     'handlerFor' => Lyra::Aws::Subnet
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
	//     'handlerFor' => Lyra::Aws::Vpc
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
	//       ),
	//       Parameter(
	//         'name' => 'notice',
	//         'type' => String
	//       ),
	//       Parameter(
	//         'name' => 'notice2',
	//         'type' => String
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
	//               'name' => 'vpcId',
	//               'type' => Any
	//             )],
	//           'resourceType' => Lyra::Aws::Vpc,
	//           'style' => 'resource'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::notice'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Testdata::AttachPp'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'vpcId',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'notice',
	//               'type' => String
	//             )],
	//           'interface' => Puppet::Do,
	//           'style' => 'stateless'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'attach::notice2'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Testdata::AttachPp'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'subnetId',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'notice2',
	//               'type' => String
	//             )],
	//           'interface' => Puppet::Do,
	//           'style' => 'stateless'
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
	//               'name' => 'vpcId',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'subnetId',
	//               'type' => Any
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
	//           'name' => 'Testdata::AttachPp'
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
	//                   'name' => 'keyName',
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
	//                   'value' => 'instanceId'
	//                 ),
	//                 Parameter(
	//                   'name' => 'value',
	//                   'type' => Any,
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
	//               'name' => 'internetGatewayId',
	//               'type' => Any
	//             )],
	//           'resourceType' => Lyra::Aws::InternetGateway,
	//           'style' => 'resource'
	//         }
	//       )],
	//     'style' => 'workflow'
	//   }
	// )
}
