package unifi

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &trafficRouteResource{}
	_ resource.ResourceWithImportState = &trafficRouteResource{}
	_ resource.ResourceWithIdentity    = &trafficRouteResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &trafficRouteResource{}
	_ list.ListResourceWithConfigure = &trafficRouteResource{}
)

func NewTrafficRouteResource() resource.Resource {
	return &trafficRouteResource{}
}

func NewTrafficRouteListResource() list.ListResource {
	return &trafficRouteResource{}
}

// trafficRouteResource defines the resource implementation.
type trafficRouteResource struct {
	client *Client
}

// trafficRoutePortRangeModel describes a nested port_ranges entry.
type trafficRoutePortRangeModel struct {
	Start types.Int64 `tfsdk:"start"`
	Stop  types.Int64 `tfsdk:"stop"`
}

func (m trafficRoutePortRangeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"start": types.Int64Type,
		"stop":  types.Int64Type,
	}
}

// trafficRouteDomainModel describes a nested domains entry.
type trafficRouteDomainModel struct {
	Domain     types.String `tfsdk:"domain"`
	PortRanges types.List   `tfsdk:"port_ranges"`
	Ports      types.List   `tfsdk:"ports"`
}

func (m trafficRouteDomainModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"domain": types.StringType,
		"port_ranges": types.ListType{
			ElemType: types.ObjectType{AttrTypes: trafficRoutePortRangeModel{}.AttributeTypes()},
		},
		"ports": types.ListType{ElemType: types.Int64Type},
	}
}

// trafficRouteIPAddressModel describes a nested ip_addresses entry.
type trafficRouteIPAddressModel struct {
	IPOrSubnet types.String `tfsdk:"ip_or_subnet"`
	PortRanges types.List   `tfsdk:"port_ranges"`
	Ports      types.List   `tfsdk:"ports"`
}

func (m trafficRouteIPAddressModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip_or_subnet": types.StringType,
		"port_ranges": types.ListType{
			ElemType: types.ObjectType{AttrTypes: trafficRoutePortRangeModel{}.AttributeTypes()},
		},
		"ports": types.ListType{ElemType: types.Int64Type},
	}
}

// trafficRouteIPRangeModel describes a nested ip_ranges entry.
type trafficRouteIPRangeModel struct {
	Start types.String `tfsdk:"start"`
	Stop  types.String `tfsdk:"stop"`
}

func (m trafficRouteIPRangeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"start": types.StringType,
		"stop":  types.StringType,
	}
}

// sourceNetworkModel describes a nested source.networks entry.
type sourceNetworkModel struct {
	ID types.String `tfsdk:"id"`
}

func (m sourceNetworkModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}

// sourceClientModel describes a nested source.clients entry.
type sourceClientModel struct {
	MAC types.String `tfsdk:"mac"`
}

func (m sourceClientModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mac": types.StringType,
	}
}

// sourceModel describes the nested source attribute.
type sourceModel struct {
	Networks types.List `tfsdk:"networks"`
	Clients  types.List `tfsdk:"clients"`
}

func (m sourceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"networks": types.ListType{
			ElemType: types.ObjectType{AttrTypes: sourceNetworkModel{}.AttributeTypes()},
		},
		"clients": types.ListType{
			ElemType: types.ObjectType{AttrTypes: sourceClientModel{}.AttributeTypes()},
		},
	}
}

