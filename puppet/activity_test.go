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
		rs := s.Invoke(ctx, puppet.ManifestLoaderID, "loadManifest", types.WrapString("testdata"), types.WrapString("testdata/aws_example.pp")).(serviceapi.Definition)
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
	//     'name' => 'aws_example'
	//   ),
	//   'serviceId' => TypedName(
	//     'namespace' => 'service',
	//     'name' => 'Testdata::Aws_examplePp'
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
	//           'name' => 'aws_example::vpc'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Testdata::Aws_examplePp'
	//         ),
	//         'properties' => {
	//           'input' => [
	//             Parameter(
	//               'name' => 'tags',
	//               'type' => Any
	//             )],
	//           'output' => [
	//             Parameter(
	//               'name' => 'vpcId',
	//               'type' => Any
	//             )],
	//           'resourceType' => Aws::Vpc,
	//           'style' => 'resource'
	//         }
	//       ),
	//       Service::Definition(
	//         'identifier' => TypedName(
	//           'namespace' => 'definition',
	//           'name' => 'aws_example::subnet'
	//         ),
	//         'serviceId' => TypedName(
	//           'namespace' => 'service',
	//           'name' => 'Testdata::Aws_examplePp'
	//         ),
	//         'properties' => {
	//           'input' => [
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
	//           'resourceType' => Aws::Subnet,
	//           'style' => 'resource'
	//         }
	//       )],
	//     'style' => 'workflow'
	//   }
	// )
}
