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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var (
	_ resource.Resource                = &firewallPolicyResource{}
	_ resource.ResourceWithImportState = &firewallPolicyResource{}
)

func NewFirewallPolicyResource() resource.Resource {
	return &firewallPolicyResource{}
}

type firewallPolicyResource struct {
	client *Client
}

// firewallPolicyModel is the Terraform resource model.
type firewallPolicyModel struct {
	ID                 types.String `tfsdk:"id"`
	Site               types.String `tfsdk:"site"`
	Name               types.String `tfsdk:"name"`
	Action             types.String `tfsdk:"action"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	Protocol           types.String `tfsdk:"protocol"`
	Description        types.String `tfsdk:"description"`
	Logging            types.Bool   `tfsdk:"logging"`
	Index              types.Int64  `tfsdk:"index"`
	CreateAllowRespond types.Bool   `tfsdk:"create_allow_respond"`
	IPVersion          types.String `tfsdk:"ip_version"`
	Source             types.Object `tfsdk:"source"`
	Destination        types.Object `tfsdk:"destination"`
}

// firewallPolicyEndpointModel is the nested source/destination block model.
type firewallPolicyEndpointModel struct {
	ZoneID           types.String `tfsdk:"zone_id"`
	MatchingTarget   types.String `tfsdk:"matching_target"`
	NetworkIDs       types.List   `tfsdk:"network_ids"`
	ClientMACs       types.List   `tfsdk:"client_macs"`
	IPs              types.List   `tfsdk:"ips"`
	PortGroupID      types.String `tfsdk:"port_group_id"`
	PortMatchingType types.String `tfsdk:"port_matching_type"`
}

func (m firewallPolicyEndpointModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"zone_id":            types.StringType,
		"matching_target":    types.StringType,
		"network_ids":        types.ListType{ElemType: types.StringType},
		"client_macs":        types.ListType{ElemType: types.StringType},
		"ips":                types.ListType{ElemType: types.StringType},
		"port_group_id":      types.StringType,
		"port_matching_type": types.StringType,
	}
}

func (r *firewallPolicyResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_policy"
}

func (r *firewallPolicyResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	endpointAttrs := map[string]schema.Attribute{
		"zone_id": schema.StringAttribute{
			MarkdownDescription: "The ID of the firewall zone this endpoint belongs to. Use the `unifi_firewall_zone` data source to look up zone IDs by name.",
			Required:            true,
		},
		"matching_target": schema.StringAttribute{
			MarkdownDescription: "What to match: `ANY`, `NETWORK`, `CLIENT`, `IP`, `DEVICE`, or `MAC`.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("ANY", "NETWORK", "CLIENT", "IP", "DEVICE", "MAC"),
			},
		},
		"network_ids": schema.ListAttribute{
			MarkdownDescription: "List of UniFi network IDs to match. Used when `matching_target` is `NETWORK`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
		},
		"client_macs": schema.ListAttribute{
			MarkdownDescription: "List of client MAC addresses to match. Used when `matching_target` is `CLIENT`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
		},
		"ips": schema.ListAttribute{
			MarkdownDescription: "List of IP addresses or CIDR ranges to match. Used when `matching_target` is `IP`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
		},
		"port_group_id": schema.StringAttribute{
			MarkdownDescription: "ID of a `unifi_firewall_group` (port-group type) to match. Used when `port_matching_type` is `OBJECT`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"port_matching_type": schema.StringAttribute{
			MarkdownDescription: "How to match ports: `ANY`, `SPECIFIC`, or `OBJECT` (port group).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("ANY"),
			Validators: []validator.String{
				stringvalidator.OneOf("ANY", "SPECIFIC", "OBJECT"),
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UniFi zone-based firewall policy (UniFi Network 8.x+). " +
			"Zone-based firewall policies replace the legacy firewall rules and are displayed " +
			"under Settings → Security → Firewall Policies in the UniFi UI.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall policy.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the UniFi site. Defaults to the site configured in the provider.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the firewall policy.",
				Required:            true,
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "The action to take when the policy matches: `ALLOW`, `BLOCK`, or `REJECT`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALLOW", "BLOCK", "REJECT"),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the policy is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol to match: `all`, `tcp`, `udp`, or `tcp_udp`. Defaults to `all`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("all"),
				Validators: []validator.String{
					stringvalidator.OneOf("all", "tcp", "udp", "tcp_udp"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description for the policy.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"logging": schema.BoolAttribute{
				MarkdownDescription: "Whether to log packets matching this policy. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"index": schema.Int64Attribute{
				MarkdownDescription: "The ordering index of the policy. UniFi auto-assigns this if not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"create_allow_respond": schema.BoolAttribute{
				MarkdownDescription: "When `true`, UniFi automatically creates a matching rule to allow established/related return traffic. Recommended for `ALLOW` policies. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ip_version": schema.StringAttribute{
				MarkdownDescription: "The IP version to match: `BOTH`, `IPV4`, or `IPV6`. Defaults to `IPV4`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("IPV4"),
				Validators: []validator.String{
					stringvalidator.OneOf("BOTH", "IPV4", "IPV6"),
				},
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "The source endpoint of the policy.",
				Required:            true,
				Attributes:          endpointAttrs,
			},
			"destination": schema.SingleNestedAttribute{
				MarkdownDescription: "The destination endpoint of the policy.",
				Required:            true,
				Attributes:          endpointAttrs,
			},
		},
	}
}

func (r *firewallPolicyResource) Configure(
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

func (r *firewallPolicyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan firewallPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	fp, diags := modelToFirewallPolicy(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateFirewallPolicy(ctx, site, fp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Firewall Policy",
			"Could not create firewall policy: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(firewallPolicyToModel(ctx, created, &plan)...)
	plan.Site = types.StringValue(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallPolicyResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state firewallPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	fp, err := r.client.GetFirewallPolicy(ctx, site, state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Firewall Policy",
			"Could not read firewall policy "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(firewallPolicyToModel(ctx, fp, &state)...)
	state.Site = types.StringValue(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallPolicyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan firewallPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state firewallPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	plan.ID = state.ID

	fp, diags := modelToFirewallPolicy(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateFirewallPolicy(ctx, site, fp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Firewall Policy",
			"Could not update firewall policy "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(firewallPolicyToModel(ctx, updated, &plan)...)
	plan.Site = types.StringValue(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallPolicyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state firewallPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeleteFirewallPolicy(ctx, site, state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); !ok {
			resp.Diagnostics.AddError(
				"Error Deleting Firewall Policy",
				"Could not delete firewall policy "+state.ID.ValueString()+": "+err.Error(),
			)
		}
	}
}

func (r *firewallPolicyResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func modelToFirewallPolicy(ctx context.Context, model firewallPolicyModel) (*unifi.FirewallPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics

	fp := &unifi.FirewallPolicy{
		ID:                 model.ID.ValueString(),
		Name:               model.Name.ValueString(),
		Action:             model.Action.ValueString(),
		Enabled:            model.Enabled.ValueBool(),
		Protocol:           model.Protocol.ValueString(),
		Description:        model.Description.ValueString(),
		Logging:            model.Logging.ValueBool(),
		CreateAllowRespond: model.CreateAllowRespond.ValueBool(),
		Version:            model.IPVersion.ValueString(),
		ConnectionStates:   []string{},
		Schedule: &unifi.FirewallPolicySchedule{
			Mode: "ALWAYS",
		},
	}

	if !model.Index.IsNull() && !model.Index.IsUnknown() {
		idx := model.Index.ValueInt64()
		fp.Index = &idx
	}

	var srcModel firewallPolicyEndpointModel
	diags.Append(model.Source.As(ctx, &srcModel, basetypes.ObjectAsOptions{})...)
	if !diags.HasError() {
		fp.Source = endpointModelToSource(ctx, srcModel, &diags)
	}

	var dstModel firewallPolicyEndpointModel
	diags.Append(model.Destination.As(ctx, &dstModel, basetypes.ObjectAsOptions{})...)
	if !diags.HasError() {
		fp.Destination = endpointModelToDestination(ctx, dstModel, &diags)
	}

	return fp, diags
}

func endpointModelToSource(ctx context.Context, m firewallPolicyEndpointModel, diags *diag.Diagnostics) *unifi.FirewallPolicySource {
	ep := &unifi.FirewallPolicySource{
		ZoneID:           m.ZoneID.ValueString(),
		MatchingTarget:   m.MatchingTarget.ValueString(),
		PortGroupID:      m.PortGroupID.ValueString(),
		PortMatchingType: m.PortMatchingType.ValueString(),
	}
	if !m.NetworkIDs.IsNull() && !m.NetworkIDs.IsUnknown() {
		diags.Append(m.NetworkIDs.ElementsAs(ctx, &ep.NetworkIDs, false)...)
	}
	if !m.ClientMACs.IsNull() && !m.ClientMACs.IsUnknown() {
		diags.Append(m.ClientMACs.ElementsAs(ctx, &ep.ClientMACs, false)...)
	}
	if !m.IPs.IsNull() && !m.IPs.IsUnknown() {
		diags.Append(m.IPs.ElementsAs(ctx, &ep.IPs, false)...)
	}
	return ep
}

func endpointModelToDestination(ctx context.Context, m firewallPolicyEndpointModel, diags *diag.Diagnostics) *unifi.FirewallPolicyDestination {
	ep := &unifi.FirewallPolicyDestination{
		ZoneID:           m.ZoneID.ValueString(),
		MatchingTarget:   m.MatchingTarget.ValueString(),
		PortGroupID:      m.PortGroupID.ValueString(),
		PortMatchingType: m.PortMatchingType.ValueString(),
	}
	if !m.NetworkIDs.IsNull() && !m.NetworkIDs.IsUnknown() {
		diags.Append(m.NetworkIDs.ElementsAs(ctx, &ep.NetworkIDs, false)...)
	}
	if !m.ClientMACs.IsNull() && !m.ClientMACs.IsUnknown() {
		diags.Append(m.ClientMACs.ElementsAs(ctx, &ep.ClientMACs, false)...)
	}
	if !m.IPs.IsNull() && !m.IPs.IsUnknown() {
		diags.Append(m.IPs.ElementsAs(ctx, &ep.IPs, false)...)
	}
	return ep
}

func firewallPolicyToModel(ctx context.Context, fp *unifi.FirewallPolicy, model *firewallPolicyModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(fp.ID)
	model.Name = types.StringValue(fp.Name)
	model.Action = types.StringValue(fp.Action)
	model.Enabled = types.BoolValue(fp.Enabled)
	model.Protocol = types.StringValue(fp.Protocol)
	model.Description = types.StringValue(fp.Description)
	model.Logging = types.BoolValue(fp.Logging)
	model.CreateAllowRespond = types.BoolValue(fp.CreateAllowRespond)
	model.IPVersion = types.StringValue(fp.Version)

	if fp.Index != nil {
		model.Index = types.Int64Value(*fp.Index)
	}

	if fp.Source != nil {
		srcModel := apiSourceToEndpointModel(ctx, fp.Source, &diags)
		srcObj, d := types.ObjectValueFrom(ctx, firewallPolicyEndpointModel{}.AttributeTypes(), srcModel)
		diags.Append(d...)
		model.Source = srcObj
	}

	if fp.Destination != nil {
		dstModel := apiDestinationToEndpointModel(ctx, fp.Destination, &diags)
		dstObj, d := types.ObjectValueFrom(ctx, firewallPolicyEndpointModel{}.AttributeTypes(), dstModel)
		diags.Append(d...)
		model.Destination = dstObj
	}

	return diags
}

func apiSourceToEndpointModel(ctx context.Context, src *unifi.FirewallPolicySource, diags *diag.Diagnostics) firewallPolicyEndpointModel {
	m := firewallPolicyEndpointModel{
		ZoneID:           types.StringValue(src.ZoneID),
		MatchingTarget:   types.StringValue(src.MatchingTarget),
		PortGroupID:      types.StringValue(src.PortGroupID),
		PortMatchingType: types.StringValue(src.PortMatchingType),
	}
	networkIDs, d := types.ListValueFrom(ctx, types.StringType, src.NetworkIDs)
	diags.Append(d...)
	m.NetworkIDs = networkIDs

	clientMACs, d := types.ListValueFrom(ctx, types.StringType, src.ClientMACs)
	diags.Append(d...)
	m.ClientMACs = clientMACs

	ips, d := types.ListValueFrom(ctx, types.StringType, src.IPs)
	diags.Append(d...)
	m.IPs = ips

	return m
}

func apiDestinationToEndpointModel(ctx context.Context, dst *unifi.FirewallPolicyDestination, diags *diag.Diagnostics) firewallPolicyEndpointModel {
	m := firewallPolicyEndpointModel{
		ZoneID:           types.StringValue(dst.ZoneID),
		MatchingTarget:   types.StringValue(dst.MatchingTarget),
		PortGroupID:      types.StringValue(dst.PortGroupID),
		PortMatchingType: types.StringValue(dst.PortMatchingType),
	}
	networkIDs, d := types.ListValueFrom(ctx, types.StringType, dst.NetworkIDs)
	diags.Append(d...)
	m.NetworkIDs = networkIDs

	clientMACs, d := types.ListValueFrom(ctx, types.StringType, dst.ClientMACs)
	diags.Append(d...)
	m.ClientMACs = clientMACs

	ips, d := types.ListValueFrom(ctx, types.StringType, dst.IPs)
	diags.Append(d...)
	m.IPs = ips

	return m
}
