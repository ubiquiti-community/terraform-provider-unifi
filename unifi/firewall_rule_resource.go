package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var (
	_ resource.Resource                 = &firewallRuleResource{}
	_ resource.ResourceWithImportState  = &firewallRuleResource{}
	_ resource.ResourceWithIdentity     = &firewallRuleResource{}
	_ resource.ResourceWithUpgradeState = &firewallRuleResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &firewallRuleResource{}
	_ list.ListResourceWithConfigure = &firewallRuleResource{}
)

func NewFirewallRuleResource() resource.Resource {
	return &firewallRuleResource{}
}

func NewFirewallRuleListResource() list.ListResource {
	return &firewallRuleResource{}
}

type firewallRuleResource struct {
	client *Client
}

// firewallRuleIdentityModel describes the resource identity data model.
type firewallRuleIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

// firewallRuleListConfigModel describes the list configuration model.
type firewallRuleListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// firewallRuleListFilterModel represents a single name/value filter entry.
type firewallRuleListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type firewallRuleResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Site                types.String   `tfsdk:"site"`
	Name                types.String   `tfsdk:"name"`
	Action              types.String   `tfsdk:"action"`
	Ruleset             types.String   `tfsdk:"ruleset"`
	RuleIndex           types.Int64    `tfsdk:"rule_index"`
	Protocol            types.String   `tfsdk:"protocol"`
	ProtocolV6          types.String   `tfsdk:"protocol_v6"`
	ICMP                types.Object   `tfsdk:"icmp"`
	Enabled             types.Bool     `tfsdk:"enabled"`
	Source              types.Object   `tfsdk:"source"`
	Destination         types.Object   `tfsdk:"destination"`
	Logging             types.Bool     `tfsdk:"logging"`
	State               types.Object   `tfsdk:"state"`
	IPSec               types.String   `tfsdk:"ip_sec"`
	SettingPreference   types.String   `tfsdk:"setting_preference"`
	ProtocolMatchExcept types.Bool     `tfsdk:"protocol_match_excepted"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// firewallRuleICMPModel is the `icmp` nested object.
type firewallRuleICMPModel struct {
	Typename   types.String `tfsdk:"typename"`
	V6Typename types.String `tfsdk:"v6_typename"`
}

func firewallRuleICMPAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"typename":    types.StringType,
		"v6_typename": types.StringType,
	}
}

// firewallRuleSourceModel is the `source` nested object.
type firewallRuleSourceModel struct {
	NetworkID        types.String       `tfsdk:"network_id"`
	NetworkType      types.String       `tfsdk:"network_type"`
	FirewallGroupIDs types.Set          `tfsdk:"firewall_group_ids"`
	Address          types.String       `tfsdk:"address"`
	AddressIPv6      types.String       `tfsdk:"address_ipv6"`
	Port             types.String       `tfsdk:"port"`
	Mac              hwtypes.MACAddress `tfsdk:"mac"`
}

func firewallRuleSourceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"network_id":         types.StringType,
		"network_type":       types.StringType,
		"firewall_group_ids": types.SetType{ElemType: types.StringType},
		"address":            types.StringType,
		"address_ipv6":       types.StringType,
		"port":               types.StringType,
		"mac":                hwtypes.MACAddressType{},
	}
}

// firewallRuleDestinationModel is the `destination` nested object.
type firewallRuleDestinationModel struct {
	NetworkID        types.String `tfsdk:"network_id"`
	NetworkType      types.String `tfsdk:"network_type"`
	FirewallGroupIDs types.Set    `tfsdk:"firewall_group_ids"`
	Address          types.String `tfsdk:"address"`
	AddressIPv6      types.String `tfsdk:"address_ipv6"`
	Port             types.String `tfsdk:"port"`
}

func firewallRuleDestinationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"network_id":         types.StringType,
		"network_type":       types.StringType,
		"firewall_group_ids": types.SetType{ElemType: types.StringType},
		"address":            types.StringType,
		"address_ipv6":       types.StringType,
		"port":               types.StringType,
	}
}

// firewallRuleStateModel is the `state` nested object.
type firewallRuleStateModel struct {
	Established types.Bool `tfsdk:"established"`
	Invalid     types.Bool `tfsdk:"invalid"`
	New         types.Bool `tfsdk:"new"`
	Related     types.Bool `tfsdk:"related"`
}

func firewallRuleStateAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"established": types.BoolType,
		"invalid":     types.BoolType,
		"new":         types.BoolType,
		"related":     types.BoolType,
	}
}

// Object-level defaults reproduce the values the flat attributes used to send
// when the practitioner left a whole group out of configuration, so the request
// body on create is unchanged by the nesting.

func firewallRuleSourceDefault() types.Object {
	return types.ObjectValueMust(firewallRuleSourceAttrTypes(), map[string]attr.Value{
		"network_id":         types.StringNull(),
		"network_type":       types.StringValue("NETv4"),
		"firewall_group_ids": types.SetNull(types.StringType),
		"address":            types.StringNull(),
		"address_ipv6":       types.StringNull(),
		"port":               types.StringNull(),
		"mac":                hwtypes.NewMACAddressNull(),
	})
}

func firewallRuleDestinationDefault() types.Object {
	return types.ObjectValueMust(firewallRuleDestinationAttrTypes(), map[string]attr.Value{
		"network_id":         types.StringNull(),
		"network_type":       types.StringValue("NETv4"),
		"firewall_group_ids": types.SetNull(types.StringType),
		"address":            types.StringNull(),
		"address_ipv6":       types.StringNull(),
		"port":               types.StringNull(),
	})
}

func firewallRuleStateDefault() types.Object {
	return types.ObjectValueMust(firewallRuleStateAttrTypes(), map[string]attr.Value{
		"established": types.BoolValue(false),
		"invalid":     types.BoolValue(false),
		"new":         types.BoolValue(false),
		"related":     types.BoolValue(false),
	})
}

func (r *firewallRuleResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *firewallRuleResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
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

func (r *firewallRuleResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		// v1: the flat src_*, dst_*, icmp_* and state_* attributes moved into
		//     the nested source, destination, icmp and state objects. See
		//     UpgradeState.
		Version:             1,
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
				MarkdownDescription: "The index of the rule. Must be in one of the interface-specific blocks: " +
					"`2000-2999` (LAN), `3000-3999` (WAN), `4000-4999` (GUEST), or their high-range " +
					"equivalents `20000-29999`, `30000-39999`, `40000-49999` used by newer UniFi OS versions.",
				Required: true,
				Validators: []validator.Int64{
					int64validator.Any(
						int64validator.Between(2000, 2999),
						int64validator.Between(3000, 3999),
						int64validator.Between(4000, 4999),
						int64validator.Between(20000, 29999),
						int64validator.Between(30000, 39999),
						int64validator.Between(40000, 49999),
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
			"icmp": schema.SingleNestedAttribute{
				MarkdownDescription: "ICMP type matching for the firewall rule.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"typename": schema.StringAttribute{
						MarkdownDescription: "ICMP type name.",
						Optional:            true,
					},
					"v6_typename": schema.StringAttribute{
						MarkdownDescription: "ICMPv6 type name.",
						Optional:            true,
					},
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the rule should be enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "The source match criteria of the firewall rule.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(firewallRuleSourceDefault()),
				Attributes: map[string]schema.Attribute{
					"network_id": schema.StringAttribute{
						MarkdownDescription: "The source network ID for the firewall rule.",
						Optional:            true,
					},
					"network_type": schema.StringAttribute{
						MarkdownDescription: "The source network type of the firewall rule. Can be one of `ADDRv4` or `NETv4`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("NETv4"),
						Validators: []validator.String{
							stringvalidator.OneOf("ADDRv4", "NETv4"),
						},
					},
					"firewall_group_ids": schema.SetAttribute{
						MarkdownDescription: "The source firewall group IDs for the firewall rule.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"address": schema.StringAttribute{
						MarkdownDescription: "The source address for the firewall rule.",
						Optional:            true,
					},
					"address_ipv6": schema.StringAttribute{
						MarkdownDescription: "The IPv6 source address for the firewall rule.",
						Optional:            true,
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "The source port of the firewall rule.",
						Optional:            true,
					},
					"mac": schema.StringAttribute{
						MarkdownDescription: "The source MAC address of the firewall rule.",
						CustomType:          hwtypes.MACAddressType{},
						Optional:            true,
					},
				},
			},
			"destination": schema.SingleNestedAttribute{
				MarkdownDescription: "The destination match criteria of the firewall rule.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(firewallRuleDestinationDefault()),
				Attributes: map[string]schema.Attribute{
					"network_id": schema.StringAttribute{
						MarkdownDescription: "The destination network ID of the firewall rule.",
						Optional:            true,
					},
					"network_type": schema.StringAttribute{
						MarkdownDescription: "The destination network type of the firewall rule. Can be one of `ADDRv4` or `NETv4`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("NETv4"),
						Validators: []validator.String{
							stringvalidator.OneOf("ADDRv4", "NETv4"),
						},
					},
					"firewall_group_ids": schema.SetAttribute{
						MarkdownDescription: "The destination firewall group IDs of the firewall rule.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"address": schema.StringAttribute{
						MarkdownDescription: "The destination address of the firewall rule.",
						Optional:            true,
					},
					"address_ipv6": schema.StringAttribute{
						MarkdownDescription: "The IPv6 destination address of the firewall rule.",
						Optional:            true,
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "The destination port of the firewall rule.",
						Optional:            true,
					},
				},
			},
			"logging": schema.BoolAttribute{
				MarkdownDescription: "Enable logging for the firewall rule.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"state": schema.SingleNestedAttribute{
				MarkdownDescription: "Connection state matching for the firewall rule.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(firewallRuleStateDefault()),
				Attributes: map[string]schema.Attribute{
					"established": schema.BoolAttribute{
						MarkdownDescription: "Match where the state is established.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"invalid": schema.BoolAttribute{
						MarkdownDescription: "Match where the state is invalid.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"new": schema.BoolAttribute{
						MarkdownDescription: "Match where the state is new.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"related": schema.BoolAttribute{
						MarkdownDescription: "Match where the state is related.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
			"ip_sec": schema.StringAttribute{
				MarkdownDescription: "Specify whether the rule matches on IPsec packets. Can be one of `match-ipset` or `match-none`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("match-ipsec", "match-none"),
				},
			},
			"setting_preference": schema.StringAttribute{
				MarkdownDescription: "Whether the rule is managed automatically by the controller or manually. Can be one of `auto` or `manual`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"protocol_match_excepted": schema.BoolAttribute{
				MarkdownDescription: "Match packets that do NOT match the specified protocol (protocol negation).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
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

	createTimeout, timeoutDiags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	firewallRule, convDiags := r.modelToFirewallRule(ctx, &data)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	createdFirewallRule, err := r.client.CreateFirewallRule(ctx, site, firewallRule)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Firewall Rule",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.firewallRuleToModel(ctx, createdFirewallRule, &data, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identity := firewallRuleIdentityModel{
		ID:   data.ID,
		Site: data.Site,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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

	readTimeout, timeoutDiags := data.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// Read identity, falling back to state for resources created before
	// identity support. This also lets Read work from an identity-only state
	// (the refresh right after an identity-based import).
	var identity firewallRuleIdentityModel
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		identity.ID = data.ID
		identity.Site = data.Site
	}

	id := data.ID.ValueString()
	if id == "" {
		id = identity.ID.ValueString()
	}
	site := data.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}

	firewallRule, err := r.client.GetFirewallRule(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Firewall Rule",
			"Could not read firewall rule with ID "+id+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.firewallRuleToModel(ctx, firewallRule, &data, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if identity.ID.IsNull() || identity.ID.ValueString() == "" {
		identity.ID = data.ID
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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

	updateTimeout, timeoutDiags := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	firewallRule, convDiags := r.modelToFirewallRule(ctx, &state)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	firewallRule.ID = state.ID.ValueString()

	updatedFirewallRule, err := r.client.UpdateFirewallRule(ctx, site, firewallRule)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Firewall Rule",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.firewallRuleToModel(ctx, updatedFirewallRule, &state, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = plan.Timeouts

	// Identity should not change during update; fall back to state for
	// resources created before identity support.
	identity := firewallRuleIdentityModel{
		ID:   state.ID,
		Site: state.Site,
	}
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.ValueString() == "" {
			identity.ID = state.ID
		}
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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

	deleteTimeout, timeoutDiags := data.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeleteFirewallRule(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Firewall Rule",
			err.Error(),
		)
		return
	}
}

func (r *firewallRuleResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Identity-based import (import block with identity, Terraform 1.12+).
	if req.ID == "" {
		var identity firewallRuleIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
		if !identity.Site.IsNull() && identity.Site.ValueString() != "" {
			resp.Diagnostics.Append(
				resp.State.SetAttribute(ctx, path.Root("site"), identity.Site)...,
			)
		}
		return
	}

	// Import by ID string ("id" or "site:id").
	idParts := strings.Split(req.ID, ":")

	var site, id string
	switch len(idParts) {
	case 2:
		site, id = idParts[0], idParts[1]
	case 1:
		id = idParts[0]
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'site:id' or 'id'",
		)
		return
	}

	if site != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)

	// Mirror into identity so it is populated from the first refresh on.
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), id)...)
		if site != "" {
			resp.Diagnostics.Append(
				resp.Identity.SetAttribute(ctx, path.Root("site"), site)...,
			)
		}
	}
}

// UpgradeState migrates prior firewall rule state to the current schema version.
//
//	v0 -> current: the flat src_*, dst_*, icmp_* and state_* attributes moved
//	    into the nested source, destination, icmp and state objects. See
//	    nestFirewallRuleState.
func (r *firewallRuleResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(
				ctx context.Context,
				req resource.UpgradeStateRequest,
				resp *resource.UpgradeStateResponse,
			) {
				if req.RawState == nil {
					return
				}
				dv, err := util.UpgradeRawState(
					schemaType,
					req.RawState.JSON,
					nestFirewallRuleState,
				)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to upgrade firewall rule state",
						err.Error(),
					)
					return
				}
				resp.DynamicValue = dv
			},
		},
	}
}

// nestFirewallRuleState rewrites flat v0 firewall rule state into the
// nested-object layout introduced in schema v1. Keys that are absent are
// skipped, so it is safe to run on state written before a field existed.
func nestFirewallRuleState(state map[string]any) {
	util.NestFields(state, "source", map[string]string{
		"src_network_id":         "network_id",
		"src_network_type":       "network_type",
		"src_firewall_group_ids": "firewall_group_ids",
		"src_address":            "address",
		"src_address_ipv6":       "address_ipv6",
		"src_port":               "port",
		"src_mac":                "mac",
	})
	util.NestFields(state, "destination", map[string]string{
		"dst_network_id":         "network_id",
		"dst_network_type":       "network_type",
		"dst_firewall_group_ids": "firewall_group_ids",
		"dst_address":            "address",
		"dst_address_ipv6":       "address_ipv6",
		"dst_port":               "port",
	})
	util.NestFields(state, "icmp", map[string]string{
		"icmp_typename":    "typename",
		"icmp_v6_typename": "v6_typename",
	})
	// icmp is Optional-only: an unset group is a null object, not an object
	// of nulls, so v0 state with neither ICMP type set must not upgrade into
	// an `icmp = {}` that the next plan would want to remove.
	util.WithObject(state, "icmp", func(icmp map[string]any) {
		if icmp["typename"] == nil && icmp["v6_typename"] == nil {
			state["icmp"] = nil
		}
	})
	util.NestFields(state, "state", map[string]string{
		"state_established": "established",
		"state_invalid":     "invalid",
		"state_new":         "new",
		"state_related":     "related",
	})
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
	// Nested groups: re-assert every sub-attribute the plan knows, keeping
	// the state's value for the rest (exactly as the flat attributes did).
	state.ICMP = util.OverlayKnownObject(ctx, plan.ICMP, state.ICMP)
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	state.Source = util.OverlayKnownObject(ctx, plan.Source, state.Source)
	state.Destination = util.OverlayKnownObject(ctx, plan.Destination, state.Destination)
	if !plan.Logging.IsNull() && !plan.Logging.IsUnknown() {
		state.Logging = plan.Logging
	}
	state.State = util.OverlayKnownObject(ctx, plan.State, state.State)
	if !plan.IPSec.IsNull() && !plan.IPSec.IsUnknown() {
		state.IPSec = plan.IPSec
	}
	if !plan.SettingPreference.IsNull() && !plan.SettingPreference.IsUnknown() {
		state.SettingPreference = plan.SettingPreference
	}
	if !plan.ProtocolMatchExcept.IsNull() && !plan.ProtocolMatchExcept.IsUnknown() {
		state.ProtocolMatchExcept = plan.ProtocolMatchExcept
	}
}

func (r *firewallRuleResource) modelToFirewallRule(
	ctx context.Context,
	model *firewallRuleResourceModel,
) (*unifi.FirewallRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	firewallRule := &unifi.FirewallRule{
		Name:      model.Name.ValueString(),
		Action:    model.Action.ValueString(),
		Ruleset:   model.Ruleset.ValueString(),
		RuleIndex: model.RuleIndex.ValueInt64Pointer(),
		Enabled:   model.Enabled.ValueBool(),
	}

	if !model.Protocol.IsNull() {
		firewallRule.Protocol = model.Protocol.ValueString()
	}
	if !model.ProtocolV6.IsNull() {
		firewallRule.ProtocolV6 = model.ProtocolV6.ValueString()
	}

	// A null or unknown nested group contributes nothing, exactly as the
	// flat attributes did when unset.
	if icmp, ok, d := util.ObjectAs[firewallRuleICMPModel](ctx, model.ICMP); ok {
		if !icmp.Typename.IsNull() {
			firewallRule.ICMPTypename = icmp.Typename.ValueString()
		}
		if !icmp.V6Typename.IsNull() {
			firewallRule.ICMPv6Typename = icmp.V6Typename.ValueString()
		}
	} else {
		diags.Append(d...)
	}

	if src, ok, d := util.ObjectAs[firewallRuleSourceModel](ctx, model.Source); ok {
		if !src.NetworkID.IsNull() {
			firewallRule.SrcNetworkID = src.NetworkID.ValueString()
		}
		if !src.NetworkType.IsNull() {
			firewallRule.SrcNetworkType = src.NetworkType.ValueString()
		}
		if !src.FirewallGroupIDs.IsNull() {
			var groupIDs []string
			diags.Append(src.FirewallGroupIDs.ElementsAs(ctx, &groupIDs, false)...)
			firewallRule.SrcFirewallGroupIDs = groupIDs
		}
		if !src.Address.IsNull() {
			firewallRule.SrcAddress = src.Address.ValueString()
		}
		if !src.AddressIPv6.IsNull() {
			firewallRule.SrcAddressIPV6 = src.AddressIPv6.ValueString()
		}
		if !src.Port.IsNull() {
			firewallRule.SrcPort = src.Port.ValueString()
		}
		if !src.Mac.IsNull() {
			firewallRule.SrcMACAddress = src.Mac.ValueString()
		}
	} else {
		diags.Append(d...)
	}

	if dst, ok, d := util.ObjectAs[firewallRuleDestinationModel](ctx, model.Destination); ok {
		if !dst.NetworkID.IsNull() {
			firewallRule.DstNetworkID = dst.NetworkID.ValueString()
		}
		if !dst.NetworkType.IsNull() {
			firewallRule.DstNetworkType = dst.NetworkType.ValueString()
		}
		if !dst.FirewallGroupIDs.IsNull() {
			var groupIDs []string
			diags.Append(dst.FirewallGroupIDs.ElementsAs(ctx, &groupIDs, false)...)
			firewallRule.DstFirewallGroupIDs = groupIDs
		}
		if !dst.Address.IsNull() {
			firewallRule.DstAddress = dst.Address.ValueString()
		}
		if !dst.AddressIPv6.IsNull() {
			firewallRule.DstAddressIPV6 = dst.AddressIPv6.ValueString()
		}
		if !dst.Port.IsNull() {
			firewallRule.DstPort = dst.Port.ValueString()
		}
	} else {
		diags.Append(d...)
	}

	if !model.Logging.IsNull() {
		firewallRule.Logging = model.Logging.ValueBool()
	}

	if st, ok, d := util.ObjectAs[firewallRuleStateModel](ctx, model.State); ok {
		if !st.Established.IsNull() {
			firewallRule.StateEstablished = st.Established.ValueBool()
		}
		if !st.Invalid.IsNull() {
			firewallRule.StateInvalid = st.Invalid.ValueBool()
		}
		if !st.New.IsNull() {
			firewallRule.StateNew = st.New.ValueBool()
		}
		if !st.Related.IsNull() {
			firewallRule.StateRelated = st.Related.ValueBool()
		}
	} else {
		diags.Append(d...)
	}

	if !model.IPSec.IsNull() {
		firewallRule.IPSec = model.IPSec.ValueString()
	}
	if !model.SettingPreference.IsNull() {
		firewallRule.SettingPreference = model.SettingPreference.ValueString()
	}
	firewallRule.ProtocolMatchExcepted = model.ProtocolMatchExcept.ValueBool()

	return firewallRule, diags
}

func (r *firewallRuleResource) firewallRuleToModel(
	ctx context.Context,
	firewallRule *unifi.FirewallRule,
	model *firewallRuleResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(firewallRule.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(firewallRule.Name)
	model.Action = types.StringValue(firewallRule.Action)
	model.Ruleset = types.StringValue(firewallRule.Ruleset)
	model.RuleIndex = types.Int64PointerValue(firewallRule.RuleIndex)
	model.Enabled = types.BoolValue(firewallRule.Enabled)
	model.Protocol = stringOrNull(firewallRule.Protocol)
	model.ProtocolV6 = stringOrNull(firewallRule.ProtocolV6)

	icmp, d := firewallRuleICMPToFramework(ctx, firewallRule, model.ICMP)
	diags.Append(d...)
	model.ICMP = icmp

	srcGroupIDs, d := firewallRuleGroupIDsToFramework(ctx, firewallRule.SrcFirewallGroupIDs)
	diags.Append(d...)
	source, d := types.ObjectValueFrom(ctx, firewallRuleSourceAttrTypes(), firewallRuleSourceModel{
		NetworkID:        stringOrNull(firewallRule.SrcNetworkID),
		NetworkType:      firewallRuleNetworkTypeToFramework(firewallRule.SrcNetworkType),
		FirewallGroupIDs: srcGroupIDs,
		Address:          stringOrNull(firewallRule.SrcAddress),
		AddressIPv6:      stringOrNull(firewallRule.SrcAddressIPV6),
		Port:             stringOrNull(firewallRule.SrcPort),
		Mac:              util.MACValueOrNull(firewallRule.SrcMACAddress),
	})
	diags.Append(d...)
	model.Source = source

	dstGroupIDs, d := firewallRuleGroupIDsToFramework(ctx, firewallRule.DstFirewallGroupIDs)
	diags.Append(d...)
	destination, d := types.ObjectValueFrom(
		ctx,
		firewallRuleDestinationAttrTypes(),
		firewallRuleDestinationModel{
			NetworkID:        stringOrNull(firewallRule.DstNetworkID),
			NetworkType:      firewallRuleNetworkTypeToFramework(firewallRule.DstNetworkType),
			FirewallGroupIDs: dstGroupIDs,
			Address:          stringOrNull(firewallRule.DstAddress),
			AddressIPv6:      stringOrNull(firewallRule.DstAddressIPV6),
			Port:             stringOrNull(firewallRule.DstPort),
		},
	)
	diags.Append(d...)
	model.Destination = destination

	model.Logging = types.BoolValue(firewallRule.Logging)

	state, d := types.ObjectValueFrom(ctx, firewallRuleStateAttrTypes(), firewallRuleStateModel{
		Established: types.BoolValue(firewallRule.StateEstablished),
		Invalid:     types.BoolValue(firewallRule.StateInvalid),
		New:         types.BoolValue(firewallRule.StateNew),
		Related:     types.BoolValue(firewallRule.StateRelated),
	})
	diags.Append(d...)
	model.State = state

	model.IPSec = stringOrNull(firewallRule.IPSec)
	model.SettingPreference = stringOrNull(firewallRule.SettingPreference)
	model.ProtocolMatchExcept = types.BoolValue(firewallRule.ProtocolMatchExcepted)

	return diags
}

// firewallRuleICMPToFramework builds the `icmp` object from the API response.
// Both leaves are Optional-only, so a rule with neither ICMP type set reads
// back as a null object (as the flat attributes read back as null). The one
// exception is a practitioner-supplied empty block (`icmp = {}`), which has no
// API representation and is kept as configured.
func firewallRuleICMPToFramework(
	ctx context.Context,
	firewallRule *unifi.FirewallRule,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	icmp := firewallRuleICMPModel{
		Typename:   stringOrNull(firewallRule.ICMPTypename),
		V6Typename: stringOrNull(firewallRule.ICMPv6Typename),
	}
	if icmp.Typename.IsNull() && icmp.V6Typename.IsNull() && !isKnownEmptyObject(prior) {
		return types.ObjectNull(firewallRuleICMPAttrTypes()), nil
	}
	return types.ObjectValueFrom(ctx, firewallRuleICMPAttrTypes(), icmp)
}

// isKnownEmptyObject reports whether obj is a known, non-null object whose
// attributes are all null - the value of an empty `x = {}` block.
func isKnownEmptyObject(obj types.Object) bool {
	if obj.IsNull() || obj.IsUnknown() {
		return false
	}
	for _, v := range obj.Attributes() {
		if !v.IsNull() {
			return false
		}
	}
	return true
}

// firewallRuleNetworkTypeToFramework mirrors the `network_type` default: the
// controller omits it for rules that match by network, so an empty value
// reads back as `NETv4`.
func firewallRuleNetworkTypeToFramework(networkType string) types.String {
	if networkType == "" {
		return types.StringValue("NETv4")
	}
	return types.StringValue(networkType)
}

// firewallRuleGroupIDsToFramework converts a firewall group ID list to a Set,
// null when empty.
func firewallRuleGroupIDsToFramework(
	ctx context.Context,
	ids []string,
) (types.Set, diag.Diagnostics) {
	if len(ids) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, ids)
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *firewallRuleResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List firewall rules in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list firewall rules from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`, `ruleset`, `action`, `enabled`.",
							Required:            true,
						},
						"value": listschema.StringAttribute{
							MarkdownDescription: "The value to filter by.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

// List implements [list.ListResource].
func (r *firewallRuleResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config firewallRuleListConfigModel

	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Process filter blocks.
	var filters []firewallRuleListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	rules, err := r.client.ListFirewallRule(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Firewall Rules", "Could not list firewall rules: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, rule := range rules {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if rule.Name != val {
					continue
				}
			}

			// Apply ruleset filter.
			if val, ok := postFilters["ruleset"]; ok {
				if rule.Ruleset != val {
					continue
				}
			}

			// Apply action filter.
			if val, ok := postFilters["action"]; ok {
				if rule.Action != val {
					continue
				}
			}

			// Apply enabled filter.
			if val, ok := postFilters["enabled"]; ok {
				enabled := fmt.Sprintf("%t", rule.Enabled)
				if enabled != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if rule.Name != "" {
				result.DisplayName = rule.Name
			} else {
				result.DisplayName = rule.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(rule.ID),
				)...,
			)
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("site"),
					types.StringValue(site),
				)...,
			)

			// Convert to model.
			var model firewallRuleResourceModel
			ruleCopy := rule
			result.Diagnostics.Append(r.firewallRuleToModel(ctx, &ruleCopy, &model, site)...)
			model.Timeouts = timeoutsNullValue()
			result.Diagnostics.Append(result.Resource.Set(ctx, model)...)

			if !push(result) {
				return
			}
		}
	}
}
