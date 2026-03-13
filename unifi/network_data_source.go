package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &networkDataSource{}

func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource defines the data source implementation.
type networkDataSource struct {
	client *Client
}

// dhcpV6ServerModel describes the DHCPv6 server configuration (data source only).
type dhcpV6ServerModel struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	DNSAuto    types.Bool   `tfsdk:"dns_auto"`
	DNSServers types.List   `tfsdk:"dns_servers"`
	Lease      types.Int64  `tfsdk:"lease"`
	Start      types.String `tfsdk:"start"`
	Stop       types.String `tfsdk:"stop"`
}

func (m dhcpV6ServerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":     types.BoolType,
		"dns_auto":    types.BoolType,
		"dns_servers": types.ListType{ElemType: types.StringType},
		"lease":       types.Int64Type,
		"start":       types.StringType,
		"stop":        types.StringType,
	}
}

// networkDataSourceModel describes the data source data model.
type networkDataSourceModel struct {
	// Lookup keys
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
	Name types.String `tfsdk:"name"`

	// Fields shared with resource (all Computed)
	Enabled                types.Bool   `tfsdk:"enabled"`
	AutoScale              types.Bool   `tfsdk:"auto_scale"`
	Subnet                 types.String `tfsdk:"subnet"`
	DomainName             types.String `tfsdk:"domain_name"`
	Vlan                   types.Int64  `tfsdk:"vlan"`
	NetworkIsolation       types.Bool   `tfsdk:"network_isolation"`
	SettingPreference      types.String `tfsdk:"setting_preference"`
	InternetAccess         types.Bool   `tfsdk:"internet_access"`
	IgmpSnooping           types.Bool   `tfsdk:"igmp_snooping"`
	MulticastDNS           types.Bool   `tfsdk:"multicast_dns"`
	GatewayType            types.String `tfsdk:"gateway_type"`
	IPv6InterfaceType      types.String `tfsdk:"ipv6_interface_type"`
	LteLan                 types.Bool   `tfsdk:"lte_lan"`
	IPAliases              types.List   `tfsdk:"ip_aliases"`
	IPv6Aliases            types.List   `tfsdk:"ipv6_aliases"`
	ThirdPartyGateway      types.Bool   `tfsdk:"third_party_gateway"`
	NatOutboundIPAddresses types.List   `tfsdk:"nat_outbound_ip_addresses"`
	DhcpGuarding           types.Object `tfsdk:"dhcp_guarding"`
	DhcpServer             types.Object `tfsdk:"dhcp_server"`
	DhcpRelay              types.Object `tfsdk:"dhcp_relay"`

	// Data-source-only informational fields
	Purpose      types.String `tfsdk:"purpose"`
	NetworkGroup types.String `tfsdk:"network_group"`

	// IPv6 detail fields (DS-only)
	IPv6StaticSubnet        types.String `tfsdk:"ipv6_static_subnet"`
	IPv6PDInterface         types.String `tfsdk:"ipv6_pd_interface"`
	IPv6PDPrefixID          types.String `tfsdk:"ipv6_pd_prefixid"`
	IPv6PDStart             types.String `tfsdk:"ipv6_pd_start"`
	IPv6PDStop              types.String `tfsdk:"ipv6_pd_stop"`
	IPv6RA                  types.Bool   `tfsdk:"ipv6_ra"`
	IPv6RAPreferredLifetime types.Int64  `tfsdk:"ipv6_ra_preferred_lifetime"`
	IPv6RAPriority          types.String `tfsdk:"ipv6_ra_priority"`
	IPv6RAValidLifetime     types.Int64  `tfsdk:"ipv6_ra_valid_lifetime"`

	// DHCPv6 server (DS-only)
	DhcpV6Server types.Object `tfsdk:"dhcp_v6_server"`

	// WAN fields (DS-only)
	WanDNS          types.List   `tfsdk:"wan_dns"`
	WanEgressQOS    types.Int64  `tfsdk:"wan_egress_qos"`
	WanGateway      types.String `tfsdk:"wan_gateway"`
	WanGatewayV6    types.String `tfsdk:"wan_gateway_v6"`
	WanIP           types.String `tfsdk:"wan_ip"`
	WanNetmask      types.String `tfsdk:"wan_netmask"`
	WanNetworkGroup types.String `tfsdk:"wan_network_group"`
	WanType         types.String `tfsdk:"wan_type"`
	WanTypeV6       types.String `tfsdk:"wan_type_v6"`
	WanUsername     types.String `tfsdk:"wan_username"`
}

