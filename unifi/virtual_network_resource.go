package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &virtualNetworkResource{}
	_ resource.ResourceWithImportState = &virtualNetworkResource{}
)

func NewVirtualNetworkResource() resource.Resource {
	return &virtualNetworkResource{}
}

// virtualNetworkResource defines the resource implementation.
type virtualNetworkResource struct {
	client *Client
}

// dhcpBootModel describes the DHCP boot configuration.
type dhcpBootModel struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	Server   types.String `tfsdk:"server"`
	Filename types.String `tfsdk:"filename"`
}

func (m dhcpBootModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":  types.BoolType,
		"server":   types.StringType,
		"filename": types.StringType,
	}
}

// winsModel describes the WINS configuration.
type winsModel struct {
	Enabled   types.Bool `tfsdk:"enabled"`
	Addresses types.List `tfsdk:"addresses"`
}

func (m winsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":   types.BoolType,
		"addresses": types.ListType{ElemType: types.StringType},
	}
}

// dhcpServerModel describes the DHCP server configuration.
type dhcpServerModel struct {
	Boot              types.Object `tfsdk:"boot"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Start             types.String `tfsdk:"start"`
	Stop              types.String `tfsdk:"stop"`
	GatewayEnabled    types.Bool   `tfsdk:"gateway_enabled"`
	ConflictChecking  types.Bool   `tfsdk:"conflict_checking"`
	NtpEnabled        types.Bool   `tfsdk:"ntp_enabled"`
	TimeOffsetEnabled types.Bool   `tfsdk:"time_offset_enabled"`
	DnsEnabled        types.Bool   `tfsdk:"dns_enabled"`
	Leasetime         types.Int64  `tfsdk:"leasetime"`
	Wins              types.Object `tfsdk:"wins"`
	WpadUrl           types.String `tfsdk:"wpad_url"`
	TftpServer        types.String `tfsdk:"tftp_server"`
	UnifiController   types.String `tfsdk:"unifi_controller"`
	DnsServers        types.List   `tfsdk:"dns_servers"`
}

func (m dhcpServerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"boot":                types.ObjectType{AttrTypes: dhcpBootModel{}.AttributeTypes()},
		"enabled":             types.BoolType,
		"start":               types.StringType,
		"stop":                types.StringType,
		"gateway_enabled":     types.BoolType,
		"conflict_checking":   types.BoolType,
		"ntp_enabled":         types.BoolType,
		"time_offset_enabled": types.BoolType,
		"dns_enabled":         types.BoolType,
		"leasetime":           types.Int64Type,
		"wins":                types.ObjectType{AttrTypes: winsModel{}.AttributeTypes()},
		"wpad_url":            types.StringType,
		"tftp_server":         types.StringType,
		"unifi_controller":    types.StringType,
		"dns_servers":         types.ListType{ElemType: types.StringType},
	}
}

type natOutboundIPAddressesModel struct {
	IPAddress       types.String `tfsdk:"ip_address"`                  // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	IPAddressPool   types.List   `tfsdk:"ip_address_pool,omitempty"`   // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])-(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$
	Mode            types.String `tfsdk:"mode,omitempty"`              // all|ip_address|ip_address_pool
	WANNetworkGroup types.String `tfsdk:"wan_network_group,omitempty"` // WAN[2-9]?
}

func (d natOutboundIPAddressesModel) AttributeTypes() map[string]attr.Type {
	return natOutboundIPAddresses()
}

func natOutboundIPAddresses() map[string]attr.Type {
	return map[string]attr.Type{
		"ip_address":        types.StringType,
		"ip_address_pool":   types.ListType{ElemType: types.StringType},
		"mode":              types.StringType,
		"wan_network_group": types.StringType,
	}
}

// dhcpRelayModel describes the DHCP relay configuration.
type dhcpRelayModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Servers types.List `tfsdk:"servers"`
}

func (d dhcpRelayModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// virtualNetworkResourceModel describes the resource data model.
type virtualNetworkResourceModel struct {
	ID                      types.String         `tfsdk:"id"`
	Site                    types.String         `tfsdk:"site"`
	Enabled                 types.Bool           `tfsdk:"enabled"`
	Name                    types.String         `tfsdk:"name"`
	NatOutboundIPAddresses  types.List           `tfsdk:"nat_outbound_ip_addresses"`
	AutoScaleEnabled        types.Bool           `tfsdk:"auto_scale_enabled"`
	Subnet                  cidrtypes.IPv4Prefix `tfsdk:"subnet"`
	DomainName              types.String         `tfsdk:"domain_name"`
	Vlan                    types.Int64          `tfsdk:"vlan"`
	NetworkGroup            types.String         `tfsdk:"network_group"`
	NetworkIsolationEnabled types.Bool           `tfsdk:"network_isolation_enabled"`
	SettingPreference       types.String         `tfsdk:"setting_preference"`
	InternetAccessEnabled   types.Bool           `tfsdk:"internet_access_enabled"`
	IgmpSnooping            types.Bool           `tfsdk:"igmp_snooping"`
	MdnsEnabled             types.Bool           `tfsdk:"mdns_enabled"`
	GatewayType             types.String         `tfsdk:"gateway_type"`
	IPv6InterfaceType       types.String         `tfsdk:"ipv6_interface_type"`
	LteLanEnabled           types.Bool           `tfsdk:"lte_lan_enabled"`
	IPAliases               types.List           `tfsdk:"ip_aliases"`
	IPv6Aliases             types.List           `tfsdk:"ipv6_aliases"`
	DhcpguardEnabled        types.Bool           `tfsdk:"dhcpguard_enabled"`
	VlanEnabled             types.Bool           `tfsdk:"vlan_enabled"`
	DhcpServer              types.Object         `tfsdk:"dhcp_server"`
	DhcpRelay               types.Object         `tfsdk:"dhcp_relay"`
}

func (r *virtualNetworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_virtual_network"
}

func (r *virtualNetworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_virtual_network` manages virtual networks (VLANs) in the UniFi controller.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the virtual network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the virtual network with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the virtual network is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the virtual network.",
				Required:            true,
			},
			"nat_outbound_ip_addresses": schema.ListNestedAttribute{
				MarkdownDescription: "List of NAT outbound IP addresses.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The IP address.",
							Optional:            true,
						},
						"ip_address_pool": schema.ListAttribute{
							MarkdownDescription: "The IP address pool.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"mode": schema.StringAttribute{
							MarkdownDescription: "The mode.",
							Optional:            true,
						},
						"wan_network_group": schema.StringAttribute{
							MarkdownDescription: "The WAN network group.",
							Optional:            true,
						},
					},
				},
			},
			"auto_scale_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether auto-scaling is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The IP subnet of the network in CIDR notation.",
				Required:            true,
				CustomType:          cidrtypes.IPv4PrefixType{},
			},
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "The domain name for the network.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					validators.DomainNameValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "The VLAN ID for the network.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 4094),
				},
			},
			"network_group": schema.StringAttribute{
				MarkdownDescription: "The network group. Defaults to 'LAN'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("LAN"),
			},
			"network_isolation_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether network isolation is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"setting_preference": schema.StringAttribute{
				MarkdownDescription: "Setting preference. Must be one of `auto` or `manual`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("auto"),
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"internet_access_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether internet access is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"igmp_snooping": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether IGMP snooping is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"mdns_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether mDNS is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"gateway_type": schema.StringAttribute{
				MarkdownDescription: "The gateway type. Must be one of `default` or `switch`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				Validators: []validator.String{
					stringvalidator.OneOf("default", "switch"),
				},
			},
			"ipv6_interface_type": schema.StringAttribute{
				MarkdownDescription: "Specifies which type of IPv6 connection to use. Must be one of `none`, `pd`, or `static`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("none"),
				Validators: []validator.String{
					stringvalidator.OneOf("none", "pd", "static"),
				},
			},
			"lte_lan_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether LTE LAN is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"ip_aliases": schema.ListAttribute{
				MarkdownDescription: "List of IP aliases for the network.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"ipv6_aliases": schema.ListAttribute{
				MarkdownDescription: "List of IPv6 aliases for the network.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"dhcpguard_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether DHCP guard is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"vlan_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether VLAN is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"dhcp_server": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP server configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"boot": schema.SingleNestedAttribute{
						MarkdownDescription: "DHCP boot settings.",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Toggles DHCP boot options.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"server": schema.StringAttribute{
								MarkdownDescription: "TFTP server for boot options.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"filename": schema.StringAttribute{
								MarkdownDescription: "Boot filename.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP server is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"start": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address where the DHCP range starts.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"stop": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address where the DHCP range stops.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"gateway_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP gateway is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"conflict_checking": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP conflict checking is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"ntp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP NTP is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"time_offset_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP time offset is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"dns_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP DNS is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"leasetime": schema.Int64Attribute{
						MarkdownDescription: "Specifies the lease time for DHCP addresses in seconds.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(86400),
					},
					"wins": schema.SingleNestedAttribute{
						MarkdownDescription: "WINS server configuration.",
						Optional:            true,
						Computed:            true,
						Default: objectdefault.StaticValue(
							types.ObjectValueMust(map[string]attr.Type{
								"enabled":   types.BoolType,
								"addresses": types.ListType{ElemType: types.StringType},
							}, map[string]attr.Value{
								"enabled":   types.BoolValue(false),
								"addresses": types.ListNull(types.StringType),
							}),
						),
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP WINS is enabled.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"addresses": schema.ListAttribute{
								MarkdownDescription: "List of WINS server addresses (maximum 2).",
								Optional:            true,
								ElementType:         types.StringType,
								Validators: []validator.List{
									listvalidator.SizeAtMost(2),
								},
							},
						},
					},
					"wpad_url": schema.StringAttribute{
						MarkdownDescription: "WPAD URL for proxy auto-configuration.",
						Optional:            true,
					},
					"tftp_server": schema.StringAttribute{
						MarkdownDescription: "TFTP server address.",
						Optional:            true,
					},
					"unifi_controller": schema.StringAttribute{
						MarkdownDescription: "UniFi controller IP address.",
						Optional:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
					},
					"dns_servers": schema.ListAttribute{
						MarkdownDescription: "List of DNS server addresses for DHCP clients.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtMost(4),
						},
					},
				},
			},
			"dhcp_relay": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP relay configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP relay is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "List of DHCP relay server addresses.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtMost(4),
						},
					},
				},
			},
		},
	}
}

