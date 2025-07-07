package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &networkFrameworkResource{}
var _ resource.ResourceWithImportState = &networkFrameworkResource{}

func NewNetworkFrameworkResource() resource.Resource {
	return &networkFrameworkResource{}
}

// networkFrameworkResource defines the resource implementation.
type networkFrameworkResource struct {
	client *client
}

// networkFrameworkResourceModel describes the resource data model.
type networkFrameworkResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Site        types.String `tfsdk:"site"`
	Name        types.String `tfsdk:"name"`
	Purpose     types.String `tfsdk:"purpose"`
	VlanID      types.Int64  `tfsdk:"vlan_id"`
	Subnet      types.String `tfsdk:"subnet"`
	NetworkGroup types.String `tfsdk:"network_group"`
	
	// DHCP Settings
	DhcpStart         types.String `tfsdk:"dhcp_start"`
	DhcpStop          types.String `tfsdk:"dhcp_stop"`
	DhcpEnabled       types.Bool   `tfsdk:"dhcp_enabled"`
	DhcpLease         types.Int64  `tfsdk:"dhcp_lease"`
	DhcpDNS           types.List   `tfsdk:"dhcp_dns"`
	DhcpdBootEnabled  types.Bool   `tfsdk:"dhcpd_boot_enabled"`
	DhcpdBootServer   types.String `tfsdk:"dhcpd_boot_server"`
	DhcpdBootFilename types.String `tfsdk:"dhcpd_boot_filename"`
	DhcpRelayEnabled  types.Bool   `tfsdk:"dhcp_relay_enabled"`
	
	// DHCPv6 Settings
	DhcpV6DNS        types.List `tfsdk:"dhcp_v6_dns"`
	DhcpV6DNSAuto    types.Bool `tfsdk:"dhcp_v6_dns_auto"`
	DhcpV6Enabled    types.Bool `tfsdk:"dhcp_v6_enabled"`
	DhcpV6Lease      types.Int64 `tfsdk:"dhcp_v6_lease"`
	DhcpV6PDStart    types.String `tfsdk:"dhcp_v6_pd_start"`
	DhcpV6PDStop     types.String `tfsdk:"dhcp_v6_pd_stop"`
	DhcpV6Start      types.String `tfsdk:"dhcp_v6_start"`
	DhcpV6Stop       types.String `tfsdk:"dhcp_v6_stop"`
	
	// IPv6 Settings
	IPv6InterfaceType types.String `tfsdk:"ipv6_interface_type"`
	IPv6PDPrefixid    types.String `tfsdk:"ipv6_pd_prefixid"`
	IPv6PDStart       types.String `tfsdk:"ipv6_pd_start"`
	IPv6PDStop        types.String `tfsdk:"ipv6_pd_stop"`
	IPv6RAPriority    types.String `tfsdk:"ipv6_ra_priority"`
	IPv6RAValidLifetime types.Int64 `tfsdk:"ipv6_ra_valid_lifetime"`
	IPv6RAPreferredLifetime types.Int64 `tfsdk:"ipv6_ra_preferred_lifetime"`
	IPv6RAEnable      types.Bool   `tfsdk:"ipv6_ra_enable"`
	IPv6Static        types.List   `tfsdk:"ipv6_static"`
	
	// WAN Settings
	WANType              types.String `tfsdk:"wan_type"`
	WANUsername          types.String `tfsdk:"wan_username"`
	WANPassword          types.String `tfsdk:"wan_password"`
	WANIp                types.String `tfsdk:"wan_ip"`
	WANGateway           types.String `tfsdk:"wan_gateway"`
	WANNetmask           types.String `tfsdk:"wan_netmask"`
	WANDNS               types.List   `tfsdk:"wan_dns"`
	WANNetworkGroup      types.String `tfsdk:"wan_network_group"`
	WANDHCPV6            types.Bool   `tfsdk:"wan_dhcp_v6"`
	WANGatewayV6         types.String `tfsdk:"wan_gateway_v6"`
	WANIPv6              types.String `tfsdk:"wan_ipv6"`
	WANPrefixlen         types.Int64  `tfsdk:"wan_prefixlen"`
	WANTypeV6            types.String `tfsdk:"wan_type_v6"`
	
	// WireGuard Settings
	WireguardClientMode            types.String `tfsdk:"wireguard_client_mode"`
	WireguardClientPeerIP          types.String `tfsdk:"wireguard_client_peer_ip"`
	WireguardClientPeerPort        types.Int64  `tfsdk:"wireguard_client_peer_port"`
	WireguardClientPeerPublicKey   types.String `tfsdk:"wireguard_client_peer_public_key"`
	WireguardClientPresharedKey    types.String `tfsdk:"wireguard_client_preshared_key"`
	WireguardClientPresharedKeyEnabled types.Bool `tfsdk:"wireguard_client_preshared_key_enabled"`
	WireguardID                    types.Int64  `tfsdk:"wireguard_id"`
	WireguardPublicKey            types.String `tfsdk:"wireguard_public_key"`
	WireguardPrivateKey           types.String `tfsdk:"wireguard_private_key"`
}

