package puppetwf_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-evaluator/puppet"
	"github.com/lyraproj/puppet-workflow/puppetwf"
	"github.com/lyraproj/servicesdk/grpc"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/stretchr/testify/assert"
)

func withSampleService(sf func(pdsl.EvaluationContext, serviceapi.Service)) {
	puppet.Do(func(ctx pdsl.EvaluationContext) {
		// Command to start plug-in and read a given manifest
		err := os.Chdir(`testdata`)
		if err != nil {
			panic(err)
		}
		cmd := exec.Command("go", "run", "../../main/main.go", "--debug")

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

/*
func withSampleLocalService(sf func(pdsl.EvaluationContext, serviceapi.Service)) {
	pwf.WithService(`Puppet`, sf)
}
*/

func TestStep(t *testing.T) {
	withSampleService(func(ctx pdsl.EvaluationContext, s serviceapi.Service) {
		s.Metadata(ctx)
		rs := s.Invoke(ctx, puppetwf.ManifestLoaderID, "loadManifest", types.WrapString("testdata"), types.WrapString("aws_example.pp")).(serviceapi.Definition)
		v := s.Invoke(ctx, rs.Identifier().Name(), "metadata")
		assert.Implements(t, (*px.List)(nil), v, "metadata type")
		vl := v.(px.List)
		assert.Equal(t, 2, vl.Len(), `metadata return values size`)
		dv := vl.At(1)
		assert.Implements(t, (*px.List)(nil), v, "metadata definitions type")
		dl := dv.(px.List)
		assert.Equal(t, 1, dl.Len(), `metadata definitions list size`)
		dv = dl.At(0)
		assert.Implements(t, (*serviceapi.Definition)(nil), dv, "metadata definitions type")
		def := dv.(serviceapi.Definition)
		assert.Equal(t, `Service::Definition(
  'identifier' => TypedName(
    'namespace' => 'definition',
    'name' => 'aws_example'
  ),
  'serviceId' => TypedName(
    'namespace' => 'service',
    'name' => 'Aws_examplePp'
  ),
  'properties' => {
    'parameters' => [
      Parameter(
        'name' => 'tags',
        'type' => Hash[String, String],
        'value' => Deferred(
          'name' => 'lookup',
          'arguments' => ['aws.tags']
        )
      )],
    'returns' => [
      Parameter(
        'name' => 'vpcId',
        'type' => String
      ),
      Parameter(
        'name' => 'subnetId',
        'type' => String
      )],
    'steps' => [
      Service::Definition(
        'identifier' => TypedName(
          'namespace' => 'definition',
          'name' => 'aws_example::vpc'
        ),
        'serviceId' => TypedName(
          'namespace' => 'service',
          'name' => 'Aws_examplePp'
        ),
        'properties' => {
          'parameters' => [
            Parameter(
              'name' => 'tags',
              'type' => Any
            )],
          'returns' => [
            Parameter(
              'name' => 'vpcId',
              'type' => Any
            )],
          'resourceType' => Aws::Vpc,
          'style' => 'resource',
          'origin' => ''
        }
      ),
      Service::Definition(
        'identifier' => TypedName(
          'namespace' => 'definition',
          'name' => 'aws_example::subnet'
        ),
        'serviceId' => TypedName(
          'namespace' => 'service',
          'name' => 'Aws_examplePp'
        ),
        'properties' => {
          'parameters' => [
            Parameter(
              'name' => 'tags',
              'type' => Any
            ),
            Parameter(
              'name' => 'vpcId',
              'type' => Any
            )],
          'returns' => [
            Parameter(
              'name' => 'subnetId',
              'type' => Any
            )],
          'resourceType' => Aws::Subnet,
          'style' => 'resource',
          'origin' => ''
        }
      )],
    'style' => 'workflow',
    'origin' => ''
  }
)`, px.ToPrettyString(def))
	})
}
