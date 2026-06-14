package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &siteFrameworkResource{}
	_ resource.ResourceWithImportState = &siteFrameworkResource{}
	_ resource.ResourceWithIdentity    = &siteFrameworkResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &siteFrameworkResource{}
	_ list.ListResourceWithConfigure = &siteFrameworkResource{}
)

func NewSiteFrameworkResource() resource.Resource {
	return &siteFrameworkResource{}
}

func NewSiteListResource() list.ListResource {
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

// siteListConfigModel describes the list configuration model. Sites are global
// (not site-scoped), so there is no `site` attribute.
type siteListConfigModel struct {
	Filter types.List `tfsdk:"filter"`
}

// siteListFilterModel represents a single name/value filter entry.
type siteListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *siteFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *siteFrameworkResource) IdentitySchema(
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

func (r *siteFrameworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
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

func (r *siteFrameworkResource) Configure(
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

func (r *siteFrameworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan siteFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	description := plan.Description.ValueString()

	// Create the Site
	sites, err := r.client.CreateSite(ctx, description)
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

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *siteFrameworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state siteFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	var site *unifi.Site

	if !state.ID.IsNull() && !state.ID.IsUnknown() {
		// Get the Site from the API
		site, err = r.client.GetSite(ctx, state.ID.ValueString())
		if err != nil {
			if _, ok := err.(*unifi.NotFoundError); ok {
				resp.State.RemoveResource(ctx)
				return
			} else {
				resp.Diagnostics.AddError(
					"Error Reading Site",
					"Could not read site with ID "+state.ID.ValueString()+": "+err.Error(),
				)
				return
			}
		}
	} else {
		site, err = r.client.GetSiteByName(ctx, state.Name.ValueString())
		if err != nil {
			if _, ok := err.(*unifi.NotFoundError); ok {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError(
				"Error Reading Site",
				"Could not read site with Name "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Convert API response to model
	diags = r.siteToModel(ctx, site, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *siteFrameworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
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
	updatedSites, err := r.client.UpdateSite(ctx, name, description)
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

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *siteFrameworkResource) applyPlanToState(
	_ context.Context,
	plan *siteFrameworkResourceModel,
	state *siteFrameworkResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	// Note: Name cannot be changed after creation, so we don't apply it from plan
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		state.Description = plan.Description
	}
}

func (r *siteFrameworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state siteFrameworkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	_, err := r.client.DeleteSite(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Site",
			"Could not delete site with ID "+id+": "+err.Error(),
		)
		return
	}
}

func (r *siteFrameworkResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	rootAttributeName := "name"
	if after, ok := strings.CutPrefix(req.ID, "name="); ok {
		req.ID = after
	} else if regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(req.ID) {
		rootAttributeName = "id"
	}

	resource.ImportStatePassthroughID(ctx, path.Root(rootAttributeName), req, resp)
}

// Helper functions for conversion and merging

func (r *siteFrameworkResource) siteToModel(
	_ context.Context,
	site *unifi.Site,
	model *siteFrameworkResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if site == nil {
		// Defensive: the read paths now return before reaching here on a
		// not-found, but never dereference a nil site (it previously panicked —
		// #261, e.g. importing with an identifier that is neither a 24-hex id
		// nor a known site name).
		diags.AddError(
			"Site Not Found",
			"No site matched the given identifier. Import a site by its 24-hex "+
				"id, or by name with 'name=<site-name>' (e.g. 'name=default').",
		)
		return diags
	}

	if site.ID == "" && site.Name == "" {
		// If both ID and Name are empty, we can't import this site
		diags.AddError(
			"Invalid Site",
			"Site must have either an ID or Name to be imported",
		)
		return diags
	}

	model.ID = types.StringValue(site.ID)
	model.Name = types.StringValue(site.Name)
	model.Description = types.StringValue(site.Description)

	return diags
}

// ListResourceConfigSchema implements [list.ListResource]. Sites are global, so
// the config has no `site` attribute.
func (r *siteFrameworkResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List sites in the UniFi controller.",
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`, `description`.",
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
func (r *siteFrameworkResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config siteListConfigModel

	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	// Process filter blocks.
	var filters []siteListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	// Sites are global; ListSites takes no site argument.
	sites, err := r.client.ListSites(ctx)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(
			"Error Listing Sites",
			"Could not list sites: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for i := range sites {
			site := sites[i]

			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if site.Name != val {
					continue
				}
			}

			// Apply description filter.
			if val, ok := postFilters["description"]; ok {
				if site.Description != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer description, fall back to name then ID.
			switch {
			case site.Description != "":
				result.DisplayName = site.Description
			case site.Name != "":
				result.DisplayName = site.Name
			default:
				result.DisplayName = site.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(site.ID),
				)...,
			)

			// Convert to model.
			var model siteFrameworkResourceModel
			result.Diagnostics.Append(r.siteToModel(ctx, &site, &model)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
