package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
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
	IPv6                   types.Object `tfsdk:"ipv6"`
	LteLan                 types.Bool   `tfsdk:"lte_lan"`
	IPAliases              types.List   `tfsdk:"ip_aliases"`
	ThirdPartyGateway      types.Bool   `tfsdk:"third_party_gateway"`
	NatOutboundIPAddresses types.List   `tfsdk:"nat_outbound_ip_addresses"`
	DhcpGuarding           types.Object `tfsdk:"dhcp_guarding"`
	DhcpServer             types.Object `tfsdk:"dhcp_server"`
	DhcpRelay              types.Object `tfsdk:"dhcp_relay"`

	// Data-source-only informational fields
	Purpose      types.String `tfsdk:"purpose"`
	NetworkGroup types.String `tfsdk:"network_group"`

	// DHCPv6 server (DS-only)
	DhcpV6Server types.Object `tfsdk:"dhcp_v6_server"`

	// WAN settings (DS-only)
	Wan types.Object `tfsdk:"wan"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// networkDataSourceIPv6Model is the data source's `ipv6` nested object. It
// carries the subset of the resource's ipv6 group the data source exposes
// (no client_address_assignment, no pd.auto_prefixid_enabled); `ra` shares
// networkIPv6RAModel with the resource.
type networkDataSourceIPv6Model struct {
	InterfaceType types.String `tfsdk:"interface_type"`
	StaticSubnet  types.String `tfsdk:"static_subnet"`
	Aliases       types.List   `tfsdk:"aliases"`
	RA            types.Object `tfsdk:"ra"`
	PD            types.Object `tfsdk:"pd"`
}

func networkDataSourceIPv6AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interface_type": types.StringType,
		"static_subnet":  types.StringType,
		"aliases":        types.ListType{ElemType: types.StringType},
		"ra":             types.ObjectType{AttrTypes: networkIPv6RAAttrTypes()},
		"pd":             types.ObjectType{AttrTypes: networkDataSourceIPv6PDAttrTypes()},
	}
}

// networkDataSourceIPv6PDModel is the data source's `ipv6.pd` nested object.
type networkDataSourceIPv6PDModel struct {
	Interface types.String `tfsdk:"interface"`
	Prefixid  types.String `tfsdk:"prefixid"`
	Start     types.String `tfsdk:"start"`
	Stop      types.String `tfsdk:"stop"`
}

func networkDataSourceIPv6PDAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interface": types.StringType,
		"prefixid":  types.StringType,
		"start":     types.StringType,
		"stop":      types.StringType,
	}
}

// networkDataSourceWanModel is the data source's `wan` nested object.
type networkDataSourceWanModel struct {
	DNS          types.List   `tfsdk:"dns"`
	EgressQOS    types.Int64  `tfsdk:"egress_qos"`
	Gateway      types.String `tfsdk:"gateway"`
	GatewayV6    types.String `tfsdk:"gateway_v6"`
	IP           types.String `tfsdk:"ip"`
	Netmask      types.String `tfsdk:"netmask"`
	NetworkGroup types.String `tfsdk:"network_group"`
	Type         types.String `tfsdk:"type"`
	TypeV6       types.String `tfsdk:"type_v6"`
	Username     types.String `tfsdk:"username"`
}

func networkDataSourceWanAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"dns":           types.ListType{ElemType: types.StringType},
		"egress_qos":    types.Int64Type,
		"gateway":       types.StringType,
		"gateway_v6":    types.StringType,
		"ip":            types.StringType,
		"netmask":       types.StringType,
		"network_group": types.StringType,
		"type":          types.StringType,
		"type_v6":       types.StringType,
		"username":      types.StringType,
	}
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
			"ipv6": schema.SingleNestedAttribute{
				MarkdownDescription: "IPv6 settings of the network.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"interface_type": schema.StringAttribute{
						MarkdownDescription: "Specifies which type of IPv6 connection to use.",
						Computed:            true,
					},
					"static_subnet": schema.StringAttribute{
						MarkdownDescription: "The static IPv6 subnet (when `interface_type` is `static`).",
						Computed:            true,
					},
					"aliases": schema.ListAttribute{
						MarkdownDescription: "List of IPv6 aliases for the network.",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"ra": schema.SingleNestedAttribute{
						MarkdownDescription: "IPv6 Router Advertisement (RA) settings.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether to enable IPv6 router advertisements.",
								Computed:            true,
							},
							"priority": schema.StringAttribute{
								MarkdownDescription: "IPv6 router advertisement priority. One of `high`, `medium`, or `low`.",
								Computed:            true,
							},
							"preferred_lifetime": schema.StringAttribute{
								MarkdownDescription: "Preferred lifetime for IPv6 RA, as a Go duration string.",
								CustomType:          timetypes.GoDurationType{},
								Computed:            true,
							},
							"valid_lifetime": schema.StringAttribute{
								MarkdownDescription: "Total lifetime for the IPv6 RA address, as a Go duration string.",
								CustomType:          timetypes.GoDurationType{},
								Computed:            true,
							},
						},
					},
					"pd": schema.SingleNestedAttribute{
						MarkdownDescription: "IPv6 Prefix Delegation (PD) settings.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"interface": schema.StringAttribute{
								MarkdownDescription: "Specifies which WAN interface to use for IPv6 PD. One of `wan` or `wan2`.",
								Computed:            true,
							},
							"prefixid": schema.StringAttribute{
								MarkdownDescription: "Specifies the IPv6 Prefix ID.",
								Computed:            true,
							},
							"start": schema.StringAttribute{
								MarkdownDescription: "Start address of the DHCPv6 range when `ipv6.interface_type` is `pd`.",
								Computed:            true,
							},
							"stop": schema.StringAttribute{
								MarkdownDescription: "End address of the DHCPv6 range when `ipv6.interface_type` is `pd`.",
								Computed:            true,
							},
						},
					},
				},
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
					"ntp": schema.SingleNestedAttribute{
						MarkdownDescription: "NTP servers handed out to DHCP clients.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP NTP is enabled.",
								Computed:            true,
							},
							"servers": schema.ListAttribute{
								MarkdownDescription: "List of NTP server addresses for DHCP clients.",
								Computed:            true,
								ElementType:         types.StringType,
							},
						},
					},
					"time_offset_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP time offset is enabled.",
						Computed:            true,
					},
					"dns": schema.SingleNestedAttribute{
						MarkdownDescription: "DNS servers handed out to DHCP clients.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP DNS is enabled.",
								Computed:            true,
							},
							"servers": schema.ListAttribute{
								MarkdownDescription: "List of DNS server addresses for DHCP clients.",
								Computed:            true,
								ElementType:         types.StringType,
							},
						},
					},
					"leasetime": schema.StringAttribute{
						MarkdownDescription: "Specifies the DHCP lease time, as a Go duration string.",
						CustomType:          timetypes.GoDurationType{},
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
			"dhcp_v6_server": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCPv6 server configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether stateful DHCPv6 is enabled.",
						Computed:            true,
					},
					"dns": schema.SingleNestedAttribute{
						MarkdownDescription: "DNS servers handed out to DHCPv6 clients.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"auto": schema.BoolAttribute{
								MarkdownDescription: "When true, upstream DNS entries are propagated. When false, `servers` are used.",
								Computed:            true,
							},
							"servers": schema.ListAttribute{
								MarkdownDescription: "IPv6 DNS server addresses for DHCPv6 clients.",
								Computed:            true,
								ElementType:         types.StringType,
							},
						},
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
			"wan": schema.SingleNestedAttribute{
				MarkdownDescription: "WAN settings, populated for networks with the `wan` purpose.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"dns": schema.ListAttribute{
						MarkdownDescription: "DNS server IPs of the WAN.",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"egress_qos": schema.Int64Attribute{
						MarkdownDescription: "Specifies the WAN egress quality of service.",
						Computed:            true,
					},
					"gateway": schema.StringAttribute{
						MarkdownDescription: "The IPv4 gateway of the WAN.",
						Computed:            true,
					},
					"gateway_v6": schema.StringAttribute{
						MarkdownDescription: "The IPv6 gateway of the WAN.",
						Computed:            true,
					},
					"ip": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address of the WAN.",
						Computed:            true,
					},
					"netmask": schema.StringAttribute{
						MarkdownDescription: "The IPv4 netmask of the WAN.",
						Computed:            true,
					},
					"network_group": schema.StringAttribute{
						MarkdownDescription: "Specifies the WAN network group. One of `WAN`, `WAN2` or `WAN_LTE_FAILOVER`.",
						Computed:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "Specifies the IPv4 WAN connection type. One of `disabled`, `static`, `dhcp`, or `pppoe`.",
						Computed:            true,
					},
					"type_v6": schema.StringAttribute{
						MarkdownDescription: "Specifies the IPv6 WAN connection type. One of `disabled`, `static`, or `dhcpv6`.",
						Computed:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "Specifies the IPv4 WAN username.",
						Computed:            true,
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx),
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

	readTimeout, timeoutDiags := config.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

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

	// ipv6 (aliases is not available in the current API and stays null)
	{
		raObj, d := types.ObjectValueFrom(ctx, networkIPv6RAAttrTypes(), networkIPv6RAModel{
			Enabled:  types.BoolValue(network.IPV6RaEnabled),
			Priority: types.StringPointerValue(network.IPV6RaPriority),
			PreferredLifetime: util.DurationPtrValue(
				network.IPV6RaPreferredLifetime,
				time.Second,
			),
			ValidLifetime: util.DurationPtrValue(network.IPV6RaValidLifetime, time.Second),
		})
		diags.Append(d...)
		pdObj, d := types.ObjectValueFrom(
			ctx,
			networkDataSourceIPv6PDAttrTypes(),
			networkDataSourceIPv6PDModel{
				Interface: types.StringPointerValue(network.IPV6PDInterface),
				Prefixid:  stringOrNull(network.IPV6PDPrefixid),
				Start:     types.StringPointerValue(network.IPV6PDStart),
				Stop:      types.StringPointerValue(network.IPV6PDStop),
			},
		)
		diags.Append(d...)
		ipv6Obj, d := types.ObjectValueFrom(
			ctx,
			networkDataSourceIPv6AttrTypes(),
			networkDataSourceIPv6Model{
				InterfaceType: types.StringPointerValue(network.IPV6InterfaceType),
				StaticSubnet:  types.StringPointerValue(network.IPV6Subnet),
				Aliases:       types.ListNull(types.StringType),
				RA:            raObj,
				PD:            pdObj,
			},
		)
		diags.Append(d...)
		model.IPv6 = ipv6Obj
	}

	// nat_outbound_ip_addresses — not populated by API read
	model.NatOutboundIPAddresses = types.ListNull(
		types.ObjectType{AttrTypes: natOutboundIPAddresses()},
	)

	// dhcp_guarding
	{
		var serversList types.List
		servers := collectNonEmptyStrings(network.DHCPDIP1, network.DHCPDIP2, network.DHCPDIP3)
		if len(servers) > 0 {
			l, d := types.ListValueFrom(ctx, types.StringType, servers)
			diags.Append(d...)
			serversList = l
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

		ntpServers := collectNonEmptyStringPointers(network.DHCPDNtp1, network.DHCPDNtp2)
		var ntpServersList types.List
		if len(ntpServers) > 0 {
			ntpServersList, d = types.ListValueFrom(ctx, types.StringType, ntpServers)
			diags.Append(d...)
		} else {
			ntpServersList = types.ListNull(types.StringType)
		}

		dnsObj, d := types.ObjectValueFrom(
			ctx,
			dhcpServerOptionModel{}.AttributeTypes(),
			dhcpServerOptionModel{
				Enabled: types.BoolValue(network.DHCPDDNSEnabled),
				Servers: dnsServersList,
			},
		)
		diags.Append(d...)
		ntpObj, d := types.ObjectValueFrom(
			ctx,
			dhcpServerOptionModel{}.AttributeTypes(),
			dhcpServerOptionModel{
				Enabled: types.BoolValue(network.DHCPDNtpEnabled),
				Servers: ntpServersList,
			},
		)
		diags.Append(d...)

		dhcpServerValue := dhcpServerModel{
			Boot:              dhcpBootObj,
			Enabled:           types.BoolValue(network.DHCPDEnabled),
			Start:             types.StringPointerValue(network.DHCPDStart),
			Stop:              types.StringPointerValue(network.DHCPDStop),
			GatewayEnabled:    types.BoolValue(network.DHCPDGatewayEnabled),
			ConflictChecking:  types.BoolValue(network.DHCPDConflictChecking),
			Ntp:               ntpObj,
			TimeOffsetEnabled: types.BoolValue(network.DHCPDTimeOffsetEnabled),
			Dns:               dnsObj,
			Leasetime:         util.DurationPtrValue(network.DHCPDLeaseTime, time.Second),
			Wins:              winsObj,
			WpadUrl:           strPtrToType(network.DHCPDWPAdUrl),
			TftpServer:        strPtrToType(network.DHCPDTFTPServer),
			UnifiController:   strPtrToType(network.DHCPDUnifiController),
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
		var relayServersVal types.List
		if len(network.DHCPRelayServers) > 0 {
			l, d := types.ListValueFrom(ctx, types.StringType, network.DHCPRelayServers)
			diags.Append(d...)
			relayServersVal = l
		} else {
			relayServersVal = types.ListNull(types.StringType)
		}
		dhcpRelayValue := dhcpRelayModel{
			Enabled: types.BoolValue(network.DHCPRelayEnabled),
			Servers: relayServersVal,
		}
		dhcpRelayObj, d := types.ObjectValueFrom(
			ctx,
			dhcpRelayValue.AttributeTypes(),
			dhcpRelayValue,
		)
		diags.Append(d...)
		model.DhcpRelay = dhcpRelayObj
	}

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

		dhcpv6DNSObj, d := types.ObjectValueFrom(
			ctx,
			dhcpV6DNSModel{}.AttributeTypes(),
			dhcpV6DNSModel{
				Auto:    types.BoolValue(network.DHCPDV6DNSAuto),
				Servers: dhcpv6DNSList,
			},
		)
		diags.Append(d...)

		dhcpV6ServerValue := dhcpV6ServerModel{
			Enabled: types.BoolValue(network.DHCPDV6Enabled),
			DNS:     dhcpv6DNSObj,
			Lease:   types.Int64PointerValue(network.DHCPDV6LeaseTime),
			Start:   types.StringPointerValue(network.DHCPDV6Start),
			Stop:    types.StringPointerValue(network.DHCPDV6Stop),
		}
		dhcpV6ServerObj, d := types.ObjectValueFrom(
			ctx,
			dhcpV6ServerValue.AttributeTypes(),
			dhcpV6ServerValue,
		)
		diags.Append(d...)
		model.DhcpV6Server = dhcpV6ServerObj
	}

	// wan
	{
		wanDNS := collectNonEmptyStringPointers(network.WANDNS1, network.WANDNS2)
		wanDNS = append(wanDNS, collectNonEmptyStrings(network.WANDNS3, network.WANDNS4)...)
		var wanDNSList types.List
		if len(wanDNS) > 0 {
			l, d := types.ListValueFrom(ctx, types.StringType, wanDNS)
			diags.Append(d...)
			wanDNSList = l
		} else {
			wanDNSList = types.ListNull(types.StringType)
		}

		wanObj, d := types.ObjectValueFrom(
			ctx,
			networkDataSourceWanAttrTypes(),
			networkDataSourceWanModel{
				DNS:          wanDNSList,
				EgressQOS:    types.Int64PointerValue(network.WANEgressQOS),
				Gateway:      types.StringPointerValue(network.WANGateway),
				GatewayV6:    stringOrNull(network.WANGatewayV6),
				IP:           types.StringPointerValue(network.WANIP),
				Netmask:      types.StringPointerValue(network.WANNetmask),
				NetworkGroup: types.StringPointerValue(network.WANNetworkGroup),
				Type:         types.StringPointerValue(network.WANType),
				TypeV6:       types.StringPointerValue(network.WANTypeV6),
				Username:     stringOrNull(network.WANUsername),
			},
		)
		diags.Append(d...)
		model.Wan = wanObj
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
