package unifi

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

// destinationIPModel describes a nested destination.ip entry.
type destinationIPModel struct {
	Address types.String `tfsdk:"address"`
	Ports   types.List   `tfsdk:"ports"`
}

func (m destinationIPModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"address": types.StringType,
		"ports":   types.ListType{ElemType: types.StringType},
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

// destinationModel describes the nested destination attribute.
type destinationModel struct {
	Domain types.List `tfsdk:"domain"`
	IP     types.List `tfsdk:"ip"`
	Region types.List `tfsdk:"region"`
}

func (m destinationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"domain": types.ListType{ElemType: types.StringType},
		"ip": types.ListType{
			ElemType: types.ObjectType{AttrTypes: destinationIPModel{}.AttributeTypes()},
		},
		"region": types.ListType{ElemType: types.StringType},
	}
}

// trafficRouteResourceModel describes the resource data model.
type trafficRouteResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Site              types.String `tfsdk:"site"`
	Description       types.String `tfsdk:"description"`
	Destination       types.Object `tfsdk:"destination"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	KillSwitchEnabled types.Bool   `tfsdk:"kill_switch_enabled"`
	NetworkID         types.String `tfsdk:"network_id"`
	NextHop           types.String `tfsdk:"next_hop"`
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
			"destination": schema.SingleNestedAttribute{
				MarkdownDescription: "Destination filter for this traffic route. Specify exactly one of `domain`, `region`, or `ip`. When omitted, the route matches all internet traffic.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"domain": schema.ListAttribute{
						MarkdownDescription: "List of domain names to match.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("region"),
								path.MatchRelative().AtParent().AtName("ip"),
							),
						},
					},
					"ip": schema.ListNestedAttribute{
						MarkdownDescription: "List of IP address, subnet, or IP range entries to match. Use CIDR notation (e.g. `10.0.0.0/8`) for subnets, or a hyphenated range (e.g. `192.168.10.1-192.168.10.255`) for IP ranges.",
						Optional:            true,
						Validators: []validator.List{
							listvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("domain"),
								path.MatchRelative().AtParent().AtName("region"),
							),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"address": schema.StringAttribute{
									MarkdownDescription: "An IP address, CIDR subnet, or hyphenated IP range to match.",
									Required:            true,
								},
								"ports": schema.ListAttribute{
									MarkdownDescription: "List of ports or port ranges to match. Use a single number (e.g. `80`) for individual ports, or a hyphenated range (e.g. `8080-8090`) for port ranges. Only supported for IP addresses and subnets, not IP ranges.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
					"region": schema.ListAttribute{
						MarkdownDescription: "List of regions to match.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("domain"),
								path.MatchRelative().AtParent().AtName("ip"),
							),
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
			"kill_switch_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the kill switch is enabled. When enabled, traffic is blocked if the target network/VPN is unavailable.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
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
		MatchingTarget:    "INTERNET",
		NetworkID:         networkID,
		NextHop:           model.NextHop.ValueString(),
	}

	// Destination
	hasDest := !model.Destination.IsNull() && !model.Destination.IsUnknown()
	var dest destinationModel
	if hasDest {
		diags.Append(model.Destination.As(ctx, &dest, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
	}

	// Domains
	if hasDest && !dest.Domain.IsNull() && !dest.Domain.IsUnknown() {
		route.MatchingTarget = "DOMAIN"
		var domains []string
		diags.Append(dest.Domain.ElementsAs(ctx, &domains, false)...)
		if diags.HasError() {
			return nil, diags
		}

		route.Domains = make([]unifi.TrafficRouteDomains, len(domains))
		for i, d := range domains {
			route.Domains[i] = unifi.TrafficRouteDomains{
				Domain: d,
			}
		}
	}

	// Regions
	if hasDest && !dest.Region.IsNull() && !dest.Region.IsUnknown() {
		route.MatchingTarget = "REGION"
		var regions []string
		diags.Append(dest.Region.ElementsAs(ctx, &regions, false)...)
		route.Regions = regions
	}

	// IP (addresses and ranges combined)
	if hasDest && !dest.IP.IsNull() && !dest.IP.IsUnknown() {
		route.MatchingTarget = "IP"
		var ips []destinationIPModel
		diags.Append(dest.IP.ElementsAs(ctx, &ips, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for _, ip := range ips {
			address := ip.Address.ValueString()

			// Detect IP range (contains "-" but is not CIDR)
			if strings.Contains(address, "-") {
				parts := strings.SplitN(address, "-", 2)
				entry := unifi.TrafficRouteIPRanges{
					Start:   strings.TrimSpace(parts[0]),
					Stop:    strings.TrimSpace(parts[1]),
					Version: unifi.TrafficRouteIPVersionV4,
				}
				if ipAddr, err := netip.ParseAddr(entry.Start); err == nil && ipAddr.Is6() {
					entry.Version = unifi.TrafficRouteIPVersionV6
				}
				route.IPRanges = append(route.IPRanges, entry)
			} else {
				entry := unifi.TrafficRouteIPAddresses{
					Address: address,
					Version: unifi.TrafficRouteIPVersionV4,
				}
				if ipAddr, err := netip.ParseAddr(address); err == nil && ipAddr.Is6() {
					entry.Version = unifi.TrafficRouteIPVersionV6
				}

				// Parse ports
				if !ip.Ports.IsNull() && !ip.Ports.IsUnknown() {
					var portStrs []string
					diags.Append(ip.Ports.ElementsAs(ctx, &portStrs, false)...)
					for _, ps := range portStrs {
						if strings.Contains(ps, "-") {
							rangeParts := strings.SplitN(ps, "-", 2)
							start, err1 := strconv.ParseInt(
								strings.TrimSpace(rangeParts[0]),
								10,
								64,
							)
							stop, err2 := strconv.ParseInt(strings.TrimSpace(rangeParts[1]), 10, 64)
							if err1 != nil || err2 != nil {
								diags.AddError(
									"Invalid Port Range",
									fmt.Sprintf("could not parse port range %q", ps),
								)
								return nil, diags
							}
							entry.PortRanges = append(
								entry.PortRanges,
								unifi.TrafficRoutePortRanges{
									Start: &start,
									Stop:  &stop,
								},
							)
						} else {
							port, err := strconv.ParseInt(strings.TrimSpace(ps), 10, 64)
							if err != nil {
								diags.AddError(
									"Invalid Port",
									fmt.Sprintf("could not parse port %q", ps),
								)
								return nil, diags
							}
							entry.Ports = append(entry.Ports, port)
						}
					}
				}

				route.IPAddresses = append(route.IPAddresses, entry)
			}
		}
	}

	// Initialize empty slices for nil arrays.
	if route.Domains == nil {
		route.Domains = []unifi.TrafficRouteDomains{}
	}
	if route.Regions == nil {
		route.Regions = []string{}
	}
	if route.IPAddresses == nil {
		route.IPAddresses = []unifi.TrafficRouteIPAddresses{}
	}
	if route.IPRanges == nil {
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
	model.NetworkID = util.StringValueOrNull(route.NetworkID)
	model.NextHop = util.StringValueOrNull(route.NextHop)

	// Domains
	var domainsList types.List
	if len(route.Domains) > 0 {
		elements := make([]attr.Value, len(route.Domains))
		for i, dom := range route.Domains {
			elements[i] = types.StringValue(dom.Domain)
		}
		var d diag.Diagnostics
		domainsList, d = types.ListValue(types.StringType, elements)
		diags.Append(d...)
	} else {
		domainsList = types.ListNull(types.StringType)
	}

	// Regions
	var regionsList types.List
	if len(route.Regions) > 0 {
		elements := make([]attr.Value, len(route.Regions))
		for i, reg := range route.Regions {
			elements[i] = types.StringValue(reg)
		}
		var d diag.Diagnostics
		regionsList, d = types.ListValue(types.StringType, elements)
		diags.Append(d...)
	} else {
		regionsList = types.ListNull(types.StringType)
	}

	// IP (merge IPAddresses and IPRanges into unified list)
	var ipList types.List
	if len(route.IPAddresses) > 0 || len(route.IPRanges) > 0 {
		var ipElements []attr.Value

		for _, addr := range route.IPAddresses {
			// Build ports list from Ports + PortRanges
			var portStrings []attr.Value
			for _, p := range addr.Ports {
				portStrings = append(portStrings, types.StringValue(strconv.FormatInt(p, 10)))
			}
			for _, pr := range addr.PortRanges {
				if pr.Start != nil && pr.Stop != nil {
					portStrings = append(portStrings, types.StringValue(
						strconv.FormatInt(*pr.Start, 10)+"-"+strconv.FormatInt(*pr.Stop, 10),
					))
				}
			}

			var ports types.List
			if len(portStrings) > 0 {
				var d diag.Diagnostics
				ports, d = types.ListValue(types.StringType, portStrings)
				diags.Append(d...)
			} else {
				ports = types.ListNull(types.StringType)
			}

			ipModel := destinationIPModel{
				Address: types.StringValue(addr.Address),
				Ports:   ports,
			}

			obj, d := types.ObjectValueFrom(ctx, destinationIPModel{}.AttributeTypes(), ipModel)
			diags.Append(d...)
			ipElements = append(ipElements, obj)
		}

		for _, ipRange := range route.IPRanges {
			ipModel := destinationIPModel{
				Address: types.StringValue(ipRange.Start + "-" + ipRange.Stop),
				Ports:   types.ListNull(types.StringType),
			}

			obj, d := types.ObjectValueFrom(ctx, destinationIPModel{}.AttributeTypes(), ipModel)
			diags.Append(d...)
			ipElements = append(ipElements, obj)
		}

		var d diag.Diagnostics
		ipList, d = types.ListValue(
			types.ObjectType{AttrTypes: destinationIPModel{}.AttributeTypes()},
			ipElements,
		)
		diags.Append(d...)
	} else {
		ipList = types.ListNull(
			types.ObjectType{AttrTypes: destinationIPModel{}.AttributeTypes()},
		)
	}

	// Build destination object.
	if len(route.Domains) > 0 || len(route.Regions) > 0 || len(route.IPAddresses) > 0 ||
		len(route.IPRanges) > 0 {
		dest := destinationModel{
			Domain: domainsList,
			IP:     ipList,
			Region: regionsList,
		}
		var d diag.Diagnostics
		model.Destination, d = types.ObjectValueFrom(ctx, destinationModel{}.AttributeTypes(), dest)
		diags.Append(d...)
	} else {
		model.Destination = types.ObjectNull(destinationModel{}.AttributeTypes())
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
