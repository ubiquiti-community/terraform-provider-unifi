package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &siteToSiteVPNResource{}
	_ resource.ResourceWithImportState  = &siteToSiteVPNResource{}
	_ resource.ResourceWithIdentity     = &siteToSiteVPNResource{}
	_ resource.ResourceWithUpgradeState = &siteToSiteVPNResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &siteToSiteVPNResource{}
	_ list.ListResourceWithConfigure = &siteToSiteVPNResource{}
)

func NewSiteToSiteVPNResource() resource.Resource {
	return &siteToSiteVPNResource{}
}

func NewSiteToSiteVPNListResource() list.ListResource {
	return &siteToSiteVPNResource{}
}

// siteToSiteVPNResource defines the resource implementation.
type siteToSiteVPNResource struct {
	client *Client
}

// siteToSiteVPNResourceModel describes the resource data model. It wraps a
// purpose="site-vpn", vpn_type="ipsec-vpn" network — the UniFi manual
// site-to-site IPsec VPN.
type siteToSiteVPNResourceModel struct {
	ID             types.String         `tfsdk:"id"`
	Site           types.String         `tfsdk:"site"`
	Name           types.String         `tfsdk:"name"`
	Enabled        types.Bool           `tfsdk:"enabled"`
	Interface      types.String         `tfsdk:"interface"`
	PeerIP         iptypes.IPv4Address  `tfsdk:"peer_ip"`
	LocalIP        iptypes.IPv4Address  `tfsdk:"local_ip"`
	KeyExchange    types.String         `tfsdk:"key_exchange"`
	PreSharedKey   types.String         `tfsdk:"pre_shared_key"`
	PreSharedKeyWO types.String         `tfsdk:"pre_shared_key_wo"`
	RemoteSubnets  types.List           `tfsdk:"remote_subnets"`
	Profile        types.String         `tfsdk:"profile"`
	IKEEncryption  types.String         `tfsdk:"ike_encryption"`
	IKEHash        types.String         `tfsdk:"ike_hash"`
	IKEDhGroup     types.Int64          `tfsdk:"ike_dh_group"`
	IKELifetime    timetypes.GoDuration `tfsdk:"ike_lifetime"`
	ESPEncryption  types.String         `tfsdk:"esp_encryption"`
	ESPHash        types.String         `tfsdk:"esp_hash"`
	ESPDhGroup     types.Int64          `tfsdk:"esp_dh_group"`
	ESPLifetime    timetypes.GoDuration `tfsdk:"esp_lifetime"`
	PFS            types.Bool           `tfsdk:"pfs"`
	DynamicRouting types.Bool           `tfsdk:"dynamic_routing"`
	RouteDistance  types.Int64          `tfsdk:"route_distance"`
}

// siteToSiteVPNListConfigModel describes the list configuration model.
type siteToSiteVPNListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// siteToSiteVPNListFilterModel represents a single name/value filter entry.
type siteToSiteVPNListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *siteToSiteVPNResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_site_to_site_vpn"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *siteToSiteVPNResource) IdentitySchema(
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

