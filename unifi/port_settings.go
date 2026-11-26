package unifi

import (
	"context"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Port settings shared by unifi_port_profile and unifi_device.port_override
// that are richer than a single flat field: link debounce (a value and its
// auto flag) and the QoS profile (a mode plus a list of match/mark policies).
// go-unifi generates separate structs for the two resources (PortProfile* and
// Device*) with the same JSON shape, so the conversions go through the
// API-neutral portQOSProfileAPI.

// ---- link_debounce ----------------------------------------------------------

// portLinkDebounceModel is the `link_debounce` nested object: the controller's
// link_debounce (milliseconds) and link_debounce_auto fields.
type portLinkDebounceModel struct {
	Auto  types.Bool           `tfsdk:"auto"`
	Delay timetypes.GoDuration `tfsdk:"delay"`
}

func portLinkDebounceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"auto":  types.BoolType,
		"delay": timetypes.GoDurationType{},
	}
}

// portLinkDebounceSchema is the `link_debounce` nested attribute. subject is
// the noun used in descriptions ("port profile", "port").
func portLinkDebounceSchema(subject string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Link debounce for the " + subject + ": delays link state changes to ignore brief " +
			"interruptions and reduce link flapping. Higher values can improve stability but may slow " +
			"failover or recovery. Taken from the controller when not set.",
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"auto": schema.BoolAttribute{
				Description: "Let the switch's global link debounce setting decide (the UI's " +
					"\"Automatically set\"); `delay` is then not used. Taken from the controller when not set.",
				Optional: true,
				Computed: true,
			},
			"delay": schema.StringAttribute{
				Description: "How long a link state change must last before the port acts on it, as a Go " +
					"duration string (e.g. `500ms`, `2s`): `0s` to `5s` in steps of `100ms`. " +
					"Taken from the controller when not set.",
				CustomType: timetypes.GoDurationType{},
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{
					validators.GoDurationBetween(0, 5*time.Second),
					validators.GoDurationMultipleOf(100 * time.Millisecond),
				},
			},
		},
	}
}

// portLinkDebounceObject builds the `link_debounce` object from API values. A
// nil delay maps to a null duration.
func portLinkDebounceObject(
	ctx context.Context,
	auto *bool,
	delayMs *int64,
) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, portLinkDebounceAttrTypes(), portLinkDebounceModel{
		Auto:  types.BoolPointerValue(auto),
		Delay: util.DurationPtrValue(delayMs, time.Millisecond),
	})
}

// ---- qos_profile ------------------------------------------------------------

// The controller's qos_profile_mode values for a port profile: `custom` takes
// the policies, the others are the UI's Pro AV presets.
var portQOSProfileModes = []string{
	"custom",
	"unifi_play",
	"aes67_audio",
	"crestron_audio_video",
	"dante_audio",
	"ndi_aes67_audio",
	"ndi_dante_audio",
	"qsys_audio_video",
	"qsys_video_dante_audio",
	"sdvoe_aes67_audio",
	"sdvoe_dante_audio",
	"shure_audio",
}

// The DSCP values the controller accepts for marking (the class selectors,
// AFxy and EF), as opposed to any 0-63 value for matching.
var portQOSMarkingDSCP = []int64{
	0, 8, 16, 24, 32, 40, 48, 56,
	10, 12, 14, 18, 20, 22, 26, 28, 30, 34, 36, 38, 44, 46,
}

// portQOSProtocolPattern is the controller's pattern for qos_matching.protocol:
// an IP protocol number or name.
var portQOSProtocolPattern = regexp.MustCompile(
	`^(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])|` +
		`ah|ax\.25|dccp|ddp|egp|eigrp|encap|esp|etherip|fc|ggp|gre|hip|hmp|icmp|idpr-cmtp|idrp|igmp|igp|ip|` +
		`ipcomp|ipencap|ipip|ipv6|ipv6-frag|ipv6-icmp|ipv6-nonxt|ipv6-opts|ipv6-route|isis|iso-tp4|l2tp|manet|` +
		`mobility-header|mpls-in-ip|ospf|pim|pup|rdp|rohc|rspf|rsvp|sctp|shim6|skip|st|tcp|udp|udplite|vmtp|` +
		`vrrp|wesp|xns-idp|xtp)$`,
)

type portQOSProfileModel struct {
	Mode     types.String `tfsdk:"mode"`
	Policies types.List   `tfsdk:"policies"`
}

type portQOSPolicyModel struct {
	Matching types.Object `tfsdk:"matching"`
	Marking  types.Object `tfsdk:"marking"`
}

