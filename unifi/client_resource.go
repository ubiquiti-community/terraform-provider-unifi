package unifi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &clientResource{}
	_ resource.ResourceWithImportState = &clientResource{}
	_ resource.ResourceWithIdentity    = &clientResource{}
	_ resource.ResourceWithImportState = &clientResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &clientResource{}
	_ list.ListResourceWithConfigure = &clientResource{}
)

const (
	defaultSkipForgetOnDestroy = false
	defaultAllowExisting       = true
)

func NewClientResource() resource.Resource {
	return &clientResource{}
}

func NewClientListResource() list.ListResource {
	return &clientResource{}
}

// clientResource defines the resource implementation.
type clientResource struct {
	client *Client

	// Cache group name → ID lookups per site to avoid repeated API calls during List.
	groupCacheMu sync.Mutex
	groupCache   map[string]map[string]string // site → (name → id)
}

// qosRateModel describes the nested qos_rate attribute.
type qosRateModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	MaxUp   types.Int64  `tfsdk:"max_up"`
	MaxDown types.Int64  `tfsdk:"max_down"`
}

func (m qosRateModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":       types.StringType,
		"name":     types.StringType,
		"max_up":   types.Int64Type,
		"max_down": types.Int64Type,
	}
}

// clientResourceModel describes the resource data model.
type clientResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	MAC            types.String `tfsdk:"mac"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	QOSRate        types.Object `tfsdk:"qos_rate"`
	Note           types.String `tfsdk:"note"`
	FixedIP        types.String `tfsdk:"fixed_ip"`
	FixedApMAC     types.String `tfsdk:"fixed_ap_mac"`
	NetworkID      types.String `tfsdk:"network_id"`
	Groups         types.List   `tfsdk:"groups"`
	Blocked        types.Bool   `tfsdk:"blocked"`
	LocalDNSRecord types.String `tfsdk:"local_dns_record"`

	// These control import and create behavior to allow the resource to take over existing clients instead of erroring, and to allow it to just be removed from Terraform management without deleting in UniFi.
	AllowExisting       types.Bool `tfsdk:"allow_existing"`
	SkipForgetOnDestroy types.Bool `tfsdk:"skip_forget_on_destroy"`

	// Computed attributes
	Hostname types.String `tfsdk:"hostname"`
}

type clientIdentityModel struct {
	MAC types.String `tfsdk:"mac"`
}

// clientListConfigModel describes the list configuration model.
type clientListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Group  types.String `tfsdk:"group"`
	Filter types.List   `tfsdk:"filter"`
}

// clientListFilterModel represents a single name/values filter entry.
type clientListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *clientResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *clientResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"mac": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

func (r *clientResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a client of the network, identified by unique MAC addresses.

Clients are created in the controller when observed on the network, so the resource defaults to allowing itself to just take over management of a MAC address, but this can be turned off.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the client with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the client.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"qos_rate": schema.SingleNestedAttribute{
				MarkdownDescription: "QoS rate limiting configuration. Controls the client group (usergroup) used for bandwidth limits.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "The ID of the client group (usergroup). If set, this group is used directly.",
						Optional:            true,
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The name of the client group. If set, the group is looked up or created by name.",
						Optional:            true,
						Computed:            true,
					},
					"max_up": schema.Int64Attribute{
						MarkdownDescription: "Maximum upload rate in kbps.",
						Optional:            true,
						Computed:            true,
					},
					"max_down": schema.Int64Attribute{
						MarkdownDescription: "Maximum download rate in kbps.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A note with additional information for the client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fixed_ip": schema.StringAttribute{
				MarkdownDescription: "A fixed IPv4 address for this client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fixed_ap_mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the access point to which this client should be fixed.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID for this client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of network members group names for this client.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this client should be blocked from the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"local_dns_record": schema.StringAttribute{
				MarkdownDescription: "Specifies the local DNS record for this client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_existing": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should just take over control of an existing client.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(defaultAllowExisting),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"skip_forget_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should tell the controller to \"forget\" the client on destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(defaultSkipForgetOnDestroy),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clientResource) Configure(
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

func (r *clientResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan clientResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize identity from plan
	id := clientIdentityModel{
		MAC: plan.MAC,
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the plan to UniFi Client struct
	client, diags := r.planToClient(ctx, site, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowExisting := plan.AllowExisting.ValueBool()

	// Create the Client
	createdClient, err := r.client.CreateClient(ctx, site, client)
	if err != nil {
		var apiErr *unifi.APIError
		if !errors.As(err, &apiErr) || (apiErr.Message != "api.err.MacUsed" || !allowExisting) {
			resp.Diagnostics.AddError(
				"Error Creating Client",
				"Could not create client: "+err.Error(),
			)
			return
		}

		// MAC in use, just absorb the existing client
		mac := plan.MAC.ValueString()
		existingClient, err := r.client.GetClientByMAC(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Existing Client",
				"Could not get existing client with MAC "+mac+": "+err.Error(),
			)
			return
		}

		pclient, err := r.client.GetClient(ctx, site, existingClient.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Existing Client by ID",
				"Could not get existing client with ID "+existingClient.ID+": "+err.Error(),
			)
		}

		if pclient != nil {
			createdClient = pclient
		} else {
			createdClient = existingClient
		}
	}

	createdClient, diags = r.reconcileCreatedClient(ctx, site, createdClient, client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert response back to model
	diags = r.clientToModel(ctx, createdClient, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update identity with final MAC
	id.MAC = plan.MAC

	resp.Diagnostics.Append(resp.Identity.Set(ctx, id)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clientResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state clientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get identity (MAC address)
	id := clientIdentityModel{}
	if d := req.Identity.Get(ctx, &id); d.HasError() || id.MAC.IsNull() {
		// Fall back to state MAC if identity not available
		id.MAC = state.MAC
	}

	mac := id.MAC.ValueString()
	if mac == "" {
		resp.Diagnostics.AddError(
			"Invalid State",
			"Client must have a MAC address",
		)
		return
	}

	if state.MAC.IsNull() || state.MAC.IsUnknown() {
		state.MAC = id.MAC
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the Client from the API
	var client *unifi.Client
	var err error

	// If we have an ID in state, use it (normal operation after first apply/import)
	if !state.ID.IsNull() && state.ID.ValueString() != "" {
		tflog.Debug(ctx, "Reading client by ID", map[string]any{"id": state.ID.ValueString()})
		client, err = r.client.GetClient(ctx, site, state.ID.ValueString())
		if err != nil {
			if _, ok := err.(*unifi.NotFoundError); ok {
				// Client was deleted externally - remove from state
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError(
				"Error Reading Client",
				"Could not read client with ID "+state.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else {
		// No ID in state - this is during import, use MAC
		tflog.Debug(ctx, "Reading client by MAC (import scenario)", map[string]any{"mac": mac})
		client, err = r.client.GetClientByMAC(ctx, site, mac)
		if err != nil {
			if _, ok := err.(*unifi.NotFoundError); ok {
				// Client doesn't exist yet - create it during import
				tflog.Info(
					ctx,
					"Client not found during import, creating",
					map[string]any{"mac": mac},
				)
				newClient := &unifi.Client{MAC: mac}
				_, createErr := r.client.CreateClient(ctx, site, newClient)
				if createErr != nil {
					// CreateClient may return NotFoundError if the API returns empty data
					// but the client was still created. Try to fetch it.
					if _, ok := createErr.(*unifi.NotFoundError); !ok {
						resp.Diagnostics.AddError(
							"Error Creating Client During Import",
							"Client with MAC "+mac+" does not exist and could not be created: "+createErr.Error(),
						)
						return
					}
				}

				// Fetch the client we just created
				tflog.Debug(
					ctx,
					"Attempting to fetch newly created client",
					map[string]any{"mac": mac, "site": site},
				)

				// List all clients to find the one we created
				allClients, listErr := r.client.ListClient(ctx, site)
				if listErr != nil {
					resp.Diagnostics.AddError(
						"Error Listing Clients After Creation",
						"Could not list clients: "+listErr.Error(),
					)
					return
				}

				tflog.Debug(ctx, "Listed all clients", map[string]any{"count": len(allClients)})

				// Find our client by MAC (case-insensitive)
				var foundClient *unifi.Client
				for i := range allClients {
					if strings.EqualFold(allClients[i].MAC, mac) {
						foundClient = &allClients[i]
						break
					}
				}

				if foundClient == nil {
					resp.Diagnostics.AddError(
						"Error Reading Client After Creation",
						"Client with MAC "+mac+" was created but could not be found in the client list",
					)
					return
				}
				client = foundClient
			} else {
				resp.Diagnostics.AddError(
					"Error Reading Client",
					"Could not read client with MAC "+mac+": "+err.Error(),
				)
				return
			}
		}
	}

	// Convert API response to model
	resp.Diagnostics.Append(r.clientToModel(ctx, client, &state, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id.MAC.IsNull() || id.MAC.IsUnknown() {
		id.MAC = state.MAC
	}

	if state.AllowExisting.IsNull() || state.AllowExisting.IsUnknown() {
		state.AllowExisting = types.BoolValue(defaultAllowExisting)
	}

	if state.SkipForgetOnDestroy.IsNull() || state.SkipForgetOnDestroy.IsUnknown() {
		state.SkipForgetOnDestroy = types.BoolValue(defaultSkipForgetOnDestroy)
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, id)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clientResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state clientResourceModel
	var plan clientResourceModel

	// Read the current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError(
			"Invalid State",
			"Client must have an ID",
		)
		return
	}

	// Step 1: Get the current client by ID
	currentClient, err := r.client.GetClient(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			// Client no longer exists, recreate it
			planClient, diags := r.planToClient(ctx, site, plan)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			createdClient, err := r.client.CreateClient(ctx, site, planClient)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Recreating Client",
					"Could not recreate client: "+err.Error(),
				)
				return
			}

			createdClient, diags = r.reconcileCreatedClient(ctx, site, createdClient, planClient)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Convert response back to model
			diags = r.clientToModel(ctx, createdClient, &state, site)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Update identity with MAC
			identityModel := clientIdentityModel{
				MAC: state.MAC,
			}

			resp.Diagnostics.Append(resp.Identity.Set(ctx, identityModel)...)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Client for Update",
			"Could not read client with ID "+id+": "+err.Error(),
		)
		return
	}

	// Step 2: Convert plan to client format to get the desired changes
	planClient, diags := r.planToClient(ctx, site, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 3: Merge the plan changes into the current client
	mergedClient := r.mergeClient(currentClient, planClient)

	// Step 4: Update the client via API
	updatedClient, err := r.client.UpdateClient(ctx, site, mergedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Client",
			"Could not update client with ID "+id+": "+err.Error(),
		)
		return
	}

	// Step 6: Convert the fetched client to state model
	diags = r.clientToModel(ctx, updatedClient, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update identity with MAC
	identityModel := clientIdentityModel{
		MAC: state.MAC,
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, identityModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *clientResource) applyPlanToState( //nolint:unused
	_ context.Context,
	plan *clientResourceModel,
	state *clientResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.MAC.IsNull() && !plan.MAC.IsUnknown() {
		state.MAC = plan.MAC
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		state.DisplayName = plan.DisplayName
	}
	if !plan.QOSRate.IsNull() && !plan.QOSRate.IsUnknown() {
		state.QOSRate = plan.QOSRate
	}
	if !plan.Note.IsNull() && !plan.Note.IsUnknown() {
		state.Note = plan.Note
	}
	if !plan.FixedIP.IsNull() && !plan.FixedIP.IsUnknown() {
		state.FixedIP = plan.FixedIP
	}
	if !plan.FixedApMAC.IsNull() && !plan.FixedApMAC.IsUnknown() {
		state.FixedApMAC = plan.FixedApMAC
	}
	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		state.NetworkID = plan.NetworkID
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		state.Groups = plan.Groups
	}
	if !plan.Blocked.IsNull() && !plan.Blocked.IsUnknown() {
		state.Blocked = plan.Blocked
	}
	if !plan.LocalDNSRecord.IsNull() && !plan.LocalDNSRecord.IsUnknown() {
		state.LocalDNSRecord = plan.LocalDNSRecord
	}
	if !plan.AllowExisting.IsNull() && !plan.AllowExisting.IsUnknown() {
		state.AllowExisting = plan.AllowExisting
	}
	if !plan.SkipForgetOnDestroy.IsNull() && !plan.SkipForgetOnDestroy.IsUnknown() {
		state.SkipForgetOnDestroy = plan.SkipForgetOnDestroy
	}
	// Note: Computed attributes (Hostname, IP) are not applied from plan
}

func (r *clientResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state clientResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	skipForget := state.SkipForgetOnDestroy.ValueBool()

	if skipForget {
		// Just remove from Terraform state without telling UniFi to forget
		return
	}

	// lookup MAC instead of trusting state
	c, err := r.client.GetClient(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client for Delete",
			"Could not read client with ID "+id+": "+err.Error(),
		)
		return
	}

	err = r.client.DeleteClientByMAC(ctx, site, c.MAC)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Client",
			"Could not delete client with MAC "+c.MAC+": "+err.Error(),
		)
		return
	}
}

func (r *clientResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	if req.ID != "" {
		if !strings.Contains(req.ID, ":") {
			resp.Diagnostics.AddError(
				"Invalid import ID",
				"Client can only be imported using a MAC address",
			)
		}
		// Set identity with the MAC
		idModel := clientIdentityModel{MAC: types.StringValue(req.ID)}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Import the state using MAC attribute
	resource.ImportStatePassthroughWithIdentity(
		ctx,
		path.Root("mac"),
		path.Root("mac"),
		req,
		resp,
	)
}

// Helper functions for conversion and merging

func (r *clientResource) planToClient(
	ctx context.Context,
	site string,
	plan clientResourceModel,
) (*unifi.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.ID.IsNull() && plan.Name.IsNull() && plan.MAC.IsNull() {
		diags.AddError(
			"Invalid Client",
			"Client must have either an ID, Name, or MAC to be imported",
		)
		return nil, diags
	}

	fixedIP := plan.FixedIP.ValueString()
	fixedApMAC := plan.FixedApMAC.ValueString()
	localDNSRecord := plan.LocalDNSRecord.ValueString()
	networkID := plan.NetworkID.ValueString()

	// Resolve tag names to network members group IDs
	var networkMembersGroupIDs []string
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		var tagNames []string
		diags.Append(plan.Groups.ElementsAs(ctx, &tagNames, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, name := range tagNames {
			id, err := r.resolveGroupID(ctx, site, name)
			if err != nil {
				diags.AddError(
					"Error Resolving Tag",
					fmt.Sprintf("Could not resolve tag %q: %s", name, err.Error()),
				)
				return nil, diags
			}
			networkMembersGroupIDs = append(networkMembersGroupIDs, id)
		}
	}

	client := &unifi.Client{
		ID:          plan.ID.ValueString(),
		MAC:         plan.MAC.ValueString(),
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Note:        plan.Note.ValueString(),
		Blocked:     plan.Blocked.ValueBoolPointer(),

		// FixedIP and its enable flag
		FixedIP:    fixedIP,
		UseFixedIP: fixedIP != "",

		// FixedAp and its enable flag
		FixedApMAC:     fixedApMAC,
		FixedApEnabled: fixedApMAC != "",

		// LocalDNSRecord and its enable flag
		LocalDNSRecord:        localDNSRecord,
		LocalDNSRecordEnabled: localDNSRecord != "",

		// NetworkID maps to VirtualNetworkOverrideID with its enable flag
		VirtualNetworkOverrideID: networkID,

		// Network members group IDs
		NetworkMembersGroupIDs: networkMembersGroupIDs,
	}

	if networkID != "" {
		client.VirtualNetworkOverrideEnabled = util.Ptr(true)
	}

	// Resolve qos_rate to a client group (usergroup) ID.
	if !plan.QOSRate.IsNull() && !plan.QOSRate.IsUnknown() {
		var qos qosRateModel
		d := plan.QOSRate.As(ctx, &qos, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}

		groupID, d := r.resolveClientGroup(ctx, site, qos)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		client.UserGroupID = groupID
	}

	return client, diags
}

func (r *clientResource) reconcileCreatedClient(
	ctx context.Context,
	site string,
	currentClient *unifi.Client,
	plannedClient *unifi.Client,
) (*unifi.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if currentClient == nil || currentClient.ID == "" {
		existingClient, err := r.client.GetClientByMAC(ctx, site, plannedClient.MAC)
		if err != nil {
			diags.AddError(
				"Error Reading Created Client",
				"Could not read created client with MAC "+plannedClient.MAC+": "+err.Error(),
			)
			return nil, diags
		}

		fullClient, err := r.client.GetClient(ctx, site, existingClient.ID)
		if err == nil {
			currentClient = fullClient
		} else {
			currentClient = existingClient
		}
	}

	mergedClient := r.mergeClient(currentClient, plannedClient)
	tflog.Debug(ctx, "Merging created client", map[string]any{
		"site":        site,
		"mac":         plannedClient.MAC,
		"merged_id":   mergedClient.ID,
	})
	updatedClient, err := r.client.UpdateClient(ctx, site, mergedClient)
	if err != nil {
		diags.AddError(
			"Error Updating Created Client",
			"Could not update created client: "+err.Error(),
		)
		return nil, diags
	}

	if updatedClient != nil && updatedClient.ID != "" {
		freshClient, err := r.client.GetClient(ctx, site, updatedClient.ID)
		if err == nil {
			return freshClient, diags
		}
	}

	return updatedClient, diags
}

func (r *clientResource) clientToModel(
	ctx context.Context,
	client *unifi.Client,
	model *clientResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if client.ID == "" && client.Name == "" && client.MAC == "" {
		diags.AddError(
			"Invalid Client",
			"Client must have either an ID, Name, or MAC to be imported",
		)
		return diags
	}

	model.ID = util.StringValueOrNull(client.ID)
	model.Site = util.StringValueOrNull(site)
	model.MAC = util.StringValueOrNull(client.MAC)
	model.Name = util.StringValueOrNull(client.Name)
	model.DisplayName = util.StringValueOrNull(client.DisplayName)
	model.Note = util.StringValueOrNull(client.Note)
	model.FixedIP = util.StringValueOrNull(client.FixedIP)
	model.FixedApMAC = util.StringValueOrNull(client.FixedApMAC)
	model.NetworkID = util.StringValueOrNull(client.VirtualNetworkOverrideID)

	// Populate qos_rate from the client's UserGroupID by looking up the client group.
	if client.UserGroupID != "" {
		group, err := r.client.GetClientGroup(ctx, site, client.UserGroupID)
		if err != nil {
			diags.AddError(
				"Error Reading Client Group",
				fmt.Sprintf("Could not read client group %q: %s", client.UserGroupID, err.Error()),
			)
			return diags
		}
		qos := qosRateModel{
			ID:      types.StringValue(group.ID),
			Name:    types.StringValue(group.Name),
			MaxUp:   types.Int64PointerValue(group.QOSRateMaxUp),
			MaxDown: types.Int64PointerValue(group.QOSRateMaxDown),
		}
		var objDiags diag.Diagnostics
		model.QOSRate, objDiags = types.ObjectValueFrom(ctx, qosRateModel{}.AttributeTypes(), qos)
		diags.Append(objDiags...)
	} else {
		model.QOSRate = types.ObjectNull(qosRateModel{}.AttributeTypes())
	}

	// Resolve NetworkMembersGroupIDs (IDs) back to tag names
	if len(client.NetworkMembersGroupIDs) > 0 {
		tagNames, err := r.resolveGroupNames(ctx, site, client.NetworkMembersGroupIDs)
		if err != nil {
			diags.AddError("Error Resolving Groups", err.Error())
			return diags
		}
		elements := make([]attr.Value, len(tagNames))
		for i, name := range tagNames {
			elements[i] = types.StringValue(name)
		}
		var listDiags diag.Diagnostics
		model.Groups, listDiags = types.ListValue(types.StringType, elements)
		diags.Append(listDiags...)
	} else {
		model.Groups = types.ListNull(types.StringType)
	}

	model.Blocked = types.BoolPointerValue(client.Blocked)
	model.LocalDNSRecord = util.StringValueOrNull(client.LocalDNSRecord)

	// Computed attributes
	model.Hostname = util.StringValueOrNull(client.Hostname)

	return diags
}

func (r *clientResource) mergeClient(
	existing *unifi.Client,
	planned *unifi.Client,
) *unifi.Client {
	// Start with the existing client to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values - these are all writable fields
	merged.Name = planned.Name
	merged.DisplayName = planned.DisplayName
	merged.UserGroupID = planned.UserGroupID
	merged.Note = planned.Note
	merged.Blocked = planned.Blocked
	merged.NetworkMembersGroupIDs = planned.NetworkMembersGroupIDs

	// FixedIP and its enable flag
	merged.FixedIP = planned.FixedIP
	merged.UseFixedIP = planned.FixedIP != ""

	// LocalDNSRecord and its enable flag
	merged.LocalDNSRecord = planned.LocalDNSRecord
	merged.LocalDNSRecordEnabled = planned.LocalDNSRecord != ""

	// NetworkID (maps to VirtualNetworkOverrideID) and its enable flag
	merged.VirtualNetworkOverrideID = planned.VirtualNetworkOverrideID

	if planned.VirtualNetworkOverrideID != "" {
		merged.VirtualNetworkOverrideEnabled = util.Ptr(true)
	}

	// FixedAP and its enable flag
	merged.FixedApMAC = planned.FixedApMAC
	merged.FixedApEnabled = planned.FixedApMAC != ""

	return &merged
}

func (r *clientResource) ListResourceConfigSchema(
	ctx context.Context,
	req list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List clients in a site, optionally filtered by network, group, connection type, or vendor.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list clients from.",
				Optional:            true,
			},
			"group": listschema.StringAttribute{
				MarkdownDescription: "Filter clients by network members group name.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `site`, `network_id`, `network_name`, `group`, `wired`, `blocked`, `oui`.",
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

// resolveGroupNames looks up network members group names by their IDs.
// Uses the same cache as resolveGroupID (populating it if needed).
func (r *clientResource) resolveGroupNames(
	ctx context.Context,
	site string,
	ids []string,
) ([]string, error) {
	r.groupCacheMu.Lock()
	defer r.groupCacheMu.Unlock()

	if r.groupCache == nil {
		r.groupCache = make(map[string]map[string]string)
	}

	// Ensure cache is populated for this site.
	if _, ok := r.groupCache[site]; !ok {
		groups, err := r.client.ListNetworkMembersGroups(ctx, site)
		if err != nil {
			return nil, fmt.Errorf("listing network members groups: %w", err)
		}
		siteCache := make(map[string]string, len(groups))
		for _, g := range groups {
			siteCache[g.Name] = g.ID
		}
		r.groupCache[site] = siteCache
	}

	// Build reverse map id → name from cache.
	siteCache := r.groupCache[site]
	idToName := make(map[string]string, len(siteCache))
	for name, id := range siteCache {
		idToName[id] = name
	}

	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name, ok := idToName[id]
		if !ok {
			return nil, fmt.Errorf("network members group with ID %q not found", id)
		}
		names = append(names, name)
	}
	return names, nil
}

// resolveGroupID looks up a network members group by name and returns its ID.
// Results are cached per site to avoid repeated API calls.
func (r *clientResource) resolveGroupID(
	ctx context.Context,
	site, groupName string,
) (string, error) {
	r.groupCacheMu.Lock()
	defer r.groupCacheMu.Unlock()

	if r.groupCache == nil {
		r.groupCache = make(map[string]map[string]string)
	}

	if siteCache, ok := r.groupCache[site]; ok {
		if id, ok := siteCache[groupName]; ok {
			return id, nil
		}
	}

	// Fetch all groups for this site and populate the cache.
	groups, err := r.client.ListNetworkMembersGroups(ctx, site)
	if err != nil {
		return "", fmt.Errorf("listing network members groups: %w", err)
	}

	siteCache := make(map[string]string, len(groups))
	for _, g := range groups {
		siteCache[g.Name] = g.ID
	}
	r.groupCache[site] = siteCache

	id, ok := siteCache[groupName]
	if ok {
		return id, nil
	}

	// Group not found — create it.
	created, err := r.client.CreateNetworkMembersGroup(ctx, site, &unifi.NetworkMembersGroup{
		Name:    groupName,
		Members: []string{},
		Type:    "CLIENTS",
	})
	if err != nil {
		return "", fmt.Errorf("creating network members group %q: %w", groupName, err)
	}

	siteCache[groupName] = created.ID
	return created.ID, nil
}

// resolveClientGroup resolves a qosRateModel to a client group (usergroup) ID.
// If ID is set, it is used directly.
// If name is set, the group is looked up or created by name with the specified rates.
// If neither is set, a name is derived from max_up/max_down and the group is looked up or created.
func (r *clientResource) resolveClientGroup(
	ctx context.Context,
	site string,
	qos qosRateModel,
) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Case 1: ID is explicitly provided — use it directly.
	if !qos.ID.IsNull() && !qos.ID.IsUnknown() && qos.ID.ValueString() != "" {
		return qos.ID.ValueString(), diags
	}

	// Determine the group name.
	var groupName string
	if !qos.Name.IsNull() && !qos.Name.IsUnknown() && qos.Name.ValueString() != "" {
		// Case 2: Name is provided.
		groupName = qos.Name.ValueString()
	} else {
		// Case 3: Auto-generate name from rate limits.
		maxUp := int64(-1)
		maxDown := int64(-1)
		if !qos.MaxUp.IsNull() && !qos.MaxUp.IsUnknown() {
			maxUp = qos.MaxUp.ValueInt64()
		}
		if !qos.MaxDown.IsNull() && !qos.MaxDown.IsUnknown() {
			maxDown = qos.MaxDown.ValueInt64()
		}
		groupName = fmt.Sprintf("qos-up%d-down%d", maxUp, maxDown)
	}

	// Look up or create the client group by name.
	groups, err := r.client.ListClientGroup(ctx, site)
	if err != nil {
		diags.AddError("Error Listing Client Groups", "Could not list client groups: "+err.Error())
		return "", diags
	}

	for _, g := range groups {
		if g.Name == groupName {
			// Update rates if they differ from what's requested.
			needsUpdate := false
			if !qos.MaxUp.IsNull() && !qos.MaxUp.IsUnknown() {
				desired := qos.MaxUp.ValueInt64()
				if g.QOSRateMaxUp == nil || *g.QOSRateMaxUp != desired {
					g.QOSRateMaxUp = &desired
					needsUpdate = true
				}
			}
			if !qos.MaxDown.IsNull() && !qos.MaxDown.IsUnknown() {
				desired := qos.MaxDown.ValueInt64()
				if g.QOSRateMaxDown == nil || *g.QOSRateMaxDown != desired {
					g.QOSRateMaxDown = &desired
					needsUpdate = true
				}
			}
			if needsUpdate {
				_, err := r.client.UpdateClientGroup(ctx, site, &g)
				if err != nil {
					diags.AddError(
						"Error Updating Client Group",
						fmt.Sprintf("Could not update client group %q: %s", groupName, err.Error()),
					)
					return "", diags
				}
			}
			return g.ID, diags
		}
	}

	// Group not found — create it.
	newGroup := &unifi.ClientGroup{
		Name: groupName,
	}
	if !qos.MaxUp.IsNull() && !qos.MaxUp.IsUnknown() {
		v := qos.MaxUp.ValueInt64()
		newGroup.QOSRateMaxUp = &v
	}
	if !qos.MaxDown.IsNull() && !qos.MaxDown.IsUnknown() {
		v := qos.MaxDown.ValueInt64()
		newGroup.QOSRateMaxDown = &v
	}

	created, err := r.client.CreateClientGroup(ctx, site, newGroup)
	if err != nil {
		diags.AddError(
			"Error Creating Client Group",
			fmt.Sprintf("Could not create client group %q: %s", groupName, err.Error()),
		)
		return "", diags
	}

	return created.ID, diags
}

func (r *clientResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config clientListConfigModel

	// Read list config data into the model.
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// apiFilters are passed directly to ListClientFiltered.
	// postFilters require in-memory evaluation after the API responds.
	// Each postFilter entry maps a field name to an OR-set of accepted values.
	apiFilters := make(map[string]string)
	postFilters := make(map[string]map[string]struct{})

	// Resolve the group attribute to an ID for post-filtering.
	var groupIDFilter string
	if !config.Group.IsNull() && !config.Group.IsUnknown() {
		groupID, err := r.resolveGroupID(ctx, site, config.Group.ValueString())
		if err != nil {
			var d diag.Diagnostics
			d.AddError("Error Resolving Group", err.Error())
			stream.Results = list.ListResultsStreamDiagnostics(d)
			return
		}
		groupIDFilter = groupID
	}

	// Process generic filter blocks.
	// API-passthrough names: oui, blocked, is_wired (first value used).
	// Post-filter names: network_id, network_name, name, display_name, fixed_ip (OR across values).

	filters := []clientListFilterModel{}
	config.Filter.ElementsAs(ctx, &filters, false)

	for _, f := range filters {
		name := f.Name.ValueString()
		value := f.Value.ValueString()

		switch name {
		case "network_id", "network_name", "name", "display_name", "fixed_ip":
			set := make(map[string]struct{}, 1)
			set[value] = struct{}{}
			postFilters[name] = set
		default:
			// Pass first value to the API; the API does not support OR within a field.
			apiFilters[name] = value
		}
	}

	// Fetch clients — use filtered endpoint only when API filters are present.
	var clients []unifi.Client
	var err error
	if len(apiFilters) > 0 {
		clients, err = r.client.ListClientFiltered(ctx, site, apiFilters)
	} else {
		clients, err = r.client.ListClient(ctx, site)
	}
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Clients", "Could not list clients: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	// Fetch active client info for display-name enrichment and network_name post-filtering.
	// Failures are non-fatal — enrichment is skipped if unavailable.
	infoByUserID := make(map[string]*unifi.ClientInfo)
	if activeClients, infoErr := r.client.ListClientInfo(ctx, site); infoErr == nil {
		for i := range activeClients {
			ci := &activeClients[i]
			if ci.UserId != "" {
				infoByUserID[ci.UserId] = ci
			}
		}
	}

	networkIDFilter := postFilters["network_id"]
	networkNameFilter := postFilters["network_name"]
	nameFilter := postFilters["name"]
	displayNameFilter := postFilters["display_name"]
	fixedIPFilter := postFilters["fixed_ip"]

	// Define the function that will push results into the stream.
	stream.Results = func(push func(list.ListResult) bool) {
		for _, client := range clients {
			info := infoByUserID[client.ID]

			// Post-filter by group ID: check if the resolved group ID is in the client's group list.
			if groupIDFilter != "" {
				found := slices.Contains(client.NetworkMembersGroupIDs, groupIDFilter)
				if !found {
					continue
				}
			}

			// Post-filter by name.
			if len(nameFilter) > 0 {
				if _, ok := nameFilter[client.Name]; !ok {
					continue
				}
			}

			// Post-filter by display_name.
			if len(displayNameFilter) > 0 {
				if _, ok := displayNameFilter[client.DisplayName]; !ok {
					continue
				}
			}

			// Post-filter by fixed_ip.
			if len(fixedIPFilter) > 0 {
				if _, ok := fixedIPFilter[client.FixedIP]; !ok {
					continue
				}
			}

			// Post-filter by network_id (OR across values): match VirtualNetworkOverrideID or NetworkID.
			if len(networkIDFilter) > 0 {
				clientNetworkID := client.VirtualNetworkOverrideID
				if clientNetworkID == "" {
					clientNetworkID = client.NetworkID
				}
				if _, ok := networkIDFilter[clientNetworkID]; !ok {
					continue
				}
			}

			// Post-filter by network_name (OR across values): uses active ClientInfo data.
			if len(networkNameFilter) > 0 {
				netName := ""
				if info != nil {
					netName = info.NetworkName
				}
				if _, ok := networkNameFilter[netName]; !ok {
					continue
				}
			}

			// Initialize a new result object for each client.
			result := req.NewListResult(ctx)

			// Set display name: prefer user-assigned name, then ClientInfo hostname,
			// then the stored hostname, falling back to MAC address.
			switch {
			case client.Name != "":
				result.DisplayName = client.Name
			case info != nil && info.Hostname != "":
				result.DisplayName = info.Hostname
			case client.Hostname != "":
				result.DisplayName = client.Hostname
			default:
				result.DisplayName = client.MAC
			}

			// Set resource identity (MAC address).
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("mac"),
					types.StringValue(client.MAC),
				)...)

			// Convert client to resource model and set the resource data.
			var model clientResourceModel
			result.Diagnostics.Append(r.clientToModel(ctx, &client, &model, site)...)
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			// Send the result to the stream.
			if !push(result) {
				return
			}
		}
	}
}