func (r *networkFrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_network` manages WAN/LAN/VLAN networks.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the network with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the network.",
				Required:            true,
			},
			"purpose": schema.StringAttribute{
				MarkdownDescription: "The purpose of the network. Must be one of `corporate`, `guest`, `wan`, `vlan-only`, or `vpn-client`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("corporate", "guest", "wan", "vlan-only", "vpn-client"),
				},
			},
			"vlan_id": schema.Int64Attribute{
				MarkdownDescription: "The VLAN ID of the network.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 4096),
				},
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The subnet of the network. Must be a valid CIDR address.",
				Optional:            true,
				Validators: []validator.String{
					// TODO: Add CIDR validation
				},
			},
			"network_group": schema.StringAttribute{
				MarkdownDescription: "The group of the network.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("LAN"),
			},
			
			// DHCP Settings
			"dhcp_start": schema.StringAttribute{
				MarkdownDescription: "The IPv4 address where the DHCP range of addresses starts.",
				Optional:            true,
				Validators: []validator.String{
					// TODO: Add IPv4 validation
				},
			},
			"dhcp_stop": schema.StringAttribute{
				MarkdownDescription: "The IPv4 address where the DHCP range of addresses stops.",
				Optional:            true,
				Validators: []validator.String{
					// TODO: Add IPv4 validation
				},
			},
			"dhcp_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether DHCP is enabled or not on this network.",
				Optional:            true,
			},
			"dhcp_lease": schema.Int64Attribute{
				MarkdownDescription: "Specifies the lease time for DHCP addresses in seconds.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(86400),
			},
			"dhcp_dns": schema.ListAttribute{
				MarkdownDescription: "Specifies the IPv4 addresses for the DNS server to be returned from the DHCP server. Leave blank to disable this feature.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(4),
				},
			},
			"dhcpd_boot_enabled": schema.BoolAttribute{
				MarkdownDescription: "Toggles on the DHCP boot options. Should be set to true when you want to have dhcpd_boot_filename, and dhcpd_boot_server to take effect.",
				Optional:            true,
			},
			"dhcpd_boot_server": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPv4 address of a TFTP server to network boot from.",
				Optional:            true,
			},
			"dhcpd_boot_filename": schema.StringAttribute{
				MarkdownDescription: "Specifies the file to PXE boot from on the dhcpd_boot_server.",
				Optional:            true,
			},
			"dhcp_relay_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether DHCP relay is enabled or not on this network.",
				Optional:            true,
			},
			
			// DHCPv6 Settings
			"dhcp_v6_dns": schema.ListAttribute{
				MarkdownDescription: "Specifies the IPv6 addresses for the DNS server to be returned from the DHCP server. Used if `dhcp_v6_dns_auto` is set to `false`.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(4),
				},
			},
			"dhcp_v6_dns_auto": schema.BoolAttribute{
				MarkdownDescription: "Specifies DNS source to propagate. If set `false` the entries in `dhcp_v6_dns` are used, the upstream entries otherwise",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"dhcp_v6_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable stateful DHCPv6 for static configuration.",
				Optional:            true,
			},
			"dhcp_v6_lease": schema.Int64Attribute{
				MarkdownDescription: "Specifies the lease time for DHCPv6 addresses in seconds.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(86400),
			},
			"dhcp_v6_pd_start": schema.StringAttribute{
				MarkdownDescription: "Start address of the DHCPv6 Prefix Delegation pool. Used if `ipv6_interface_type` is set to `pd`.",
				Optional:            true,
			},
			"dhcp_v6_pd_stop": schema.StringAttribute{
				MarkdownDescription: "End address of the DHCPv6 Prefix Delegation pool. Used if `ipv6_interface_type` is set to `pd`.",
				Optional:            true,
			},
			"dhcp_v6_start": schema.StringAttribute{
				MarkdownDescription: "Start address of the DHCPv6 pool. Used if `dhcp_v6_enabled` is set to `true`.",
				Optional:            true,
			},
			"dhcp_v6_stop": schema.StringAttribute{
				MarkdownDescription: "End address of the DHCPv6 pool. Used if `dhcp_v6_enabled` is set to `true`.",
				Optional:            true,
			},
			
			// IPv6 Settings  
			"ipv6_interface_type": schema.StringAttribute{
				MarkdownDescription: "Specifies which type of IPv6 connection to use. Must be one of either `none`, `pd`, or `static`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^(none|pd|static)$"),
						"invalid IPv6 interface type",
					),
				},
			},
			"ipv6_pd_prefixid": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPv6 Prefix ID.",
				Optional:            true,
			},
			"ipv6_pd_start": schema.StringAttribute{
				MarkdownDescription: "Start address of the DHCPv6 Prefix Delegation pool. Used if `ipv6_interface_type` is set to `pd`.",
				Optional:            true,
			},
			"ipv6_pd_stop": schema.StringAttribute{
				MarkdownDescription: "End address of the DHCPv6 Prefix Delegation pool. Used if `ipv6_interface_type` is set to `pd`.",
				Optional:            true,
			},
			"ipv6_ra_priority": schema.StringAttribute{
				MarkdownDescription: "IPv6 router advertisement priority. Must be one of either `high`, `medium`, or `low`",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^(high|medium|low)$"),
						"invalid IPv6 RA priority",
					),
				},
			},
			"ipv6_ra_valid_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Lifetime in which the prefix is valid for the purpose of on-link determination. Value is in seconds.",
				Optional:            true,
			},
			"ipv6_ra_preferred_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Lifetime in which addresses generated from the prefix remain preferred. Value is in seconds.",
				Optional:            true,
			},
			"ipv6_ra_enable": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to enable router advertisements or not.",
				Optional:            true,
			},
			"ipv6_static": schema.ListAttribute{
				MarkdownDescription: "Specifies the static IPv6 addresses for the network.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			
			// WAN Settings
			"wan_type": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPV4 WAN connection type. Must be one of either `disabled`, `dhcp`, `static`, or `pppoe`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^(disabled|dhcp|static|pppoe)$"),
						"invalid WAN connection type",
					),
				},
			},
			"wan_username": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPV4 WAN username.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[^"' ]+`),
						"invalid WAN username",
					),
				},
			},
			"wan_password": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPV4 WAN password.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[^"' ]+`),
						"invalid WAN password",
					),
				},
			},
			"wan_ip": schema.StringAttribute{
				MarkdownDescription: "The IPv4 address of the WAN.",
				Optional:            true,
			},
			"wan_gateway": schema.StringAttribute{
				MarkdownDescription: "The IPv4 gateway of the WAN.",
				Optional:            true,
			},
			"wan_netmask": schema.StringAttribute{
				MarkdownDescription: "The IPv4 netmask of the WAN.",
				Optional:            true,
			},
			"wan_dns": schema.ListAttribute{
				MarkdownDescription: "DNS servers IPs of the WAN.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"wan_network_group": schema.StringAttribute{
				MarkdownDescription: "Specifies the WAN network group. Must be one of either `WAN`, `WAN2` or `WAN_LTE_FAILOVER`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^(WAN[2]?|WAN_LTE_FAILOVER)$"),
						"invalid WAN network group",
					),
				},
			},
			"wan_dhcp_v6": schema.BoolAttribute{
				MarkdownDescription: "Enable stateful DHCPv6 for the WAN.",
				Optional:            true,
			},
			"wan_gateway_v6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 gateway of the WAN.",
				Optional:            true,
			},
			"wan_ipv6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 address of the WAN.",
				Optional:            true,
			},
			"wan_prefixlen": schema.Int64Attribute{
				MarkdownDescription: "The IPv6 prefix length of the WAN. Must be between 1 and 128.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 128),
				},
			},
			"wan_type_v6": schema.StringAttribute{
				MarkdownDescription: "Specifies the IPV6 WAN connection type. Must be one of either `disabled`, `dhcpv6`, or `static`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^(disabled|dhcpv6|static)$"),
						"invalid WANv6 connection type",
					),
				},
			},
			
			// WireGuard Settings
			"wireguard_client_mode": schema.StringAttribute{
				MarkdownDescription: "Specifies the Wireguard client mode. Must be one of either `file` or `manual`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^(file|manual)$"),
						"invalid Wireguard client mode",
					),
				},
			},
			"wireguard_client_peer_ip": schema.StringAttribute{
				MarkdownDescription: "Specifies the Wireguard client peer IP.",
				Optional:            true,
			},
			"wireguard_client_peer_port": schema.Int64Attribute{
				MarkdownDescription: "Specifies the Wireguard client peer port.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"wireguard_client_peer_public_key": schema.StringAttribute{
				MarkdownDescription: "Specifies the Wireguard client peer public key.",
				Optional:            true,
			},
			"wireguard_client_preshared_key": schema.StringAttribute{
				MarkdownDescription: "Specifies the Wireguard client preshared key.",
				Optional:            true,
			},
			"wireguard_client_preshared_key_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the Wireguard client preshared key is enabled or not.",
				Optional:            true,
			},
			"wireguard_id": schema.Int64Attribute{
				MarkdownDescription: "Specifies the Wireguard ID.",
				Optional:            true,
			},
			"wireguard_public_key": schema.StringAttribute{
				MarkdownDescription: "Specifies the Wireguard public key.",
				Optional:            true,
			},
			"wireguard_private_key": schema.StringAttribute{
				MarkdownDescription: "Specifies the Wireguard private key.",
				Optional:            true,
			},
		},
	}
}

