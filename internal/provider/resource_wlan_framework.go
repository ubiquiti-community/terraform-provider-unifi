package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
var _ resource.Resource = &wlanFrameworkResource{}
var _ resource.ResourceWithImportState = &wlanFrameworkResource{}

func NewWLANFrameworkResource() resource.Resource {
	return &wlanFrameworkResource{}
}

// wlanFrameworkResource defines the resource implementation.
type wlanFrameworkResource struct {
	client *client
}

// wlanScheduleModel represents a schedule block for WLAN
type wlanScheduleModel struct {
	DayOfWeek   types.String `tfsdk:"day_of_week"`
	StartHour   types.Int64  `tfsdk:"start_hour"`
	StartMinute types.Int64  `tfsdk:"start_minute"`
	Duration    types.Int64  `tfsdk:"duration"`
	Name        types.String `tfsdk:"name"`
}

// wlanFrameworkResourceModel describes the resource data model.
type wlanFrameworkResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Site                     types.String `tfsdk:"site"`
	Name                     types.String `tfsdk:"name"`
	UserGroupID              types.String `tfsdk:"user_group_id"`
	Security                 types.String `tfsdk:"security"`
	WPA3Support              types.Bool   `tfsdk:"wpa3_support"`
	WPA3Transition           types.Bool   `tfsdk:"wpa3_transition"`
	PMFMode                  types.String `tfsdk:"pmf_mode"`
	Passphrase               types.String `tfsdk:"passphrase"`
	HideSSID                 types.Bool   `tfsdk:"hide_ssid"`
	IsGuest                  types.Bool   `tfsdk:"is_guest"`
	MulticastEnhance         types.Bool   `tfsdk:"multicast_enhance"`
	MacFilterEnabled         types.Bool   `tfsdk:"mac_filter_enabled"`
	MacFilterList            types.Set    `tfsdk:"mac_filter_list"`
	MacFilterPolicy          types.String `tfsdk:"mac_filter_policy"`
	RadiusProfileID          types.String `tfsdk:"radius_profile_id"`
	Schedule                 types.List   `tfsdk:"schedule"`
	No2GhzOui                types.Bool   `tfsdk:"no2ghz_oui"`
	L2Isolation              types.Bool   `tfsdk:"l2_isolation"`
	ProxyArp                 types.Bool   `tfsdk:"proxy_arp"`
	BssTransition            types.Bool   `tfsdk:"bss_transition"`
	Uapsd                    types.Bool   `tfsdk:"uapsd"`
	FastRoamingEnabled       types.Bool   `tfsdk:"fast_roaming_enabled"`
	MinimumDataRate2GKbps    types.Int64  `tfsdk:"minimum_data_rate_2g_kbps"`
	MinimumDataRate5GKbps    types.Int64  `tfsdk:"minimum_data_rate_5g_kbps"`
	MinrateSettingPreference types.String `tfsdk:"minrate_setting_preference"`
}

func (r *wlanFrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wlan"
}

