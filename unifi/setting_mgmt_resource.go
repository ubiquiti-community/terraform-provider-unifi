package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var (
	_ resource.Resource                = &settingMgmtResource{}
	_ resource.ResourceWithImportState = &settingMgmtResource{}
)

func NewSettingMgmtResource() resource.Resource {
	return &settingMgmtResource{}
}

type settingMgmtResource struct {
	client *Client
}

type sshKeyModel struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Key     types.String `tfsdk:"key"`
	Comment types.String `tfsdk:"comment"`
}

type settingMgmtResourceModel struct {
	ID          types.String  `tfsdk:"id"`
	Site        types.String  `tfsdk:"site"`
	AutoUpgrade types.Bool    `tfsdk:"auto_upgrade"`
	SSHEnabled  types.Bool    `tfsdk:"ssh_enabled"`
	SSHKey      []sshKeyModel `tfsdk:"ssh_key"`
}

func (r *settingMgmtResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_setting_mgmt"
}

func (r *settingMgmtResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages settings for a unifi site.",

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
			"auto_upgrade": schema.BoolAttribute{
				MarkdownDescription: "Automatically upgrade device firmware.",
				Optional:            true,
			},
			"ssh_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable SSH authentication.",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"ssh_key": schema.SetNestedBlock{
				MarkdownDescription: "SSH key.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of SSH key.",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Type of SSH key, e.g. ssh-rsa.",
							Required:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "Public SSH key.",
							Optional:            true,
						},
						"comment": schema.StringAttribute{
							MarkdownDescription: "Comment.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (r *settingMgmtResource) Configure(
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

func (r *settingMgmtResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data settingMgmtResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	setting := r.modelToSettingMgmt(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Setting management uses update for both create and update operations
	createdSetting, err := r.client.UpdateSettingMgmt(ctx, site, setting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Setting Management",
			"Could not create setting management, unexpected error: "+err.Error(),
		)
		return
	}

	r.settingMgmtToModel(ctx, createdSetting, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingMgmtResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data settingMgmtResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	setting, err := r.client.GetSettingMgmt(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Setting Management",
			"Could not read setting management with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.settingMgmtToModel(ctx, setting, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingMgmtResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state settingMgmtResourceModel
	var plan settingMgmtResourceModel

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

	setting := r.modelToSettingMgmt(ctx, &state)
	setting.ID = state.ID.ValueString()

	updatedSetting, err := r.client.UpdateSettingMgmt(ctx, site, setting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Setting Management",
			"Could not update setting management, unexpected error: "+err.Error(),
		)
		return
	}

	r.settingMgmtToModel(ctx, updatedSetting, &state, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *settingMgmtResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data settingMgmtResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Setting management cannot be deleted, it's a configuration resource
	// Just remove from state
}

func (r *settingMgmtResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) == 2 {
		site := idParts[0]
		id := idParts[1]

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	if len(idParts) == 1 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		return
	}

	resp.Diagnostics.AddError(
		"Invalid Import ID",
		"Import ID must be in format 'site:id' or 'id'",
	)
}

func (r *settingMgmtResource) applyPlanToState(
	_ context.Context,
	plan *settingMgmtResourceModel,
	state *settingMgmtResourceModel,
) {
	if !plan.AutoUpgrade.IsNull() && !plan.AutoUpgrade.IsUnknown() {
		state.AutoUpgrade = plan.AutoUpgrade
	}
	if !plan.SSHEnabled.IsNull() && !plan.SSHEnabled.IsUnknown() {
		state.SSHEnabled = plan.SSHEnabled
	}
	if plan.SSHKey != nil {
		state.SSHKey = plan.SSHKey
	}
}

func (r *settingMgmtResource) modelToSettingMgmt(
	_ context.Context,
	model *settingMgmtResourceModel,
) *unifi.SettingMgmt {
	setting := &unifi.SettingMgmt{}

	if !model.AutoUpgrade.IsNull() {
		setting.AutoUpgrade = model.AutoUpgrade.ValueBool()
	}
	if !model.SSHEnabled.IsNull() {
		setting.XSshEnabled = model.SSHEnabled.ValueBool()
	}

	for _, sshKey := range model.SSHKey {
		setting.XSshKeys = append(setting.XSshKeys, unifi.SettingMgmtXSshKeys{
			Name:    sshKey.Name.ValueString(),
			KeyType: sshKey.Type.ValueString(),
			Key:     sshKey.Key.ValueString(),
			Comment: sshKey.Comment.ValueString(),
		})
	}

	return setting
}

func (r *settingMgmtResource) settingMgmtToModel(
	_ context.Context,
	setting *unifi.SettingMgmt,
	model *settingMgmtResourceModel,
	site string,
) {
	model.ID = types.StringValue(setting.ID)
	model.Site = types.StringValue(site)
	model.AutoUpgrade = types.BoolValue(setting.AutoUpgrade)
	model.SSHEnabled = types.BoolValue(setting.XSshEnabled)

	model.SSHKey = []sshKeyModel{}
	for _, sshKey := range setting.XSshKeys {
		model.SSHKey = append(model.SSHKey, sshKeyModel{
			Name:    types.StringValue(sshKey.Name),
			Type:    types.StringValue(sshKey.KeyType),
			Key:     types.StringValue(sshKey.Key),
			Comment: types.StringValue(sshKey.Comment),
		})
	}
}
