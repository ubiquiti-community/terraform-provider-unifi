package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	_ resource.Resource                = &wanResource{}
	_ resource.ResourceWithImportState = &wanResource{}
)

func NewWANResource() resource.Resource {
	return &wanResource{}
}

// wanResource defines the resource implementation.
type wanResource struct {
	client *Client
}

// wanResourceModel describes the resource data model.
type wanResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
	Name types.String `tfsdk:"name"`

	// WAN Type Settings
	Type   types.String `tfsdk:"type"`
	TypeV6 types.String `tfsdk:"type_v6"`

	// VLAN Settings
	Vlan types.Object `tfsdk:"vlan"`

	// QoS Settings
	EgressQoS types.Object `tfsdk:"egress_qos"`
	DHCPCoS   types.Int64  `tfsdk:"dhcp_cos"`
	DHCPV6CoS types.Int64  `tfsdk:"dhcpv6_cos"`

	// DNS Settings
	DNS1              types.String `tfsdk:"dns1"`
	DNS2              types.String `tfsdk:"dns2"`
	IPv6DNS1          types.String `tfsdk:"ipv6_dns1"`
	IPv6DNS2          types.String `tfsdk:"ipv6_dns2"`
	DNSPreference     types.String `tfsdk:"dns_preference"`
	IPv6DNSPreference types.String `tfsdk:"ipv6_dns_preference"`

	// DHCPv6 Settings
	DHCPV6PDSize     types.Int64 `tfsdk:"dhcpv6_pd_size"`
	DHCPV6PDSizeAuto types.Bool  `tfsdk:"dhcpv6_pd_size_auto"`
	DHCPV6Options    types.List  `tfsdk:"dhcpv6_options"`

	// IPv6 Settings
	IPv6WANDelegationType types.String `tfsdk:"ipv6_wan_delegation_type"`

	// Smart Queue Settings
	SmartQ types.Object `tfsdk:"smartq"`

	// UPnP Settings
	UPnPEnabled       types.Bool   `tfsdk:"upnp_enabled"`
	UPnPWANInterface  types.String `tfsdk:"upnp_wan_interface"`
	UPnPNatPMPEnabled types.Bool   `tfsdk:"upnp_nat_pmp_enabled"`
	UPnPSecureMode    types.Bool   `tfsdk:"upnp_secure_mode"`

	// Load Balance Settings
	LoadBalanceType   types.String `tfsdk:"load_balance_type"`
	LoadBalanceWeight types.Int64  `tfsdk:"load_balance_weight"`
	FailoverPriority  types.Int64  `tfsdk:"failover_priority"`

	// IGMP Settings
	IGMPProxyFor      types.String `tfsdk:"igmp_proxy_for"`
	IGMPProxyUpstream types.Bool   `tfsdk:"igmp_proxy_upstream"`

	// Additional Settings
	ReportWANEvent types.Bool `tfsdk:"report_wan_event"`
	Enabled        types.Bool `tfsdk:"enabled"`
	DHCPOptions    types.List `tfsdk:"dhcp_options"`
	IPAliases      types.List `tfsdk:"ip_aliases"`

	// Provider Capabilities
	ProviderCapabilities types.Object `tfsdk:"provider_capabilities"`
}

// vlanModel describes the VLAN configuration.
type vlanModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	ID      types.Int64 `tfsdk:"id"`
}

func (m vlanModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"id":      types.Int64Type,
	}
}

// egressQosModel describes the Egress QoS configuration.
type egressQosModel struct {
	Enabled  types.Bool  `tfsdk:"enabled"`
	Priority types.Int64 `tfsdk:"priority"`
}

func (m egressQosModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":  types.BoolType,
		"priority": types.Int64Type,
	}
}

// smartqModel describes the Smart Queue configuration.
type smartqModel struct {
	Enabled  types.Bool  `tfsdk:"enabled"`
	UpRate   types.Int64 `tfsdk:"up_rate"`
	DownRate types.Int64 `tfsdk:"down_rate"`
}

func (m smartqModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":   types.BoolType,
		"up_rate":   types.Int64Type,
		"down_rate": types.Int64Type,
	}
}

// providerCapabilitiesModel describes the provider capabilities nested object.
type providerCapabilitiesModel struct {
	DownloadKbps types.Int64 `tfsdk:"download_kilobits_per_second"`
	UploadKbps   types.Int64 `tfsdk:"upload_kilobits_per_second"`
}

func (m providerCapabilitiesModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"download_kilobits_per_second": types.Int64Type,
		"upload_kilobits_per_second":   types.Int64Type,
	}
}

// dhcpOptionModel describes a DHCPv6 option.
type dhcpOptionModel struct {
	OptionNumber types.Int64  `tfsdk:"option_number"`
	Value        types.String `tfsdk:"value"`
}

func (m dhcpOptionModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"option_number": types.Int64Type,
		"value":         types.StringType,
	}
}

func (r *wanResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_wan"
}

