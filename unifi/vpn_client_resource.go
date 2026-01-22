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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &vpnClientResource{}
	_ resource.ResourceWithImportState = &vpnClientResource{}
)

func NewVPNClientResource() resource.Resource {
	return &vpnClientResource{}
}

// vpnClientResource defines the resource implementation.
type vpnClientResource struct {
	client *Client
}

// wireguardConfigurationModel describes the WireGuard configuration file upload.
type wireguardConfigurationModel struct {
	Content  types.String `tfsdk:"content"`
	Filename types.String `tfsdk:"filename"`
}

func (m wireguardConfigurationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"content":  types.StringType,
		"filename": types.StringType,
	}
}

// wireguardPeerModel describes the WireGuard peer configuration for manual mode.
type wireguardPeerModel struct {
	IP        types.String `tfsdk:"ip"`
	Port      types.Int64  `tfsdk:"port"`
	PublicKey types.String `tfsdk:"public_key"`
}

func (m wireguardPeerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip":         types.StringType,
		"port":       types.Int64Type,
		"public_key": types.StringType,
	}
}

// wireguardModel describes the WireGuard VPN configuration.
type wireguardModel struct {
	PrivateKey          types.String `tfsdk:"private_key"`
	Configuration       types.Object `tfsdk:"configuration"`
	Peer                types.Object `tfsdk:"peer"`
	PresharedKeyEnabled types.Bool   `tfsdk:"preshared_key_enabled"`
	PresharedKey        types.String `tfsdk:"preshared_key"`
	Interface           types.String `tfsdk:"interface"`
	DnsServers          types.List   `tfsdk:"dns_servers"`
}

func (m wireguardModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"private_key": types.StringType,
		"configuration": types.ObjectType{
			AttrTypes: wireguardConfigurationModel{}.AttributeTypes(),
		},
		"peer":                  types.ObjectType{AttrTypes: wireguardPeerModel{}.AttributeTypes()},
		"preshared_key_enabled": types.BoolType,
		"preshared_key":         types.StringType,
		"interface":             types.StringType,
		"dns_servers":           types.ListType{ElemType: types.StringType},
	}
}

// vpnClientResourceModel describes the resource data model.
type vpnClientResourceModel struct {
	ID           types.String         `tfsdk:"id"`
	Site         types.String         `tfsdk:"site"`
	Name         types.String         `tfsdk:"name"`
	Enabled      types.Bool           `tfsdk:"enabled"`
	Subnet       cidrtypes.IPv4Prefix `tfsdk:"subnet"`
	DefaultRoute types.Bool           `tfsdk:"default_route"`
	PullDNS      types.Bool           `tfsdk:"pull_dns"`
	Wireguard    types.Object         `tfsdk:"wireguard"`
}

func (r *vpnClientResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_vpn_client"
}

func (r *vpnClientResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_vpn_client` manages WireGuard VPN client connections in the UniFi controller.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the VPN client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the VPN client with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VPN client.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the VPN client is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The local IP address for the WireGuard tunnel in CIDR notation (e.g., `10.0.0.2/24`).",
				Required:            true,
				CustomType:          cidrtypes.IPv4PrefixType{},
			},
			"default_route": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to use the VPN as the default route.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"pull_dns": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to pull DNS servers from the VPN.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"wireguard": schema.SingleNestedAttribute{
				MarkdownDescription: "WireGuard VPN configuration. Specify either `configuration` for file upload mode or `peer` for manual configuration.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"private_key": schema.StringAttribute{
						MarkdownDescription: "WireGuard private key for this client.",
						Required:            true,
						Sensitive:           true,
					},
					"configuration": schema.SingleNestedAttribute{
						MarkdownDescription: "File-based WireGuard configuration. Provide a complete WireGuard .conf file.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"content": schema.StringAttribute{
								MarkdownDescription: "Base64-encoded WireGuard configuration file content.",
								Required:            true,
								Sensitive:           true,
							},
							"filename": schema.StringAttribute{
								MarkdownDescription: "Filename of the WireGuard configuration file.",
								Required:            true,
							},
						},
					},
					"peer": schema.SingleNestedAttribute{
						MarkdownDescription: "Manual WireGuard peer configuration. Specify peer endpoint and public key.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"ip": schema.StringAttribute{
								MarkdownDescription: "WireGuard peer endpoint IP address.",
								Required:            true,
							},
							"port": schema.Int64Attribute{
								MarkdownDescription: "WireGuard peer endpoint port.",
								Required:            true,
								Validators: []validator.Int64{
									int64validator.Between(1, 65535),
								},
							},
							"public_key": schema.StringAttribute{
								MarkdownDescription: "WireGuard peer public key.",
								Required:            true,
								Sensitive:           true,
							},
						},
					},
					"preshared_key_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether to use a preshared key for additional security.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"preshared_key": schema.StringAttribute{
						MarkdownDescription: "WireGuard preshared key. Required when preshared_key_enabled is true.",
						Optional:            true,
						Sensitive:           true,
					},
					"interface": schema.StringAttribute{
						MarkdownDescription: "WAN interface to use for the VPN connection (e.g., `wan`, `wan2`).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("wan"),
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^wan[2-9]?$`),
								"must be 'wan' or 'wan2' through 'wan9'",
							),
						},
					},
					"dns_servers": schema.ListAttribute{
						MarkdownDescription: "DNS servers for the WireGuard interface. Required for manual mode. Must specify 1-2 DNS server addresses.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.SizeBetween(1, 2),
						},
					},
				},
			},
		},
	}
}