func (r *virtualNetworkResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.client = client
}

func (r *virtualNetworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data virtualNetworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.Network
	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the network
	createdNetwork, err := r.client.CreateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Virtual Network",
			err.Error(),
		)
		return
	}

	// Convert back to model, passing the plan data to preserve null values
	var planData virtualNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, createdNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualNetworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data virtualNetworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	var err error
	var network *unifi.Network

	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		// Get the network by ID
		network, err = r.client.GetNetwork(ctx, site, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Virtual Network",
				"Could not read virtual network ID "+data.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else {
		// Get the network by name
		network, err = r.client.GetNetworkByName(ctx, site, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Virtual Network",
				"Could not read virtual network name "+data.Name.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Convert to model, passing the current state to preserve null values
	var priorState virtualNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.networkToModel(ctx, network, &data, site, &priorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualNetworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data virtualNetworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.Network
	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	network.ID = data.ID.ValueString()

	// Update the network
	updatedNetwork, err := r.client.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Virtual Network",
			err.Error(),
		)
		return
	}

	// Convert back to model, passing the plan data to preserve null values
	var planData virtualNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, updatedNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualNetworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data virtualNetworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the network
	name := data.Name.ValueString()
	err := r.client.DeleteNetwork(ctx, site, data.ID.ValueString(), name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Virtual Network",
			err.Error(),
		)
		return
	}
}

