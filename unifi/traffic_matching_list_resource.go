package unifi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

// Traffic-matching-list types and their corresponding per-item types in the
// Integration API.
const (
	trafficMatchingListTypeAddresses = "IPV4_ADDRESSES"
	trafficMatchingListTypePorts     = "PORTS"

	trafficMatchingListItemIPAddress = "IP_ADDRESS"
	trafficMatchingListItemPort      = "PORT_NUMBER"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &trafficMatchingListResource{}
	_ resource.ResourceWithImportState = &trafficMatchingListResource{}
	_ resource.ResourceWithIdentity    = &trafficMatchingListResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &trafficMatchingListResource{}
	_ list.ListResourceWithConfigure = &trafficMatchingListResource{}
)

func NewTrafficMatchingListResource() resource.Resource {
	return &trafficMatchingListResource{}
}

func NewTrafficMatchingListListResource() list.ListResource {
	return &trafficMatchingListResource{}
}

// trafficMatchingListResource manages a reusable set of match criteria (IPv4
// addresses or ports) exposed by the UniFi Network Integration API.
type trafficMatchingListResource struct {
	client *Client
}

// trafficMatchingListResourceModel describes the resource data model.
type trafficMatchingListResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Site     types.String   `tfsdk:"site"`
	Name     types.String   `tfsdk:"name"`
	Type     types.String   `tfsdk:"type"`
	Items    types.List     `tfsdk:"items"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// trafficMatchingListIdentityModel describes the resource identity data model.
type trafficMatchingListIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

// trafficMatchingListListConfigModel describes the list configuration model.
type trafficMatchingListListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// trafficMatchingListListFilterModel represents a single name/value filter entry.
type trafficMatchingListListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *trafficMatchingListResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_traffic_matching_list"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *trafficMatchingListResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"site": identityschema.StringAttribute{
				OptionalForImport: true,
			},
		},
	}
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity]. See
// siteIdentityUpgraders.
func (r *trafficMatchingListResource) UpgradeIdentity(
	_ context.Context,
) map[int64]resource.IdentityUpgrader {
	return siteIdentityUpgraders(func() *Client { return r.client })
}

func (r *trafficMatchingListResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_traffic_matching_list` manages a reusable set of match criteria " +
			"(IPv4 addresses or ports) exposed by the UniFi Network Integration API and referenced " +
			"by firewall and traffic-management rules.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the traffic matching list.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The site the traffic matching list belongs to: a site name " +
					"(for example `default`) or a site UUID. The UniFi Integration API addresses sites " +
					"by UUID, so a name is resolved to its UUID automatically. Defaults to the " +
					"provider's configured site.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the traffic matching list.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the list. One of `IPV4_ADDRESSES` or `PORTS`. " +
					"This determines how `items` are interpreted.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						trafficMatchingListTypeAddresses,
						trafficMatchingListTypePorts,
					),
				},
			},
			"items": schema.ListAttribute{
				MarkdownDescription: "The list entries. For `type = IPV4_ADDRESSES` these are IPv4 addresses " +
					"(e.g. `192.0.2.4`); for `type = PORTS` these are port numbers given as strings " +
					"(e.g. `\"80\"`, `\"443\"`).",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

func (r *trafficMatchingListResource) Configure(
	_ context.Context,
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

func (r *trafficMatchingListResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan trafficMatchingListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	site := r.siteOrDefault(plan.Site)
	siteID, err := r.client.IntegrationSiteID(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(siteResolveError(site, err))
		return
	}

	apiList, diags := r.modelToAPI(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateTrafficMatchingList(ctx, siteID, apiList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Traffic Matching List",
			fmt.Sprintf("Could not create traffic matching list: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(r.apiToModel(ctx, created, &plan, site)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, trafficMatchingListIdentityModel{
		ID:   plan.ID,
		Site: plan.Site,
	})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficMatchingListResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state trafficMatchingListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	// Read identity, falling back to state for resources created before
	// identity support (also lets Read run from an identity-only state).
	var identity trafficMatchingListIdentityModel
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		identity.ID = state.ID
		identity.Site = state.Site
	}

	id := state.ID.ValueString()
	if id == "" {
		id = identity.ID.ValueString()
	}
	site := state.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}

	siteID, err := r.client.IntegrationSiteID(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(siteResolveError(site, err))
		return
	}

	apiList, err := r.client.GetTrafficMatchingList(ctx, siteID, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Traffic Matching List",
			fmt.Sprintf("Could not read traffic matching list %s: %s", id, err),
		)
		return
	}

	resp.Diagnostics.Append(r.apiToModel(ctx, apiList, &state, site)...)

	if identity.ID.IsNull() || identity.ID.ValueString() == "" {
		identity.ID = state.ID
	}
	if identity.Site.IsNull() || identity.Site.ValueString() == "" {
		identity.Site = state.Site
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *trafficMatchingListResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan trafficMatchingListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state trafficMatchingListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	site := r.siteOrDefault(plan.Site)
	siteID, err := r.client.IntegrationSiteID(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(siteResolveError(site, err))
		return
	}
	id := state.ID.ValueString()

	apiList, diags := r.modelToAPI(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiList.ID = id

	updated, err := r.client.UpdateTrafficMatchingList(ctx, siteID, apiList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Traffic Matching List",
			fmt.Sprintf("Could not update traffic matching list %s: %s", id, err),
		)
		return
	}

	resp.Diagnostics.Append(r.apiToModel(ctx, updated, &plan, site)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, trafficMatchingListIdentityModel{
		ID:   plan.ID,
		Site: plan.Site,
	})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficMatchingListResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state trafficMatchingListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	site := r.siteOrDefault(state.Site)
	siteID, err := r.client.IntegrationSiteID(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return // the site is gone, so the list is too
		}
		resp.Diagnostics.AddError(siteResolveError(site, err))
		return
	}

	err = r.client.DeleteTrafficMatchingList(ctx, siteID, state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Traffic Matching List",
			fmt.Sprintf(
				"Could not delete traffic matching list %s: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}
}

func (r *trafficMatchingListResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Identity-based import (import block with identity, Terraform 1.12+).
	if req.ID == "" {
		var identity trafficMatchingListIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
		if !identity.Site.IsNull() && identity.Site.ValueString() != "" {
			resp.Diagnostics.Append(
				resp.State.SetAttribute(ctx, path.Root("site"), identity.Site)...,
			)
		}
		return
	}

	// Import by ID string ("id" or "site:id").
	idParts, diags := util.ParseImportID(req.ID, 1, 2)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := idParts["site"]
	id := idParts["id"]

	if site != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	}
	if id != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	}

	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), id)...)
		if site != "" {
			resp.Diagnostics.Append(
				resp.Identity.SetAttribute(ctx, path.Root("site"), site)...,
			)
		}
	}
}