func (r *wanResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "WAN network resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the WAN network",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the site to associate the WAN network with",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the WAN network",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("dhcp"),
				MarkdownDescription: "The WAN type (dhcp, static, pppoe)",
				Validators: []validator.String{
					stringvalidator.OneOf("dhcp", "static", "pppoe", "disabled"),
				},
			},
			"type_v6": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The IPv6 WAN type (dhcpv6, static, disabled)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("dhcpv6", "static", "disabled"),
				},
			},
			"vlan": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "VLAN configuration",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether VLAN is enabled",
					},
					"id": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(0),
						MarkdownDescription: "The VLAN ID",
						Validators: []validator.Int64{
							int64validator.Between(0, 4094),
						},
					},
				},
			},
			"egress_qos": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Egress QoS configuration",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether egress QoS is enabled",
					},
					"priority": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(0),
						MarkdownDescription: "Egress QoS priority",
						Validators: []validator.Int64{
							int64validator.Between(0, 7),
						},
					},
				},
			},
			"dhcp_cos": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DHCP Class of Service",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(0, 7),
				},
			},
			"dhcpv6_cos": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DHCPv6 Class of Service",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(0, 7),
				},
			},
			"dns1": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Primary DNS server",
				Validators: []validator.String{
					validators.IPv4Validator(),
				},
			},
			"dns2": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Secondary DNS server",
				Validators: []validator.String{
					validators.IPv4Validator(),
				},
			},
			"ipv6_dns1": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Primary IPv6 DNS server",
			},
			"ipv6_dns2": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Secondary IPv6 DNS server",
			},
			"dns_preference": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DNS preference (auto, manual)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"ipv6_dns_preference": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "IPv6 DNS preference (auto, manual)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"dhcpv6_pd_size": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DHCPv6 prefix delegation size",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(48, 64),
				},
			},
			"dhcpv6_pd_size_auto": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether DHCPv6 PD size is automatic",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dhcpv6_options": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "DHCPv6 options",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"option_number": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "DHCPv6 option number (1, 11, 15, 16, or 17)",
							Validators: []validator.Int64{
								int64validator.OneOf(1, 11, 15, 16, 17),
							},
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "DHCPv6 option value",
						},
					},
				},
			},
			"ipv6_wan_delegation_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "IPv6 WAN delegation type (pd, single_network, none)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("pd", "single_network", "none"),
				},
			},
			"smartq": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Smart Queue configuration",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether Smart Queue is enabled",
					},
					"up_rate": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Smart Queue upload rate in kbps",
					},
					"down_rate": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Smart Queue download rate in kbps",
					},
				},
			},
			"upnp_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether UPnP is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"upnp_wan_interface": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UPnP WAN interface",
			},
			"upnp_nat_pmp_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether UPnP NAT-PMP is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"upnp_secure_mode": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether UPnP secure mode is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"load_balance_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Load balance type (failover-only, weighted)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("failover-only", "weighted"),
				},
			},
			"load_balance_weight": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Load balance weight",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"failover_priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Failover priority",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"igmp_proxy_for": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "IGMP proxy for (none, lan, guest)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("none", "lan", "guest"),
				},
			},
			"igmp_proxy_upstream": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether IGMP proxy upstream is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"report_wan_event": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to report WAN events",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the WAN network is enabled",
			},
			"dhcp_options": schema.ListAttribute{
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"option_number": types.Int64Type,
						"value":         types.StringType,
					},
				},
				Optional:            true,
				MarkdownDescription: "DHCP options",
			},
			"ip_aliases": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "IP aliases",
			},
			"provider_capabilities": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "WAN provider capabilities",
				Attributes: map[string]schema.Attribute{
					"download_kilobits_per_second": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Download speed in kilobits per second",
					},
					"upload_kilobits_per_second": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Upload speed in kilobits per second",
					},
				},
			},
		},
	}
}

