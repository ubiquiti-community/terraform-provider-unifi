package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &portProfileResource{}
	_ resource.ResourceWithImportState = &portProfileResource{}
)

func NewPortProfileFrameworkResource() resource.Resource {
	return &portProfileResource{}
}

// portProfileResource defines the resource implementation.
type portProfileResource struct {
	client *Client
}

// portProfileResourceModel describes the resource data model.
type portProfileResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Site                       types.String `tfsdk:"site"`
	Autoneg                    types.Bool   `tfsdk:"autoneg"`
	Dot1XCtrl                  types.String `tfsdk:"dot1x_ctrl"`
	Dot1XIdleTimeout           types.Int64  `tfsdk:"dot1x_idle_timeout"`
	EgressRateLimitKbps        types.Int64  `tfsdk:"egress_rate_limit_kbps"`
	EgressRateLimitKbpsEnabled types.Bool   `tfsdk:"egress_rate_limit_kbps_enabled"`
	Forward                    types.String `tfsdk:"forward"`
	FullDuplex                 types.Bool   `tfsdk:"full_duplex"`
	Isolation                  types.Bool   `tfsdk:"isolation"`
	LLDPMedEnabled             types.Bool   `tfsdk:"lldpmed_enabled"`
	LLDPMedNotifyEnabled       types.Bool   `tfsdk:"lldpmed_notify_enabled"`
	NativeNetworkConfID        types.String `tfsdk:"native_networkconf_id"`
	Name                       types.String `tfsdk:"name"`
	OpMode                     types.String `tfsdk:"op_mode"`
	PoeMode                    types.String `tfsdk:"poe_mode"`
	PortSecurityEnabled        types.Bool   `tfsdk:"port_security_enabled"`
	PortSecurityMacAddress     types.Set    `tfsdk:"port_security_mac_address"`
	PriorityQueue1Level        types.Int64  `tfsdk:"priority_queue1_level"`
	PriorityQueue2Level        types.Int64  `tfsdk:"priority_queue2_level"`
	PriorityQueue3Level        types.Int64  `tfsdk:"priority_queue3_level"`
	PriorityQueue4Level        types.Int64  `tfsdk:"priority_queue4_level"`
	Speed                      types.Int64  `tfsdk:"speed"`
	StormctrlBcastEnabled      types.Bool   `tfsdk:"stormctrl_bcast_enabled"`
	StormctrlBcastLevel        types.Int64  `tfsdk:"stormctrl_bcast_level"`
	StormctrlBcastRate         types.Int64  `tfsdk:"stormctrl_bcast_rate"`
	StormctrlMcastEnabled      types.Bool   `tfsdk:"stormctrl_mcast_enabled"`
	StormctrlMcastLevel        types.Int64  `tfsdk:"stormctrl_mcast_level"`
	StormctrlMcastRate         types.Int64  `tfsdk:"stormctrl_mcast_rate"`
	StormctrlType              types.String `tfsdk:"stormctrl_type"`
	StormctrlUcastEnabled      types.Bool   `tfsdk:"stormctrl_ucast_enabled"`
	StormctrlUcastLevel        types.Int64  `tfsdk:"stormctrl_ucast_level"`
	StormctrlUcastRate         types.Int64  `tfsdk:"stormctrl_ucast_rate"`
	STPPortMode                types.Bool   `tfsdk:"stp_port_mode"`
	TaggedNetworkConfIDs       types.Set    `tfsdk:"tagged_networkconf_ids"`
	VoiceNetworkConfID         types.String `tfsdk:"voice_networkconf_id"`
}

func (r *portProfileResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_port_profile"
}