// trafficRouteResourceModel describes the resource data model.
type trafficRouteResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Site              types.String `tfsdk:"site"`
	Description       types.String `tfsdk:"description"`
	Domains           types.List   `tfsdk:"domains"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	IPAddresses       types.List   `tfsdk:"ip_addresses"`
	IPRanges          types.List   `tfsdk:"ip_ranges"`
	KillSwitchEnabled types.Bool   `tfsdk:"kill_switch_enabled"`
	MatchingTarget    types.String `tfsdk:"matching_target"`
	NetworkID         types.String `tfsdk:"network_id"`
	NextHop           types.String `tfsdk:"next_hop"`
	Regions           types.List   `tfsdk:"regions"`
	Source            types.Object `tfsdk:"source"`
}

type trafficRouteIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// trafficRouteListConfigModel describes the list configuration model.
type trafficRouteListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// trafficRouteListFilterModel represents a single name/value filter entry.
type trafficRouteListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *trafficRouteResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_traffic_route"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *trafficRouteResource) IdentitySchema(
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

func (r *trafficRouteResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a traffic route in the UniFi controller. Traffic routes allow you to steer traffic matching specific destinations through a chosen network or VPN.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the traffic route.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the traffic route with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the traffic route (max 128 characters).",
				Optional:            true,
			},
			"domains": schema.ListNestedAttribute{
				MarkdownDescription: "List of domain entries to match for this traffic route.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							MarkdownDescription: "The domain name to match.",
							Required:            true,
						},
						"port_ranges": schema.ListNestedAttribute{
							MarkdownDescription: "List of port ranges to match.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"start": schema.Int64Attribute{
										MarkdownDescription: "The start port of the range.",
										Required:            true,
									},
									"stop": schema.Int64Attribute{
										MarkdownDescription: "The stop port of the range.",
										Required:            true,
									},
								},
							},
						},
						"ports": schema.ListAttribute{
							MarkdownDescription: "List of individual ports to match.",
							Optional:            true,
							ElementType:         types.Int64Type,
						},
					},
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the traffic route is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"ip_addresses": schema.ListNestedAttribute{
				MarkdownDescription: "List of IP address or subnet entries to match.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_or_subnet": schema.StringAttribute{
							MarkdownDescription: "An IP address or CIDR subnet to match.",
							Required:            true,
						},
						"port_ranges": schema.ListNestedAttribute{
							MarkdownDescription: "List of port ranges to match.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"start": schema.Int64Attribute{
										MarkdownDescription: "The start port of the range.",
										Required:            true,
									},
									"stop": schema.Int64Attribute{
										MarkdownDescription: "The stop port of the range.",
										Required:            true,
									},
								},
							},
						},
						"ports": schema.ListAttribute{
							MarkdownDescription: "List of individual ports to match.",
							Optional:            true,
							ElementType:         types.Int64Type,
						},
					},
				},
			},
			"ip_ranges": schema.ListNestedAttribute{
				MarkdownDescription: "List of IP range entries to match.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							MarkdownDescription: "The start IP address of the range.",
							Required:            true,
						},
						"stop": schema.StringAttribute{
							MarkdownDescription: "The stop IP address of the range.",
							Required:            true,
						},
					},
				},
			},
			"kill_switch_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the kill switch is enabled. When enabled, traffic is blocked if the target network/VPN is unavailable.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"matching_target": schema.StringAttribute{
				MarkdownDescription: "The matching target type for the traffic route (e.g. `INTERNET`, `DOMAIN`, `IP`, `REGION`).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("INTERNET", "DOMAIN", "IP", "REGION"),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network or VPN to route matching traffic through. Defaults to the primary WAN network.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"next_hop": schema.StringAttribute{
				MarkdownDescription: "The next hop for the traffic route.",
				Optional:            true,
			},
			"regions": schema.ListAttribute{
				MarkdownDescription: "List of regions to match for this traffic route.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "Source filter for this traffic route. When omitted, the route applies to all clients.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"networks": schema.ListNestedAttribute{
						MarkdownDescription: "List of networks whose traffic this route applies to.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "The ID of the network.",
									Required:            true,
								},
							},
						},
					},
					"clients": schema.ListNestedAttribute{
						MarkdownDescription: "List of client devices whose traffic this route applies to.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"mac": schema.StringAttribute{
									MarkdownDescription: "The MAC address of the client device.",
									Required:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *trafficRouteResource) Configure(
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

func (r *trafficRouteResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan trafficRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	body, diags := r.modelToAPI(ctx, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateTrafficRoute(ctx, site, body)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Traffic Route", err.Error())
		return
	}

	resp.Diagnostics.Append(r.apiToModel(ctx, created, &plan, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := trafficRouteIdentityModel{ID: plan.ID}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficRouteResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state trafficRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	if id == "" {
		// Try identity
		var idModel trafficRouteIdentityModel
		if d := req.Identity.Get(ctx, &idModel); !d.HasError() {
			id = idModel.ID.ValueString()
		}
	}

	if id == "" {
		resp.Diagnostics.AddError("Invalid State", "Traffic route must have an ID")
		return
	}

	route, err := r.client.GetTrafficRoute(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Traffic Route", err.Error())
		return
	}

	resp.Diagnostics.Append(r.apiToModel(ctx, route, &state, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := trafficRouteIdentityModel{ID: state.ID}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *trafficRouteResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state trafficRouteResourceModel
	var plan trafficRouteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	body, diags := r.modelToAPI(ctx, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body.ID = state.ID.ValueString()

	updated, err := r.client.UpdateTrafficRoute(ctx, site, body)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Traffic Route", err.Error())
		return
	}

	resp.Diagnostics.Append(r.apiToModel(ctx, updated, &plan, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := trafficRouteIdentityModel{ID: plan.ID}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficRouteResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state trafficRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()

	err := r.client.DeleteTrafficRoute(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Traffic Route", err.Error())
	}
}

func (r *trafficRouteResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		req.ID = idParts[1]
	}

	idModel := trafficRouteIdentityModel{ID: types.StringValue(req.ID)}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resource.ImportStatePassthroughWithIdentity(
		ctx,
		path.Root("id"),
		path.Root("id"),
		req,
		resp,
	)
}

// modelToAPI converts the Terraform model to the UniFi API struct.
func (r *trafficRouteResource) modelToAPI(
	ctx context.Context,
	model *trafficRouteResourceModel,
	site string,
) (*unifi.TrafficRoute, diag.Diagnostics) {
	var diags diag.Diagnostics

	networkID := model.NetworkID.ValueString()
	if networkID == "" {
		var err error
		networkID, err = r.defaultWANNetworkID(ctx, site)
		if err != nil {
			diags.AddError("Error Finding Default WAN Network", err.Error())
			return nil, diags
		}
	}

	route := &unifi.TrafficRoute{
		Description:       model.Description.ValueString(),
		Enabled:           model.Enabled.ValueBool(),
		KillSwitchEnabled: model.KillSwitchEnabled.ValueBool(),
		MatchingTarget:    model.MatchingTarget.ValueString(),
		NetworkID:         networkID,
		NextHop:           model.NextHop.ValueString(),
	}

	// Domains
	if !model.Domains.IsNull() && !model.Domains.IsUnknown() {
		var domains []trafficRouteDomainModel
		diags.Append(model.Domains.ElementsAs(ctx, &domains, false)...)
		if diags.HasError() {
			return nil, diags
		}

		route.Domains = make([]unifi.TrafficRouteDomains, len(domains))
		for i, d := range domains {
			entry := unifi.TrafficRouteDomains{
				Domain: d.Domain.ValueString(),
			}

			if !d.PortRanges.IsNull() && !d.PortRanges.IsUnknown() {
				var portRanges []trafficRoutePortRangeModel
				diags.Append(d.PortRanges.ElementsAs(ctx, &portRanges, false)...)
				entry.PortRanges = make([]unifi.TrafficRoutePortRanges, len(portRanges))
				for j, pr := range portRanges {
					entry.PortRanges[j] = unifi.TrafficRoutePortRanges{
						PortStart: pr.Start.ValueInt64Pointer(),
						PortStop:  pr.Stop.ValueInt64Pointer(),
					}
				}
			}

			if !d.Ports.IsNull() && !d.Ports.IsUnknown() {
				var ports []int64
				diags.Append(d.Ports.ElementsAs(ctx, &ports, false)...)
				entry.Ports = ports
			}

			route.Domains[i] = entry
		}
	} else {
		route.Domains = []unifi.TrafficRouteDomains{}
	}

	// Regions
	if !model.Regions.IsNull() && !model.Regions.IsUnknown() {
		var regions []string
		diags.Append(model.Regions.ElementsAs(ctx, &regions, false)...)
		route.Regions = regions
	} else {
		route.Regions = []string{}
	}

	// IP Addresses
	if !model.IPAddresses.IsNull() && !model.IPAddresses.IsUnknown() {
		var ipAddrs []trafficRouteIPAddressModel
		diags.Append(model.IPAddresses.ElementsAs(ctx, &ipAddrs, false)...)
		if diags.HasError() {
			return nil, diags
		}

		route.IPAddresses = make([]unifi.TrafficRouteIPAddresses, len(ipAddrs))
		for i, addr := range ipAddrs {
			entry := unifi.TrafficRouteIPAddresses{
				IPOrSubnet: addr.IPOrSubnet.ValueString(),
				IPVersion:  unifi.TrafficRouteIPVersionV4,
			}

			if ipAddr, err := netip.ParseAddr(
				addr.IPOrSubnet.ValueString(),
			); err == nil &&
				ipAddr.IsValid() {
				if ipAddr.Is6() {
					entry.IPVersion = unifi.TrafficRouteIPVersionV6
				}
			}

			if !addr.PortRanges.IsNull() && !addr.PortRanges.IsUnknown() {
				var portRanges []trafficRoutePortRangeModel
				diags.Append(addr.PortRanges.ElementsAs(ctx, &portRanges, false)...)
				entry.PortRanges = make([]unifi.TrafficRoutePortRanges, len(portRanges))
				for j, pr := range portRanges {
					entry.PortRanges[j] = unifi.TrafficRoutePortRanges{
						PortStart: pr.Start.ValueInt64Pointer(),
						PortStop:  pr.Stop.ValueInt64Pointer(),
					}
				}
			}

			if !addr.Ports.IsNull() && !addr.Ports.IsUnknown() {
				var ports []int64
				diags.Append(addr.Ports.ElementsAs(ctx, &ports, false)...)
				entry.Ports = ports
			}

			route.IPAddresses[i] = entry
		}
	} else {
		route.IPAddresses = []unifi.TrafficRouteIPAddresses{}
	}

	// IP Ranges
	if !model.IPRanges.IsNull() && !model.IPRanges.IsUnknown() {
		var ipRanges []trafficRouteIPRangeModel
		diags.Append(model.IPRanges.ElementsAs(ctx, &ipRanges, false)...)
		if diags.HasError() {
			return nil, diags
		}

		route.IPRanges = make([]unifi.TrafficRouteIPRanges, len(ipRanges))
		for i, r := range ipRanges {
			entry := unifi.TrafficRouteIPRanges{
				IPStart:   r.Start.ValueString(),
				IPStop:    r.Stop.ValueString(),
				IPVersion: unifi.TrafficRouteIPVersionV4,
			}

			if ipAddr, err := netip.ParseAddr(entry.IPStart); err == nil &&
				ipAddr.IsValid() {
				if ipAddr.Is6() {
					entry.IPVersion = unifi.TrafficRouteIPVersionV6
				}
			}

			route.IPRanges[i] = entry
		}
	} else {
		route.IPRanges = []unifi.TrafficRouteIPRanges{}
	}

	// Source → TargetDevices
	if !model.Source.IsNull() && !model.Source.IsUnknown() {
		var src sourceModel
		diags.Append(model.Source.As(ctx, &src, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		var devices []unifi.TrafficRouteTargetDevices

		// Networks
		if !src.Networks.IsNull() && !src.Networks.IsUnknown() {
			var networks []sourceNetworkModel
			diags.Append(src.Networks.ElementsAs(ctx, &networks, false)...)
			for _, n := range networks {
				devices = append(devices, unifi.TrafficRouteTargetDevices{
					NetworkID: n.ID.ValueString(),
					Type:      "NETWORK",
				})
			}
		}

		// Clients
		if !src.Clients.IsNull() && !src.Clients.IsUnknown() {
			var clients []sourceClientModel
			diags.Append(src.Clients.ElementsAs(ctx, &clients, false)...)
			for _, c := range clients {
				devices = append(devices, unifi.TrafficRouteTargetDevices{
					ClientMAC: c.MAC.ValueString(),
					Type:      "CLIENT",
				})
			}
		}

		if len(devices) > 0 {
			route.TargetDevices = devices
		} else {
			route.TargetDevices = []unifi.TrafficRouteTargetDevices{{Type: "ALL_CLIENTS"}}
		}
	} else {
		route.TargetDevices = []unifi.TrafficRouteTargetDevices{{Type: "ALL_CLIENTS"}}
	}

	return route, diags
}

// apiToModel converts the UniFi API struct to the Terraform model.
func (r *trafficRouteResource) apiToModel(
	ctx context.Context,
	route *unifi.TrafficRoute,
	model *trafficRouteResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(route.ID)
	model.Site = util.StringValueOrNull(site)
	model.Description = util.StringValueOrNull(route.Description)
	model.Enabled = types.BoolValue(route.Enabled)
	model.KillSwitchEnabled = types.BoolValue(route.KillSwitchEnabled)
	model.MatchingTarget = util.StringValueOrNull(route.MatchingTarget)
	model.NetworkID = util.StringValueOrNull(route.NetworkID)
	model.NextHop = util.StringValueOrNull(route.NextHop)

	// Domains
	if len(route.Domains) > 0 {
		elements := make([]attr.Value, len(route.Domains))
		for i, dom := range route.Domains {
			// Port ranges
			var portRanges types.List
			if len(dom.PortRanges) > 0 {
				prElems := make([]attr.Value, len(dom.PortRanges))
				for j, pr := range dom.PortRanges {
					prModel := trafficRoutePortRangeModel{
						Start: types.Int64PointerValue(pr.PortStart),
						Stop:  types.Int64PointerValue(pr.PortStop),
					}
					var d diag.Diagnostics
					prElems[j], d = types.ObjectValueFrom(
						ctx,
						trafficRoutePortRangeModel{}.AttributeTypes(),
						prModel,
					)
					diags.Append(d...)
				}
				var d diag.Diagnostics
				portRanges, d = types.ListValue(
					types.ObjectType{AttrTypes: trafficRoutePortRangeModel{}.AttributeTypes()},
					prElems,
				)
				diags.Append(d...)
			} else {
				portRanges = types.ListNull(
					types.ObjectType{AttrTypes: trafficRoutePortRangeModel{}.AttributeTypes()},
				)
			}

			// Ports
			var ports types.List
			if len(dom.Ports) > 0 {
				pElems := make([]attr.Value, len(dom.Ports))
				for j, p := range dom.Ports {
					pElems[j] = types.Int64Value(p)
				}
				var d diag.Diagnostics
				ports, d = types.ListValue(types.Int64Type, pElems)
				diags.Append(d...)
			} else {
				ports = types.ListNull(types.Int64Type)
			}

			domModel := trafficRouteDomainModel{
				Domain:     types.StringValue(dom.Domain),
				PortRanges: portRanges,
				Ports:      ports,
			}

			var d diag.Diagnostics
			elements[i], d = types.ObjectValueFrom(
				ctx,
				trafficRouteDomainModel{}.AttributeTypes(),
				domModel,
			)
			diags.Append(d...)
		}

		var d diag.Diagnostics
		model.Domains, d = types.ListValue(
			types.ObjectType{AttrTypes: trafficRouteDomainModel{}.AttributeTypes()},
			elements,
		)
		diags.Append(d...)
	} else {
		model.Domains = types.ListNull(
			types.ObjectType{AttrTypes: trafficRouteDomainModel{}.AttributeTypes()},
		)
	}

	// Regions
	if len(route.Regions) > 0 {
		elements := make([]attr.Value, len(route.Regions))
		for i, reg := range route.Regions {
			elements[i] = types.StringValue(reg)
		}
		var d diag.Diagnostics
		model.Regions, d = types.ListValue(types.StringType, elements)
		diags.Append(d...)
	} else {
		model.Regions = types.ListNull(types.StringType)
	}

	// IP Addresses
	if len(route.IPAddresses) > 0 {
		elements := make([]attr.Value, len(route.IPAddresses))
		for i, addr := range route.IPAddresses {
			// Port ranges
			var portRanges types.List
			if len(addr.PortRanges) > 0 {
				prElems := make([]attr.Value, len(addr.PortRanges))
				for j, pr := range addr.PortRanges {
					prModel := trafficRoutePortRangeModel{
						Start: types.Int64PointerValue(pr.PortStart),
						Stop:  types.Int64PointerValue(pr.PortStop),
					}
					var d diag.Diagnostics
					prElems[j], d = types.ObjectValueFrom(
						ctx,
						trafficRoutePortRangeModel{}.AttributeTypes(),
						prModel,
					)
					diags.Append(d...)
				}
				var d diag.Diagnostics
				portRanges, d = types.ListValue(
					types.ObjectType{AttrTypes: trafficRoutePortRangeModel{}.AttributeTypes()},
					prElems,
				)
				diags.Append(d...)
			} else {
				portRanges = types.ListNull(
					types.ObjectType{AttrTypes: trafficRoutePortRangeModel{}.AttributeTypes()},
				)
			}

			// Ports
			var ports types.List
			if len(addr.Ports) > 0 {
				pElems := make([]attr.Value, len(addr.Ports))
				for j, p := range addr.Ports {
					pElems[j] = types.Int64Value(p)
				}
				var d diag.Diagnostics
				ports, d = types.ListValue(types.Int64Type, pElems)
				diags.Append(d...)
			} else {
				ports = types.ListNull(types.Int64Type)
			}

			addrModel := trafficRouteIPAddressModel{
				IPOrSubnet: types.StringValue(addr.IPOrSubnet),
				PortRanges: portRanges,
				Ports:      ports,
			}

			var d diag.Diagnostics
			elements[i], d = types.ObjectValueFrom(
				ctx,
				trafficRouteIPAddressModel{}.AttributeTypes(),
				addrModel,
			)
			diags.Append(d...)
		}

		var d diag.Diagnostics
		model.IPAddresses, d = types.ListValue(
			types.ObjectType{AttrTypes: trafficRouteIPAddressModel{}.AttributeTypes()},
			elements,
		)
		diags.Append(d...)
	} else {
		model.IPAddresses = types.ListNull(
			types.ObjectType{AttrTypes: trafficRouteIPAddressModel{}.AttributeTypes()},
		)
	}

	// IP Ranges
	if len(route.IPRanges) > 0 {
		elements := make([]attr.Value, len(route.IPRanges))
		for i, ipRange := range route.IPRanges {
			rangeModel := trafficRouteIPRangeModel{
				Start: types.StringValue(ipRange.IPStart),
				Stop:  types.StringValue(ipRange.IPStop),
			}

			var d diag.Diagnostics
			elements[i], d = types.ObjectValueFrom(
				ctx,
				trafficRouteIPRangeModel{}.AttributeTypes(),
				rangeModel,
			)
			diags.Append(d...)
		}

		var d diag.Diagnostics
		model.IPRanges, d = types.ListValue(
			types.ObjectType{AttrTypes: trafficRouteIPRangeModel{}.AttributeTypes()},
			elements,
		)
		diags.Append(d...)
	} else {
		model.IPRanges = types.ListNull(
			types.ObjectType{AttrTypes: trafficRouteIPRangeModel{}.AttributeTypes()},
		)
	}

	// TargetDevices → Source
	var networkElements []attr.Value
	var clientElements []attr.Value

	for _, td := range route.TargetDevices {
		switch td.Type {
		case "NETWORK":
			obj, d := types.ObjectValueFrom(
				ctx,
				sourceNetworkModel{}.AttributeTypes(),
				sourceNetworkModel{
					ID: types.StringValue(td.NetworkID),
				},
			)
			diags.Append(d...)
			networkElements = append(networkElements, obj)
		case "CLIENT":
			obj, d := types.ObjectValueFrom(
				ctx,
				sourceClientModel{}.AttributeTypes(),
				sourceClientModel{
					MAC: types.StringValue(td.ClientMAC),
				},
			)
			diags.Append(d...)
			clientElements = append(clientElements, obj)
		case "ALL_CLIENTS":
			// ALL_CLIENTS is the default; represented by omitting source
		}
	}

	if len(networkElements) > 0 || len(clientElements) > 0 {
		var networksList types.List
		if len(networkElements) > 0 {
			var d diag.Diagnostics
			networksList, d = types.ListValue(
				types.ObjectType{AttrTypes: sourceNetworkModel{}.AttributeTypes()},
				networkElements,
			)
			diags.Append(d...)
		} else {
			networksList = types.ListNull(
				types.ObjectType{AttrTypes: sourceNetworkModel{}.AttributeTypes()},
			)
		}

		var clientsList types.List
		if len(clientElements) > 0 {
			var d diag.Diagnostics
			clientsList, d = types.ListValue(
				types.ObjectType{AttrTypes: sourceClientModel{}.AttributeTypes()},
				clientElements,
			)
			diags.Append(d...)
		} else {
			clientsList = types.ListNull(
				types.ObjectType{AttrTypes: sourceClientModel{}.AttributeTypes()},
			)
		}

		var d diag.Diagnostics
		model.Source, d = types.ObjectValueFrom(ctx, sourceModel{}.AttributeTypes(), sourceModel{
			Networks: networksList,
			Clients:  clientsList,
		})
		diags.Append(d...)
	} else {
		model.Source = types.ObjectNull(sourceModel{}.AttributeTypes())
	}

	return diags
}

func (r *trafficRouteResource) defaultWANNetworkID(
	ctx context.Context,
	site string,
) (string, error) {
	networks, err := r.client.ListNetwork(ctx, site)
	if err != nil {
		return "", fmt.Errorf("unable to list networks: %w", err)
	}
	for _, n := range networks {
		if n.Purpose == unifi.PurposeWAN && n.WANNetworkGroup != nil &&
			*n.WANNetworkGroup == "WAN" {
			return n.ID, nil
		}
	}
	return "", fmt.Errorf("no default WAN network found")
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *trafficRouteResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List traffic routes in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list traffic routes from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `enabled`, `matching_target`, `network_id`, `description`.",
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
func (r *trafficRouteResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config trafficRouteListConfigModel

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
	var filters []trafficRouteListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	routes, err := r.client.ListTrafficRoute(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Traffic Routes", "Could not list traffic routes: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, route := range routes {
			// Apply enabled filter.
			if val, ok := postFilters["enabled"]; ok {
				enabled := fmt.Sprintf("%t", route.Enabled)
				if enabled != val {
					continue
				}
			}

			// Apply matching_target filter.
			if val, ok := postFilters["matching_target"]; ok {
				if route.MatchingTarget != val {
					continue
				}
			}

			// Apply network_id filter.
			if val, ok := postFilters["network_id"]; ok {
				if route.NetworkID != val {
					continue
				}
			}

			// Apply description filter.
			if val, ok := postFilters["description"]; ok {
				if route.Description != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer description, fall back to ID.
			if route.Description != "" {
				result.DisplayName = route.Description
			} else {
				result.DisplayName = route.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(route.ID),
				)...,
			)

			// Convert to model.
			var model trafficRouteResourceModel
			result.Diagnostics.Append(r.apiToModel(ctx, &route, &model, site)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
