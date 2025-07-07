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
	_ resource.Resource                = &settingUSGResource{}
	_ resource.ResourceWithImportState = &settingUSGResource{}
)

func NewSettingUSGResource() resource.Resource {
	return &settingUSGResource{}
}

type settingUSGResource struct {
	client *Client
}

type settingUSGResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Site                types.String `tfsdk:"site"`
	MulticastDNSEnabled types.Bool   `tfsdk:"multicast_dns_enabled"`
}

func (r *settingUSGResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting_usg"
}

func (r *settingUSGResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages USG settings for a unifi site.",

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
			"multicast_dns_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable multicast DNS.",
				Optional:            true,
			},
		},
	}
}

func (r *settingUSGResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *settingUSGResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data settingUSGResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	setting := r.modelToSettingUSG(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdSetting, err := r.client.Client.UpdateSettingUsg(ctx, site, setting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Setting USG",
			"Could not create setting USG, unexpected error: "+err.Error(),
		)
		return
	}

	r.settingUSGToModel(ctx, createdSetting, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingUSGResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data settingUSGResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	setting, err := r.client.Client.GetSettingUsg(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Setting USG",
			"Could not read setting USG: "+err.Error(),
		)
		return
	}

	r.settingUSGToModel(ctx, setting, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingUSGResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state settingUSGResourceModel
	var plan settingUSGResourceModel

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

	setting := r.modelToSettingUSG(ctx, &state)

	updatedSetting, err := r.client.Client.UpdateSettingUsg(ctx, site, setting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Setting USG",
			"Could not update setting USG, unexpected error: "+err.Error(),
		)
		return
	}

	r.settingUSGToModel(ctx, updatedSetting, &state, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *settingUSGResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// USG settings are typically reset to defaults rather than deleted
}

func (r *settingUSGResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), req.ID)...)
}

func (r *settingUSGResource) applyPlanToState(ctx context.Context, plan *settingUSGResourceModel, state *settingUSGResourceModel) {
	if !plan.MulticastDNSEnabled.IsNull() && !plan.MulticastDNSEnabled.IsUnknown() {
		state.MulticastDNSEnabled = plan.MulticastDNSEnabled
	}
}

func (r *settingUSGResource) modelToSettingUSG(ctx context.Context, model *settingUSGResourceModel) *unifi.SettingUsg {
	setting := &unifi.SettingUsg{}

	if !model.MulticastDNSEnabled.IsNull() {
		setting.MdnsEnabled = model.MulticastDNSEnabled.ValueBool()
	}

	return setting
}

func (r *settingUSGResource) settingUSGToModel(ctx context.Context, setting *unifi.SettingUsg, model *settingUSGResourceModel, site string) {
	model.ID = types.StringValue(setting.ID)
	model.Site = types.StringValue(site)
	model.MulticastDNSEnabled = types.BoolValue(setting.MdnsEnabled)
}
