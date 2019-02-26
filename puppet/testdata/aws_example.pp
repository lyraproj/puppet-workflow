workflow aws_example {
  typespace => 'aws',
  input => (
    Hash[String,String] $tags = lookup('aws.tags'),
  ),
  output => (
    String $vpcId,
    String $subnetId,
  )
} {
  resource vpc {
    input  => ($tags),
    output => ($vpcId)
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
    input  => ($tags, $vpcId),
    output => ($subnetId)
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
