package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
var (
	_ resource.Resource                = &clientGroupFrameworkResource{}
	_ resource.ResourceWithImportState = &clientGroupFrameworkResource{}
)

func NewClientGroupFrameworkResource() resource.Resource {
	return &clientGroupFrameworkResource{}
}

// clientGroupFrameworkResource defines the resource implementation.
type clientGroupFrameworkResource struct {
	client *Client
}

// clientGroupFrameworkResourceModel describes the resource data model.
type clientGroupFrameworkResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	Name           types.String `tfsdk:"name"`
	QOSRateMaxDown types.Int64  `tfsdk:"qos_rate_max_down"`
	QOSRateMaxUp   types.Int64  `tfsdk:"qos_rate_max_up"`
}

func (r *clientGroupFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client_group"
}

func (r *clientGroupFrameworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a client group, which can be used to limit bandwidth for groups of clients.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the client group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the client group with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the client group.",
				Required:            true,
			},
			"qos_rate_max_down": schema.Int64Attribute{
				MarkdownDescription: "The QOS maximum download rate.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.OneOf(
						-1,
						2,
						3,
						4,
						5,
						6,
						7,
						8,
						9,
						10,
						15,
						20,
						25,
						30,
						40,
						50,
						75,
						100,
						150,
						200,
						250,
						300,
						400,
						500,
						750,
						1000,
						1500,
						2000,
						2500,
						3000,
						4000,
						5000,
						7500,
						10000,
						15000,
						20000,
						25000,
						30000,
						40000,
						50000,
						75000,
						100000,
					),
				},
			},
			"qos_rate_max_up": schema.Int64Attribute{
				MarkdownDescription: "The QOS maximum upload rate.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.OneOf(
						-1,
						2,
						3,
						4,
						5,
						6,
						7,
						8,
						9,
						10,
						15,
						20,
						25,
						30,
						40,
						50,
						75,
						100,
						150,
						200,
						250,
						300,
						400,
						500,
						750,
						1000,
						1500,
						2000,
						2500,
						3000,
						4000,
						5000,
						7500,
						10000,
						15000,
						20000,
						25000,
						30000,
						40000,
						50000,
						75000,
						100000,
					),
				},
			},
		},
	}
}

func (r *clientGroupFrameworkResource) Configure(
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

func (r *clientGroupFrameworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan clientGroupFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the plan to UniFi ClientGroup struct
	clientGroup, diags := r.planToClientGroup(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the ClientGroup
	createdClientGroup, err := r.client.CreateClientGroup(ctx, site, clientGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Client Group",
			"Could not create client group: "+err.Error(),
		)
		return
	}

	// Convert response back to model
	diags = r.clientGroupToModel(ctx, createdClientGroup, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clientGroupFrameworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state clientGroupFrameworkResourceModel

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

	// Get the ClientGroup from the API
	clientGroup, err := r.client.GetClientGroup(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client Group",
			"Could not read client group with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.clientGroupToModel(ctx, clientGroup, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *clientGroupFrameworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state clientGroupFrameworkResourceModel
	var plan clientGroupFrameworkResourceModel

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
	clientGroup, diags := r.planToClientGroup(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set required fields for update
	clientGroup.ID = state.ID.ValueString()
	clientGroup.SiteID = site

	// Step 4: Send to API
	updatedClientGroup, err := r.client.UpdateClientGroup(ctx, site, clientGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Client Group",
			"Could not update client group with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	diags = r.clientGroupToModel(ctx, updatedClientGroup, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *clientGroupFrameworkResource) applyPlanToState(
	_ context.Context,
	plan *clientGroupFrameworkResourceModel,
	state *clientGroupFrameworkResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.QOSRateMaxDown.IsNull() && !plan.QOSRateMaxDown.IsUnknown() {
		state.QOSRateMaxDown = plan.QOSRateMaxDown
	}
	if !plan.QOSRateMaxUp.IsNull() && !plan.QOSRateMaxUp.IsUnknown() {
		state.QOSRateMaxUp = plan.QOSRateMaxUp
	}
}

func (r *clientGroupFrameworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state clientGroupFrameworkResourceModel

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

	err := r.client.DeleteClientGroup(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Client Group",
			"Could not delete client group with ID "+id+": "+err.Error(),
		)
		return
	}
}

func (r *clientGroupFrameworkResource) ImportState(
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

func (r *clientGroupFrameworkResource) planToClientGroup(
	_ context.Context,
	plan clientGroupFrameworkResourceModel,
) (*unifi.ClientGroup, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.ID.IsNull() && plan.Name.IsNull() {
		diags.AddError(
			"Invalid Client Group",
			"Client Group must have either an ID or Name to be imported",
		)
		return nil, diags
	}

	clientGroup := &unifi.ClientGroup{
		ID:             plan.ID.ValueString(),
		Name:           plan.Name.ValueString(),
		QOSRateMaxDown: plan.QOSRateMaxDown.ValueInt64(),
		QOSRateMaxUp:   plan.QOSRateMaxUp.ValueInt64(),
	}

	return clientGroup, diags
}

func (r *clientGroupFrameworkResource) clientGroupToModel(
	_ context.Context,
	clientGroup *unifi.ClientGroup,
	model *clientGroupFrameworkResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if model.ID.IsNull() && model.Name.IsNull() {
		diags.AddError(
			"Invalid Client Group",
			"Client Group must have either an ID or Name to be imported",
		)
		return diags
	}

	model.ID = types.StringValue(clientGroup.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(clientGroup.Name)
	model.QOSRateMaxDown = types.Int64Value(clientGroup.QOSRateMaxDown)
	model.QOSRateMaxUp = types.Int64Value(clientGroup.QOSRateMaxUp)

	return diags
}