type portQOSMatchingModel struct {
	CosCode          types.Int64  `tfsdk:"cos_code"`
	DscpCode         types.Int64  `tfsdk:"dscp_code"`
	DstPort          types.Int64  `tfsdk:"dst_port"`
	IPPrecedenceCode types.Int64  `tfsdk:"ip_precedence_code"`
	Protocol         types.String `tfsdk:"protocol"`
	SrcPort          types.Int64  `tfsdk:"src_port"`
}

type portQOSMarkingModel struct {
	CosCode          types.Int64 `tfsdk:"cos_code"`
	DscpCode         types.Int64 `tfsdk:"dscp_code"`
	IPPrecedenceCode types.Int64 `tfsdk:"ip_precedence_code"`
	Queue            types.Int64 `tfsdk:"queue"`
}

func portQOSMatchingAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cos_code":           types.Int64Type,
		"dscp_code":          types.Int64Type,
		"dst_port":           types.Int64Type,
		"ip_precedence_code": types.Int64Type,
		"protocol":           types.StringType,
		"src_port":           types.Int64Type,
	}
}

func portQOSMarkingAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cos_code":           types.Int64Type,
		"dscp_code":          types.Int64Type,
		"ip_precedence_code": types.Int64Type,
		"queue":              types.Int64Type,
	}
}

func portQOSPolicyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"matching": types.ObjectType{AttrTypes: portQOSMatchingAttrTypes()},
		"marking":  types.ObjectType{AttrTypes: portQOSMarkingAttrTypes()},
	}
}

func portQOSProfileAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":     types.StringType,
		"policies": types.ListType{ElemType: types.ObjectType{AttrTypes: portQOSPolicyAttrTypes()}},
	}
}

// portQOSProfileSchema is the `qos_profile` nested attribute.
//
// Every leaf is Optional+Computed, not just the object: Terraform keeps the
// prior value of an unset Optional+Computed nested attribute only when none of
// its set leaves is Optional-only (otherwise it infers the block was
// configured before and proposes null), so a QoS profile set in the UI on a
// resource that leaves qos_profile out would plan a change on every run.
// Computed leaves would in turn keep a leaf that configuration removes, so
// ModifyPlan plans a configured qos_profile exactly as written
// (planConfiguredObject).
func portQOSProfileSchema(subject string, modes []string) schema.SingleNestedAttribute {
	code := func(desc string, maxVal int64) schema.Int64Attribute {
		return schema.Int64Attribute{
			Description: desc,
			Optional:    true,
			Computed:    true,
			Validators:  []validator.Int64{int64validator.Between(0, maxVal)},
		}
	}
	port := func(desc string) schema.Int64Attribute {
		return code(desc, 65535)
	}
	modeExtra := "."
	if len(modes) > len(portQOSProfileModes) {
		modeExtra = ", or `smpte_st2110`."
	}
	return schema.SingleNestedAttribute{
		Description: "QoS profile for the " + subject + " (the UI's Pro AV and custom QoS): a preset `mode` " +
			"for an AV protocol, or `custom` with your own `policies`. Taken from the controller when not " +
			"set; `qos_profile = {}` removes it. The UI does not offer Pro AV and EEE together on one port.",
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				Description: "The QoS profile: `custom` (use `policies`), or a Pro AV preset: `unifi_play`, " +
					"`aes67_audio`, `crestron_audio_video`, `dante_audio`, `ndi_aes67_audio`, `ndi_dante_audio`, " +
					"`qsys_audio_video`, `qsys_video_dante_audio`, `sdvoe_aes67_audio`, `sdvoe_dante_audio` " +
					"or `shure_audio`" + modeExtra,
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{stringvalidator.OneOf(modes...)},
			},
			"policies": schema.ListNestedAttribute{
				Description: "The `custom` profile's policies, in order: traffic that matches `matching` gets " +
					"the `marking`.",
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"matching": schema.SingleNestedAttribute{
							Description: "Which traffic the policy applies to. Every field set must match.",
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"cos_code": code(
									"802.1p class of service, `0` to `7`.",
									7,
								),
								"dscp_code": code("DSCP value, `0` to `63`.", 63),
								"dst_port": port(
									"Destination TCP/UDP port, `0` to `65535`.",
								),
								"ip_precedence_code": code("IP precedence, `0` to `7`.", 7),
								"protocol": schema.StringAttribute{
									Description: "IP protocol, as a number (`0` to `255`) or a name " +
										"(`tcp`, `udp`, `icmp`, `igmp`, `pim`, ...).",
									Optional: true,
									Computed: true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(
											portQOSProtocolPattern,
											"must be an IP protocol number (0-255) or a protocol name the controller knows",
										),
									},
								},
								"src_port": port("Source TCP/UDP port, `0` to `65535`."),
							},
						},
						"marking": schema.SingleNestedAttribute{
							Description: "What matching traffic gets: an egress `queue`, which the controller " +
								"requires whenever a marking is set, and optionally DSCP, CoS or IP precedence " +
								"values to write.",
							Optional: true,
							Computed: true,
							Validators: []validator.Object{
								objectvalidator.AlsoRequires(path.MatchRelative().AtName("queue")),
							},
							Attributes: map[string]schema.Attribute{
								"cos_code": code("802.1p class of service to set, `0` to `7`.", 7),
								"dscp_code": schema.Int64Attribute{
									Description: "DSCP value to set: a class selector (`0`, `8`, ..., `56`), " +
										"an AF value (`10`, `12`, `14`, ..., `38`) or EF (`46`), or `44`.",
									Optional: true,
									Computed: true,
									Validators: []validator.Int64{
										int64validator.OneOf(portQOSMarkingDSCP...),
									},
								},
								"ip_precedence_code": code("IP precedence to set, `0` to `7`.", 7),
								"queue":              code("Switch egress queue, `0` to `7`.", 7),
							},
						},
					},
				},
			},
		},
	}
}

