package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var (
	_ resource.Resource                = &settingRadiusResource{}
	_ resource.ResourceWithImportState = &settingRadiusResource{}
)

func NewSettingRadiusResource() resource.Resource {
	return &settingRadiusResource{}
}

type settingRadiusResource struct {
	client *Client
}

type settingRadiusResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Site                  types.String `tfsdk:"site"`
	AccountingEnabled     types.Bool   `tfsdk:"accounting_enabled"`
	InterimUpdateInterval types.Int64  `tfsdk:"interim_update_interval"`
	// Add other RADIUS settings fields as needed
}

func (r *settingRadiusResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_setting_radius"
}

func (r *settingRadiusResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages RADIUS settings for a unifi site.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the settings.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the settings with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"accounting_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable RADIUS accounting.",
				Optional:            true,
			},
			"interim_update_interval": schema.Int64Attribute{
				MarkdownDescription: "Interim update interval in seconds.",
				Optional:            true,
			},
		},
	}
}

func (r *settingRadiusResource) Configure(
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

func (r *settingRadiusResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data settingRadiusResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	setting := r.modelToSettingRadius(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdSetting, err := r.client.UpdateSettingRadius(ctx, site, setting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Setting RADIUS",
			"Could not create setting RADIUS, unexpected error: "+err.Error(),
		)
		return
	}

	r.settingRadiusToModel(ctx, createdSetting, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingRadiusResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data settingRadiusResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	setting, err := r.client.GetSettingRadius(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Setting RADIUS",
			"Could not read setting RADIUS: "+err.Error(),
		)
		return
	}

	r.settingRadiusToModel(ctx, setting, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingRadiusResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state settingRadiusResourceModel
	var plan settingRadiusResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	setting := r.modelToSettingRadius(ctx, &state)

	updatedSetting, err := r.client.UpdateSettingRadius(ctx, site, setting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Setting RADIUS",
			"Could not update setting RADIUS, unexpected error: "+err.Error(),
		)
		return
	}

	r.settingRadiusToModel(ctx, updatedSetting, &state, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *settingRadiusResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	// RADIUS settings are typically reset to defaults rather than deleted
}

func (r *settingRadiusResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), req.ID)...)
}

func (r *settingRadiusResource) applyPlanToState(
	_ context.Context,
	plan *settingRadiusResourceModel,
	state *settingRadiusResourceModel,
) {
	if !plan.AccountingEnabled.IsNull() && !plan.AccountingEnabled.IsUnknown() {
		state.AccountingEnabled = plan.AccountingEnabled
	}
	if !plan.InterimUpdateInterval.IsNull() && !plan.InterimUpdateInterval.IsUnknown() {
		state.InterimUpdateInterval = plan.InterimUpdateInterval
	}
}

func (r *settingRadiusResource) modelToSettingRadius(
	_ context.Context,
	model *settingRadiusResourceModel,
) *unifi.SettingRadius {
	setting := &unifi.SettingRadius{}

	if !model.AccountingEnabled.IsNull() {
		setting.AccountingEnabled = model.AccountingEnabled.ValueBool()
	}
	if !model.InterimUpdateInterval.IsNull() {
		setting.InterimUpdateInterval = int(model.InterimUpdateInterval.ValueInt64())
	}

	return setting
}

func (r *settingRadiusResource) settingRadiusToModel(
	_ context.Context,
	setting *unifi.SettingRadius,
	model *settingRadiusResourceModel,
	site string,
) {
	model.ID = types.StringValue(setting.ID)
	model.Site = types.StringValue(site)
	model.AccountingEnabled = types.BoolValue(setting.AccountingEnabled)
	model.InterimUpdateInterval = types.Int64Value(int64(setting.InterimUpdateInterval))
}