func (r *wanResource) Configure(
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

func (r *wanResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan wanResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.Network
	network, diags := r.modelToNetwork(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the network
	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdNetwork, err := r.client.CreateNetwork(ctx, site, network)
	if err != nil {
		// If a WAN configuration already exists for this network group,
		// adopt the existing one and update it with the planned configuration.
		if strings.Contains(err.Error(), "WanConfigurationForNetworkGroupAlreadyExists") {
			createdNetwork, err = r.adoptExistingWAN(ctx, site, network)
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("Unable to adopt existing WAN network, got error: %s", err),
				)
				return
			}

			// For adoption: read the existing WAN's state, then overlay with
			// only the values the user explicitly configured (from req.Config,
			// which has null for fields not set in HCL, unlike req.Plan which
			// has defaults applied).
			var config wanResourceModel
			resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Start with the adopted WAN's actual state
			var state wanResourceModel
			diags = r.networkToModel(ctx, createdNetwork, &state, site)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Overlay explicit config values onto the API state
			r.overlayConfig(&state, &config, &plan)

			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create WAN network, got error: %s", err),
		)
		return
	}

	// For normal creation, read the API response into a fresh model
	// (not the plan, which may have unknown values for Computed fields).
	var config wanResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state wanResourceModel
	diags = r.networkToModel(ctx, createdNetwork, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overlay explicit config values onto the API state
	r.overlayConfig(&state, &config, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// adoptExistingWAN finds the existing WAN network in the given network group and updates it.
func (r *wanResource) adoptExistingWAN(
	ctx context.Context,
	site string,
	network *unifi.Network,
) (*unifi.Network, error) {
	networks, err := r.client.ListNetwork(ctx, site)
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}

	var existing *unifi.Network
	for _, n := range networks {
		if n.Purpose == unifi.PurposeWAN && n.WANNetworkGroup != nil &&
			*n.WANNetworkGroup == "WAN" {
			existing = &n
			break
		}
	}
	if existing == nil {
		return nil, fmt.Errorf("existing WAN network not found despite creation conflict")
	}

	network.ID = existing.ID
	return r.client.UpdateNetwork(ctx, site, network)
}

// overlayConfig applies only explicitly-configured values from config onto state.
// The config has null for fields not set in HCL; plan has defaults applied.
// For non-null config fields, we use the plan value (which includes any validation/transform).
// For null config fields, we keep the state value (from the API).
func (r *wanResource) overlayConfig(
	state *wanResourceModel,
	config *wanResourceModel,
	plan *wanResourceModel,
) {
	if !config.Name.IsNull() {
		state.Name = plan.Name
	}
	if !config.Type.IsNull() {
		state.Type = plan.Type
	}
	if !config.TypeV6.IsNull() {
		state.TypeV6 = plan.TypeV6
	}
	if !config.Vlan.IsNull() {
		state.Vlan = plan.Vlan
	}
	if !config.EgressQoS.IsNull() {
		state.EgressQoS = plan.EgressQoS
	}
	if !config.DHCPCoS.IsNull() {
		state.DHCPCoS = plan.DHCPCoS
	}
	if !config.DHCPV6CoS.IsNull() {
		state.DHCPV6CoS = plan.DHCPV6CoS
	}
	if !config.DNS1.IsNull() {
		state.DNS1 = plan.DNS1
	}
	if !config.DNS2.IsNull() {
		state.DNS2 = plan.DNS2
	}
	if !config.IPv6DNS1.IsNull() {
		state.IPv6DNS1 = plan.IPv6DNS1
	}
	if !config.IPv6DNS2.IsNull() {
		state.IPv6DNS2 = plan.IPv6DNS2
	}
	if !config.DNSPreference.IsNull() {
		state.DNSPreference = plan.DNSPreference
	}
	if !config.IPv6DNSPreference.IsNull() {
		state.IPv6DNSPreference = plan.IPv6DNSPreference
	}
	if !config.DHCPV6PDSize.IsNull() {
		state.DHCPV6PDSize = plan.DHCPV6PDSize
	}
	if !config.DHCPV6PDSizeAuto.IsNull() {
		state.DHCPV6PDSizeAuto = plan.DHCPV6PDSizeAuto
	}
	if !config.IPv6WANDelegationType.IsNull() {
		state.IPv6WANDelegationType = plan.IPv6WANDelegationType
	}
	if !config.SmartQ.IsNull() {
		state.SmartQ = plan.SmartQ
	}
	if !config.UPnPEnabled.IsNull() {
		state.UPnPEnabled = plan.UPnPEnabled
	}
	if !config.UPnPWANInterface.IsNull() {
		state.UPnPWANInterface = plan.UPnPWANInterface
	}
	if !config.UPnPNatPMPEnabled.IsNull() {
		state.UPnPNatPMPEnabled = plan.UPnPNatPMPEnabled
	}
	if !config.UPnPSecureMode.IsNull() {
		state.UPnPSecureMode = plan.UPnPSecureMode
	}
	if !config.LoadBalanceType.IsNull() {
		state.LoadBalanceType = plan.LoadBalanceType
	}
	if !config.LoadBalanceWeight.IsNull() {
		state.LoadBalanceWeight = plan.LoadBalanceWeight
	}
	if !config.FailoverPriority.IsNull() {
		state.FailoverPriority = plan.FailoverPriority
	}
	if !config.IGMPProxyFor.IsNull() {
		state.IGMPProxyFor = plan.IGMPProxyFor
	}
	if !config.IGMPProxyUpstream.IsNull() {
		state.IGMPProxyUpstream = plan.IGMPProxyUpstream
	}
	if !config.ReportWANEvent.IsNull() {
		state.ReportWANEvent = plan.ReportWANEvent
	}
	if !config.Enabled.IsNull() {
		state.Enabled = plan.Enabled
	}
	if !config.DHCPOptions.IsNull() {
		state.DHCPOptions = plan.DHCPOptions
	}
	if !config.DHCPV6Options.IsNull() {
		state.DHCPV6Options = plan.DHCPV6Options
	}
	if !config.IPAliases.IsNull() {
		state.IPAliases = plan.IPAliases
	}
	if !config.ProviderCapabilities.IsNull() {
		state.ProviderCapabilities = plan.ProviderCapabilities
	}
}

func (r *wanResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state wanResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	var network *unifi.Network
	var err error

	if state.ID.IsNull() || state.ID.IsUnknown() {
		network, err = r.client.GetNetworkByName(ctx, site, state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read WAN network, got error: %s", err),
			)
			return
		}
	} else {
		// Get the network
		network, err = r.client.GetNetwork(ctx, site, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read WAN network, got error: %s", err),
			)
			return
		}
	}

	// Convert to model
	diags := r.networkToModel(ctx, network, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wanResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state wanResourceModel
	var plan wanResourceModel

	// Step 1: Read the current state (which already contains API values from previous reads)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 2: Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)

	// Step 3: Convert the updated state to API format
	network, diags := r.modelToNetwork(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Step 4: Send to API
	network.ID = state.ID.ValueString()
	updatedNetwork, err := r.client.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update WAN network, got error: %s", err),
		)
		return
	}

	// Step 5: Update state with API response
	diags = r.networkToModel(ctx, updatedNetwork, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *wanResource) applyPlanToState(
	_ context.Context,
	plan *wanResourceModel,
	state *wanResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		state.Type = plan.Type
	}
	if !plan.TypeV6.IsNull() && !plan.TypeV6.IsUnknown() {
		state.TypeV6 = plan.TypeV6
	}
	if !plan.Vlan.IsNull() && !plan.Vlan.IsUnknown() {
		state.Vlan = plan.Vlan
	}
	if !plan.EgressQoS.IsNull() && !plan.EgressQoS.IsUnknown() {
		state.EgressQoS = plan.EgressQoS
	}
	if !plan.DHCPCoS.IsNull() && !plan.DHCPCoS.IsUnknown() {
		state.DHCPCoS = plan.DHCPCoS
	}
	if !plan.DHCPV6CoS.IsNull() && !plan.DHCPV6CoS.IsUnknown() {
		state.DHCPV6CoS = plan.DHCPV6CoS
	}
	if !plan.DNS1.IsNull() && !plan.DNS1.IsUnknown() {
		state.DNS1 = plan.DNS1
	}
	if !plan.DNS2.IsNull() && !plan.DNS2.IsUnknown() {
		state.DNS2 = plan.DNS2
	}
	if !plan.IPv6DNS1.IsNull() && !plan.IPv6DNS1.IsUnknown() {
		state.IPv6DNS1 = plan.IPv6DNS1
	}
	if !plan.IPv6DNS2.IsNull() && !plan.IPv6DNS2.IsUnknown() {
		state.IPv6DNS2 = plan.IPv6DNS2
	}
	if !plan.DNSPreference.IsNull() && !plan.DNSPreference.IsUnknown() {
		state.DNSPreference = plan.DNSPreference
	}
	if !plan.IPv6DNSPreference.IsNull() && !plan.IPv6DNSPreference.IsUnknown() {
		state.IPv6DNSPreference = plan.IPv6DNSPreference
	}
	if !plan.DHCPV6PDSize.IsNull() && !plan.DHCPV6PDSize.IsUnknown() {
		state.DHCPV6PDSize = plan.DHCPV6PDSize
	}
	if !plan.DHCPV6PDSizeAuto.IsNull() && !plan.DHCPV6PDSizeAuto.IsUnknown() {
		state.DHCPV6PDSizeAuto = plan.DHCPV6PDSizeAuto
	}
	if !plan.DHCPV6Options.IsNull() && !plan.DHCPV6Options.IsUnknown() {
		state.DHCPV6Options = plan.DHCPV6Options
	}
	if !plan.IPv6WANDelegationType.IsNull() && !plan.IPv6WANDelegationType.IsUnknown() {
		state.IPv6WANDelegationType = plan.IPv6WANDelegationType
	}
	if !plan.SmartQ.IsNull() && !plan.SmartQ.IsUnknown() {
		state.SmartQ = plan.SmartQ
	}
	if !plan.UPnPEnabled.IsNull() && !plan.UPnPEnabled.IsUnknown() {
		state.UPnPEnabled = plan.UPnPEnabled
	}
	if !plan.UPnPWANInterface.IsNull() && !plan.UPnPWANInterface.IsUnknown() {
		state.UPnPWANInterface = plan.UPnPWANInterface
	}
	if !plan.UPnPNatPMPEnabled.IsNull() && !plan.UPnPNatPMPEnabled.IsUnknown() {
		state.UPnPNatPMPEnabled = plan.UPnPNatPMPEnabled
	}
	if !plan.UPnPSecureMode.IsNull() && !plan.UPnPSecureMode.IsUnknown() {
		state.UPnPSecureMode = plan.UPnPSecureMode
	}
	if !plan.LoadBalanceType.IsNull() && !plan.LoadBalanceType.IsUnknown() {
		state.LoadBalanceType = plan.LoadBalanceType
	}
	if !plan.LoadBalanceWeight.IsNull() && !plan.LoadBalanceWeight.IsUnknown() {
		state.LoadBalanceWeight = plan.LoadBalanceWeight
	}
	if !plan.FailoverPriority.IsNull() && !plan.FailoverPriority.IsUnknown() {
		state.FailoverPriority = plan.FailoverPriority
	}
	if !plan.IGMPProxyFor.IsNull() && !plan.IGMPProxyFor.IsUnknown() {
		state.IGMPProxyFor = plan.IGMPProxyFor
	}
	if !plan.IGMPProxyUpstream.IsNull() && !plan.IGMPProxyUpstream.IsUnknown() {
		state.IGMPProxyUpstream = plan.IGMPProxyUpstream
	}
	if !plan.ReportWANEvent.IsNull() && !plan.ReportWANEvent.IsUnknown() {
		state.ReportWANEvent = plan.ReportWANEvent
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.DHCPOptions.IsNull() && !plan.DHCPOptions.IsUnknown() {
		state.DHCPOptions = plan.DHCPOptions
	}
	if !plan.IPAliases.IsNull() && !plan.IPAliases.IsUnknown() {
		state.IPAliases = plan.IPAliases
	}
	if !plan.ProviderCapabilities.IsNull() && !plan.ProviderCapabilities.IsUnknown() {
		state.ProviderCapabilities = plan.ProviderCapabilities
	}
}

func (r *wanResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state wanResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the network
	networkName := state.Name.ValueString()
	networkID := state.ID.ValueString()
	err := r.client.DeleteNetwork(ctx, site, networkID, networkName)
	if err != nil {
		// WAN networks cannot be deleted from the controller; removing from state only.
		if strings.Contains(err.Error(), "NoDelete") {
			return
		}
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete WAN network, got error: %s", err),
		)
		return
	}
}

