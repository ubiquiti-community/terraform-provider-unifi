package unifi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &clientFrameworkResource{}
	_ resource.ResourceWithImportState = &clientFrameworkResource{}
)

func NewClientFrameworkResource() resource.Resource {
	return &clientFrameworkResource{}
}

// clientFrameworkResource defines the resource implementation.
type clientFrameworkResource struct {
	client *Client
}

// clientFrameworkResourceModel describes the resource data model.
type clientFrameworkResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Site                types.String `tfsdk:"site"`
	MAC                 types.String `tfsdk:"mac"`
	Name                types.String `tfsdk:"name"`
	GroupID             types.String `tfsdk:"group_id"`
	Note                types.String `tfsdk:"note"`
	FixedIP             types.String `tfsdk:"fixed_ip"`
	NetworkID           types.String `tfsdk:"network_id"`
	Blocked             types.Bool   `tfsdk:"blocked"`
	DevIDOverride       types.Int64  `tfsdk:"dev_id_override"`
	LocalDNSRecord      types.String `tfsdk:"local_dns_record"`
	AllowExisting       types.Bool   `tfsdk:"allow_existing"`
	SkipForgetOnDestroy types.Bool   `tfsdk:"skip_forget_on_destroy"`

	// Computed attributes
	Hostname types.String `tfsdk:"hostname"`
	IP       types.String `tfsdk:"ip"`
}

