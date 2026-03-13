package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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
	_ resource.Resource                = &vpnServerResource{}
	_ resource.ResourceWithImportState = &vpnServerResource{}
	_ resource.ResourceWithIdentity    = &vpnServerResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &vpnServerResource{}
	_ list.ListResourceWithConfigure = &vpnServerResource{}
)

func NewVPNServerResource() resource.Resource {
	return &vpnServerResource{}
}

func NewVPNServerListResource() list.ListResource {
	return &vpnServerResource{}
}

// vpnServerResource defines the resource implementation.
type vpnServerResource struct {
	client *Client
}

type vpnServerIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// vpnServerListConfigModel describes the list configuration model.
type vpnServerListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// vpnServerListFilterModel represents a single name/value filter entry.
type vpnServerListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// vpnServerDNSModel describes the DNS configuration for VPN clients.
type vpnServerDNSModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Servers types.List `tfsdk:"servers"`
}

func (m vpnServerDNSModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// vpnServerWANModel describes the WAN binding configuration shared across VPN types.
type vpnServerWANModel struct {
	IP        types.String `tfsdk:"ip"`
	Interface types.String `tfsdk:"interface"`
}

func (m vpnServerWANModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip":        types.StringType,
		"interface": types.StringType,
	}
}

// vpnServerWireguardModel describes the WireGuard-specific server configuration.
type vpnServerWireguardModel struct {
	PrivateKey types.String `tfsdk:"private_key"`
	PublicKey  types.String `tfsdk:"public_key"`
	Port       types.Int64  `tfsdk:"port"`
}

func (m vpnServerWireguardModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"private_key": types.StringType,
		"public_key":  types.StringType,
		"port":        types.Int64Type,
	}
}

// vpnServerL2TPModel describes the L2TP-specific server configuration.
type vpnServerL2TPModel struct {
	AllowWeakCiphers types.Bool   `tfsdk:"allow_weak_ciphers"`
	PreSharedKey     types.String `tfsdk:"pre_shared_key"`
}

func (m vpnServerL2TPModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"allow_weak_ciphers": types.BoolType,
		"pre_shared_key":     types.StringType,
	}
}

// vpnServerOpenVPNModel describes the OpenVPN-specific server configuration.
type vpnServerOpenVPNModel struct {
	Port             types.Int64  `tfsdk:"port"`
	Mode             types.String `tfsdk:"mode"`
	EncryptionCipher types.String `tfsdk:"encryption_cipher"`
	ServerCrt        types.String `tfsdk:"server_crt"`
	ServerKey        types.String `tfsdk:"server_key"`
	DhKey            types.String `tfsdk:"dh_key"`
	SharedClientKey  types.String `tfsdk:"shared_client_key"`
	SharedClientCrt  types.String `tfsdk:"shared_client_crt"`
	AuthKey          types.String `tfsdk:"auth_key"`
	CaCrt            types.String `tfsdk:"ca_crt"`
	CaKey            types.String `tfsdk:"ca_key"`
}

func (m vpnServerOpenVPNModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"port":              types.Int64Type,
		"mode":              types.StringType,
		"encryption_cipher": types.StringType,
		"server_crt":        types.StringType,
		"server_key":        types.StringType,
		"dh_key":            types.StringType,
		"shared_client_key": types.StringType,
		"shared_client_crt": types.StringType,
		"auth_key":          types.StringType,
		"ca_crt":            types.StringType,
		"ca_key":            types.StringType,
	}
}

// vpnServerResourceModel describes the resource data model.
type vpnServerResourceModel struct {
	ID              types.String         `tfsdk:"id"`
	Site            types.String         `tfsdk:"site"`
	Name            types.String         `tfsdk:"name"`
	Enabled         types.Bool           `tfsdk:"enabled"`
	Subnet          cidrtypes.IPv4Prefix `tfsdk:"subnet"`
	DNS             types.Object         `tfsdk:"dns"`
	WAN             types.Object         `tfsdk:"wan"`
	RADIUSProfileID types.String         `tfsdk:"radiusprofile_id"`
	Wireguard       types.Object         `tfsdk:"wireguard"`
	L2TP            types.Object         `tfsdk:"l2tp"`
	OpenVPN         types.Object         `tfsdk:"openvpn"`
}

