package unifi

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &siteFrameworkResource{}
var _ resource.ResourceWithImportState = &siteFrameworkResource{}

func NewSiteFrameworkResource() resource.Resource {
	return &siteFrameworkResource{}
}

// siteFrameworkResource defines the resource implementation.
type siteFrameworkResource struct {
	client *Client
}

// siteFrameworkResourceModel describes the resource data model.
type siteFrameworkResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *siteFrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Unifi sites",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the site.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the site.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the site.",
				Required:            true,
			},
		},
	}
}

func (r *siteFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	description := plan.Description.ValueString()

	// Create the Site
	sites, err := r.client.Client.CreateSite(ctx, description)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Site",
			"Could not create site: "+err.Error(),
		)
		return
	}

	if len(sites) == 0 {
		resp.Diagnostics.AddError(
			"Error Creating Site",
			"No site returned from CreateSite call",
		)
		return
	}

	createdSite := sites[0]

	// Convert response back to model
	diags = r.siteToModel(ctx, &createdSite, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *siteFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	// Get the Site from the API
	site, err := r.client.Client.GetSite(ctx, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Site",
			"Could not read site with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.siteToModel(ctx, site, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *siteFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state siteFrameworkResourceModel
	var plan siteFrameworkResourceModel

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

	// Step 3: Convert the updated state to API format
	// Note: Site name cannot be changed after creation, only description
	id := state.ID.ValueString()
	name := state.Name.ValueString()
	description := state.Description.ValueString()

	// Step 4: Send to API
	updatedSites, err := r.client.Client.UpdateSite(ctx, name, description)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Site",
			"Could not update site with ID "+id+": "+err.Error(),
		)
		return
	}

	if len(updatedSites) == 0 {
		resp.Diagnostics.AddError(
			"Error Updating Site",
			"No site returned from UpdateSite call",
		)
		return
	}

	updatedSite := updatedSites[0]

	// Step 5: Update state with API response
	diags := r.siteToModel(ctx, &updatedSite, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown
func (r *siteFrameworkResource) applyPlanToState(ctx context.Context, plan *siteFrameworkResourceModel, state *siteFrameworkResourceModel) {
	// Apply plan values to state, but only if plan value is not null/unknown
	// Note: Name cannot be changed after creation, so we don't apply it from plan
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		state.Description = plan.Description
	}
}

func (r *siteFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	_, err := r.client.Client.DeleteSite(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Site",
			"Could not delete site with ID "+id+": "+err.Error(),
		)
		return
	}
}

func (r *siteFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID

	// First try to import by ID
	_, err := r.client.Client.GetSite(ctx, id)
	if err != nil {
		var nf *unifi.NotFoundError
		if !errors.As(err, &nf) {
			resp.Diagnostics.AddError(
				"Error Importing Site",
				"Could not read site with ID "+id+": "+err.Error(),
			)
			return
		}
	} else {
		// ID is valid
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		return
	}

	// If not found by ID, try to lookup site by name
	sites, err := r.client.Client.ListSites(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Sites for Import",
			"Could not list sites: "+err.Error(),
		)
		return
	}

	for _, s := range sites {
		if s.Name == id {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), s.ID)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Site Not Found",
		fmt.Sprintf("Unable to find site %q on controller", id),
	)
}

// Helper functions for conversion and merging

func (r *siteFrameworkResource) siteToModel(ctx context.Context, site *unifi.Site, model *siteFrameworkResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(site.ID)
	model.Name = types.StringValue(site.Name)
	model.Description = types.StringValue(site.Description)

	return diags
}

func (r *siteFrameworkResource) mergeSite(existing *unifi.Site, planned *unifi.Site) *unifi.Site {
	// Start with the existing site to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values (only description can be changed)
	merged.Description = planned.Description

	return &merged
}