func (r *clientFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

func (r *clientFrameworkResource) Schema(
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
				Required:            true,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The group ID to attach to the client (controls QoS and other group-based settings).",
				Optional:            true,
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A note with additional information for the client.",
				Optional:            true,
			},
			"fixed_ip": schema.StringAttribute{
				MarkdownDescription: "A fixed IPv4 address for this client.",
				Optional:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID for this client.",
				Optional:            true,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this client should be blocked from the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"dev_id_override": schema.Int64Attribute{
				MarkdownDescription: "Override the device fingerprint.",
				Optional:            true,
			},
			"local_dns_record": schema.StringAttribute{
				MarkdownDescription: "Specifies the local DNS record for this client.",
				Optional:            true,
			},
			"allow_existing": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should just take over control of an existing client.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"skip_forget_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should tell the controller to \"forget\" the client on destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clientFrameworkResource) Configure(
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

func (r *clientFrameworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan clientFrameworkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the plan to UniFi Client struct
	client, diags := r.planToClient(ctx, plan)
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

		// Implement merge pattern for existing client
		mergedClient := r.mergeClient(existingClient, client)
		updatedClient, err := r.client.UpdateClient(ctx, site, mergedClient)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Existing Client",
				"Could not update existing client: "+err.Error(),
			)
			return
		}
		createdClient = updatedClient
	}

	// Convert response back to model
	diags = r.clientToModel(ctx, createdClient, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clientFrameworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state clientFrameworkResourceModel

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

	// Get the Client from the API
	client, err := r.client.GetClient(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client",
			"Could not read client with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert API response to model
	diags = r.clientToModel(ctx, client, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *clientFrameworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state clientFrameworkResourceModel
	var plan clientFrameworkResourceModel

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
	client, diags := r.planToClient(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 4: Send to API
	client.ID = state.ID.ValueString()
	updatedClient, err := r.client.UpdateClient(ctx, site, client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Client",
			"Could not update client with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	diags = r.clientToModel(ctx, updatedClient, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *clientFrameworkResource) applyPlanToState(
	_ context.Context,
	plan *clientFrameworkResourceModel,
	state *clientFrameworkResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.MAC.IsNull() && !plan.MAC.IsUnknown() {
		state.MAC = plan.MAC
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		state.GroupID = plan.GroupID
	}
	if !plan.Note.IsNull() && !plan.Note.IsUnknown() {
		state.Note = plan.Note
	}
	if !plan.FixedIP.IsNull() && !plan.FixedIP.IsUnknown() {
		state.FixedIP = plan.FixedIP
	}
	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		state.NetworkID = plan.NetworkID
	}
	if !plan.Blocked.IsNull() && !plan.Blocked.IsUnknown() {
		state.Blocked = plan.Blocked
	}
	if !plan.DevIDOverride.IsNull() && !plan.DevIDOverride.IsUnknown() {
		state.DevIDOverride = plan.DevIDOverride
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

func (r *clientFrameworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state clientFrameworkResourceModel

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

func (r *clientFrameworkResource) ImportState(
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

func (r *clientFrameworkResource) planToClient(
	_ context.Context,
	plan clientFrameworkResourceModel,
) (*unifi.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.ID.IsNull() && plan.Name.IsNull() && plan.MAC.IsNull() {
		diags.AddError(
			"Invalid Client",
			"Client must have either an ID, Name, or MAC to be imported",
		)
		return nil, diags
	}

	client := &unifi.Client{
		ID:          plan.ID.ValueString(),
		MAC:         plan.MAC.ValueString(),
		Name:        plan.Name.ValueString(),
		UserGroupID: plan.GroupID.ValueString(),
		Note:        plan.Note.ValueString(),
		FixedIP:     plan.FixedIP.ValueString(),
		NetworkID:   plan.NetworkID.ValueString(),
		Blocked: func() string {
			if plan.Blocked.ValueBool() {
				return "true"
			} else {
				return "false"
			}
		}(),
		LocalDNSRecord: plan.LocalDNSRecord.ValueString(),
	}

	// Note: DevIDOverride is not available in the Client type
	// if !plan.DevIDOverride.IsNull() && !plan.DevIDOverride.IsUnknown() {
	// 	client.DevIdOverride = int(plan.DevIDOverride.ValueInt64())
	// }

	return client, diags
}

func (r *clientFrameworkResource) clientToModel(
	_ context.Context,
	client *unifi.Client,
	model *clientFrameworkResourceModel,
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

	model.ID = types.StringValue(client.ID)
	model.Site = types.StringValue(site)
	model.MAC = types.StringValue(client.MAC)
	model.Name = types.StringValue(client.Name)

	if client.UserGroupID != "" {
		model.GroupID = types.StringValue(client.UserGroupID)
	} else {
		model.GroupID = types.StringNull()
	}

	if client.Note != "" {
		model.Note = types.StringValue(client.Note)
	} else {
		model.Note = types.StringNull()
	}

	if client.FixedIP != "" {
		model.FixedIP = types.StringValue(client.FixedIP)
	} else {
		model.FixedIP = types.StringNull()
	}

	if client.NetworkID != "" {
		model.NetworkID = types.StringValue(client.NetworkID)
	} else {
		model.NetworkID = types.StringNull()
	}

	// Blocked field is string in Client type
	model.Blocked = types.BoolValue(client.Blocked == "true")

	// DevIdOverride not available in Client type
	model.DevIDOverride = types.Int64Null()

	if client.LocalDNSRecord != "" {
		model.LocalDNSRecord = types.StringValue(client.LocalDNSRecord)
	} else {
		model.LocalDNSRecord = types.StringNull()
	}

	// Computed attributes
	if client.Hostname != "" {
		model.Hostname = types.StringValue(client.Hostname)
	} else {
		model.Hostname = types.StringNull()
	}

	if client.LastSeen != "" {
		model.IP = types.StringValue(client.LastSeen)
	} else {
		model.IP = types.StringNull()
	}

	return diags
}

func (r *clientFrameworkResource) mergeClient(
	existing *unifi.Client,
	planned *unifi.Client,
) *unifi.Client {
	// Start with the existing client to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values
	merged.Name = planned.Name
	merged.UserGroupID = planned.UserGroupID
	merged.Note = planned.Note
	merged.FixedIP = planned.FixedIP
	merged.NetworkID = planned.NetworkID
	merged.Blocked = planned.Blocked
	merged.LocalDNSRecord = planned.LocalDNSRecord

	return &merged
}
