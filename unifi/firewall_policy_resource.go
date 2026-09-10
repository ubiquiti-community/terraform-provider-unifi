package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

var (
	_ resource.Resource                 = &firewallPolicyResource{}
	_ resource.ResourceWithImportState  = &firewallPolicyResource{}
	_ resource.ResourceWithIdentity     = &firewallPolicyResource{}
	_ resource.ResourceWithModifyPlan   = &firewallPolicyResource{}
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

// firewallPolicyIdentityModel describes the resource identity data model.
type firewallPolicyIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
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
	// (an omitted connection_state_type/icmp.typename makes the PUT fail HTTP 400).
	ConnectionStateType types.String   `tfsdk:"connection_state_type"`
	ConnectionStates    types.List     `tfsdk:"connection_states"`
	ICMP                types.Object   `tfsdk:"icmp"`
	Schedule            types.Object   `tfsdk:"schedule"`
	Source              types.Object   `tfsdk:"source"`
	Destination         types.Object   `tfsdk:"destination"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// firewallPolicyICMPModel is the nested `icmp` object: the controller-managed
// ICMP/ICMPv6 type matching modes that must be round-tripped on every PUT.
type firewallPolicyICMPModel struct {
	Typename   types.String `tfsdk:"typename"`
	V6Typename types.String `tfsdk:"v6_typename"`
}

func (m firewallPolicyICMPModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"typename":    types.StringType,
		"v6_typename": types.StringType,
	}
}

// firewallPolicyScheduleModel is the complete schedule shape returned by the
// zone-based firewall API. Date is used by ONE_TIME_ONLY on older firmware;
// DateStart/DateEnd are also returned by newer Network application versions.
type firewallPolicyScheduleModel struct {
	Date         types.String `tfsdk:"date"`
	DateStart    types.String `tfsdk:"date_start"`
	DateEnd      types.String `tfsdk:"date_end"`
	Mode         types.String `tfsdk:"mode"`
	Normalize    types.Bool   `tfsdk:"normalize"`
	RepeatOnDays types.Set    `tfsdk:"repeat_on_days"`
	Time         types.Object `tfsdk:"time"`
}

func (m firewallPolicyScheduleModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"date":           types.StringType,
		"date_start":     types.StringType,
		"date_end":       types.StringType,
		"mode":           types.StringType,
		"normalize":      types.BoolType,
		"repeat_on_days": types.SetType{ElemType: types.StringType},
		"time": types.ObjectType{
			AttrTypes: firewallPolicyScheduleTimeModel{}.AttributeTypes(),
		},
	}
}

// firewallPolicyScheduleTimeModel is the nested `schedule.time` object.
type firewallPolicyScheduleTimeModel struct {
	AllDay types.Bool   `tfsdk:"all_day"`
	Range  types.Object `tfsdk:"range"`
}

func (m firewallPolicyScheduleTimeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"all_day": types.BoolType,
		"range": types.ObjectType{
			AttrTypes: firewallPolicyScheduleTimeRangeModel{}.AttributeTypes(),
		},
	}
}

// firewallPolicyScheduleTimeRangeModel is the nested `schedule.time.range`
// object holding the HH:MM boundaries of a timed schedule.
type firewallPolicyScheduleTimeRangeModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

func (m firewallPolicyScheduleTimeRangeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"start": types.StringType,
		"end":   types.StringType,
	}
}

// firewallPolicyScheduleTimeValue builds a known `schedule.time` object from
// its leaves. The read path, plan normalization and tests all go through it so
// the object shape is always the same: a known object whose leaves may be
// null, exactly as the former flat attributes were always present in state.
func firewallPolicyScheduleTimeValue(allDay types.Bool, start, end types.String) types.Object {
	timeRange := types.ObjectValueMust(
		firewallPolicyScheduleTimeRangeModel{}.AttributeTypes(),
		map[string]attr.Value{"start": start, "end": end},
	)
	return types.ObjectValueMust(
		firewallPolicyScheduleTimeModel{}.AttributeTypes(),
		map[string]attr.Value{"all_day": allDay, "range": timeRange},
	)
}

// firewallPolicyScheduleTimeFields splits a `schedule.time` object into its
// leaves with the null/unknown semantics the flat attributes had: an unknown
// object (or range) yields unknown leaves and a null one yields null leaves.
func firewallPolicyScheduleTimeFields(obj types.Object) (types.Bool, types.String, types.String) {
	if obj.IsUnknown() {
		return types.BoolUnknown(), types.StringUnknown(), types.StringUnknown()
	}
	allDay, start, end := types.BoolNull(), types.StringNull(), types.StringNull()
	if obj.IsNull() {
		return allDay, start, end
	}
	attrs := obj.Attributes()
	if v, ok := attrs["all_day"].(types.Bool); ok {
		allDay = v
	}
	timeRange, ok := attrs["range"].(types.Object)
	switch {
	case !ok || timeRange.IsNull():
		return allDay, start, end
	case timeRange.IsUnknown():
		return allDay, types.StringUnknown(), types.StringUnknown()
	}
	rangeAttrs := timeRange.Attributes()
	if v, ok := rangeAttrs["start"].(types.String); ok {
		start = v
	}
	if v, ok := rangeAttrs["end"].(types.String); ok {
		end = v
	}
	return allDay, start, end
}

func firewallPolicyScheduleAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"date": schema.StringAttribute{
			MarkdownDescription: "Date used by `ONE_TIME_ONLY`, in `YYYY-MM-DD` format.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{stringvalidator.RegexMatches(
				regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`), "must use YYYY-MM-DD format",
			)},
		},
		"date_start": schema.StringAttribute{
			MarkdownDescription: "Start date used by `CUSTOM`, in `YYYY-MM-DD` format.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{stringvalidator.RegexMatches(
				regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`), "must use YYYY-MM-DD format",
			)},
		},
		"date_end": schema.StringAttribute{
			MarkdownDescription: "End date used by `CUSTOM`, in `YYYY-MM-DD` format.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{stringvalidator.RegexMatches(
				regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`), "must use YYYY-MM-DD format",
			)},
		},
		"mode": schema.StringAttribute{
			MarkdownDescription: "Schedule mode.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{stringvalidator.OneOf(
				"ALWAYS", "EVERY_DAY", "EVERY_WEEK", "ONE_TIME_ONLY", "CUSTOM",
			)},
		},
		"normalize": schema.BoolAttribute{
			MarkdownDescription: "Clear inherited fields that are unused by the selected mode.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"repeat_on_days": schema.SetAttribute{
			MarkdownDescription: "Weekdays on which the policy is active.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			Validators: []validator.Set{setvalidator.ValueStringsAre(
				stringvalidator.OneOf("mon", "tue", "wed", "thu", "fri", "sat", "sun"),
			)},
		},
		"time": schema.SingleNestedAttribute{
			MarkdownDescription: "Time of day during which a timed schedule is active.",
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"all_day": schema.BoolAttribute{
					MarkdownDescription: "Whether the policy is active all day.",
					Optional:            true,
					Computed:            true,
				},
				"range": schema.SingleNestedAttribute{
					MarkdownDescription: "Time range during which the policy is active when `all_day` is false.",
					Optional:            true,
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							MarkdownDescription: "Start time in 24-hour `HH:MM` format.",
							Optional:            true,
							Computed:            true,
							Validators:          []validator.String{validators.TimeOfDay()},
						},
						"end": schema.StringAttribute{
							MarkdownDescription: "End time in 24-hour `HH:MM` format.",
							Optional:            true,
							Computed:            true,
							Validators:          []validator.String{validators.TimeOfDay()},
						},
					},
				},
			},
		},
	}
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
	IPGroupID        types.String `tfsdk:"ip_group_id"`
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
		"ip_group_id":          types.StringType,
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
			"site": identityschema.StringAttribute{
				OptionalForImport: true,
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
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"client_macs": schema.ListAttribute{
			MarkdownDescription: "List of client MAC addresses to match. Used when `matching_target` is `CLIENT`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"ips": schema.ListAttribute{
			MarkdownDescription: "List of IP addresses or CIDR ranges to match. Used when `matching_target` is `IP`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"web_domains": schema.ListAttribute{
			MarkdownDescription: "List of domains/FQDNs to match. Used when `matching_target` is `WEB`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
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
		"ip_group_id": schema.StringAttribute{
			MarkdownDescription: "ID of a `unifi_firewall_group` (address-group type) to match. Used when `matching_target` is `IP` with `matching_target_type = OBJECT`.",
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
		Version: 2,
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
				MarkdownDescription: "The ordering index of the policy within its zone-pair, " +
					"assigned by the controller. **Read-only:** UniFi does not accept a " +
					"client-supplied index on create or update (the policy is always appended " +
					"to the end of its source/destination zone-pair), and the supported API " +
					"exposes no reorder operation, so policy ordering cannot be managed through " +
					"this provider. Reorder policies in the UniFi UI if needed.",
				Computed: true,
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
				MarkdownDescription: "Connection-state matching mode: `ALL` (any state), `RESPOND_ONLY` (established/related returns), or `CUSTOM` (match the states listed in `connection_states`). Optional: if omitted the controller assigns it (defaults to `ALL`) and the provider round-trips the value so updates are accepted.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALL", "RESPOND_ONLY", "CUSTOM"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_states": schema.ListAttribute{
				MarkdownDescription: "Connection states matched when `connection_state_type` is `CUSTOM` (`NEW`, `ESTABLISHED`, `RELATED`, `INVALID`). Optional: leave unset for `ALL`/`RESPOND_ONLY` and the controller manages it; the provider round-trips the value so a `CUSTOM` policy's states are not dropped on update (which the firmware rejects with HTTP 400).",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("NEW", "ESTABLISHED", "RELATED", "INVALID"),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"icmp": schema.SingleNestedAttribute{
				MarkdownDescription: "ICMP type matching modes. Managed by the UniFi controller; the provider round-trips them so updates are accepted.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"typename": schema.StringAttribute{
						MarkdownDescription: "ICMP type matching mode. Managed by the UniFi controller; the provider round-trips it so updates are accepted.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"v6_typename": schema.StringAttribute{
						MarkdownDescription: "ICMPv6 type matching mode. Managed by the UniFi controller; the provider round-trips it so updates are accepted.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"schedule": schema.SingleNestedAttribute{
				MarkdownDescription: "When the policy is active. The complete controller value is " +
					"round-tripped so updating another policy field does not reset its schedule. " +
					"Supported modes are `ALWAYS`, `EVERY_DAY`, `EVERY_WEEK`, `ONE_TIME_ONLY`, " +
					"and `CUSTOM`. Timed modes require `time.all_day`; when false, both " +
					"`time.range` boundaries are required. `EVERY_WEEK` also requires weekdays, `ONE_TIME_ONLY` " +
					"requires `date` and a time range, and `CUSTOM` requires a date range and weekdays. " +
					"Set `normalize` to clear inherited fields unused by the selected mode.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Object{firewallPolicyScheduleValidator{}},
				Attributes: firewallPolicyScheduleAttributes(),
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

func (r *firewallPolicyResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var value types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("schedule"), &value)...)
	if resp.Diagnostics.HasError() || value.IsNull() || value.IsUnknown() {
		return
	}
	var schedule firewallPolicyScheduleModel
	resp.Diagnostics.Append(value.As(ctx, &schedule, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || !normalizeFirewallPolicyScheduleModel(&schedule) {
		return
	}
	normalized, diags := types.ObjectValueFrom(ctx, schedule.AttributeTypes(), schedule)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("schedule"), normalized)...)
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
	identity := firewallPolicyIdentityModel{
		ID:   plan.ID,
		Site: plan.Site,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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

	// Read identity, falling back to state for resources created before
	// identity support. This also lets Read work from an identity-only state
	// (the refresh right after an identity-based import).
	var identity firewallPolicyIdentityModel
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		identity.ID = state.ID
		identity.Site = state.Site
	}

	id := state.ID.ValueString()
	if id == "" {
		id = identity.ID.ValueString()
	}
	site := state.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}

	fp, err := r.client.GetFirewallPolicy(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Firewall Policy",
			"Could not read firewall policy "+id+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(firewallPolicyToModel(ctx, fp, &state)...)
	state.Site = types.StringValue(site)
	if identity.ID.IsNull() || identity.ID.ValueString() == "" {
		identity.ID = state.ID
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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

	// Identity should not change during update; fall back to state for
	// resources created before identity support.
	identity := firewallPolicyIdentityModel{
		ID:   plan.ID,
		Site: plan.Site,
	}
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.ValueString() == "" {
			identity.ID = plan.ID
		}
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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
	// Import by ID string ("id" or "site:id").
	if req.ID != "" {
		id := req.ID
		site := ""
		idParts := strings.SplitN(req.ID, ":", 2)
		if len(idParts) == 2 {
			site, id = idParts[0], idParts[1]
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
		return
	}

	// Identity-based import (import block with identity, Terraform 1.12+).
	var identity firewallPolicyIdentityModel
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
}

// ---------------------------------------------------------------------------
// State upgrade
// ---------------------------------------------------------------------------

// UpgradeState migrates prior firewall policy state to the current schema
// version.
//
//	v0 -> current: source/destination `port` changed from an integer to a
//	    string (#286, #288); 0/null become "no port".
//	v1 -> current: icmp_typename/icmp_v6_typename moved under `icmp`, and
//	    schedule.time_all_day/time_range_start/time_range_end moved under
//	    `schedule.time` (see nestFirewallPolicyState).
//
// Each upgrader targets the CURRENT schema type, so v0 state picks up the
// nesting rewrite as well.
func (r *firewallPolicyResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	upgrader := func(rewrite func(state map[string]any)) resource.StateUpgrader {
		return resource.StateUpgrader{
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
					func(state map[string]any) {
						rewrite(state)
						nestFirewallPolicyState(state)
					},
				)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to upgrade firewall policy state", err.Error(),
					)
					return
				}
				resp.DynamicValue = dv
			},
		}
	}

	return map[int64]resource.StateUpgrader{
		0: upgrader(func(state map[string]any) {
			for _, key := range []string{"source", "destination"} {
				util.WithObject(state, key, upgradeFirewallPolicyPortV0)
			}
		}),
		1: upgrader(func(map[string]any) {}),
	}
}

