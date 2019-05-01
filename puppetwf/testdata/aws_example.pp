workflow aws_example {
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
    returns => ($vpcId),
    type => Aws::Vpc
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
    returns => ($subnetId),
    type => Aws::Subnet
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
