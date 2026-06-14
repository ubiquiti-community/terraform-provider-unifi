package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &wireguardPeerResource{}
	_ resource.ResourceWithImportState = &wireguardPeerResource{}
	_ resource.ResourceWithIdentity    = &wireguardPeerResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &wireguardPeerResource{}
	_ list.ListResourceWithConfigure = &wireguardPeerResource{}
)

func NewWireguardPeerResource() resource.Resource {
	return &wireguardPeerResource{}
}

func NewWireguardPeerListResource() list.ListResource {
	return &wireguardPeerResource{}
}

// wireguardPeerResource defines the resource implementation.
type wireguardPeerResource struct {
	client *Client
}

// wireguardPeerResourceModel describes the resource data model.
type wireguardPeerResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Site        types.String   `tfsdk:"site"`
	NetworkID   types.String   `tfsdk:"network_id"`
	Name        types.String   `tfsdk:"name"`
	InterfaceIP types.String   `tfsdk:"interface_ip"`
	PublicKey   types.String   `tfsdk:"public_key"`
	AllowedIPs  types.List     `tfsdk:"allowed_ips"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

// wireguardPeerListConfigModel describes the list configuration model. Peers
// belong to a WireGuard server network, so `network_id` is required.
type wireguardPeerListConfigModel struct {
	Site      types.String `tfsdk:"site"`
	NetworkID types.String `tfsdk:"network_id"`
	Filter    types.List   `tfsdk:"filter"`
}

// wireguardPeerListFilterModel represents a single name/value filter entry.
type wireguardPeerListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *wireguardPeerResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_wireguard_peer"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *wireguardPeerResource) IdentitySchema(
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

func (r *wireguardPeerResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WireGuard peer (client) of a WireGuard VPN server. " +
			"The server is a `unifi_vpn_server` resource (or a network with `vpn_type = \"wireguard-server\"`); " +
			"its network ID is referenced via `network_id`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the WireGuard peer.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the WireGuard server belongs to.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the WireGuard server network the peer connects to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the WireGuard peer.",
				Required:            true,
			},
			"interface_ip": schema.StringAttribute{
				MarkdownDescription: "The tunnel IP assigned to the peer, without mask (e.g. `192.0.2.10`). " +
					"Must be within the WireGuard server's subnet.",
				Required: true,
				Validators: []validator.String{
					validators.IPv4Validator(),
				},
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The WireGuard public key of the peer.",
				Required:            true,
			},
			"allowed_ips": schema.ListAttribute{
				MarkdownDescription: "Additional CIDRs routed to this peer beyond its tunnel IP.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default: listdefault.StaticValue(
					types.ListValueMust(types.StringType, []attr.Value{}),
				),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(validators.CIDRValidator()),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *wireguardPeerResource) Configure(
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

func (r *wireguardPeerResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data wireguardPeerResourceModel

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

	peer, diags := r.modelToPeer(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdPeer, err := r.client.CreateWireGuardPeer(ctx, site, data.NetworkID.ValueString(), peer)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating WireGuard Peer",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.peerToModel(ctx, createdPeer, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wireguardPeerResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data wireguardPeerResourceModel

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

	peer, err := r.client.GetWireGuardPeer(
		ctx,
		site,
		data.NetworkID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading WireGuard Peer",
			"Could not read WireGuard peer with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.peerToModel(ctx, peer, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wireguardPeerResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data wireguardPeerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := data.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	peer, diags := r.modelToPeer(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	peer.ID = data.ID.ValueString()

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	updatedPeer, err := r.client.UpdateWireGuardPeer(ctx, site, data.NetworkID.ValueString(), peer)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating WireGuard Peer",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.peerToModel(ctx, updatedPeer, &data, site)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wireguardPeerResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data wireguardPeerResourceModel

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

	err := r.client.DeleteWireGuardPeer(
		ctx,
		site,
		data.NetworkID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting WireGuard Peer",
			err.Error(),
		)
	}
}

func (r *wireguardPeerResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import format: "site:network_id:id" or "network_id:id" for default site
	idParts := strings.Split(req.ID, ":")

	switch len(idParts) {
	case 3:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(
			resp.State.SetAttribute(ctx, path.Root("network_id"), idParts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[2])...)
	case 2:
		resp.Diagnostics.Append(
			resp.State.SetAttribute(ctx, path.Root("network_id"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'site:network_id:id' or 'network_id:id'",
		)
	}
}

// modelToPeer converts the Terraform model to the API struct.
func (r *wireguardPeerResource) modelToPeer(
	ctx context.Context,
	model *wireguardPeerResourceModel,
) (*unifi.WireGuardPeer, diag.Diagnostics) {
	peer := &unifi.WireGuardPeer{
		Name:        model.Name.ValueString(),
		InterfaceIP: model.InterfaceIP.ValueString(),
		PublicKey:   model.PublicKey.ValueString(),
		AllowedIPs:  []string{},
	}

	var diags diag.Diagnostics
	if !model.AllowedIPs.IsNull() && !model.AllowedIPs.IsUnknown() {
		diags = model.AllowedIPs.ElementsAs(ctx, &peer.AllowedIPs, false)
	}

	return peer, diags
}

// peerToModel converts the API struct to the Terraform model.
func (r *wireguardPeerResource) peerToModel(
	ctx context.Context,
	peer *unifi.WireGuardPeer,
	model *wireguardPeerResourceModel,
	site string,
) diag.Diagnostics {
	model.ID = types.StringValue(peer.ID)
	model.Site = types.StringValue(site)
	model.NetworkID = types.StringValue(peer.NetworkID)
	model.Name = types.StringValue(peer.Name)
	model.InterfaceIP = types.StringValue(peer.InterfaceIP)
	model.PublicKey = types.StringValue(peer.PublicKey)

	allowedIPs, diags := types.ListValueFrom(ctx, types.StringType, peer.AllowedIPs)
	model.AllowedIPs = allowedIPs
	return diags
}

// ListResourceConfigSchema implements [list.ListResource]. Peers belong to a
// WireGuard server network, so `network_id` is required.
func (r *wireguardPeerResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List WireGuard peers of a WireGuard server network.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site the WireGuard server belongs to.",
				Optional:            true,
			},
			"network_id": listschema.StringAttribute{
				MarkdownDescription: "The ID of the WireGuard server network to list peers from.",
				Required:            true,
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
func (r *wireguardPeerResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config wireguardPeerListConfigModel

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
	var filters []wireguardPeerListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	// Peers belong to a VPN-server network; network_id is required.
	peers, err := r.client.ListWireGuardPeers(ctx, site, config.NetworkID.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError(
			"Error Listing WireGuard Peers",
			"Could not list WireGuard peers: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for i := range peers {
			peer := peers[i]

			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if peer.Name != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if peer.Name != "" {
				result.DisplayName = peer.Name
			} else {
				result.DisplayName = peer.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(peer.ID),
				)...,
			)

			// Convert to model.
			var model wireguardPeerResourceModel
			result.Diagnostics.Append(r.peerToModel(ctx, &peer, &model, site)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