func (d *networkDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *networkDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_network` data source can be used to retrieve settings for a network by name or ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the network with.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the network.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			// Fields shared with resource (all Computed)
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the network is enabled.",
				Computed:            true,
			},
			"auto_scale": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether auto-scaling is enabled.",
				Computed:            true,
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The IP subnet of the network in CIDR notation.",
				Computed:            true,
			},
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "The domain name for the network.",
				Computed:            true,
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "The VLAN ID for the network.",
				Computed:            true,
			},
			"network_isolation": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether network isolation is enabled.",
				Computed:            true,
			},
			"setting_preference": schema.StringAttribute{
				MarkdownDescription: "Setting preference. One of `auto` or `manual`.",
				Computed:            true,
			},
			"internet_access": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether internet access is enabled.",
				Computed:            true,
			},
			"igmp_snooping": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether IGMP snooping is enabled.",
				Computed:            true,
			},
			"multicast_dns": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether mDNS is enabled.",
				Computed:            true,
			},
			"gateway_type": schema.StringAttribute{
				MarkdownDescription: "The gateway type.",
				Computed:            true,
			},
			"ipv6_interface_type": schema.StringAttribute{
				MarkdownDescription: "Specifies which type of IPv6 connection to use.",
				Computed:            true,
			},
			"lte_lan": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether LTE LAN is enabled.",
				Computed:            true,
			},
			"ip_aliases": schema.ListAttribute{
				MarkdownDescription: "List of IP aliases for the network.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"ipv6_aliases": schema.ListAttribute{
				MarkdownDescription: "List of IPv6 aliases for the network.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"third_party_gateway": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this network uses a third-party gateway.",
				Computed:            true,
			},
			"nat_outbound_ip_addresses": schema.ListNestedAttribute{
				MarkdownDescription: "List of NAT outbound IP addresses.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The IP address.",
							Computed:            true,
						},
						"ip_address_pool": schema.ListAttribute{
							MarkdownDescription: "The IP address pool.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"mode": schema.StringAttribute{
							MarkdownDescription: "The mode.",
							Computed:            true,
						},
						"wan_network_group": schema.StringAttribute{
							MarkdownDescription: "The WAN network group.",
							Computed:            true,
						},
					},
				},
			},
			"dhcp_guarding": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP guarding configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP guarding is enabled.",
						Computed:            true,
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "List of allowed DHCP server IP addresses.",
						Computed:            true,
						ElementType:         types.StringType,
					},
				},
			},
			"dhcp_server": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP server configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"boot": schema.SingleNestedAttribute{
						MarkdownDescription: "DHCP boot settings.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Toggles DHCP boot options.",
								Computed:            true,
							},
							"server": schema.StringAttribute{
								MarkdownDescription: "TFTP server for boot options.",
								Computed:            true,
							},
							"filename": schema.StringAttribute{
								MarkdownDescription: "Boot filename.",
								Computed:            true,
							},
						},
					},
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP server is enabled.",
						Computed:            true,
					},
					"start": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address where the DHCP range starts.",
						Computed:            true,
					},
					"stop": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address where the DHCP range stops.",
						Computed:            true,
					},
					"gateway_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP gateway is enabled.",
						Computed:            true,
					},
					"conflict_checking": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP conflict checking is enabled.",
						Computed:            true,
					},
					"ntp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP NTP is enabled.",
						Computed:            true,
					},
					"time_offset_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP time offset is enabled.",
						Computed:            true,
					},
					"dns_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP DNS is enabled.",
						Computed:            true,
					},
					"leasetime": schema.Int64Attribute{
						MarkdownDescription: "Specifies the lease time for DHCP addresses in seconds.",
						Computed:            true,
					},
					"wins": schema.SingleNestedAttribute{
						MarkdownDescription: "WINS server configuration.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP WINS is enabled.",
								Computed:            true,
							},
							"addresses": schema.ListAttribute{
								MarkdownDescription: "List of WINS server addresses.",
								Computed:            true,
								ElementType:         types.StringType,
							},
						},
					},
					"wpad_url": schema.StringAttribute{
						MarkdownDescription: "WPAD URL for proxy auto-configuration.",
						Computed:            true,
					},
					"tftp_server": schema.StringAttribute{
						MarkdownDescription: "TFTP server address.",
						Computed:            true,
					},
					"unifi_controller": schema.StringAttribute{
						MarkdownDescription: "UniFi controller IP address.",
						Computed:            true,
					},
					"dns_servers": schema.ListAttribute{
						MarkdownDescription: "List of DNS server addresses for DHCP clients.",
						Computed:            true,
						ElementType:         types.StringType,
					},
				},
			},
			"dhcp_relay": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP relay configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP relay is enabled.",
						Computed:            true,
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "List of DHCP relay server addresses.",
						Computed:            true,
						ElementType:         types.StringType,
					},
				},
			},
			// DS-only fields
			"purpose": schema.StringAttribute{
				MarkdownDescription: "The purpose of the network. One of `corporate`, `guest`, `wan`, or `vlan-only`.",
				Computed:            true,
			},
			"network_group": schema.StringAttribute{
				MarkdownDescription: "The group of the network.",
				Computed:            true,
			},
			"ipv6_static_subnet": schema.StringAttribute{
				MarkdownDescription: "The static IPv6 subnet (when `ipv6_interface_type` is `static`).",
				Computed:            true,
			},
			"ipv6_pd_interface": schema.StringAttribute{
				MarkdownDescription: "Specifies which WAN interface to use for IPv6 PD. One of `wan` or `wan2`.",
				Computed:            true,
			},
			"ipv6_pd_prefixid": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPv6 Prefix ID.",
				Computed:            true,
			},
			"ipv6_pd_start": schema.StringAttribute{
				MarkdownDescription: "Start address of the DHCPv6 range when `ipv6_interface_type` is `pd`.",
				Computed:            true,
			},
			"ipv6_pd_stop": schema.StringAttribute{
				MarkdownDescription: "End address of the DHCPv6 range when `ipv6_interface_type` is `pd`.",
				Computed:            true,
			},
			"ipv6_ra": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to enable IPv6 router advertisements.",
				Computed:            true,
			},
			"ipv6_ra_preferred_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Preferred lifetime for IPv6 RA in which the address can be used.",
				Computed:            true,
			},
			"ipv6_ra_priority": schema.StringAttribute{
				MarkdownDescription: "IPv6 router advertisement priority. One of `high`, `medium`, or `low`.",
				Computed:            true,
			},
			"ipv6_ra_valid_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Total lifetime in which the IPv6 RA address can be used.",
				Computed:            true,
			},
			"dhcp_v6_server": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCPv6 server configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether stateful DHCPv6 is enabled.",
						Computed:            true,
					},
					"dns_auto": schema.BoolAttribute{
						MarkdownDescription: "When true, upstream DNS entries are propagated. When false, `dns_servers` are used.",
						Computed:            true,
					},
					"dns_servers": schema.ListAttribute{
						MarkdownDescription: "IPv6 DNS server addresses for DHCPv6 clients.",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"lease": schema.Int64Attribute{
						MarkdownDescription: "Lease time for DHCPv6 addresses in seconds.",
						Computed:            true,
					},
					"start": schema.StringAttribute{
						MarkdownDescription: "Start address of the DHCPv6 range.",
						Computed:            true,
					},
					"stop": schema.StringAttribute{
						MarkdownDescription: "End address of the DHCPv6 range.",
						Computed:            true,
					},
				},
			},
			"wan_dns": schema.ListAttribute{
				MarkdownDescription: "DNS server IPs of the WAN.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"wan_egress_qos": schema.Int64Attribute{
				MarkdownDescription: "Specifies the WAN egress quality of service.",
				Computed:            true,
			},
			"wan_gateway": schema.StringAttribute{
				MarkdownDescription: "The IPv4 gateway of the WAN.",
				Computed:            true,
			},
			"wan_gateway_v6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 gateway of the WAN.",
				Computed:            true,
			},
			"wan_ip": schema.StringAttribute{
				MarkdownDescription: "The IPv4 address of the WAN.",
				Computed:            true,
			},
			"wan_netmask": schema.StringAttribute{
				MarkdownDescription: "The IPv4 netmask of the WAN.",
				Computed:            true,
			},
			"wan_network_group": schema.StringAttribute{
				MarkdownDescription: "Specifies the WAN network group. One of `WAN`, `WAN2` or `WAN_LTE_FAILOVER`.",
				Computed:            true,
			},
			"wan_type": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPv4 WAN connection type. One of `disabled`, `static`, `dhcp`, or `pppoe`.",
				Computed:            true,
			},
			"wan_type_v6": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPv6 WAN connection type. One of `disabled`, `static`, or `dhcpv6`.",
				Computed:            true,
			},
			"wan_username": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPv4 WAN username.",
				Computed:            true,
			},
		},
	}
}

func (d *networkDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	d.client = client
}

func (d *networkDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var config networkDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	var network *unifi.Network
	var err error

	if !config.ID.IsNull() && !config.ID.IsUnknown() {
		id := config.ID.ValueString()
		network, err = d.client.GetNetwork(ctx, site, id)
		if err != nil {
			if _, ok := err.(*unifi.NotFoundError); ok {
				resp.Diagnostics.AddError(
					"Network Not Found",
					fmt.Sprintf("Network with ID %s not found: %s", id, err),
				)
				return
			}
			resp.Diagnostics.AddError(
				"Error Reading Network",
				fmt.Sprintf("Could not read network with ID %s: %s", id, err),
			)
			return
		}
	} else if !config.Name.IsNull() && !config.Name.IsUnknown() {
		name := config.Name.ValueString()
		network, err = d.client.GetNetworkByName(ctx, site, name)
		if err != nil {
			resp.Diagnostics.AddError(
				"Network Not Found",
				fmt.Sprintf("Network with name %s not found", name),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be specified",
		)
		return
	}

	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.Diagnostics.AddError(
				"Network Not Found",
				fmt.Sprintf("Network not found: %s", err),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Network",
			fmt.Sprintf("Could not read network: %s", err),
		)
		return
	}

	// Set attributes from API response
	d.setDataSourceData(ctx, &resp.Diagnostics, network, &config, site)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}

// Helper method to set data source data from API response.
func (d *networkDataSource) setDataSourceData(
	ctx context.Context,
	diags *diag.Diagnostics,
	network *unifi.Network,
	model *networkDataSourceModel,
	site string,
) {
	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringPointerValue(network.Name)
	model.Purpose = types.StringValue(network.Purpose)
	model.NetworkGroup = types.StringPointerValue(network.NetworkGroup)

	// Shared with resource fields
	model.Enabled = types.BoolValue(network.Enabled)
	model.AutoScale = types.BoolValue(network.AutoScaleEnabled)
	model.Subnet = types.StringPointerValue(network.IPSubnet)
	model.DomainName = types.StringPointerValue(network.DomainName)
	model.Vlan = types.Int64PointerValue(network.VLAN)
	model.NetworkIsolation = types.BoolValue(network.NetworkIsolationEnabled)
	model.SettingPreference = types.StringPointerValue(network.SettingPreference)
	model.InternetAccess = types.BoolValue(network.InternetAccessEnabled)
	model.IgmpSnooping = types.BoolValue(network.IGMPSnooping)
	model.MulticastDNS = types.BoolValue(network.MdnsEnabled)
	model.GatewayType = types.StringPointerValue(network.GatewayType)
	model.IPv6InterfaceType = types.StringPointerValue(network.IPV6InterfaceType)
	model.LteLan = types.BoolValue(network.LteLanEnabled)
	model.ThirdPartyGateway = types.BoolValue(network.Purpose == unifi.PurposeVLANOnly)

	// ip_aliases
	if len(network.IPAliases) > 0 {
		ipAliasesList, d := types.ListValueFrom(ctx, types.StringType, network.IPAliases)
		diags.Append(d...)
		model.IPAliases = ipAliasesList
	} else {
		model.IPAliases = types.ListNull(types.StringType)
	}

	// ipv6_aliases — not available in current API
	model.IPv6Aliases = types.ListNull(types.StringType)

	// nat_outbound_ip_addresses — not populated by API read
	model.NatOutboundIPAddresses = types.ListNull(
		types.ObjectType{AttrTypes: natOutboundIPAddresses()},
	)

	// dhcp_guarding
	{
		var serversList types.List
		if network.Purpose == unifi.PurposeVLANOnly {
			servers := collectNonEmptyStrings(network.DHCPDIP1, network.DHCPDIP2, network.DHCPDIP3)
			if len(servers) > 0 {
				l, d := types.ListValueFrom(ctx, types.StringType, servers)
				diags.Append(d...)
				serversList = l
			} else {
				serversList = types.ListNull(types.StringType)
			}
		} else {
			serversList = types.ListNull(types.StringType)
		}
		dhcpGuardingValue := dhcpGuardingModel{
			Enabled: types.BoolValue(network.DHCPguardEnabled),
			Servers: serversList,
		}
		dhcpGuardingObj, d := types.ObjectValueFrom(
			ctx,
			dhcpGuardingValue.AttributeTypes(),
			dhcpGuardingValue,
		)
		diags.Append(d...)
		model.DhcpGuarding = dhcpGuardingObj
	}

	// dhcp_server
	{
		strPtrToType := func(ptr *string) types.String {
			if ptr == nil || *ptr == "" {
				return types.StringNull()
			}
			return types.StringValue(*ptr)
		}

		dhcpBootValue := dhcpBootModel{
			Enabled:  types.BoolValue(network.DHCPDBootEnabled),
			Server:   types.StringValue(network.DHCPDBootServer),
			Filename: strPtrToType(network.DHCPDBootFilename),
		}
		dhcpBootObj, d := types.ObjectValueFrom(ctx, dhcpBootValue.AttributeTypes(), dhcpBootValue)
		diags.Append(d...)

		dnsServers := collectNonEmptyStrings(
			network.DHCPDDNS1, network.DHCPDDNS2, network.DHCPDDNS3, network.DHCPDDNS4,
		)
		var dnsServersList types.List
		if len(dnsServers) > 0 {
			dnsServersList, d = types.ListValueFrom(ctx, types.StringType, dnsServers)
			diags.Append(d...)
		} else {
			dnsServersList = types.ListNull(types.StringType)
		}

		winsAddresses := collectNonEmptyStringPointers(network.DHCPDWins1, network.DHCPDWins2)
		var winsAddressesList types.List
		if len(winsAddresses) > 0 {
			winsAddressesList, d = types.ListValueFrom(ctx, types.StringType, winsAddresses)
			diags.Append(d...)
		} else {
			winsAddressesList = types.ListNull(types.StringType)
		}

		winsValue := winsModel{
			Enabled:   types.BoolValue(network.DHCPDWinsEnabled),
			Addresses: winsAddressesList,
		}
		winsObj, d := types.ObjectValueFrom(ctx, winsValue.AttributeTypes(), winsValue)
		diags.Append(d...)

		dhcpServerValue := dhcpServerModel{
			Boot:              dhcpBootObj,
			Enabled:           types.BoolValue(network.DHCPDEnabled),
			Start:             types.StringPointerValue(network.DHCPDStart),
			Stop:              types.StringPointerValue(network.DHCPDStop),
			GatewayEnabled:    types.BoolValue(network.DHCPDGatewayEnabled),
			ConflictChecking:  types.BoolValue(network.DHCPDConflictChecking),
			NtpEnabled:        types.BoolValue(network.DHCPDNtpEnabled),
			TimeOffsetEnabled: types.BoolValue(network.DHCPDTimeOffsetEnabled),
			DnsEnabled:        types.BoolValue(network.DHCPDDNSEnabled),
			Leasetime:         types.Int64PointerValue(network.DHCPDLeaseTime),
			Wins:              winsObj,
			WpadUrl:           strPtrToType(network.DHCPDWPAdUrl),
			TftpServer:        strPtrToType(network.DHCPDTFTPServer),
			UnifiController:   strPtrToType(network.DHCPDUnifiController),
			DnsServers:        dnsServersList,
		}
		dhcpServerObj, d := types.ObjectValueFrom(
			ctx,
			dhcpServerValue.AttributeTypes(),
			dhcpServerValue,
		)
		diags.Append(d...)
		model.DhcpServer = dhcpServerObj
	}

	// dhcp_relay
	{
		dhcpRelayValue := dhcpRelayModel{
			Enabled: types.BoolValue(network.DHCPRelayEnabled),
			Servers: types.ListNull(types.StringType),
		}
		dhcpRelayObj, d := types.ObjectValueFrom(
			ctx,
			dhcpRelayValue.AttributeTypes(),
			dhcpRelayValue,
		)
		diags.Append(d...)
		model.DhcpRelay = dhcpRelayObj
	}

	// DS-only IPv6 fields
	model.IPv6StaticSubnet = types.StringPointerValue(network.IPV6Subnet)
	model.IPv6PDInterface = types.StringPointerValue(network.IPV6PDInterface)
	if network.IPV6PDPrefixid == "" {
		model.IPv6PDPrefixID = types.StringNull()
	} else {
		model.IPv6PDPrefixID = types.StringValue(network.IPV6PDPrefixid)
	}
	model.IPv6PDStart = types.StringPointerValue(network.IPV6PDStart)
	model.IPv6PDStop = types.StringPointerValue(network.IPV6PDStop)
	model.IPv6RA = types.BoolValue(network.IPV6RaEnabled)
	model.IPv6RAPreferredLifetime = types.Int64PointerValue(network.IPV6RaPreferredLifetime)
	model.IPv6RAPriority = types.StringPointerValue(network.IPV6RaPriority)
	model.IPv6RAValidLifetime = types.Int64PointerValue(network.IPV6RaValidLifetime)

	// dhcp_v6_server
	{
		dhcpv6DNS := collectNonEmptyStringPointers(
			network.DHCPDV6DNS1, network.DHCPDV6DNS2,
			network.DHCPDV6DNS3, network.DHCPDV6DNS4,
		)
		var dhcpv6DNSList types.List
		if len(dhcpv6DNS) > 0 {
			l, d := types.ListValueFrom(ctx, types.StringType, dhcpv6DNS)
			diags.Append(d...)
			dhcpv6DNSList = l
		} else {
			dhcpv6DNSList = types.ListNull(types.StringType)
		}

		dhcpV6ServerValue := dhcpV6ServerModel{
			Enabled:    types.BoolValue(network.DHCPDV6Enabled),
			DNSAuto:    types.BoolValue(network.DHCPDV6DNSAuto),
			DNSServers: dhcpv6DNSList,
			Lease:      types.Int64PointerValue(network.DHCPDV6LeaseTime),
			Start:      types.StringPointerValue(network.DHCPDV6Start),
			Stop:       types.StringPointerValue(network.DHCPDV6Stop),
		}
		dhcpV6ServerObj, d := types.ObjectValueFrom(
			ctx,
			dhcpV6ServerValue.AttributeTypes(),
			dhcpV6ServerValue,
		)
		diags.Append(d...)
		model.DhcpV6Server = dhcpV6ServerObj
	}

	// WAN DNS
	wanDNS := collectNonEmptyStringPointers(network.WANDNS1, network.WANDNS2)
	wanDNS = append(wanDNS, collectNonEmptyStrings(network.WANDNS3, network.WANDNS4)...)
	if len(wanDNS) > 0 {
		wanDNSList, d := types.ListValueFrom(ctx, types.StringType, wanDNS)
		diags.Append(d...)
		model.WanDNS = wanDNSList
	} else {
		model.WanDNS = types.ListNull(types.StringType)
	}

	model.WanEgressQOS = types.Int64PointerValue(network.WANEgressQOS)
	model.WanGateway = types.StringPointerValue(network.WANGateway)
	if network.WANGatewayV6 == "" {
		model.WanGatewayV6 = types.StringNull()
	} else {
		model.WanGatewayV6 = types.StringValue(network.WANGatewayV6)
	}
	model.WanIP = types.StringPointerValue(network.WANIP)
	model.WanNetmask = types.StringPointerValue(network.WANNetmask)
	model.WanNetworkGroup = types.StringPointerValue(network.WANNetworkGroup)
	model.WanType = types.StringPointerValue(network.WANType)
	model.WanTypeV6 = types.StringPointerValue(network.WANTypeV6)
	if network.WANUsername == "" {
		model.WanUsername = types.StringNull()
	} else {
		model.WanUsername = types.StringValue(network.WANUsername)
	}
}

// collectNonEmptyStrings returns a slice of non-empty strings from the provided values.
func collectNonEmptyStrings(vals ...string) []string {
	var result []string
	for _, v := range vals {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// collectNonEmptyStringPointers returns a slice of non-nil, non-empty strings from the provided pointers.
func collectNonEmptyStringPointers(ptrs ...*string) []string {
	var result []string
	for _, p := range ptrs {
		if p != nil && *p != "" {
			result = append(result, *p)
		}
	}
	return result
}
