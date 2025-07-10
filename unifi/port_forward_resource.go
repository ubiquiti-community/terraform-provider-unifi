package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

var (
	_ resource.Resource                = &portForwardResource{}
	_ resource.ResourceWithImportState = &portForwardResource{}
)

func NewPortForwardResource() resource.Resource {
	return &portForwardResource{}
}

type portForwardResource struct {
	client *Client
}

type portForwardResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Site                 types.String `tfsdk:"site"`
	DstPort              types.String `tfsdk:"dst_port"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	FwdIP                types.String `tfsdk:"fwd_ip"`
	FwdPort              types.String `tfsdk:"fwd_port"`
	Log                  types.Bool   `tfsdk:"log"`
	Name                 types.String `tfsdk:"name"`
	PortForwardInterface types.String `tfsdk:"port_forward_interface"`
	Protocol             types.String `tfsdk:"protocol"`
	SrcIP                types.String `tfsdk:"src_ip"`
}

func (r *portForwardResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_port_forward"
}

func (r *portForwardResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a port forwarding rule on the gateway.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the port forwarding rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the port forwarding rule with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dst_port": schema.StringAttribute{
				MarkdownDescription: "The destination port for the forwarding.",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the port forwarding rule is enabled or not.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				DeprecationMessage:  "This attribute will be removed in a future release. Instead of disabling a port forwarding rule you can remove it from your configuration.",
			},
			"fwd_ip": schema.StringAttribute{
				MarkdownDescription: "The IPv4 address to forward traffic to.",
				Optional:            true,
				Validators: []validator.String{
					validators.IPv4Validator(),
				},
			},
			"fwd_port": schema.StringAttribute{
				MarkdownDescription: "The port to forward traffic to.",
				Optional:            true,
			},
			"log": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to log forwarded traffic or not.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the port forwarding rule.",
				Optional:            true,
			},
			"port_forward_interface": schema.StringAttribute{
				MarkdownDescription: "The port forwarding interface. Can be `wan`, `wan2`, or `both`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("wan", "wan2", "both"),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol for the port forwarding rule. Can be `tcp`, `udp`, or `tcp_udp`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("tcp_udp"),
				Validators: []validator.String{
					stringvalidator.OneOf("tcp_udp", "tcp", "udp"),
				},
			},
			"src_ip": schema.StringAttribute{
				MarkdownDescription: "The source IPv4 address (or CIDR) of the port forwarding rule. For all traffic, specify `any`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("any"),
			},
		},
	}
}

func (r *portForwardResource) Configure(
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

func (r *portForwardResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data portForwardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	portForward := r.modelToPortForward(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdPortForward, err := r.client.CreatePortForward(ctx, site, portForward)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Port Forward",
			"Could not create port forward, unexpected error: "+err.Error(),
		)
		return
	}

	r.portForwardToModel(ctx, createdPortForward, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *portForwardResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data portForwardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	portForward, err := r.client.GetPortForward(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Port Forward",
			"Could not read port forward with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.portForwardToModel(ctx, portForward, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *portForwardResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state portForwardResourceModel
	var plan portForwardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	portForward := r.modelToPortForward(ctx, &state)
	portForward.ID = state.ID.ValueString()

	updatedPortForward, err := r.client.UpdatePortForward(ctx, site, portForward)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Port Forward",
			"Could not update port forward, unexpected error: "+err.Error(),
		)
		return
	}

	r.portForwardToModel(ctx, updatedPortForward, &state, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portForwardResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data portForwardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeletePortForward(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Port Forward",
			"Could not delete port forward, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *portForwardResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) == 2 {
		site := idParts[0]
		id := idParts[1]

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	if len(idParts) == 1 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		return
	}

	resp.Diagnostics.AddError(
		"Invalid Import ID",
		"Import ID must be in format 'site:id' or 'id'",
	)
}

func (r *portForwardResource) applyPlanToState(
	_ context.Context,
	plan *portForwardResourceModel,
	state *portForwardResourceModel,
) {
	if !plan.DstPort.IsNull() && !plan.DstPort.IsUnknown() {
		state.DstPort = plan.DstPort
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.FwdIP.IsNull() && !plan.FwdIP.IsUnknown() {
		state.FwdIP = plan.FwdIP
	}
	if !plan.FwdPort.IsNull() && !plan.FwdPort.IsUnknown() {
		state.FwdPort = plan.FwdPort
	}
	if !plan.Log.IsNull() && !plan.Log.IsUnknown() {
		state.Log = plan.Log
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.PortForwardInterface.IsNull() && !plan.PortForwardInterface.IsUnknown() {
		state.PortForwardInterface = plan.PortForwardInterface
	}
	if !plan.Protocol.IsNull() && !plan.Protocol.IsUnknown() {
		state.Protocol = plan.Protocol
	}
	if !plan.SrcIP.IsNull() && !plan.SrcIP.IsUnknown() {
		state.SrcIP = plan.SrcIP
	}
}

func (r *portForwardResource) modelToPortForward(
	_ context.Context,
	model *portForwardResourceModel,
) *unifi.PortForward {
	portForward := &unifi.PortForward{
		Enabled: model.Enabled.ValueBool(),
		Log:     model.Log.ValueBool(),
		Proto:   model.Protocol.ValueString(),
		Src:     model.SrcIP.ValueString(),
	}

	if !model.DstPort.IsNull() {
		portForward.DstPort = model.DstPort.ValueString()
	}
	if !model.FwdIP.IsNull() {
		portForward.Fwd = model.FwdIP.ValueString()
	}
	if !model.FwdPort.IsNull() {
		portForward.FwdPort = model.FwdPort.ValueString()
	}
	if !model.Name.IsNull() {
		portForward.Name = model.Name.ValueString()
	}
	if !model.PortForwardInterface.IsNull() {
		portForward.PfwdInterface = model.PortForwardInterface.ValueString()
	}

	return portForward
}

func (r *portForwardResource) portForwardToModel(
	_ context.Context,
	portForward *unifi.PortForward,
	model *portForwardResourceModel,
	site string,
) {
	model.ID = types.StringValue(portForward.ID)
	model.Site = types.StringValue(site)
	model.Enabled = types.BoolValue(portForward.Enabled)
	model.Log = types.BoolValue(portForward.Log)
	model.Protocol = types.StringValue(portForward.Proto)
	model.SrcIP = types.StringValue(portForward.Src)

	if portForward.DstPort != "" {
		model.DstPort = types.StringValue(portForward.DstPort)
	} else {
		model.DstPort = types.StringNull()
	}

	if portForward.Fwd != "" {
		model.FwdIP = types.StringValue(portForward.Fwd)
	} else {
		model.FwdIP = types.StringNull()
	}

	if portForward.FwdPort != "" {
		model.FwdPort = types.StringValue(portForward.FwdPort)
	} else {
		model.FwdPort = types.StringNull()
	}

	if portForward.Name != "" {
		model.Name = types.StringValue(portForward.Name)
	} else {
		model.Name = types.StringNull()
	}

	if portForward.PfwdInterface != "" {
		model.PortForwardInterface = types.StringValue(portForward.PfwdInterface)
	} else {
		model.PortForwardInterface = types.StringNull()
	}
}
