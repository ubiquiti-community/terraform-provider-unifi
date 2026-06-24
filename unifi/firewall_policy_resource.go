package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
	_ resource.Resource                 = &firewallPolicyResource{}
	_ resource.ResourceWithImportState  = &firewallPolicyResource{}
	_ resource.ResourceWithIdentity     = &firewallPolicyResource{}
	_ resource.ResourceWithUpgradeState = &firewallPolicyResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &firewallPolicyResource{}
	_ list.ListResourceWithConfigure = &firewallPolicyResource{}
)

func NewFirewallPolicyResource() resource.Resource {
	return &firewallPolicyResource{}
}

func NewFirewallPolicyListResource() list.ListResource {
	return &firewallPolicyResource{}
}

// firewallPolicyListConfigModel describes the list configuration model.
type firewallPolicyListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// firewallPolicyListFilterModel represents a single name/value filter entry.
type firewallPolicyListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
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
	ConnectionStateType types.String   `tfsdk:"connection_state_type"`
	ConnectionStates    types.List     `tfsdk:"connection_states"`
	ICMPTypename        types.String   `tfsdk:"icmp_typename"`
	ICMPV6Typename      types.String   `tfsdk:"icmp_v6_typename"`
	Source              types.Object   `tfsdk:"source"`
	Destination         types.Object   `tfsdk:"destination"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// firewallPolicyEndpointModel is the nested source/destination block model.
type firewallPolicyEndpointModel struct {
	ZoneID           types.String `tfsdk:"zone_id"`
	MatchingTarget   types.String `tfsdk:"matching_target"`
	NetworkIDs       types.List   `tfsdk:"network_ids"`
	ClientMACs       types.List   `tfsdk:"client_macs"`
	IPs              types.List   `tfsdk:"ips"`
	WebDomains       types.List   `tfsdk:"web_domains"`
	Port             types.String `tfsdk:"port"`
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
		"port":                 types.StringType,
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

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *firewallPolicyResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
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
		"port": schema.StringAttribute{
			MarkdownDescription: "Port(s) to match when `port_matching_type` is `SPECIFIC`. " +
				"A single port (`161`) or a comma-separated list of ports/ranges " +
				"(`80,443`, `8000-8100`). Leave unset for no port match.",
			Optional: true,
			Computed: true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(
					regexp.MustCompile(`^[0-9]{1,5}(-[0-9]{1,5})?(,[0-9]{1,5}(-[0-9]{1,5})?)*$`),
					"must be a port number or a comma-separated list of ports/ranges "+
						`(e.g. "80,443" or "8000-8100")`,
				),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
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
		Version: 1,
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
				MarkdownDescription: "The protocol to match: `all`, `tcp`, `udp`, `tcp_udp`, " +
					"`icmp`, or `icmpv6`. Defaults to `all`. Note: for `icmp`/`icmpv6` " +
					"policies the controller rejects `create_allow_respond = true` " +
					"(`FirewallPolicyCreateRespondTrafficPolicyNotAllowed`) — keep it " +
					"`false` and add an explicit reverse policy if you need the reply.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("all"),
				Validators: []validator.String{
					stringvalidator.OneOf("all", "tcp", "udp", "tcp_udp", "icmp", "icmpv6"),
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
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
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

	createTimeout, timeoutDiags := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

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
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
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

	readTimeout, timeoutDiags := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

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
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
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

	updateTimeout, timeoutDiags := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

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

	// matching_target_type is firmware-derived: the controller (and the
	// provider's own firewallPolicyMatchingTargetType helper) may set it to a
	// concrete value during the PUT (e.g. "" -> "SPECIFIC" for a non-ANY match),
	// which the planned value cannot anticipate. It is Computed +
	// UseStateForUnknown, so the planned value is the prior-state value; capture
	// it now and re-assert it on the post-apply state so Terraform's
	// "inconsistent result after apply" check passes for policies whose state
	// still carries an empty type (#324). The next Read reconciles state with the
	// controller's value.
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
		plan.Destination = withMatchingTargetType(
			ctx,
			plan.Destination,
			plannedDstMTT,
			&resp.Diagnostics,
		)
	}

	plan.Site = types.StringValue(site)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
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

	deleteTimeout, timeoutDiags := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

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
// State upgrade (schema v0 -> v1: port int64 -> string)
// ---------------------------------------------------------------------------

// firewallPolicyEndpointModelV0 mirrors firewallPolicyEndpointModel but with the
// pre-v1 integer `port`. It exists only to decode prior state during upgrade.
type firewallPolicyEndpointModelV0 struct {
	ZoneID             types.String `tfsdk:"zone_id"`
	MatchingTarget     types.String `tfsdk:"matching_target"`
	NetworkIDs         types.List   `tfsdk:"network_ids"`
	ClientMACs         types.List   `tfsdk:"client_macs"`
	IPs                types.List   `tfsdk:"ips"`
	WebDomains         types.List   `tfsdk:"web_domains"`
	Port               types.Int64  `tfsdk:"port"`
	PortGroupID        types.String `tfsdk:"port_group_id"`
	PortMatchingType   types.String `tfsdk:"port_matching_type"`
	MatchingTargetType types.String `tfsdk:"matching_target_type"`
}

func (r *firewallPolicyResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	// Build the prior (v0) schema from the current one and swap the
	// source/destination `port` back to an integer — that is the only
	// structural difference. Deriving it from the live schema keeps the
	// upgrader correct as the rest of the schema evolves.
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	priorSchema := schemaResp.Schema
	priorSchema.Version = 0
	for _, key := range []string{"source", "destination"} {
		nested, ok := priorSchema.Attributes[key].(schema.SingleNestedAttribute)
		if !ok {
			continue
		}
		attrs := make(map[string]schema.Attribute, len(nested.Attributes))
		for k, v := range nested.Attributes {
			attrs[k] = v
		}
		attrs["port"] = schema.Int64Attribute{Optional: true, Computed: true}
		nested.Attributes = attrs
		priorSchema.Attributes[key] = nested
	}

	return map[int64]resource.StateUpgrader{
		// v0 modeled `port` as an integer, which both dropped multi-port values
		// (#286) and serialized portless endpoints as the invalid "0" (#288).
		// v1 models it as a string; convert the stored number, treating 0/null
		// as "no port".
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(
				ctx context.Context,
				req resource.UpgradeStateRequest,
				resp *resource.UpgradeStateResponse,
			) {
				var state firewallPolicyModel
				resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
				if resp.Diagnostics.HasError() {
					return
				}

				state.Source = upgradeFirewallPolicyEndpointV0(
					ctx, state.Source, &resp.Diagnostics,
				)
				state.Destination = upgradeFirewallPolicyEndpointV0(
					ctx, state.Destination, &resp.Diagnostics,
				)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			},
		},
	}
}

func upgradeFirewallPolicyEndpointV0(
	ctx context.Context,
	obj types.Object,
	diags *diag.Diagnostics,
) types.Object {
	newTypes := firewallPolicyEndpointModel{}.AttributeTypes()
	if obj.IsNull() {
		return types.ObjectNull(newTypes)
	}
	if obj.IsUnknown() {
		return types.ObjectUnknown(newTypes)
	}

	var v0 firewallPolicyEndpointModelV0
	diags.Append(obj.As(ctx, &v0, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return obj
	}

	port := types.StringNull()
	if !v0.Port.IsNull() && !v0.Port.IsUnknown() && v0.Port.ValueInt64() != 0 {
		port = types.StringValue(strconv.FormatInt(v0.Port.ValueInt64(), 10))
	}

	upgraded := firewallPolicyEndpointModel{
		ZoneID:             v0.ZoneID,
		MatchingTarget:     v0.MatchingTarget,
		NetworkIDs:         v0.NetworkIDs,
		ClientMACs:         v0.ClientMACs,
		IPs:                v0.IPs,
		WebDomains:         v0.WebDomains,
		Port:               port,
		PortGroupID:        v0.PortGroupID,
		PortMatchingType:   v0.PortMatchingType,
		MatchingTargetType: v0.MatchingTargetType,
	}

	newObj, d := types.ObjectValueFrom(ctx, newTypes, upgraded)
	diags.Append(d...)
	return newObj
}

// portToStringValue maps the API port string to a Terraform value. The API
// returns "" for a portless endpoint and historically "0" for policies created
// by older provider versions (#288); both map to null so plans stay clean.
func portToStringValue(p string) types.String {
	if p == "" || p == "0" {
		return types.StringNull()
	}
	return types.StringValue(p)
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

// firewallPolicyMatchingTargetType ensures a concrete matching_target_type is
// sent for a specific (non-ANY) match. The controller rejects an IP/NETWORK/etc.
// match whose matching_target_type is empty (#293,
// api.err.MissingFirewallPolicySourceMatchingTargetType) — which happens when a
// source is switched from ANY to a specific target, leaving the round-tripped
// type empty or a stale "ANY". A controller-assigned type (SPECIFIC/OBJECT/LIST)
// is preserved.
func firewallPolicyMatchingTargetType(matchingTarget, currentType string) string {
	if matchingTarget != "" && matchingTarget != "ANY" &&
		(currentType == "" || currentType == "ANY") {
		return "SPECIFIC"
	}
	return currentType
}

func endpointModelToSource(
	ctx context.Context,
	m firewallPolicyEndpointModel,
	diags *diag.Diagnostics,
) *unifi.FirewallPolicySource {
	ep := &unifi.FirewallPolicySource{
		ZoneID:         m.ZoneID.ValueString(),
		MatchingTarget: m.MatchingTarget.ValueString(),
		MatchingTargetType: firewallPolicyMatchingTargetType(
			m.MatchingTarget.ValueString(), m.MatchingTargetType.ValueString(),
		),
		Port:             m.Port.ValueString(),
		PortGroupID:      m.PortGroupID.ValueString(),
		PortMatchingType: m.PortMatchingType.ValueString(),
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
		ZoneID:         m.ZoneID.ValueString(),
		MatchingTarget: m.MatchingTarget.ValueString(),
		MatchingTargetType: firewallPolicyMatchingTargetType(
			m.MatchingTarget.ValueString(), m.MatchingTargetType.ValueString(),
		),
		Port:             m.Port.ValueString(),
		PortGroupID:      m.PortGroupID.ValueString(),
		PortMatchingType: m.PortMatchingType.ValueString(),
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
		Port:               portToStringValue(src.Port),
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
		Port:               portToStringValue(dst.Port),
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

// ---------------------------------------------------------------------------
// List resource
// ---------------------------------------------------------------------------

// firewallPolicyListToModel populates the model's schema fields directly from
// the API struct for listing. It reuses the nil-safe firewallPolicyToModel
// flatten helper (which faithfully maps the source/destination nested objects)
// and sets the site so the listed resource is self-contained.
func (r *firewallPolicyResource) firewallPolicyListToModel(
	ctx context.Context,
	api *unifi.FirewallPolicy,
	model *firewallPolicyModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(firewallPolicyToModel(ctx, api, model)...)
	model.Site = types.StringValue(site)
	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *firewallPolicyResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List firewall policies in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list firewall policies from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`, `action`, `enabled`.",
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
func (r *firewallPolicyResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config firewallPolicyListConfigModel

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
	var filters []firewallPolicyListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	policies, err := r.client.ListFirewallPolicy(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(
			"Error Listing Firewall Policies",
			"Could not list firewall policies: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, policy := range policies {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if policy.Name != val {
					continue
				}
			}

			// Apply action filter.
			if val, ok := postFilters["action"]; ok {
				if policy.Action != val {
					continue
				}
			}

			// Apply enabled filter.
			if val, ok := postFilters["enabled"]; ok {
				enabled := fmt.Sprintf("%t", policy.Enabled)
				if enabled != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to ID.
			if policy.Name != "" {
				result.DisplayName = policy.Name
			} else {
				result.DisplayName = policy.ID
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(policy.ID),
				)...,
			)

			// Convert to model.
			p := policy
			var model firewallPolicyModel
			result.Diagnostics.Append(r.firewallPolicyListToModel(ctx, &p, &model, site)...)
			if !result.Diagnostics.HasError() {
				model.Timeouts = timeoutsNullValue()
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}