func (r *vpnServerResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_vpn_server"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *vpnServerResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

func (r *vpnServerResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	wanInterfaceValidator := stringvalidator.RegexMatches(
		regexp.MustCompile(`^wan[2-9]?$`),
		"must be 'wan' or 'wan2' through 'wan9'",
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_vpn_server` manages VPN server configurations (WireGuard, L2TP, or OpenVPN) in the UniFi controller.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the VPN server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the VPN server with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VPN server.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the VPN server is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The VPN server subnet in CIDR notation (e.g., `10.100.0.1/24`). The first address is the server's tunnel IP.",
				Required:            true,
				CustomType:          cidrtypes.IPv4PrefixType{},
			},
			"dns": schema.SingleNestedAttribute{
				MarkdownDescription: "DNS configuration pushed to VPN clients.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether custom DNS servers are enabled for VPN clients. Defaults to `true` when `servers` is non-empty.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "DNS servers to push to VPN clients.",
						Optional:            true,
						ElementType:         types.StringType,
					},
				},
			},
			"wan": schema.SingleNestedAttribute{
				MarkdownDescription: "WAN binding configuration for the VPN server.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						MarkdownDescription: "Local WAN IP to bind the VPN server to. Use `any` to listen on all addresses.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("any"),
					},
					"interface": schema.StringAttribute{
						MarkdownDescription: "WAN interface to use for the VPN server (e.g., `wan`, `wan2`).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("wan"),
						Validators: []validator.String{
							wanInterfaceValidator,
						},
					},
				},
			},
			"radiusprofile_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the RADIUS profile to use for authentication. Applicable to L2TP and OpenVPN server types.",
				Optional:            true,
			},
			"wireguard": schema.SingleNestedAttribute{
				MarkdownDescription: "WireGuard VPN server configuration. Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"private_key": schema.StringAttribute{
						MarkdownDescription: "WireGuard private key for this server. If not specified, one will be generated by the controller.",
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
					},
					"public_key": schema.StringAttribute{
						MarkdownDescription: "WireGuard public key for this server. Computed from the private key.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "UDP port for the WireGuard server to listen on.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(51820),
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
				},
			},
			"l2tp": schema.SingleNestedAttribute{
				MarkdownDescription: "L2TP VPN server configuration. Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"allow_weak_ciphers": schema.BoolAttribute{
						MarkdownDescription: "Allow weak ciphers for L2TP connections.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"pre_shared_key": schema.StringAttribute{
						MarkdownDescription: "IPsec pre-shared key for L2TP. Required by the UniFi controller.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"openvpn": schema.SingleNestedAttribute{
				MarkdownDescription: "OpenVPN server configuration. Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"port": schema.Int64Attribute{
						MarkdownDescription: "Port for the OpenVPN server to listen on.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(1194),
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"mode": schema.StringAttribute{
						MarkdownDescription: "OpenVPN mode.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("server"),
						Validators: []validator.String{
							stringvalidator.OneOf("server", "site-to-site"),
						},
					},
					"encryption_cipher": schema.StringAttribute{
						MarkdownDescription: "Encryption cipher for OpenVPN.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("AES_256_GCM"),
						Validators: []validator.String{
							stringvalidator.OneOf("AES_256_GCM", "AES_256_CBC", "BF_CBC"),
						},
					},
					"server_crt": schema.StringAttribute{
						MarkdownDescription: "Server certificate generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"server_key": schema.StringAttribute{
						MarkdownDescription: "Server private key generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"dh_key": schema.StringAttribute{
						MarkdownDescription: "Diffie-Hellman parameters generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"shared_client_key": schema.StringAttribute{
						MarkdownDescription: "Shared client private key generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"shared_client_crt": schema.StringAttribute{
						MarkdownDescription: "Shared client certificate generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"auth_key": schema.StringAttribute{
						MarkdownDescription: "OpenVPN static authentication key generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ca_crt": schema.StringAttribute{
						MarkdownDescription: "CA certificate generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ca_key": schema.StringAttribute{
						MarkdownDescription: "CA private key generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *vpnServerResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
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

func (r *vpnServerResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdNetwork, err := r.client.CreateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating VPN Server",
			err.Error(),
		)
		return
	}

	var planData vpnServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, createdNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := vpnServerIdentityModel{ID: data.ID}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpnServerResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data vpnServerResourceModel

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
		network, err = r.client.GetNetwork(ctx, site, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VPN Server",
				"Could not read VPN server ID "+data.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else {
		network, err = r.client.GetNetworkByName(ctx, site, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VPN Server",
				"Could not read VPN server name "+data.Name.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	var priorState vpnServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.networkToModel(ctx, network, &data, site, &priorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := vpnServerIdentityModel{ID: data.ID}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpnServerResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	updatedNetwork, err := r.client.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VPN Server",
			err.Error(),
		)
		return
	}

	var planData vpnServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, updatedNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := vpnServerIdentityModel{ID: data.ID}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpnServerResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	name := data.Name.ValueString()
	err := r.client.DeleteNetwork(ctx, site, data.ID.ValueString(), name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VPN Server",
			err.Error(),
		)
		return
	}
}

func (r *vpnServerResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		req.ID = idParts[1]
	}

	if strings.HasPrefix(req.ID, "name=") {
		req.ID = strings.TrimPrefix(req.ID, "name=")
		resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
	} else if regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(req.ID) {
		idModel := vpnServerIdentityModel{ID: types.StringValue(req.ID)}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resource.ImportStatePassthroughWithIdentity(
			ctx,
			path.Root("id"),
			path.Root("id"),
			req,
			resp,
		)
	} else {
		resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
	}
}

// modelToNetwork converts from Terraform model to unifi.Network.
func (r *vpnServerResource) modelToNetwork(
	ctx context.Context,
	model *vpnServerResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:              model.Name.ValueStringPointer(),
		Purpose:           unifi.PurposeUserVPN,
		Enabled:           model.Enabled.ValueBool(),
		IPSubnet:          model.Subnet.ValueStringPointer(),
		SettingPreference: util.Ptr("manual"),
	}

	// Determine VPN type from which nested block is configured
	hasWireguard := !model.Wireguard.IsNull() && !model.Wireguard.IsUnknown()
	hasL2TP := !model.L2TP.IsNull() && !model.L2TP.IsUnknown()
	hasOpenVPN := !model.OpenVPN.IsNull() && !model.OpenVPN.IsUnknown()

	switch {
	case hasWireguard:
		network.VPNType = util.Ptr("wireguard-server")
	case hasL2TP:
		network.VPNType = util.Ptr("l2tp-server")
	case hasOpenVPN:
		network.VPNType = util.Ptr("openvpn-server")
	default:
		diags.AddError(
			"Missing VPN Type Configuration",
			"Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
		)
		return nil, diags
	}

	// Handle DNS configuration (shared across all VPN types)
	if !model.DNS.IsNull() && !model.DNS.IsUnknown() {
		var dns vpnServerDNSModel
		d := model.DNS.As(ctx, &dns, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !dns.Enabled.IsNull() && !dns.Enabled.IsUnknown() {
				network.DHCPDDNSEnabled = dns.Enabled.ValueBool()
			}

			if !dns.Servers.IsNull() && !dns.Servers.IsUnknown() {
				var dnsServers []string
				d := dns.Servers.ElementsAs(ctx, &dnsServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					if len(dnsServers) > 0 {
						network.DHCPDDNS1 = dnsServers[0]
						// Default enabled to true when servers are specified
						if dns.Enabled.IsNull() || dns.Enabled.IsUnknown() {
							network.DHCPDDNSEnabled = true
						}
					}
					if len(dnsServers) > 1 {
						network.DHCPDDNS2 = dnsServers[1]
					}
				}
			}
		}
	}

	// Handle WAN configuration (shared, but mapped to VPN-type-specific API fields)
	if !model.WAN.IsNull() && !model.WAN.IsUnknown() {
		var wan vpnServerWANModel
		d := model.WAN.As(ctx, &wan, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			switch {
			case hasWireguard:
				network.WireguardLocalWANIP = wan.IP.ValueStringPointer()
				network.WireguardInterface = wan.Interface.ValueStringPointer()
			case hasL2TP:
				network.L2TpLocalWANIP = wan.IP.ValueStringPointer()
				network.L2TpInterface = wan.Interface.ValueStringPointer()
			case hasOpenVPN:
				network.OpenVPNLocalWANIP = wan.IP.ValueStringPointer()
				network.OpenVPNInterface = wan.Interface.ValueStringPointer()
			}
		}
	}

	// RADIUS profile ID (applicable to L2TP and OpenVPN)
	if !model.RADIUSProfileID.IsNull() && !model.RADIUSProfileID.IsUnknown() {
		network.RADIUSProfileID = model.RADIUSProfileID.ValueStringPointer()
	}

	// Handle WireGuard-specific configuration
	if hasWireguard {
		var wireguard vpnServerWireguardModel
		d := model.Wireguard.As(ctx, &wireguard, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !wireguard.PrivateKey.IsNull() && !wireguard.PrivateKey.IsUnknown() {
				network.WireguardPrivateKey = wireguard.PrivateKey.ValueStringPointer()
			}
			network.LocalPort = wireguard.Port.ValueInt64Pointer()
		}
	}

	// Handle L2TP-specific configuration
	if hasL2TP {
		var l2tp vpnServerL2TPModel
		d := model.L2TP.As(ctx, &l2tp, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.L2TpAllowWeakCiphers = l2tp.AllowWeakCiphers.ValueBool()
			network.IPSecPreSharedKey = l2tp.PreSharedKey.ValueStringPointer()
		}
	}

	// Handle OpenVPN-specific configuration
	if hasOpenVPN {
		var openvpn vpnServerOpenVPNModel
		d := model.OpenVPN.As(ctx, &openvpn, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.LocalPort = openvpn.Port.ValueInt64Pointer()
			network.OpenVPNMode = openvpn.Mode.ValueStringPointer()
			network.OpenVPNEncryptionCipher = openvpn.EncryptionCipher.ValueStringPointer()

			// Send computed certificate/key fields back on updates
			network.ServerCrt = openvpn.ServerCrt.ValueStringPointer()
			network.ServerKey = openvpn.ServerKey.ValueStringPointer()
			network.DhKey = openvpn.DhKey.ValueStringPointer()
			network.SharedClientKey = openvpn.SharedClientKey.ValueStringPointer()
			network.SharedClientCrt = openvpn.SharedClientCrt.ValueStringPointer()
			network.AuthKey = openvpn.AuthKey.ValueStringPointer()
			network.CaCrt = openvpn.CaCrt.ValueStringPointer()
			network.CaKey = openvpn.CaKey.ValueStringPointer()
		}
	}

	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model.
func (r *vpnServerResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *vpnServerResourceModel,
	site string,
	priorState *vpnServerResourceModel,
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

	// Build DNS nested object
	{
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

		dnsValue := vpnServerDNSModel{
			Enabled: types.BoolValue(network.DHCPDDNSEnabled),
			Servers: dnsServersList,
		}
		var d diag.Diagnostics
		model.DNS, d = types.ObjectValueFrom(ctx, vpnServerDNSModel{}.AttributeTypes(), dnsValue)
		diags.Append(d...)
	}

	// Determine VPN type from the API response
	vpnType := ""
	if network.VPNType != nil {
		vpnType = *network.VPNType
	}

	// Build WAN nested object with VPN-type-specific API field mapping
	{
		var wanIP, wanIface *string
		switch vpnType {
		case "wireguard-server":
			wanIP = network.WireguardLocalWANIP
			wanIface = network.WireguardInterface
		case "l2tp-server":
			wanIP = network.L2TpLocalWANIP
			wanIface = network.L2TpInterface
		case "openvpn-server":
			wanIP = network.OpenVPNLocalWANIP
			wanIface = network.OpenVPNInterface
		}

		wanValue := vpnServerWANModel{
			IP:        types.StringPointerValue(wanIP),
			Interface: types.StringPointerValue(wanIface),
		}
		var d diag.Diagnostics
		model.WAN, d = types.ObjectValueFrom(ctx, vpnServerWANModel{}.AttributeTypes(), wanValue)
		diags.Append(d...)
	}

	// RADIUS profile ID
	if network.RADIUSProfileID != nil && *network.RADIUSProfileID != "" {
		model.RADIUSProfileID = types.StringPointerValue(network.RADIUSProfileID)
	} else {
		model.RADIUSProfileID = types.StringNull()
	}

	// Build VPN-type-specific nested objects
	switch vpnType {
	case "wireguard-server":
		// Preserve private key from prior state if the API doesn't return it
		privateKeyVal := types.StringPointerValue(network.WireguardPrivateKey)
		if (privateKeyVal.IsNull() || privateKeyVal.ValueString() == "") &&
			priorState != nil && !priorState.Wireguard.IsNull() && !priorState.Wireguard.IsUnknown() {
			var priorWG vpnServerWireguardModel
			d := priorState.Wireguard.As(ctx, &priorWG, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if !diags.HasError() {
				privateKeyVal = priorWG.PrivateKey
			}
		}

		strPtrToType := func(ptr *string) types.String {
			if ptr == nil || *ptr == "" {
				return types.StringNull()
			}
			return types.StringValue(*ptr)
		}

		wireguardValue := vpnServerWireguardModel{
			PrivateKey: privateKeyVal,
			PublicKey:  strPtrToType(network.WireguardPublicKey),
			Port:       types.Int64PointerValue(network.LocalPort),
		}
		var d diag.Diagnostics
		model.Wireguard, d = types.ObjectValueFrom(
			ctx,
			vpnServerWireguardModel{}.AttributeTypes(),
			wireguardValue,
		)
		diags.Append(d...)
		model.L2TP = types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes())
		model.OpenVPN = types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes())

	case "l2tp-server":
		// Preserve pre-shared key from prior state since the API does not return it
		pskVal := types.StringPointerValue(network.IPSecPreSharedKey)
		if (pskVal.IsNull() || pskVal.ValueString() == "") &&
			priorState != nil && !priorState.L2TP.IsNull() && !priorState.L2TP.IsUnknown() {
			var priorL2TP vpnServerL2TPModel
			d := priorState.L2TP.As(ctx, &priorL2TP, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if !diags.HasError() {
				pskVal = priorL2TP.PreSharedKey
			}
		}

		l2tpValue := vpnServerL2TPModel{
			AllowWeakCiphers: types.BoolValue(network.L2TpAllowWeakCiphers),
			PreSharedKey:     pskVal,
		}
		var d diag.Diagnostics
		model.L2TP, d = types.ObjectValueFrom(ctx, vpnServerL2TPModel{}.AttributeTypes(), l2tpValue)
		diags.Append(d...)
		model.Wireguard = types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())
		model.OpenVPN = types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes())

	case "openvpn-server":
		openvpnValue := vpnServerOpenVPNModel{
			Port:             types.Int64PointerValue(network.LocalPort),
			Mode:             types.StringPointerValue(network.OpenVPNMode),
			EncryptionCipher: types.StringPointerValue(network.OpenVPNEncryptionCipher),
			ServerCrt:        types.StringPointerValue(network.ServerCrt),
			ServerKey:        types.StringPointerValue(network.ServerKey),
			DhKey:            types.StringPointerValue(network.DhKey),
			SharedClientKey:  types.StringPointerValue(network.SharedClientKey),
			SharedClientCrt:  types.StringPointerValue(network.SharedClientCrt),
			AuthKey:          types.StringPointerValue(network.AuthKey),
			CaCrt:            types.StringPointerValue(network.CaCrt),
			CaKey:            types.StringPointerValue(network.CaKey),
		}
		var d diag.Diagnostics
		model.OpenVPN, d = types.ObjectValueFrom(
			ctx,
			vpnServerOpenVPNModel{}.AttributeTypes(),
			openvpnValue,
		)
		diags.Append(d...)
		model.Wireguard = types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())
		model.L2TP = types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes())

	default:
		// Unknown VPN type — null out all type-specific blocks
		model.Wireguard = types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())
		model.L2TP = types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes())
		model.OpenVPN = types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes())
	}

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *vpnServerResource) ListResourceConfigSchema(
	ctx context.Context,
	req list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List VPN servers in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list VPN servers from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`.",
							Required:            true,
						},
						"value": listschema.StringAttribute{
							MarkdownDescription: "The value to filter by.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

// List implements [list.ListResource].
func (r *vpnServerResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config vpnServerListConfigModel

	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Process filter blocks.
	var filters []vpnServerListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	networks, err := r.client.ListNetwork(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing VPN Servers", "Could not list networks: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, network := range networks {
			// Filter by purpose: only remote-user-vpn networks.
			if network.Purpose != unifi.PurposeUserVPN {
				continue
			}

			// Apply name filter if specified.
			if nameFilter, ok := postFilters["name"]; ok {
				if network.Name == nil || *network.Name != nameFilter {
					continue
				}
			}

			result := req.NewListResult(ctx)
			if network.Name != nil {
				result.DisplayName = *network.Name
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(network.ID),
				)...,
			)

			// Convert to model.
			var model vpnServerResourceModel
			result.Diagnostics.Append(
				r.networkToModel(ctx, &network, &model, site, &vpnServerResourceModel{})...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
