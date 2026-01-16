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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
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
	VlanEnabled types.Bool  `tfsdk:"vlan_enabled"`
	Vlan        types.Int64 `tfsdk:"vlan"`

	// QoS Settings
	EgressQoS        types.Int64 `tfsdk:"egress_qos"`
	EgressQoSEnabled types.Bool  `tfsdk:"egress_qos_enabled"`
	DHCPCoS          types.Int64 `tfsdk:"dhcp_cos"`
	DHCPV6CoS        types.Int64 `tfsdk:"dhcpv6_cos"`

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
	SmartQEnabled  types.Bool  `tfsdk:"smartq_enabled"`
	SmartQUpRate   types.Int64 `tfsdk:"smartq_up_rate"`
	SmartQDownRate types.Int64 `tfsdk:"smartq_down_rate"`

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

// providerCapabilitiesModel describes the provider capabilities nested object.
type providerCapabilitiesModel struct {
	DownloadKbps types.Int64 `tfsdk:"download_kilobits_per_second"`
	UploadKbps   types.Int64 `tfsdk:"upload_kilobits_per_second"`
}

// dhcpOptionModel describes a DHCPv6 option.
type dhcpOptionModel struct {
	OptionNumber types.Int64  `tfsdk:"option_number"`
	Value        types.String `tfsdk:"value"`
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
				Default:             stringdefault.StaticString("dhcpv6"),
				MarkdownDescription: "The IPv6 WAN type (dhcpv6, static, disabled)",
				Validators: []validator.String{
					stringvalidator.OneOf("dhcpv6", "static", "disabled"),
				},
			},
			"vlan_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether VLAN is enabled",
			},
			"vlan": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "The VLAN ID",
				Validators: []validator.Int64{
					int64validator.Between(0, 4094),
				},
			},
			"egress_qos": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "Egress QoS priority",
				Validators: []validator.Int64{
					int64validator.Between(0, 7),
				},
			},
			"egress_qos_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether egress QoS is enabled",
			},
			"dhcp_cos": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "DHCP Class of Service",
				Validators: []validator.Int64{
					int64validator.Between(0, 7),
				},
			},
			"dhcpv6_cos": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "DHCPv6 Class of Service",
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
				Default:             stringdefault.StaticString("auto"),
				MarkdownDescription: "DNS preference (auto, manual)",
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"ipv6_dns_preference": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("auto"),
				MarkdownDescription: "IPv6 DNS preference (auto, manual)",
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"dhcpv6_pd_size": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(56),
				MarkdownDescription: "DHCPv6 prefix delegation size",
				Validators: []validator.Int64{
					int64validator.Between(48, 64),
				},
			},
			"dhcpv6_pd_size_auto": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether DHCPv6 PD size is automatic",
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
				Default:             stringdefault.StaticString("pd"),
				MarkdownDescription: "IPv6 WAN delegation type (pd, static)",
				Validators: []validator.String{
					stringvalidator.OneOf("pd", "static"),
				},
			},
			"smartq_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether Smart Queue is enabled",
			},
			"smartq_up_rate": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Smart Queue upload rate in kbps",
			},
			"smartq_down_rate": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Smart Queue download rate in kbps",
			},
			"upnp_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether UPnP is enabled",
			},
			"upnp_wan_interface": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UPnP WAN interface",
			},
			"upnp_nat_pmp_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether UPnP NAT-PMP is enabled",
			},
			"upnp_secure_mode": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether UPnP secure mode is enabled",
			},
			"load_balance_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("failover-only"),
				MarkdownDescription: "Load balance type (failover-only, weighted)",
				Validators: []validator.String{
					stringvalidator.OneOf("failover-only", "weighted"),
				},
			},
			"load_balance_weight": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(50),
				MarkdownDescription: "Load balance weight",
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"failover_priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				MarkdownDescription: "Failover priority",
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"igmp_proxy_for": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("none"),
				MarkdownDescription: "IGMP proxy for (none, lan, guest)",
				Validators: []validator.String{
					stringvalidator.OneOf("none", "lan", "guest"),
				},
			},
			"igmp_proxy_upstream": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether IGMP proxy upstream is enabled",
			},
			"report_wan_event": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to report WAN events",
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
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create WAN network, got error: %s", err),
		)
		return
	}

	// Convert back to model
	diags = r.networkToModel(ctx, createdNetwork, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	if !plan.VlanEnabled.IsNull() && !plan.VlanEnabled.IsUnknown() {
		state.VlanEnabled = plan.VlanEnabled
	}
	if !plan.Vlan.IsNull() && !plan.Vlan.IsUnknown() {
		state.Vlan = plan.Vlan
	}
	if !plan.EgressQoS.IsNull() && !plan.EgressQoS.IsUnknown() {
		state.EgressQoS = plan.EgressQoS
	}
	if !plan.EgressQoSEnabled.IsNull() && !plan.EgressQoSEnabled.IsUnknown() {
		state.EgressQoSEnabled = plan.EgressQoSEnabled
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
	if !plan.SmartQEnabled.IsNull() && !plan.SmartQEnabled.IsUnknown() {
		state.SmartQEnabled = plan.SmartQEnabled
	}
	if !plan.SmartQUpRate.IsNull() && !plan.SmartQUpRate.IsUnknown() {
		state.SmartQUpRate = plan.SmartQUpRate
	}
	if !plan.SmartQDownRate.IsNull() && !plan.SmartQDownRate.IsUnknown() {
		state.SmartQDownRate = plan.SmartQDownRate
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
func (r *wanResource) modelToNetwork(
	ctx context.Context,
	model *wanResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:                model.Name.ValueString(),
		Purpose:             "wan", // Statically set to "wan"
		WANNetworkGroup:     "WAN", // Statically set to "WAN"
		HiddenID:            "WAN", // Statically set to "WAN"
		WANType:             model.Type.ValueString(),
		WANTypeV6:           model.TypeV6.ValueString(),
		WANVLANEnabled:      model.VlanEnabled.ValueBool(),
		WANVLAN:             model.Vlan.ValueInt64(),
		WANEgressQOS:        model.EgressQoS.ValueInt64(),
		WANEgressQOSEnabled: model.EgressQoSEnabled.ValueBoolPointer(),
		WANDHCPCos:          model.DHCPCoS.ValueInt64(),
		WANDHCPv6Cos:        model.DHCPV6CoS.ValueInt64(),
	}

	// DNS Settings
	if !model.DNS1.IsNull() && !model.DNS1.IsUnknown() {
		network.WANDNS1 = model.DNS1.ValueString()
	}
	if !model.DNS2.IsNull() && !model.DNS2.IsUnknown() {
		network.WANDNS2 = model.DNS2.ValueString()
	}
	if !model.IPv6DNS1.IsNull() && !model.IPv6DNS1.IsUnknown() {
		network.WANIPV6DNS1 = model.IPv6DNS1.ValueString()
	}
	if !model.IPv6DNS2.IsNull() && !model.IPv6DNS2.IsUnknown() {
		network.WANIPV6DNS2 = model.IPv6DNS2.ValueString()
	}
	network.WANDNSPreference = model.DNSPreference.ValueString()
	network.WANIPV6DNSPreference = model.IPv6DNSPreference.ValueString()

	// DHCPv6 Settings
	network.WANDHCPv6PDSize = model.DHCPV6PDSize.ValueInt64()
	network.WANDHCPv6PDSizeAuto = model.DHCPV6PDSizeAuto.ValueBool()
	network.IPV6WANDelegationType = model.IPv6WANDelegationType.ValueString()

	// Convert DHCPv6 options list
	if !model.DHCPV6Options.IsNull() && !model.DHCPV6Options.IsUnknown() {
		var dhcpV6Options []dhcpOptionModel
		diags.Append(model.DHCPV6Options.ElementsAs(ctx, &dhcpV6Options, false)...)
		if !diags.HasError() {
			network.WANDHCPv6Options = make([]unifi.NetworkWANDHCPv6Options, len(dhcpV6Options))
			for i, opt := range dhcpV6Options {
				network.WANDHCPv6Options[i] = unifi.NetworkWANDHCPv6Options{
					OptionNumber: opt.OptionNumber.ValueInt64(),
					Value:        opt.Value.ValueString(),
				}
			}
		}
	}

	// Smart Queue Settings
	network.WANSmartQEnabled = model.SmartQEnabled.ValueBool()
	if !model.SmartQUpRate.IsNull() && !model.SmartQUpRate.IsUnknown() {
		network.WANSmartQUpRate = model.SmartQUpRate.ValueInt64()
	}
	if !model.SmartQDownRate.IsNull() && !model.SmartQDownRate.IsUnknown() {
		network.WANSmartQDownRate = model.SmartQDownRate.ValueInt64()
	}

	// UPnP Settings
	network.UPnPLanEnabled = model.UPnPEnabled.ValueBool()
	if !model.UPnPWANInterface.IsNull() && !model.UPnPWANInterface.IsUnknown() {
		network.UPnPWANInterface = model.UPnPWANInterface.ValueString()
	}
	network.UPnPNatPMPEnabled = model.UPnPEnabled.ValueBoolPointer()
	network.UPnPSecureMode = model.UPnPSecureMode.ValueBoolPointer()

	// Load Balance Settings
	network.WANLoadBalanceType = model.LoadBalanceType.ValueString()
	network.WANLoadBalanceWeight = model.LoadBalanceWeight.ValueInt64()
	network.WANFailoverPriority = model.FailoverPriority.ValueInt64()

	// IGMP Settings
	network.IGMPProxyFor = model.IGMPProxyFor.ValueString()
	network.IGMPProxyUpstream = model.IGMPProxyUpstream.ValueBool()

	// Additional Settings
	network.ReportWANEvent = model.ReportWANEvent.ValueBool()
	network.Enabled = model.Enabled.ValueBool()

	// Convert DHCP options list
	if !model.DHCPOptions.IsNull() && !model.DHCPOptions.IsUnknown() {
		var dhcpOptionsModel []dhcpOptionModel
		diags.Append(model.DHCPOptions.ElementsAs(ctx, &dhcpOptionsModel, false)...)
		if !diags.HasError() {
			dhcpOptions := make([]unifi.NetworkWANDHCPOptions, len(dhcpOptionsModel))
			for i, opt := range dhcpOptionsModel {
				dhcpOptions[i] = unifi.NetworkWANDHCPOptions{
					OptionNumber: opt.OptionNumber.ValueInt64(),
					Value:        opt.Value.ValueString(),
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
			network.WANProviderCapabilities = unifi.NetworkWANProviderCapabilities{
				DownloadKilobitsPerSecond: providerCaps.DownloadKbps.ValueInt64(),
				UploadKilobitsPerSecond:   providerCaps.UploadKbps.ValueInt64(),
			}
		}
	}

	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model.
func (r *wanResource) networkToModel(
	_ context.Context,
	network *unifi.Network,
	model *wanResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(network.Name)

	// WAN Type Settings
	if network.WANType == "" {
		model.Type = types.StringNull()
	} else {
		model.Type = types.StringValue(network.WANType)
	}

	if network.WANTypeV6 == "" {
		model.TypeV6 = types.StringNull()
	} else {
		model.TypeV6 = types.StringValue(network.WANTypeV6)
	}

	// VLAN Settings
	model.VlanEnabled = types.BoolValue(network.WANVLANEnabled)
	model.Vlan = types.Int64Value(int64(network.WANVLAN))

	// QoS Settings
	model.EgressQoS = types.Int64Value(int64(network.WANEgressQOS))
	model.EgressQoSEnabled = types.BoolPointerValue(network.WANEgressQOSEnabled)
	model.DHCPCoS = types.Int64Value(int64(network.WANDHCPCos))
	model.DHCPV6CoS = types.Int64Value(int64(network.WANDHCPv6Cos))

	// DNS Settings
	if network.WANDNS1 == "" {
		model.DNS1 = types.StringNull()
	} else {
		model.DNS1 = types.StringValue(network.WANDNS1)
	}

	if network.WANDNS2 == "" {
		model.DNS2 = types.StringNull()
	} else {
		model.DNS2 = types.StringValue(network.WANDNS2)
	}

	if network.WANIPV6DNS1 == "" {
		model.IPv6DNS1 = types.StringNull()
	} else {
		model.IPv6DNS1 = types.StringValue(network.WANIPV6DNS1)
	}

	if network.WANIPV6DNS2 == "" {
		model.IPv6DNS2 = types.StringNull()
	} else {
		model.IPv6DNS2 = types.StringValue(network.WANIPV6DNS2)
	}

	if network.WANDNSPreference == "" {
		model.DNSPreference = types.StringNull()
	} else {
		model.DNSPreference = types.StringValue(network.WANDNSPreference)
	}

	if network.WANIPV6DNSPreference == "" {
		model.IPv6DNSPreference = types.StringNull()
	} else {
		model.IPv6DNSPreference = types.StringValue(network.WANIPV6DNSPreference)
	}

	// DHCPv6 Settings
	model.DHCPV6PDSize = types.Int64Value(int64(network.WANDHCPv6PDSize))
	model.DHCPV6PDSizeAuto = types.BoolValue(network.WANDHCPv6PDSizeAuto)

	if network.IPV6WANDelegationType == "" {
		model.IPv6WANDelegationType = types.StringNull()
	} else {
		model.IPv6WANDelegationType = types.StringValue(network.IPV6WANDelegationType)
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
					"option_number": types.Int64Value(int64(opt.OptionNumber)),
					"value":         types.StringValue(opt.Value),
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

	// Smart Queue Settings
	model.SmartQEnabled = types.BoolValue(network.WANSmartQEnabled)
	if network.WANSmartQUpRate == 0 {
		model.SmartQUpRate = types.Int64Null()
	} else {
		model.SmartQUpRate = types.Int64Value(int64(network.WANSmartQUpRate))
	}
	if network.WANSmartQDownRate == 0 {
		model.SmartQDownRate = types.Int64Null()
	} else {
		model.SmartQDownRate = types.Int64Value(int64(network.WANSmartQDownRate))
	}

	// UPnP Settings
	model.UPnPEnabled = types.BoolPointerValue(network.UPnPEnabled)
	if network.UPnPWANInterface == "" {
		model.UPnPWANInterface = types.StringNull()
	} else {
		model.UPnPWANInterface = types.StringValue(network.UPnPWANInterface)
	}
	model.UPnPNatPMPEnabled = types.BoolPointerValue(network.UPnPNatPMPEnabled)
	model.UPnPSecureMode = types.BoolPointerValue(network.UPnPSecureMode)

	// Load Balance Settings
	if network.WANLoadBalanceType == "" {
		model.LoadBalanceType = types.StringNull()
	} else {
		model.LoadBalanceType = types.StringValue(network.WANLoadBalanceType)
	}
	model.LoadBalanceWeight = types.Int64Value(int64(network.WANLoadBalanceWeight))
	model.FailoverPriority = types.Int64Value(int64(network.WANFailoverPriority))

	// IGMP Settings
	if network.IGMPProxyFor == "" {
		model.IGMPProxyFor = types.StringNull()
	} else {
		model.IGMPProxyFor = types.StringValue(network.IGMPProxyFor)
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
					"option_number": types.Int64Value(int64(opt.OptionNumber)),
					"value":         types.StringValue(opt.Value),
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

	// Provider Capabilities
	if network.WANProviderCapabilities.DownloadKilobitsPerSecond > 0 ||
		network.WANProviderCapabilities.UploadKilobitsPerSecond > 0 {
		providerCapsAttrTypes := map[string]attr.Type{
			"download_kilobits_per_second": types.Int64Type,
			"upload_kilobits_per_second":   types.Int64Type,
		}
		providerCapsValues := map[string]attr.Value{
			"download_kilobits_per_second": types.Int64Value(
				int64(network.WANProviderCapabilities.DownloadKilobitsPerSecond),
			),
			"upload_kilobits_per_second": types.Int64Value(
				int64(network.WANProviderCapabilities.UploadKilobitsPerSecond),
			),
		}
		model.ProviderCapabilities, diags = types.ObjectValue(
			providerCapsAttrTypes,
			providerCapsValues,
		)
	} else {
		model.ProviderCapabilities = types.ObjectNull(map[string]attr.Type{
			"download_kilobits_per_second": types.Int64Type,
			"upload_kilobits_per_second":   types.Int64Type,
		})
	}

	return diags
}
