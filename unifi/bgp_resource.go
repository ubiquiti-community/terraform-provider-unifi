package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &bgpResource{}
	_ resource.ResourceWithImportState = &bgpResource{}
)

func NewBGPResource() resource.Resource {
	return &bgpResource{}
}

// bgpResource defines the resource implementation.
type bgpResource struct {
	client *Client
}

// bgpResourceModel describes the resource data model.
type bgpResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Config         types.String `tfsdk:"config"`
	UploadFileName types.String `tfsdk:"upload_file_name"`
	Description    types.String `tfsdk:"description"`
}

func (r *bgpResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_bgp"
}

func (r *bgpResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages BGP configuration for the UniFi Controller.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the BGP configuration.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the BGP configuration with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable BGP routing.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "The FRRouting BGP daemon configuration.",
				Optional:            true,
			},
			"upload_file_name": schema.StringAttribute{
				MarkdownDescription: "The name of the uploaded configuration file.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("frr.conf"),
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the BGP configuration.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("BGP Configuration"),
			},
		},
	}
}

func (r *bgpResource) Configure(
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

func (r *bgpResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data bgpResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.BGPConfig
	bgpConfig := r.modelToBGP(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the BGP configuration
	createdBGPConfig, err := r.client.CreateBGPConfig(ctx, site, bgpConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating BGP Configuration",
			"Could not create BGP configuration, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert back to model
	r.bgpToModel(ctx, createdBGPConfig, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bgpResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data bgpResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the BGP configuration from the API
	bgpConfig, err := r.client.GetBGPConfig(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading BGP Configuration",
			"Could not read BGP configuration: "+err.Error(),
		)
		return
	}

	// Convert to model
	r.bgpToModel(ctx, bgpConfig, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bgpResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state bgpResourceModel
	var plan bgpResourceModel

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
	bgpConfig := r.modelToBGP(ctx, &state)
	bgpConfig.ID = state.ID.ValueString()

	// Send to API
	updatedBGPConfig, err := r.client.UpdateBGPConfig(ctx, site, bgpConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating BGP Configuration",
			"Could not update BGP configuration, unexpected error: "+err.Error(),
		)
		return
	}

	// Update state with API response
	r.bgpToModel(ctx, updatedBGPConfig, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bgpResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data bgpResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the BGP configuration
	err := r.client.DeleteBGPConfig(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting BGP Configuration",
			"Could not delete BGP configuration, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *bgpResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(
		ctx,
		path.Root("id"),
		req,
		resp,
	)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *bgpResource) applyPlanToState(
	_ context.Context,
	plan *bgpResourceModel,
	state *bgpResourceModel,
) {
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		state.Config = plan.Config
	}
	if !plan.UploadFileName.IsNull() && !plan.UploadFileName.IsUnknown() {
		state.UploadFileName = plan.UploadFileName
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		state.Description = plan.Description
	}
}

// modelToBGP converts the Terraform model to the API struct.
func (r *bgpResource) modelToBGP(
	_ context.Context,
	model *bgpResourceModel,
) *unifi.BGPConfig {
	bgpConfig := &unifi.BGPConfig{
		Enabled:        model.Enabled.ValueBool(),
		Config:         model.Config.ValueString(),
		UploadFileName: model.UploadFileName.ValueString(),
		Description:    model.Description.ValueString(),
	}

	return bgpConfig
}

// bgpToModel converts the API struct to the Terraform model.
func (r *bgpResource) bgpToModel(
	_ context.Context,
	bgpConfig *unifi.BGPConfig,
	model *bgpResourceModel,
	site string,
) {
	model.ID = types.StringValue(bgpConfig.ID)
	model.Site = types.StringValue(site)
	model.Enabled = types.BoolValue(bgpConfig.Enabled)

	if bgpConfig.Config != "" {
		model.Config = types.StringValue(bgpConfig.Config)
	} else {
		model.Config = types.StringNull()
	}

	if bgpConfig.UploadFileName != "" {
		model.UploadFileName = types.StringValue(bgpConfig.UploadFileName)
	} else {
		model.UploadFileName = types.StringNull()
	}

	if bgpConfig.Description != "" {
		model.Description = types.StringValue(bgpConfig.Description)
	} else {
		model.Description = types.StringNull()
	}
}
