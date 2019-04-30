workflow aws_example {
  typespace => 'Aws',
  parameters => (
    Hash[String,String] $tags = lookup('aws.tags'),
  ),
  returns => (
    String $vpcId,
    String $subnetId,
  )
} {
  resource vpc {
    parameters  => ($tags),
    returns => ($vpcId)
  }{
    amazonProvidedIpv6CidrBlock => false,
    cidrBlock => '192.168.0.0/16',
    enableDnsHostnames => false,
    enableDnsSupport => false,
    isDefault => false,
    state => 'available',
    tags => $tags,
  }
  resource subnet {
    parameters  => ($tags, $vpcId),
    returns => ($subnetId)
  }{
    vpcId => $vpcId,
    cidrBlock => '192.168.1.0/24',
    ipv6CidrBlock => '',
    tags => $tags,
    assignIpv6AddressOnCreation => false,
    mapPublicIpOnLaunch => false,
    defaultForAz => false,
    state => 'available',
  }
}
