package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &firewallGroupResource{}
	_ resource.ResourceWithImportState = &firewallGroupResource{}
)

func NewFirewallGroupFrameworkResource() resource.Resource {
	return &firewallGroupResource{}
}

// firewallGroupResource defines the resource implementation.
type firewallGroupResource struct {
	client *Client
}

// firewallGroupResourceModel describes the resource data model.
type firewallGroupResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Site    types.String `tfsdk:"site"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Members types.Set    `tfsdk:"members"`
}

func (r *firewallGroupResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_group"
}

func (r *firewallGroupResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "`unifi_firewall_group` manages groups of addresses or ports for use in firewall rules (`unifi_firewall_rule`).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the firewall group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "The name of the site to associate the firewall group with.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the firewall group.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the firewall group. Must be one of: `address-group`, `port-group`, or `ipv6-address-group`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("address-group", "port-group", "ipv6-address-group"),
				},
			},
			"members": schema.SetAttribute{
				Description: "The members of the firewall group.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *firewallGroupResource) Configure(
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

func (r *firewallGroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan firewallGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert model to API request
	firewallGroup, err := r.modelToAPIFirewallGroup(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Firewall Group",
			fmt.Sprintf("Could not convert firewall group to API format: %s", err),
		)
		return
	}

	apiFirewallGroup, err := r.client.CreateFirewallGroup(ctx, site, firewallGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Firewall Group",
			fmt.Sprintf("Could not create firewall group: %s", err),
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(apiFirewallGroup.ID)
	plan.Site = types.StringValue(site)
	r.setResourceData(ctx, apiFirewallGroup, &plan, site)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallGroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state firewallGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	firewallGroup, err := r.client.GetFirewallGroup(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Firewall Group",
			fmt.Sprintf("Could not read firewall group %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, firewallGroup, &state, site)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallGroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan firewallGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state firewallGroupResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()

	// Read current firewall group and merge with planned changes
	currentFirewallGroup, err := r.client.GetFirewallGroup(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Firewall Group for Update",
			fmt.Sprintf("Could not read firewall group %s for update: %s", id, err),
		)
		return
	}

	// Apply current API values to state
	r.setResourceData(ctx, currentFirewallGroup, &state, site)

	// Apply plan changes to the state (merge pattern)
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		state.Type = plan.Type
	}
	if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
		state.Members = plan.Members
	}

	// Convert updated state to API request
	firewallGroup, err := r.modelToAPIFirewallGroup(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Firewall Group for Update",
			fmt.Sprintf("Could not convert firewall group to API format: %s", err),
		)
		return
	}

	firewallGroup.ID = id
	firewallGroup.SiteID = site

	apiFirewallGroup, err := r.client.UpdateFirewallGroup(ctx, site, firewallGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Firewall Group",
			fmt.Sprintf("Could not update firewall group %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, apiFirewallGroup, &state, site)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallGroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state firewallGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeleteFirewallGroup(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Firewall Group",
			fmt.Sprintf("Could not delete firewall group %s: %s", id, err),
		)
		return
	}
}

func (r *firewallGroupResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts, diags := ParseImportID(req.ID, 1, 2)
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

func (r *firewallGroupResource) modelToAPIFirewallGroup(
	ctx context.Context,
	model *firewallGroupResourceModel,
) (*unifi.FirewallGroup, error) {
	var members []string
	if !model.Members.IsNull() && !model.Members.IsUnknown() {
		diags := model.Members.ElementsAs(ctx, &members, false)
		if diags.HasError() {
			return nil, fmt.Errorf("could not convert members to string slice")
		}
	}

	return &unifi.FirewallGroup{
		Name:         model.Name.ValueString(),
		GroupType:    model.Type.ValueString(),
		GroupMembers: members,
	}, nil
}

func (r *firewallGroupResource) setResourceData(
	ctx context.Context,
	firewallGroup *unifi.FirewallGroup,
	model *firewallGroupResourceModel,
	site string,
) {
	model.Site = types.StringValue(site)

	if firewallGroup.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(firewallGroup.Name)
	}

	if firewallGroup.GroupType == "" {
		model.Type = types.StringNull()
	} else {
		model.Type = types.StringValue(firewallGroup.GroupType)
	}

	if len(firewallGroup.GroupMembers) == 0 {
		model.Members = types.SetNull(types.StringType)
	} else {
		membersList := make([]types.String, len(firewallGroup.GroupMembers))
		for i, member := range firewallGroup.GroupMembers {
			membersList[i] = types.StringValue(member)
		}
		membersSet, _ := types.SetValueFrom(ctx, types.StringType, membersList)
		model.Members = membersSet
	}
}