func (r *siteToSiteVPNResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	cipherValues := []string{"aes128", "aes192", "aes256", "3des"}
	hashValues := []string{"sha1", "md5", "sha256", "sha384", "sha512"}

	resp.Schema = schema.Schema{
		// v1: ike_lifetime/esp_lifetime changed from Int64 (seconds) to GoDuration.
		Version: 1,
		MarkdownDescription: "Manages a manual site-to-site IPsec VPN (the UniFi " +
			"`Settings → VPN → Site-to-Site` network, `purpose = site-vpn`, " +
			"`vpn_type = ipsec-vpn`). The advanced IKE/ESP attributes only apply " +
			"when `profile = customized`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the site-to-site VPN network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the VPN with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the site-to-site VPN.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the tunnel is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"interface": schema.StringAttribute{
				MarkdownDescription: "The local WAN interface the tunnel binds to (e.g. `wan`, `wan2`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("wan"),
				Validators: []validator.String{
					stringvalidator.OneOf("wan", "wan2"),
				},
			},
			"peer_ip": schema.StringAttribute{
				MarkdownDescription: "The public IP address of the remote VPN gateway (peer).",
				CustomType:          iptypes.IPv4AddressType{},
				Required:            true,
				Validators: []validator.String{
					validators.IPv4Validator(),
				},
			},
			"local_ip": schema.StringAttribute{
				MarkdownDescription: "The local IP used for the tunnel. Defaults to the WAN address when omitted.",
				CustomType:          iptypes.IPv4AddressType{},
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key_exchange": schema.StringAttribute{
				MarkdownDescription: "IKE key-exchange version. One of `ikev1` or `ikev2`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("ikev2"),
				Validators: []validator.String{
					stringvalidator.OneOf("ikev1", "ikev2"),
				},
			},
			"pre_shared_key": schema.StringAttribute{
				MarkdownDescription: "The IPsec pre-shared key. Stored in state — use " +
					"`pre_shared_key_wo` to avoid persisting the secret.",
				Optional:  true,
				Sensitive: true,
			},
			"pre_shared_key_wo": schema.StringAttribute{
				MarkdownDescription: "Write-only equivalent of `pre_shared_key` (Terraform 1.11+). " +
					"Used at apply time but never written to state, so it can be sourced from " +
					"an ephemeral resource (e.g. a Vault secret). Mutually exclusive with " +
					"`pre_shared_key`.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("pre_shared_key")),
				},
			},
			"remote_subnets": schema.ListAttribute{
				MarkdownDescription: "The remote site's subnets reachable through the tunnel (CIDR).",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(validators.CIDRValidator()),
				},
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: "IPsec profile. One of `customized`, `azure_dynamic`, or " +
					"`azure_static`. Set to `customized` to tune the IKE/ESP attributes below; " +
					"the controller may derive the ESP values from the IKE ones.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("customized", "azure_dynamic", "azure_static"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ike_encryption": schema.StringAttribute{
				MarkdownDescription: "IKE (phase 1) encryption. Only used when `profile = customized`.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(cipherValues...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ike_hash": schema.StringAttribute{
				MarkdownDescription: "IKE (phase 1) hash. Only used when `profile = customized`.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(hashValues...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ike_dh_group": schema.Int64Attribute{
				MarkdownDescription: "IKE (phase 1) Diffie-Hellman group. Only used when `profile = customized`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ike_lifetime": schema.StringAttribute{
				MarkdownDescription: "IKE (phase 1) security-association lifetime, as a Go " +
					"duration string (e.g. `8h`, `28800s`).",
				CustomType:    timetypes.GoDurationType{},
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"esp_encryption": schema.StringAttribute{
				MarkdownDescription: "ESP (phase 2) encryption. Only used when `profile = customized`.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(cipherValues...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"esp_hash": schema.StringAttribute{
				MarkdownDescription: "ESP (phase 2) hash. Only used when `profile = customized`.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf(hashValues...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"esp_dh_group": schema.Int64Attribute{
				MarkdownDescription: "ESP (phase 2) Diffie-Hellman group (PFS). Only used when `profile = customized`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"esp_lifetime": schema.StringAttribute{
				MarkdownDescription: "ESP (phase 2) security-association lifetime, as a Go " +
					"duration string (e.g. `1h`, `3600s`).",
				CustomType:    timetypes.GoDurationType{},
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pfs": schema.BoolAttribute{
				MarkdownDescription: "Whether Perfect Forward Secrecy is enabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"dynamic_routing": schema.BoolAttribute{
				MarkdownDescription: "Whether IPsec dynamic routing is enabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"route_distance": schema.Int64Attribute{
				MarkdownDescription: "The route distance (administrative metric) for tunnel routes (1-255).",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 255)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

// UpgradeState migrates v0 state (ike_lifetime/esp_lifetime stored as integer
// seconds) to v1 (GoDuration strings).
func (r *siteToSiteVPNResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(
				ctx context.Context,
				req resource.UpgradeStateRequest,
				resp *resource.UpgradeStateResponse,
			) {
				if req.RawState == nil {
					return
				}
				dv, err := util.UpgradeDurationRawState(
					schemaType,
					req.RawState.JSON,
					func(state map[string]any) {
						util.SetDurationField(state, "ike_lifetime", time.Second)
						util.SetDurationField(state, "esp_lifetime", time.Second)
					},
				)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to upgrade site-to-site VPN state",
						err.Error(),
					)
					return
				}
				resp.DynamicValue = dv
			},
		},
	}
}

func (r *siteToSiteVPNResource) ConfigValidators(
	ctx context.Context,
) []resource.ConfigValidator {
	return nil
}