func (r *wlanFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WiFi network / SSID in UniFi Controller",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the WLAN.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the WLAN with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The SSID of the network.",
				Required:            true,
			},
			"user_group_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user group to use for this network.",
				Required:            true,
			},
			"security": schema.StringAttribute{
				MarkdownDescription: "The type of WiFi security for this network.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("wpapsk", "wpaeap", "open"),
				},
			},
			"wpa3_support": schema.BoolAttribute{
				MarkdownDescription: "Enable WPA 3 support (security must be `wpapsk` and PMF must be turned on).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"wpa3_transition": schema.BoolAttribute{
				MarkdownDescription: "Enable WPA 3 and WPA 2 support (security must be `wpapsk` and `wpa3_support` must be true).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"pmf_mode": schema.StringAttribute{
				MarkdownDescription: "Enable Protected Management Frames. This cannot be disabled if using WPA 3.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				Validators: []validator.String{
					stringvalidator.OneOf("required", "optional", "disabled"),
				},
			},
			"passphrase": schema.StringAttribute{
				MarkdownDescription: "The passphrase for the network, this is only required if `security` is not set to `open`.",
				Optional:            true,
				Sensitive:           true,
			},
			"hide_ssid": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not to hide the SSID from broadcast.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_guest": schema.BoolAttribute{
				MarkdownDescription: "Indicates that this is a guest WLAN and should use guest behaviors.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"multicast_enhance": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not Multicast Enhance is turned of for the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"mac_filter_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not the MAC filter is turned of for the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"mac_filter_list": schema.SetAttribute{
				MarkdownDescription: "List of MAC addresses to filter (only valid if `mac_filter_enabled` is `true`).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"mac_filter_policy": schema.StringAttribute{
				MarkdownDescription: "MAC address filter policy (only valid if `mac_filter_enabled` is `true`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("deny"),
				Validators: []validator.String{
					stringvalidator.OneOf("allow", "deny"),
				},
			},
			"radius_profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the RADIUS profile to use when security `wpaeap`.",
				Optional:            true,
			},
			"no2ghz_oui": schema.BoolAttribute{
				MarkdownDescription: "Connect high performance clients to 5 GHz only.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"l2_isolation": schema.BoolAttribute{
				MarkdownDescription: "Isolates stations on layer 2 (ethernet) level.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"proxy_arp": schema.BoolAttribute{
				MarkdownDescription: "Reduces airtime usage by allowing APs to \"proxy\" common broadcast frames as unicast.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"bss_transition": schema.BoolAttribute{
				MarkdownDescription: "Improves client roaming by providing connection details of nearby APs.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"uapsd": schema.BoolAttribute{
				MarkdownDescription: "Enable Unscheduled Automatic Power Save Delivery.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"fast_roaming_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable fast roaming, aka 802.11r.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"minimum_data_rate_2g_kbps": schema.Int64Attribute{
				MarkdownDescription: "Minimum data rate for 2G clients in Kbps.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.OneOf(0, 1000, 2000, 5500, 6000, 9000, 11000, 12000, 18000, 24000, 36000, 48000, 54000),
				},
			},
			"minimum_data_rate_5g_kbps": schema.Int64Attribute{
				MarkdownDescription: "Minimum data rate for 5G clients in Kbps.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.OneOf(0, 6000, 9000, 12000, 18000, 24000, 36000, 48000, 54000),
				},
			},
			"minrate_setting_preference": schema.StringAttribute{
				MarkdownDescription: "Minimum rate setting preference.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("auto"),
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
		},

		Blocks: map[string]schema.Block{
			"schedule": schema.ListNestedBlock{
				MarkdownDescription: "Start and stop schedules for the WLAN",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"day_of_week": schema.StringAttribute{
							MarkdownDescription: "Day of week for the block.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("sun", "mon", "tue", "wed", "thu", "fri", "sat"),
							},
						},
						"start_hour": schema.Int64Attribute{
							MarkdownDescription: "Start hour for the block (0-23).",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(0, 23),
							},
						},
						"start_minute": schema.Int64Attribute{
							MarkdownDescription: "Start minute for the block (0-59).",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							Validators: []validator.Int64{
								int64validator.Between(0, 59),
							},
						},
						"duration": schema.Int64Attribute{
							MarkdownDescription: "Length of the block in minutes.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the block.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (r *wlanFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *wlanFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wlanFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	// Convert the plan to UniFi WLAN struct
	wlan, diags := r.planToWLAN(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the WLAN
	createdWLAN, err := r.client.c.CreateWLAN(ctx, site, wlan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating WLAN",
			"Could not create WLAN: "+err.Error(),
		)
		return
	}

	// Convert response back to model
	diags = r.wlanToModel(ctx, createdWLAN, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *wlanFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wlanFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	id := state.ID.ValueString()

	// Get the WLAN from the API
	wlan, err := r.client.c.GetWLAN(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading WLAN",
			"Could not read WLAN with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.wlanToModel(ctx, wlan, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *wlanFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wlanFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	id := plan.ID.ValueString()

	// Implement UniFi API update pattern: read-merge-update
	// 1. Read existing resource from API
	existingWLAN, err := r.client.c.GetWLAN(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading WLAN for Update",
			"Could not read WLAN with ID "+id+": "+err.Error(),
		)
		return
	}

	// 2. Convert plan to WLAN struct
	planWLAN, diags := r.planToWLAN(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 3. Merge planned changes with existing values (UniFi requires full objects)
	mergedWLAN := r.mergeWLAN(existingWLAN, planWLAN)

	// 4. Update the WLAN
	updatedWLAN, err := r.client.c.UpdateWLAN(ctx, site, mergedWLAN)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating WLAN",
			"Could not update WLAN with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert response back to model
	diags = r.wlanToModel(ctx, updatedWLAN, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *wlanFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wlanFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	id := state.ID.ValueString()

	err := r.client.c.DeleteWLAN(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting WLAN",
			"Could not delete WLAN with ID "+id+": "+err.Error(),
		)
		return
	}
}

func (r *wlanFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
	} else {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	}
}

// Helper functions for conversion and merging

func (r *wlanFrameworkResource) planToWLAN(ctx context.Context, plan wlanFrameworkResourceModel) (*unifi.WLAN, diag.Diagnostics) {
	var diags diag.Diagnostics

	wlan := &unifi.WLAN{
		ID:                       plan.ID.ValueString(),
		Name:                     plan.Name.ValueString(),
		UserGroupID:              plan.UserGroupID.ValueString(),
		Security:                 plan.Security.ValueString(),
		WPA3Support:              plan.WPA3Support.ValueBool(),
		WPA3Transition:           plan.WPA3Transition.ValueBool(),
		PMFMode:                  plan.PMFMode.ValueString(),
		XPassphrase:              plan.Passphrase.ValueString(),
		HideSSID:                 plan.HideSSID.ValueBool(),
		IsGuest:                  plan.IsGuest.ValueBool(),
		MACFilterEnabled:         plan.MacFilterEnabled.ValueBool(),
		MACFilterPolicy:          plan.MacFilterPolicy.ValueString(),
		RADIUSProfileID:          plan.RadiusProfileID.ValueString(),
		No2GhzOui:                plan.No2GhzOui.ValueBool(),
		L2Isolation:              plan.L2Isolation.ValueBool(),
		ProxyArp:                 plan.ProxyArp.ValueBool(),
		BssTransition:            plan.BssTransition.ValueBool(),
		UapsdEnabled:             plan.Uapsd.ValueBool(),
		FastRoamingEnabled:       plan.FastRoamingEnabled.ValueBool(),
		MinrateSettingPreference: plan.MinrateSettingPreference.ValueString(),
		MinrateNgEnabled:         plan.MinimumDataRate2GKbps.ValueInt64() != 0,
		MinrateNgDataRateKbps:    int(plan.MinimumDataRate2GKbps.ValueInt64()),
		MinrateNaEnabled:         plan.MinimumDataRate5GKbps.ValueInt64() != 0,
		MinrateNaDataRateKbps:    int(plan.MinimumDataRate5GKbps.ValueInt64()),
		
		// Set defaults that UniFi expects
		GroupRekey:         3600,
		DTIMMode:           "default",
		WPAEnc:             "ccmp",
		WPAMode:            "wpa2",
		Enabled:            true,
		NameCombineEnabled: true,
	}

	// Handle MAC filter list
	if !plan.MacFilterList.IsNull() && !plan.MacFilterList.IsUnknown() {
		var macList []types.String
		diags.Append(plan.MacFilterList.ElementsAs(ctx, &macList, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for _, mac := range macList {
			wlan.MACFilterList = append(wlan.MACFilterList, mac.ValueString())
		}
	}

	// Handle schedule
	if !plan.Schedule.IsNull() && !plan.Schedule.IsUnknown() {
		var schedules []wlanScheduleModel
		diags.Append(plan.Schedule.ElementsAs(ctx, &schedules, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for _, sched := range schedules {
			wlan.ScheduleWithDuration = append(wlan.ScheduleWithDuration, unifi.WLANScheduleWithDuration{
				StartDaysOfWeek: []string{sched.DayOfWeek.ValueString()},
				StartHour:       int(sched.StartHour.ValueInt64()),
				StartMinute:     int(sched.StartMinute.ValueInt64()),
				DurationMinutes: int(sched.Duration.ValueInt64()),
				Name:            sched.Name.ValueString(),
			})
		}
		wlan.ScheduleEnabled = len(wlan.ScheduleWithDuration) > 0
	}

	return wlan, diags
}

func (r *wlanFrameworkResource) wlanToModel(ctx context.Context, wlan *unifi.WLAN, model *wlanFrameworkResourceModel, site string) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(wlan.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(wlan.Name)
	model.UserGroupID = types.StringValue(wlan.UserGroupID)
	model.Security = types.StringValue(wlan.Security)
	model.WPA3Support = types.BoolValue(wlan.WPA3Support)
	model.WPA3Transition = types.BoolValue(wlan.WPA3Transition)
	
	if wlan.PMFMode != "" {
		model.PMFMode = types.StringValue(wlan.PMFMode)
	} else {
		model.PMFMode = types.StringValue("disabled")
	}
	
	// Only set passphrase if it's not empty (don't overwrite sensitive data unnecessarily)
	if wlan.XPassphrase != "" {
		model.Passphrase = types.StringValue(wlan.XPassphrase)
	}
	
	model.HideSSID = types.BoolValue(wlan.HideSSID)
	model.IsGuest = types.BoolValue(wlan.IsGuest)
	model.MulticastEnhance = types.BoolValue(wlan.MulticastEnhanceEnabled)
	model.MacFilterEnabled = types.BoolValue(wlan.MACFilterEnabled)
	
	if wlan.MACFilterPolicy != "" {
		model.MacFilterPolicy = types.StringValue(wlan.MACFilterPolicy)
	} else {
		model.MacFilterPolicy = types.StringValue("deny")
	}
	
	if wlan.RADIUSProfileID != "" {
		model.RadiusProfileID = types.StringValue(wlan.RADIUSProfileID)
	} else {
		model.RadiusProfileID = types.StringNull()
	}
	
	model.No2GhzOui = types.BoolValue(wlan.No2GhzOui)
	model.L2Isolation = types.BoolValue(wlan.L2Isolation)
	model.ProxyArp = types.BoolValue(wlan.ProxyArp)
	model.BssTransition = types.BoolValue(wlan.BssTransition)
	model.Uapsd = types.BoolValue(wlan.UapsdEnabled)
	model.FastRoamingEnabled = types.BoolValue(wlan.FastRoamingEnabled)
	
	if wlan.MinrateSettingPreference != "" {
		model.MinrateSettingPreference = types.StringValue(wlan.MinrateSettingPreference)
	} else {
		model.MinrateSettingPreference = types.StringValue("auto")
	}
	
	model.MinimumDataRate2GKbps = types.Int64Value(int64(wlan.MinrateNgDataRateKbps))
	model.MinimumDataRate5GKbps = types.Int64Value(int64(wlan.MinrateNaDataRateKbps))

	// Handle MAC filter list
	if len(wlan.MACFilterList) > 0 {
		macValues := make([]attr.Value, len(wlan.MACFilterList))
		for i, mac := range wlan.MACFilterList {
			macValues[i] = types.StringValue(mac)
		}
		macSet, d := types.SetValue(types.StringType, macValues)
		diags.Append(d...)
		model.MacFilterList = macSet
	} else {
		model.MacFilterList = types.SetNull(types.StringType)
	}

	// Handle schedule - convert WLANScheduleWithDuration back to individual schedule entries
	if len(wlan.ScheduleWithDuration) > 0 {
		var scheduleValues []attr.Value
		for _, sched := range wlan.ScheduleWithDuration {
			// Each schedule can have multiple days of week, so we need to expand them
			for _, dow := range sched.StartDaysOfWeek {
				scheduleObj, d := types.ObjectValue(
					map[string]attr.Type{
						"day_of_week":   types.StringType,
						"start_hour":    types.Int64Type,
						"start_minute":  types.Int64Type,
						"duration":      types.Int64Type,
						"name":          types.StringType,
					},
					map[string]attr.Value{
						"day_of_week":   types.StringValue(dow),
						"start_hour":    types.Int64Value(int64(sched.StartHour)),
						"start_minute":  types.Int64Value(int64(sched.StartMinute)),
						"duration":      types.Int64Value(int64(sched.DurationMinutes)),
						"name":          types.StringValue(sched.Name),
					},
				)
				diags.Append(d...)
				scheduleValues = append(scheduleValues, scheduleObj)
			}
		}
		scheduleList, d := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"day_of_week":   types.StringType,
					"start_hour":    types.Int64Type,
					"start_minute":  types.Int64Type,
					"duration":      types.Int64Type,
					"name":          types.StringType,
				},
			},
			scheduleValues,
		)
		diags.Append(d...)
		model.Schedule = scheduleList
	} else {
		model.Schedule = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"day_of_week":   types.StringType,
				"start_hour":    types.Int64Type,
				"start_minute":  types.Int64Type,
				"duration":      types.Int64Type,
				"name":          types.StringType,
			},
		})
	}

	return diags
}