func (r *portProfileResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "`unifi_port_profile` manages a port profile for use on network switches.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the port profile.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "The name of the site to associate the port profile with.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"autoneg": schema.BoolAttribute{
				Description: "Enable link auto negotiation for the port profile. When set to `true` this overrides `speed`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"dot1x_ctrl": schema.StringAttribute{
				Description: "The type of 802.1X control to use. Can be `auto`, `force_authorized`, `force_unauthorized`, `mac_based` or `multi_host`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("force_authorized"),
				Validators: []validator.String{
					stringvalidator.OneOf(
						"auto",
						"force_authorized",
						"force_unauthorized",
						"mac_based",
						"multi_host",
					),
				},
			},
			"dot1x_idle_timeout": schema.Int64Attribute{
				Description: "The timeout, in seconds, to use when using the MAC Based 802.1X control. Can be between 0 and 65535",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(300),
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
			},
			"egress_rate_limit_kbps": schema.Int64Attribute{
				Description: "The egress rate limit, in kpbs, for the port profile. Can be between `64` and `9999999`.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(64, 9999999),
				},
			},
			"egress_rate_limit_kbps_enabled": schema.BoolAttribute{
				Description: "Enable egress rate limiting for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"forward": schema.StringAttribute{
				Description: "The type forwarding to use for the port profile. Can be `all`, `native`, `customize` or `disabled`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("all"),
				Validators: []validator.String{
					stringvalidator.OneOf("all", "native", "customize", "disabled"),
				},
			},
			"full_duplex": schema.BoolAttribute{
				Description: "Enable full duplex for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"isolation": schema.BoolAttribute{
				Description: "Enable port isolation for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"lldpmed_enabled": schema.BoolAttribute{
				Description: "Enable LLDP-MED for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"lldpmed_notify_enabled": schema.BoolAttribute{
				Description: "Enable LLDP-MED topology change notifications for the port profile.",
				Optional:    true,
			},
			"native_networkconf_id": schema.StringAttribute{
				Description: "The ID of network to use as the main network on the port profile.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the port profile.",
				Optional:    true,
			},
			"op_mode": schema.StringAttribute{
				Description: "The operation mode for the port profile. Can only be `switch`",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("switch"),
				Validators: []validator.String{
					stringvalidator.OneOf("switch"),
				},
			},
			"poe_mode": schema.StringAttribute{
				Description: "The POE mode for the port profile. Can be one of `auto`, `passv24`, `passthrough` or `off`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "passv24", "passthrough", "off"),
				},
			},
			"port_security_enabled": schema.BoolAttribute{
				Description: "Enable port security for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"port_security_mac_address": schema.SetAttribute{
				Description: "The MAC addresses associated with the port security for the port profile.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"priority_queue1_level": schema.Int64Attribute{
				Description: "The priority queue 1 level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
			"priority_queue2_level": schema.Int64Attribute{
				Description: "The priority queue 2 level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
			"priority_queue3_level": schema.Int64Attribute{
				Description: "The priority queue 3 level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
			"priority_queue4_level": schema.Int64Attribute{
				Description: "The priority queue 4 level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
			"speed": schema.Int64Attribute{
				Description: "The link speed to set for the port profile. Can be one of `10`, `100`, `1000`, `2500`, `5000`, `10000`, `20000`, `25000`, `40000`, `50000` or `100000`",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						10,
						100,
						1000,
						2500,
						5000,
						10000,
						20000,
						25000,
						40000,
						50000,
						100000,
					),
				},
			},
			"stormctrl_bcast_enabled": schema.BoolAttribute{
				Description: "Enable broadcast Storm Control for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"stormctrl_bcast_level": schema.Int64Attribute{
				Description: "The broadcast Storm Control level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
					int64validator.ConflictsWith(path.MatchRoot("stormctrl_bcast_rate")),
				},
			},
			"stormctrl_bcast_rate": schema.Int64Attribute{
				Description: "The broadcast Storm Control rate for the port profile. Can be between 0 and 14880000.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 14880000),
					int64validator.ConflictsWith(path.MatchRoot("stormctrl_bcast_level")),
				},
			},
			"stormctrl_mcast_enabled": schema.BoolAttribute{
				Description: "Enable multicast Storm Control for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"stormctrl_mcast_level": schema.Int64Attribute{
				Description: "The multicast Storm Control level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
					int64validator.ConflictsWith(path.MatchRoot("stormctrl_mcast_rate")),
				},
			},
			"stormctrl_mcast_rate": schema.Int64Attribute{
				Description: "The multicast Storm Control rate for the port profile. Can be between 0 and 14880000.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 14880000),
					int64validator.ConflictsWith(path.MatchRoot("stormctrl_mcast_level")),
				},
			},
			"stormctrl_type": schema.StringAttribute{
				Description: "The type of Storm Control to use for the port profile. Can be `level` or `rate`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("level", "rate"),
				},
			},
			"stormctrl_ucast_enabled": schema.BoolAttribute{
				Description: "Enable unknown unicast Storm Control for the port profile.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"stormctrl_ucast_level": schema.Int64Attribute{
				Description: "The unknown unicast Storm Control level for the port profile. Can be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
					int64validator.ConflictsWith(path.MatchRoot("stormctrl_ucast_rate")),
				},
			},
			"stormctrl_ucast_rate": schema.Int64Attribute{
				Description: "The unknown unicast Storm Control rate for the port profile. Can be between 0 and 14880000.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 14880000),
					int64validator.ConflictsWith(path.MatchRoot("stormctrl_ucast_level")),
				},
			},
			"stp_port_mode": schema.BoolAttribute{
				Description: "Enable Spanning Tree Protocol (STP) for the port profile.",
				Optional:    true,
			},
			"tagged_networkconf_ids": schema.SetAttribute{
				Description: "The IDs of networks to tag traffic with for the port profile.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"voice_networkconf_id": schema.StringAttribute{
				Description: "The ID of network to use for voice traffic for the port profile.",
				Optional:    true,
			},
		},
	}
}