// nestFirewallPolicyState rewrites flat v0/v1 firewall policy state into the
// nested-object layout introduced in schema v2. Keys that are absent are
// skipped, so it is safe to run on state from any earlier version.
func nestFirewallPolicyState(state map[string]any) {
	util.NestFields(state, "icmp", map[string]string{
		"icmp_typename":    "typename",
		"icmp_v6_typename": "v6_typename",
	})
	util.WithObject(state, "schedule", func(schedule map[string]any) {
		util.NestFields(schedule, "time", map[string]string{
			"time_all_day":     "all_day",
			"time_range_start": "range_start",
			"time_range_end":   "range_end",
		})
		util.WithObject(schedule, "time", func(t map[string]any) {
			util.NestFields(t, "range", map[string]string{
				"range_start": "start",
				"range_end":   "end",
			})
		})
	})
}

// upgradeFirewallPolicyPortV0 converts a v0 endpoint's integer `port` to the
// v1 string form. v0 both dropped multi-port values (#286) and serialized
// portless endpoints as the invalid "0" (#288), so 0/null become null.
func upgradeFirewallPolicyPortV0(endpoint map[string]any) {
	raw, ok := endpoint["port"]
	if !ok || raw == nil {
		return
	}
	port := ""
	switch v := raw.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			port = strconv.FormatInt(i, 10)
		} else {
			port = v.String()
		}
	case float64:
		port = strconv.FormatInt(int64(v), 10)
	case string:
		port = v
	}
	if port == "" || port == "0" {
		endpoint["port"] = nil
		return
	}
	endpoint["port"] = port
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
		ConnectionStates:    []string{},
	}

	// A null/unknown icmp object contributes nothing, as the unset flat
	// attributes did.
	icmp, icmpKnown, icmpDiags := util.ObjectAs[firewallPolicyICMPModel](ctx, model.ICMP)
	diags.Append(icmpDiags...)
	if icmpKnown {
		fp.ICMPTypename = icmp.Typename.ValueString()
		fp.ICMPV6Typename = icmp.V6Typename.ValueString()
	}

	if model.Schedule.IsNull() || model.Schedule.IsUnknown() {
		fp.Schedule = &unifi.FirewallPolicySchedule{Mode: "ALWAYS"}
	} else {
		var schedule firewallPolicyScheduleModel
		diags.Append(model.Schedule.As(ctx, &schedule, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			normalizeFirewallPolicyScheduleModel(&schedule)
			allDay, rangeStart, rangeEnd := firewallPolicyScheduleTimeFields(schedule.Time)
			var timeAllDay *bool
			if !allDay.IsNull() && !allDay.IsUnknown() {
				timeAllDay = allDay.ValueBoolPointer()
			}
			fp.Schedule = &unifi.FirewallPolicySchedule{
				Date:           schedule.Date.ValueString(),
				DateStart:      schedule.DateStart.ValueString(),
				DateEnd:        schedule.DateEnd.ValueString(),
				Mode:           schedule.Mode.ValueString(),
				TimeAllDay:     timeAllDay,
				TimeRangeStart: rangeStart.ValueString(),
				TimeRangeEnd:   rangeEnd.ValueString(),
			}
			if !schedule.RepeatOnDays.IsNull() && !schedule.RepeatOnDays.IsUnknown() {
				diags.Append(
					schedule.RepeatOnDays.ElementsAs(ctx, &fp.Schedule.RepeatOnDays, false)...,
				)
			}
		}
	}

	// Round-trip the connection states (e.g. ["NEW"]) the controller reported.
	// Omitting them makes a CUSTOM-state policy's PUT fail with HTTP 400 (#227).
	if !model.ConnectionStates.IsNull() && !model.ConnectionStates.IsUnknown() {
		diags.Append(model.ConnectionStates.ElementsAs(ctx, &fp.ConnectionStates, false)...)
	}

	// index is controller-assigned and read-only: UniFi ignores a client-supplied
	// value on create/update (the policy is appended to the end of its zone-pair) and
	// the supported API exposes no reorder operation, so we never send it (#348).

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
// type empty or a stale "ANY". A match that references an IP group via
// ip_group_id (#316) requires "OBJECT" instead: the controller rejects a group
// reference sent with "SPECIFIC" (api.err.EmptyFirewallDestinationIps), and on
// create the type is never controller-assigned, so a group reference derives
// "OBJECT" — overriding a stale ""/"ANY"/"SPECIFIC" from state (e.g. when a
// policy is switched from literal ips to a group). A controller-assigned
// "OBJECT"/"LIST" is preserved.
func firewallPolicyMatchingTargetType(matchingTarget, currentType, ipGroupID string) string {
	if ipGroupID != "" && currentType != "OBJECT" && currentType != "LIST" {
		return "OBJECT"
	}
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
			m.IPGroupID.ValueString(),
		),
		Port:             m.Port.ValueString(),
		PortGroupID:      m.PortGroupID.ValueString(),
		IPGroupID:        m.IPGroupID.ValueString(),
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
			m.IPGroupID.ValueString(),
		),
		Port:             m.Port.ValueString(),
		PortGroupID:      m.PortGroupID.ValueString(),
		IPGroupID:        m.IPGroupID.ValueString(),
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
	icmp, icmpDiags := types.ObjectValueFrom(
		ctx, firewallPolicyICMPModel{}.AttributeTypes(), firewallPolicyICMPModel{
			Typename:   types.StringValue(fp.ICMPTypename),
			V6Typename: types.StringValue(fp.ICMPV6Typename),
		},
	)
	diags.Append(icmpDiags...)
	model.ICMP = icmp
	if fp.Schedule == nil {
		model.Schedule = types.ObjectNull(firewallPolicyScheduleModel{}.AttributeTypes())
	} else {
		normalize := types.BoolValue(false)
		if !model.Schedule.IsNull() && !model.Schedule.IsUnknown() {
			var prior firewallPolicyScheduleModel
			d := model.Schedule.As(ctx, &prior, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if !prior.Normalize.IsNull() && !prior.Normalize.IsUnknown() {
				normalize = prior.Normalize
			}
		}
		repeatOnDays := types.SetValueMust(types.StringType, []attr.Value{})
		if fp.Schedule.RepeatOnDays != nil {
			var scheduleDiags diag.Diagnostics
			repeatOnDays, scheduleDiags = types.SetValueFrom(
				ctx, types.StringType, fp.Schedule.RepeatOnDays,
			)
			diags.Append(scheduleDiags...)
		}
		schedule := firewallPolicyScheduleModel{
			Date:         util.StringValueOrNull(fp.Schedule.Date),
			DateStart:    util.StringValueOrNull(fp.Schedule.DateStart),
			DateEnd:      util.StringValueOrNull(fp.Schedule.DateEnd),
			Mode:         util.StringValueOrNull(fp.Schedule.Mode),
			Normalize:    normalize,
			RepeatOnDays: repeatOnDays,
			Time: firewallPolicyScheduleTimeValue(
				types.BoolPointerValue(fp.Schedule.TimeAllDay),
				util.StringValueOrNull(fp.Schedule.TimeRangeStart),
				util.StringValueOrNull(fp.Schedule.TimeRangeEnd),
			),
		}
		var scheduleDiags diag.Diagnostics
		model.Schedule, scheduleDiags = types.ObjectValueFrom(
			ctx, firewallPolicyScheduleModel{}.AttributeTypes(), schedule,
		)
		diags.Append(scheduleDiags...)
	}

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

func normalizeFirewallPolicyScheduleModel(schedule *firewallPolicyScheduleModel) bool {
	if schedule.Normalize.IsNull() || schedule.Normalize.IsUnknown() ||
		!schedule.Normalize.ValueBool() || schedule.Mode.IsNull() || schedule.Mode.IsUnknown() {
		return false
	}
	emptyDays := types.SetValueMust(types.StringType, []attr.Value{})
	switch schedule.Mode.ValueString() {
	case "ALWAYS":
		schedule.Date, schedule.DateStart, schedule.DateEnd = types.StringNull(), types.StringNull(), types.StringNull()
		schedule.RepeatOnDays = emptyDays
		schedule.Time = firewallPolicyScheduleTimeValue(
			types.BoolNull(), types.StringNull(), types.StringNull(),
		)
	case "EVERY_DAY":
		schedule.Date, schedule.DateStart, schedule.DateEnd = types.StringNull(), types.StringNull(), types.StringNull()
		schedule.RepeatOnDays = emptyDays
	case "EVERY_WEEK":
		schedule.Date, schedule.DateStart, schedule.DateEnd = types.StringNull(), types.StringNull(), types.StringNull()
	case "ONE_TIME_ONLY":
		schedule.DateStart, schedule.DateEnd = types.StringNull(), types.StringNull()
		schedule.RepeatOnDays = emptyDays
	case "CUSTOM":
		schedule.Date = types.StringNull()
	}
	// An all-day schedule carries no time range, whatever the plan held.
	if allDay, _, _ := firewallPolicyScheduleTimeFields(schedule.Time); !allDay.IsNull() &&
		!allDay.IsUnknown() && allDay.ValueBool() {
		schedule.Time = firewallPolicyScheduleTimeValue(
			allDay, types.StringNull(), types.StringNull(),
		)
	}
	return true
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
		IPGroupID:          types.StringValue(src.IPGroupID),
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
		IPGroupID:          types.StringValue(dst.IPGroupID),
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
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("site"),
					types.StringValue(site),
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
