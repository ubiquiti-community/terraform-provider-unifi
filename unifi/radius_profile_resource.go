package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

var (
	_ resource.Resource                 = &radiusProfileResource{}
	_ resource.ResourceWithImportState  = &radiusProfileResource{}
	_ resource.ResourceWithIdentity     = &radiusProfileResource{}
	_ resource.ResourceWithUpgradeState = &radiusProfileResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &radiusProfileResource{}
	_ list.ListResourceWithConfigure = &radiusProfileResource{}
)

func NewRadiusProfileResource() resource.Resource {
	return &radiusProfileResource{}
}

func NewRadiusProfileListResource() list.ListResource {
	return &radiusProfileResource{}
}

type radiusProfileResource struct {
	client *Client
}

type radiusServerModel struct {
	IP     types.String `tfsdk:"ip"`
	Port   types.Int64  `tfsdk:"port"`
	Secret types.String `tfsdk:"secret"`
}

type radiusProfileResourceModel struct {
	ID                types.String        `tfsdk:"id"`
	Site              types.String        `tfsdk:"site"`
	Name              types.String        `tfsdk:"name"`
	AccountingEnabled types.Bool          `tfsdk:"accounting_enabled"`
	InterimUpdate     types.Object        `tfsdk:"interim_update"`
	UseUSGAcctServer  types.Bool          `tfsdk:"use_usg_acct_server"`
	UseUSGAuthServer  types.Bool          `tfsdk:"use_usg_auth_server"`
	Vlan              types.Object        `tfsdk:"vlan"`
	AuthServer        []radiusServerModel `tfsdk:"auth_server"`
	AcctServer        []radiusServerModel `tfsdk:"acct_server"`
	Timeouts          timeouts.Value      `tfsdk:"timeouts"`
}

// radiusProfileInterimUpdateModel is the `interim_update` nested object
// (formerly the flat interim_update_enabled / interim_update_interval).
type radiusProfileInterimUpdateModel struct {
	Enabled  types.Bool           `tfsdk:"enabled"`
	Interval timetypes.GoDuration `tfsdk:"interval"`
}

func radiusProfileInterimUpdateAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":  types.BoolType,
		"interval": timetypes.GoDurationType{},
	}
}

// radiusProfileVlanModel is the `vlan` nested object (formerly the flat
// vlan_enabled / vlan_wlan_mode).
type radiusProfileVlanModel struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	WlanMode types.String `tfsdk:"wlan_mode"`
}

func radiusProfileVlanAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":   types.BoolType,
		"wlan_mode": types.StringType,
	}
}

// Object-level defaults reproduce the values the flat attributes used to send
// when the practitioner left a whole group out of configuration, so the
// request body on create is unchanged by the nesting.

func radiusProfileInterimUpdateDefault() types.Object {
	return types.ObjectValueMust(radiusProfileInterimUpdateAttrTypes(), map[string]attr.Value{
		"enabled":  types.BoolValue(false),
		"interval": timetypes.NewGoDurationValue(time.Hour),
	})
}

func radiusProfileVlanDefault() types.Object {
	return types.ObjectValueMust(radiusProfileVlanAttrTypes(), map[string]attr.Value{
		"enabled":   types.BoolValue(false),
		"wlan_mode": types.StringValue(""),
	})
}

// radiusProfileIdentityModel describes the resource identity data model.
type radiusProfileIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

// radiusProfileListConfigModel describes the list configuration model.
type radiusProfileListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// radiusProfileListFilterModel represents a single name/value filter entry.
type radiusProfileListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *radiusProfileResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_radius_profile"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *radiusProfileResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"site": identityschema.StringAttribute{
				OptionalForImport: true,
			},
		},
	}
}