// portQOSProfileAPI is the API-neutral form of a qos_profile.
type portQOSProfileAPI struct {
	Mode     string
	Policies []portQOSPolicyAPI
}

type portQOSPolicyAPI struct {
	Matching *portQOSMatchingAPI
	Marking  *portQOSMarkingAPI
}

type portQOSMatchingAPI struct {
	CosCode, DscpCode, DstPort, IPPrecedenceCode, SrcPort *int64
	Protocol                                              string
}

type portQOSMarkingAPI struct {
	CosCode, DscpCode, IPPrecedenceCode, Queue *int64
}

// portQOSProfileObject builds the `qos_profile` object. A nil profile (none on
// the controller) is a null object. The controller drops an empty policy list
// from the body, so prior carries the model's previous object: a known empty
// `policies` there stays an empty list instead of reading back as null.
func portQOSProfileObject(
	ctx context.Context,
	q *portQOSProfileAPI,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if q == nil {
		return types.ObjectNull(portQOSProfileAttrTypes()), diags
	}

	policyType := types.ObjectType{AttrTypes: portQOSPolicyAttrTypes()}
	policies := types.ListNull(policyType)
	if len(q.Policies) > 0 {
		items := make([]portQOSPolicyModel, 0, len(q.Policies))
		for _, p := range q.Policies {
			item := portQOSPolicyModel{
				Matching: types.ObjectNull(portQOSMatchingAttrTypes()),
				Marking:  types.ObjectNull(portQOSMarkingAttrTypes()),
			}
			if m := p.Matching; m != nil {
				obj, d := types.ObjectValueFrom(
					ctx,
					portQOSMatchingAttrTypes(),
					portQOSMatchingModel{
						CosCode:          types.Int64PointerValue(m.CosCode),
						DscpCode:         types.Int64PointerValue(m.DscpCode),
						DstPort:          types.Int64PointerValue(m.DstPort),
						IPPrecedenceCode: types.Int64PointerValue(m.IPPrecedenceCode),
						Protocol:         util.StringValueOrNull(m.Protocol),
						SrcPort:          types.Int64PointerValue(m.SrcPort),
					},
				)
				diags.Append(d...)
				item.Matching = obj
			}
			if m := p.Marking; m != nil {
				obj, d := types.ObjectValueFrom(ctx, portQOSMarkingAttrTypes(), portQOSMarkingModel{
					CosCode:          types.Int64PointerValue(m.CosCode),
					DscpCode:         types.Int64PointerValue(m.DscpCode),
					IPPrecedenceCode: types.Int64PointerValue(m.IPPrecedenceCode),
					Queue:            types.Int64PointerValue(m.Queue),
				})
				diags.Append(d...)
				item.Marking = obj
			}
			items = append(items, item)
		}
		list, d := types.ListValueFrom(ctx, policyType, items)
		diags.Append(d...)
		policies = list
	} else if p, ok, _ := util.ObjectAs[portQOSProfileModel](ctx, prior); ok &&
		!p.Policies.IsNull() && !p.Policies.IsUnknown() && len(p.Policies.Elements()) == 0 {
		policies = types.ListValueMust(policyType, []attr.Value{})
	}

	obj, d := types.ObjectValueFrom(ctx, portQOSProfileAttrTypes(), portQOSProfileModel{
		Mode:     util.StringValueOrNull(q.Mode),
		Policies: policies,
	})
	diags.Append(d...)
	return obj, diags
}

