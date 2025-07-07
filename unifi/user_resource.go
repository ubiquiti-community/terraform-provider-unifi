package unifi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &userFrameworkResource{}
	_ resource.ResourceWithImportState = &userFrameworkResource{}
)

func NewUserFrameworkResource() resource.Resource {
	return &userFrameworkResource{}
}

// userFrameworkResource defines the resource implementation.
type userFrameworkResource struct {
	client *Client
}

// userFrameworkResourceModel describes the resource data model.
type userFrameworkResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Site                types.String `tfsdk:"site"`
	MAC                 types.String `tfsdk:"mac"`
	Name                types.String `tfsdk:"name"`
	UserGroupID         types.String `tfsdk:"user_group_id"`
	Note                types.String `tfsdk:"note"`
	FixedIP             types.String `tfsdk:"fixed_ip"`
	NetworkID           types.String `tfsdk:"network_id"`
	Blocked             types.Bool   `tfsdk:"blocked"`
	DevIDOverride       types.Int64  `tfsdk:"dev_id_override"`
	LocalDNSRecord      types.String `tfsdk:"local_dns_record"`
	AllowExisting       types.Bool   `tfsdk:"allow_existing"`
	SkipForgetOnDestroy types.Bool   `tfsdk:"skip_forget_on_destroy"`

	// Computed attributes
	Hostname types.String `tfsdk:"hostname"`
	IP       types.String `tfsdk:"ip"`
}

func (r *userFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userFrameworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a user (or "client" in the UI) of the network, identified by unique MAC addresses.

Users are created in the controller when observed on the network, so the resource defaults to allowing itself to just take over management of a MAC address, but this can be turned off.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the user with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Required:            true,
			},
			"user_group_id": schema.StringAttribute{
				MarkdownDescription: "The user group ID for the user.",
				Optional:            true,
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A note with additional information for the user.",
				Optional:            true,
			},
			"fixed_ip": schema.StringAttribute{
				MarkdownDescription: "A fixed IPv4 address for this user.",
				Optional:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID for this user.",
				Optional:            true,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this user should be blocked from the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"dev_id_override": schema.Int64Attribute{
				MarkdownDescription: "Override the device fingerprint.",
				Optional:            true,
			},
			"local_dns_record": schema.StringAttribute{
				MarkdownDescription: "Specifies the local DNS record for this user.",
				Optional:            true,
			},
			"allow_existing": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should just take over control of an existing user.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"skip_forget_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should tell the controller to \"forget\" the user on destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *userFrameworkResource) Configure(
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

func (r *userFrameworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan userFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the plan to UniFi User struct
	user, diags := r.planToUser(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowExisting := plan.AllowExisting.ValueBool()

	// Create the User
	createdUser, err := r.client.Client.CreateUser(ctx, site, user)
	if err != nil {
		var apiErr *unifi.APIError
		if !errors.As(err, &apiErr) || (apiErr.Message != "api.err.MacUsed" || !allowExisting) {
			resp.Diagnostics.AddError(
				"Error Creating User",
				"Could not create user: "+err.Error(),
			)
			return
		}

		// MAC in use, just absorb the existing user
		mac := plan.MAC.ValueString()
		existingUser, err := r.client.Client.GetUserByMAC(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Existing User",
				"Could not get existing user with MAC "+mac+": "+err.Error(),
			)
			return
		}

		// Implement merge pattern for existing user
		mergedUser := r.mergeUser(existingUser, user)
		updatedUser, err := r.client.Client.UpdateUser(ctx, site, mergedUser)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Existing User",
				"Could not update existing user: "+err.Error(),
			)
			return
		}
		createdUser = updatedUser
	}

	// Convert response back to model
	diags = r.userToModel(ctx, createdUser, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *userFrameworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state userFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()

	// Get the User from the API
	user, err := r.client.Client.GetUser(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User",
			"Could not read user with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.userToModel(ctx, user, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *userFrameworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state userFrameworkResourceModel
	var plan userFrameworkResourceModel

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
	user, diags := r.planToUser(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 4: Send to API
	user.ID = state.ID.ValueString()
	updatedUser, err := r.client.Client.UpdateUser(ctx, site, user)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating User",
			"Could not update user with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	diags = r.userToModel(ctx, updatedUser, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown
func (r *userFrameworkResource) applyPlanToState(
	ctx context.Context,
	plan *userFrameworkResourceModel,
	state *userFrameworkResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.MAC.IsNull() && !plan.MAC.IsUnknown() {
		state.MAC = plan.MAC
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.UserGroupID.IsNull() && !plan.UserGroupID.IsUnknown() {
		state.UserGroupID = plan.UserGroupID
	}
	if !plan.Note.IsNull() && !plan.Note.IsUnknown() {
		state.Note = plan.Note
	}
	if !plan.FixedIP.IsNull() && !plan.FixedIP.IsUnknown() {
		state.FixedIP = plan.FixedIP
	}
	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		state.NetworkID = plan.NetworkID
	}
	if !plan.Blocked.IsNull() && !plan.Blocked.IsUnknown() {
		state.Blocked = plan.Blocked
	}
	if !plan.DevIDOverride.IsNull() && !plan.DevIDOverride.IsUnknown() {
		state.DevIDOverride = plan.DevIDOverride
	}
	if !plan.LocalDNSRecord.IsNull() && !plan.LocalDNSRecord.IsUnknown() {
		state.LocalDNSRecord = plan.LocalDNSRecord
	}
	if !plan.AllowExisting.IsNull() && !plan.AllowExisting.IsUnknown() {
		state.AllowExisting = plan.AllowExisting
	}
	if !plan.SkipForgetOnDestroy.IsNull() && !plan.SkipForgetOnDestroy.IsUnknown() {
		state.SkipForgetOnDestroy = plan.SkipForgetOnDestroy
	}
	// Note: Computed attributes (Hostname, IP) are not applied from plan
}

func (r *userFrameworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state userFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	skipForget := state.SkipForgetOnDestroy.ValueBool()

	if skipForget {
		// Just remove from Terraform state without telling UniFi to forget
		return
	}

	// lookup MAC instead of trusting state
	u, err := r.client.Client.GetUser(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User for Delete",
			"Could not read user with ID "+id+": "+err.Error(),
		)
		return
	}

	err = r.client.Client.DeleteUserByMAC(ctx, site, u.MAC)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting User",
			"Could not delete user with MAC "+u.MAC+": "+err.Error(),
		)
		return
	}
}