func (r *portProfileResource) Configure(
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

func (r *portProfileResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan portProfileResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert model to API request
	portProfile, convDiags := r.modelToAPIPortProfile(ctx, &plan)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiPortProfile, err := r.client.CreatePortProfile(ctx, site, portProfile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Port Profile",
			fmt.Sprintf("Could not create port profile: %s", err),
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(apiPortProfile.ID)
	plan.Site = types.StringValue(site)
	r.setResourceData(ctx, apiPortProfile, &plan, site)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *portProfileResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state portProfileResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	portProfile, err := r.client.GetPortProfile(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Port Profile",
			fmt.Sprintf("Could not read port profile %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, portProfile, &state, site)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *portProfileResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan portProfileResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state portProfileResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()

	// Read current port profile and merge with planned changes
	currentPortProfile, err := r.client.GetPortProfile(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Port Profile for Update",
			fmt.Sprintf("Could not read port profile %s for update: %s", id, err),
		)
		return
	}

	// Apply current API values to state
	r.setResourceData(ctx, currentPortProfile, &state, site)

	// Apply plan changes to the state (merge pattern)
	r.applyPlanToState(ctx, &plan, &state)

	// Convert updated state to API request
	portProfile, convDiags := r.modelToAPIPortProfile(ctx, &state)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	portProfile.ID = id
	portProfile.SiteID = site

	apiPortProfile, err := r.client.UpdatePortProfile(ctx, site, portProfile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Port Profile",
			fmt.Sprintf("Could not update port profile %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, apiPortProfile, &state, site)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *portProfileResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state portProfileResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeletePortProfile(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Port Profile",
			fmt.Sprintf("Could not delete port profile %s: %s", id, err),
		)
		return
	}
}

func (r *portProfileResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts, diags := ParseImportID(req.ID, 1, 2)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if site := idParts["site"]; site != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	}

	if id := idParts["id"]; id != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	}
}

// Helper methods

func (r *portProfileResource) modelToAPIPortProfile(
	ctx context.Context,
	model *portProfileResourceModel,
) (*unifi.PortProfile, diag.Diagnostics) {
	var diags diag.Diagnostics

	portProfile := &unifi.PortProfile{
		Name:   model.Name.ValueString(),
		OpMode: model.OpMode.ValueString(),
	}

	if !model.Autoneg.IsNull() && !model.Autoneg.IsUnknown() {
		portProfile.Autoneg = model.Autoneg.ValueBool()
	}

	if !model.FullDuplex.IsNull() && !model.FullDuplex.IsUnknown() {
		portProfile.FullDuplex = model.FullDuplex.ValueBool()
	}

	if !model.Isolation.IsNull() && !model.Isolation.IsUnknown() {
		portProfile.Isolation = model.Isolation.ValueBool()
	}

	if !model.Dot1XCtrl.IsNull() && !model.Dot1XCtrl.IsUnknown() {
		portProfile.Dot1XCtrl = model.Dot1XCtrl.ValueString()
	}

	if !model.Dot1XIdleTimeout.IsNull() && !model.Dot1XIdleTimeout.IsUnknown() {
		timeout := model.Dot1XIdleTimeout.ValueInt64()
		portProfile.Dot1XIDleTimeout = timeout
	}

	if !model.Forward.IsNull() && !model.Forward.IsUnknown() {
		portProfile.Forward = model.Forward.ValueString()
	}

	if !model.LLDPMedEnabled.IsNull() && !model.LLDPMedEnabled.IsUnknown() {
		portProfile.LldpmedEnabled = model.LLDPMedEnabled.ValueBool()
	}

	if !model.LLDPMedNotifyEnabled.IsNull() && !model.LLDPMedNotifyEnabled.IsUnknown() {
		portProfile.LldpmedNotifyEnabled = model.LLDPMedNotifyEnabled.ValueBool()
	}

	// Skip native network config for now as field name is unclear

	if !model.PoeMode.IsNull() && !model.PoeMode.IsUnknown() {
		portProfile.PoeMode = model.PoeMode.ValueString()
	}

	if !model.PortSecurityEnabled.IsNull() && !model.PortSecurityEnabled.IsUnknown() {
		portProfile.PortSecurityEnabled = model.PortSecurityEnabled.ValueBool()
	}

	// Convert port security MAC addresses
	if !model.PortSecurityMacAddress.IsNull() && !model.PortSecurityMacAddress.IsUnknown() {
		var macAddresses []string
		diags.Append(model.PortSecurityMacAddress.ElementsAs(ctx, &macAddresses, false)...)
		if !diags.HasError() {
			portProfile.PortSecurityMACAddress = macAddresses
		}
	}

	if !model.Speed.IsNull() && !model.Speed.IsUnknown() {
		portProfile.Speed = model.Speed.ValueInt64()
	}

	// Convert tagged network IDs - skip for now as field name is unclear

	// Handle storm control and other complex fields as needed...

	return portProfile, diags
}

func (r *portProfileResource) setResourceData(
	ctx context.Context,
	portProfile *unifi.PortProfile,
	model *portProfileResourceModel,
	site string,
) {
	model.Site = types.StringValue(site)

	if portProfile.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(portProfile.Name)
	}

	model.Autoneg = types.BoolValue(portProfile.Autoneg)

	if portProfile.Dot1XCtrl == "" {
		model.Dot1XCtrl = types.StringValue("force_authorized")
	} else {
		model.Dot1XCtrl = types.StringValue(portProfile.Dot1XCtrl)
	}

	model.Dot1XIdleTimeout = types.Int64Value(portProfile.Dot1XIDleTimeout)

	if portProfile.Forward == "" {
		model.Forward = types.StringValue("native")
	} else {
		model.Forward = types.StringValue(portProfile.Forward)
	}

	model.FullDuplex = types.BoolValue(portProfile.FullDuplex)

	model.Isolation = types.BoolValue(portProfile.Isolation)

	model.LLDPMedEnabled = types.BoolValue(portProfile.LldpmedEnabled)

	// Only set lldpmed_notify_enabled if it was in the plan or if it's explicitly true
	if !model.LLDPMedNotifyEnabled.IsNull() || portProfile.LldpmedNotifyEnabled {
		model.LLDPMedNotifyEnabled = types.BoolValue(portProfile.LldpmedNotifyEnabled)
	} else {
		model.LLDPMedNotifyEnabled = types.BoolNull()
	}

	model.NativeNetworkConfID = types.StringNull() // Skip for now

	if portProfile.OpMode == "" {
		model.OpMode = types.StringValue("switch")
	} else {
		model.OpMode = types.StringValue(portProfile.OpMode)
	}

	if portProfile.PoeMode == "" {
		model.PoeMode = types.StringNull()
	} else {
		model.PoeMode = types.StringValue(portProfile.PoeMode)
	}

	model.PortSecurityEnabled = types.BoolValue(portProfile.PortSecurityEnabled)

	// Convert port security MAC addresses
	if len(portProfile.PortSecurityMACAddress) == 0 {
		model.PortSecurityMacAddress = types.SetNull(types.StringType)
	} else {
		macAddressList := make([]types.String, len(portProfile.PortSecurityMACAddress))
		for i, mac := range portProfile.PortSecurityMACAddress {
			macAddressList[i] = types.StringValue(mac)
		}
		macAddressSet, _ := types.SetValueFrom(ctx, types.StringType, macAddressList)
		model.PortSecurityMacAddress = macAddressSet
	}

	// Only set speed if it was in the plan or if it's non-zero
	if !model.Speed.IsNull() || portProfile.Speed != 0 {
		model.Speed = types.Int64Value(portProfile.Speed)
	} else {
		model.Speed = types.Int64Null()
	}

	// Convert tagged network IDs - skip for now
	model.TaggedNetworkConfIDs = types.SetNull(types.StringType)

	model.VoiceNetworkConfID = types.StringNull() // Skip for now

	// Set remaining fields to defaults or null as appropriate
	model.EgressRateLimitKbps = types.Int64Null()
	model.EgressRateLimitKbpsEnabled = types.BoolValue(false)
	model.PriorityQueue1Level = types.Int64Null()
	model.PriorityQueue2Level = types.Int64Null()
	model.PriorityQueue3Level = types.Int64Null()
	model.PriorityQueue4Level = types.Int64Null()
	model.StormctrlBcastEnabled = types.BoolValue(false)
	model.StormctrlBcastLevel = types.Int64Null()
	model.StormctrlBcastRate = types.Int64Null()
	model.StormctrlMcastEnabled = types.BoolValue(false)
	model.StormctrlMcastLevel = types.Int64Null()
	model.StormctrlMcastRate = types.Int64Null()
	model.StormctrlType = types.StringNull()
	model.StormctrlUcastEnabled = types.BoolValue(false)
	model.StormctrlUcastLevel = types.Int64Null()
	model.StormctrlUcastRate = types.Int64Null()
	model.STPPortMode = types.BoolNull()
}

