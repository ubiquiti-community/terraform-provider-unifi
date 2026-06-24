package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
	// Firmware-managed fields the controller requires back on every PUT. They are
	// not user-settable; the provider round-trips them so updates don't drop them
	// (an omitted connection_state_type/icmp_typename makes the PUT fail HTTP 400).
	ConnectionStateType types.String `tfsdk:"connection_state_type"`
	ConnectionStates    types.List   `tfsdk:"connection_states"`
	ICMPTypename        types.String `tfsdk:"icmp_typename"`
	ICMPV6Typename      types.String `tfsdk:"icmp_v6_typename"`
	Source              types.Object `tfsdk:"source"`
	Destination         types.Object `tfsdk:"destination"`
}

// firewallPolicyEndpointModel is the nested source/destination block model.
type firewallPolicyEndpointModel struct {
	ZoneID           types.String `tfsdk:"zone_id"`
	MatchingTarget   types.String `tfsdk:"matching_target"`
	NetworkIDs       types.List   `tfsdk:"network_ids"`
	ClientMACs       types.List   `tfsdk:"client_macs"`
	IPs              types.List   `tfsdk:"ips"`
	WebDomains       types.List   `tfsdk:"web_domains"`
	Port             types.Int64  `tfsdk:"port"`
	PortGroupID      types.String `tfsdk:"port_group_id"`
	PortMatchingType types.String `tfsdk:"port_matching_type"`
	// Firmware-managed; round-tripped so updates keep it (a PUT that omits
	// source/destination matching_target_type is rejected with HTTP 400).
	MatchingTargetType types.String `tfsdk:"matching_target_type"`
}