func (r *siteToSiteVPNResource) Configure(
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

func (r *siteToSiteVPNResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data siteToSiteVPNResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	usedWO := r.applyPreSharedKeyWO(ctx, req.Config, network, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	site := r.siteOrDefault(data.Site)

	created, err := r.client.CreateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Site-to-Site VPN", err.Error())
		return
	}

	resp.Diagnostics.Append(r.networkToModel(ctx, created, &data, site)...)
	if usedWO {
		data.PreSharedKey = types.StringNull()
	}
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *siteToSiteVPNResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data siteToSiteVPNResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := r.siteOrDefault(data.Site)

	network, err := r.client.GetNetwork(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Site-to-Site VPN",
			"Could not read network "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.networkToModel(ctx, network, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *siteToSiteVPNResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data siteToSiteVPNResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	network.ID = data.ID.ValueString()

	usedWO := r.applyPreSharedKeyWO(ctx, req.Config, network, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	site := r.siteOrDefault(data.Site)

	updated, err := r.client.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Site-to-Site VPN", err.Error())
		return
	}

	resp.Diagnostics.Append(r.networkToModel(ctx, updated, &data, site)...)
	if usedWO {
		data.PreSharedKey = types.StringNull()
	}
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *siteToSiteVPNResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data siteToSiteVPNResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := r.siteOrDefault(data.Site)

	err := r.client.DeleteNetwork(ctx, site, data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Site-to-Site VPN", err.Error())
		return
	}
}

func (r *siteToSiteVPNResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import format: "site:id" or just "id" for the default site. The pre-shared
	// key is not recovered on import (write the secret in config and re-apply).
	idParts := strings.Split(req.ID, ":")
	switch len(idParts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'site:id' or 'id'",
		)
	}
}

func (r *siteToSiteVPNResource) siteOrDefault(site types.String) string {
	if s := site.ValueString(); s != "" {
		return s
	}
	return r.client.Site
}

// applyPreSharedKeyWO reads the write-only pre_shared_key_wo from config and, if
// set, uses it for the API call. Returns true when the write-only key was used
// (so the caller can keep pre_shared_key null in state).
func (r *siteToSiteVPNResource) applyPreSharedKeyWO(
	ctx context.Context,
	config tfsdk.Config,
	network *unifi.Network,
	diags *diag.Diagnostics,
) bool {
	var wo types.String
	diags.Append(config.GetAttribute(ctx, path.Root("pre_shared_key_wo"), &wo)...)
	if diags.HasError() || wo.IsNull() || wo.IsUnknown() {
		return false
	}
	network.IPSecPreSharedKey = util.Ptr(wo.ValueString())
	return true
}

// modelToNetwork converts the Terraform model to the go-unifi Network struct.
// The pre-shared key from config (pre_shared_key) is set here; the write-only
// variant is applied separately in Create/Update.
func (r *siteToSiteVPNResource) modelToNetwork(
	ctx context.Context,
	model *siteToSiteVPNResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Purpose:             unifi.PurposeSiteVPN,
		Name:                util.Ptr(model.Name.ValueString()),
		Enabled:             model.Enabled.ValueBool(),
		VPNType:             util.Ptr("ipsec-vpn"),
		IPSecInterface:      optStr(model.Interface),
		IPSecPeerIP:         optStr(model.PeerIP),
		IPSecLocalIP:        optStr(model.LocalIP),
		IPSecKeyExchange:    optStr(model.KeyExchange),
		IPSecProfile:        optStr(model.Profile),
		IPSecEncryption:     optStr(model.IKEEncryption),
		IPSecHash:           optStr(model.IKEHash),
		IPSecDhGroup:        optInt64(model.IKEDhGroup),
		IPSecIkeLifetime:    util.DurationUnitsPtr(model.IKELifetime, time.Second),
		IPSecEspEncryption:  optStr(model.ESPEncryption),
		IPSecEspHash:        optStr(model.ESPHash),
		IPSecEspDhGroup:     optInt64(model.ESPDhGroup),
		IPSecEspLifetime:    util.DurationUnitsPtr(model.ESPLifetime, time.Second),
		IPSecPfs:            model.PFS.ValueBool(),
		IPSecDynamicRouting: model.DynamicRouting.ValueBool(),
		RouteDistance:       optInt64(model.RouteDistance),
	}

	if !model.PreSharedKey.IsNull() && !model.PreSharedKey.IsUnknown() {
		network.IPSecPreSharedKey = model.PreSharedKey.ValueStringPointer()
	}

	if !model.RemoteSubnets.IsNull() && !model.RemoteSubnets.IsUnknown() {
		diags.Append(model.RemoteSubnets.ElementsAs(ctx, &network.RemoteVPNSubnets, false)...)
	}

	return network, diags
}

// networkToModel maps the API network back onto the model. It deliberately does
// not touch pre_shared_key / pre_shared_key_wo: the controller echoes the secret
// on read, so re-importing it would cause perpetual diffs for write-only users
// and pointless drift for the stored variant. The configured value is preserved.
func (r *siteToSiteVPNResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *siteToSiteVPNResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	if network.Name != nil {
		model.Name = types.StringValue(*network.Name)
	}
	model.Enabled = types.BoolValue(network.Enabled)
	model.Interface = stringPtrOrNull(network.IPSecInterface)
	model.PeerIP = util.IPv4PtrValueOrNull(network.IPSecPeerIP)
	model.LocalIP = util.IPv4PtrValueOrNull(network.IPSecLocalIP)
	model.KeyExchange = stringPtrOrNull(network.IPSecKeyExchange)
	model.Profile = stringPtrOrNull(network.IPSecProfile)
	model.IKEEncryption = stringPtrOrNull(network.IPSecEncryption)
	model.IKEHash = stringPtrOrNull(network.IPSecHash)
	model.IKEDhGroup = types.Int64PointerValue(network.IPSecDhGroup)
	model.IKELifetime = util.DurationPtrValue(network.IPSecIkeLifetime, time.Second)
	model.ESPEncryption = stringPtrOrNull(network.IPSecEspEncryption)
	model.ESPHash = stringPtrOrNull(network.IPSecEspHash)
	model.ESPDhGroup = types.Int64PointerValue(network.IPSecEspDhGroup)
	model.ESPLifetime = util.DurationPtrValue(network.IPSecEspLifetime, time.Second)
	model.PFS = types.BoolValue(network.IPSecPfs)
	model.DynamicRouting = types.BoolValue(network.IPSecDynamicRouting)
	model.RouteDistance = types.Int64PointerValue(network.RouteDistance)

	subnets, subnetDiags := types.ListValueFrom(ctx, types.StringType, network.RemoteVPNSubnets)
	diags.Append(subnetDiags...)
	model.RemoteSubnets = subnets

	return diags
}

// optStr returns the string pointer for an optional attribute, or nil when the
// value is null, unknown, or empty — so the marshaler omits it rather than
// sending "" (which the controller rejects for IP/enum fields).
// optStr accepts any framework string-backed value (types.String or the custom
// nettypes values, which all embed basetypes.StringValue) and returns nil for
// null/unknown/empty.
func optStr(s interface {
	IsNull() bool
	IsUnknown() bool
	ValueString() string
},
) *string {
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return nil
	}
	v := s.ValueString()
	return &v
}

// optInt64 returns the int64 pointer for an optional attribute, or nil when the
// value is null, unknown, or zero — so the marshaler omits it rather than
// sending 0 (which the controller rejects for the lifetime/group/distance fields).
func optInt64(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() || v.ValueInt64() == 0 {
		return nil
	}
	return v.ValueInt64Pointer()
}

// stringPtrOrNull maps an API string pointer to types.String, treating nil and the
// empty string as null (the provider's null-vs-empty convention).
func stringPtrOrNull(v *string) types.String {
	if v == nil || *v == "" {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *siteToSiteVPNResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List site-to-site VPN networks in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list site-to-site VPN networks from.",
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
func (r *siteToSiteVPNResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config siteToSiteVPNListConfigModel

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
	var filters []siteToSiteVPNListFilterModel
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
		d.AddError(
			"Error Listing Site-to-Site VPNs",
			"Could not list site-to-site VPNs: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, network := range networks {
			// Filter to only site-to-site IPsec VPN networks.
			if network.Purpose != unifi.PurposeSiteVPN ||
				network.VPNType == nil || *network.VPNType != "ipsec-vpn" {
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
			var model siteToSiteVPNResourceModel
			result.Diagnostics.Append(r.networkToModel(ctx, &network, &model, site)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
