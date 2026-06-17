package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &radiusUserResource{}
	_ resource.ResourceWithImportState = &radiusUserResource{}
	_ resource.ResourceWithIdentity    = &radiusUserResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &radiusUserResource{}
	_ list.ListResourceWithConfigure = &radiusUserResource{}
)

func NewRadiusUserResource() resource.Resource {
	return &radiusUserResource{}
}

func NewRadiusUserListResource() list.ListResource {
	return &radiusUserResource{}
}

// radiusUserResource defines the resource implementation.
type radiusUserResource struct {
	client *Client
}

// radiusUserResourceModel describes the resource data model.
type radiusUserResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	Site             types.String   `tfsdk:"site"`
	Name             types.String   `tfsdk:"name"`
	Password         types.String   `tfsdk:"password"`
	TunnelType       types.Int64    `tfsdk:"tunnel_type"`
	TunnelMediumType types.Int64    `tfsdk:"tunnel_medium_type"`
	NetworkID        types.String   `tfsdk:"network_id"`
	VLAN             types.Int64    `tfsdk:"vlan"`
	TunnelConfigType types.String   `tfsdk:"tunnel_config_type"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// radiusUserListConfigModel describes the list configuration model.
type radiusUserListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// radiusUserListFilterModel represents a single name/value filter entry.
type radiusUserListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *radiusUserResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_radius_user"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *radiusUserResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

func (r *radiusUserResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a RADIUS user account

To authenticate devices based on MAC address, use the MAC address as the username and password under client creation.
Convert lowercase letters to uppercase, and also remove colons or periods from the MAC address.

ATTENTION: If the user profile does not include a VLAN, the client will fall back to the untagged VLAN.

NOTE: MAC-based authentication accounts can only be used for wireless and wired clients. L2TP remote access does not apply.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the account.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the account with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the account.",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the account.",
				Required:            true,
				Sensitive:           true,
			},
			"tunnel_type": schema.Int64Attribute{
				MarkdownDescription: "See [RFC 2868](https://www.rfc-editor.org/rfc/rfc2868) section 3.1. " +
					"Valid values are 1-13; `13` (VLAN) is the most common.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(3),
				Validators: []validator.Int64{
					int64validator.Between(1, 13),
				},
			},
			"tunnel_medium_type": schema.Int64Attribute{
				MarkdownDescription: "See [RFC 2868](https://www.rfc-editor.org/rfc/rfc2868) section 3.2",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(6),
				Validators: []validator.Int64{
					int64validator.Between(1, 15),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID of the network for this account. When set and `vlan` is omitted, the account inherits that network's VLAN (so RADIUS/MAB VLAN assignment is applied).",
				Optional:            true,
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "VLAN assigned to the account. If omitted but `network_id` is set, it is derived from that network's VLAN. If neither is set, the client falls back to the untagged VLAN.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(2, 4009),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tunnel_config_type": schema.StringAttribute{
				MarkdownDescription: "The tunnel configuration type. Can be `vpn`, `802.1x`, or `custom`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("vpn", "802.1x", "custom"),
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

func (r *radiusUserResource) Configure(
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

func (r *radiusUserResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data radiusUserResourceModel

	// Read Terraform plan data into the model
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

	// Convert to unifi.Account
	account := r.modelToRadiusUser(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Derive the VLAN from network_id when not set explicitly (#67).
	vlan, vlanDiags := r.resolveVLAN(ctx, &data, site)
	resp.Diagnostics.Append(vlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	account.VLAN = vlan

	// Create the account
	createdAccount, err := r.client.CreateAccount(ctx, site, account)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Radius User",
			err.Error(),
		)
		return
	}

	// Convert back to model
	r.radiusUserToModel(ctx, createdAccount, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *radiusUserResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data radiusUserResourceModel

	// Read Terraform prior state data into the model
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

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the account from the API
	account, err := r.client.GetAccount(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Radius User",
			"Could not read radius user with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert to model
	r.radiusUserToModel(ctx, account, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *radiusUserResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state radiusUserResourceModel
	var plan radiusUserResourceModel

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

	updateTimeout, timeoutDiags := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Step 2: Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)
	state.Timeouts = plan.Timeouts

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Step 3: Convert the updated state to API format
	account := r.modelToRadiusUser(ctx, &state)
	account.ID = state.ID.ValueString()

	// Derive the VLAN from network_id when not set explicitly (#67).
	vlan, vlanDiags := r.resolveVLAN(ctx, &state, site)
	resp.Diagnostics.Append(vlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	account.VLAN = vlan

	// Step 4: Send to API
	updatedAccount, err := r.client.UpdateAccount(ctx, site, account)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Radius User",
			err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	r.radiusUserToModel(ctx, updatedAccount, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *radiusUserResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data radiusUserResourceModel

	// Read Terraform prior state data into the model
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

	// Delete the account
	err := r.client.DeleteAccount(ctx, site, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Radius User",
			err.Error(),
		)
		return
	}
}

func (r *radiusUserResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import format: "site:id" or just "id" for default site
	idParts := strings.Split(req.ID, ":")

	if len(idParts) == 2 {
		// site:id format
		site := idParts[0]
		id := idParts[1]

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	if len(idParts) == 1 {
		// Just id, use default site
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		return
	}

	resp.Diagnostics.AddError(
		"Invalid Import ID",
		"Import ID must be in format 'site:id' or 'id'",
	)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *radiusUserResource) applyPlanToState(
	_ context.Context,
	plan *radiusUserResourceModel,
	state *radiusUserResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		state.Password = plan.Password
	}
	if !plan.TunnelType.IsNull() && !plan.TunnelType.IsUnknown() {
		state.TunnelType = plan.TunnelType
	}
	if !plan.TunnelMediumType.IsNull() && !plan.TunnelMediumType.IsUnknown() {
		state.TunnelMediumType = plan.TunnelMediumType
	}
	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		state.NetworkID = plan.NetworkID
	}
	if !plan.VLAN.IsNull() && !plan.VLAN.IsUnknown() {
		state.VLAN = plan.VLAN
	}
	if !plan.TunnelConfigType.IsNull() && !plan.TunnelConfigType.IsUnknown() {
		state.TunnelConfigType = plan.TunnelConfigType
	}
}

// modelToRadiusUser converts the Terraform model to the API struct.
func (r *radiusUserResource) modelToRadiusUser(
	_ context.Context,
	model *radiusUserResourceModel,
) *unifi.Account {
	account := &unifi.Account{
		Name:     model.Name.ValueString(),
		Password: model.Password.ValueString(),
	}

	account.TunnelType = model.TunnelType.ValueInt64Pointer()

	account.TunnelMediumType = model.TunnelMediumType.ValueInt64Pointer()

	if !model.NetworkID.IsNull() {
		account.NetworkID = model.NetworkID.ValueString()
	}
	if !model.VLAN.IsNull() {
		account.VLAN = model.VLAN.ValueInt64Pointer()
	}
	if !model.TunnelConfigType.IsNull() {
		account.TunnelConfigType = model.TunnelConfigType.ValueString()
	}

	return account
}

// resolveVLAN determines the VLAN to assign to the account. An explicit `vlan`
// always wins. Otherwise, when `network_id` is set, the account inherits that
// network's VLAN — the controller leaves the account `vlan` blank when only
// networkconf_id is sent, so RADIUS/MAB VLAN assignment never applies (#67).
// Returns nil when neither yields a VLAN (client falls back to the untagged VLAN).
//
// Note: because `vlan` is computed and stable, changing `network_id` later does
// not re-derive the VLAN on its own; set `vlan` explicitly (or clear it) to
// force a refresh.
func (r *radiusUserResource) resolveVLAN(
	ctx context.Context,
	model *radiusUserResourceModel,
	site string,
) (*int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !model.VLAN.IsNull() && !model.VLAN.IsUnknown() {
		return model.VLAN.ValueInt64Pointer(), diags
	}

	networkID := model.NetworkID.ValueString()
	if model.NetworkID.IsNull() || networkID == "" {
		return nil, diags
	}

	network, err := r.client.GetNetwork(ctx, site, networkID)
	if err != nil {
		diags.AddError(
			"Error Deriving VLAN from network_id",
			"Could not look up network "+networkID+
				" to derive the account VLAN: "+err.Error(),
		)
		return nil, diags
	}

	return network.VLAN, diags
}

// radiusUserToModel converts the API struct to the Terraform model.
func (r *radiusUserResource) radiusUserToModel(
	_ context.Context,
	account *unifi.Account,
	model *radiusUserResourceModel,
	site string,
) {
	model.ID = types.StringValue(account.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(account.Name)
	model.Password = types.StringValue(account.Password)
	model.TunnelType = types.Int64PointerValue(account.TunnelType)
	model.TunnelMediumType = types.Int64PointerValue(account.TunnelMediumType)

	if account.NetworkID != "" {
		model.NetworkID = types.StringValue(account.NetworkID)
	} else {
		model.NetworkID = types.StringNull()
	}

	model.VLAN = types.Int64PointerValue(account.VLAN)

	if account.TunnelConfigType != "" {
		model.TunnelConfigType = types.StringValue(account.TunnelConfigType)
	} else {
		model.TunnelConfigType = types.StringNull()
	}
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *radiusUserResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List RADIUS user accounts in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list RADIUS users from.",
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
func (r *radiusUserResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config radiusUserListConfigModel

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
	var filters []radiusUserListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	accounts, err := r.client.ListAccount(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing RADIUS Users", "Could not list radius users: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, account := range accounts {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if account.Name != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if account.Name != "" {
				result.DisplayName = account.Name
			} else {
				result.DisplayName = account.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(account.ID),
				)...,
			)

			// Convert to model.
			var model radiusUserResourceModel
			r.radiusUserToModel(ctx, &account, &model, site)
			model.Timeouts = timeoutsNullValue()
			result.Diagnostics.Append(result.Resource.Set(ctx, model)...)

			if !push(result) {
				return
			}
		}
	}
}
