package provider

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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &userGroupFrameworkResource{}
var _ resource.ResourceWithImportState = &userGroupFrameworkResource{}

func NewUserGroupFrameworkResource() resource.Resource {
	return &userGroupFrameworkResource{}
}

// userGroupFrameworkResource defines the resource implementation.
type userGroupFrameworkResource struct {
	client *client
}

// userGroupFrameworkResourceModel describes the resource data model.
type userGroupFrameworkResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Site            types.String `tfsdk:"site"`
	Name            types.String `tfsdk:"name"`
	QOSRateMaxDown  types.Int64  `tfsdk:"qos_rate_max_down"`
	QOSRateMaxUp    types.Int64  `tfsdk:"qos_rate_max_up"`
}

func (r *userGroupFrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (r *userGroupFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a user group (called "client group" in the UI), which can be used to limit bandwidth for groups of users.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the user group with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user group.",
				Required:            true,
			},
			"qos_rate_max_down": schema.Int64Attribute{
				MarkdownDescription: "The QOS maximum download rate.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.OneOf(-1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 25, 30, 40, 50, 75, 100, 150, 200, 250, 300, 400, 500, 750, 1000, 1500, 2000, 2500, 3000, 4000, 5000, 7500, 10000, 15000, 20000, 25000, 30000, 40000, 50000, 75000, 100000),
				},
			},
			"qos_rate_max_up": schema.Int64Attribute{
				MarkdownDescription: "The QOS maximum upload rate.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.OneOf(-1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 25, 30, 40, 50, 75, 100, 150, 200, 250, 300, 400, 500, 750, 1000, 1500, 2000, 2500, 3000, 4000, 5000, 7500, 10000, 15000, 20000, 25000, 30000, 40000, 50000, 75000, 100000),
				},
			},
		},
	}
}

func (r *userGroupFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userGroupFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userGroupFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.site
	}

	// Convert the plan to UniFi UserGroup struct
	userGroup, diags := r.planToUserGroup(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the UserGroup
	createdUserGroup, err := r.client.c.CreateUserGroup(ctx, site, userGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating User Group",
			"Could not create user group: "+err.Error(),
		)
		return
	}

	// Convert response back to model
	diags = r.userGroupToModel(ctx, createdUserGroup, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *userGroupFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userGroupFrameworkResourceModel

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

	// Get the UserGroup from the API
	userGroup, err := r.client.c.GetUserGroup(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User Group",
			"Could not read user group with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.userGroupToModel(ctx, userGroup, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *userGroupFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userGroupFrameworkResourceModel

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
	existingUserGroup, err := r.client.c.GetUserGroup(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User Group for Update",
			"Could not read user group with ID "+id+": "+err.Error(),
		)
		return
	}

	// 2. Convert plan to UserGroup struct
	planUserGroup, diags := r.planToUserGroup(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set required fields for update
	planUserGroup.ID = id
	planUserGroup.SiteID = site

	// 3. Merge planned changes with existing values (UniFi requires full objects)
	mergedUserGroup := r.mergeUserGroup(existingUserGroup, planUserGroup)

	// 4. Update the UserGroup
	updatedUserGroup, err := r.client.c.UpdateUserGroup(ctx, site, mergedUserGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating User Group",
			"Could not update user group with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert response back to model
	diags = r.userGroupToModel(ctx, updatedUserGroup, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *userGroupFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userGroupFrameworkResourceModel

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

	err := r.client.c.DeleteUserGroup(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting User Group",
			"Could not delete user group with ID "+id+": "+err.Error(),
		)
		return
	}
}

func (r *userGroupFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
	} else {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	}
}

// Helper functions for conversion and merging

func (r *userGroupFrameworkResource) planToUserGroup(ctx context.Context, plan userGroupFrameworkResourceModel) (*unifi.UserGroup, diag.Diagnostics) {
	var diags diag.Diagnostics

	userGroup := &unifi.UserGroup{
		ID:             plan.ID.ValueString(),
		Name:           plan.Name.ValueString(),
		QOSRateMaxDown: int(plan.QOSRateMaxDown.ValueInt64()),
		QOSRateMaxUp:   int(plan.QOSRateMaxUp.ValueInt64()),
	}

	return userGroup, diags
}

func (r *userGroupFrameworkResource) userGroupToModel(ctx context.Context, userGroup *unifi.UserGroup, model *userGroupFrameworkResourceModel, site string) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(userGroup.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(userGroup.Name)
	model.QOSRateMaxDown = types.Int64Value(int64(userGroup.QOSRateMaxDown))
	model.QOSRateMaxUp = types.Int64Value(int64(userGroup.QOSRateMaxUp))

	return diags
}

func (r *userGroupFrameworkResource) mergeUserGroup(existing *unifi.UserGroup, planned *unifi.UserGroup) *unifi.UserGroup {
	// Start with the existing user group to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values
	merged.Name = planned.Name
	merged.QOSRateMaxDown = planned.QOSRateMaxDown
	merged.QOSRateMaxUp = planned.QOSRateMaxUp

	// Preserve required fields for update
	merged.ID = planned.ID
	merged.SiteID = planned.SiteID

	return &merged
}