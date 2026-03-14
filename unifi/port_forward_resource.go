package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// portForwardWanModel describes the WAN configuration for a port forwarding rule.
type portForwardWanModel struct {
	Interface types.String `tfsdk:"interface"`
	IPAddress types.String `tfsdk:"ip_address"`
	Port      types.String `tfsdk:"port"`
}

func (m portForwardWanModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interface":  types.StringType,
		"ip_address": types.StringType,
		"port":       types.StringType,
	}
}

// portForwardForwardModel describes the forward destination for a port forwarding rule.
type portForwardForwardModel struct {
	IP   types.String `tfsdk:"ip"`
	Port types.String `tfsdk:"port"`
}

func (m portForwardForwardModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip":   types.StringType,
		"port": types.StringType,
	}
}

// portForwardSourceLimitingModel describes the source limiting configuration.
type portForwardSourceLimitingModel struct {
	IP              types.String `tfsdk:"ip"`
	FirewallGroupID types.String `tfsdk:"firewall_group_id"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Type            types.String `tfsdk:"type"`
}

func (m portForwardSourceLimitingModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip":                types.StringType,
		"firewall_group_id": types.StringType,
		"enabled":           types.BoolType,
		"type":              types.StringType,
	}
}

type portForwardResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	Name           types.String `tfsdk:"name"`
	Wan            types.Object `tfsdk:"wan"`
	Forward        types.Object `tfsdk:"forward"`
	SourceLimiting types.Object `tfsdk:"source_limiting"`
	Protocol       types.String `tfsdk:"protocol"`
	Logging        types.Bool   `tfsdk:"logging"`
	Enabled        types.Bool   `tfsdk:"enabled"`
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
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the port forwarding rule.",
				Optional:            true,
			},
			"wan": schema.SingleNestedAttribute{
				MarkdownDescription: "WAN configuration for the port forwarding rule.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"interface": schema.StringAttribute{
						MarkdownDescription: "The WAN interface. Can be `wan`, `wan2`, or `both`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("wan", "wan2", "both"),
						},
					},
					"ip_address": schema.StringAttribute{
						MarkdownDescription: "The WAN IP address for the port forwarding rule. Use `any` for all addresses.",
						Optional:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "The WAN port or port range (e.g. `1-10,11,12`).",
						Optional:            true,
					},
				},
			},
			"forward": schema.SingleNestedAttribute{
				MarkdownDescription: "Forward destination configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						MarkdownDescription: "The forward IPv4 address to send traffic to.",
						Optional:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "The forward port or port range (e.g. `1-10,11,12`).",
						Optional:            true,
					},
				},
			},
			"source_limiting": schema.SingleNestedAttribute{
				MarkdownDescription: "Source limiting configuration for the port forwarding rule.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						MarkdownDescription: "The source IPv4 address (or CIDR) of the port forwarding rule. For all traffic, specify `any`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("any"),
					},
					"firewall_group_id": schema.StringAttribute{
						MarkdownDescription: "The ID of the firewall group to use for source limiting.",
						Optional:            true,
					},
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether source limiting is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "The source limiting type. Can be `ip` or `firewall_group`. Inferred automatically when only one of `ip` or `firewall_group_id` is set.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("ip", "firewall_group"),
						},
					},
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
			"logging": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether to enable syslog logging for forwarded traffic.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the port forwarding rule is enabled or not.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				DeprecationMessage:  "This attribute will be removed in a future release. Instead of disabling a port forwarding rule you can remove it from your configuration.",
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

	portForward, diags := r.modelToPortForward(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdPortForward, err := r.client.CreatePortForward(ctx, site, portForward)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Port Forward",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.portForwardToModel(ctx, createdPortForward, &data, site)...)

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

	resp.Diagnostics.Append(r.portForwardToModel(ctx, portForward, &data, site)...)

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

	portForward, diags := r.modelToPortForward(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	portForward.ID = state.ID.ValueString()

	updatedPortForward, err := r.client.UpdatePortForward(ctx, site, portForward)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Port Forward",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.portForwardToModel(ctx, updatedPortForward, &state, site)...)

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
			err.Error(),
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
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Wan.IsNull() && !plan.Wan.IsUnknown() {
		state.Wan = plan.Wan
	}
	if !plan.Forward.IsNull() && !plan.Forward.IsUnknown() {
		state.Forward = plan.Forward
	}
	if !plan.SourceLimiting.IsNull() && !plan.SourceLimiting.IsUnknown() {
		state.SourceLimiting = plan.SourceLimiting
	}
	if !plan.Protocol.IsNull() && !plan.Protocol.IsUnknown() {
		state.Protocol = plan.Protocol
	}
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		state.Logging = plan.Logging
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
}

func (r *portForwardResource) modelToPortForward(
	ctx context.Context,
	model *portForwardResourceModel,
) (*unifi.PortForward, diag.Diagnostics) {
	var diags diag.Diagnostics

	portForward := &unifi.PortForward{
		Enabled: model.Enabled.ValueBool(),
		Log:     model.Logging.ValueBool(),
		Proto:   model.Protocol.ValueString(),
	}

	if !model.Name.IsNull() {
		portForward.Name = model.Name.ValueString()
	}

	if !model.Wan.IsNull() && !model.Wan.IsUnknown() {
		var wan portForwardWanModel
		diags.Append(model.Wan.As(ctx, &wan, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		if !wan.Interface.IsNull() {
			portForward.PfwdInterface = wan.Interface.ValueString()
		}
		if !wan.IPAddress.IsNull() {
			portForward.DestinationIP = wan.IPAddress.ValueString()
		}
		if !wan.Port.IsNull() {
			portForward.DstPort = wan.Port.ValueString()
		}
	}

	if !model.Forward.IsNull() && !model.Forward.IsUnknown() {
		var fwd portForwardForwardModel
		diags.Append(model.Forward.As(ctx, &fwd, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		if !fwd.IP.IsNull() {
			portForward.Fwd = fwd.IP.ValueString()
		}
		if !fwd.Port.IsNull() {
			portForward.FwdPort = fwd.Port.ValueString()
		}
	}

	if !model.SourceLimiting.IsNull() && !model.SourceLimiting.IsUnknown() {
		var src portForwardSourceLimitingModel
		diags.Append(model.SourceLimiting.As(ctx, &src, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		portForward.Src = src.IP.ValueString()
		portForward.SrcLimitingEnabled = src.Enabled.ValueBool()
		if !src.FirewallGroupID.IsNull() {
			portForward.SrcFirewallGroupID = src.FirewallGroupID.ValueString()
		}

		// Determine type: use explicit value if set, otherwise infer from which field is populated.
		switch {
		case !src.Type.IsNull() && !src.Type.IsUnknown():
			portForward.SrcLimitingType = src.Type.ValueString()
		case !src.FirewallGroupID.IsNull():
			portForward.SrcLimitingType = "firewall_group"
		default:
			portForward.SrcLimitingType = "ip"
		}
	}

	return portForward, diags
}

func (r *portForwardResource) portForwardToModel(
	ctx context.Context,
	portForward *unifi.PortForward,
	model *portForwardResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(portForward.ID)
	model.Site = types.StringValue(site)
	model.Enabled = types.BoolValue(portForward.Enabled)
	model.Logging = types.BoolValue(portForward.Log)
	model.Protocol = types.StringValue(portForward.Proto)

	if portForward.Name != "" {
		model.Name = types.StringValue(portForward.Name)
	} else {
		model.Name = types.StringNull()
	}

	// WAN nested object
	if portForward.PfwdInterface != "" || portForward.DestinationIP != "" ||
		portForward.DstPort != "" {
		wanValue := portForwardWanModel{
			Interface: stringValueOrNull(portForward.PfwdInterface),
			IPAddress: stringValueOrNull(portForward.DestinationIP),
			Port:      stringValueOrNull(portForward.DstPort),
		}
		wanObj, d := types.ObjectValueFrom(ctx, wanValue.AttributeTypes(), wanValue)
		diags.Append(d...)
		model.Wan = wanObj
	} else if !model.Wan.IsNull() {
		// Preserve non-null state with null fields
		wanValue := portForwardWanModel{
			Interface: types.StringNull(),
			IPAddress: types.StringNull(),
			Port:      types.StringNull(),
		}
		wanObj, d := types.ObjectValueFrom(ctx, wanValue.AttributeTypes(), wanValue)
		diags.Append(d...)
		model.Wan = wanObj
	} else {
		model.Wan = types.ObjectNull(portForwardWanModel{}.AttributeTypes())
	}

	// Forward nested object
	if portForward.Fwd != "" || portForward.FwdPort != "" {
		fwdValue := portForwardForwardModel{
			IP:   stringValueOrNull(portForward.Fwd),
			Port: stringValueOrNull(portForward.FwdPort),
		}
		fwdObj, d := types.ObjectValueFrom(ctx, fwdValue.AttributeTypes(), fwdValue)
		diags.Append(d...)
		model.Forward = fwdObj
	} else if !model.Forward.IsNull() {
		fwdValue := portForwardForwardModel{
			IP:   types.StringNull(),
			Port: types.StringNull(),
		}
		fwdObj, d := types.ObjectValueFrom(ctx, fwdValue.AttributeTypes(), fwdValue)
		diags.Append(d...)
		model.Forward = fwdObj
	} else {
		model.Forward = types.ObjectNull(portForwardForwardModel{}.AttributeTypes())
	}

	// Source limiting nested object
	if portForward.Src != "" || portForward.SrcLimitingEnabled ||
		portForward.SrcFirewallGroupID != "" ||
		portForward.SrcLimitingType != "" {
		srcValue := portForwardSourceLimitingModel{
			IP:              stringValueOrNull(portForward.Src),
			FirewallGroupID: stringValueOrNull(portForward.SrcFirewallGroupID),
			Enabled:         types.BoolValue(portForward.SrcLimitingEnabled),
			Type:            stringValueOrNull(portForward.SrcLimitingType),
		}
		srcObj, d := types.ObjectValueFrom(ctx, srcValue.AttributeTypes(), srcValue)
		diags.Append(d...)
		model.SourceLimiting = srcObj
	} else if !model.SourceLimiting.IsNull() {
		srcValue := portForwardSourceLimitingModel{
			IP:              types.StringNull(),
			FirewallGroupID: types.StringNull(),
			Enabled:         types.BoolValue(false),
			Type:            types.StringNull(),
		}
		srcObj, d := types.ObjectValueFrom(ctx, srcValue.AttributeTypes(), srcValue)
		diags.Append(d...)
		model.SourceLimiting = srcObj
	} else {
		model.SourceLimiting = types.ObjectNull(portForwardSourceLimitingModel{}.AttributeTypes())
	}

	return diags
}

func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
