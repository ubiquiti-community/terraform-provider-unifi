package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
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
	_ resource.ResourceWithIdentity    = &dynamicDNSResource{}
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

// dynamicDNSResourceIdentityModel describes the resource identity data model.
type dynamicDNSResourceIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
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
				MarkdownDescription: "The name of the site to associate with the dynamic DNS resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseNonNullStateForUnknown(),
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

func (r *dynamicDNSResource) IdentitySchema(
	ctx context.Context,
	req resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
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
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...); resp.Diagnostics.HasError() {
		return
	}

	// Get site from request identity if provided, otherwise use provider default
	site := r.client.Site
	if !data.Site.IsNull() && !data.Site.IsUnknown() {
		site = data.Site.ValueString()
	}

	// Convert to unifi.DynamicDNS
	dynamicDNS := r.modelToDynamicDNS(ctx, &data)

	// Create the dynamic DNS
	createdDynamicDNS, err := r.client.CreateDynamicDNS(ctx, site, dynamicDNS)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Dynamic DNS",
			err.Error(),
		)
		return
	}

	// Convert back to model
	r.dynamicDNSToModel(ctx, createdDynamicDNS, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Set identity
	identity := dynamicDNSResourceIdentityModel{
		ID:   types.StringValue(createdDynamicDNS.ID),
		Site: types.StringValue(site),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *dynamicDNSResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data dynamicDNSResourceModel
	var identity dynamicDNSResourceIdentityModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read identity
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := identity.ID.ValueString()
	site := identity.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the dynamic DNS from the API
	dynamicDNS, err := r.client.GetDynamicDNS(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Dynamic DNS",
			"Could not read dynamic DNS with ID "+id+": "+err.Error(),
		)
		return
	}

	// Convert to model
	r.dynamicDNSToModel(ctx, dynamicDNS, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// Re-set identity (should be unchanged)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *dynamicDNSResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state dynamicDNSResourceModel
	var plan dynamicDNSResourceModel
	var identity dynamicDNSResourceIdentityModel

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

	// Read identity
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)

	id := identity.ID.ValueString()
	site := identity.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the updated state to API format
	dynamicDNS := r.modelToDynamicDNS(ctx, &state)
	dynamicDNS.ID = id
	dynamicDNS.SiteID = site

	// Send to API
	updatedDynamicDNS, err := r.client.UpdateDynamicDNS(ctx, site, dynamicDNS)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Dynamic DNS",
			err.Error(),
		)
		return
	}

	// Update state with API response
	r.dynamicDNSToModel(ctx, updatedDynamicDNS, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	// Identity should not change during update
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *dynamicDNSResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data dynamicDNSResourceModel
	var identity dynamicDNSResourceIdentityModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read identity
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := identity.ID.ValueString()
	site := identity.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the dynamic DNS
	err := r.client.DeleteDynamicDNS(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Dynamic DNS",
			err.Error(),
		)
		return
	}
}

func (r *dynamicDNSResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughWithIdentity(
		ctx,
		path.Root("id"),
		path.Root("id"),
		req,
		resp,
	)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *dynamicDNSResource) applyPlanToState(
	_ context.Context,
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
	_ context.Context,
	model *dynamicDNSResourceModel,
) *unifi.DynamicDNS {
	dynamicDNS := &unifi.DynamicDNS{
		ID:        model.ID.ValueString(),
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
	_ context.Context,
	dynamicDNS *unifi.DynamicDNS,
	model *dynamicDNSResourceModel,
	site string,
) {
	model.ID = types.StringValue(dynamicDNS.ID)
	model.Interface = types.StringValue(dynamicDNS.Interface)
	model.Service = types.StringValue(dynamicDNS.Service)
	model.HostName = types.StringValue(dynamicDNS.HostName)

	if site != "" {
		model.Site = types.StringValue(site)
	} else {
		model.Site = types.StringNull()
	}

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
