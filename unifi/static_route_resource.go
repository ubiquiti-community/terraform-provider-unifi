package unifi

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                     = &staticRouteFrameworkResource{}
	_ resource.ResourceWithImportState      = &staticRouteFrameworkResource{}
	_ resource.ResourceWithConfigValidators = &staticRouteFrameworkResource{}
	_ resource.ResourceWithIdentity         = &staticRouteFrameworkResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &staticRouteFrameworkResource{}
	_ list.ListResourceWithConfigure = &staticRouteFrameworkResource{}
)

func NewStaticRouteFrameworkResource() resource.Resource {
	return &staticRouteFrameworkResource{}
}

func NewStaticRouteListResource() list.ListResource {
	return &staticRouteFrameworkResource{}
}

// staticRouteFrameworkResource defines the resource implementation.
type staticRouteFrameworkResource struct {
	client *Client
}

// staticRouteListConfigModel describes the list configuration model.
type staticRouteListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// staticRouteListFilterModel represents a single name/value filter entry.
type staticRouteListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// staticRouteFrameworkResourceModel describes the resource data model.
type staticRouteFrameworkResourceModel struct {
	ID            types.String      `tfsdk:"id"`
	Site          types.String      `tfsdk:"site"`
	Name          types.String      `tfsdk:"name"`
	Network       types.String      `tfsdk:"network"`
	Type          types.String      `tfsdk:"type"`
	Distance      types.Int64       `tfsdk:"distance"`
	NextHop       iptypes.IPAddress `tfsdk:"next_hop"`
	Interface     types.String      `tfsdk:"interface"`
	Enabled       types.Bool        `tfsdk:"enabled"`
	GatewayDevice types.String      `tfsdk:"gateway_device"`
	GatewayType   types.String      `tfsdk:"gateway_type"`
	Timeouts      timeouts.Value    `tfsdk:"timeouts"`
}

func (r *staticRouteFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_static_route"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *staticRouteFrameworkResource) IdentitySchema(
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
				MarkdownDescription: "The next hop of the static route (only valid for `nexthop-route` type). Accepts IPv4 or IPv6 addresses.",
				CustomType:          iptypes.IPAddressType{},
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.Any(validators.IPv4Validator(), validators.IPv6Validator()),
				},
			},
			"interface": schema.StringAttribute{
				MarkdownDescription: "The interface of the static route (only valid for `interface-route` type). This can be `WAN1`, `WAN2`, or a network ID.",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the static route is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"gateway_device": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the gateway device, used when `gateway_type` is `switch`.",
				Optional:            true,
				Validators: []validator.String{
					validators.MACAddressValidator(),
				},
			},
			"gateway_type": schema.StringAttribute{
				MarkdownDescription: "The type of gateway for the static route. Can be `default` or `switch`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				Validators: []validator.String{
					stringvalidator.OneOf("default", "switch"),
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
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

	createTimeout, timeoutDiags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

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
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
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
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
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
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
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

func (r *staticRouteFrameworkResource) ConfigValidators(
	_ context.Context,
) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&staticRouteIPVersionValidator{},
	}
}

// staticRouteIPVersionValidator ensures network and next_hop use the same IP version.
type staticRouteIPVersionValidator struct{}

func (v *staticRouteIPVersionValidator) Description(_ context.Context) string {
	return "network and next_hop must use the same IP version (both IPv4 or both IPv6)"
}

func (v *staticRouteIPVersionValidator) MarkdownDescription(_ context.Context) string {
	return "network and next_hop must use the same IP version (both IPv4 or both IPv6)"
}

func (v *staticRouteIPVersionValidator) ValidateResource(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	// next_hop uses the iptypes.IPAddress custom type, so it must be read into a
	// matching value — reading it into types.String fails config conversion.
	var network types.String
	var nextHop iptypes.IPAddress
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("network"), &network)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("next_hop"), &nextHop)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only validate when both are known and next_hop is set.
	if network.IsNull() || network.IsUnknown() || nextHop.IsNull() || nextHop.IsUnknown() {
		return
	}

	// Convert next_hop via the custom type's built-in netip.Addr conversion
	// rather than re-parsing the raw string.
	hopAddr, diags := nextHop.ValueIPAddress()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// network is a CIDR string (already shape-validated by CIDRValidator); parse
	// it to a netip.Prefix so both sides are compared as netip values.
	prefix, err := netip.ParsePrefix(network.ValueString())
	if err != nil {
		return // malformed CIDR is already reported by the network attribute validator
	}

	if !ipVersionsMatch(prefix, hopAddr) {
		resp.Diagnostics.AddAttributeError(
			path.Root("next_hop"),
			"IP Version Mismatch",
			fmt.Sprintf(
				"network %q and next_hop %q must use the same IP version (both IPv4 or both IPv6)",
				network.ValueString(),
				hopAddr.String(),
			),
		)
	}
}

