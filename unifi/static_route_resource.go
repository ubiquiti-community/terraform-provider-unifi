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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &staticRouteFrameworkResource{}
	_ resource.ResourceWithImportState = &staticRouteFrameworkResource{}
)

func NewStaticRouteFrameworkResource() resource.Resource {
	return &staticRouteFrameworkResource{}
}

// staticRouteFrameworkResource defines the resource implementation.
type staticRouteFrameworkResource struct {
	client *Client
}

// staticRouteFrameworkResourceModel describes the resource data model.
type staticRouteFrameworkResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Site      types.String `tfsdk:"site"`
	Name      types.String `tfsdk:"name"`
	Network   types.String `tfsdk:"network"`
	Type      types.String `tfsdk:"type"`
	Distance  types.Int64  `tfsdk:"distance"`
	NextHop   types.String `tfsdk:"next_hop"`
	Interface types.String `tfsdk:"interface"`
}

func (r *staticRouteFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_static_route"
}

func (r *staticRouteFrameworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a static route for the USG.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the static route.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the static route with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the static route.",
				Required:            true,
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "The network subnet address.",
				Required:            true,
				Validators: []validator.String{
					validators.CIDRValidator(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of static route. Can be `interface-route`, `nexthop-route`, or `blackhole`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("interface-route", "nexthop-route", "blackhole"),
				},
			},
			"distance": schema.Int64Attribute{
				MarkdownDescription: "The distance of the static route.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 255),
				},
			},
			"next_hop": schema.StringAttribute{
				MarkdownDescription: "The next hop of the static route (only valid for `nexthop-route` type).",
				Optional:            true,
				Validators: []validator.String{
					validators.IPv4Validator(),
				},
			},
			"interface": schema.StringAttribute{
				MarkdownDescription: "The interface of the static route (only valid for `interface-route` type). This can be `WAN1`, `WAN2`, or a network ID.",
				Optional:            true,
			},
		},
	}
}

func (r *staticRouteFrameworkResource) Configure(
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

func (r *staticRouteFrameworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data staticRouteFrameworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.Routing
	routing := r.modelToRouting(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the static route
	createdRouting, err := r.client.CreateRouting(ctx, site, routing)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Static Route",
			err.Error(),
		)
		return
	}

	// Convert back to model
	r.routingToModel(ctx, createdRouting, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *staticRouteFrameworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data staticRouteFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the static route from the API
	routing, err := r.client.GetRouting(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Static Route",
			"Could not read static route with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert to model
	r.routingToModel(ctx, routing, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *staticRouteFrameworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state staticRouteFrameworkResourceModel
	var plan staticRouteFrameworkResourceModel

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
	routing := r.modelToRouting(ctx, &state)
	routing.ID = state.ID.ValueString()

	// Step 4: Send to API
	updatedRouting, err := r.client.UpdateRouting(ctx, site, routing)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Static Route",
			err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	r.routingToModel(ctx, updatedRouting, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *staticRouteFrameworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data staticRouteFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the static route
	err := r.client.DeleteRouting(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Static Route",
			err.Error(),
		)
		return
	}
}

func (r *staticRouteFrameworkResource) ImportState(
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
func (r *staticRouteFrameworkResource) applyPlanToState(
	_ context.Context,
	plan *staticRouteFrameworkResourceModel,
	state *staticRouteFrameworkResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Network.IsNull() && !plan.Network.IsUnknown() {
		state.Network = plan.Network
	}
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		state.Type = plan.Type
	}
	if !plan.Distance.IsNull() && !plan.Distance.IsUnknown() {
		state.Distance = plan.Distance
	}
	if !plan.NextHop.IsNull() && !plan.NextHop.IsUnknown() {
		state.NextHop = plan.NextHop
	}
	if !plan.Interface.IsNull() && !plan.Interface.IsUnknown() {
		state.Interface = plan.Interface
	}
}

// modelToRouting converts the Terraform model to the API struct.
func (r *staticRouteFrameworkResource) modelToRouting(
	_ context.Context,
	model *staticRouteFrameworkResourceModel,
) *unifi.Routing {
	routeType := model.Type.ValueString()

	routing := &unifi.Routing{
		Enabled:             true,
		Type:                "static-route",
		Name:                model.Name.ValueString(),
		StaticRouteNetwork:  model.Network.ValueString(), // TODO: Apply cidrZeroBased if needed
		StaticRouteDistance: model.Distance.ValueInt64(),
		StaticRouteType:     routeType,
	}

	switch routeType {
	case "interface-route":
		if !model.Interface.IsNull() {
			routing.StaticRouteInterface = model.Interface.ValueString()
		}
	case "nexthop-route":
		if !model.NextHop.IsNull() {
			routing.StaticRouteNexthop = model.NextHop.ValueString()
		}
	case "blackhole":
		// No additional fields needed
	}

	return routing
}

// routingToModel converts the API struct to the Terraform model.
func (r *staticRouteFrameworkResource) routingToModel(
	_ context.Context,
	routing *unifi.Routing,
	model *staticRouteFrameworkResourceModel,
	site string,
) {
	model.ID = types.StringValue(routing.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(routing.Name)
	model.Network = types.StringValue(routing.StaticRouteNetwork)
	model.Type = types.StringValue(routing.StaticRouteType)
	model.Distance = types.Int64Value(int64(routing.StaticRouteDistance))

	if routing.StaticRouteNexthop != "" {
		model.NextHop = types.StringValue(routing.StaticRouteNexthop)
	} else {
		model.NextHop = types.StringNull()
	}

	if routing.StaticRouteInterface != "" {
		model.Interface = types.StringValue(routing.StaticRouteInterface)
	} else {
		model.Interface = types.StringNull()
	}
}
