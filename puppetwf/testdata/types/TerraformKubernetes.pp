
# this file is prefixed "aaa" so that it is processed first as it contains types that are needed by other workflows
# this file is generated
type TerraformKubernetes = TypeSet[{
  pcore_uri => 'http://puppet.com/2016.1/pcore',
  pcore_version => '1.0.0',
  name_authority => 'http://puppet.com/2016.1/runtime',
  name => 'TerraformKubernetes',
  version => '0.1.0',
  types => {
    Kubernetes_namespace => {
      attributes => {
        'kubernetes_namespace_id' => {
          'type' => Optional[String],
          'value' => undef
        },
        'metadata' => Kubernetes_namespace_metadata_721
      }
    },
    Kubernetes_namespace_metadata_721 => {
      attributes => {
        'kubernetes_namespace_metadata_721_id' => {
          'type' => Optional[String],
          'value' => undef
        },
        'name' => {
          'type' => Optional[String],
          'value' => undef
        },
        'resource_version' => {
          'type' => Optional[String],
          'value' => undef
        },
        'self_link' => {
          'type' => Optional[String],
          'value' => undef
        }
      }
    },
  }
}]