func (r *wanResource) ImportState(
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
// Only fields with known values are set on the Network struct; null/unknown
// fields are left as nil so marshalWAN omits them via omitempty.
func (r *wanResource) modelToNetwork(
	ctx context.Context,
	model *wanResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:            model.Name.ValueStringPointer(),
		Purpose:         unifi.PurposeWAN, // Statically set to "wan"
		WANNetworkGroup: util.Ptr("WAN"),  // Statically set to "WAN"
		HiddenID:        "WAN",            // Statically set to "WAN"
		Enabled:         model.Enabled.ValueBool(),
	}

	// WAN Type — type has a Default so it's always known;
	// type_v6 may be unknown on Create.
	if !model.Type.IsNull() && !model.Type.IsUnknown() {
		network.WANType = model.Type.ValueStringPointer()
	}
	if !model.TypeV6.IsNull() && !model.TypeV6.IsUnknown() {
		network.WANTypeV6 = model.TypeV6.ValueStringPointer()
	}

	// DHCP CoS
	if !model.DHCPCoS.IsNull() && !model.DHCPCoS.IsUnknown() {
		network.WANDHCPCos = model.DHCPCoS.ValueInt64Pointer()
	}
	if !model.DHCPV6CoS.IsNull() && !model.DHCPV6CoS.IsUnknown() {
		network.WANDHCPv6Cos = model.DHCPV6CoS.ValueInt64Pointer()
	}

	// DNS Settings
	if !model.DNS1.IsNull() && !model.DNS1.IsUnknown() {
		network.WANDNS1 = model.DNS1.ValueStringPointer()
	}
	if !model.DNS2.IsNull() && !model.DNS2.IsUnknown() {
		network.WANDNS2 = model.DNS2.ValueStringPointer()
	}
	if !model.IPv6DNS1.IsNull() && !model.IPv6DNS1.IsUnknown() {
		network.WANIPV6DNS1 = model.IPv6DNS1.ValueStringPointer()
	}
	if !model.IPv6DNS2.IsNull() && !model.IPv6DNS2.IsUnknown() {
		network.WANIPV6DNS2 = model.IPv6DNS2.ValueStringPointer()
	}
	if !model.DNSPreference.IsNull() && !model.DNSPreference.IsUnknown() {
		network.WANDNSPreference = model.DNSPreference.ValueStringPointer()
	}
	if !model.IPv6DNSPreference.IsNull() && !model.IPv6DNSPreference.IsUnknown() {
		network.WANIPV6DNSPreference = model.IPv6DNSPreference.ValueStringPointer()
	}

	// Handle VLAN configuration
	if !model.Vlan.IsNull() && !model.Vlan.IsUnknown() {
		var vlan vlanModel
		d := model.Vlan.As(ctx, &vlan, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.WANVLANEnabled = vlan.Enabled.ValueBool()
			network.WANVLAN = vlan.ID.ValueInt64Pointer()
		}
	}

	// Handle Egress QoS configuration
	if !model.EgressQoS.IsNull() && !model.EgressQoS.IsUnknown() {
		var egressQos egressQosModel
		d := model.EgressQoS.As(ctx, &egressQos, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.WANEgressQOSEnabled = egressQos.Enabled.ValueBoolPointer()
			network.WANEgressQOS = egressQos.Priority.ValueInt64Pointer()
		}
	}

	// DHCPv6 Settings
	if !model.DHCPV6PDSize.IsNull() && !model.DHCPV6PDSize.IsUnknown() {
		network.WANDHCPv6PDSize = model.DHCPV6PDSize.ValueInt64Pointer()
	}
	if !model.DHCPV6PDSizeAuto.IsNull() && !model.DHCPV6PDSizeAuto.IsUnknown() {
		network.WANDHCPv6PDSizeAuto = model.DHCPV6PDSizeAuto.ValueBool()
	}
	if !model.IPv6WANDelegationType.IsNull() && !model.IPv6WANDelegationType.IsUnknown() {
		network.IPV6WANDelegationType = model.IPv6WANDelegationType.ValueStringPointer()
	}

	// Convert DHCPv6 options list
	if !model.DHCPV6Options.IsNull() && !model.DHCPV6Options.IsUnknown() {
		var dhcpV6Options []dhcpOptionModel
		diags.Append(model.DHCPV6Options.ElementsAs(ctx, &dhcpV6Options, false)...)
		if !diags.HasError() {
			network.WANDHCPv6Options = make([]unifi.NetworkWANDHCPv6Options, len(dhcpV6Options))
			for i, opt := range dhcpV6Options {
				network.WANDHCPv6Options[i] = unifi.NetworkWANDHCPv6Options{
					OptionNumber: opt.OptionNumber.ValueInt64Pointer(),
					Value:        opt.Value.ValueStringPointer(),
				}
			}
		}
	}

	// Handle Smart Queue configuration
	if !model.SmartQ.IsNull() && !model.SmartQ.IsUnknown() {
		var smartq smartqModel
		d := model.SmartQ.As(ctx, &smartq, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.WANSmartQEnabled = smartq.Enabled.ValueBool()
			network.WANSmartQUpRate = smartq.UpRate.ValueInt64Pointer()
			network.WANSmartQDownRate = smartq.DownRate.ValueInt64Pointer()
		}
	}

	// UPnP Settings
	if !model.UPnPEnabled.IsNull() && !model.UPnPEnabled.IsUnknown() {
		network.UPnPEnabled = model.UPnPEnabled.ValueBoolPointer()
	}
	if !model.UPnPWANInterface.IsNull() && !model.UPnPWANInterface.IsUnknown() {
		network.UPnPWANInterface = model.UPnPWANInterface.ValueStringPointer()
	}
	if !model.UPnPNatPMPEnabled.IsNull() && !model.UPnPNatPMPEnabled.IsUnknown() {
		network.UPnPNatPMPEnabled = model.UPnPNatPMPEnabled.ValueBoolPointer()
	}
	if !model.UPnPSecureMode.IsNull() && !model.UPnPSecureMode.IsUnknown() {
		network.UPnPSecureMode = model.UPnPSecureMode.ValueBoolPointer()
	}

	// Load Balance Settings
	if !model.LoadBalanceType.IsNull() && !model.LoadBalanceType.IsUnknown() {
		network.WANLoadBalanceType = model.LoadBalanceType.ValueStringPointer()
	}
	if !model.LoadBalanceWeight.IsNull() && !model.LoadBalanceWeight.IsUnknown() {
		network.WANLoadBalanceWeight = model.LoadBalanceWeight.ValueInt64Pointer()
	}
	if !model.FailoverPriority.IsNull() && !model.FailoverPriority.IsUnknown() {
		network.WANFailoverPriority = model.FailoverPriority.ValueInt64Pointer()
	}

	// IGMP Settings
	if !model.IGMPProxyFor.IsNull() && !model.IGMPProxyFor.IsUnknown() {
		network.IGMPProxyFor = model.IGMPProxyFor.ValueStringPointer()
	}
	if !model.IGMPProxyUpstream.IsNull() && !model.IGMPProxyUpstream.IsUnknown() {
		network.IGMPProxyUpstream = model.IGMPProxyUpstream.ValueBool()
	}

	// Additional Settings
	if !model.ReportWANEvent.IsNull() && !model.ReportWANEvent.IsUnknown() {
		network.ReportWANEvent = model.ReportWANEvent.ValueBool()
	}

	// Convert DHCP options list
	if !model.DHCPOptions.IsNull() && !model.DHCPOptions.IsUnknown() {
		var dhcpOptionsModel []dhcpOptionModel
		diags.Append(model.DHCPOptions.ElementsAs(ctx, &dhcpOptionsModel, false)...)
		if !diags.HasError() {
			dhcpOptions := make([]unifi.NetworkWANDHCPOptions, len(dhcpOptionsModel))
			for i, opt := range dhcpOptionsModel {
				dhcpOptions[i] = unifi.NetworkWANDHCPOptions{
					OptionNumber: opt.OptionNumber.ValueInt64Pointer(),
					Value:        opt.Value.ValueStringPointer(),
				}
			}
			network.WANDHCPOptions = dhcpOptions
		}
	}

	// Convert IP aliases list
	if !model.IPAliases.IsNull() && !model.IPAliases.IsUnknown() {
		var ipAliases []string
		diags.Append(model.IPAliases.ElementsAs(ctx, &ipAliases, false)...)
		network.WANIPAliases = ipAliases
	}

	// Provider Capabilities
	if !model.ProviderCapabilities.IsNull() && !model.ProviderCapabilities.IsUnknown() {
		var providerCaps providerCapabilitiesModel
		diags.Append(
			model.ProviderCapabilities.As(ctx, &providerCaps, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			network.WANProviderCapabilities = &unifi.NetworkWANProviderCapabilities{
				DownloadKilobitsPerSecond: providerCaps.DownloadKbps.ValueInt64Pointer(),
				UploadKilobitsPerSecond:   providerCaps.UploadKbps.ValueInt64Pointer(),
			}
		}
	}

	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model.
func (r *wanResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *wanResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringPointerValue(network.Name)

	// WAN Type Settings — only overwrite when API returns a value
	if network.WANType != nil {
		model.Type = types.StringValue(*network.WANType)
	}
	if network.WANTypeV6 != nil {
		model.TypeV6 = types.StringValue(*network.WANTypeV6)
	}

	// VLAN Settings
	vlanValue := vlanModel{
		Enabled: types.BoolValue(network.WANVLANEnabled),
		ID:      types.Int64PointerValue(network.WANVLAN),
	}
	vlanObj, d := types.ObjectValueFrom(ctx, vlanValue.AttributeTypes(), vlanValue)
	diags.Append(d...)
	model.Vlan = vlanObj

	// Egress QoS Settings — only create/update the object if model already has it or API has data
	hasEgressQosData := network.WANEgressQOSEnabled != nil || network.WANEgressQOS != nil
	if !model.EgressQoS.IsNull() || hasEgressQosData {
		var currentEgressQos egressQosModel
		if !model.EgressQoS.IsNull() && !model.EgressQoS.IsUnknown() {
			d := model.EgressQoS.As(ctx, &currentEgressQos, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.WANEgressQOSEnabled != nil {
			currentEgressQos.Enabled = types.BoolValue(*network.WANEgressQOSEnabled)
		}
		if network.WANEgressQOS != nil {
			currentEgressQos.Priority = types.Int64Value(*network.WANEgressQOS)
		}
		egressQosObj, d := types.ObjectValueFrom(
			ctx,
			currentEgressQos.AttributeTypes(),
			currentEgressQos,
		)
		diags.Append(d...)
		model.EgressQoS = egressQosObj
	}

	// QoS Settings — only overwrite when API returns a value
	if network.WANDHCPCos != nil {
		model.DHCPCoS = types.Int64Value(*network.WANDHCPCos)
	}
	if network.WANDHCPv6Cos != nil {
		model.DHCPV6CoS = types.Int64Value(*network.WANDHCPv6Cos)
	}

	// DNS Settings — only overwrite when API returns a value
	if network.WANDNS1 != nil {
		model.DNS1 = types.StringValue(*network.WANDNS1)
	}
	if network.WANDNS2 != nil {
		model.DNS2 = types.StringValue(*network.WANDNS2)
	}
	if network.WANIPV6DNS1 != nil {
		model.IPv6DNS1 = types.StringValue(*network.WANIPV6DNS1)
	}
	if network.WANIPV6DNS2 != nil {
		model.IPv6DNS2 = types.StringValue(*network.WANIPV6DNS2)
	}

	// DNS Preference — only overwrite when API returns a value
	if network.WANDNSPreference != nil {
		model.DNSPreference = types.StringValue(*network.WANDNSPreference)
	}
	if network.WANIPV6DNSPreference != nil {
		model.IPv6DNSPreference = types.StringValue(*network.WANIPV6DNSPreference)
	}

	// DHCPv6 Settings — only overwrite when API returns a value
	if network.WANDHCPv6PDSize != nil {
		model.DHCPV6PDSize = types.Int64Value(*network.WANDHCPv6PDSize)
	}
	model.DHCPV6PDSizeAuto = types.BoolValue(network.WANDHCPv6PDSizeAuto)

	if network.IPV6WANDelegationType != nil {
		model.IPv6WANDelegationType = types.StringValue(*network.IPV6WANDelegationType)
	}

	// Convert DHCPv6 options to list
	if len(network.WANDHCPv6Options) > 0 {
		dhcpV6OptionAttrTypes := map[string]attr.Type{
			"option_number": types.Int64Type,
			"value":         types.StringType,
		}
		dhcpV6OptionsValues := make([]attr.Value, len(network.WANDHCPv6Options))
		for i, opt := range network.WANDHCPv6Options {
			dhcpV6OptionsValues[i], _ = types.ObjectValue(
				dhcpV6OptionAttrTypes,
				map[string]attr.Value{
					"option_number": types.Int64PointerValue(opt.OptionNumber),
					"value":         types.StringPointerValue(opt.Value),
				},
			)
		}
		model.DHCPV6Options, diags = types.ListValue(
			types.ObjectType{AttrTypes: dhcpV6OptionAttrTypes},
			dhcpV6OptionsValues,
		)
	} else {
		dhcpV6OptionAttrTypes := map[string]attr.Type{
			"option_number": types.Int64Type,
			"value":         types.StringType,
		}
		model.DHCPV6Options = types.ListNull(types.ObjectType{AttrTypes: dhcpV6OptionAttrTypes})
	}

	// Smart Queue Settings — only create/update the object if model already has it or API has data
	hasSmartqData := network.WANSmartQEnabled || network.WANSmartQUpRate != nil ||
		network.WANSmartQDownRate != nil
	if !model.SmartQ.IsNull() || hasSmartqData {
		var currentSmartq smartqModel
		if !model.SmartQ.IsNull() && !model.SmartQ.IsUnknown() {
			d := model.SmartQ.As(ctx, &currentSmartq, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if hasSmartqData {
			currentSmartq.Enabled = types.BoolValue(network.WANSmartQEnabled)
			if network.WANSmartQUpRate != nil {
				currentSmartq.UpRate = types.Int64Value(*network.WANSmartQUpRate)
			}
			if network.WANSmartQDownRate != nil {
				currentSmartq.DownRate = types.Int64Value(*network.WANSmartQDownRate)
			}
		}
		smartqObj, d := types.ObjectValueFrom(ctx, currentSmartq.AttributeTypes(), currentSmartq)
		diags.Append(d...)
		model.SmartQ = smartqObj
	}

	// UPnP Settings — only overwrite when API returns a value
	if network.UPnPEnabled != nil {
		model.UPnPEnabled = types.BoolValue(*network.UPnPEnabled)
	}
	if network.UPnPWANInterface != nil {
		model.UPnPWANInterface = types.StringValue(*network.UPnPWANInterface)
	}
	if network.UPnPNatPMPEnabled != nil {
		model.UPnPNatPMPEnabled = types.BoolValue(*network.UPnPNatPMPEnabled)
	}
	if network.UPnPSecureMode != nil {
		model.UPnPSecureMode = types.BoolValue(*network.UPnPSecureMode)
	}

	// Load Balance Settings — only overwrite when API returns a value
	if network.WANLoadBalanceType != nil {
		model.LoadBalanceType = types.StringValue(*network.WANLoadBalanceType)
	}
	if network.WANLoadBalanceWeight != nil {
		model.LoadBalanceWeight = types.Int64Value(*network.WANLoadBalanceWeight)
	}
	if network.WANFailoverPriority != nil {
		model.FailoverPriority = types.Int64Value(*network.WANFailoverPriority)
	}

	// IGMP Settings — only overwrite when API returns a value
	if network.IGMPProxyFor != nil {
		model.IGMPProxyFor = types.StringValue(*network.IGMPProxyFor)
	}
	model.IGMPProxyUpstream = types.BoolValue(network.IGMPProxyUpstream)

	// Additional Settings
	model.ReportWANEvent = types.BoolValue(network.ReportWANEvent)
	model.Enabled = types.BoolValue(network.Enabled)

	// Convert DHCP options to list
	if len(network.WANDHCPOptions) > 0 {
		dhcpOptionAttrTypes := map[string]attr.Type{
			"option_number": types.Int64Type,
			"value":         types.StringType,
		}
		dhcpOptionsValues := make([]attr.Value, len(network.WANDHCPOptions))
		for i, opt := range network.WANDHCPOptions {
			dhcpOptionsValues[i], _ = types.ObjectValue(
				dhcpOptionAttrTypes,
				map[string]attr.Value{
					"option_number": types.Int64PointerValue(opt.OptionNumber),
					"value":         types.StringPointerValue(opt.Value),
				},
			)
		}
		model.DHCPOptions, diags = types.ListValue(
			types.ObjectType{AttrTypes: dhcpOptionAttrTypes},
			dhcpOptionsValues,
		)
	} else {
		dhcpOptionAttrTypes := map[string]attr.Type{
			"option_number": types.Int64Type,
			"value":         types.StringType,
		}
		model.DHCPOptions = types.ListNull(types.ObjectType{AttrTypes: dhcpOptionAttrTypes})
	}

	// Convert IP aliases to list
	if len(network.WANIPAliases) > 0 {
		ipAliasesValues := make([]attr.Value, len(network.WANIPAliases))
		for i, alias := range network.WANIPAliases {
			ipAliasesValues[i] = types.StringValue(alias)
		}
		model.IPAliases, diags = types.ListValue(types.StringType, ipAliasesValues)
	} else {
		model.IPAliases = types.ListNull(types.StringType)
	}

	// Provider Capabilities — only overwrite when API returns data
	if network.WANProviderCapabilities != nil &&
		(network.WANProviderCapabilities.DownloadKilobitsPerSecond != nil ||
			network.WANProviderCapabilities.UploadKilobitsPerSecond != nil) {
		providerCapsAttrTypes := map[string]attr.Type{
			"download_kilobits_per_second": types.Int64Type,
			"upload_kilobits_per_second":   types.Int64Type,
		}
		providerCapsValues := map[string]attr.Value{
			"download_kilobits_per_second": types.Int64PointerValue(
				network.WANProviderCapabilities.DownloadKilobitsPerSecond,
			),
			"upload_kilobits_per_second": types.Int64PointerValue(
				network.WANProviderCapabilities.UploadKilobitsPerSecond,
			),
		}
		var d diag.Diagnostics
		model.ProviderCapabilities, d = types.ObjectValue(
			providerCapsAttrTypes,
			providerCapsValues,
		)
		diags.Append(d...)
	}
	// If API returns nil, preserve existing model.ProviderCapabilities

	// Apply schema defaults for fields that are still null/unknown (handles import case
	// where there's no previous state to preserve).
	applyWANDefaults(model)

	return diags
}

// applyWANDefaults ensures fields that are still null or unknown have properly-typed values.
// For complex types (objects, lists), this sets typed null values so the framework can
// distinguish between "not configured" and "configured as empty" correctly.
// For scalar fields, unknown values are converted to null (which is valid for Computed
// fields). This prevents "unknown value after apply" errors.
func applyWANDefaults(model *wanResourceModel) {
	// Convert any remaining unknown scalars to null — unknown is not valid after apply.
	if model.Type.IsUnknown() {
		model.Type = types.StringNull()
	}
	if model.TypeV6.IsUnknown() {
		model.TypeV6 = types.StringNull()
	}
	if model.DHCPCoS.IsUnknown() {
		model.DHCPCoS = types.Int64Null()
	}
	if model.DHCPV6CoS.IsUnknown() {
		model.DHCPV6CoS = types.Int64Null()
	}
	if model.DNSPreference.IsUnknown() {
		model.DNSPreference = types.StringNull()
	}
	if model.IPv6DNSPreference.IsUnknown() {
		model.IPv6DNSPreference = types.StringNull()
	}
	if model.DHCPV6PDSize.IsUnknown() {
		model.DHCPV6PDSize = types.Int64Null()
	}
	if model.DHCPV6PDSizeAuto.IsUnknown() {
		model.DHCPV6PDSizeAuto = types.BoolNull()
	}
	if model.IPv6WANDelegationType.IsUnknown() {
		model.IPv6WANDelegationType = types.StringNull()
	}
	if model.UPnPEnabled.IsUnknown() {
		model.UPnPEnabled = types.BoolNull()
	}
	if model.UPnPNatPMPEnabled.IsUnknown() {
		model.UPnPNatPMPEnabled = types.BoolNull()
	}
	if model.UPnPSecureMode.IsUnknown() {
		model.UPnPSecureMode = types.BoolNull()
	}
	if model.LoadBalanceType.IsUnknown() {
		model.LoadBalanceType = types.StringNull()
	}
	if model.LoadBalanceWeight.IsUnknown() {
		model.LoadBalanceWeight = types.Int64Null()
	}
	if model.FailoverPriority.IsUnknown() {
		model.FailoverPriority = types.Int64Null()
	}
	if model.IGMPProxyFor.IsUnknown() {
		model.IGMPProxyFor = types.StringNull()
	}
	if model.IGMPProxyUpstream.IsUnknown() {
		model.IGMPProxyUpstream = types.BoolNull()
	}
	if model.ReportWANEvent.IsUnknown() {
		model.ReportWANEvent = types.BoolNull()
	}
	if model.Enabled.IsUnknown() {
		model.Enabled = types.BoolNull()
	}
	// Nested objects need properly-typed null values
	if model.EgressQoS.IsNull() || model.EgressQoS.IsUnknown() {
		model.EgressQoS = types.ObjectNull(egressQosModel{}.AttributeTypes())
	}
	if model.SmartQ.IsNull() || model.SmartQ.IsUnknown() {
		model.SmartQ = types.ObjectNull(smartqModel{}.AttributeTypes())
	}
	if model.Vlan.IsNull() || model.Vlan.IsUnknown() {
		model.Vlan = types.ObjectNull(vlanModel{}.AttributeTypes())
	}
	if model.ProviderCapabilities.IsNull() || model.ProviderCapabilities.IsUnknown() {
		model.ProviderCapabilities = types.ObjectNull(providerCapabilitiesModel{}.AttributeTypes())
	}
	// List types need properly-typed null values
	if model.DHCPOptions.IsNull() || model.DHCPOptions.IsUnknown() {
		model.DHCPOptions = types.ListNull(
			types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()},
		)
	}
	if model.DHCPV6Options.IsNull() || model.DHCPV6Options.IsUnknown() {
		model.DHCPV6Options = types.ListNull(
			types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()},
		)
	}
	if model.IPAliases.IsNull() || model.IPAliases.IsUnknown() {
		model.IPAliases = types.ListNull(types.StringType)
	}
}
