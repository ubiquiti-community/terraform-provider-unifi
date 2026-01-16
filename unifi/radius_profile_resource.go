package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

var (
	_ resource.Resource                = &radiusProfileResource{}
	_ resource.ResourceWithImportState = &radiusProfileResource{}
)

func NewRadiusProfileResource() resource.Resource {
	return &radiusProfileResource{}
}

type radiusProfileResource struct {
	client *Client
}

type radiusServerModel struct {
	IP      types.String `tfsdk:"ip"`
	Port    types.Int64  `tfsdk:"port"`
	XSecret types.String `tfsdk:"x_secret"`
}

type radiusProfileResourceModel struct {
	ID                    types.String        `tfsdk:"id"`
	Site                  types.String        `tfsdk:"site"`
	Name                  types.String        `tfsdk:"name"`
	AccountingEnabled     types.Bool          `tfsdk:"accounting_enabled"`
	InterimUpdateEnabled  types.Bool          `tfsdk:"interim_update_enabled"`
	InterimUpdateInterval types.Int64         `tfsdk:"interim_update_interval"`
	UseUSGAcctServer      types.Bool          `tfsdk:"use_usg_acct_server"`
	UseUSGAuthServer      types.Bool          `tfsdk:"use_usg_auth_server"`
	VlanEnabled           types.Bool          `tfsdk:"vlan_enabled"`
	VlanWlanMode          types.String        `tfsdk:"vlan_wlan_mode"`
	AuthServer            []radiusServerModel `tfsdk:"auth_server"`
	AcctServer            []radiusServerModel `tfsdk:"acct_server"`
}

func (r *radiusProfileResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_radius_profile"
}

func (r *radiusProfileResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages RADIUS profiles.",

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
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the profile.",
				Required:            true,
			},
			"accounting_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to use RADIUS accounting.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"interim_update_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to use interim_update.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"interim_update_interval": schema.Int64Attribute{
				MarkdownDescription: "Specifies interim_update interval.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
			},
			"use_usg_acct_server": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to use usg as a RADIUS accounting server.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"use_usg_auth_server": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to use usg as a RADIUS authentication server.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"vlan_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to use vlan on wired connections.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"vlan_wlan_mode": schema.StringAttribute{
				MarkdownDescription: "Specifies whether to use vlan on wireless connections. Must be one of `disabled`, `optional`, or `required`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.OneOf("disabled", "optional", "required"),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"auth_server": schema.ListNestedBlock{
				MarkdownDescription: "RADIUS authentication servers.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							MarkdownDescription: "IP address of authentication service server.",
							Required:            true,
							Validators: []validator.String{
								validators.IPv4Validator(),
							},
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "Port of authentication service.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1812),
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"x_secret": schema.StringAttribute{
							MarkdownDescription: "Shared secret for authentication server.",
							Required:            true,
							Sensitive:           true,
						},
					},
				},
			},
			"acct_server": schema.ListNestedBlock{
				MarkdownDescription: "RADIUS accounting servers.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							MarkdownDescription: "IP address of accounting service server.",
							Required:            true,
							Validators: []validator.String{
								validators.IPv4Validator(),
							},
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "Port of accounting service.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1813),
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"x_secret": schema.StringAttribute{
							MarkdownDescription: "Shared secret for accounting server.",
							Required:            true,
							Sensitive:           true,
						},
					},
				},
			},
		},
	}
}

