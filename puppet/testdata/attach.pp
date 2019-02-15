type Lyra::Aws::Resource = {
  attributes => {
    ensure => Enum[absent, present],
    region => String,
    tags => Hash[String,String]
  }
}

type Lyra::Aws::Vpc = Lyra::Aws::Resource{
  attributes => {
    vpcId => { type => Optional[String], value => 'FAKED_VPC_ID' },
    cidrBlock => String,
    enableDnsHostnames => Boolean,
    enableDnsSupport => Boolean
  }
}
type Lyra::Aws::VpcHandler = {
  functions => {
    read => Callable[[String], Optional[Lyra::Aws::Vpc]],
    delete => Callable[[String], Boolean],
    create => Callable[[Lyra::Aws::Vpc], Tuple[Lyra::Aws::Vpc,String]]
  }
}
function lyra::aws::vpchandler::read(String $externalId) >> Optional[Lyra::Aws::Vpc] {
  return undef
}
function lyra::aws::vpchandler::create(Lyra::Aws::Vpc $r) >> Tuple[Lyra::Aws::Vpc,String] {
  $rc = Lyra::Aws::Vpc(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    vpcId => 'external-vpc-id',
    cidrBlock => $r.cidrBlock,
    enableDnsHostnames => $r.enableDnsHostnames,
    enableDnsSupport => $r.enableDnsSupport
  )
  return [$rc,$rc.vpcId]
}
function lyra::aws::vpchandler::delete(String $externalId) >> Boolean {
  return true
}
registerHandler(Lyra::Aws::Vpc, Lyra::Aws::VpcHandler())


type Lyra::Aws::Subnet = Lyra::Aws::Resource{
  attributes => {
    subnetId => { type => Optional[String], value => 'FAKED_SUBNET_ID' },
    vpcId => String,
    cidrBlock => String,
    mapPublicIpOnLaunch => Boolean
  }
}
type Lyra::Aws::SubnetHandler = {
  functions => {
    read => Callable[[String], Optional[Lyra::Aws::Subnet]],
    delete => Callable[[String], Boolean],
    create => Callable[[Lyra::Aws::Subnet], Tuple[Lyra::Aws::Subnet,String]]
  }
}
function lyra::aws::subnethandler::read(String $externalId) >> Optional[Lyra::Aws::Subnet] {
  return undef
}
function lyra::aws::subnethandler::create(Lyra::Aws::Subnet $r) >> Tuple[Lyra::Aws::Subnet,String] {
  $rc = Lyra::Aws::Subnet(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    subnetId => 'external-subnet-id',
    vpcId => $r.vpcId,
    cidrBlock => $r.cidrBlock,
    mapPublicIpOnLaunch => $r.mapPublicIpOnLaunch
  )
  return [$rc,$rc.subnetId]
}
function lyra::aws::subnethandler::delete(String $externalId) >> Boolean {
  return true
}
registerHandler(Lyra::Aws::Subnet, Lyra::Aws::SubnetHandler())


type Lyra::Aws::Instance = Lyra::Aws::Resource{
  attributes => {
    instanceId => { type => Optional[String], value => 'FAKED_INSTANCE_ID' },
    instanceType => String,
    imageId => String,
    keyName => String,
    publicIp => { type => Optional[String], value => 'FAKED_PUBLIC_IP' },
    privateIp => { type => Optional[String], value => 'FAKED_PRIVATE_IP' },
  }
}
type Lyra::Aws::InstanceHandler = {
  functions => {
    read => Callable[[String], Optional[Lyra::Aws::Instance]],
    delete => Callable[[String], Boolean],
    create => Callable[[Lyra::Aws::Instance], Tuple[Lyra::Aws::Instance,String]]
  }
}
function lyra::aws::instancehandler::read(String $externalId) >> Optional[Lyra::Aws::Instance] {
  return undef
}
function lyra::aws::instancehandler::create(Lyra::Aws::Instance $r) >> Tuple[Lyra::Aws::Instance,String] {
  $rc = Lyra::Aws::Instance(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    instanceId => 'external-instance-id',
    instanceType => $r.instanceType,
    imageId => $r.imageId,
    keyName => $r.keyName,
    publicIp => '192.168.0.20',
    privateIp => '192.168.1.20'
  )
  return [$rc,$rc.instanceId]
}
function lyra::aws::instancehandler::delete(String $externalId) >> Boolean {
  return true
}
function lyra::aws::instancehandler::update(String $externalId, Lyra::Aws::Vpc $r) >> Lyra::Aws::Instance {
  return $resource
}
registerHandler(Lyra::Aws::Instance, Lyra::Aws::InstanceHandler())