func (r *virtualNetworkResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		req.ID = idParts[1]
	}

	rootAttributeName := "name"
	if strings.HasPrefix(req.ID, "name=") {
		req.ID = strings.TrimPrefix(req.ID, "name=")
	} else if regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(req.ID) {
		rootAttributeName = "id"
	}

	resource.ImportStatePassthroughID(ctx, path.Root(rootAttributeName), req, resp)
}

// modelToNetwork converts from Terraform model to unifi.Network.
func (r *virtualNetworkResource) modelToNetwork(
	ctx context.Context,
	model *virtualNetworkResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:                    model.Name.ValueStringPointer(),
		Purpose:                 unifi.PurposeCorporate,
		NetworkGroup:            util.Ptr("LAN"),
		AutoScaleEnabled:        model.AutoScaleEnabled.ValueBool(),
		IPSubnet:                model.Subnet.ValueStringPointer(),
		NetworkIsolationEnabled: model.NetworkIsolationEnabled.ValueBool(),
		SettingPreference:       model.SettingPreference.ValueStringPointer(),
		InternetAccessEnabled:   model.InternetAccessEnabled.ValueBool(),
		MdnsEnabled:             model.MdnsEnabled.ValueBool(),
		GatewayType:             model.GatewayType.ValueStringPointer(),
		IPV6InterfaceType:       model.IPv6InterfaceType.ValueStringPointer(),
		LteLanEnabled:           model.LteLanEnabled.ValueBool(),
		VLANEnabled:             model.VlanEnabled.ValueBool(),
		Enabled:                 model.Enabled.ValueBool(),
		IGMPSnooping:            model.IgmpSnooping.ValueBool(),
		DHCPguardEnabled:        model.DhcpguardEnabled.ValueBool(),
		IPAliases:               []string{},
	}

	// Handle domain name - set to empty string if null
	network.DomainName = model.DomainName.ValueStringPointer()

	// Handle optional int64 pointer fields
	network.VLAN = model.Vlan.ValueInt64Pointer()

	// Handle NAT outbound IP addresses
	if !model.NatOutboundIPAddresses.IsNull() && !model.NatOutboundIPAddresses.IsUnknown() {
		var natIPs []natOutboundIPAddressesModel
		d := model.NatOutboundIPAddresses.ElementsAs(ctx, &natIPs, true)
		diags.Append(d...)
		if !diags.HasError() {
			for _, natIP := range natIPs {
				v := unifi.NetworkNATOutboundIPAddresses{
					IPAddress:       natIP.IPAddress.ValueString(),
					Mode:            natIP.Mode.ValueStringPointer(),
					WANNetworkGroup: natIP.WANNetworkGroup.ValueStringPointer(),
				}
				network.NATOutboundIPAddresses = append(network.NATOutboundIPAddresses, v)
			}
		}
	}

	// Handle IP aliases
	if !model.IPAliases.IsNull() && !model.IPAliases.IsUnknown() {
		var ipAliases []string
		d := model.IPAliases.ElementsAs(ctx, &ipAliases, false)
		diags.Append(d...)
		if !diags.HasError() {
			network.IPAliases = ipAliases
		}
	}

	// Handle IPv6 aliases
	if !model.IPv6Aliases.IsNull() && !model.IPv6Aliases.IsUnknown() {
		var ipv6Aliases []string
		d := model.IPv6Aliases.ElementsAs(ctx, &ipv6Aliases, false)
		diags.Append(d...)
		if !diags.HasError() {
			// IPv6Aliases field not available in API
		}
	}

	// Handle DHCP server configuration
	if !model.DhcpServer.IsNull() && !model.DhcpServer.IsUnknown() {
		var dhcpServer dhcpServerModel
		d := model.DhcpServer.As(ctx, &dhcpServer, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			// Handle DHCP boot configuration
			if !dhcpServer.Boot.IsNull() && !dhcpServer.Boot.IsUnknown() {
				var dhcpBoot dhcpBootModel
				d := dhcpServer.Boot.As(ctx, &dhcpBoot, basetypes.ObjectAsOptions{})
				diags.Append(d...)
				if !diags.HasError() {
					network.DHCPDBootEnabled = dhcpBoot.Enabled.ValueBool()
					if dhcpBoot.Server.IsNull() || dhcpBoot.Server.IsUnknown() {
						network.DHCPDBootServer = ""
					} else {
						network.DHCPDBootServer = dhcpBoot.Server.ValueString()
					}
					if dhcpBoot.Filename.IsNull() || dhcpBoot.Filename.IsUnknown() {
						network.DHCPDBootFilename = util.Ptr("")
					} else {
						network.DHCPDBootFilename = dhcpBoot.Filename.ValueStringPointer()
					}
				}
			} else {
				network.DHCPDBootEnabled = false
				network.DHCPDBootServer = ""
				network.DHCPDBootFilename = util.Ptr("")
			}
			network.DHCPDEnabled = dhcpServer.Enabled.ValueBool()
			network.DHCPDStart = dhcpServer.Start.ValueStringPointer()
			network.DHCPDStop = dhcpServer.Stop.ValueStringPointer()
			network.DHCPDGatewayEnabled = dhcpServer.GatewayEnabled.ValueBool()
			network.DHCPDConflictChecking = dhcpServer.ConflictChecking.ValueBool()
			network.DHCPDNtpEnabled = dhcpServer.NtpEnabled.ValueBool()
			network.DHCPDTimeOffsetEnabled = dhcpServer.TimeOffsetEnabled.ValueBool()
			network.DHCPDDNSEnabled = dhcpServer.DnsEnabled.ValueBool()
			network.DHCPDLeaseTime = dhcpServer.Leasetime.ValueInt64Pointer()

			// Handle WINS configuration
			if !dhcpServer.Wins.IsNull() && !dhcpServer.Wins.IsUnknown() {
				var wins winsModel
				d := dhcpServer.Wins.As(ctx, &wins, basetypes.ObjectAsOptions{})
				diags.Append(d...)
				if !diags.HasError() {
					network.DHCPDWinsEnabled = wins.Enabled.ValueBool()
					if !wins.Addresses.IsNull() && !wins.Addresses.IsUnknown() {
						var addresses []string
						d := wins.Addresses.ElementsAs(ctx, &addresses, false)
						diags.Append(d...)
						if !diags.HasError() {
							for i, addr := range addresses {
								if i >= 2 {
									break
								}
								switch i {
								case 0:
									network.DHCPDWins1 = util.Ptr(addr)
								case 1:
									network.DHCPDWins2 = util.Ptr(addr)
								}
							}
							// Set remaining WINS servers to empty string
							for i := len(addresses); i < 2; i++ {
								switch i {
								case 0:
									network.DHCPDWins1 = util.Ptr("")
								case 1:
									network.DHCPDWins2 = util.Ptr("")
								}
							}
						}
					} else {
						network.DHCPDWins1 = util.Ptr("")
						network.DHCPDWins2 = util.Ptr("")
					}
				}
			} else {
				network.DHCPDWinsEnabled = false
				network.DHCPDWins1 = util.Ptr("")
				network.DHCPDWins2 = util.Ptr("")
			}

			if dhcpServer.WpadUrl.IsNull() || dhcpServer.WpadUrl.IsUnknown() {
				network.DHCPDWPAdUrl = util.Ptr("")
			} else {
				network.DHCPDWPAdUrl = dhcpServer.WpadUrl.ValueStringPointer()
			}

			if dhcpServer.TftpServer.IsNull() || dhcpServer.TftpServer.IsUnknown() {
				network.DHCPDTFTPServer = util.Ptr("")
			} else {
				network.DHCPDTFTPServer = dhcpServer.TftpServer.ValueStringPointer()
			}

			if dhcpServer.UnifiController.IsNull() || dhcpServer.UnifiController.IsUnknown() {
				network.DHCPDUnifiController = util.Ptr("")
			} else {
				network.DHCPDUnifiController = dhcpServer.UnifiController.ValueStringPointer()
			}

			// Handle DNS servers
			if !dhcpServer.DnsServers.IsNull() && !dhcpServer.DnsServers.IsUnknown() {
				var dnsServers []string
				d := dhcpServer.DnsServers.ElementsAs(ctx, &dnsServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					for i, dns := range dnsServers {
						if i >= 4 {
							break
						}
						switch i {
						case 0:
							network.DHCPDDNS1 = dns
						case 1:
							network.DHCPDDNS2 = dns
						case 2:
							network.DHCPDDNS3 = dns
						case 3:
							network.DHCPDDNS4 = dns
						}
					}
					// Set remaining DNS servers to empty string
					for i := len(dnsServers); i < 4; i++ {
						switch i {
						case 0:
							network.DHCPDDNS1 = ""
						case 1:
							network.DHCPDDNS2 = ""
						case 2:
							network.DHCPDDNS3 = ""
						case 3:
							network.DHCPDDNS4 = ""
						}
					}
				}
			} else {
				// Set all DNS servers to empty string when not configured
				network.DHCPDDNS1 = ""
				network.DHCPDDNS2 = ""
				network.DHCPDDNS3 = ""
				network.DHCPDDNS4 = ""
			}
		}
	} else {
		// Set defaults when DHCP server is not configured
		network.DHCPDBootEnabled = false
		network.DHCPDBootServer = ""
		network.DHCPDBootFilename = util.Ptr("")
		network.DHCPDEnabled = true
		network.DHCPDGatewayEnabled = false
		network.DHCPDConflictChecking = true
		network.DHCPDNtpEnabled = false
		network.DHCPDTimeOffsetEnabled = false
		network.DHCPDDNSEnabled = false
		network.DHCPDLeaseTime = util.Ptr(int64(86400))
		network.DHCPDWinsEnabled = false
		network.DHCPDWins1 = util.Ptr("")
		network.DHCPDWins2 = util.Ptr("")
		network.DHCPDWPAdUrl = util.Ptr("")
		network.DHCPDTFTPServer = util.Ptr("")
		network.DHCPDUnifiController = util.Ptr("")
		network.DHCPDDNS1 = ""
		network.DHCPDDNS2 = ""
		network.DHCPDDNS3 = ""
		network.DHCPDDNS4 = ""
	}

	// Handle DHCP relay configuration
	if !model.DhcpRelay.IsNull() && !model.DhcpRelay.IsUnknown() {
		var dhcpRelay dhcpRelayModel
		d := model.DhcpRelay.As(ctx, &dhcpRelay, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.DHCPRelayEnabled = dhcpRelay.Enabled.ValueBool()
			if !dhcpRelay.Servers.IsNull() && !dhcpRelay.Servers.IsUnknown() {
				var servers []string
				d := dhcpRelay.Servers.ElementsAs(ctx, &servers, false)
				diags.Append(d...)
				if !diags.HasError() {
					// DHCPRelayServers field not available in current API version
				}
			}
		}
	} else {
		// Set defaults when DHCP relay is not configured
		network.DHCPRelayEnabled = false
	}

	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model.
