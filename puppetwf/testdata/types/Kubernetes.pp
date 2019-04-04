
# this file is prefixed "aaa" so that it is processed first as it contains types that are needed by other workflows
# this file is generated
type Kubernetes = TypeSet[{
  pcore_uri => 'http://puppet.com/2016.1/pcore',
  pcore_version => '1.0.0',
  name_authority => 'http://puppet.com/2016.1/runtime',
  name => 'Kubernetes',
  version => '0.1.0',
  types => {
    Namespace => {
      attributes => {
        'namespace_id' => Optional[String],
        'metadata'     => Object[{
          attributes => {
            'name' => Optional[String],
            'resource_version' => Optional[String],
            'self_link' => Optional[String]
          }
        }]
      }
    }
  }
}]