func (r *wlanFrameworkResource) mergeWLAN(existing *unifi.WLAN, planned *unifi.WLAN) *unifi.WLAN {
	// Start with the existing WLAN to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values
	merged.Name = planned.Name
	merged.UserGroupID = planned.UserGroupID
	merged.Security = planned.Security
	merged.WPA3Support = planned.WPA3Support
	merged.WPA3Transition = planned.WPA3Transition
	merged.PMFMode = planned.PMFMode
	merged.XPassphrase = planned.XPassphrase
	merged.HideSSID = planned.HideSSID
	merged.IsGuest = planned.IsGuest
	merged.MulticastEnhanceEnabled = planned.MulticastEnhanceEnabled
	merged.MACFilterEnabled = planned.MACFilterEnabled
	merged.MACFilterList = planned.MACFilterList
	merged.MACFilterPolicy = planned.MACFilterPolicy
	merged.RADIUSProfileID = planned.RADIUSProfileID
	merged.ScheduleWithDuration = planned.ScheduleWithDuration
	merged.ScheduleEnabled = planned.ScheduleEnabled
	merged.No2GhzOui = planned.No2GhzOui
	merged.L2Isolation = planned.L2Isolation
	merged.ProxyArp = planned.ProxyArp
	merged.BssTransition = planned.BssTransition
	merged.UapsdEnabled = planned.UapsdEnabled
	merged.FastRoamingEnabled = planned.FastRoamingEnabled
	merged.MinrateSettingPreference = planned.MinrateSettingPreference
	merged.MinrateNgEnabled = planned.MinrateNgEnabled
	merged.MinrateNgDataRateKbps = planned.MinrateNgDataRateKbps
	merged.MinrateNaEnabled = planned.MinrateNaEnabled
	merged.MinrateNaDataRateKbps = planned.MinrateNaDataRateKbps

	return &merged
}