type Lyra::Aws::InternetGateway = Lyra::Aws::Resource{
  attributes => {
    internetGatewayId => { type => Optional[String], value => 'FAKED_GATEWAY_ID' }
  }
}
type Lyra::Aws::InternetGatewayHandler = {
  functions => {
    read => Callable[[String], Optional[Lyra::Aws::InternetGateway]],
    delete => Callable[[String], Boolean],
    create => Callable[[Lyra::Aws::InternetGateway], Tuple[Lyra::Aws::InternetGateway,String]]
  }
}
function lyra::aws::internetGatewayhandler::read(String $externalId) >> Optional[Lyra::Aws::InternetGateway] {
  return undef
}
function lyra::aws::internetGatewayhandler::create(Lyra::Aws::InternetGateway $r) >> Tuple[Lyra::Aws::InternetGateway,String] {
  $rc = Lyra::Aws::InternetGateway(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    internetGatewayId => 'external-internetGatewayId'
  )
  return [$rc,$rc.internetGatewayId]
}
function lyra::aws::internetGatewayhandler::delete(String $externalId) >> Boolean {
  return true
}
registerHandler(Lyra::Aws::InternetGateway, Lyra::Aws::InternetGatewayHandler())

workflow attach {
  typespace => 'lyra::aws',
  input => (
    String $region = lookup('aws.region'),
    Hash[String,String] $tags = lookup('aws.tags'),
    String $keyName = lookup('aws.keyname'),
    Integer $ec2Cnt = lookup('aws.instance.count')
  ),
  output => (
    String $vpcId,
    String $subnetId,
    String $internetGatewayId,
    Hash[String, Struct[publicIp => String, privateIp => String]] $nodes,
    String $notice,
    String $notice2
  )
} {
  resource vpc {
    input  => ($region, $tags),
    output => ($vpcId)
  }{
    ensure => present,
    region => $region,
    cidrBlock => '192.168.0.0/16',
    tags => $tags,
    enableDnsHostnames => true,
    enableDnsSupport => true
  }

  function notice($vpcId) >> Struct[notice=>String] {
    $s = "created VPC with ID ${vpcId}"
    notice("created VPC with ID ${vpcId}")
    return { notice=>$s }
  }

  stateless notice2 {
    input => ($subnetId),
    output => (String $notice2)
  } {
    $s = "created Subnet with ID ${subnetId}"
    notice($s)
    return { notice2 => $s }
  }

  resource subnet {
    input  => ($region, $tags, $vpcId),
    output => ($subnetId)
  }{
    ensure => present,
    region => $region,
    vpcId => $vpcId,
    cidrBlock => '192.168.1.0/24',
    tags => $tags,
    mapPublicIpOnLaunch => true
  }

  resource instance {
    input => ($n, $region, $keyName, $tags),
    output => ($key = instanceId, $value = (publicIp, privateIp))
  } $nodes = times($ec2Cnt) |$n| {
    region => $region,
    ensure => present,
    instanceId => String($n, '%X'),
    imageId => 'ami-f90a4880',
    instanceType => 't2.nano',
    keyName => $keyName,
    tags => $tags
  }

  resource internetgateway {
    input => ($region, $tags),
    output => ($internetGatewayId)
  } {
    ensure => present,
    region => $region,
    tags   => $tags,
  }
}