func (r *vpnClientResource) Configure(
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

func (r *vpnClientResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data vpnClientResourceModel

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
			"Error Creating VPN Client",
			err.Error(),
		)
		return
	}

	// Convert back to model
	var planData vpnClientResourceModel
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

func (r *vpnClientResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data vpnClientResourceModel

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
				"Error Reading VPN Client",
				"Could not read VPN client ID "+data.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else {
		// Get the network by name
		network, err = r.client.GetNetworkByName(ctx, site, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VPN Client",
				"Could not read VPN client name "+data.Name.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Convert to model
	var priorState vpnClientResourceModel
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

func (r *vpnClientResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data vpnClientResourceModel

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
			"Error Updating VPN Client",
			err.Error(),
		)
		return
	}

	// Convert back to model
	var planData vpnClientResourceModel
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

func (r *vpnClientResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data vpnClientResourceModel

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
			"Error Deleting VPN Client",
			err.Error(),
		)
		return
	}
}

func (r *vpnClientResource) ImportState(
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
func (r *vpnClientResource) modelToNetwork(
	ctx context.Context,
	model *vpnClientResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:                  model.Name.ValueStringPointer(),
		Purpose:               unifi.PurposeVPNClient,
		Enabled:               model.Enabled.ValueBool(),
		IPSubnet:              model.Subnet.ValueStringPointer(),
		VPNType:               util.Ptr("wireguard-client"),
		VPNClientDefaultRoute: model.DefaultRoute.ValueBool(),
		VPNClientPullDNS:      model.PullDNS.ValueBool(),
	}

	// Handle WireGuard configuration
	if !model.Wireguard.IsNull() && !model.Wireguard.IsUnknown() {
		var wireguard wireguardModel
		d := model.Wireguard.As(ctx, &wireguard, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.WireguardPrivateKey = wireguard.PrivateKey.ValueStringPointer()
			network.WireguardClientPresharedKeyEnabled = wireguard.PresharedKeyEnabled.ValueBool()
			network.WireguardInterface = wireguard.Interface.ValueStringPointer()

			// Handle DNS servers
			if !wireguard.DnsServers.IsNull() && !wireguard.DnsServers.IsUnknown() {
				var dnsServers []string
				d := wireguard.DnsServers.ElementsAs(ctx, &dnsServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					if len(dnsServers) > 0 {
						network.DHCPDDNS1 = dnsServers[0]
					}
					if len(dnsServers) > 1 {
						network.DHCPDDNS2 = dnsServers[1]
					}
				}
			}

			// Check if configuration (file mode) is set
			if !wireguard.Configuration.IsNull() && !wireguard.Configuration.IsUnknown() {
				var config wireguardConfigurationModel
				d := wireguard.Configuration.As(ctx, &config, basetypes.ObjectAsOptions{})
				diags.Append(d...)
				if !diags.HasError() {
					network.WireguardClientMode = util.Ptr("file")
					network.WireguardClientConfigurationFile = config.Content.ValueStringPointer()
					network.WireguardClientConfigurationFilename = config.Filename.ValueStringPointer()
				}
			} else if !wireguard.Peer.IsNull() && !wireguard.Peer.IsUnknown() {
				// Check if peer (manual mode) is set
				var peer wireguardPeerModel
				d := wireguard.Peer.As(ctx, &peer, basetypes.ObjectAsOptions{})
				diags.Append(d...)
				if !diags.HasError() {
					network.WireguardClientMode = util.Ptr("manual")
					network.WireguardClientPeerIP = peer.IP.ValueStringPointer()
					network.WireguardClientPeerPort = peer.Port.ValueInt64Pointer()
					network.WireguardClientPeerPublicKey = peer.PublicKey.ValueStringPointer()
				}
			}

			// Preshared key (optional for both modes)
			if wireguard.PresharedKeyEnabled.ValueBool() {
				network.WireguardClientPresharedKey = wireguard.PresharedKey.ValueStringPointer()
			}
		}
	}

	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model.
func (r *vpnClientResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *vpnClientResourceModel,
	site string,
	previousModel *vpnClientResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringPointerValue(network.Name)
	model.Enabled = types.BoolValue(network.Enabled)
	if network.IPSubnet != nil {
		model.Subnet = cidrtypes.NewIPv4PrefixValue(*network.IPSubnet)
	} else {
		model.Subnet = cidrtypes.NewIPv4PrefixNull()
	}
	model.DefaultRoute = types.BoolValue(network.VPNClientDefaultRoute)
	model.PullDNS = types.BoolValue(network.VPNClientPullDNS)

	// Helper function to convert empty strings to null
	strPtrToType := func(ptr *string) types.String {
		if ptr == nil || *ptr == "" {
			return types.StringNull()
		}
		return types.StringValue(*ptr)
	}

	// Build WireGuard configuration
	var configurationObj types.Object
	var peerObj types.Object

	// Determine mode from API response and build appropriate nested objects
	if network.WireguardClientMode != nil && *network.WireguardClientMode == "file" {
		// File mode: populate configuration
		configValue := wireguardConfigurationModel{
			Content:  strPtrToType(network.WireguardClientConfigurationFile),
			Filename: strPtrToType(network.WireguardClientConfigurationFilename),
		}
		var d diag.Diagnostics
		configurationObj, d = types.ObjectValueFrom(ctx, configValue.AttributeTypes(), configValue)
		diags.Append(d...)
		peerObj = types.ObjectNull(wireguardPeerModel{}.AttributeTypes())
	} else if network.WireguardClientMode != nil && *network.WireguardClientMode == "manual" {
		// Manual mode: populate peer
		peerValue := wireguardPeerModel{
			IP:        strPtrToType(network.WireguardClientPeerIP),
			Port:      types.Int64PointerValue(network.WireguardClientPeerPort),
			PublicKey: strPtrToType(network.WireguardClientPeerPublicKey),
		}
		var d diag.Diagnostics
		peerObj, d = types.ObjectValueFrom(ctx, peerValue.AttributeTypes(), peerValue)
		diags.Append(d...)
		configurationObj = types.ObjectNull(wireguardConfigurationModel{}.AttributeTypes())
	} else {
		// No mode set - both null
		configurationObj = types.ObjectNull(wireguardConfigurationModel{}.AttributeTypes())
		peerObj = types.ObjectNull(wireguardPeerModel{}.AttributeTypes())
	}

	// Build DNS servers list
	var dnsServersList types.List
	var dnsServers []string
	if network.DHCPDDNS1 != "" {
		dnsServers = append(dnsServers, network.DHCPDDNS1)
	}
	if network.DHCPDDNS2 != "" {
		dnsServers = append(dnsServers, network.DHCPDDNS2)
	}
	if len(dnsServers) > 0 {
		var d diag.Diagnostics
		dnsServersList, d = types.ListValueFrom(ctx, types.StringType, dnsServers)
		diags.Append(d...)
	} else {
		dnsServersList = types.ListNull(types.StringType)
	}

	wireguardValue := wireguardModel{
		PrivateKey:          strPtrToType(network.WireguardPrivateKey),
		Configuration:       configurationObj,
		Peer:                peerObj,
		PresharedKeyEnabled: types.BoolValue(network.WireguardClientPresharedKeyEnabled),
		PresharedKey:        strPtrToType(network.WireguardClientPresharedKey),
		Interface:           types.StringPointerValue(network.WireguardInterface),
		DnsServers:          dnsServersList,
	}

	wireguardObj, d := types.ObjectValueFrom(
		ctx,
		wireguardValue.AttributeTypes(),
		wireguardValue,
	)
	diags.Append(d...)
	model.Wireguard = wireguardObj

	return diags
}
