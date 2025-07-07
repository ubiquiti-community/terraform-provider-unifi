package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
)

var (
	_ resource.Resource                = &firewallRuleResource{}
	_ resource.ResourceWithImportState = &firewallRuleResource{}
)

func NewFirewallRuleResource() resource.Resource {
	return &firewallRuleResource{}
}

type firewallRuleResource struct {
	client *Client
}

type firewallRuleResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Site                types.String `tfsdk:"site"`
	Name                types.String `tfsdk:"name"`
	Action              types.String `tfsdk:"action"`
	Ruleset             types.String `tfsdk:"ruleset"`
	RuleIndex           types.Int64  `tfsdk:"rule_index"`
	Protocol            types.String `tfsdk:"protocol"`
	ProtocolV6          types.String `tfsdk:"protocol_v6"`
	ICMPTypename        types.String `tfsdk:"icmp_typename"`
	ICMPV6Typename      types.String `tfsdk:"icmp_v6_typename"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	SrcNetworkID        types.String `tfsdk:"src_network_id"`
	SrcNetworkType      types.String `tfsdk:"src_network_type"`
	SrcFirewallGroupIDs types.Set    `tfsdk:"src_firewall_group_ids"`
	SrcAddress          types.String `tfsdk:"src_address"`
	SrcAddressIPv6      types.String `tfsdk:"src_address_ipv6"`
	SrcPort             types.String `tfsdk:"src_port"`
	SrcMac              types.String `tfsdk:"src_mac"`
	DstNetworkID        types.String `tfsdk:"dst_network_id"`
	DstNetworkType      types.String `tfsdk:"dst_network_type"`
	DstFirewallGroupIDs types.Set    `tfsdk:"dst_firewall_group_ids"`
	DstAddress          types.String `tfsdk:"dst_address"`
	DstAddressIPv6      types.String `tfsdk:"dst_address_ipv6"`
	DstPort             types.String `tfsdk:"dst_port"`
	Logging             types.Bool   `tfsdk:"logging"`
	StateEstablished    types.Bool   `tfsdk:"state_established"`
	StateInvalid        types.Bool   `tfsdk:"state_invalid"`
	StateNew            types.Bool   `tfsdk:"state_new"`
	StateRelated        types.Bool   `tfsdk:"state_related"`
	IPSec               types.String `tfsdk:"ip_sec"`
}

func (r *firewallRuleResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule"
}

func (r *firewallRuleResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an individual firewall rule on the gateway.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the firewall rule with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the firewall rule.",
				Required:            true,
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "The action of the firewall rule. Must be one of `drop`, `accept`, or `reject`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("drop", "accept", "reject"),
				},
			},
			"ruleset": schema.StringAttribute{
				MarkdownDescription: "The ruleset for the rule. This is from the perspective of the security gateway.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"WAN_IN", "WAN_OUT", "WAN_LOCAL",
						"LAN_IN", "LAN_OUT", "LAN_LOCAL",
						"GUEST_IN", "GUEST_OUT", "GUEST_LOCAL",
						"WANv6_IN", "WANv6_OUT", "WANv6_LOCAL",
						"LANv6_IN", "LANv6_OUT", "LANv6_LOCAL",
						"GUESTv6_IN", "GUESTv6_OUT", "GUESTv6_LOCAL",
					),
				},
			},
			"rule_index": schema.Int64Attribute{
				MarkdownDescription: "The index of the rule. Must be >= 2000 < 3000 or >= 4000 < 5000.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Any(
						int64validator.Between(2000, 2999),
						int64validator.Between(4000, 4999),
					),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol of the rule.",
				Optional:            true,
			},
			"protocol_v6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 protocol of the rule.",
				Optional:            true,
			},
			"icmp_typename": schema.StringAttribute{
				MarkdownDescription: "ICMP type name.",
				Optional:            true,
			},
			"icmp_v6_typename": schema.StringAttribute{
				MarkdownDescription: "ICMPv6 type name.",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the rule should be enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"src_network_id": schema.StringAttribute{
				MarkdownDescription: "The source network ID for the firewall rule.",
				Optional:            true,
			},
			"src_network_type": schema.StringAttribute{
				MarkdownDescription: "The source network type of the firewall rule. Can be one of `ADDRv4` or `NETv4`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("NETv4"),
				Validators: []validator.String{
					stringvalidator.OneOf("ADDRv4", "NETv4"),
				},
			},
			"src_firewall_group_ids": schema.SetAttribute{
				MarkdownDescription: "The source firewall group IDs for the firewall rule.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"src_address": schema.StringAttribute{
				MarkdownDescription: "The source address for the firewall rule.",
				Optional:            true,
			},
			"src_address_ipv6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 source address for the firewall rule.",
				Optional:            true,
			},
			"src_port": schema.StringAttribute{
				MarkdownDescription: "The source port of the firewall rule.",
				Optional:            true,
			},
			"src_mac": schema.StringAttribute{
				MarkdownDescription: "The source MAC address of the firewall rule.",
				Optional:            true,
			},
			"dst_network_id": schema.StringAttribute{
				MarkdownDescription: "The destination network ID of the firewall rule.",
				Optional:            true,
			},
			"dst_network_type": schema.StringAttribute{
				MarkdownDescription: "The destination network type of the firewall rule. Can be one of `ADDRv4` or `NETv4`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("NETv4"),
				Validators: []validator.String{
					stringvalidator.OneOf("ADDRv4", "NETv4"),
				},
			},
			"dst_firewall_group_ids": schema.SetAttribute{
				MarkdownDescription: "The destination firewall group IDs of the firewall rule.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"dst_address": schema.StringAttribute{
				MarkdownDescription: "The destination address of the firewall rule.",
				Optional:            true,
			},
			"dst_address_ipv6": schema.StringAttribute{
				MarkdownDescription: "The IPv6 destination address of the firewall rule.",
				Optional:            true,
			},
			"dst_port": schema.StringAttribute{
				MarkdownDescription: "The destination port of the firewall rule.",
				Optional:            true,
			},
			"logging": schema.BoolAttribute{
				MarkdownDescription: "Enable logging for the firewall rule.",
				Optional:            true,
			},
			"state_established": schema.BoolAttribute{
				MarkdownDescription: "Match where the state is established.",
				Optional:            true,
			},
			"state_invalid": schema.BoolAttribute{
				MarkdownDescription: "Match where the state is invalid.",
				Optional:            true,
			},
			"state_new": schema.BoolAttribute{
				MarkdownDescription: "Match where the state is new.",
				Optional:            true,
			},
			"state_related": schema.BoolAttribute{
				MarkdownDescription: "Match where the state is related.",
				Optional:            true,
			},
			"ip_sec": schema.StringAttribute{
				MarkdownDescription: "Specify whether the rule matches on IPsec packets. Can be one of `match-ipset` or `match-none`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("match-ipsec", "match-none"),
				},
			},
		},
	}
}

func (r *firewallRuleResource) Configure(
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

func (r *firewallRuleResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data firewallRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallRule := r.modelToFirewallRule(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdFirewallRule, err := r.client.Client.CreateFirewallRule(ctx, site, firewallRule)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Firewall Rule",
			"Could not create firewall rule, unexpected error: "+err.Error(),
		)
		return
	}

	r.firewallRuleToModel(ctx, createdFirewallRule, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallRuleResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data firewallRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	firewallRule, err := r.client.Client.GetFirewallRule(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Firewall Rule",
			"Could not read firewall rule with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.firewallRuleToModel(ctx, firewallRule, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallRuleResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state firewallRuleResourceModel
	var plan firewallRuleResourceModel

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

	firewallRule := r.modelToFirewallRule(ctx, &state)
	firewallRule.ID = state.ID.ValueString()

	updatedFirewallRule, err := r.client.Client.UpdateFirewallRule(ctx, site, firewallRule)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Firewall Rule",
			"Could not update firewall rule, unexpected error: "+err.Error(),
		)
		return
	}

	r.firewallRuleToModel(ctx, updatedFirewallRule, &state, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallRuleResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data firewallRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.Client.DeleteFirewallRule(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Firewall Rule",
			"Could not delete firewall rule, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *firewallRuleResource) ImportState(
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

func (r *firewallRuleResource) applyPlanToState(
	ctx context.Context,
	plan *firewallRuleResourceModel,
	state *firewallRuleResourceModel,
) {
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Action.IsNull() && !plan.Action.IsUnknown() {
		state.Action = plan.Action
	}
	if !plan.Ruleset.IsNull() && !plan.Ruleset.IsUnknown() {
		state.Ruleset = plan.Ruleset
	}
	if !plan.RuleIndex.IsNull() && !plan.RuleIndex.IsUnknown() {
		state.RuleIndex = plan.RuleIndex
	}
	if !plan.Protocol.IsNull() && !plan.Protocol.IsUnknown() {
		state.Protocol = plan.Protocol
	}
	if !plan.ProtocolV6.IsNull() && !plan.ProtocolV6.IsUnknown() {
		state.ProtocolV6 = plan.ProtocolV6
	}
	if !plan.ICMPTypename.IsNull() && !plan.ICMPTypename.IsUnknown() {
		state.ICMPTypename = plan.ICMPTypename
	}
	if !plan.ICMPV6Typename.IsNull() && !plan.ICMPV6Typename.IsUnknown() {
		state.ICMPV6Typename = plan.ICMPV6Typename
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.SrcNetworkID.IsNull() && !plan.SrcNetworkID.IsUnknown() {
		state.SrcNetworkID = plan.SrcNetworkID
	}
	if !plan.SrcNetworkType.IsNull() && !plan.SrcNetworkType.IsUnknown() {
		state.SrcNetworkType = plan.SrcNetworkType
	}
	if !plan.SrcFirewallGroupIDs.IsNull() && !plan.SrcFirewallGroupIDs.IsUnknown() {
		state.SrcFirewallGroupIDs = plan.SrcFirewallGroupIDs
	}
	if !plan.SrcAddress.IsNull() && !plan.SrcAddress.IsUnknown() {
		state.SrcAddress = plan.SrcAddress
	}
	if !plan.SrcAddressIPv6.IsNull() && !plan.SrcAddressIPv6.IsUnknown() {
		state.SrcAddressIPv6 = plan.SrcAddressIPv6
	}
	if !plan.SrcPort.IsNull() && !plan.SrcPort.IsUnknown() {
		state.SrcPort = plan.SrcPort
	}
	if !plan.SrcMac.IsNull() && !plan.SrcMac.IsUnknown() {
		state.SrcMac = plan.SrcMac
	}
	if !plan.DstNetworkID.IsNull() && !plan.DstNetworkID.IsUnknown() {
		state.DstNetworkID = plan.DstNetworkID
	}
	if !plan.DstNetworkType.IsNull() && !plan.DstNetworkType.IsUnknown() {
		state.DstNetworkType = plan.DstNetworkType
	}
	if !plan.DstFirewallGroupIDs.IsNull() && !plan.DstFirewallGroupIDs.IsUnknown() {
		state.DstFirewallGroupIDs = plan.DstFirewallGroupIDs
	}
	if !plan.DstAddress.IsNull() && !plan.DstAddress.IsUnknown() {
		state.DstAddress = plan.DstAddress
	}
	if !plan.DstAddressIPv6.IsNull() && !plan.DstAddressIPv6.IsUnknown() {
		state.DstAddressIPv6 = plan.DstAddressIPv6
	}
	if !plan.DstPort.IsNull() && !plan.DstPort.IsUnknown() {
		state.DstPort = plan.DstPort
	}
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		state.Logging = plan.Logging
	}
	if !plan.StateEstablished.IsNull() && !plan.StateEstablished.IsUnknown() {
		state.StateEstablished = plan.StateEstablished
	}
	if !plan.StateInvalid.IsNull() && !plan.StateInvalid.IsUnknown() {
		state.StateInvalid = plan.StateInvalid
	}
	if !plan.StateNew.IsNull() && !plan.StateNew.IsUnknown() {
		state.StateNew = plan.StateNew
	}
	if !plan.StateRelated.IsNull() && !plan.StateRelated.IsUnknown() {
		state.StateRelated = plan.StateRelated
	}
	if !plan.IPSec.IsNull() && !plan.IPSec.IsUnknown() {
		state.IPSec = plan.IPSec
	}
}

func (r *firewallRuleResource) modelToFirewallRule(
	ctx context.Context,
	model *firewallRuleResourceModel,
) *unifi.FirewallRule {
	firewallRule := &unifi.FirewallRule{
		Name:      model.Name.ValueString(),
		Action:    model.Action.ValueString(),
		Ruleset:   model.Ruleset.ValueString(),
		RuleIndex: int(model.RuleIndex.ValueInt64()),
		Enabled:   model.Enabled.ValueBool(),
	}

	if !model.Protocol.IsNull() {
		firewallRule.Protocol = model.Protocol.ValueString()
	}
	if !model.ProtocolV6.IsNull() {
		firewallRule.ProtocolV6 = model.ProtocolV6.ValueString()
	}
	if !model.ICMPTypename.IsNull() {
		firewallRule.ICMPTypename = model.ICMPTypename.ValueString()
	}
	if !model.ICMPV6Typename.IsNull() {
		firewallRule.ICMPv6Typename = model.ICMPV6Typename.ValueString()
	}

	if !model.SrcNetworkID.IsNull() {
		firewallRule.SrcNetworkID = model.SrcNetworkID.ValueString()
	}
	if !model.SrcNetworkType.IsNull() {
		firewallRule.SrcNetworkType = model.SrcNetworkType.ValueString()
	}
	if !model.SrcFirewallGroupIDs.IsNull() {
		var groupIDs []string
		model.SrcFirewallGroupIDs.ElementsAs(ctx, &groupIDs, false)
		firewallRule.SrcFirewallGroupIDs = groupIDs
	}
	if !model.SrcAddress.IsNull() {
		firewallRule.SrcAddress = model.SrcAddress.ValueString()
	}
	if !model.SrcAddressIPv6.IsNull() {
		firewallRule.SrcAddressIPV6 = model.SrcAddressIPv6.ValueString()
	}
	if !model.SrcPort.IsNull() {
		firewallRule.SrcPort = model.SrcPort.ValueString()
	}

	if !model.DstNetworkID.IsNull() {
		firewallRule.DstNetworkID = model.DstNetworkID.ValueString()
	}
	if !model.DstNetworkType.IsNull() {
		firewallRule.DstNetworkType = model.DstNetworkType.ValueString()
	}
	if !model.DstFirewallGroupIDs.IsNull() {
		var groupIDs []string
		model.DstFirewallGroupIDs.ElementsAs(ctx, &groupIDs, false)
		firewallRule.DstFirewallGroupIDs = groupIDs
	}
	if !model.DstAddress.IsNull() {
		firewallRule.DstAddress = model.DstAddress.ValueString()
	}
	if !model.DstAddressIPv6.IsNull() {
		firewallRule.DstAddressIPV6 = model.DstAddressIPv6.ValueString()
	}
	if !model.DstPort.IsNull() {
		firewallRule.DstPort = model.DstPort.ValueString()
	}

	if !model.Logging.IsNull() {
		firewallRule.Logging = model.Logging.ValueBool()
	}
	if !model.StateEstablished.IsNull() {
		firewallRule.StateEstablished = model.StateEstablished.ValueBool()
	}
	if !model.StateInvalid.IsNull() {
		firewallRule.StateInvalid = model.StateInvalid.ValueBool()
	}
	if !model.StateNew.IsNull() {
		firewallRule.StateNew = model.StateNew.ValueBool()
	}
	if !model.StateRelated.IsNull() {
		firewallRule.StateRelated = model.StateRelated.ValueBool()
	}
	if !model.IPSec.IsNull() {
		firewallRule.IPSec = model.IPSec.ValueString()
	}

	return firewallRule
}

func (r *firewallRuleResource) firewallRuleToModel(
	ctx context.Context,
	firewallRule *unifi.FirewallRule,
	model *firewallRuleResourceModel,
	site string,
) {
	model.ID = types.StringValue(firewallRule.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(firewallRule.Name)
	model.Action = types.StringValue(firewallRule.Action)
	model.Ruleset = types.StringValue(firewallRule.Ruleset)
	model.RuleIndex = types.Int64Value(int64(firewallRule.RuleIndex))
	model.Enabled = types.BoolValue(firewallRule.Enabled)

	if firewallRule.Protocol != "" {
		model.Protocol = types.StringValue(firewallRule.Protocol)
	} else {
		model.Protocol = types.StringNull()
	}

	if firewallRule.ProtocolV6 != "" {
		model.ProtocolV6 = types.StringValue(firewallRule.ProtocolV6)
	} else {
		model.ProtocolV6 = types.StringNull()
	}

	if firewallRule.ICMPTypename != "" {
		model.ICMPTypename = types.StringValue(firewallRule.ICMPTypename)
	} else {
		model.ICMPTypename = types.StringNull()
	}

	if firewallRule.ICMPv6Typename != "" {
		model.ICMPV6Typename = types.StringValue(firewallRule.ICMPv6Typename)
	} else {
		model.ICMPV6Typename = types.StringNull()
	}

	if firewallRule.SrcNetworkID != "" {
		model.SrcNetworkID = types.StringValue(firewallRule.SrcNetworkID)
	} else {
		model.SrcNetworkID = types.StringNull()
	}

	if firewallRule.SrcNetworkType != "" {
		model.SrcNetworkType = types.StringValue(firewallRule.SrcNetworkType)
	} else {
		model.SrcNetworkType = types.StringValue("NETv4")
	}

	if len(firewallRule.SrcFirewallGroupIDs) > 0 {
		groupIDs, _ := types.SetValueFrom(ctx, types.StringType, firewallRule.SrcFirewallGroupIDs)
		model.SrcFirewallGroupIDs = groupIDs
	} else {
		model.SrcFirewallGroupIDs = types.SetNull(types.StringType)
	}

	if firewallRule.SrcAddress != "" {
		model.SrcAddress = types.StringValue(firewallRule.SrcAddress)
	} else {
		model.SrcAddress = types.StringNull()
	}

	if firewallRule.SrcAddressIPV6 != "" {
		model.SrcAddressIPv6 = types.StringValue(firewallRule.SrcAddressIPV6)
	} else {
		model.SrcAddressIPv6 = types.StringNull()
	}

	if firewallRule.SrcPort != "" {
		model.SrcPort = types.StringValue(firewallRule.SrcPort)
	} else {
		model.SrcPort = types.StringNull()
	}

	model.SrcMac = types.StringNull()

	if firewallRule.DstNetworkID != "" {
		model.DstNetworkID = types.StringValue(firewallRule.DstNetworkID)
	} else {
		model.DstNetworkID = types.StringNull()
	}

	if firewallRule.DstNetworkType != "" {
		model.DstNetworkType = types.StringValue(firewallRule.DstNetworkType)
	} else {
		model.DstNetworkType = types.StringValue("NETv4")
	}

	if len(firewallRule.DstFirewallGroupIDs) > 0 {
		groupIDs, _ := types.SetValueFrom(ctx, types.StringType, firewallRule.DstFirewallGroupIDs)
		model.DstFirewallGroupIDs = groupIDs
	} else {
		model.DstFirewallGroupIDs = types.SetNull(types.StringType)
	}

	if firewallRule.DstAddress != "" {
		model.DstAddress = types.StringValue(firewallRule.DstAddress)
	} else {
		model.DstAddress = types.StringNull()
	}

	if firewallRule.DstAddressIPV6 != "" {
		model.DstAddressIPv6 = types.StringValue(firewallRule.DstAddressIPV6)
	} else {
		model.DstAddressIPv6 = types.StringNull()
	}

	if firewallRule.DstPort != "" {
		model.DstPort = types.StringValue(firewallRule.DstPort)
	} else {
		model.DstPort = types.StringNull()
	}

	model.Logging = types.BoolValue(firewallRule.Logging)
	model.StateEstablished = types.BoolValue(firewallRule.StateEstablished)
	model.StateInvalid = types.BoolValue(firewallRule.StateInvalid)
	model.StateNew = types.BoolValue(firewallRule.StateNew)
	model.StateRelated = types.BoolValue(firewallRule.StateRelated)

	if firewallRule.IPSec != "" {
		model.IPSec = types.StringValue(firewallRule.IPSec)
	} else {
		model.IPSec = types.StringNull()
	}
}
