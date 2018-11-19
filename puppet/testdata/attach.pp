type Genesis::Aws::Resource = {
  attributes => {
    ensure => Enum[absent, present],
    region => String,
    tags => Hash[String,String]
  }
}

type Genesis::Aws::Vpc = Genesis::Aws::Resource{
  attributes => {
    vpc_id => { type => Optional[String], value => 'FAKED_VPC_ID' },
    cidr_block => String,
    enable_dns_hostnames => Boolean,
    enable_dns_support => Boolean
  }
}
type Genesis::Aws::VpcHandler = {
  functions => {
    read => Callable[[String], Optional[Genesis::Aws::Vpc]],
    delete => Callable[[String], Boolean],
    create => Callable[[Genesis::Aws::Vpc], Tuple[Genesis::Aws::Vpc,String]]
  }
}
function genesis::aws::vpchandler::read(String $external_id) >> Optional[Genesis::Aws::Vpc] {
  return undef
}
function genesis::aws::vpchandler::create(Genesis::Aws::Vpc $r) >> Tuple[Genesis::Aws::Vpc,String] {
  $rc = Genesis::Aws::Vpc(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    vpc_id => 'external-vpc-id',
    cidr_block => $r.cidr_block,
    enable_dns_hostnames => $r.enable_dns_hostnames,
    enable_dns_support => $r.enable_dns_support
  )
  return [$rc,$rc.vpc_id]
}
function genesis::aws::vpchandler::delete(String $external_id) >> Boolean {
  return true
}
register_handler(Genesis::Aws::Vpc, Genesis::Aws::VpcHandler())


type Genesis::Aws::Subnet = Genesis::Aws::Resource{
  attributes => {
    subnet_id => { type => Optional[String], value => 'FAKED_SUBNET_ID' },
    vpc_id => String,
    cidr_block => String,
    map_public_ip_on_launch => Boolean
  }
}
type Genesis::Aws::SubnetHandler = {
  functions => {
    read => Callable[[String], Optional[Genesis::Aws::Subnet]],
    delete => Callable[[String], Boolean],
    create => Callable[[Genesis::Aws::Subnet], Tuple[Genesis::Aws::Subnet,String]]
  }
}
function genesis::aws::subnethandler::read(String $external_id) >> Optional[Genesis::Aws::Subnet] {
  return undef
}
function genesis::aws::subnethandler::create(Genesis::Aws::Subnet $r) >> Tuple[Genesis::Aws::Subnet,String] {
  $rc = Genesis::Aws::Subnet(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    subnet_id => 'external-subnet-id',
    vpc_id => $r.vpc_id,
    cidr_block => $r.cidr_block,
    map_public_ip_on_launch => $r.map_public_ip_on_launch
  )
  return [$rc,$rc.subnet_id]
}
function genesis::aws::subnethandler::delete(String $external_id) >> Boolean {
  return true
}
register_handler(Genesis::Aws::Subnet, Genesis::Aws::SubnetHandler())


type Genesis::Aws::Instance = Genesis::Aws::Resource{
  attributes => {
    instance_id => { type => Optional[String], value => 'FAKED_INSTANCE_ID' },
    instance_type => String,
    image_id => String,
    key_name => String,
    public_ip => { type => Optional[String], value => 'FAKED_PUBLIC_IP' },
    private_ip => { type => Optional[String], value => 'FAKED_PRIVATE_IP' },
  }
}
type Genesis::Aws::InstanceHandler = {
  functions => {
    read => Callable[[String], Optional[Genesis::Aws::Instance]],
    delete => Callable[[String], Boolean],
    create => Callable[[Genesis::Aws::Instance], Tuple[Genesis::Aws::Instance,String]]
  }
}
function genesis::aws::instancehandler::read(String $external_id) >> Optional[Genesis::Aws::Instance] {
  return undef
}
function genesis::aws::instancehandler::create(Genesis::Aws::Instance $r) >> Tuple[Genesis::Aws::Instance,String] {
  $rc = Genesis::Aws::Instance(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    instance_id => 'external-instance-id',
    instance_type => $r.instance_type,
    image_id => $r.image_id,
    key_name => $r.key_name,
    public_ip => '192.168.0.20',
    private_ip => '192.168.1.20'
  )
  return [$rc,$rc.instance_id]
}
function genesis::aws::instancehandler::delete(String $external_id) >> Boolean {
  return true
}
function genesis::aws::instancehandler::update(String $external_id, Genesis::Aws::Vpc $r) >> Genesis::Aws::Instance {
  return $resource
}
register_handler(Genesis::Aws::Instance, Genesis::Aws::InstanceHandler())


type Genesis::Aws::InternetGateway = Genesis::Aws::Resource{
  attributes => {
    internet_gateway_id => { type => Optional[String], value => 'FAKED_GATEWAY_ID' }
  }
}
type Genesis::Aws::InternetGatewayHandler = {
  functions => {
    read => Callable[[String], Optional[Genesis::Aws::InternetGateway]],
    delete => Callable[[String], Boolean],
    create => Callable[[Genesis::Aws::InternetGateway], Tuple[Genesis::Aws::InternetGateway,String]]
  }
}
function genesis::aws::internetGatewayhandler::read(String $external_id) >> Optional[Genesis::Aws::InternetGateway] {
  return undef
}
function genesis::aws::internetGatewayhandler::create(Genesis::Aws::InternetGateway $r) >> Tuple[Genesis::Aws::InternetGateway,String] {
  $rc = Genesis::Aws::InternetGateway(
    ensure => $r.ensure,
    region => $r.region,
    tags => $r.tags,
    internet_gateway_id => 'external-internet_gateway_id'
  )
  return [$rc,$rc.internet_gateway_id]
}
function genesis::aws::internetGatewayhandler::delete(String $external_id) >> Boolean {
  return true
}
register_handler(Genesis::Aws::InternetGateway, Genesis::Aws::InternetGatewayHandler())

workflow attach {
  typespace => 'genesis::aws',
  input => (
    String $region = lookup('aws.region'),
    Hash[String,String] $tags = lookup('aws.tags'),
    String $key_name = lookup('aws.keyname'),
    Integer $ec2_cnt = lookup('aws.instance.count')
  ),
  output => (
    String $vpc_id,
    String $subnet_id,
    String $internet_gateway_id,
    Hash[String, Struct[public_ip => String, private_ip => String]] $nodes
  )
} {
  resource vpc {
    input  => ($region, $tags),
    output => ($vpc_id)
  }{
    ensure => present,
    region => $region,
    cidr_block => '192.168.0.0/16',
    tags => $tags,
    enable_dns_hostnames => true,
    enable_dns_support => true
  }

  resource subnet {
    input  => ($region, $tags, $vpc_id),
    output => ($subnet_id)
  }{
    ensure => present,
    region => $region,
    vpc_id => $vpc_id,
    cidr_block => '192.168.1.0/24',
    tags => $tags,
    map_public_ip_on_launch => true
  }

  resource instance {
    input => ($n, $region, $key_name, $tags),
    output => ($key = instance_id, $value = (public_ip, private_ip))
  } $nodes = times($ec2_cnt) |$n| {
    region => $region,
    ensure => present,
    instance_id => String($n, '%X'),
    image_id => 'ami-f90a4880',
    instance_type => 't2.nano',
    key_name => $key_name,
    tags => $tags
  }

  resource internetgateway {
    input => ($region, $tags),
    output => ($internet_gateway_id)
  } {
    ensure => present,
    region => $region,
    tags   => $tags,
  }
}