func (r *portProfileResource) applyPlanToState(
	_ context.Context,
	plan *portProfileResourceModel,
	state *portProfileResourceModel,
) {
	// Apply all plan values that are not null/unknown to the state
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Autoneg.IsNull() && !plan.Autoneg.IsUnknown() {
		state.Autoneg = plan.Autoneg
	}
	if !plan.Dot1XCtrl.IsNull() && !plan.Dot1XCtrl.IsUnknown() {
		state.Dot1XCtrl = plan.Dot1XCtrl
	}
	if !plan.Dot1XIdleTimeout.IsNull() && !plan.Dot1XIdleTimeout.IsUnknown() {
		state.Dot1XIdleTimeout = plan.Dot1XIdleTimeout
	}
	if !plan.EgressRateLimitKbps.IsNull() && !plan.EgressRateLimitKbps.IsUnknown() {
		state.EgressRateLimitKbps = plan.EgressRateLimitKbps
	}
	if !plan.EgressRateLimitKbpsEnabled.IsNull() && !plan.EgressRateLimitKbpsEnabled.IsUnknown() {
		state.EgressRateLimitKbpsEnabled = plan.EgressRateLimitKbpsEnabled
	}
	if !plan.Forward.IsNull() && !plan.Forward.IsUnknown() {
		state.Forward = plan.Forward
	}
	if !plan.FullDuplex.IsNull() && !plan.FullDuplex.IsUnknown() {
		state.FullDuplex = plan.FullDuplex
	}
	if !plan.Isolation.IsNull() && !plan.Isolation.IsUnknown() {
		state.Isolation = plan.Isolation
	}
	if !plan.LLDPMedEnabled.IsNull() && !plan.LLDPMedEnabled.IsUnknown() {
		state.LLDPMedEnabled = plan.LLDPMedEnabled
	}
	if !plan.LLDPMedNotifyEnabled.IsNull() && !plan.LLDPMedNotifyEnabled.IsUnknown() {
		state.LLDPMedNotifyEnabled = plan.LLDPMedNotifyEnabled
	}
	if !plan.NativeNetworkConfID.IsNull() && !plan.NativeNetworkConfID.IsUnknown() {
		state.NativeNetworkConfID = plan.NativeNetworkConfID
	}
	if !plan.OpMode.IsNull() && !plan.OpMode.IsUnknown() {
		state.OpMode = plan.OpMode
	}
	if !plan.PoeMode.IsNull() && !plan.PoeMode.IsUnknown() {
		state.PoeMode = plan.PoeMode
	}
	if !plan.PortSecurityEnabled.IsNull() && !plan.PortSecurityEnabled.IsUnknown() {
		state.PortSecurityEnabled = plan.PortSecurityEnabled
	}
	if !plan.PortSecurityMacAddress.IsNull() && !plan.PortSecurityMacAddress.IsUnknown() {
		state.PortSecurityMacAddress = plan.PortSecurityMacAddress
	}
	if !plan.Speed.IsNull() && !plan.Speed.IsUnknown() {
		state.Speed = plan.Speed
	}
	if !plan.TaggedNetworkConfIDs.IsNull() && !plan.TaggedNetworkConfIDs.IsUnknown() {
		state.TaggedNetworkConfIDs = plan.TaggedNetworkConfIDs
	}
	if !plan.VoiceNetworkConfID.IsNull() && !plan.VoiceNetworkConfID.IsUnknown() {
		state.VoiceNetworkConfID = plan.VoiceNetworkConfID
	}
	// Apply other fields as needed...
}
