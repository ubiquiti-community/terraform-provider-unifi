package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &dynamicDNSResource{}
	_ resource.ResourceWithImportState = &dynamicDNSResource{}
)

func NewDynamicDNSResource() resource.Resource {
	return &dynamicDNSResource{}
}

// dynamicDNSResource defines the resource implementation.
type dynamicDNSResource struct {
	client *Client
}

// dynamicDNSResourceModel describes the resource data model.
type dynamicDNSResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Site      types.String `tfsdk:"site"`
	Interface types.String `tfsdk:"interface"`
	Service   types.String `tfsdk:"service"`
	HostName  types.String `tfsdk:"host_name"`
	Server    types.String `tfsdk:"server"`
	Login     types.String `tfsdk:"login"`
	Password  types.String `tfsdk:"password"`
}

func (r *dynamicDNSResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_dynamic_dns"
}

func (r *dynamicDNSResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages dynamic DNS settings for different providers.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the dynamic DNS.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the dynamic DNS with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"interface": schema.StringAttribute{
				MarkdownDescription: "The interface for the dynamic DNS. Can be `wan` or `wan2`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("wan"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("wan", "wan2"),
				},
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "The Dynamic DNS service provider, various values are supported (for example `dyndns`, etc.).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host_name": schema.StringAttribute{
				MarkdownDescription: "The host name to update in the dynamic DNS service.",
				Required:            true,
			},
			"server": schema.StringAttribute{
				MarkdownDescription: "The server for the dynamic DNS service.",
				Optional:            true,
			},
			"login": schema.StringAttribute{
				MarkdownDescription: "The login for the dynamic DNS service.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the dynamic DNS service.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *dynamicDNSResource) Configure(
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

func (r *dynamicDNSResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data dynamicDNSResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.DynamicDNS
	dynamicDNS := r.modelToDynamicDNS(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the dynamic DNS
	createdDynamicDNS, err := r.client.Client.CreateDynamicDNS(ctx, site, dynamicDNS)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Dynamic DNS",
			"Could not create dynamic DNS, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert back to model
	r.dynamicDNSToModel(ctx, createdDynamicDNS, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dynamicDNSResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data dynamicDNSResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the dynamic DNS from the API
	dynamicDNS, err := r.client.Client.GetDynamicDNS(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Dynamic DNS",
			"Could not read dynamic DNS with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert to model
	r.dynamicDNSToModel(ctx, dynamicDNS, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dynamicDNSResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state dynamicDNSResourceModel
	var plan dynamicDNSResourceModel

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

	// Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the updated state to API format
	dynamicDNS := r.modelToDynamicDNS(ctx, &state)
	dynamicDNS.ID = state.ID.ValueString()
	dynamicDNS.SiteID = site

	// Send to API
	updatedDynamicDNS, err := r.client.Client.UpdateDynamicDNS(ctx, site, dynamicDNS)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Dynamic DNS",
			"Could not update dynamic DNS, unexpected error: "+err.Error(),
		)
		return
	}

	// Update state with API response
	r.dynamicDNSToModel(ctx, updatedDynamicDNS, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dynamicDNSResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data dynamicDNSResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the dynamic DNS
	err := r.client.Client.DeleteDynamicDNS(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Dynamic DNS",
			"Could not delete dynamic DNS, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *dynamicDNSResource) ImportState(
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

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *dynamicDNSResource) applyPlanToState(
	ctx context.Context,
	plan *dynamicDNSResourceModel,
	state *dynamicDNSResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.Interface.IsNull() && !plan.Interface.IsUnknown() {
		state.Interface = plan.Interface
	}
	if !plan.Service.IsNull() && !plan.Service.IsUnknown() {
		state.Service = plan.Service
	}
	if !plan.HostName.IsNull() && !plan.HostName.IsUnknown() {
		state.HostName = plan.HostName
	}
	if !plan.Server.IsNull() && !plan.Server.IsUnknown() {
		state.Server = plan.Server
	}
	if !plan.Login.IsNull() && !plan.Login.IsUnknown() {
		state.Login = plan.Login
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		state.Password = plan.Password
	}
}

// modelToDynamicDNS converts the Terraform model to the API struct.
func (r *dynamicDNSResource) modelToDynamicDNS(
	ctx context.Context,
	model *dynamicDNSResourceModel,
) *unifi.DynamicDNS {
	dynamicDNS := &unifi.DynamicDNS{
		Interface: model.Interface.ValueString(),
		Service:   model.Service.ValueString(),
		HostName:  model.HostName.ValueString(),
	}

	if !model.Server.IsNull() {
		dynamicDNS.Server = model.Server.ValueString()
	}
	if !model.Login.IsNull() {
		dynamicDNS.Login = model.Login.ValueString()
	}
	if !model.Password.IsNull() {
		dynamicDNS.XPassword = model.Password.ValueString()
	}

	return dynamicDNS
}

// dynamicDNSToModel converts the API struct to the Terraform model.
func (r *dynamicDNSResource) dynamicDNSToModel(
	ctx context.Context,
	dynamicDNS *unifi.DynamicDNS,
	model *dynamicDNSResourceModel,
	site string,
) {
	model.ID = types.StringValue(dynamicDNS.ID)
	model.Site = types.StringValue(site)
	model.Interface = types.StringValue(dynamicDNS.Interface)
	model.Service = types.StringValue(dynamicDNS.Service)
	model.HostName = types.StringValue(dynamicDNS.HostName)

	if dynamicDNS.Server != "" {
		model.Server = types.StringValue(dynamicDNS.Server)
	} else {
		model.Server = types.StringNull()
	}

	if dynamicDNS.Login != "" {
		model.Login = types.StringValue(dynamicDNS.Login)
	} else {
		model.Login = types.StringNull()
	}

	if dynamicDNS.XPassword != "" {
		model.Password = types.StringValue(dynamicDNS.XPassword)
	} else {
		model.Password = types.StringNull()
	}
}
