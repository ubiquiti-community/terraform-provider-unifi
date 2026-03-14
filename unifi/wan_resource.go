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

	// DNS Settings
	DNS types.Object `tfsdk:"dns"`

	// DHCP Settings
	DHCP   types.Object `tfsdk:"dhcp"`
	DHCPv6 types.Object `tfsdk:"dhcpv6"`

	// Smart Queue Settings
	SmartQ types.Object `tfsdk:"smartq"`

	// UPnP Settings
	UPnP types.Object `tfsdk:"upnp"`

	// Load Balance Settings
	LoadBalance types.Object `tfsdk:"load_balance"`

	// IGMP Settings
	IGMPProxy types.Object `tfsdk:"igmp_proxy"`

	// Additional Settings
	ReportWANEvent types.Bool `tfsdk:"report_wan_event"`
	Enabled        types.Bool `tfsdk:"enabled"`
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

// dnsModel describes the DNS configuration nested object.
type dnsModel struct {
	Primary        types.String `tfsdk:"primary"`
	Secondary      types.String `tfsdk:"secondary"`
	IPv6Primary    types.String `tfsdk:"ipv6_primary"`
	IPv6Secondary  types.String `tfsdk:"ipv6_secondary"`
	Preference     types.String `tfsdk:"preference"`
	IPv6Preference types.String `tfsdk:"ipv6_preference"`
}

func (m dnsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"primary":         types.StringType,
		"secondary":       types.StringType,
		"ipv6_primary":    types.StringType,
		"ipv6_secondary":  types.StringType,
		"preference":      types.StringType,
		"ipv6_preference": types.StringType,
	}
}

// upnpModel describes the UPnP configuration nested object.
type upnpModel struct {
	Enabled       types.Bool   `tfsdk:"enabled"`
	WANInterface  types.String `tfsdk:"wan_interface"`
	NatPMPEnabled types.Bool   `tfsdk:"nat_pmp_enabled"`
	SecureMode    types.Bool   `tfsdk:"secure_mode"`
}

func (m upnpModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":         types.BoolType,
		"wan_interface":   types.StringType,
		"nat_pmp_enabled": types.BoolType,
		"secure_mode":     types.BoolType,
	}
}

// loadBalanceModel describes the load balance configuration nested object.
type loadBalanceModel struct {
	Type             types.String `tfsdk:"type"`
	Weight           types.Int64  `tfsdk:"weight"`
	FailoverPriority types.Int64  `tfsdk:"failover_priority"`
}

func (m loadBalanceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":              types.StringType,
		"weight":            types.Int64Type,
		"failover_priority": types.Int64Type,
	}
}

// igmpProxyModel describes the IGMP proxy configuration nested object.
type igmpProxyModel struct {
	Downstream types.String `tfsdk:"downstream"`
	Upstream   types.Bool   `tfsdk:"upstream"`
}

func (m igmpProxyModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"downstream": types.StringType,
		"upstream":   types.BoolType,
	}
}

// dhcpv6WanModel describes the DHCPv6 WAN configuration nested object.
type dhcpv6WanModel struct {
	CoS            types.Int64  `tfsdk:"cos"`
	PDSize         types.Int64  `tfsdk:"pd_size"`
	PDSizeAuto     types.Bool   `tfsdk:"pd_size_auto"`
	Options        types.List   `tfsdk:"options"`
	DelegationType types.String `tfsdk:"wan_delegation_type"`
}

func (m dhcpv6WanModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cos":          types.Int64Type,
		"pd_size":      types.Int64Type,
		"pd_size_auto": types.BoolType,
		"options": types.ListType{
			ElemType: types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()},
		},
		"wan_delegation_type": types.StringType,
	}
}

// dhcpWanModel describes the DHCP WAN configuration nested object.
type dhcpWanModel struct {
	CoS     types.Int64 `tfsdk:"cos"`
	Options types.List  `tfsdk:"options"`
}

