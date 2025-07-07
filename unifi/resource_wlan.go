package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &wlanResource{}
var _ resource.ResourceWithImportState = &wlanResource{}

func NewWlanResource() resource.Resource {
	return &wlanResource{}
}

// wlanResource defines the resource implementation.
type wlanResource struct {
	client *frameworkClient
}

// wlanResourceModel describes the resource data model.
type wlanResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Site             types.String `tfsdk:"site"`
	Name             types.String `tfsdk:"name"`
	UserGroupID      types.String `tfsdk:"user_group_id"`
	Security         types.String `tfsdk:"security"`
	WPA3Support      types.Bool   `tfsdk:"wpa3_support"`
	WPA3Transition   types.Bool   `tfsdk:"wpa3_transition"`
	PMFMode          types.String `tfsdk:"pmf_mode"`
	Passphrase       types.String `tfsdk:"passphrase"`
	HideSSID         types.Bool   `tfsdk:"hide_ssid"`
	IsGuest          types.Bool   `tfsdk:"is_guest"`
	NetworkID        types.String `tfsdk:"network_id"`
	WLANGroupID      types.String `tfsdk:"wlan_group_id"`
	// Add more fields as needed
}

func (r *wlanResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wlan"
}

func (r *wlanResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "`unifi_wlan` manages a WiFi network / SSID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the wlan with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The SSID of the network.",
				Required:            true,
			},
			"user_group_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user group to use for this network.",
				Required:            true,
			},
			"security": schema.StringAttribute{
				MarkdownDescription: "The type of WiFi security for this network. Valid values are: `wpapsk`, `wpaeap`, and `open`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("wpapsk", "wpaeap", "open"),
				},
			},
			"wpa3_support": schema.BoolAttribute{
				MarkdownDescription: "Enable WPA 3 support (security must be `wpapsk` and PMF must be turned on).",
				Optional:            true,
			},
			"wpa3_transition": schema.BoolAttribute{
				MarkdownDescription: "Enable WPA 3 and WPA 2 support (security must be `wpapsk` and `wpa3_support` must be true).",
				Optional:            true,
			},
			"pmf_mode": schema.StringAttribute{
				MarkdownDescription: "Enable Protected Management Frames. This cannot be disabled if using WPA 3. Valid values are `required`, `optional` and `disabled`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				Validators: []validator.String{
					stringvalidator.OneOf("required", "optional", "disabled"),
				},
			},
			"passphrase": schema.StringAttribute{
				MarkdownDescription: "The passphrase for the network, this is only required if `security` is not set to `open`.",
				Optional:            true,
				Sensitive:           true,
			},
			"hide_ssid": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether or not to hide the SSID from broadcast.",
				Optional:            true,
			},
			"is_guest": schema.BoolAttribute{
				MarkdownDescription: "Indicates that this is a guest network.",
				Optional:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID of the network for this SSID.",
				Optional:            true,
			},
			"wlan_group_id": schema.StringAttribute{
				MarkdownDescription: "ID of the WLAN group to use for this network.",
				Optional:            true,
			},
		},
	}
}

func (r *wlanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*frameworkClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *frameworkClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *wlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data wlanResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement the actual API call to create the WLAN
	// For now, we'll just set a placeholder ID
	data.ID = types.StringValue("placeholder-id")

	// Example of how you would handle null vs empty values in Framework
	if data.Site.IsNull() {
		data.Site = types.StringValue("default")
	}

	// Write logs using the tflog package
	// tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data wlanResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement the actual API call to read the WLAN
	// If the resource no longer exists, call resp.State.RemoveResource(ctx)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data wlanResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement the actual API call to update the WLAN

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data wlanResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement the actual API call to delete the WLAN
}

func (r *wlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}