func (r *networkFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *networkFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data networkFrameworkResourceModel

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
		site = r.client.site
	}

	// Create the network
	createdNetwork, err := r.client.c.CreateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Network",
			"Could not create network, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert back to model
	diags = r.networkToModel(ctx, createdNetwork, &data, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data networkFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	// Get the network
	network, err := r.client.c.GetNetwork(ctx, site, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network",
			"Could not read network ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert to model
	diags := r.networkToModel(ctx, network, &data, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data networkFrameworkResourceModel

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
		site = r.client.site
	}

	// Update the network
	network.ID = data.ID.ValueString()
	updatedNetwork, err := r.client.c.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Network",
			"Could not update network, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert back to model
	diags = r.networkToModel(ctx, updatedNetwork, &data, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data networkFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	// Delete the network
	name := data.Name.ValueString() // Get name for deletion
	err := r.client.c.DeleteNetwork(ctx, site, data.ID.ValueString(), name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Network",
			"Could not delete network, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *networkFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "site:id" or "name=NetworkName"
	idParts := strings.Split(req.ID, ":")
	
	if len(idParts) == 2 {
		// site:id format
		site := idParts[0]
		id := idParts[1]
		
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}
	
	// Check for name=NetworkName format
	if strings.HasPrefix(req.ID, "name=") {
		networkName := strings.TrimPrefix(req.ID, "name=")
		
		// Find network by name in default site
		site := r.client.site
		networkID, err := getNetworkIDByNameFramework(ctx, r.client.c.(*lazyClient), site, networkName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Finding Network",
				fmt.Sprintf("Could not find network with name %s: %s", networkName, err.Error()),
			)
			return
		}
		
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), networkID)...)
		return
	}
	
	// Default format: just the ID in default site
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), r.client.site)...)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to find network ID by name
func getNetworkIDByNameFramework(ctx context.Context, client *lazyClient, site, name string) (string, error) {
	networks, err := client.ListNetwork(ctx, site)
	if err != nil {
		return "", err
	}
	
	for _, network := range networks {
		if network.Name == name {
			return network.ID, nil
		}
	}
	
	return "", fmt.Errorf("network with name %s not found", name)
}