// previousModel is the model from the plan or previous state, used to preserve null values.
func (r *virtualNetworkResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *virtualNetworkResourceModel,
	site string,
	previousModel *virtualNetworkResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringPointerValue(network.Name)
	model.AutoScaleEnabled = types.BoolValue(network.AutoScaleEnabled)
	if network.IPSubnet != nil {
		model.Subnet = cidrtypes.NewIPv4PrefixValue(*network.IPSubnet)
	} else {
		model.Subnet = cidrtypes.NewIPv4PrefixNull()
	}
	model.NetworkGroup = types.StringPointerValue(network.NetworkGroup)
	model.NetworkIsolationEnabled = types.BoolValue(network.NetworkIsolationEnabled)
	model.SettingPreference = types.StringPointerValue(network.SettingPreference)
	model.InternetAccessEnabled = types.BoolValue(network.InternetAccessEnabled)
	model.MdnsEnabled = types.BoolValue(network.MdnsEnabled)
	model.GatewayType = types.StringPointerValue(network.GatewayType)
	model.IPv6InterfaceType = types.StringPointerValue(network.IPV6InterfaceType)
	model.LteLanEnabled = types.BoolValue(network.LteLanEnabled)
	model.VlanEnabled = types.BoolValue(network.VLANEnabled)
	model.Enabled = types.BoolValue(network.Enabled)
	model.IgmpSnooping = types.BoolValue(network.IGMPSnooping)
	model.DhcpguardEnabled = types.BoolValue(network.DHCPguardEnabled)

	// Handle optional fields
	model.DomainName = types.StringPointerValue(network.DomainName)

	model.Vlan = types.Int64PointerValue(network.VLAN)

	// Handle lists - for now set to null
	model.NatOutboundIPAddresses = types.ListNull(
		types.ObjectType{AttrTypes: natOutboundIPAddresses()},
	)
	model.IPAliases = types.ListNull(types.StringType)
	model.IPv6Aliases = types.ListNull(types.StringType)

	// Determine if this is an import (name/ID is set but required field Subnet is null)
	isImport := previousModel != nil && previousModel.Subnet.IsNull()

	// Only populate dhcp_server if:
	// 1. It was configured in the previous state (not null), OR
	// 2. This is an import and DHCP is enabled (populate everything during import)
	shouldPopulateDhcp := false
	if previousModel != nil {
		shouldPopulateDhcp = !previousModel.DhcpServer.IsNull() ||
			(isImport && network.DHCPDEnabled)
	}

	if shouldPopulateDhcp {
		// Helper function to convert empty strings to null
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

		dhcpBootObj, d := types.ObjectValueFrom(
			ctx,
			dhcpBootValue.AttributeTypes(),
			dhcpBootValue,
		)
		diags.Append(d...)

		// Build DNS servers list from DHCPDDNS1-4
		var dnsServers []string
		if network.DHCPDDNS1 != "" {
			dnsServers = append(dnsServers, network.DHCPDDNS1)
		}
		if network.DHCPDDNS2 != "" {
			dnsServers = append(dnsServers, network.DHCPDDNS2)
		}
		if network.DHCPDDNS3 != "" {
			dnsServers = append(dnsServers, network.DHCPDDNS3)
		}
		if network.DHCPDDNS4 != "" {
			dnsServers = append(dnsServers, network.DHCPDDNS4)
		}

		var dnsServersList types.List
		if len(dnsServers) > 0 {
			dnsServersList, d = types.ListValueFrom(ctx, types.StringType, dnsServers)
			diags.Append(d...)
		} else {
			dnsServersList = types.ListNull(types.StringType)
		}

		// Build WINS addresses list from DHCPDWins1-2
		var winsAddresses []string
		if network.DHCPDWins1 != nil && *network.DHCPDWins1 != "" {
			winsAddresses = append(winsAddresses, *network.DHCPDWins1)
		}
		if network.DHCPDWins2 != nil && *network.DHCPDWins2 != "" {
			winsAddresses = append(winsAddresses, *network.DHCPDWins2)
		}

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
			GatewayEnabled:    types.BoolValue(network.DHCPDGatewayEnabled),
			ConflictChecking:  types.BoolValue(network.DHCPDConflictChecking),
			NtpEnabled:        types.BoolValue(network.DHCPDNtpEnabled),
			TimeOffsetEnabled: types.BoolValue(network.DHCPDTimeOffsetEnabled),
			DnsEnabled:        types.BoolValue(network.DHCPDDNSEnabled),
			Leasetime:         types.Int64PointerValue(network.DHCPDLeaseTime),
			Wins:              winsObj,
			WpadUrl:           strPtrToType(network.DHCPDWPAdUrl),
			Start:             types.StringPointerValue(network.DHCPDStart),
			Stop:              types.StringPointerValue(network.DHCPDStop),
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
	} else {
		// Keep dhcp_server null if it wasn't in the plan/state
		model.DhcpServer = types.ObjectNull(dhcpServerModel{}.AttributeTypes())
	}

	// Only populate dhcp_relay if:
	// 1. It was configured in the previous state (not null), OR
	// 2. This is an import and DHCP relay is enabled (populate everything during import)
	shouldPopulateRelay := false
	if previousModel != nil {
		shouldPopulateRelay = !previousModel.DhcpRelay.IsNull() ||
			(isImport && network.DHCPRelayEnabled)
	}

	if shouldPopulateRelay {
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
	} else {
		// Keep dhcp_relay null if it wasn't in the plan/state
		model.DhcpRelay = types.ObjectNull(dhcpRelayModel{}.AttributeTypes())
	}

	return diags
}
