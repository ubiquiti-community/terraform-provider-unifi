package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "`unifi_network` data source can be used to retrieve settings for a network by name or ID.",

		ReadContext: dataNetworkRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Description:   "The ID of the network.",
				Type:          schema.TypeString,
				Computed:      true,
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"site": {
				Description: "The name of the site to associate the network with.",
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
			},
			"name": {
				Description:   "The name of the network.",
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"id"},
			},

			// read-only / computed
			"purpose": {
				Description: "The purpose of the network. One of `corporate`, `guest`, `wan`, or `vlan-only`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"vlan_id": {
				Description: "The VLAN ID of the network.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"subnet": {
				Description: "The subnet of the network (CIDR address).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"network_group": {
				Description: "The group of the network.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcp_start": {
				Description: "The IPv4 address where the DHCP range of addresses starts.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcp_stop": {
				Description: "The IPv4 address where the DHCP range of addresses stops.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcp_enabled": {
				Description: "whether DHCP is enabled or not on this network.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcp_lease": {
				Description: "lease time for DHCP addresses.",
				Type:        schema.TypeInt,
				Computed:    true,
			},

			"dhcp_dns": {
				Description: "IPv4 addresses for the DNS server to be returned from the DHCP " +
					"server.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"dhcpd_boot_enabled": {
				Description: "Toggles on the DHCP boot options. will be set to true if you have dhcpd_boot_filename, and dhcpd_boot_server set.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpd_boot_server": {
				Description: "IPv4 address of a TFTP server to network boot from.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_boot_filename": {
				Description: "the file to PXE boot from on the dhcpd_boot_server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcp_v6_dns": {
				Description: "Specifies the IPv6 addresses for the DNS server to be returned from the DHCP " +
					"server. Used if `dhcp_v6_dns_auto` is set to `false`.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"dhcp_v6_dns_auto": {
				Description: "Specifies DNS source to propagate. If set `false` the entries in `dhcp_v6_dns` are used, the upstream entries otherwise",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcp_v6_enabled": {
				Description: "Enable stateful DHCPv6 for static configuration.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcp_v6_lease": {
				Description: "Specifies the lease time for DHCPv6 addresses.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"dhcp_v6_start": {
				Description: "Start address of the DHCPv6 range. Used in static DHCPv6 configuration.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcp_v6_stop": {
				Description: "End address of the DHCPv6 range. Used in static DHCPv6 configuration.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"domain_name": {
				Description: "The domain name of this network.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"igmp_snooping": {
				Description: "Specifies whether IGMP snooping is enabled or not.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"ip_subnet": {
				Description: "The IPv4 subnet of the network in CIDR notation.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_interface_type": {
				Description: "Specifies which type of IPv6 connection to use. Must be one of either `static`, `pd`, or `none`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_static_subnet": {
				Description: "Specifies the static IPv6 subnet (when ipv6_interface_type is 'static').",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_pd_interface": {
				Description: "Specifies which WAN interface to use for IPv6 PD. Must be one of either `wan` or `wan2`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_pd_prefixid": {
				Description: "Specifies the IPv6 Prefix ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_pd_start": {
				Description: "Start address of the DHCPv6 range. Used if `ipv6_interface_type` is set to `pd`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_pd_stop": {
				Description: "End address of the DHCPv6 range. Used if `ipv6_interface_type` is set to `pd`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_ra_enable": {
				Description: "Specifies whether to enable router advertisements or not.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"ipv6_ra_preferred_lifetime": {
				Description: "Lifetime in which the address can be used. Address becomes deprecated afterwards. Must be lower than or equal to `ipv6_ra_valid_lifetime`",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"ipv6_ra_priority": {
				Description: "IPv6 router advertisement priority. Must be one of either `high`, `medium`, or `low`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"ipv6_ra_valid_lifetime": {
				Description: "Total lifetime in which the address can be used. Must be equal to or greater than `ipv6_ra_preferred_lifetime`.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"multicast_dns": {
				Description: "Specifies whether Multicast DNS (mDNS) is enabled or not on the network (Controller >=v7).",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"wan_ip": {
				Description: "The IPv4 address of the WAN.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_netmask": {
				Description: "The IPv4 netmask of the WAN.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_gateway": {
				Description: "The IPv4 gateway of the WAN.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_dns": {
				Description: "DNS servers IPs of the WAN.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"wan_type": {
				Description: "Specifies the IPV4 WAN connection type. One of either `disabled`, `static`, `dhcp`, or `pppoe`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_networkgroup": {
				Description: "Specifies the WAN network group. One of either `WAN`, `WAN2` or `WAN_LTE_FAILOVER`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_egress_qos": {
				Description: "Specifies the WAN egress quality of service.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_username": {
				Description: "Specifies the IPV4 WAN username.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"x_wan_password": {
				Description: "Specifies the IPV4 WAN password.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_type_v6": {
				Description: "Specifies the IPV6 WAN connection type. Must be one of either `disabled`, `static`, or `dhcpv6`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_dhcp_v6_pd_size": {
				Description: "Specifies the IPv6 prefix size to request from ISP. Must be a number between 48 and 64.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_ipv6": {
				Description: "The IPv6 address of the WAN.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_gateway_v6": {
				Description: "The IPv6 gateway of the WAN.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_prefixlen": {
				Description: "The IPv6 prefix length of the WAN. Must be between 1 and 128.",
				Type:        schema.TypeInt,
				Computed:    true,
			},

			// Additional fields from Network struct
			"hidden": {
				Description: "Specifies whether the network is hidden.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"hidden_id": {
				Description: "The hidden network ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"no_delete": {
				Description: "Specifies whether the network can be deleted.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"no_edit": {
				Description: "Specifies whether the network can be edited.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"auto_scale_enabled": {
				Description: "Specifies whether auto scaling is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpd_gateway": {
				Description: "The IPv4 address of the DHCP gateway.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_gateway_enabled": {
				Description: "Specifies whether the DHCP gateway is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpd_dns_enabled": {
				Description: "Specifies whether DHCP DNS is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpd_ntp_1": {
				Description: "IPv4 address of the first NTP server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_ntp_2": {
				Description: "IPv4 address of the second NTP server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_ntp_enabled": {
				Description: "Specifies whether DHCP NTP is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpd_tftp_server": {
				Description: "IPv4 address of the TFTP server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_time_offset": {
				Description: "The time offset for DHCP.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"dhcpd_time_offset_enabled": {
				Description: "Specifies whether DHCP time offset is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpd_unifi_controller": {
				Description: "IPv4 address of the UniFi Controller.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_wpad_url": {
				Description: "The WPAD URL for DHCP.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_wins_1": {
				Description: "IPv4 address of the first WINS server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_wins_2": {
				Description: "IPv4 address of the second WINS server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"dhcpd_wins_enabled": {
				Description: "Specifies whether DHCP WINS is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcp_relay_enabled": {
				Description: "Specifies whether DHCP relay is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcpguard_enabled": {
				Description: "Specifies whether DHCP guard is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dhcp_v6_allow_slaac": {
				Description: "Specifies whether DHCPv6 SLAAC is allowed.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"enabled": {
				Description: "Specifies whether the network is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"internet_access_enabled": {
				Description: "Specifies whether internet access is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"is_nat": {
				Description: "Specifies whether NAT is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"network_isolation_enabled": {
				Description: "Specifies whether network isolation is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"upnp_lan_enabled": {
				Description: "Specifies whether UPnP LAN is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"vlan_enabled": {
				Description: "Specifies whether VLAN is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dpi_enabled": {
				Description: "Specifies whether DPI (Deep Packet Inspection) is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"dpigroup_id": {
				Description: "The DPI group ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"exposed_to_site_vpn": {
				Description: "Specifies whether the network is exposed to site VPN.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"gateway_device": {
				Description: "The MAC address of the gateway device.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"gateway_type": {
				Description: "The gateway type (default or switch).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"igmp_fastleave": {
				Description: "Specifies whether IGMP fast leave is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"igmp_groupmembership": {
				Description: "IGMP group membership interval.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"igmp_maxresponse": {
				Description: "IGMP maximum response time.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"igmp_mcrtrexpiretime": {
				Description: "IGMP multicast router expiration time.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"igmp_proxy_downstream": {
				Description: "Specifies whether IGMP proxy downstream is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"igmp_proxy_upstream": {
				Description: "Specifies whether IGMP proxy upstream is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"igmp_querier": {
				Description: "IPv4 address of the IGMP querier.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"igmp_suppression": { // Note: UniFi API field name differs from schema name
				Description: "Specifies whether IGMP suppression is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"interface_mtu": {
				Description: "The MTU for the interface.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"interface_mtu_enabled": {
				Description: "Specifies whether interface MTU is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"lte_lan_enabled": {
				Description: "Specifies whether LTE LAN is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"mac_override": {
				Description: "MAC address override for the network interface.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"mac_override_enabled": {
				Description: "Specifies whether MAC override is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"priority": {
				Description: "The priority of the network (1-4).",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"radiusprofile_id": {
				Description: "The RADIUS profile ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"usergroup_id": {
				Description: "The user group ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"report_wan_event": {
				Description: "Specifies whether to report WAN events.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"setting_preference": {
				Description: "Setting preference (auto or manual).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"single_network_lan": {
				Description: "Single network LAN configuration.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_load_balance_type": {
				Description: "WAN load balance type (failover-only or weighted).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_load_balance_weight": {
				Description: "WAN load balance weight.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_smartq_enabled": {
				Description: "Specifies whether WAN Smart Queue is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"wan_smartq_up_rate": {
				Description: "WAN Smart Queue upload rate.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_smartq_down_rate": {
				Description: "WAN Smart Queue download rate.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_vlan": {
				Description: "WAN VLAN ID.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_vlan_enabled": {
				Description: "Specifies whether WAN VLAN is enabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"wan_dhcp_cos": {
				Description: "WAN DHCP Class of Service.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"wan_dns_preference": {
				Description: "WAN DNS preference (auto or manual).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wan_ipv6_dns": {
				Description: "WAN IPv6 DNS servers.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"wan_ipv6_dns_preference": {
				Description: "WAN IPv6 DNS preference (auto or manual).",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c, ok := meta.(*client)
	if !ok {
		return diag.Errorf("meta is not of type *client")
	}

	name, ok := d.Get("name").(string)
	if !ok {
		return diag.Errorf("name is not a string")
	}
	site, ok := d.Get("site").(string)
	if !ok {
		return diag.Errorf("site is not a string")
	}
	id, ok := d.Get("id").(string)
	if !ok {
		return diag.Errorf("id is not a string")
	}
	if site == "" {
		site = c.site
	}
	if (name == "" && id == "") || (name != "" && id != "") {
		return diag.Errorf("One of 'name' OR 'id' is required")
	}

	networks, err := c.c.ListNetwork(ctx, site)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, n := range networks {
		if (name != "" && n.Name == name) || (id != "" && n.ID == id) {
			dhcpDNS := []string{}
			for _, dns := range []string{
				n.DHCPDDNS1,
				n.DHCPDDNS2,
				n.DHCPDDNS3,
				n.DHCPDDNS4,
			} {
				if dns == "" {
					continue
				}
				dhcpDNS = append(dhcpDNS, dns)
			}

			// Handle DHCPv6 DNS
			dhcpV6DNS := []string{}
			for _, dns := range []string{
				n.DHCPDV6DNS1,
				n.DHCPDV6DNS2,
				n.DHCPDV6DNS3,
				n.DHCPDV6DNS4,
			} {
				if dns == "" {
					continue
				}
				dhcpV6DNS = append(dhcpV6DNS, dns)
			}
			wanDNS := []string{}
			for _, dns := range []string{
				n.WANDNS1,
				n.WANDNS2,
				n.WANDNS3,
				n.WANDNS4,
			} {
				if dns == "" {
					continue
				}
				wanDNS = append(wanDNS, dns)
			}

			d.SetId(n.ID)
			_ = d.Set("site", site)
			_ = d.Set("name", n.Name)
			_ = d.Set("purpose", n.Purpose)
			_ = d.Set("vlan_id", n.VLAN)
			_ = d.Set("subnet", cidrZeroBased(n.IPSubnet))
			_ = d.Set("network_group", n.NetworkGroup)
			_ = d.Set("dhcp_dns", dhcpDNS)
			_ = d.Set("dhcp_v6_dns", dhcpV6DNS)
			_ = d.Set("dhcp_v6_dns_auto", n.DHCPDV6DNSAuto)
			_ = d.Set("dhcp_v6_enabled", n.DHCPDV6Enabled)
			_ = d.Set("dhcp_v6_lease", n.DHCPDV6LeaseTime)
			_ = d.Set("dhcp_v6_start", n.DHCPDV6Start)
			_ = d.Set("dhcp_v6_stop", n.DHCPDV6Stop)
			_ = d.Set("dhcp_start", n.DHCPDStart)
			_ = d.Set("dhcp_stop", n.DHCPDStop)
			_ = d.Set("dhcp_enabled", n.DHCPDEnabled)
			_ = d.Set("dhcp_lease", n.DHCPDLeaseTime)
			_ = d.Set("dhcpd_boot_enabled", n.DHCPDBootEnabled)
			_ = d.Set("dhcpd_boot_server", n.DHCPDBootServer)
			_ = d.Set("dhcpd_boot_filename", n.DHCPDBootFilename)
			_ = d.Set("domain_name", n.DomainName)
			_ = d.Set("ip_subnet", n.IPSubnet)
			_ = d.Set("igmp_snooping", n.IGMPSnooping)
			_ = d.Set("ipv6_interface_type", n.IPV6InterfaceType)
			_ = d.Set("ipv6_static_subnet", n.IPV6Subnet)
			_ = d.Set("ipv6_pd_interface", n.IPV6PDInterface)
			_ = d.Set("ipv6_pd_prefixid", n.IPV6PDPrefixid)
			_ = d.Set("ipv6_pd_start", n.IPV6PDStart)
			_ = d.Set("ipv6_pd_stop", n.IPV6PDStop)
			_ = d.Set("ipv6_ra_enable", n.IPV6RaEnabled)
			_ = d.Set("ipv6_ra_preferred_lifetime", n.IPV6RaPreferredLifetime)
			_ = d.Set("ipv6_ra_priority", n.IPV6RaPriority)
			_ = d.Set("ipv6_ra_valid_lifetime", n.IPV6RaValidLifetime)
			_ = d.Set("multicast_dns", n.MdnsEnabled)
			_ = d.Set("wan_ip", n.WANIP)
			_ = d.Set("wan_netmask", n.WANNetmask)
			_ = d.Set("wan_gateway", n.WANGateway)
			_ = d.Set("wan_type", n.WANType)
			_ = d.Set("wan_dns", wanDNS)
			_ = d.Set("wan_networkgroup", n.WANNetworkGroup)
			_ = d.Set("wan_egress_qos", n.WANEgressQOS)
			_ = d.Set("wan_username", n.WANUsername)
			_ = d.Set("x_wan_password", n.XWANPassword)
			_ = d.Set("wan_type_v6", n.WANTypeV6)
			_ = d.Set("wan_dhcp_v6_pd_size", n.WANDHCPv6PDSize)
			_ = d.Set("wan_ipv6", n.WANIPV6)
			_ = d.Set("wan_gateway_v6", n.WANGatewayV6)
			_ = d.Set("wan_prefixlen", n.WANPrefixlen)
			_ = d.Set("hidden", n.Hidden)
			_ = d.Set("hidden_id", n.HiddenID)
			_ = d.Set("no_delete", n.NoDelete)
			_ = d.Set("no_edit", n.NoEdit)
			_ = d.Set("auto_scale_enabled", n.AutoScaleEnabled)
			_ = d.Set("dhcpd_gateway", n.DHCPDGateway)
			_ = d.Set("dhcpd_gateway_enabled", n.DHCPDGatewayEnabled)
			_ = d.Set("dhcpd_dns_enabled", n.DHCPDDNSEnabled)
			_ = d.Set("dhcpd_ntp_1", n.DHCPDNtp1)
			_ = d.Set("dhcpd_ntp_2", n.DHCPDNtp2)
			_ = d.Set("dhcpd_ntp_enabled", n.DHCPDNtpEnabled)
			_ = d.Set("dhcpd_tftp_server", n.DHCPDTFTPServer)
			_ = d.Set("dhcpd_time_offset", n.DHCPDTimeOffset)
			_ = d.Set("dhcpd_time_offset_enabled", n.DHCPDTimeOffsetEnabled)
			_ = d.Set("dhcpd_unifi_controller", n.DHCPDUnifiController)
			_ = d.Set("dhcpd_wpad_url", n.DHCPDWPAdUrl)
			_ = d.Set("dhcpd_wins_1", n.DHCPDWins1)
			_ = d.Set("dhcpd_wins_2", n.DHCPDWins2)
			_ = d.Set("dhcpd_wins_enabled", n.DHCPDWinsEnabled)
			_ = d.Set("dhcp_relay_enabled", n.DHCPRelayEnabled)
			_ = d.Set("dhcpguard_enabled", n.DHCPguardEnabled)
			_ = d.Set("dhcp_v6_allow_slaac", n.DHCPDV6AllowSlaac)
			_ = d.Set("enabled", n.Enabled)
			_ = d.Set("internet_access_enabled", n.InternetAccessEnabled)
			_ = d.Set("is_nat", n.IsNAT)
			_ = d.Set("network_isolation_enabled", n.NetworkIsolationEnabled)
			_ = d.Set("upnp_lan_enabled", n.UpnpLanEnabled)
			_ = d.Set("vlan_enabled", n.VLANEnabled)
			_ = d.Set("dpi_enabled", n.DPIEnabled)
			_ = d.Set("dpigroup_id", n.DPIgroupID)
			_ = d.Set("exposed_to_site_vpn", n.ExposedToSiteVPN)
			_ = d.Set("gateway_device", n.GatewayDevice)
			_ = d.Set("gateway_type", n.GatewayType)
			_ = d.Set("igmp_fastleave", n.IGMPFastleave)
			_ = d.Set("igmp_groupmembership", n.IGMPGroupmembership)
			_ = d.Set("igmp_maxresponse", n.IGMPMaxresponse)
			_ = d.Set("igmp_mcrtrexpiretime", n.IGMPMcrtrexpiretime)
			_ = d.Set("igmp_proxy_downstream", n.IGMPProxyDownstream)
			_ = d.Set("igmp_proxy_upstream", n.IGMPProxyUpstream)
			_ = d.Set("igmp_querier", n.IGMPQuerier)
			_ = d.Set("igmp_suppression", n.IGMPSupression) // Note: API has misspelled field name
			_ = d.Set("interface_mtu", n.InterfaceMtu)
			_ = d.Set("interface_mtu_enabled", n.InterfaceMtuEnabled)
			_ = d.Set("lte_lan_enabled", n.LteLanEnabled)
			_ = d.Set("mac_override", n.MACOverride)
			_ = d.Set("mac_override_enabled", n.MACOverrideEnabled)
			_ = d.Set("priority", n.Priority)
			_ = d.Set("radiusprofile_id", n.RADIUSProfileID)
			_ = d.Set("usergroup_id", n.UserGroupID)
			_ = d.Set("report_wan_event", n.ReportWANEvent)
			_ = d.Set("setting_preference", n.SettingPreference)
			_ = d.Set("single_network_lan", n.SingleNetworkLan)
			_ = d.Set("wan_load_balance_type", n.WANLoadBalanceType)
			_ = d.Set("wan_load_balance_weight", n.WANLoadBalanceWeight)
			_ = d.Set("wan_smartq_enabled", n.WANSmartqEnabled)
			_ = d.Set("wan_smartq_up_rate", n.WANSmartqUpRate)
			_ = d.Set("wan_smartq_down_rate", n.WANSmartqDownRate)
			_ = d.Set("wan_vlan", n.WANVLAN)
			_ = d.Set("wan_vlan_enabled", n.WANVLANEnabled)
			_ = d.Set("wan_dhcp_cos", n.WANDHCPCos)
			_ = d.Set("wan_dns_preference", n.WANDNSPreference)

			// Handle WAN IPv6 DNS
			wanIPv6DNS := []string{}
			for _, dns := range []string{
				n.WANIPV6DNS1,
				n.WANIPV6DNS2,
			} {
				if dns == "" {
					continue
				}
				wanIPv6DNS = append(wanIPv6DNS, dns)
			}
			_ = d.Set("wan_ipv6_dns", wanIPv6DNS)
			_ = d.Set("wan_ipv6_dns_preference", n.WANIPV6DNSPreference)

			return nil
		}
	}

	return diag.Errorf("network not found with name %s", name)
}