func (r *radiusProfileResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		// v1: interim_update_interval changed from Int64 (seconds) to GoDuration.
		// v2: interim_update_* and vlan_* nested into `interim_update` and `vlan`.
		Version:             2,
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
			"interim_update": schema.SingleNestedAttribute{
				MarkdownDescription: "RADIUS interim accounting update settings.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(radiusProfileInterimUpdateDefault()),
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether to use interim_update.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"interval": schema.StringAttribute{
						MarkdownDescription: "Specifies the RADIUS interim update interval, as a Go " +
							"duration string (e.g. `1h`, `3600s`). Defaults to `1h0m0s`.",
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
						Default:    stringdefault.StaticString("1h0m0s"),
						Validators: []validator.String{
							validators.GoDurationMultipleOf(time.Second),
						},
					},
				},
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
			"vlan": schema.SingleNestedAttribute{
				MarkdownDescription: "Dynamic VLAN assignment settings.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(radiusProfileVlanDefault()),
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether to use vlan on wired connections.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"wlan_mode": schema.StringAttribute{
						MarkdownDescription: "Specifies whether to use vlan on wireless connections. Must be one of `disabled`, `optional`, or `required`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						Validators: []validator.String{
							stringvalidator.OneOf("disabled", "optional", "required"),
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
		Blocks: map[string]schema.Block{
			"auth_server": schema.ListNestedBlock{
				MarkdownDescription: "RADIUS authentication servers.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							MarkdownDescription: "IP address of the authentication server. " +
								"Optional: the controller-managed default profile (e.g. the " +
								"one created when a gateway RADIUS/VPN service is enabled, with " +
								"`use_usg_auth_server = true`) returns a server entry without an " +
								"IP, so importing it must not force one.",
							Optional: true,
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
						"secret": schema.StringAttribute{
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
							MarkdownDescription: "IP address of the accounting server. " +
								"Optional: the controller-managed default profile returns a " +
								"server entry without an IP, so importing it must not force one.",
							Optional: true,
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
						"secret": schema.StringAttribute{
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

// UpgradeState migrates prior-version state to the current schema:
//
//	v0 -> current: interim_update_interval changed from integer seconds to a
//	    GoDuration string.
//	v1 -> current: the flat interim_update_* and vlan_* attributes moved into
//	    the nested `interim_update` and `vlan` objects (see
//	    nestRadiusProfileState).
func (r *radiusProfileResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	upgrader := func(rewrite func(state map[string]any)) resource.StateUpgrader {
		return resource.StateUpgrader{
			StateUpgrader: func(
				ctx context.Context,
				req resource.UpgradeStateRequest,
				resp *resource.UpgradeStateResponse,
			) {
				if req.RawState == nil {
					return
				}
				dv, err := util.UpgradeRawState(
					schemaType,
					req.RawState.JSON,
					func(state map[string]any) {
						rewrite(state)
						nestRadiusProfileState(state)
					},
				)
				if err != nil {
					resp.Diagnostics.AddError("Failed to upgrade RADIUS profile state", err.Error())
					return
				}
				resp.DynamicValue = dv
			},
		}
	}

	return map[int64]resource.StateUpgrader{
		0: upgrader(func(state map[string]any) {
			util.SetDurationField(state, "interim_update_interval", time.Second)
		}),
		1: upgrader(func(map[string]any) {}),
	}
}

// nestRadiusProfileState rewrites decoded prior state so the flat
// interim_update_* and vlan_* keys move under their nested objects.
func nestRadiusProfileState(state map[string]any) {
	util.NestFields(state, "interim_update", map[string]string{
		"interim_update_enabled":  "enabled",
		"interim_update_interval": "interval",
	})
	util.NestFields(state, "vlan", map[string]string{
		"vlan_enabled":   "enabled",
		"vlan_wlan_mode": "wlan_mode",
	})
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

	createTimeout, timeoutDiags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	radiusProfile, d := r.modelToRadiusProfile(ctx, &data)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	resp.Diagnostics.Append(r.radiusProfileToModel(ctx, createdRadiusProfile, &data, site)...)

	identity := radiusProfileIdentityModel{
		ID:   data.ID,
		Site: types.StringValue(site),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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

	readTimeout, timeoutDiags := data.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// Read identity, falling back to state for resources created before
	// identity support (or refreshed from a string import).
	var identity radiusProfileIdentityModel
	identityStored := req.Identity != nil && !req.Identity.Raw.IsFullyNull()
	if identityStored {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if identity.ID.IsNull() || identity.ID.ValueString() == "" {
		identity.ID = data.ID
	}
	if identity.Site.IsNull() || identity.Site.ValueString() == "" {
		identity.Site = data.Site
	}

	id := data.ID.ValueString()
	if id == "" {
		// Identity-only state (e.g. the refresh right after an identity-based
		// import of an old state): look the profile up by the identity id.
		id = identity.ID.ValueString()
	}

	site := identity.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	radiusProfile, err := r.client.GetRADIUSProfile(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading RADIUS Profile",
			"Could not read RADIUS profile with ID "+id+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.radiusProfileToModel(ctx, radiusProfile, &data, site)...)

	// Terraform rejects any modification of a stored identity (even filling a
	// previously-null attribute), so pass a stored identity through untouched
	// (resp.Identity is pre-populated from it) and only derive a fresh one
	// from state when none exists yet.
	if !identityStored {
		identity = radiusProfileIdentityModel{
			ID:   data.ID,
			Site: types.StringValue(site),
		}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
	}
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

	updateTimeout, timeoutDiags := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	radiusProfile, d := r.modelToRadiusProfile(ctx, &state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	radiusProfile.ID = state.ID.ValueString()

	updatedRadiusProfile, err := r.client.UpdateRADIUSProfile(ctx, site, radiusProfile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating RADIUS Profile",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.radiusProfileToModel(ctx, updatedRadiusProfile, &state, site)...)

	state.Timeouts = plan.Timeouts

	// Pass a stored identity through untouched; derive it from state only for
	// resources created before identity support.
	if req.Identity == nil || req.Identity.Raw.IsFullyNull() {
		identity := radiusProfileIdentityModel{
			ID:   state.ID,
			Site: types.StringValue(site),
		}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
	}
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

	deleteTimeout, timeoutDiags := data.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

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
	// Import by resource identity (import block with identity, Terraform 1.12+).
	if req.ID == "" {
		var identity radiusProfileIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Invalid Import Identity",
				"RADIUS profile identity must have `id` set.",
			)
			return
		}
		resp.Diagnostics.Append(
			resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
		if !identity.Site.IsNull() && identity.Site.ValueString() != "" {
			resp.Diagnostics.Append(
				resp.State.SetAttribute(ctx, path.Root("site"), identity.Site)...)
		}
		return
	}

	// Import by ID string (terraform import CLI, or import block with id set).
	idParts := strings.Split(req.ID, ":")

	if len(idParts) == 2 {
		site := idParts[0]
		id := idParts[1]

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	if len(idParts) == 1 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), req.ID)...)
		return
	}

	resp.Diagnostics.AddError(
		"Invalid Import ID",
		"Import ID must be in format 'site:id' or 'id'",
	)
}

func (r *radiusProfileResource) applyPlanToState(
	ctx context.Context,
	plan *radiusProfileResourceModel,
	state *radiusProfileResourceModel,
) {
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.AccountingEnabled.IsNull() && !plan.AccountingEnabled.IsUnknown() {
		state.AccountingEnabled = plan.AccountingEnabled
	}
	// Nested groups: every leaf the practitioner set in the plan is re-asserted
	// on top of state, exactly as the flat attributes were.
	state.InterimUpdate = util.OverlayKnownObject(ctx, plan.InterimUpdate, state.InterimUpdate)
	if !plan.UseUSGAcctServer.IsNull() && !plan.UseUSGAcctServer.IsUnknown() {
		state.UseUSGAcctServer = plan.UseUSGAcctServer
	}
	if !plan.UseUSGAuthServer.IsNull() && !plan.UseUSGAuthServer.IsUnknown() {
		state.UseUSGAuthServer = plan.UseUSGAuthServer
	}
	state.Vlan = util.OverlayKnownObject(ctx, plan.Vlan, state.Vlan)
	if plan.AuthServer != nil {
		state.AuthServer = plan.AuthServer
	}
	if plan.AcctServer != nil {
		state.AcctServer = plan.AcctServer
	}
}

func (r *radiusProfileResource) modelToRadiusProfile(
	ctx context.Context,
	model *radiusProfileResourceModel,
) (*unifi.RADIUSProfile, diag.Diagnostics) {
	var diags diag.Diagnostics

	radiusProfile := &unifi.RADIUSProfile{
		Name:              model.Name.ValueString(),
		AccountingEnabled: model.AccountingEnabled.ValueBool(),
		UseUsgAcctServer:  model.UseUSGAcctServer.ValueBool(),
		UseUsgAuthServer:  model.UseUSGAuthServer.ValueBool(),
	}

	// Nested groups. A null/unknown group contributes nothing, exactly as its
	// flat attributes did when unset.
	if iu, ok, d := util.ObjectAs[radiusProfileInterimUpdateModel](ctx, model.InterimUpdate); ok {
		radiusProfile.InterimUpdateEnabled = iu.Enabled.ValueBool()
		radiusProfile.InterimUpdateInterval = util.DurationUnitsPtr(iu.Interval, time.Second)
	} else {
		diags.Append(d...)
	}
	if vlan, ok, d := util.ObjectAs[radiusProfileVlanModel](ctx, model.Vlan); ok {
		radiusProfile.VLANEnabled = vlan.Enabled.ValueBool()
		radiusProfile.VLANWLANMode = vlan.WlanMode.ValueString()
	} else {
		diags.Append(d...)
	}

	for _, authServer := range model.AuthServer {
		radiusProfile.AuthServers = append(
			radiusProfile.AuthServers,
			unifi.RADIUSProfileAuthServers{
				IP:     authServer.IP.ValueString(),
				Port:   authServer.Port.ValueInt64Pointer(),
				Secret: authServer.Secret.ValueString(),
			},
		)
	}

	for _, acctServer := range model.AcctServer {
		radiusProfile.AcctServers = append(
			radiusProfile.AcctServers,
			unifi.RADIUSProfileAcctServers{
				IP:     acctServer.IP.ValueString(),
				Port:   acctServer.Port.ValueInt64Pointer(),
				Secret: acctServer.Secret.ValueString(),
			},
		)
	}

	return radiusProfile, diags
}

func (r *radiusProfileResource) radiusProfileToModel(
	ctx context.Context,
	radiusProfile *unifi.RADIUSProfile,
	model *radiusProfileResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(radiusProfile.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(radiusProfile.Name)
	model.AccountingEnabled = types.BoolValue(radiusProfile.AccountingEnabled)
	model.UseUSGAcctServer = types.BoolValue(radiusProfile.UseUsgAcctServer)
	model.UseUSGAuthServer = types.BoolValue(radiusProfile.UseUsgAuthServer)

	interimUpdate, d := types.ObjectValueFrom(
		ctx,
		radiusProfileInterimUpdateAttrTypes(),
		radiusProfileInterimUpdateModel{
			Enabled:  types.BoolValue(radiusProfile.InterimUpdateEnabled),
			Interval: util.DurationPtrValue(radiusProfile.InterimUpdateInterval, time.Second),
		},
	)
	diags.Append(d...)
	model.InterimUpdate = interimUpdate

	vlan, d := types.ObjectValueFrom(ctx, radiusProfileVlanAttrTypes(), radiusProfileVlanModel{
		Enabled:  types.BoolValue(radiusProfile.VLANEnabled),
		WlanMode: types.StringValue(radiusProfile.VLANWLANMode),
	})
	diags.Append(d...)
	model.Vlan = vlan

	model.AuthServer = []radiusServerModel{}
	for _, authServer := range radiusProfile.AuthServers {
		model.AuthServer = append(model.AuthServer, radiusServerModel{
			IP:     util.StringValueOrNull(authServer.IP),
			Port:   types.Int64PointerValue(authServer.Port),
			Secret: types.StringValue(authServer.Secret),
		})
	}

	model.AcctServer = []radiusServerModel{}
	for _, acctServer := range radiusProfile.AcctServers {
		model.AcctServer = append(model.AcctServer, radiusServerModel{
			IP:     util.StringValueOrNull(acctServer.IP),
			Port:   types.Int64PointerValue(acctServer.Port),
			Secret: types.StringValue(acctServer.Secret),
		})
	}

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *radiusProfileResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List RADIUS profiles in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list RADIUS profiles from.",
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
func (r *radiusProfileResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config radiusProfileListConfigModel

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
	var filters []radiusProfileListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	profiles, err := r.client.ListRADIUSProfile(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing RADIUS Profiles", "Could not list RADIUS profiles: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, profile := range profiles {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if profile.Name != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if profile.Name != "" {
				result.DisplayName = profile.Name
			} else {
				result.DisplayName = profile.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(profile.ID),
				)...,
			)
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("site"),
					types.StringValue(site),
				)...,
			)

			// Convert to model.
			var model radiusProfileResourceModel
			result.Diagnostics.Append(r.radiusProfileToModel(ctx, &profile, &model, site)...)
			model.Timeouts = timeoutsNullValue()
			result.Diagnostics.Append(result.Resource.Set(ctx, model)...)

			if !push(result) {
				return
			}
		}
	}
}
