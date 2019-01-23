type Lyra::Aws::Resource = {
  attributes => {
    ensure => Enum[absent, present],
    region => String,
    tags => Hash[String,String]
  }
}

type Lyra::Aws::Vpc = Lyra::Aws::Resource{
  attributes => {
    vpc_id => { type => Optional[String], value => 'FAKED_VPC_ID' },
    cidr_block => String,
    enable_dns_hostnames => Boolean,
    enable_dns_support => Boolean
  }
}

type Lyra::Aws::Subnet = Lyra::Aws::Resource{
  attributes => {
    subnet_id => { type => Optional[String], value => 'FAKED_SUBNET_ID' },
    vpc_id => String,
    cidr_block => String,
    map_public_ip_on_launch => Boolean
  }
}

type Lyra::Aws::Instance = Lyra::Aws::Resource{
  attributes => {
    instance_id => { type => Optional[String], value => 'FAKED_INSTANCE_ID' },
    instance_type => String,
    image_id => String,
    key_name => String,
    public_ip => { type => Optional[String], value => 'FAKED_PUBLIC_IP' },
    private_ip => { type => Optional[String], value => 'FAKED_PRIVATE_IP' },
  }
}

type Lyra::Aws::InternetGateway = Lyra::Aws::Resource{
  attributes => {
    internet_gateway_id => { type => Optional[String], value => 'FAKED_GATEWAY_ID' }
  }
}
