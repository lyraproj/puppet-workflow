# this file is generated
type Aws = TypeSet[{
  pcore_uri => 'http://puppet.com/2016.1/pcore',
  pcore_version => '1.0.0',
  name_authority => 'http://puppet.com/2016.1/runtime',
  name => 'Aws',
  version => '0.1.0',
  types => {
    Subnet => {
      annotations => {
        Lyra::Resource => {
          'immutableAttributes' => ['tags'],
          'providedAttributes' => ['subnetId', 'availabilityZone', 'availableIpAddressCount']
        }
      },
      attributes => {
        'vpcId' => String,
        'cidrBlock' => String,
        'availabilityZone' => {
          'type' => Optional[String],
          'value' => undef
        },
        'ipv6CidrBlock' => String,
        'tags' => Hash[String, String],
        'assignIpv6AddressOnCreation' => Boolean,
        'mapPublicIpOnLaunch' => Boolean,
        'availableIpAddressCount' => {
          'type' => Optional[Integer],
          'value' => undef
        },
        'defaultForAz' => Boolean,
        'state' => String,
        'subnetId' => {
          'type' => Optional[String],
          'value' => undef
        }
      }
    },
    SubnetHandler => {
      functions => {
        'create' => Callable[Optional[Subnet], Tuple[Optional[Subnet], String]],
        'delete' => Callable[String],
        'read' => Callable[String, Optional[Subnet]]
      }
    },
    VPCHandler => {
      functions => {
        'create' => Callable[Optional[Vpc], Tuple[Optional[Vpc], String]],
        'delete' => Callable[String],
        'read' => Callable[String, Optional[Vpc]]
      }
    },
    Vpc => {
      attributes => {
        'amazonProvidedIpv6CidrBlock' => Boolean,
        'cidrBlock' => String,
        'instanceTenancy' => {
          'type' => Optional[String],
          'value' => 'default'
        },
        'enableDnsHostnames' => Boolean,
        'enableDnsSupport' => Boolean,
        'tags' => Hash[String, String],
        'vpcId' => {
          'type' => Optional[String],
          'value' => undef
        },
        'isDefault' => Boolean,
        'state' => String,
        'dhcpOptionsId' => {
          'type' => Optional[String],
          'value' => undef
        }
      }
    }
  }
}]