func (m dhcpWanModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cos": types.Int64Type,
		"options": types.ListType{
			ElemType: types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()},
		},
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
				Computed:            true,
				MarkdownDescription: "VLAN configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
				Computed:            true,
				MarkdownDescription: "Egress QoS configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
			"dns": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DNS configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"primary": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Primary DNS server",
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
					},
					"secondary": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Secondary DNS server",
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
					},
					"ipv6_primary": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Primary IPv6 DNS server",
					},
					"ipv6_secondary": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Secondary IPv6 DNS server",
					},
					"preference": schema.StringAttribute{
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
					"ipv6_preference": schema.StringAttribute{
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
				},
			},
			"dhcp": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DHCP configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"cos": schema.Int64Attribute{
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
					"options": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "DHCP options",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"option_number": schema.Int64Attribute{
									Required:            true,
									MarkdownDescription: "DHCP option number",
								},
								"value": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "DHCP option value",
								},
							},
						},
					},
				},
			},
			"dhcpv6": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DHCPv6 configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"cos": schema.Int64Attribute{
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
					"pd_size": schema.Int64Attribute{
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
					"pd_size_auto": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether DHCPv6 PD size is automatic",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"options": schema.ListNestedAttribute{
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
					"wan_delegation_type": schema.StringAttribute{
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
				},
			},
			"smartq": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Smart Queue configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
			"upnp": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "UPnP configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether UPnP is enabled",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"wan_interface": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "UPnP WAN interface",
					},
					"nat_pmp_enabled": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether UPnP NAT-PMP is enabled",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"secure_mode": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether UPnP secure mode is enabled",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"load_balance": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Load balance configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
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
					"weight": schema.Int64Attribute{
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
				},
			},
			"igmp_proxy": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "IGMP proxy configuration",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"downstream": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "IGMP proxy downstream target (none, lan, guest)",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf("none", "lan", "guest"),
						},
					},
					"upstream": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether IGMP proxy upstream is enabled",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
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
	// Note: nested objects (Vlan, EgressQoS, DNS, DHCP, DHCPv6, SmartQ, UPnP,
	// LoadBalance, IGMPProxy) are NOT overlaid from the plan because the plan
	// may contain unknown values for computed child attributes that don't have
	// prior state (e.g. on initial create). The API response in state already
	// has the correct, fully-resolved values for these objects.
	if !config.ReportWANEvent.IsNull() {
		state.ReportWANEvent = plan.ReportWANEvent
	}
	if !config.Enabled.IsNull() {
		state.Enabled = plan.Enabled
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
	if !plan.DNS.IsNull() && !plan.DNS.IsUnknown() {
		state.DNS = plan.DNS
	}
	if !plan.DHCP.IsNull() && !plan.DHCP.IsUnknown() {
		state.DHCP = plan.DHCP
	}
	if !plan.DHCPv6.IsNull() && !plan.DHCPv6.IsUnknown() {
		state.DHCPv6 = plan.DHCPv6
	}
	if !plan.SmartQ.IsNull() && !plan.SmartQ.IsUnknown() {
		state.SmartQ = plan.SmartQ
	}
	if !plan.UPnP.IsNull() && !plan.UPnP.IsUnknown() {
		state.UPnP = plan.UPnP
	}
	if !plan.LoadBalance.IsNull() && !plan.LoadBalance.IsUnknown() {
		state.LoadBalance = plan.LoadBalance
	}
	if !plan.IGMPProxy.IsNull() && !plan.IGMPProxy.IsUnknown() {
		state.IGMPProxy = plan.IGMPProxy
	}
	if !plan.ReportWANEvent.IsNull() && !plan.ReportWANEvent.IsUnknown() {
		state.ReportWANEvent = plan.ReportWANEvent
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
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

	// DNS Settings
	if !model.DNS.IsNull() && !model.DNS.IsUnknown() {
		var dns dnsModel
		d := model.DNS.As(ctx, &dns, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !dns.Primary.IsNull() && !dns.Primary.IsUnknown() {
				network.WANDNS1 = dns.Primary.ValueStringPointer()
			}
			if !dns.Secondary.IsNull() && !dns.Secondary.IsUnknown() {
				network.WANDNS2 = dns.Secondary.ValueStringPointer()
			}
			if !dns.IPv6Primary.IsNull() && !dns.IPv6Primary.IsUnknown() {
				network.WANIPV6DNS1 = dns.IPv6Primary.ValueStringPointer()
			}
			if !dns.IPv6Secondary.IsNull() && !dns.IPv6Secondary.IsUnknown() {
				network.WANIPV6DNS2 = dns.IPv6Secondary.ValueStringPointer()
			}
			if !dns.Preference.IsNull() && !dns.Preference.IsUnknown() {
				network.WANDNSPreference = dns.Preference.ValueStringPointer()
			}
			if !dns.IPv6Preference.IsNull() && !dns.IPv6Preference.IsUnknown() {
				network.WANIPV6DNSPreference = dns.IPv6Preference.ValueStringPointer()
			}
		}
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

	// DHCP Settings
	if !model.DHCP.IsNull() && !model.DHCP.IsUnknown() {
		var dhcp dhcpWanModel
		d := model.DHCP.As(ctx, &dhcp, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !dhcp.CoS.IsNull() && !dhcp.CoS.IsUnknown() {
				network.WANDHCPCos = dhcp.CoS.ValueInt64Pointer()
			}
			if !dhcp.Options.IsNull() && !dhcp.Options.IsUnknown() {
				var dhcpOptionsModel []dhcpOptionModel
				diags.Append(dhcp.Options.ElementsAs(ctx, &dhcpOptionsModel, false)...)
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
		}
	}

	// DHCPv6 Settings
	if !model.DHCPv6.IsNull() && !model.DHCPv6.IsUnknown() {
		var dhcpv6 dhcpv6WanModel
		d := model.DHCPv6.As(ctx, &dhcpv6, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !dhcpv6.CoS.IsNull() && !dhcpv6.CoS.IsUnknown() {
				network.WANDHCPv6Cos = dhcpv6.CoS.ValueInt64Pointer()
			}
			if !dhcpv6.PDSize.IsNull() && !dhcpv6.PDSize.IsUnknown() {
				network.WANDHCPv6PDSize = dhcpv6.PDSize.ValueInt64Pointer()
			}
			if !dhcpv6.PDSizeAuto.IsNull() && !dhcpv6.PDSizeAuto.IsUnknown() {
				network.WANDHCPv6PDSizeAuto = dhcpv6.PDSizeAuto.ValueBool()
			}
			if !dhcpv6.DelegationType.IsNull() && !dhcpv6.DelegationType.IsUnknown() {
				network.IPV6WANDelegationType = dhcpv6.DelegationType.ValueStringPointer()
			}
			if !dhcpv6.Options.IsNull() && !dhcpv6.Options.IsUnknown() {
				var dhcpV6Options []dhcpOptionModel
				diags.Append(dhcpv6.Options.ElementsAs(ctx, &dhcpV6Options, false)...)
				if !diags.HasError() {
					network.WANDHCPv6Options = make(
						[]unifi.NetworkWANDHCPv6Options,
						len(dhcpV6Options),
					)
					for i, opt := range dhcpV6Options {
						network.WANDHCPv6Options[i] = unifi.NetworkWANDHCPv6Options{
							OptionNumber: opt.OptionNumber.ValueInt64Pointer(),
							Value:        opt.Value.ValueStringPointer(),
						}
					}
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
	if !model.UPnP.IsNull() && !model.UPnP.IsUnknown() {
		var upnp upnpModel
		d := model.UPnP.As(ctx, &upnp, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !upnp.Enabled.IsNull() && !upnp.Enabled.IsUnknown() {
				network.UPnPEnabled = upnp.Enabled.ValueBoolPointer()
			}
			if !upnp.WANInterface.IsNull() && !upnp.WANInterface.IsUnknown() {
				network.UPnPWANInterface = upnp.WANInterface.ValueStringPointer()
			}
			if !upnp.NatPMPEnabled.IsNull() && !upnp.NatPMPEnabled.IsUnknown() {
				network.UPnPNatPMPEnabled = upnp.NatPMPEnabled.ValueBoolPointer()
			}
			if !upnp.SecureMode.IsNull() && !upnp.SecureMode.IsUnknown() {
				network.UPnPSecureMode = upnp.SecureMode.ValueBoolPointer()
			}
		}
	}

	// Load Balance Settings
	if !model.LoadBalance.IsNull() && !model.LoadBalance.IsUnknown() {
		var lb loadBalanceModel
		d := model.LoadBalance.As(ctx, &lb, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !lb.Type.IsNull() && !lb.Type.IsUnknown() {
				network.WANLoadBalanceType = lb.Type.ValueStringPointer()
			}
			if !lb.Weight.IsNull() && !lb.Weight.IsUnknown() {
				network.WANLoadBalanceWeight = lb.Weight.ValueInt64Pointer()
			}
			if !lb.FailoverPriority.IsNull() && !lb.FailoverPriority.IsUnknown() {
				network.WANFailoverPriority = lb.FailoverPriority.ValueInt64Pointer()
			}
		}
	}

	// IGMP Settings
	if !model.IGMPProxy.IsNull() && !model.IGMPProxy.IsUnknown() {
		var igmp igmpProxyModel
		d := model.IGMPProxy.As(ctx, &igmp, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !igmp.Downstream.IsNull() && !igmp.Downstream.IsUnknown() {
				network.IGMPProxyFor = igmp.Downstream.ValueStringPointer()
			}
			if !igmp.Upstream.IsNull() && !igmp.Upstream.IsUnknown() {
				network.IGMPProxyUpstream = igmp.Upstream.ValueBool()
			}
		}
	}

	// Additional Settings
	if !model.ReportWANEvent.IsNull() && !model.ReportWANEvent.IsUnknown() {
		network.ReportWANEvent = model.ReportWANEvent.ValueBool()
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

	// DNS Settings — only populate if model already has it or API has data
	hasDNSData := network.WANDNS1 != nil || network.WANDNS2 != nil ||
		network.WANIPV6DNS1 != nil || network.WANIPV6DNS2 != nil ||
		network.WANDNSPreference != nil || network.WANIPV6DNSPreference != nil
	if !model.DNS.IsNull() || hasDNSData {
		var currentDNS dnsModel
		if !model.DNS.IsNull() && !model.DNS.IsUnknown() {
			d := model.DNS.As(ctx, &currentDNS, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.WANDNS1 != nil {
			currentDNS.Primary = types.StringValue(*network.WANDNS1)
		}
		if network.WANDNS2 != nil {
			currentDNS.Secondary = types.StringValue(*network.WANDNS2)
		}
		if network.WANIPV6DNS1 != nil {
			currentDNS.IPv6Primary = types.StringValue(*network.WANIPV6DNS1)
		}
		if network.WANIPV6DNS2 != nil {
			currentDNS.IPv6Secondary = types.StringValue(*network.WANIPV6DNS2)
		}
		if network.WANDNSPreference != nil {
			currentDNS.Preference = types.StringValue(*network.WANDNSPreference)
		}
		if network.WANIPV6DNSPreference != nil {
			currentDNS.IPv6Preference = types.StringValue(*network.WANIPV6DNSPreference)
		}
		dnsObj, d := types.ObjectValueFrom(ctx, currentDNS.AttributeTypes(), currentDNS)
		diags.Append(d...)
		model.DNS = dnsObj
	}

	// DHCP Settings — only populate if model already has it or API has data
	hasDHCPData := network.WANDHCPCos != nil || len(network.WANDHCPOptions) > 0
	if !model.DHCP.IsNull() || hasDHCPData {
		var currentDHCP dhcpWanModel
		if !model.DHCP.IsNull() && !model.DHCP.IsUnknown() {
			d := model.DHCP.As(ctx, &currentDHCP, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.WANDHCPCos != nil {
			currentDHCP.CoS = types.Int64Value(*network.WANDHCPCos)
		}
		dhcpOptType := types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()}
		if len(network.WANDHCPOptions) > 0 {
			dhcpOptionsValues := make([]attr.Value, len(network.WANDHCPOptions))
			for i, opt := range network.WANDHCPOptions {
				dhcpOptionsValues[i], _ = types.ObjectValue(
					dhcpOptionModel{}.AttributeTypes(),
					map[string]attr.Value{
						"option_number": types.Int64PointerValue(opt.OptionNumber),
						"value":         types.StringPointerValue(opt.Value),
					},
				)
			}
			currentDHCP.Options, _ = types.ListValue(dhcpOptType, dhcpOptionsValues)
		} else if currentDHCP.Options.IsNull() || currentDHCP.Options.IsUnknown() {
			currentDHCP.Options = types.ListNull(dhcpOptType)
		}
		dhcpObj, d := types.ObjectValueFrom(ctx, currentDHCP.AttributeTypes(), currentDHCP)
		diags.Append(d...)
		model.DHCP = dhcpObj
	}

	// DHCPv6 Settings — only populate if model already has it or API has data
	hasDHCPv6Data := network.WANDHCPv6Cos != nil || network.WANDHCPv6PDSize != nil ||
		network.IPV6WANDelegationType != nil || len(network.WANDHCPv6Options) > 0
	if !model.DHCPv6.IsNull() || hasDHCPv6Data {
		var currentDHCPv6 dhcpv6WanModel
		if !model.DHCPv6.IsNull() && !model.DHCPv6.IsUnknown() {
			d := model.DHCPv6.As(ctx, &currentDHCPv6, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.WANDHCPv6Cos != nil {
			currentDHCPv6.CoS = types.Int64Value(*network.WANDHCPv6Cos)
		}
		if network.WANDHCPv6PDSize != nil {
			currentDHCPv6.PDSize = types.Int64Value(*network.WANDHCPv6PDSize)
		}
		currentDHCPv6.PDSizeAuto = types.BoolValue(network.WANDHCPv6PDSizeAuto)
		if network.IPV6WANDelegationType != nil {
			currentDHCPv6.DelegationType = types.StringValue(*network.IPV6WANDelegationType)
		}
		dhcpV6OptType := types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()}
		if len(network.WANDHCPv6Options) > 0 {
			dhcpV6OptionsValues := make([]attr.Value, len(network.WANDHCPv6Options))
			for i, opt := range network.WANDHCPv6Options {
				dhcpV6OptionsValues[i], _ = types.ObjectValue(
					dhcpOptionModel{}.AttributeTypes(),
					map[string]attr.Value{
						"option_number": types.Int64PointerValue(opt.OptionNumber),
						"value":         types.StringPointerValue(opt.Value),
					},
				)
			}
			currentDHCPv6.Options, _ = types.ListValue(dhcpV6OptType, dhcpV6OptionsValues)
		} else if currentDHCPv6.Options.IsNull() || currentDHCPv6.Options.IsUnknown() {
			currentDHCPv6.Options = types.ListNull(dhcpV6OptType)
		}
		dhcpv6Obj, d := types.ObjectValueFrom(ctx, currentDHCPv6.AttributeTypes(), currentDHCPv6)
		diags.Append(d...)
		model.DHCPv6 = dhcpv6Obj
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

	// UPnP Settings — only populate if model already has it or API has data
	hasUPnPData := network.UPnPEnabled != nil || network.UPnPWANInterface != nil ||
		network.UPnPNatPMPEnabled != nil || network.UPnPSecureMode != nil
	if !model.UPnP.IsNull() || hasUPnPData {
		var currentUPnP upnpModel
		if !model.UPnP.IsNull() && !model.UPnP.IsUnknown() {
			d := model.UPnP.As(ctx, &currentUPnP, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.UPnPEnabled != nil {
			currentUPnP.Enabled = types.BoolValue(*network.UPnPEnabled)
		}
		if network.UPnPWANInterface != nil {
			currentUPnP.WANInterface = types.StringValue(*network.UPnPWANInterface)
		}
		if network.UPnPNatPMPEnabled != nil {
			currentUPnP.NatPMPEnabled = types.BoolValue(*network.UPnPNatPMPEnabled)
		}
		if network.UPnPSecureMode != nil {
			currentUPnP.SecureMode = types.BoolValue(*network.UPnPSecureMode)
		}
		upnpObj, d := types.ObjectValueFrom(ctx, currentUPnP.AttributeTypes(), currentUPnP)
		diags.Append(d...)
		model.UPnP = upnpObj
	}

	// Load Balance Settings — only populate if model already has it or API has data
	hasLBData := network.WANLoadBalanceType != nil || network.WANLoadBalanceWeight != nil ||
		network.WANFailoverPriority != nil
	if !model.LoadBalance.IsNull() || hasLBData {
		var currentLB loadBalanceModel
		if !model.LoadBalance.IsNull() && !model.LoadBalance.IsUnknown() {
			d := model.LoadBalance.As(ctx, &currentLB, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.WANLoadBalanceType != nil {
			currentLB.Type = types.StringValue(*network.WANLoadBalanceType)
		}
		if network.WANLoadBalanceWeight != nil {
			currentLB.Weight = types.Int64Value(*network.WANLoadBalanceWeight)
		}
		if network.WANFailoverPriority != nil {
			currentLB.FailoverPriority = types.Int64Value(*network.WANFailoverPriority)
		}
		lbObj, d := types.ObjectValueFrom(ctx, currentLB.AttributeTypes(), currentLB)
		diags.Append(d...)
		model.LoadBalance = lbObj
	}

	// IGMP Settings — only populate if model already has it or API has data
	hasIGMPData := network.IGMPProxyFor != nil || network.IGMPProxyUpstream
	if !model.IGMPProxy.IsNull() || hasIGMPData {
		var currentIGMP igmpProxyModel
		if !model.IGMPProxy.IsNull() && !model.IGMPProxy.IsUnknown() {
			d := model.IGMPProxy.As(ctx, &currentIGMP, basetypes.ObjectAsOptions{})
			diags.Append(d...)
		}
		if network.IGMPProxyFor != nil {
			currentIGMP.Downstream = types.StringValue(*network.IGMPProxyFor)
		}
		currentIGMP.Upstream = types.BoolValue(network.IGMPProxyUpstream)
		igmpObj, d := types.ObjectValueFrom(ctx, currentIGMP.AttributeTypes(), currentIGMP)
		diags.Append(d...)
		model.IGMPProxy = igmpObj
	}

	// Additional Settings
	model.ReportWANEvent = types.BoolValue(network.ReportWANEvent)
	model.Enabled = types.BoolValue(network.Enabled)

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
	if model.DNS.IsNull() || model.DNS.IsUnknown() {
		model.DNS = types.ObjectNull(dnsModel{}.AttributeTypes())
	}
	if model.DHCP.IsNull() || model.DHCP.IsUnknown() {
		model.DHCP = types.ObjectNull(dhcpWanModel{}.AttributeTypes())
	}
	if model.DHCPv6.IsNull() || model.DHCPv6.IsUnknown() {
		model.DHCPv6 = types.ObjectNull(dhcpv6WanModel{}.AttributeTypes())
	}
	if model.UPnP.IsNull() || model.UPnP.IsUnknown() {
		model.UPnP = types.ObjectNull(upnpModel{}.AttributeTypes())
	}
	if model.LoadBalance.IsNull() || model.LoadBalance.IsUnknown() {
		model.LoadBalance = types.ObjectNull(loadBalanceModel{}.AttributeTypes())
	}
	if model.IGMPProxy.IsNull() || model.IGMPProxy.IsUnknown() {
		model.IGMPProxy = types.ObjectNull(igmpProxyModel{}.AttributeTypes())
	}
	// List types need properly-typed null values
	if model.IPAliases.IsNull() || model.IPAliases.IsUnknown() {
		model.IPAliases = types.ListNull(types.StringType)
	}
}