// portQOSProfileFromObject converts a known `qos_profile` object to the
// API-neutral form. ok is false for a null or unknown object, which sends
// nothing (the controller keeps its profile).
func portQOSProfileFromObject(
	ctx context.Context,
	obj types.Object,
) (*portQOSProfileAPI, bool, diag.Diagnostics) {
	m, ok, diags := util.ObjectAs[portQOSProfileModel](ctx, obj)
	if !ok || diags.HasError() {
		return nil, false, diags
	}
	out := &portQOSProfileAPI{}
	if !m.Mode.IsNull() && !m.Mode.IsUnknown() {
		out.Mode = m.Mode.ValueString()
	}
	if !m.Policies.IsNull() && !m.Policies.IsUnknown() {
		var items []portQOSPolicyModel
		diags.Append(m.Policies.ElementsAs(ctx, &items, false)...)
		for _, item := range items {
			var p portQOSPolicyAPI
			if mm, ok, d := util.ObjectAs[portQOSMatchingModel](ctx, item.Matching); ok {
				p.Matching = &portQOSMatchingAPI{
					CosCode:          util.ConvertInt64ToAPIValue(mm.CosCode),
					DscpCode:         util.ConvertInt64ToAPIValue(mm.DscpCode),
					DstPort:          util.ConvertInt64ToAPIValue(mm.DstPort),
					IPPrecedenceCode: util.ConvertInt64ToAPIValue(mm.IPPrecedenceCode),
					SrcPort:          util.ConvertInt64ToAPIValue(mm.SrcPort),
				}
				if !mm.Protocol.IsNull() && !mm.Protocol.IsUnknown() {
					p.Matching.Protocol = mm.Protocol.ValueString()
				}
			} else {
				diags.Append(d...)
			}
			if mk, ok, d := util.ObjectAs[portQOSMarkingModel](ctx, item.Marking); ok {
				p.Marking = &portQOSMarkingAPI{
					CosCode:          util.ConvertInt64ToAPIValue(mk.CosCode),
					DscpCode:         util.ConvertInt64ToAPIValue(mk.DscpCode),
					IPPrecedenceCode: util.ConvertInt64ToAPIValue(mk.IPPrecedenceCode),
					Queue:            util.ConvertInt64ToAPIValue(mk.Queue),
				}
			} else {
				diags.Append(d...)
			}
			out.Policies = append(out.Policies, p)
		}
	}
	return out, true, diags
}

// portProfileQOSToAPI and portProfileQOSFromAPI convert between the neutral
// form and go-unifi's PortProfileQOSProfile.
func portProfileQOSToAPI(q *portQOSProfileAPI) *unifi.PortProfileQOSProfile {
	if q == nil {
		return nil
	}
	out := &unifi.PortProfileQOSProfile{QOSProfileMode: q.Mode}
	for _, p := range q.Policies {
		var pp unifi.PortProfileQOSPolicies
		if m := p.Matching; m != nil {
			pp.QOSMatching = &unifi.PortProfileQOSMatching{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				DstPort:          m.DstPort,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Protocol:         m.Protocol,
				SrcPort:          m.SrcPort,
			}
		}
		if m := p.Marking; m != nil {
			pp.QOSMarking = &unifi.PortProfileQOSMarking{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Queue:            m.Queue,
			}
		}
		out.QOSPolicies = append(out.QOSPolicies, pp)
	}
	return out
}

func portProfileQOSFromAPI(q *unifi.PortProfileQOSProfile) *portQOSProfileAPI {
	if q == nil {
		return nil
	}
	out := &portQOSProfileAPI{Mode: q.QOSProfileMode}
	for _, p := range q.QOSPolicies {
		var np portQOSPolicyAPI
		if m := p.QOSMatching; m != nil {
			np.Matching = &portQOSMatchingAPI{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				DstPort:          m.DstPort,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Protocol:         m.Protocol,
				SrcPort:          m.SrcPort,
			}
		}
		if m := p.QOSMarking; m != nil {
			np.Marking = &portQOSMarkingAPI{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Queue:            m.Queue,
			}
		}
		out.Policies = append(out.Policies, np)
	}
	return out
}

// planConfiguredObject plans the object at p exactly as configuration writes
// it, when configuration sets it. For a nested attribute whose leaves are
// Computed (so that an unset object keeps the controller's value, see
// portQOSProfileSchema), Terraform would otherwise plan a leaf that
// configuration leaves out with its prior value, and the object could never
// lose a field or be cleared with {}. A null or unknown configuration is left
// to the attribute's plan modifiers.
func planConfiguredObject(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	p path.Path,
) {
	var configured types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &configured)...)
	if resp.Diagnostics.HasError() || configured.IsNull() || configured.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, configured)...)
}