// ipVersionsMatch reports whether a CIDR prefix and an address use the same IP
// family. Unmap collapses IPv4-mapped IPv6 addresses (::ffff:a.b.c.d) to IPv4 so
// they compare as the v4 family.
func ipVersionsMatch(prefix netip.Prefix, hop netip.Addr) bool {
	return prefix.Addr().Unmap().Is4() == hop.Unmap().Is4()
}

// validateIPVersionMatch returns an error if network (CIDR) and nextHop (IP) use different IP versions.
func validateIPVersionMatch(network, nextHop string) error {
	// Invalid network/next_hop are already reported by their field validators;
	// an invalid (zero) value here just means "nothing to compare".
	prefix, _ := netip.ParsePrefix(network)
	hop, _ := netip.ParseAddr(nextHop)
	if !prefix.IsValid() || !hop.IsValid() {
		return nil
	}

	if !ipVersionsMatch(prefix, hop) {
		return fmt.Errorf(
			"network %q and next_hop %q must use the same IP version",
			network,
			nextHop,
		)
	}
	return nil
}

// Ensure staticRouteIPVersionValidator satisfies the resource.ConfigValidator interface.
var _ resource.ConfigValidator = &staticRouteIPVersionValidator{}

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
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.GatewayDevice.IsNull() && !plan.GatewayDevice.IsUnknown() {
		state.GatewayDevice = plan.GatewayDevice
	}
	if !plan.GatewayType.IsNull() && !plan.GatewayType.IsUnknown() {
		state.GatewayType = plan.GatewayType
	}
}

// modelToRouting converts the Terraform model to the API struct.
func (r *staticRouteFrameworkResource) modelToRouting(
	_ context.Context,
	model *staticRouteFrameworkResourceModel,
) *unifi.Routing {
	routeType := model.Type.ValueString()

	routing := &unifi.Routing{
		Enabled:             model.Enabled.ValueBool(),
		Type:                "static-route",
		Name:                model.Name.ValueString(),
		StaticRouteNetwork:  model.Network.ValueString(), // TODO: Apply cidrZeroBased if needed
		StaticRouteDistance: model.Distance.ValueInt64Pointer(),
		StaticRouteType:     routeType,
		GatewayType:         model.GatewayType.ValueString(),
	}

	if !model.GatewayDevice.IsNull() {
		routing.GatewayDevice = model.GatewayDevice.ValueString()
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
	model.Distance = types.Int64PointerValue(routing.StaticRouteDistance)

	model.NextHop = iptypes.NewIPAddressNull()
	if routing.StaticRouteNexthop != "" {
		model.NextHop = iptypes.NewIPAddressValue(routing.StaticRouteNexthop)
	}

	if routing.StaticRouteInterface != "" {
		model.Interface = types.StringValue(routing.StaticRouteInterface)
	} else {
		model.Interface = types.StringNull()
	}

	model.Enabled = types.BoolValue(routing.Enabled)

	if routing.GatewayDevice != "" {
		model.GatewayDevice = types.StringValue(routing.GatewayDevice)
	} else {
		model.GatewayDevice = types.StringNull()
	}

	if routing.GatewayType != "" {
		model.GatewayType = types.StringValue(routing.GatewayType)
	} else {
		model.GatewayType = types.StringValue("default")
	}
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *staticRouteFrameworkResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List static routes in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list static routes from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`, `type`. The `type` filter matches the static route type (`interface-route`, `nexthop-route`, `blackhole`).",
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
func (r *staticRouteFrameworkResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config staticRouteListConfigModel

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
	var filters []staticRouteListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	routings, err := r.client.ListRouting(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Static Routes", "Could not list static routes: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, routing := range routings {
			// Only surface static routes.
			if routing.Type != "static-route" {
				continue
			}

			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if routing.Name != val {
					continue
				}
			}

			// Apply type filter (static route type).
			if val, ok := postFilters["type"]; ok {
				if routing.StaticRouteType != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if routing.Name != "" {
				result.DisplayName = routing.Name
			} else {
				result.DisplayName = routing.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(routing.ID),
				)...,
			)

			// Convert to model.
			var model staticRouteFrameworkResourceModel
			routingCopy := routing
			r.routingToModel(ctx, &routingCopy, &model, site)
			result.Diagnostics.Append(result.Resource.Set(ctx, model)...)

			if !push(result) {
				return
			}
		}
	}
}
