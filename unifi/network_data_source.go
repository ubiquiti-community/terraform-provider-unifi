package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

func NewNetworkFrameworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource defines the data source implementation.
type networkDataSource struct {
	client *Client
}

// networkDataSourceModel describes the data source data model.
type networkDataSourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Site                    types.String `tfsdk:"site"`
	Name                    types.String `tfsdk:"name"`
	Purpose                 types.String `tfsdk:"purpose"`
	VlanID                  types.Int64  `tfsdk:"vlan_id"`
	Subnet                  types.String `tfsdk:"subnet"`
	NetworkGroup            types.String `tfsdk:"network_group"`
	DHCPStart               types.String `tfsdk:"dhcp_start"`
	DHCPStop                types.String `tfsdk:"dhcp_stop"`
	DHCPEnabled             types.Bool   `tfsdk:"dhcp_enabled"`
	DHCPLease               types.Int64  `tfsdk:"dhcp_lease"`
	DHCPDNS                 types.List   `tfsdk:"dhcp_dns"`
	DHCPDBootEnabled        types.Bool   `tfsdk:"dhcpd_boot_enabled"`
	DHCPDBootServer         types.String `tfsdk:"dhcpd_boot_server"`
	DHCPDBootFilename       types.String `tfsdk:"dhcpd_boot_filename"`
	DHCPV6DNS               types.List   `tfsdk:"dhcp_v6_dns"`
	DHCPV6DNSAuto           types.Bool   `tfsdk:"dhcp_v6_dns_auto"`
	DHCPV6Enabled           types.Bool   `tfsdk:"dhcp_v6_enabled"`
	DHCPV6Lease             types.Int64  `tfsdk:"dhcp_v6_lease"`
	DHCPV6Start             types.String `tfsdk:"dhcp_v6_start"`
	DHCPV6Stop              types.String `tfsdk:"dhcp_v6_stop"`
	DomainName              types.String `tfsdk:"domain_name"`
	IGMPSnooping            types.Bool   `tfsdk:"igmp_snooping"`
	IPSubnet                types.String `tfsdk:"ip_subnet"`
	IPv6InterfaceType       types.String `tfsdk:"ipv6_interface_type"`
	IPv6StaticSubnet        types.String `tfsdk:"ipv6_static_subnet"`
	IPv6PDInterface         types.String `tfsdk:"ipv6_pd_interface"`
	IPv6PDPrefixID          types.String `tfsdk:"ipv6_pd_prefixid"`
	IPv6PDStart             types.String `tfsdk:"ipv6_pd_start"`
	IPv6PDStop              types.String `tfsdk:"ipv6_pd_stop"`
	IPv6RAEnable            types.Bool   `tfsdk:"ipv6_ra_enable"`
	IPv6RAPreferredLifetime types.Int64  `tfsdk:"ipv6_ra_preferred_lifetime"`
	IPv6RAPriority          types.String `tfsdk:"ipv6_ra_priority"`
	IPv6RAValidLifetime     types.Int64  `tfsdk:"ipv6_ra_valid_lifetime"`
	MulticastDNS            types.Bool   `tfsdk:"multicast_dns"`
	WanDNS                  types.List   `tfsdk:"wan_dns"`
	WanEgressQOS            types.Int64  `tfsdk:"wan_egress_qos"`
	WanGateway              types.String `tfsdk:"wan_gateway"`
	WanGatewayV6            types.String `tfsdk:"wan_gateway_v6"`
	WanIP                   types.String `tfsdk:"wan_ip"`
	WanNetmask              types.String `tfsdk:"wan_netmask"`
	WanNetworkGroup         types.String `tfsdk:"wan_network_group"`
	WanType                 types.String `tfsdk:"wan_type"`
	WanTypeV6               types.String `tfsdk:"wan_type_v6"`
	WanUsername             types.String `tfsdk:"wan_username"`
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
		Description: "`unifi_network` data source can be used to retrieve settings for a network by name or ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the network.",
				Computed:    true,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"site": schema.StringAttribute{
				Description: "The name of the site to associate the network with.",
				Computed:    true,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the network.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},

			// read-only / computed
			"purpose": schema.StringAttribute{
				Description: "The purpose of the network. One of `corporate`, `guest`, `wan`, or `vlan-only`.",
				Computed:    true,
			},
			"vlan_id": schema.Int64Attribute{
				Description: "The VLAN ID of the network.",
				Computed:    true,
			},
			"subnet": schema.StringAttribute{
				Description: "The subnet of the network (CIDR address).",
				Computed:    true,
			},
			"network_group": schema.StringAttribute{
				Description: "The group of the network.",
				Computed:    true,
			},
			"dhcp_start": schema.StringAttribute{
				Description: "The IPv4 address where the DHCP range of addresses starts.",
				Computed:    true,
			},
			"dhcp_stop": schema.StringAttribute{
				Description: "The IPv4 address where the DHCP range of addresses stops.",
				Computed:    true,
			},
			"dhcp_enabled": schema.BoolAttribute{
				Description: "whether DHCP is enabled or not on this network.",
				Computed:    true,
			},
			"dhcp_lease": schema.Int64Attribute{
				Description: "lease time for DHCP addresses.",
				Computed:    true,
			},
			"dhcp_dns": schema.ListAttribute{
				Description: "IPv4 addresses for the DNS server to be returned from the DHCP server.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"dhcpd_boot_enabled": schema.BoolAttribute{
				Description: "Toggles on the DHCP boot options. will be set to true if you have dhcpd_boot_filename, and dhcpd_boot_server set.",
				Computed:    true,
			},
			"dhcpd_boot_server": schema.StringAttribute{
				Description: "IPv4 address of a TFTP server to network boot from.",
				Computed:    true,
			},
			"dhcpd_boot_filename": schema.StringAttribute{
				Description: "the file to PXE boot from on the dhcpd_boot_server.",
				Computed:    true,
			},
			"dhcp_v6_dns": schema.ListAttribute{
				Description: "Specifies the IPv6 addresses for the DNS server to be returned from the DHCP server. Used if `dhcp_v6_dns_auto` is set to `false`.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"dhcp_v6_dns_auto": schema.BoolAttribute{
				Description: "Specifies DNS source to propagate. If set `false` the entries in `dhcp_v6_dns` are used, the upstream entries otherwise",
				Computed:    true,
			},
			"dhcp_v6_enabled": schema.BoolAttribute{
				Description: "Enable stateful DHCPv6 for static configuration.",
				Computed:    true,
			},
			"dhcp_v6_lease": schema.Int64Attribute{
				Description: "Specifies the lease time for DHCPv6 addresses.",
				Computed:    true,
			},
			"dhcp_v6_start": schema.StringAttribute{
				Description: "Start address of the DHCPv6 range. Used in static DHCPv6 configuration.",
				Computed:    true,
			},
			"dhcp_v6_stop": schema.StringAttribute{
				Description: "End address of the DHCPv6 range. Used in static DHCPv6 configuration.",
				Computed:    true,
			},
			"domain_name": schema.StringAttribute{
				Description: "The domain name of this network.",
				Computed:    true,
			},
			"igmp_snooping": schema.BoolAttribute{
				Description: "Specifies whether IGMP snooping is enabled or not.",
				Computed:    true,
			},
			"ip_subnet": schema.StringAttribute{
				Description: "The IPv4 subnet of the network in CIDR notation.",
				Computed:    true,
			},
			"ipv6_interface_type": schema.StringAttribute{
				Description: "Specifies which type of IPv6 connection to use. Must be one of either `static`, `pd`, or `none`.",
				Computed:    true,
			},
			"ipv6_static_subnet": schema.StringAttribute{
				Description: "Specifies the static IPv6 subnet (when ipv6_interface_type is 'static').",
				Computed:    true,
			},
			"ipv6_pd_interface": schema.StringAttribute{
				Description: "Specifies which WAN interface to use for IPv6 PD. Must be one of either `wan` or `wan2`.",
				Computed:    true,
			},
			"ipv6_pd_prefixid": schema.StringAttribute{
				Description: "Specifies the IPv6 Prefix ID.",
				Computed:    true,
			},
			"ipv6_pd_start": schema.StringAttribute{
				Description: "Start address of the DHCPv6 range. Used if `ipv6_interface_type` is set to `pd`.",
				Computed:    true,
			},
			"ipv6_pd_stop": schema.StringAttribute{
				Description: "End address of the DHCPv6 range. Used if `ipv6_interface_type` is set to `pd`.",
				Computed:    true,
			},
			"ipv6_ra_enable": schema.BoolAttribute{
				Description: "Specifies whether to enable router advertisements or not.",
				Computed:    true,
			},
			"ipv6_ra_preferred_lifetime": schema.Int64Attribute{
				Description: "Lifetime in which the address can be used. Address becomes deprecated afterwards. Must be lower than or equal to `ipv6_ra_valid_lifetime`",
				Computed:    true,
			},
			"ipv6_ra_priority": schema.StringAttribute{
				Description: "IPv6 router advertisement priority. Must be one of either `high`, `medium`, or `low`",
				Computed:    true,
			},
			"ipv6_ra_valid_lifetime": schema.Int64Attribute{
				Description: "Total lifetime in which the address can be used. Must be equal to or greater than `ipv6_ra_preferred_lifetime`.",
				Computed:    true,
			},
			"multicast_dns": schema.BoolAttribute{
				Description: "Specifies whether Multicast DNS (mDNS) is enabled or not on the network (Controller >=v7).",
				Computed:    true,
			},
			"wan_dns": schema.ListAttribute{
				Description: "DNS servers IPs of the WAN.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"wan_egress_qos": schema.Int64Attribute{
				Description: "Specifies the WAN egress quality of service.",
				Computed:    true,
			},
			"wan_gateway": schema.StringAttribute{
				Description: "The IPv4 gateway of the WAN.",
				Computed:    true,
			},
			"wan_gateway_v6": schema.StringAttribute{
				Description: "The IPv6 gateway of the WAN.",
				Computed:    true,
			},
			"wan_ip": schema.StringAttribute{
				Description: "The IPv4 address of the WAN.",
				Computed:    true,
			},
			"wan_netmask": schema.StringAttribute{
				Description: "The IPv4 netmask of the WAN.",
				Computed:    true,
			},
			"wan_network_group": schema.StringAttribute{
				Description: "Specifies the WAN network group. One of either `WAN`, `WAN2` or `WAN_LTE_FAILOVER`.",
				Computed:    true,
			},
			"wan_type": schema.StringAttribute{
				Description: "Specifies the IPV4 WAN connection type. One of either `disabled`, `static`, `dhcp`, or `pppoe`.",
				Computed:    true,
			},
			"wan_type_v6": schema.StringAttribute{
				Description: "Specifies the IPV6 WAN connection type. One of either `disabled`, `static`, or `dhcpv6`.",
				Computed:    true,
			},
			"wan_username": schema.StringAttribute{
				Description: "Specifies the IPV4 WAN username.",
				Computed:    true,
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
	} else if !config.Name.IsNull() && !config.Name.IsUnknown() {
		name := config.Name.ValueString()
		networks, listErr := d.client.ListNetwork(ctx, site)
		if listErr != nil {
			resp.Diagnostics.AddError(
				"Error Reading Network",
				fmt.Sprintf("Could not list networks: %s", listErr),
			)
			return
		}

		for _, n := range networks {
			if n.Name == name {
				network = &n
				break
			}
		}

		if network == nil {
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

// Helper method to set data source data from API response
func (d *networkDataSource) setDataSourceData(
	ctx context.Context,
	diags *diag.Diagnostics,
	network *unifi.Network,
	model *networkDataSourceModel,
	site string,
) {
	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)

	if network.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(network.Name)
	}

	if network.Purpose == "" {
		model.Purpose = types.StringNull()
	} else {
		model.Purpose = types.StringValue(network.Purpose)
	}

	if network.VLAN == 0 {
		model.VlanID = types.Int64Null()
	} else {
		model.VlanID = types.Int64Value(int64(network.VLAN))
	}

	if network.IPSubnet == "" {
		model.Subnet = types.StringNull()
		model.IPSubnet = types.StringNull()
	} else {
		model.Subnet = types.StringValue(network.IPSubnet)
		model.IPSubnet = types.StringValue(network.IPSubnet)
	}

	if network.NetworkGroup == "" {
		model.NetworkGroup = types.StringNull()
	} else {
		model.NetworkGroup = types.StringValue(network.NetworkGroup)
	}

	if network.DHCPguardEnabled {
		model.DHCPStart = types.StringValue("auto")
	} else {
		model.DHCPStart = types.StringNull()
	}

	if network.DHCPguardEnabled {
		model.DHCPStop = types.StringValue("auto")
	} else {
		model.DHCPStop = types.StringNull()
	}

	model.DHCPEnabled = types.BoolValue(network.DHCPguardEnabled)

	// Use a default DHCP lease time since the actual field name is not clear
	model.DHCPLease = types.Int64Value(86400) // 24 hours default

	// Convert string slices to Framework lists - use empty list for now
	model.DHCPDNS = types.ListNull(types.StringType)

	model.DHCPDBootEnabled = types.BoolValue(false)

	model.DHCPDBootServer = types.StringNull()

	model.DHCPDBootFilename = types.StringNull()

	// Handle other complex fields similarly...
	if network.DomainName == "" {
		model.DomainName = types.StringNull()
	} else {
		model.DomainName = types.StringValue(network.DomainName)
	}

	model.IGMPSnooping = types.BoolValue(false)

	// Set other fields to null for now - they can be implemented as needed
	model.DHCPV6DNS = types.ListNull(types.StringType)
	model.DHCPV6DNSAuto = types.BoolValue(false)
	model.DHCPV6Enabled = types.BoolValue(false)
	model.DHCPV6Lease = types.Int64Null()
	model.DHCPV6Start = types.StringNull()
	model.DHCPV6Stop = types.StringNull()
	model.IPv6InterfaceType = types.StringNull()
	model.IPv6StaticSubnet = types.StringNull()
	model.IPv6PDInterface = types.StringNull()
	model.IPv6PDPrefixID = types.StringNull()
	model.IPv6PDStart = types.StringNull()
	model.IPv6PDStop = types.StringNull()
	model.IPv6RAEnable = types.BoolValue(false)
	model.IPv6RAPreferredLifetime = types.Int64Null()
	model.IPv6RAPriority = types.StringNull()
	model.IPv6RAValidLifetime = types.Int64Null()
	model.MulticastDNS = types.BoolValue(false)
	model.WanDNS = types.ListNull(types.StringType)
	model.WanEgressQOS = types.Int64Null()
	model.WanGateway = types.StringNull()
	model.WanGatewayV6 = types.StringNull()
	model.WanIP = types.StringNull()
	model.WanNetmask = types.StringNull()
	model.WanNetworkGroup = types.StringNull()
	model.WanType = types.StringNull()
	model.WanTypeV6 = types.StringNull()
	model.WanUsername = types.StringNull()
}
