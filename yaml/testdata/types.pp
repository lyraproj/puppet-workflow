type Lyra::Aws::Resource = {
  attributes => {
    region => String,
    tags => Optional[Hash[String,String]]
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

type Lyra::Aws::Subnet = Lyra::Aws::Resource{
  attributes => {
    subnetId => { type => Optional[String], value => 'FAKED_SUBNET_ID' },
    vpcId => String,
    cidrBlock => String,
    mapPublicIpOnLaunch => Boolean
  }
}

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

type Lyra::Aws::InternetGateway = Lyra::Aws::Resource{
  attributes => {
    internetGatewayId => { type => Optional[String], value => 'FAKED_GATEWAY_ID' },
    nestedInfo => { type => Optional[Hash[String,Hash[String,String]]], value => undef }
  }
}
