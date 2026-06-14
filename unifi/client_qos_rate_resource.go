package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
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
	_ resource.Resource                = &clientQosRateResource{}
	_ resource.ResourceWithImportState = &clientQosRateResource{}
	_ resource.ResourceWithIdentity    = &clientQosRateResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &clientQosRateResource{}
	_ list.ListResourceWithConfigure = &clientQosRateResource{}
)

func NewClientQosRateResource() resource.Resource {
	return &clientQosRateResource{}
}

func NewClientQosRateListResource() list.ListResource {
	return &clientQosRateResource{}
}

// clientQosRateResource defines the resource implementation.
type clientQosRateResource struct {
	client *Client
}

// clientQosRateResourceModel describes the resource data model.
type clientQosRateResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	Site           types.String   `tfsdk:"site"`
	Name           types.String   `tfsdk:"name"`
	QOSRateMaxDown types.Int64    `tfsdk:"qos_rate_max_down"`
	QOSRateMaxUp   types.Int64    `tfsdk:"qos_rate_max_up"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

// clientQosRateListConfigModel describes the list configuration model.
type clientQosRateListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// clientQosRateListFilterModel represents a single name/value filter entry.
type clientQosRateListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *clientQosRateResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client_qos_rate"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *clientQosRateResource) IdentitySchema(
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

func (r *clientQosRateResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a client QOS rate, which can be used to limit bandwidth for groups of clients.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the client QOS rate.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the client QOS rate with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the client QOS rate.",
				Required:            true,
			},
			"qos_rate_max_down": schema.Int64Attribute{
				MarkdownDescription: "The QOS maximum download rate.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.Between(2, 100000),
				},
			},
			"qos_rate_max_up": schema.Int64Attribute{
				MarkdownDescription: "The QOS maximum upload rate.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.Between(2, 100000),
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

func (r *clientQosRateResource) Configure(
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

func (r *clientQosRateResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan clientQosRateResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the plan to UniFi ClientGroup struct
	clientGroup, diags := r.planToClientQosRate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the ClientGroup
	createdClientGroup, err := r.client.CreateClientGroup(ctx, site, clientGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Client QOS Rate",
			"Could not create client QOS rate: "+err.Error(),
		)
		return
	}

	// Convert response back to model
	diags = r.clientQosRateToModel(ctx, createdClientGroup, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clientQosRateResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state clientQosRateResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

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
			"Error Reading Client QOS Rate",
			"Could not read client QOS rate with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.clientQosRateToModel(ctx, clientGroup, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *clientQosRateResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state clientQosRateResourceModel
	var plan clientQosRateResourceModel

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

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Step 3: Convert the updated state to API format
	clientGroup, diags := r.planToClientQosRate(ctx, state)
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
			"Error Updating Client QOS Rate",
			"Could not update client QOS rate with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	diags = r.clientQosRateToModel(ctx, updatedClientGroup, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = plan.Timeouts

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *clientQosRateResource) applyPlanToState(
	_ context.Context,
	plan *clientQosRateResourceModel,
	state *clientQosRateResourceModel,
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

func (r *clientQosRateResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state clientQosRateResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, timeoutDiags := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

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
			"Error Deleting Client QOS Rate",
			"Could not delete client QOS rate with ID "+id+": "+err.Error(),
		)
		return
	}
}

func (r *clientQosRateResource) ImportState(
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

func (r *clientQosRateResource) planToClientQosRate(
	_ context.Context,
	plan clientQosRateResourceModel,
) (*unifi.ClientGroup, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.ID.IsNull() && plan.Name.IsNull() {
		diags.AddError(
			"Invalid Client QOS Rate",
			"Client QOS Rate must have either an ID or Name to be imported",
		)
		return nil, diags
	}

	clientGroup := &unifi.ClientGroup{
		ID:             plan.ID.ValueString(),
		Name:           plan.Name.ValueString(),
		QOSRateMaxDown: plan.QOSRateMaxDown.ValueInt64Pointer(),
		QOSRateMaxUp:   plan.QOSRateMaxUp.ValueInt64Pointer(),
	}

	return clientGroup, diags
}

func (r *clientQosRateResource) clientQosRateToModel(
	_ context.Context,
	clientGroup *unifi.ClientGroup,
	model *clientQosRateResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if model.ID.IsNull() && model.Name.IsNull() {
		diags.AddError(
			"Invalid Client QOS Rate",
			"Client QOS Rate must have either an ID or Name to be imported",
		)
		return diags
	}

	model.ID = types.StringValue(clientGroup.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(clientGroup.Name)
	model.QOSRateMaxDown = types.Int64PointerValue(clientGroup.QOSRateMaxDown)
	model.QOSRateMaxUp = types.Int64PointerValue(clientGroup.QOSRateMaxUp)

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *clientQosRateResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List client QOS rates in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list client QOS rates from.",
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
func (r *clientQosRateResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config clientQosRateListConfigModel

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
	var filters []clientQosRateListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	clientGroups, err := r.client.ListClientGroup(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(
			"Error Listing Client QOS Rates",
			"Could not list client QOS rates: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for i := range clientGroups {
			clientGroup := clientGroups[i]

			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if clientGroup.Name != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if clientGroup.Name != "" {
				result.DisplayName = clientGroup.Name
			} else {
				result.DisplayName = clientGroup.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(clientGroup.ID),
				)...,
			)

			// Convert to model. Pre-populate the identifier so the converter's
			// ID/Name guard is satisfied.
			var model clientQosRateResourceModel
			model.ID = types.StringValue(clientGroup.ID)
			result.Diagnostics.Append(r.clientQosRateToModel(ctx, &clientGroup, &model, site)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