func (r *radiusProfileResource) Configure(
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

func (r *radiusProfileResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data radiusProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	radiusProfile := r.modelToRadiusProfile(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdRadiusProfile, err := r.client.CreateRADIUSProfile(ctx, site, radiusProfile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating RADIUS Profile",
			err.Error(),
		)
		return
	}

	r.radiusProfileToModel(ctx, createdRadiusProfile, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *radiusProfileResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data radiusProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	radiusProfile, err := r.client.GetRADIUSProfile(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading RADIUS Profile",
			"Could not read RADIUS profile with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.radiusProfileToModel(ctx, radiusProfile, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *radiusProfileResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state radiusProfileResourceModel
	var plan radiusProfileResourceModel

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

	radiusProfile := r.modelToRadiusProfile(ctx, &state)
	radiusProfile.ID = state.ID.ValueString()

	updatedRadiusProfile, err := r.client.UpdateRADIUSProfile(ctx, site, radiusProfile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating RADIUS Profile",
			err.Error(),
		)
		return
	}

	r.radiusProfileToModel(ctx, updatedRadiusProfile, &state, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *radiusProfileResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data radiusProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeleteRADIUSProfile(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting RADIUS Profile",
			err.Error(),
		)
		return
	}
}

func (r *radiusProfileResource) ImportState(
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

func (r *radiusProfileResource) applyPlanToState(
	_ context.Context,
	plan *radiusProfileResourceModel,
	state *radiusProfileResourceModel,
) {
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.AccountingEnabled.IsNull() && !plan.AccountingEnabled.IsUnknown() {
		state.AccountingEnabled = plan.AccountingEnabled
	}
	if !plan.InterimUpdateEnabled.IsNull() && !plan.InterimUpdateEnabled.IsUnknown() {
		state.InterimUpdateEnabled = plan.InterimUpdateEnabled
	}
	if !plan.InterimUpdateInterval.IsNull() && !plan.InterimUpdateInterval.IsUnknown() {
		state.InterimUpdateInterval = plan.InterimUpdateInterval
	}
	if !plan.UseUSGAcctServer.IsNull() && !plan.UseUSGAcctServer.IsUnknown() {
		state.UseUSGAcctServer = plan.UseUSGAcctServer
	}
	if !plan.UseUSGAuthServer.IsNull() && !plan.UseUSGAuthServer.IsUnknown() {
		state.UseUSGAuthServer = plan.UseUSGAuthServer
	}
	if !plan.VlanEnabled.IsNull() && !plan.VlanEnabled.IsUnknown() {
		state.VlanEnabled = plan.VlanEnabled
	}
	if !plan.VlanWlanMode.IsNull() && !plan.VlanWlanMode.IsUnknown() {
		state.VlanWlanMode = plan.VlanWlanMode
	}
	if plan.AuthServer != nil {
		state.AuthServer = plan.AuthServer
	}
	if plan.AcctServer != nil {
		state.AcctServer = plan.AcctServer
	}
}

func (r *radiusProfileResource) modelToRadiusProfile(
	_ context.Context,
	model *radiusProfileResourceModel,
) *unifi.RADIUSProfile {
	radiusProfile := &unifi.RADIUSProfile{
		Name:                  model.Name.ValueString(),
		AccountingEnabled:     model.AccountingEnabled.ValueBool(),
		InterimUpdateEnabled:  model.InterimUpdateEnabled.ValueBool(),
		InterimUpdateInterval: model.InterimUpdateInterval.ValueInt64(),
		UseUsgAcctServer:      model.UseUSGAcctServer.ValueBool(),
		UseUsgAuthServer:      model.UseUSGAuthServer.ValueBool(),
		VLANEnabled:           model.VlanEnabled.ValueBool(),
		VLANWLANMode:          model.VlanWlanMode.ValueString(),
	}

	for _, authServer := range model.AuthServer {
		radiusProfile.AuthServers = append(
			radiusProfile.AuthServers,
			unifi.RADIUSProfileAuthServers{
				IP:      authServer.IP.ValueString(),
				Port:    authServer.Port.ValueInt64(),
				XSecret: authServer.XSecret.ValueString(),
			},
		)
	}

	for _, acctServer := range model.AcctServer {
		radiusProfile.AcctServers = append(
			radiusProfile.AcctServers,
			unifi.RADIUSProfileAcctServers{
				IP:      acctServer.IP.ValueString(),
				Port:    acctServer.Port.ValueInt64(),
				XSecret: acctServer.XSecret.ValueString(),
			},
		)
	}

	return radiusProfile
}

func (r *radiusProfileResource) radiusProfileToModel(
	_ context.Context,
	radiusProfile *unifi.RADIUSProfile,
	model *radiusProfileResourceModel,
	site string,
) {
	model.ID = types.StringValue(radiusProfile.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(radiusProfile.Name)
	model.AccountingEnabled = types.BoolValue(radiusProfile.AccountingEnabled)
	model.InterimUpdateEnabled = types.BoolValue(radiusProfile.InterimUpdateEnabled)
	model.InterimUpdateInterval = types.Int64Value(int64(radiusProfile.InterimUpdateInterval))
	model.UseUSGAcctServer = types.BoolValue(radiusProfile.UseUsgAcctServer)
	model.UseUSGAuthServer = types.BoolValue(radiusProfile.UseUsgAuthServer)
	model.VlanEnabled = types.BoolValue(radiusProfile.VLANEnabled)
	model.VlanWlanMode = types.StringValue(radiusProfile.VLANWLANMode)

	model.AuthServer = []radiusServerModel{}
	for _, authServer := range radiusProfile.AuthServers {
		model.AuthServer = append(model.AuthServer, radiusServerModel{
			IP:      types.StringValue(authServer.IP),
			Port:    types.Int64Value(int64(authServer.Port)),
			XSecret: types.StringValue(authServer.XSecret),
		})
	}

	model.AcctServer = []radiusServerModel{}
	for _, acctServer := range radiusProfile.AcctServers {
		model.AcctServer = append(model.AcctServer, radiusServerModel{
			IP:      types.StringValue(acctServer.IP),
			Port:    types.Int64Value(int64(acctServer.Port)),
			XSecret: types.StringValue(acctServer.XSecret),
		})
	}
}