func (m firewallPolicyEndpointModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"zone_id":              types.StringType,
		"matching_target":      types.StringType,
		"network_ids":          types.ListType{ElemType: types.StringType},
		"client_macs":          types.ListType{ElemType: types.StringType},
		"ips":                  types.ListType{ElemType: types.StringType},
		"web_domains":          types.ListType{ElemType: types.StringType},
		"port":                 types.Int64Type,
		"port_group_id":        types.StringType,
		"port_matching_type":   types.StringType,
		"matching_target_type": types.StringType,
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
			MarkdownDescription: "What to match: `ANY`, `NETWORK`, `CLIENT`, `IP`, `DEVICE`, `MAC`, or `WEB` (domains/FQDN).",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("ANY", "NETWORK", "CLIENT", "IP", "DEVICE", "MAC", "WEB"),
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
		"web_domains": schema.ListAttribute{
			MarkdownDescription: "List of domains/FQDNs to match. Used when `matching_target` is `WEB`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
		},
		"port": schema.Int64Attribute{
			MarkdownDescription: "Specific port to match. Used when `port_matching_type` is `SPECIFIC`.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.Int64{
				int64validator.Between(1, 65535),
			},
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
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
		"matching_target_type": schema.StringAttribute{
			MarkdownDescription: "How the matching target is specified (`ANY`, `SPECIFIC`, `LIST`, `OBJECT`). Managed by the UniFi controller; the provider round-trips it so updates are accepted.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
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
			"connection_state_type": schema.StringAttribute{
				MarkdownDescription: "Connection-state matching mode (`ALL`, `RESPOND_ONLY`, or `CUSTOM`). Managed by the UniFi controller; the provider round-trips it so updates are accepted.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_states": schema.ListAttribute{
				MarkdownDescription: "Connection states matched when `connection_state_type` is `CUSTOM` (e.g. `NEW`, `ESTABLISHED`, `RELATED`, `INVALID`). Managed by the UniFi controller; the provider round-trips it so a `CUSTOM` policy's states are not dropped on update (which the firmware rejects with HTTP 400).",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"icmp_typename": schema.StringAttribute{
				MarkdownDescription: "ICMP type matching mode. Managed by the UniFi controller; the provider round-trips it so updates are accepted.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"icmp_v6_typename": schema.StringAttribute{
				MarkdownDescription: "ICMPv6 type matching mode. Managed by the UniFi controller; the provider round-trips it so updates are accepted.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

	// matching_target_type is firmware-derived: the controller may re-derive it
	// during the PUT (e.g. "" -> "SPECIFIC" when a SPECIFIC port is set), which
	// the provider cannot predict. It is Computed + UseStateForUnknown, so the
	// planned value is the prior state value; capture it now and re-assert it on
	// the post-apply state so Terraform's "inconsistent result after apply" check
	// passes (#324). The next Read reconciles state with the controller's value.
	plannedSrcMTT := endpointMatchingTargetType(ctx, plan.Source, &resp.Diagnostics)
	plannedDstMTT := endpointMatchingTargetType(ctx, plan.Destination, &resp.Diagnostics)

	updated, err := r.client.UpdateFirewallPolicy(ctx, site, fp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Firewall Policy",
			"Could not update firewall policy "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(firewallPolicyToModel(ctx, updated, &plan)...)

	// Only re-assert when the plan carried a known value: if it was unknown
	// (an in-block field changed), the attribute is known-after-apply and the
	// controller's value is accepted as-is.
	if !plannedSrcMTT.IsNull() && !plannedSrcMTT.IsUnknown() {
		plan.Source = withMatchingTargetType(ctx, plan.Source, plannedSrcMTT, &resp.Diagnostics)
	}
	if !plannedDstMTT.IsNull() && !plannedDstMTT.IsUnknown() {
		plan.Destination = withMatchingTargetType(ctx, plan.Destination, plannedDstMTT, &resp.Diagnostics)
	}

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

func modelToFirewallPolicy(
	ctx context.Context,
	model firewallPolicyModel,
) (*unifi.FirewallPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics

	fp := &unifi.FirewallPolicy{
		ID:                  model.ID.ValueString(),
		Name:                model.Name.ValueString(),
		Action:              model.Action.ValueString(),
		Enabled:             model.Enabled.ValueBool(),
		Protocol:            model.Protocol.ValueString(),
		Description:         model.Description.ValueString(),
		Logging:             model.Logging.ValueBool(),
		CreateAllowRespond:  model.CreateAllowRespond.ValueBool(),
		Version:             model.IPVersion.ValueString(),
		ConnectionStateType: model.ConnectionStateType.ValueString(),
		ICMPTypename:        model.ICMPTypename.ValueString(),
		ICMPV6Typename:      model.ICMPV6Typename.ValueString(),
		ConnectionStates:    []string{},
		Schedule: &unifi.FirewallPolicySchedule{
			Mode: "ALWAYS",
		},
	}

	// Round-trip the connection states (e.g. ["NEW"]) the controller reported.
	// Omitting them makes a CUSTOM-state policy's PUT fail with HTTP 400 (#227).
	if !model.ConnectionStates.IsNull() && !model.ConnectionStates.IsUnknown() {
		diags.Append(model.ConnectionStates.ElementsAs(ctx, &fp.ConnectionStates, false)...)
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

func endpointModelToSource(
	ctx context.Context,
	m firewallPolicyEndpointModel,
	diags *diag.Diagnostics,
) *unifi.FirewallPolicySource {
	ep := &unifi.FirewallPolicySource{
		ZoneID:             m.ZoneID.ValueString(),
		MatchingTarget:     m.MatchingTarget.ValueString(),
		MatchingTargetType: m.MatchingTargetType.ValueString(),
		Port:               m.Port.ValueInt64Pointer(),
		PortGroupID:        m.PortGroupID.ValueString(),
		PortMatchingType:   m.PortMatchingType.ValueString(),
	}
	if !m.IPs.IsNull() && !m.IPs.IsUnknown() {
		diags.Append(m.IPs.ElementsAs(ctx, &ep.IPs, false)...)
	}
	if !m.NetworkIDs.IsNull() && !m.NetworkIDs.IsUnknown() {
		diags.Append(m.NetworkIDs.ElementsAs(ctx, &ep.NetworkIDs, false)...)
	}
	if !m.ClientMACs.IsNull() && !m.ClientMACs.IsUnknown() {
		diags.Append(m.ClientMACs.ElementsAs(ctx, &ep.ClientMACs, false)...)
	}
	if !m.WebDomains.IsNull() && !m.WebDomains.IsUnknown() {
		diags.Append(m.WebDomains.ElementsAs(ctx, &ep.WebDomains, false)...)
	}
	return ep
}

func endpointModelToDestination(
	ctx context.Context,
	m firewallPolicyEndpointModel,
	diags *diag.Diagnostics,
) *unifi.FirewallPolicyDestination {
	ep := &unifi.FirewallPolicyDestination{
		ZoneID:             m.ZoneID.ValueString(),
		MatchingTarget:     m.MatchingTarget.ValueString(),
		MatchingTargetType: m.MatchingTargetType.ValueString(),
		Port:               m.Port.ValueInt64Pointer(),
		PortGroupID:        m.PortGroupID.ValueString(),
		PortMatchingType:   m.PortMatchingType.ValueString(),
	}
	if !m.IPs.IsNull() && !m.IPs.IsUnknown() {
		diags.Append(m.IPs.ElementsAs(ctx, &ep.IPs, false)...)
	}
	if !m.NetworkIDs.IsNull() && !m.NetworkIDs.IsUnknown() {
		diags.Append(m.NetworkIDs.ElementsAs(ctx, &ep.NetworkIDs, false)...)
	}
	if !m.ClientMACs.IsNull() && !m.ClientMACs.IsUnknown() {
		diags.Append(m.ClientMACs.ElementsAs(ctx, &ep.ClientMACs, false)...)
	}
	if !m.WebDomains.IsNull() && !m.WebDomains.IsUnknown() {
		diags.Append(m.WebDomains.ElementsAs(ctx, &ep.WebDomains, false)...)
	}
	return ep
}

// endpointMatchingTargetType extracts the matching_target_type out of a
// source/destination object, or a null string if the object is null/unknown.
func endpointMatchingTargetType(
	ctx context.Context,
	obj types.Object,
	diags *diag.Diagnostics,
) types.String {
	if obj.IsNull() || obj.IsUnknown() {
		return types.StringNull()
	}
	var m firewallPolicyEndpointModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	return m.MatchingTargetType
}

// withMatchingTargetType returns obj with its matching_target_type replaced by
// mtt, leaving every other attribute untouched.
func withMatchingTargetType(
	ctx context.Context,
	obj types.Object,
	mtt types.String,
	diags *diag.Diagnostics,
) types.Object {
	if obj.IsNull() || obj.IsUnknown() {
		return obj
	}
	var m firewallPolicyEndpointModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	m.MatchingTargetType = mtt
	newObj, d := types.ObjectValueFrom(
		ctx,
		firewallPolicyEndpointModel{}.AttributeTypes(),
		m,
	)
	diags.Append(d...)
	return newObj
}

func firewallPolicyToModel(
	ctx context.Context,
	fp *unifi.FirewallPolicy,
	model *firewallPolicyModel,
) diag.Diagnostics {
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
	model.ConnectionStateType = types.StringValue(fp.ConnectionStateType)
	connStates, csDiags := types.ListValueFrom(ctx, types.StringType, fp.ConnectionStates)
	diags.Append(csDiags...)
	model.ConnectionStates = connStates
	model.ICMPTypename = types.StringValue(fp.ICMPTypename)
	model.ICMPV6Typename = types.StringValue(fp.ICMPV6Typename)

	if fp.Index != nil {
		model.Index = types.Int64Value(*fp.Index)
	}

	if fp.Source != nil {
		srcModel := apiSourceToEndpointModel(ctx, fp.Source, &diags)
		srcObj, d := types.ObjectValueFrom(
			ctx,
			firewallPolicyEndpointModel{}.AttributeTypes(),
			srcModel,
		)
		diags.Append(d...)
		model.Source = srcObj
	}

	if fp.Destination != nil {
		dstModel := apiDestinationToEndpointModel(ctx, fp.Destination, &diags)
		dstObj, d := types.ObjectValueFrom(
			ctx,
			firewallPolicyEndpointModel{}.AttributeTypes(),
			dstModel,
		)
		diags.Append(d...)
		model.Destination = dstObj
	}

	return diags
}

func apiSourceToEndpointModel(
	ctx context.Context,
	src *unifi.FirewallPolicySource,
	diags *diag.Diagnostics,
) firewallPolicyEndpointModel {
	m := firewallPolicyEndpointModel{
		ZoneID:             types.StringValue(src.ZoneID),
		MatchingTarget:     types.StringValue(src.MatchingTarget),
		MatchingTargetType: types.StringValue(src.MatchingTargetType),
		Port:               types.Int64PointerValue(src.Port),
		PortGroupID:        types.StringValue(src.PortGroupID),
		PortMatchingType:   types.StringValue(src.PortMatchingType),
	}
	networkIDs, nd := types.ListValueFrom(ctx, types.StringType, src.NetworkIDs)
	diags.Append(nd...)
	m.NetworkIDs = networkIDs

	clientMACs, cd := types.ListValueFrom(ctx, types.StringType, src.ClientMACs)
	diags.Append(cd...)
	m.ClientMACs = clientMACs

	ips, d := types.ListValueFrom(ctx, types.StringType, src.IPs)
	diags.Append(d...)
	m.IPs = ips

	webDomains, wd := types.ListValueFrom(ctx, types.StringType, src.WebDomains)
	diags.Append(wd...)
	m.WebDomains = webDomains

	return m
}

func apiDestinationToEndpointModel(
	ctx context.Context,
	dst *unifi.FirewallPolicyDestination,
	diags *diag.Diagnostics,
) firewallPolicyEndpointModel {
	m := firewallPolicyEndpointModel{
		ZoneID:             types.StringValue(dst.ZoneID),
		MatchingTarget:     types.StringValue(dst.MatchingTarget),
		MatchingTargetType: types.StringValue(dst.MatchingTargetType),
		Port:               types.Int64PointerValue(dst.Port),
		PortGroupID:        types.StringValue(dst.PortGroupID),
		PortMatchingType:   types.StringValue(dst.PortMatchingType),
	}
	networkIDs, nd := types.ListValueFrom(ctx, types.StringType, dst.NetworkIDs)
	diags.Append(nd...)
	m.NetworkIDs = networkIDs

	clientMACs, cd := types.ListValueFrom(ctx, types.StringType, dst.ClientMACs)
	diags.Append(cd...)
	m.ClientMACs = clientMACs

	ips, d := types.ListValueFrom(ctx, types.StringType, dst.IPs)
	diags.Append(d...)
	m.IPs = ips

	webDomains, wd := types.ListValueFrom(ctx, types.StringType, dst.WebDomains)
	diags.Append(wd...)
	m.WebDomains = webDomains

	return m
}
