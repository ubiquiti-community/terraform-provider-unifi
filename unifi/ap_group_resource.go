package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &apGroupResource{}
	_ resource.ResourceWithImportState = &apGroupResource{}
	_ resource.ResourceWithIdentity    = &apGroupResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &apGroupResource{}
	_ list.ListResourceWithConfigure = &apGroupResource{}
)

func NewAPGroupResource() resource.Resource {
	return &apGroupResource{}
}

func NewAPGroupListResource() list.ListResource {
	return &apGroupResource{}
}

// apGroupResource defines the resource implementation.
type apGroupResource struct {
	client *Client
}

// apGroupResourceModel describes the resource data model.
//
// device_macs uses the hwtypes.MACAddress element type (the same custom MAC type
// used by unifi_client) so that MAC addresses compare with semantic equality:
// the framework treats "AA-BB-CC-DD-EE-FF" and "aa:bb:cc:dd:ee:ff" as equal,
// which prevents perpetual plan diffs regardless of how the user writes them.
type apGroupResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Site       types.String   `tfsdk:"site"`
	Name       types.String   `tfsdk:"name"`
	DeviceMacs types.Set      `tfsdk:"device_macs"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

// apGroupListConfigModel describes the list configuration model.
type apGroupListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// apGroupListFilterModel represents a single name/value filter entry.
type apGroupListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *apGroupResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_ap_group"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *apGroupResource) IdentitySchema(
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

func (r *apGroupResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "`unifi_ap_group` manages a group of access points, which can be referenced from wireless networks (`unifi_wlan`) to control where an SSID is broadcast. The controller's built-in default group (\"All APs\") is read-only; updating or deleting it through this resource fails with a controller error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the AP group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "The name of the site to associate the AP group with.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the AP group.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"device_macs": schema.SetAttribute{
				Description: "The MAC addresses of the access points that are members of the group. May be empty — the controller accepts a group with no members.",
				Required:    true,
				// hwtypes.MACAddressType gives each element semantic equality so
				// case and separator differences (e.g. AA-BB-.. vs aa:bb:..) do
				// not produce spurious diffs.
				ElementType: hwtypes.MACAddressType{},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

func (r *apGroupResource) Configure(
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

func (r *apGroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan apGroupResourceModel
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

	// Convert model to API request
	apGroup, err := r.modelToAPIAPGroup(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting AP Group",
			fmt.Sprintf("Could not convert AP group to API format: %s", err),
		)
		return
	}

	apiAPGroup, err := r.client.CreateAPGroup(ctx, site, apGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating AP Group",
			fmt.Sprintf("Could not create AP group: %s", err),
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(apiAPGroup.ID)
	plan.Site = types.StringValue(site)
	resp.Diagnostics.Append(r.apGroupToModel(ctx, apiAPGroup, &plan, site)...)

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *apGroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state apGroupResourceModel
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

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	apGroup, err := r.client.GetAPGroup(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading AP Group",
			fmt.Sprintf("Could not read AP group %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	resp.Diagnostics.Append(r.apGroupToModel(ctx, apGroup, &state, site)...)

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *apGroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan apGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state apGroupResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
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

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()

	// Read current AP group and merge with planned changes
	currentAPGroup, err := r.client.GetAPGroup(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading AP Group for Update",
			fmt.Sprintf("Could not read AP group %s for update: %s", id, err),
		)
		return
	}

	// Apply current API values to state
	resp.Diagnostics.Append(r.apGroupToModel(ctx, currentAPGroup, &state, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply plan changes to the state (merge pattern)
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.DeviceMacs.IsNull() && !plan.DeviceMacs.IsUnknown() {
		state.DeviceMacs = plan.DeviceMacs
	}

	// Convert updated state to API request
	apGroup, err := r.modelToAPIAPGroup(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting AP Group for Update",
			fmt.Sprintf("Could not convert AP group to API format: %s", err),
		)
		return
	}

	// PUT keys on the ID.
	apGroup.ID = id

	apiAPGroup, err := r.client.UpdateAPGroup(ctx, site, apGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating AP Group",
			fmt.Sprintf("Could not update AP group %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	resp.Diagnostics.Append(r.apGroupToModel(ctx, apiAPGroup, &state, site)...)

	state.Timeouts = plan.Timeouts

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *apGroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state apGroupResourceModel
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

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeleteAPGroup(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting AP Group",
			fmt.Sprintf("Could not delete AP group %s: %s", id, err),
		)
		return
	}
}

func (r *apGroupResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts, diags := util.ParseImportID(req.ID, 1, 2)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if site := idParts["site"]; site != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	}

	if id := idParts["id"]; id != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	}
}

// Helper methods

func (r *apGroupResource) modelToAPIAPGroup(
	ctx context.Context,
	model *apGroupResourceModel,
) (*unifi.APGroup, error) {
	var deviceMacs []string
	if !model.DeviceMacs.IsNull() && !model.DeviceMacs.IsUnknown() {
		diags := model.DeviceMacs.ElementsAs(ctx, &deviceMacs, false)
		if diags.HasError() {
			return nil, fmt.Errorf("could not convert device_macs to string slice")
		}
	}

	// The controller stores MACs lowercased and colon-separated. Normalize on the
	// write path so the created/updated object matches the canonical form the
	// controller returns (semantic equality on device_macs guards the read path).
	for i, mac := range deviceMacs {
		deviceMacs[i] = cleanMAC(mac)
	}

	return &unifi.APGroup{
		Name:       model.Name.ValueString(),
		DeviceMacs: deviceMacs,
	}, nil
}

// apGroupToModel populates the resource model from the API struct, setting every
// schema field. It is the reusable API->model converter shared by Read, Update,
// and List.
func (r *apGroupResource) apGroupToModel(
	ctx context.Context,
	api *unifi.APGroup,
	model *apGroupResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if api.ID != "" {
		model.ID = types.StringValue(api.ID)
	}

	model.Site = types.StringValue(site)

	if api.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(api.Name)
	}

	// Map an empty membership to an empty set (not null) so a config with
	// device_macs = [] round-trips cleanly instead of showing an empty-vs-null diff.
	macs := api.DeviceMacs
	if macs == nil {
		macs = []string{}
	}
	macsSet, d := types.SetValueFrom(ctx, hwtypes.MACAddressType{}, macs)
	diags.Append(d...)
	model.DeviceMacs = macsSet

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *apGroupResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List AP groups in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list AP groups from.",
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
func (r *apGroupResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config apGroupListConfigModel

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
	var filters []apGroupListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	groups, err := r.client.ListAPGroup(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(
			"Error Listing AP Groups",
			"Could not list AP groups: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, group := range groups {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if group.Name != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if group.Name != "" {
				result.DisplayName = group.Name
			} else {
				result.DisplayName = group.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(group.ID),
				)...,
			)

			// Convert to model.
			var model apGroupResourceModel
			result.Diagnostics.Append(
				r.apGroupToModel(ctx, &group, &model, site)...,
			)
			if !result.Diagnostics.HasError() {
				model.Timeouts = timeoutsNullValue()
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
