package unifi

import (
	"context"
	"fmt"
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
	_ resource.Resource                = &firewallZoneResource{}
	_ resource.ResourceWithImportState = &firewallZoneResource{}
	_ resource.ResourceWithIdentity    = &firewallZoneResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &firewallZoneResource{}
	_ list.ListResourceWithConfigure = &firewallZoneResource{}
)

func NewFirewallZoneResource() resource.Resource {
	return &firewallZoneResource{}
}

func NewFirewallZoneListResource() list.ListResource {
	return &firewallZoneResource{}
}

// firewallZoneResource defines the resource implementation.
type firewallZoneResource struct {
	client *Client
}

// firewallZoneListConfigModel describes the list configuration model.
type firewallZoneListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// firewallZoneListFilterModel represents a single name/value filter entry.
type firewallZoneListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// firewallZoneResourceModel describes the resource data model.
type firewallZoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Site        types.String `tfsdk:"site"`
	Name        types.String `tfsdk:"name"`
	NetworkIDs  types.List   `tfsdk:"network_ids"`
	ZoneKey     types.String `tfsdk:"zone_key"`
	DefaultZone types.Bool   `tfsdk:"default_zone"`
}

func (r *firewallZoneResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_zone"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *firewallZoneResource) IdentitySchema(
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

func (r *firewallZoneResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a zone-based firewall zone (UniFi OS 8.x+). Create a zone and " +
			"attach networks to it, then reference its `id` from `unifi_firewall_policy`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall zone.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the zone belongs to.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the firewall zone.",
				Required:            true,
			},
			"network_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the networks assigned to this zone.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"zone_key": schema.StringAttribute{
				MarkdownDescription: "The controller-assigned key of the zone.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_zone": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a controller default zone.",
				Computed:            true,
			},
		},
	}
}

func (r *firewallZoneResource) Configure(
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

func (r *firewallZoneResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data firewallZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, diags := r.modelToFirewallZone(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	created, err := r.client.CreateFirewallZone(ctx, site, zone)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Firewall Zone", err.Error())
		return
	}

	resp.Diagnostics.Append(r.firewallZoneToModel(ctx, created, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallZoneResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data firewallZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	zone, err := r.client.GetFirewallZone(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Firewall Zone",
			"Could not read firewall zone with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.firewallZoneToModel(ctx, zone, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallZoneResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data firewallZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, diags := r.modelToFirewallZone(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zone.ID = data.ID.ValueString()

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	updated, err := r.client.UpdateFirewallZone(ctx, site, zone)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Firewall Zone", err.Error())
		return
	}

	resp.Diagnostics.Append(r.firewallZoneToModel(ctx, updated, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallZoneResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data firewallZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeleteFirewallZone(ctx, site, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Firewall Zone", err.Error())
		return
	}
}

func (r *firewallZoneResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import format: "site:id" or just "id" for the default site.
	idParts := strings.Split(req.ID, ":")
	switch len(idParts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'site:id' or 'id'",
		)
	}
}

// modelToFirewallZone converts the Terraform model to the API struct.
func (r *firewallZoneResource) modelToFirewallZone(
	ctx context.Context,
	model *firewallZoneResourceModel,
) (*unifi.FirewallZone, diag.Diagnostics) {
	var diags diag.Diagnostics
	zone := &unifi.FirewallZone{
		Name:       model.Name.ValueString(),
		NetworkIDs: []string{},
	}
	if !model.NetworkIDs.IsNull() && !model.NetworkIDs.IsUnknown() {
		diags = model.NetworkIDs.ElementsAs(ctx, &zone.NetworkIDs, false)
	}
	return zone, diags
}

// firewallZoneToModel converts the API struct to the Terraform model.
func (r *firewallZoneResource) firewallZoneToModel(
	ctx context.Context,
	zone *unifi.FirewallZone,
	model *firewallZoneResourceModel,
	site string,
) diag.Diagnostics {
	model.ID = types.StringValue(zone.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(zone.Name)
	model.ZoneKey = types.StringValue(zone.ZoneKey)
	model.DefaultZone = types.BoolValue(zone.DefaultZone)

	networkIDs, diags := types.ListValueFrom(ctx, types.StringType, zone.NetworkIDs)
	model.NetworkIDs = networkIDs
	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *firewallZoneResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List firewall zones in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list firewall zones from.",
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
func (r *firewallZoneResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config firewallZoneListConfigModel

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
	var filters []firewallZoneListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	zones, err := r.client.ListFirewallZone(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Firewall Zones", "Could not list firewall zones: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, zone := range zones {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if zone.Name != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if zone.Name != "" {
				result.DisplayName = zone.Name
			} else {
				result.DisplayName = zone.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(zone.ID),
				)...,
			)

			// Convert to model.
			var model firewallZoneResourceModel
			zoneCopy := zone
			result.Diagnostics.Append(r.firewallZoneToModel(ctx, &zoneCopy, &model, site)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