func (r *userFrameworkResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
	} else {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	}
}

// Helper functions for conversion and merging

func (r *userFrameworkResource) planToUser(
	ctx context.Context,
	plan userFrameworkResourceModel,
) (*unifi.User, diag.Diagnostics) {
	var diags diag.Diagnostics

	user := &unifi.User{
		ID:             plan.ID.ValueString(),
		MAC:            plan.MAC.ValueString(),
		Name:           plan.Name.ValueString(),
		UserGroupID:    plan.UserGroupID.ValueString(),
		Note:           plan.Note.ValueString(),
		FixedIP:        plan.FixedIP.ValueString(),
		NetworkID:      plan.NetworkID.ValueString(),
		Blocked:        plan.Blocked.ValueBool(),
		LocalDNSRecord: plan.LocalDNSRecord.ValueString(),
	}

	if !plan.DevIDOverride.IsNull() && !plan.DevIDOverride.IsUnknown() {
		user.DevIdOverride = int(plan.DevIDOverride.ValueInt64())
	}

	return user, diags
}

func (r *userFrameworkResource) userToModel(
	ctx context.Context,
	user *unifi.User,
	model *userFrameworkResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(user.ID)
	model.Site = types.StringValue(site)
	model.MAC = types.StringValue(user.MAC)
	model.Name = types.StringValue(user.Name)

	if user.UserGroupID != "" {
		model.UserGroupID = types.StringValue(user.UserGroupID)
	} else {
		model.UserGroupID = types.StringNull()
	}

	if user.Note != "" {
		model.Note = types.StringValue(user.Note)
	} else {
		model.Note = types.StringNull()
	}

	if user.FixedIP != "" {
		model.FixedIP = types.StringValue(user.FixedIP)
	} else {
		model.FixedIP = types.StringNull()
	}

	if user.NetworkID != "" {
		model.NetworkID = types.StringValue(user.NetworkID)
	} else {
		model.NetworkID = types.StringNull()
	}

	model.Blocked = types.BoolValue(user.Blocked)

	if user.DevIdOverride != 0 {
		model.DevIDOverride = types.Int64Value(int64(user.DevIdOverride))
	} else {
		model.DevIDOverride = types.Int64Null()
	}

	if user.LocalDNSRecord != "" {
		model.LocalDNSRecord = types.StringValue(user.LocalDNSRecord)
	} else {
		model.LocalDNSRecord = types.StringNull()
	}

	// Computed attributes
	if user.Hostname != "" {
		model.Hostname = types.StringValue(user.Hostname)
	} else {
		model.Hostname = types.StringNull()
	}

	if user.IP != "" {
		model.IP = types.StringValue(user.IP)
	} else {
		model.IP = types.StringNull()
	}

	return diags
}

func (r *userFrameworkResource) mergeUser(existing *unifi.User, planned *unifi.User) *unifi.User {
	// Start with the existing user to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values
	merged.Name = planned.Name
	merged.UserGroupID = planned.UserGroupID
	merged.Note = planned.Note
	merged.FixedIP = planned.FixedIP
	merged.NetworkID = planned.NetworkID
	merged.Blocked = planned.Blocked
	merged.DevIdOverride = planned.DevIdOverride
	merged.LocalDNSRecord = planned.LocalDNSRecord

	return &merged
}