func (r *trafficMatchingListResource) siteOrDefault(site types.String) string {
	if s := site.ValueString(); s != "" {
		return s
	}
	return r.client.Site
}

// siteResolveError builds the diagnostic summary and detail for a failed site-to-UUID
// resolution. Only a site that does not exist, or is ambiguous, points at the `site`
// setting; any other failure (network, authentication, server error) is reported as
// is, since it says nothing about the site.
func siteResolveError(site string, err error) (summary, detail string) {
	var notFound *unifi.NotFoundError
	switch {
	case errors.As(err, &notFound):
		return "Site Not Found", fmt.Sprintf(
			"No site matching %q was found in the UniFi Integration API. "+
				"Set `site` to the name or UUID of an existing site.",
			site,
		)
	case errors.Is(err, unifi.ErrAmbiguousIntegrationSite):
		return "Ambiguous Site", fmt.Sprintf(
			"%s. Set `site` to the site's UUID to choose one.",
			err,
		)
	default:
		return "Error Looking Up Site", fmt.Sprintf(
			"Could not list the sites from the UniFi Integration API to look up %q: %s",
			site,
			err,
		)
	}
}

// modelToAPI converts the Terraform model to the go-unifi TrafficMatchingList.
// Each item string becomes a typed item derived from the list type: addresses
// are sent as string IP_ADDRESS values, ports as integer PORT_NUMBER values.
func (r *trafficMatchingListResource) modelToAPI(
	ctx context.Context,
	model *trafficMatchingListResourceModel,
) (*unifi.TrafficMatchingList, diag.Diagnostics) {
	var diags diag.Diagnostics

	listType := model.Type.ValueString()

	var rawItems []string
	if !model.Items.IsNull() && !model.Items.IsUnknown() {
		diags.Append(model.Items.ElementsAs(ctx, &rawItems, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	items := make([]unifi.TrafficMatchingListItem, 0, len(rawItems))
	for _, raw := range rawItems {
		switch listType {
		case trafficMatchingListTypePorts:
			port, err := strconv.Atoi(raw)
			if err != nil {
				diags.AddAttributeError(
					path.Root("items"),
					"Invalid Port Number",
					fmt.Sprintf(
						"items must be integer port numbers when type = %s, got %q",
						trafficMatchingListTypePorts,
						raw,
					),
				)
				return nil, diags
			}
			items = append(items, unifi.TrafficMatchingListItem{
				Type:  trafficMatchingListItemPort,
				Value: port,
			})
		default:
			items = append(items, unifi.TrafficMatchingListItem{
				Type:  trafficMatchingListItemIPAddress,
				Value: raw,
			})
		}
	}

	return &unifi.TrafficMatchingList{
		Name:  model.Name.ValueString(),
		Type:  listType,
		Items: items,
	}, diags
}

// apiToModel populates the resource model from the API struct. Item values are
// stringified (JSON numbers arrive as float64 for ports), preserving order.
func (r *trafficMatchingListResource) apiToModel(
	ctx context.Context,
	api *unifi.TrafficMatchingList,
	model *trafficMatchingListResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(api.ID)
	model.Site = types.StringValue(site)
	if api.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(api.Name)
	}
	if api.Type == "" {
		model.Type = types.StringNull()
	} else {
		model.Type = types.StringValue(api.Type)
	}

	items := make([]string, 0, len(api.Items))
	for _, item := range api.Items {
		items = append(items, trafficMatchingListItemValueToString(item.Value))
	}
	itemsList, d := types.ListValueFrom(ctx, types.StringType, items)
	diags.Append(d...)
	model.Items = itemsList

	return diags
}

// trafficMatchingListItemValueToString renders a polymorphic item value as a
// string. Port numbers decode through encoding/json as float64, so they are
// formatted without a trailing ".0".
func trafficMatchingListItemValueToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatInt(int64(val), 10)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case nil:
		return ""
	default:
		return fmt.Sprint(val)
	}
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *trafficMatchingListResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List traffic matching lists in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The site name or UUID to list traffic matching lists from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`, `type`.",
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
func (r *trafficMatchingListResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config trafficMatchingListListConfigModel
	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	siteID, err := r.client.IntegrationSiteID(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(siteResolveError(site, err))
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	var filters []trafficMatchingListListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}
	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	lists, err := r.client.ListTrafficMatchingLists(ctx, siteID)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(
			"Error Listing Traffic Matching Lists",
			"Could not list traffic matching lists: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, apiList := range lists {
			if val, ok := postFilters["name"]; ok && apiList.Name != val {
				continue
			}
			if val, ok := postFilters["type"]; ok && apiList.Type != val {
				continue
			}

			result := req.NewListResult(ctx)
			if apiList.Name != "" {
				result.DisplayName = apiList.Name
			} else {
				result.DisplayName = apiList.ID
			}

			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(apiList.ID),
				)...,
			)
			result.Diagnostics.Append(
				result.Identity.SetAttribute(ctx, path.Root("site"), types.StringValue(site))...,
			)

			var model trafficMatchingListResourceModel
			result.Diagnostics.Append(r.apiToModel(ctx, &apiList, &model, site)...)
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