// modelToNetwork converts from Terraform model to unifi.Network
func (r *networkFrameworkResource) modelToNetwork(ctx context.Context, model *networkFrameworkResourceModel) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics
	
	network := &unifi.Network{
		Name:    model.Name.ValueString(),
		Purpose: model.Purpose.ValueString(),
	}
	
	if !model.VlanID.IsNull() {
		network.VLAN = int(model.VlanID.ValueInt64())
	}
	
	if !model.Subnet.IsNull() {
		network.IPSubnet = model.Subnet.ValueString()
	}
	
	if !model.NetworkGroup.IsNull() {
		network.NetworkGroup = model.NetworkGroup.ValueString()
	}
	
	// DHCP Settings
	if !model.DhcpStart.IsNull() {
		network.DHCPDStart = model.DhcpStart.ValueString()
	}
	if !model.DhcpStop.IsNull() {
		network.DHCPDStop = model.DhcpStop.ValueString()
	}
	if !model.DhcpEnabled.IsNull() {
		network.DHCPDEnabled = model.DhcpEnabled.ValueBool()
	}
	if !model.DhcpLease.IsNull() {
		network.DHCPDLeaseTime = int(model.DhcpLease.ValueInt64())
	}
	
	// Convert DHCP DNS list
	if !model.DhcpDNS.IsNull() {
		var dhcpDNS []string
		d := model.DhcpDNS.ElementsAs(ctx, &dhcpDNS, false)
		diags.Append(d...)
		if !diags.HasError() {
			network.DHCPDDNS1 = ""
			network.DHCPDDNS2 = ""
			network.DHCPDDNS3 = ""
			network.DHCPDDNS4 = ""
			for i, dns := range dhcpDNS {
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
		}
	}
	
	if !model.DhcpdBootEnabled.IsNull() {
		network.DHCPDBootEnabled = model.DhcpdBootEnabled.ValueBool()
	}
	if !model.DhcpdBootServer.IsNull() {
		network.DHCPDBootServer = model.DhcpdBootServer.ValueString()
	}
	if !model.DhcpdBootFilename.IsNull() {
		network.DHCPDBootFilename = model.DhcpdBootFilename.ValueString()
	}
	if !model.DhcpRelayEnabled.IsNull() {
		network.DHCPRelayEnabled = model.DhcpRelayEnabled.ValueBool()
	}
	
	// TODO: Add more field mappings for DHCPv6, IPv6, WAN, WireGuard settings
	
	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model
func (r *networkFrameworkResource) networkToModel(ctx context.Context, network *unifi.Network, model *networkFrameworkResourceModel, site string) diag.Diagnostics {
	var diags diag.Diagnostics
	
	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(network.Name)
	model.Purpose = types.StringValue(network.Purpose)
	
	if network.VLAN != 0 {
		model.VlanID = types.Int64Value(int64(network.VLAN))
	} else {
		model.VlanID = types.Int64Null()
	}
	
	if network.IPSubnet != "" {
		model.Subnet = types.StringValue(network.IPSubnet)
	} else {
		model.Subnet = types.StringNull()
	}
	
	if network.NetworkGroup != "" {
		model.NetworkGroup = types.StringValue(network.NetworkGroup)
	} else {
		model.NetworkGroup = types.StringValue("LAN") // Default value
	}
	
	// DHCP Settings
	if network.DHCPDStart != "" {
		model.DhcpStart = types.StringValue(network.DHCPDStart)
	} else {
		model.DhcpStart = types.StringNull()
	}
	
	if network.DHCPDStop != "" {
		model.DhcpStop = types.StringValue(network.DHCPDStop)
	} else {
		model.DhcpStop = types.StringNull()
	}
	
	model.DhcpEnabled = types.BoolValue(network.DHCPDEnabled)
	
	if network.DHCPDLeaseTime != 0 {
		model.DhcpLease = types.Int64Value(int64(network.DHCPDLeaseTime))
	} else {
		model.DhcpLease = types.Int64Value(86400) // Default value
	}
	
	// Convert DHCP DNS from individual fields to list
	dhcpDNSSlice := []string{}
	for _, dns := range []string{network.DHCPDDNS1, network.DHCPDDNS2, network.DHCPDDNS3, network.DHCPDDNS4} {
		if dns != "" {
			dhcpDNSSlice = append(dhcpDNSSlice, dns)
		}
	}
	
	if len(dhcpDNSSlice) > 0 {
		dhcpDNSValues := make([]attr.Value, len(dhcpDNSSlice))
		for i, dns := range dhcpDNSSlice {
			dhcpDNSValues[i] = types.StringValue(dns)
		}
		dhcpDNSList, d := types.ListValue(types.StringType, dhcpDNSValues)
		diags.Append(d...)
		model.DhcpDNS = dhcpDNSList
	} else {
		model.DhcpDNS = types.ListNull(types.StringType)
	}
	
	model.DhcpdBootEnabled = types.BoolValue(network.DHCPDBootEnabled)
	
	if network.DHCPDBootServer != "" {
		model.DhcpdBootServer = types.StringValue(network.DHCPDBootServer)
	} else {
		model.DhcpdBootServer = types.StringNull()
	}
	
	if network.DHCPDBootFilename != "" {
		model.DhcpdBootFilename = types.StringValue(network.DHCPDBootFilename)
	} else {
		model.DhcpdBootFilename = types.StringNull()
	}
	
	model.DhcpRelayEnabled = types.BoolValue(network.DHCPRelayEnabled)
	
	// TODO: Add more field mappings for DHCPv6, IPv6, WAN, WireGuard settings
	// For now, set remaining fields to null to prevent issues
	model.DhcpV6DNS = types.ListNull(types.StringType)
	model.DhcpV6DNSAuto = types.BoolValue(true) // Default value
	model.DhcpV6Enabled = types.BoolNull()
	model.DhcpV6Lease = types.Int64Value(86400) // Default value
	model.DhcpV6PDStart = types.StringNull()
	model.DhcpV6PDStop = types.StringNull()
	model.DhcpV6Start = types.StringNull()
	model.DhcpV6Stop = types.StringNull()
	
	// IPv6 Settings
	model.IPv6InterfaceType = types.StringNull()
	model.IPv6PDPrefixid = types.StringNull()
	model.IPv6PDStart = types.StringNull()
	model.IPv6PDStop = types.StringNull()
	model.IPv6RAPriority = types.StringNull()
	model.IPv6RAValidLifetime = types.Int64Null()
	model.IPv6RAPreferredLifetime = types.Int64Null()
	model.IPv6RAEnable = types.BoolNull()
	model.IPv6Static = types.ListNull(types.StringType)
	
	// WAN Settings
	model.WANType = types.StringNull()
	model.WANUsername = types.StringNull()
	model.WANPassword = types.StringNull()
	model.WANIp = types.StringNull()
	model.WANGateway = types.StringNull()
	model.WANNetmask = types.StringNull()
	model.WANDNS = types.ListNull(types.StringType)
	model.WANNetworkGroup = types.StringNull()
	model.WANDHCPV6 = types.BoolNull()
	model.WANGatewayV6 = types.StringNull()
	model.WANIPv6 = types.StringNull()
	model.WANPrefixlen = types.Int64Null()
	model.WANTypeV6 = types.StringNull()
	
	// WireGuard Settings
	model.WireguardClientMode = types.StringNull()
	model.WireguardClientPeerIP = types.StringNull()
	model.WireguardClientPeerPort = types.Int64Null()
	model.WireguardClientPeerPublicKey = types.StringNull()
	model.WireguardClientPresharedKey = types.StringNull()
	model.WireguardClientPresharedKeyEnabled = types.BoolNull()
	model.WireguardID = types.Int64Null()
	model.WireguardPublicKey = types.StringNull()
	model.WireguardPrivateKey = types.StringNull()
	
	return diags
}