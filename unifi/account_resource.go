package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &accountFrameworkResource{}
var _ resource.ResourceWithImportState = &accountFrameworkResource{}

func NewAccountFrameworkResource() resource.Resource {
	return &accountFrameworkResource{}
}

// accountFrameworkResource defines the resource implementation.
type accountFrameworkResource struct {
	client *Client
}

// accountFrameworkResourceModel describes the resource data model.
type accountFrameworkResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Site             types.String `tfsdk:"site"`
	Name             types.String `tfsdk:"name"`
	Password         types.String `tfsdk:"password"`
	TunnelType       types.Int64  `tfsdk:"tunnel_type"`
	TunnelMediumType types.Int64  `tfsdk:"tunnel_medium_type"`
	NetworkID        types.String `tfsdk:"network_id"`
}

func (r *accountFrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *accountFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				MarkdownDescription: "See [RFC 2868](https://www.rfc-editor.org/rfc/rfc2868) section 3.1",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(13),
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
				MarkdownDescription: "ID of the network for this account",
				Optional:            true,
			},
		},
	}
}

func (r *accountFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accountFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data accountFrameworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.Account
	account := r.modelToAccount(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the account
	createdAccount, err := r.client.Client.CreateAccount(ctx, site, account)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Account",
			"Could not create account, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert back to model
	r.accountToModel(ctx, createdAccount, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *accountFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data accountFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the account from the API
	account, err := r.client.Client.GetAccount(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Account",
			"Could not read account with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert to model
	r.accountToModel(ctx, account, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *accountFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state accountFrameworkResourceModel
	var plan accountFrameworkResourceModel

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

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Step 3: Convert the updated state to API format
	account := r.modelToAccount(ctx, &state)
	account.ID = state.ID.ValueString()

	// Step 4: Send to API
	updatedAccount, err := r.client.Client.UpdateAccount(ctx, site, account)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Account",
			"Could not update account, unexpected error: "+err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	r.accountToModel(ctx, updatedAccount, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accountFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data accountFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the account
	err := r.client.Client.DeleteAccount(ctx, site, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Account",
			"Could not delete account, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *accountFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown
func (r *accountFrameworkResource) applyPlanToState(ctx context.Context, plan *accountFrameworkResourceModel, state *accountFrameworkResourceModel) {
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
}

// modelToAccount converts the Terraform model to the API struct
func (r *accountFrameworkResource) modelToAccount(ctx context.Context, model *accountFrameworkResourceModel) *unifi.Account {
	account := &unifi.Account{
		Name:      model.Name.ValueString(),
		XPassword: model.Password.ValueString(),
	}

	if !model.TunnelType.IsNull() {
		account.TunnelType = int(model.TunnelType.ValueInt64())
	}
	
	if !model.TunnelMediumType.IsNull() {
		account.TunnelMediumType = int(model.TunnelMediumType.ValueInt64())
	}
	
	if !model.NetworkID.IsNull() {
		account.NetworkID = model.NetworkID.ValueString()
	}

	return account
}

// accountToModel converts the API struct to the Terraform model
func (r *accountFrameworkResource) accountToModel(ctx context.Context, account *unifi.Account, model *accountFrameworkResourceModel, site string) {
	model.ID = types.StringValue(account.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(account.Name)
	model.Password = types.StringValue(account.XPassword)
	model.TunnelType = types.Int64Value(int64(account.TunnelType))
	model.TunnelMediumType = types.Int64Value(int64(account.TunnelMediumType))
	
	if account.NetworkID != "" {
		model.NetworkID = types.StringValue(account.NetworkID)
	} else {
		model.NetworkID = types.StringNull()
	}
}