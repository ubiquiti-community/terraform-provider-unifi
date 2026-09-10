package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &networkResource{}
	_ resource.ResourceWithImportState  = &networkResource{}
	_ resource.ResourceWithIdentity     = &networkResource{}
	_ resource.ResourceWithModifyPlan   = &networkResource{}
	_ resource.ResourceWithUpgradeState = &networkResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &networkResource{}
	_ list.ListResourceWithConfigure = &networkResource{}
)

func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

func NewNetworkListResource() list.ListResource {
	return &networkResource{}
}

// networkResource defines the resource implementation.
type networkResource struct {
	client *Client
}

type networkIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

// networkListConfigModel describes the list configuration model.
type networkListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// networkListFilterModel represents a single name/value filter entry.
type networkListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// dhcpBootModel describes the DHCP boot configuration.
type dhcpBootModel struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	Server   types.String `tfsdk:"server"`
	Filename types.String `tfsdk:"filename"`
}

func (m dhcpBootModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":  types.BoolType,
		"server":   types.StringType,
		"filename": types.StringType,
	}
}

// winsModel describes the WINS configuration.
type winsModel struct {
	Enabled   types.Bool `tfsdk:"enabled"`
	Addresses types.List `tfsdk:"addresses"`
}

func (m winsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":   types.BoolType,
		"addresses": types.ListType{ElemType: types.StringType},
	}
}

// dhcpServerOptionModel describes a DHCP option that hands a server list to
// clients: `dhcp_server.dns` and `dhcp_server.ntp`.
type dhcpServerOptionModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Servers types.List `tfsdk:"servers"`
}

func (m dhcpServerOptionModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// dhcpServerOptionDefault reproduces what the flat dns_enabled/ntp_enabled
// (default false) and dns_servers/ntp_servers (unset) attributes planned when
// the practitioner left them out, so the request body is unchanged.
func dhcpServerOptionDefault() types.Object {
	return types.ObjectValueMust(dhcpServerOptionModel{}.AttributeTypes(), map[string]attr.Value{
		"enabled": types.BoolValue(false),
		"servers": types.ListNull(types.StringType),
	})
}

// dhcpServerModel describes the DHCP server configuration.
type dhcpServerModel struct {
	Boot              types.Object         `tfsdk:"boot"`
	Enabled           types.Bool           `tfsdk:"enabled"`
	Start             types.String         `tfsdk:"start"`
	Stop              types.String         `tfsdk:"stop"`
	GatewayEnabled    types.Bool           `tfsdk:"gateway_enabled"`
	ConflictChecking  types.Bool           `tfsdk:"conflict_checking"`
	Ntp               types.Object         `tfsdk:"ntp"`
	TimeOffsetEnabled types.Bool           `tfsdk:"time_offset_enabled"`
	Dns               types.Object         `tfsdk:"dns"`
	Leasetime         timetypes.GoDuration `tfsdk:"leasetime"`
	Wins              types.Object         `tfsdk:"wins"`
	WpadUrl           types.String         `tfsdk:"wpad_url"`
	TftpServer        types.String         `tfsdk:"tftp_server"`
	UnifiController   types.String         `tfsdk:"unifi_controller"`
}

func (m dhcpServerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"boot":              types.ObjectType{AttrTypes: dhcpBootModel{}.AttributeTypes()},
		"enabled":           types.BoolType,
		"start":             types.StringType,
		"stop":              types.StringType,
		"gateway_enabled":   types.BoolType,
		"conflict_checking": types.BoolType,
		"ntp": types.ObjectType{
			AttrTypes: dhcpServerOptionModel{}.AttributeTypes(),
		},
		"time_offset_enabled": types.BoolType,
		"dns": types.ObjectType{
			AttrTypes: dhcpServerOptionModel{}.AttributeTypes(),
		},
		"leasetime":        timetypes.GoDurationType{},
		"wins":             types.ObjectType{AttrTypes: winsModel{}.AttributeTypes()},
		"wpad_url":         types.StringType,
		"tftp_server":      types.StringType,
		"unifi_controller": types.StringType,
	}
}

type natOutboundIPAddressesModel struct {
	IPAddress       types.String `tfsdk:"ip_address"`        // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	IPAddressPool   types.List   `tfsdk:"ip_address_pool"`   // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])-(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$
	Mode            types.String `tfsdk:"mode"`              // all|ip_address|ip_address_pool
	WANNetworkGroup types.String `tfsdk:"wan_network_group"` // WAN[2-9]?
}

func (d natOutboundIPAddressesModel) AttributeTypes() map[string]attr.Type {
	return natOutboundIPAddresses()
}

func natOutboundIPAddresses() map[string]attr.Type {
	return map[string]attr.Type{
		"ip_address":        types.StringType,
		"ip_address_pool":   types.ListType{ElemType: types.StringType},
		"mode":              types.StringType,
		"wan_network_group": types.StringType,
	}
}

// dhcpGuardingModel describes the DHCP guarding configuration.
type dhcpGuardingModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Servers types.List `tfsdk:"servers"`
}

func (m dhcpGuardingModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// dhcpRelayModel describes the DHCP relay configuration.
type dhcpRelayModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Servers types.List `tfsdk:"servers"`
}

func (d dhcpRelayModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// dhcpV6DNSModel describes the `dhcp_v6_server.dns` nested object.
type dhcpV6DNSModel struct {
	Auto    types.Bool `tfsdk:"auto"`
	Servers types.List `tfsdk:"servers"`
}

func (m dhcpV6DNSModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"auto":    types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// dhcpV6DNSDefault reproduces the flat dns_auto (default false) and
// dns_servers (unset) defaults.
func dhcpV6DNSDefault() types.Object {
	return types.ObjectValueMust(dhcpV6DNSModel{}.AttributeTypes(), map[string]attr.Value{
		"auto":    types.BoolValue(false),
		"servers": types.ListNull(types.StringType),
	})
}

// dhcpV6ServerModel describes the DHCPv6 server configuration.
type dhcpV6ServerModel struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	DNS     types.Object `tfsdk:"dns"`
	Lease   types.Int64  `tfsdk:"lease"`
	Start   types.String `tfsdk:"start"`
	Stop    types.String `tfsdk:"stop"`
}

func (m dhcpV6ServerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"dns":     types.ObjectType{AttrTypes: dhcpV6DNSModel{}.AttributeTypes()},
		"lease":   types.Int64Type,
		"start":   types.StringType,
		"stop":    types.StringType,
	}
}

// networkIPv6Model is the `ipv6` nested object.
type networkIPv6Model struct {
	InterfaceType           types.String `tfsdk:"interface_type"`
	ClientAddressAssignment types.String `tfsdk:"client_address_assignment"`
	StaticSubnet            types.String `tfsdk:"static_subnet"`
	Aliases                 types.List   `tfsdk:"aliases"`
	RA                      types.Object `tfsdk:"ra"`
	PD                      types.Object `tfsdk:"pd"`
}

func networkIPv6AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interface_type":            types.StringType,
		"client_address_assignment": types.StringType,
		"static_subnet":             types.StringType,
		"aliases":                   types.ListType{ElemType: types.StringType},
		"ra":                        types.ObjectType{AttrTypes: networkIPv6RAAttrTypes()},
		"pd":                        types.ObjectType{AttrTypes: networkIPv6PDAttrTypes()},
	}
}

// networkIPv6RAModel is the `ipv6.ra` (router advertisement) nested object.
type networkIPv6RAModel struct {
	Enabled           types.Bool           `tfsdk:"enabled"`
	Priority          types.String         `tfsdk:"priority"`
	PreferredLifetime timetypes.GoDuration `tfsdk:"preferred_lifetime"`
	ValidLifetime     timetypes.GoDuration `tfsdk:"valid_lifetime"`
}

func networkIPv6RAAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":            types.BoolType,
		"priority":           types.StringType,
		"preferred_lifetime": timetypes.GoDurationType{},
		"valid_lifetime":     timetypes.GoDurationType{},
	}
}

// networkIPv6PDModel is the `ipv6.pd` (prefix delegation) nested object.
type networkIPv6PDModel struct {
	Interface           types.String `tfsdk:"interface"`
	Prefixid            types.String `tfsdk:"prefixid"`
	Start               types.String `tfsdk:"start"`
	Stop                types.String `tfsdk:"stop"`
	AutoPrefixidEnabled types.Bool   `tfsdk:"auto_prefixid_enabled"`
}

func networkIPv6PDAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interface":             types.StringType,
		"prefixid":              types.StringType,
		"start":                 types.StringType,
		"stop":                  types.StringType,
		"auto_prefixid_enabled": types.BoolType,
	}
}

// The ipv6 group deliberately has no schema Default. The framework applies
// attribute Defaults before its "did the plan change?" gate, and an object
// Default carrying computed leaves can never equal the prior state, so it
// would trip that gate on every plan and re-plan the resource's other
// Computed attributes (firewall_zone_id) as unknown forever. Instead the
// object and its ra/pd children use UseStateForUnknown, and ModifyPlan
// reproduces what the flat ipv6_* attributes planned when the practitioner
// left them out (see planIPv6Defaults): interface_type defaults to "none",
// and on create the remaining leaves take the shapes below, exactly as their
// flat predecessors did (Optional-only leaves null, Computed leaves unknown).

func networkIPv6RAPlanShape() types.Object {
	return types.ObjectValueMust(networkIPv6RAAttrTypes(), map[string]attr.Value{
		"enabled":            types.BoolUnknown(),
		"priority":           types.StringUnknown(),
		"preferred_lifetime": timetypes.NewGoDurationUnknown(),
		"valid_lifetime":     timetypes.NewGoDurationUnknown(),
	})
}

func networkIPv6PDPlanShape() types.Object {
	return types.ObjectValueMust(networkIPv6PDAttrTypes(), map[string]attr.Value{
		"interface":             types.StringNull(),
		"prefixid":              types.StringNull(),
		"start":                 types.StringUnknown(),
		"stop":                  types.StringUnknown(),
		"auto_prefixid_enabled": types.BoolUnknown(),
	})
}

func networkIPv6PlanShape() types.Object {
	return types.ObjectValueMust(networkIPv6AttrTypes(), map[string]attr.Value{
		"interface_type":            types.StringValue("none"),
		"client_address_assignment": types.StringUnknown(),
		"static_subnet":             types.StringNull(),
		"aliases":                   types.ListNull(types.StringType),
		"ra":                        networkIPv6RAPlanShape(),
		"pd":                        networkIPv6PDPlanShape(),
	})
}

// objectOfLeaves returns a known object whose every attribute is the null
// (unknown=false) or unknown (unknown=true) value of its type. It stands in
// for a null or unknown previous object where read code preserves leaves one
// by one, so each leaf is treated exactly as its flat predecessor was.
func objectOfLeaves(
	ctx context.Context,
	attrTypes map[string]attr.Type,
	unknown bool,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	vals := make(map[string]attr.Value, len(attrTypes))
	for name, t := range attrTypes {
		var raw any
		if unknown {
			raw = tftypes.UnknownValue
		}
		v, err := t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), raw))
		if err != nil {
			diags.AddError("Unable to build attribute value", err.Error())
			return types.ObjectNull(attrTypes), diags
		}
		vals[name] = v
	}
	obj, d := types.ObjectValue(attrTypes, vals)
	diags.Append(d...)
	return obj, diags
}

// networkResourceModel describes the resource data model.
type networkResourceModel struct {
	ID                     types.String         `tfsdk:"id"`
	Site                   types.String         `tfsdk:"site"`
	Enabled                types.Bool           `tfsdk:"enabled"`
	Name                   types.String         `tfsdk:"name"`
	NatOutboundIPAddresses types.List           `tfsdk:"nat_outbound_ip_addresses"`
	AutoScale              types.Bool           `tfsdk:"auto_scale"`
	Subnet                 cidrtypes.IPv4Prefix `tfsdk:"subnet"`
	DomainName             types.String         `tfsdk:"domain_name"`
	Vlan                   types.Int64          `tfsdk:"vlan"`
	NetworkIsolation       types.Bool           `tfsdk:"network_isolation"`
	SettingPreference      types.String         `tfsdk:"setting_preference"`
	InternetAccess         types.Bool           `tfsdk:"internet_access"`
	IgmpSnooping           types.Bool           `tfsdk:"igmp_snooping"`
	MulticastDNS           types.Bool           `tfsdk:"multicast_dns"`
	GatewayType            types.String         `tfsdk:"gateway_type"`
	IPv6                   types.Object         `tfsdk:"ipv6"`
	LteLan                 types.Bool           `tfsdk:"lte_lan"`
	IPAliases              types.List           `tfsdk:"ip_aliases"`
	ThirdPartyGateway      types.Bool           `tfsdk:"third_party_gateway"`
	Purpose                types.String         `tfsdk:"purpose"`
	DhcpGuarding           types.Object         `tfsdk:"dhcp_guarding"`
	DhcpServer             types.Object         `tfsdk:"dhcp_server"`
	DhcpV6Server           types.Object         `tfsdk:"dhcp_v6_server"`
	DhcpRelay              types.Object         `tfsdk:"dhcp_relay"`
	FirewallZoneID         types.String         `tfsdk:"firewall_zone_id"`
	Timeouts               timeouts.Value       `tfsdk:"timeouts"`
}

func (r *networkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *networkResource) IdentitySchema(
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

func (r *networkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		// v1: leasetime, ipv6_ra_preferred_lifetime and ipv6_ra_valid_lifetime
		// changed from Int64 (seconds) to GoDuration strings.
		// v2: the flat ipv6_* attributes moved under `ipv6` (with `ra` and `pd`
		// sub-objects), dhcp_server.dns_*/ntp_* under `dhcp_server.dns`/`ntp`,
		// and dhcp_v6_server.dns_* under `dhcp_v6_server.dns`. See UpgradeState.
		Version:             2,
		MarkdownDescription: "`unifi_network` manages networks (VLANs) in the UniFi controller.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the network with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the network is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the network.",
				Required:            true,
			},
			"nat_outbound_ip_addresses": schema.ListNestedAttribute{
				MarkdownDescription: "List of NAT outbound IP addresses.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The IP address.",
							Optional:            true,
						},
						"ip_address_pool": schema.ListAttribute{
							MarkdownDescription: "The IP address pool.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"mode": schema.StringAttribute{
							MarkdownDescription: "The mode.",
							Optional:            true,
						},
						"wan_network_group": schema.StringAttribute{
							MarkdownDescription: "The WAN network group.",
							Optional:            true,
						},
					},
				},
			},
			"auto_scale": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether auto-scaling is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The network's gateway IP and prefix in CIDR notation. The host " +
					"portion is the gateway address the controller assigns — it need not be the first " +
					"usable address: `10.0.10.1/24` uses gateway `10.0.10.1`, while `10.0.10.254/24` " +
					"uses gateway `10.0.10.254` on the same subnet. Optional: it is not required for " +
					"`vlan_only` networks (`third_party_gateway = true`), where the UniFi controller " +
					"does not manage the subnet.",
				Optional:   true,
				CustomType: cidrtypes.IPv4PrefixType{},
			},
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "The domain name for the network.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					validators.DomainNameValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "The VLAN ID for the network.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 4094),
				},
			},
			"network_isolation": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether network isolation is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"setting_preference": schema.StringAttribute{
				MarkdownDescription: "Setting preference. Must be one of `auto` or `manual`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("auto"),
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
			},
			"internet_access": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether internet access is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"igmp_snooping": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether IGMP snooping is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"multicast_dns": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether mDNS is enabled. This is " +
					"read back from the controller rather than defaulted: some " +
					"controllers (notably UniFi OS gateways) ignore `mdns_enabled` " +
					"at create/update time and always store `false`, so forcing a " +
					"`true` default produced a \"provider produced inconsistent " +
					"result after apply\" error.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway_type": schema.StringAttribute{
				MarkdownDescription: "The gateway type. Must be one of `default` or `switch`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				Validators: []validator.String{
					stringvalidator.OneOf("default", "switch"),
				},
			},
			"ipv6": schema.SingleNestedAttribute{
				MarkdownDescription: "IPv6 settings for the network. When omitted, " +
					"`interface_type` defaults to `none` and the remaining values are " +
					"read from the controller.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"interface_type": schema.StringAttribute{
						MarkdownDescription: "Specifies which type of IPv6 connection to use. Must be one of `none`, `pd`, or `static`. Defaults to `none` when not set.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("none", "pd", "static"),
						},
					},
					"client_address_assignment": schema.StringAttribute{
						MarkdownDescription: "How clients on this network obtain an IPv6 address (UI: Networks → IPv6 → Client Address Assignment). One of `slaac` (SLAAC only), `dhcpv6` (DHCPv6 only), or `slaac-dhcpv6` (both). Computed from the controller when not set.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("slaac", "dhcpv6", "slaac-dhcpv6"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"static_subnet": schema.StringAttribute{
						MarkdownDescription: "The IPv6 static subnet of the network. Only used when `interface_type` is `static`.",
						Optional:            true,
					},
					"aliases": schema.ListAttribute{
						MarkdownDescription: "List of IPv6 aliases for the network. Not currently supported: " +
							"the underlying UniFi API client has no field for this value, so a " +
							"non-empty list is rejected at plan time (#413).",
						Optional:    true,
						ElementType: types.StringType,
					},
					"ra": schema.SingleNestedAttribute{
						MarkdownDescription: "IPv6 Router Advertisement (RA) settings.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether IPv6 Router Advertisement (RA) is enabled.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"priority": schema.StringAttribute{
								MarkdownDescription: "The IPv6 Router Advertisement priority. Must be one of `high`, `medium`, or `low`.",
								Optional:            true,
								Computed:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("high", "medium", "low"),
								},
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"preferred_lifetime": schema.StringAttribute{
								MarkdownDescription: "The IPv6 Router Advertisement preferred lifetime, as a Go " +
									"duration string (e.g. `14400s`, `4h`). Must be a whole number of seconds " +
									"between `0s` and `31536000s` (1 year).",
								CustomType: timetypes.GoDurationType{},
								Optional:   true,
								Computed:   true,
								Validators: []validator.String{
									validators.GoDurationBetween(0, 31536000*time.Second),
									validators.GoDurationMultipleOf(time.Second),
								},
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"valid_lifetime": schema.StringAttribute{
								MarkdownDescription: "The IPv6 Router Advertisement valid lifetime, as a Go " +
									"duration string (e.g. `86400s`, `24h`). Must be a whole number of seconds " +
									"between `0s` and `31536000s` (1 year).",
								CustomType: timetypes.GoDurationType{},
								Optional:   true,
								Computed:   true,
								Validators: []validator.String{
									validators.GoDurationBetween(0, 31536000*time.Second),
									validators.GoDurationMultipleOf(time.Second),
								},
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"pd": schema.SingleNestedAttribute{
						MarkdownDescription: "IPv6 Prefix Delegation (PD) settings, used when `interface_type` is `pd`.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"interface": schema.StringAttribute{
								MarkdownDescription: "The IPv6 Prefix Delegation WAN interface (e.g., `wan`, `wan2`).",
								Optional:            true,
							},
							"prefixid": schema.StringAttribute{
								MarkdownDescription: "The IPv6 Prefix Delegation prefix ID (hex string, e.g., `0`, `1a`).",
								Optional:            true,
							},
							"start": schema.StringAttribute{
								MarkdownDescription: "The start of the IPv6 Prefix Delegation range (e.g. `::2`). " +
									"Required together with `stop` when `ipv6.interface_type` is " +
									"`pd`, otherwise the controller rejects the network with " +
									"`api.err.InvalidIpv6Addr`.",
								Optional: true,
								Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"stop": schema.StringAttribute{
								MarkdownDescription: "The end of the IPv6 Prefix Delegation range (e.g. `::7d1`). " +
									"Required together with `start` when `ipv6.interface_type` is `pd`.",
								Optional: true,
								Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"auto_prefixid_enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether automatic prefix ID assignment is enabled for IPv6 Prefix Delegation.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
			"lte_lan": schema.BoolAttribute{
				MarkdownDescription: "Whether this network/VLAN stays active when the " +
					"gateway fails over to a UniFi LTE (cellular) backup WAN. Maps to " +
					"the controller's `lte_lan_enabled` flag and only matters when a " +
					"UniFi LTE failover device is in use; otherwise it is cosmetic. " +
					"Defaults to `true` (network stays available during LTE failover); " +
					"set to `false` to disable it while on the LTE backup link. The " +
					"controller may set this automatically, which is why existing " +
					"networks can show differing values.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"ip_aliases": schema.ListAttribute{
				MarkdownDescription: "List of IP aliases for the network, in CIDR notation " +
					"(e.g. `192.168.2.1/24`). The controller rejects entries without a " +
					"prefix length.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"third_party_gateway": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this network uses a third-party gateway. When enabled, the network purpose is set to `vlan-only` and only VLAN ID, DHCP guarding, and basic network settings are configured.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"purpose": schema.StringAttribute{
				MarkdownDescription: "The network purpose: `corporate` (default), `guest`, or `vlan-only`. Leave unset to let the controller manage it (a `third_party_gateway` network is always `vlan-only`). **Note:** on Zone-Based-Firewall controllers the purpose is coupled to the firewall zone — a `guest` network only keeps `purpose = \"guest\"` while it belongs to the guest/Hotspot zone (assign it there via `unifi_firewall_zone`), otherwise the controller rewrites it back to `corporate` and the apply fails with an inconsistent-result error.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						unifi.PurposeCorporate,
						unifi.PurposeGuest,
						unifi.PurposeVLANOnly,
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dhcp_guarding": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP guarding configuration. Specifies allowed DHCP server IPs to prevent rogue DHCP servers on the network.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP guarding is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "List of allowed DHCP server IP addresses (maximum 3). " +
							"On `corporate` and `guest` networks the controller only honors " +
							"DHCP guarding with `setting_preference = \"manual\"`; when " +
							"`setting_preference` is not configured, the provider sets it to " +
							"`manual` automatically whenever `dhcp_guarding.enabled` is `true`.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtMost(3),
						},
					},
				},
			},
			"dhcp_server": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP server configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"boot": schema.SingleNestedAttribute{
						MarkdownDescription: "DHCP boot settings.",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Toggles DHCP boot options.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"server": schema.StringAttribute{
								MarkdownDescription: "TFTP server for boot options.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"filename": schema.StringAttribute{
								MarkdownDescription: "Boot filename.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP server is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"start": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address where the DHCP range starts.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"stop": schema.StringAttribute{
						MarkdownDescription: "The IPv4 address where the DHCP range stops.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"gateway_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP gateway is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"conflict_checking": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP conflict checking is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"ntp": schema.SingleNestedAttribute{
						MarkdownDescription: "NTP servers handed out to DHCP clients.",
						Optional:            true,
						Computed:            true,
						Default:             objectdefault.StaticValue(dhcpServerOptionDefault()),
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP NTP is enabled.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"servers": schema.ListAttribute{
								MarkdownDescription: "List of NTP server addresses for DHCP clients.",
								Optional:            true,
								ElementType:         types.StringType,
								Validators: []validator.List{
									listvalidator.SizeAtMost(2),
									listvalidator.ValueStringsAre(
										stringvalidator.Any(
											validators.IPv4Validator(),
											validators.IPv6Validator(),
										),
									),
								},
							},
						},
					},
					"time_offset_enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP time offset is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"dns": schema.SingleNestedAttribute{
						MarkdownDescription: "DNS servers handed out to DHCP clients.",
						Optional:            true,
						Computed:            true,
						Default:             objectdefault.StaticValue(dhcpServerOptionDefault()),
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP DNS is enabled.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"servers": schema.ListAttribute{
								MarkdownDescription: "List of DNS server addresses for DHCP clients.",
								Optional:            true,
								ElementType:         types.StringType,
								Validators: []validator.List{
									listvalidator.SizeAtMost(4),
								},
							},
						},
					},
					"leasetime": schema.StringAttribute{
						MarkdownDescription: "Specifies the DHCP lease time, as a Go duration " +
							"string (e.g. `24h`, `86400s`). Defaults to `24h0m0s`.",
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
						Default:    stringdefault.StaticString("24h0m0s"),
					},
					"wins": schema.SingleNestedAttribute{
						MarkdownDescription: "WINS server configuration.",
						Optional:            true,
						Computed:            true,
						Default: objectdefault.StaticValue(
							types.ObjectValueMust(map[string]attr.Type{
								"enabled":   types.BoolType,
								"addresses": types.ListType{ElemType: types.StringType},
							}, map[string]attr.Value{
								"enabled":   types.BoolValue(false),
								"addresses": types.ListNull(types.StringType),
							}),
						),
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DHCP WINS is enabled.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"addresses": schema.ListAttribute{
								MarkdownDescription: "List of WINS server addresses (maximum 2).",
								Optional:            true,
								ElementType:         types.StringType,
								Validators: []validator.List{
									listvalidator.SizeAtMost(2),
								},
							},
						},
					},
					"wpad_url": schema.StringAttribute{
						MarkdownDescription: "WPAD URL for proxy auto-configuration.",
						Optional:            true,
					},
					"tftp_server": schema.StringAttribute{
						MarkdownDescription: "TFTP server address.",
						Optional:            true,
					},
					"unifi_controller": schema.StringAttribute{
						MarkdownDescription: "UniFi controller IP address.",
						Optional:            true,
						Validators: []validator.String{
							validators.IPv4Validator(),
						},
					},
				},
			},
			"dhcp_v6_server": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCPv6 server configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether the DHCPv6 server is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"dns": schema.SingleNestedAttribute{
						MarkdownDescription: "DNS servers handed out to DHCPv6 clients.",
						Optional:            true,
						Computed:            true,
						Default:             objectdefault.StaticValue(dhcpV6DNSDefault()),
						Attributes: map[string]schema.Attribute{
							"auto": schema.BoolAttribute{
								MarkdownDescription: "Specifies whether DNS auto-discovery is enabled for DHCPv6.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"servers": schema.ListAttribute{
								MarkdownDescription: "List of DNS server addresses for DHCPv6 clients (maximum 4).",
								Optional:            true,
								ElementType:         types.StringType,
								Validators: []validator.List{
									listvalidator.SizeAtMost(4),
								},
							},
						},
					},
					"lease": schema.Int64Attribute{
						MarkdownDescription: "The lease time for DHCPv6 addresses in seconds.",
						Optional:            true,
					},
					"start": schema.StringAttribute{
						MarkdownDescription: "The start of the DHCPv6 address range.",
						Optional:            true,
					},
					"stop": schema.StringAttribute{
						MarkdownDescription: "The end of the DHCPv6 address range.",
						Optional:            true,
					},
				},
			},
			"dhcp_relay": schema.SingleNestedAttribute{
				MarkdownDescription: "DHCP relay configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether DHCP relay is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "List of DHCP relay server addresses.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtMost(4),
						},
					},
				},
			},
			"firewall_zone_id": schema.StringAttribute{
				MarkdownDescription: "The firewall zone ID assigned to this network. " +
					"Note: This field is dual-managed and can compete with `unifi_firewall_zone.network_ids`. " +
					"To prevent state drift loops, ensure you manage zone membership from exactly one side. " +
					"On Zone-Based Firewall (ZBF) controllers, this field is tightly coupled to the network's `purpose` field.",
				Optional: true,
				Computed: true,
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

// UpgradeState migrates prior schema versions to the current one:
//
//	v0 -> current: leasetime (nested in dhcp_server), ipv6_ra_preferred_lifetime
//	    and ipv6_ra_valid_lifetime changed from integer seconds to GoDuration
//	    strings.
//	v1 -> current: the flat ipv6_* attributes moved under `ipv6` (with `ra` and
//	    `pd` sub-objects), dhcp_server.dns_*/ntp_* under `dhcp_server.dns`/`ntp`
//	    and dhcp_v6_server.dns_* under `dhcp_v6_server.dns`. See nestNetworkState.
//
// Each upgrader targets the CURRENT schema type, so older state picks up every
// later change via the shared rewrite and reconciliation.
func (r *networkResource) UpgradeState(
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
						nestNetworkState(state)
					},
				)
				if err != nil {
					resp.Diagnostics.AddError("Failed to upgrade network state", err.Error())
					return
				}
				resp.DynamicValue = dv
			},
		}
	}

	return map[int64]resource.StateUpgrader{
		0: upgrader(func(state map[string]any) {
			util.SetDurationField(state, "ipv6_ra_preferred_lifetime", time.Second)
			util.SetDurationField(state, "ipv6_ra_valid_lifetime", time.Second)
			util.WithObject(state, "dhcp_server", func(dhcp map[string]any) {
				util.SetDurationField(dhcp, "leasetime", time.Second)
			})
		}),
		1: upgrader(func(map[string]any) {}),
	}
}

// nestNetworkState rewrites flat v0/v1 network state into the nested-object
// layout introduced in schema v2. Keys that are absent are skipped, so it is
// safe to run on state from any earlier version.
func nestNetworkState(state map[string]any) {
	util.NestFields(state, "ipv6", map[string]string{
		"ipv6_interface_type":            "interface_type",
		"ipv6_client_address_assignment": "client_address_assignment",
		"ipv6_static_subnet":             "static_subnet",
		"ipv6_aliases":                   "aliases",
		"ipv6_ra":                        "ra_enabled",
		"ipv6_ra_priority":               "ra_priority",
		"ipv6_ra_preferred_lifetime":     "ra_preferred_lifetime",
		"ipv6_ra_valid_lifetime":         "ra_valid_lifetime",
		"ipv6_pd_interface":              "pd_interface",
		"ipv6_pd_prefixid":               "pd_prefixid",
		"ipv6_pd_start":                  "pd_start",
		"ipv6_pd_stop":                   "pd_stop",
		"ipv6_pd_auto_prefixid_enabled":  "pd_auto_prefixid_enabled",
	})
	util.WithObject(state, "ipv6", func(v6 map[string]any) {
		util.NestFields(v6, "ra", map[string]string{
			"ra_enabled":            "enabled",
			"ra_priority":           "priority",
			"ra_preferred_lifetime": "preferred_lifetime",
			"ra_valid_lifetime":     "valid_lifetime",
		})
		util.NestFields(v6, "pd", map[string]string{
			"pd_interface":             "interface",
			"pd_prefixid":              "prefixid",
			"pd_start":                 "start",
			"pd_stop":                  "stop",
			"pd_auto_prefixid_enabled": "auto_prefixid_enabled",
		})
	})
	util.WithObject(state, "dhcp_server", func(dhcp map[string]any) {
		util.NestFields(dhcp, "dns", map[string]string{
			"dns_enabled": "enabled",
			"dns_servers": "servers",
		})
		util.NestFields(dhcp, "ntp", map[string]string{
			"ntp_enabled": "enabled",
			"ntp_servers": "servers",
		})
	})
	util.WithObject(state, "dhcp_v6_server", func(v6 map[string]any) {
		util.NestFields(v6, "dns", map[string]string{
			"dns_auto":    "auto",
			"dns_servers": "servers",
		})
	})
}

func (r *networkResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
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

// ModifyPlan rejects a configured ipv6.aliases (#413, unsupported by the
// underlying client) and forces setting_preference to "manual" when DHCP
// relay or DHCP guarding is enabled.
//
// With setting_preference "auto" the controller auto-manages the network:
// it re-enables its built-in DHCP server, which silently turns dhcp_relay
// off (the two cannot coexist), and it force-resets dhcpguard_enabled to
// false on every write (#419). Forcing "manual" makes the controller honor
// the explicit relay/guarding configuration. We only override the default;
// an explicit user-provided value is left untouched (with a warning for the
// unsatisfiable auto+guarding combination).
// planIPv6Defaults reproduces the flat ipv6_* defaults for the nested ipv6
// object without a schema Default (see the comment above
// networkIPv6RAPlanShape for why a Default cannot be used):
//
//   - interface_type defaults to "none" whenever the configuration does not
//     set it, whether the ipv6 block is present or omitted entirely;
//   - when the block is omitted and there is no prior object to carry
//     (create, or a state that never held one), the group takes its
//     create-time shape: Optional-only leaves null, Computed leaves unknown.
//
// On update with the block omitted, UseStateForUnknown has already restored
// the prior object, so only interface_type is (re)asserted here, exactly as
// the flat attribute's Default did.
func planIPv6Defaults(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) diag.Diagnostics {
	var diags diag.Diagnostics

	var configType types.String
	var planIPv6 types.Object
	diags.Append(req.Config.GetAttribute(
		ctx, path.Root("ipv6").AtName("interface_type"), &configType)...)
	diags.Append(req.Plan.GetAttribute(ctx, path.Root("ipv6"), &planIPv6)...)
	if diags.HasError() || !configType.IsNull() {
		// Configured (or unknown until apply): nothing to default.
		return diags
	}

	var configIPv6 types.Object
	diags.Append(req.Config.GetAttribute(ctx, path.Root("ipv6"), &configIPv6)...)
	if diags.HasError() || configIPv6.IsUnknown() {
		return diags
	}

	if planIPv6.IsNull() || planIPv6.IsUnknown() {
		planIPv6 = networkIPv6PlanShape()
	} else {
		attrs := planIPv6.Attributes()
		attrs["interface_type"] = types.StringValue("none")
		obj, d := types.ObjectValue(networkIPv6AttrTypes(), attrs)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		planIPv6 = obj
	}

	diags.Append(resp.Plan.SetAttribute(ctx, path.Root("ipv6"), planIPv6)...)
	return diags
}

func (r *networkResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return // resource is being destroyed
	}

	resp.Diagnostics.Append(planIPv6Defaults(ctx, req, resp)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// firewall_zone_id: Optional+Computed with no plan modifier, so on any
	// update the framework re-plans it as unknown when the config is null —
	// including updates manufactured purely by the setting_preference
	// default ("auto") flapping against a state pinned to "manual" below.
	// On controllers without zone-based firewalling the applied value is
	// always null, so the unknown never resolves to anything else, yet it
	// makes every follow-up plan non-empty (a perpetual diff for any relay
	// or guarding network). Pin the plan back to null when neither state nor
	// config carry a value; ZBF controllers assign a zone on first apply, so
	// a genuinely zone-managed network never has a null prior state here.
	if !req.State.Raw.IsNull() {
		var stateZone, configZone, planZone types.String
		resp.Diagnostics.Append(
			req.State.GetAttribute(ctx, path.Root("firewall_zone_id"), &stateZone)...)
		resp.Diagnostics.Append(
			req.Config.GetAttribute(ctx, path.Root("firewall_zone_id"), &configZone)...)
		resp.Diagnostics.Append(
			req.Plan.GetAttribute(ctx, path.Root("firewall_zone_id"), &planZone)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if stateZone.IsNull() && configZone.IsNull() && planZone.IsUnknown() {
			resp.Diagnostics.Append(
				resp.Plan.SetAttribute(
					ctx,
					path.Root("firewall_zone_id"),
					types.StringNull(),
				)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// ipv6.aliases: go-unifi's Network struct has no field for this yet, so a
	// configured value can never reach the controller. Fail fast at plan time
	// with a clear message instead of Create/Update silently dropping it and
	// producing a confusing "provider produced inconsistent result after
	// apply" error (#413).
	// Read from the plan (not config) so that unknown values derived from
	// data sources are caught here too. A wholly unknown ipv6 object hides
	// the aliases value, so it is treated as unknown aliases. The plan is
	// read from resp, where planIPv6Defaults above has already replaced the
	// unknown object an omitted block produces on create with its shape.
	aliasesPath := path.Root("ipv6").AtName("aliases")
	var ipv6Obj types.Object
	var ipv6Aliases types.List
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("ipv6"), &ipv6Obj)...)
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, aliasesPath, &ipv6Aliases)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if ipv6Obj.IsUnknown() || !ipv6Aliases.IsNull() {
		resp.Diagnostics.AddAttributeError(
			aliasesPath,
			"ipv6.aliases is not yet supported",
			"The underlying UniFi API client (go-unifi) does not currently expose "+
				"a field for ipv6.aliases, so the provider cannot send this value to "+
				"the controller even though the controller accepts and returns it. "+
				"Remove ipv6.aliases from this configuration (and make sure the "+
				"ipv6 object is known at plan time) until upstream client support "+
				"lands (see issue #413).",
		)
		return
	}

	// ip_address_pool inside nat_outbound_ip_addresses: the field is not yet
	// wired to the API request side (modelToNetwork ignores it) and Read always
	// writes null, which causes the same inconsistent-result-after-apply failure
	// as ipv6.aliases. Reject any non-null value at plan time.
	var natList types.List
	resp.Diagnostics.Append(
		req.Plan.GetAttribute(ctx, path.Root("nat_outbound_ip_addresses"), &natList)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !natList.IsNull() && !natList.IsUnknown() {
		for i, elem := range natList.Elements() {
			obj, ok := elem.(types.Object)
			if !ok {
				continue
			}
			if obj.IsNull() || obj.IsUnknown() {
				continue
			}
			poolAttr, poolOk := obj.Attributes()["ip_address_pool"]
			if poolOk && (!poolAttr.IsNull() || poolAttr.IsUnknown()) {
				resp.Diagnostics.AddAttributeError(
					path.Root("nat_outbound_ip_addresses").AtListIndex(i).AtName("ip_address_pool"),
					"ip_address_pool is not yet supported",
					"The ip_address_pool field inside nat_outbound_ip_addresses is not yet "+
						"wired to the API request side, so a configured value cannot be sent "+
						"to the controller and Read will always return null, causing an "+
						"inconsistent-result-after-apply error. Remove ip_address_pool from "+
						"this configuration until end-to-end support lands.",
				)
				return
			}
		}
	}

	// DHCP guarding on corporate/guest networks requires setting_preference =
	// "manual": with "auto" the controller auto-manages the network and
	// force-resets dhcpguard_enabled to false on write (verified against a
	// live controller: a POST carrying dhcpguard_enabled=true with
	// setting_preference="auto" is stored with dhcpguard_enabled=false for
	// purpose corporate and guest; vlan-only keeps it). This surfaced as
	// guarding silently dropped or as "provider produced inconsistent result
	// after apply" on .dhcp_guarding.enabled (#419). It is controller
	// behavior, not the historic go-unifi marshaling gap (go-unifi#68) —
	// current go-unifi serializes dhcpd_ip_1..3 for corporate, guest, and
	// vlan-only purposes alike.
	guardEnabled := false
	var guarding types.Object
	resp.Diagnostics.Append(
		req.Plan.GetAttribute(ctx, path.Root("dhcp_guarding"), &guarding)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !guarding.IsNull() && !guarding.IsUnknown() {
		var dg dhcpGuardingModel
		resp.Diagnostics.Append(guarding.As(ctx, &dg, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		guardEnabled = dg.Enabled.ValueBool()
	}

	// Effective purpose mirrors modelToNetwork: default corporate when unset,
	// third_party_gateway forces vlan-only. Only corporate/guest reset
	// guarding under "auto"; an unknown purpose is treated as at risk.
	guardAtRisk := false
	if guardEnabled {
		var purpose types.String
		resp.Diagnostics.Append(
			req.Plan.GetAttribute(ctx, path.Root("purpose"), &purpose)...)
		if resp.Diagnostics.HasError() {
			return
		}
		effectivePurpose := unifi.PurposeCorporate
		if !purpose.IsNull() && !purpose.IsUnknown() && purpose.ValueString() != "" {
			effectivePurpose = purpose.ValueString()
		}
		var thirdPartyGateway types.Bool
		resp.Diagnostics.Append(
			req.Plan.GetAttribute(ctx, path.Root("third_party_gateway"), &thirdPartyGateway)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !thirdPartyGateway.IsUnknown() && thirdPartyGateway.ValueBool() {
			effectivePurpose = unifi.PurposeVLANOnly
		}
		guardAtRisk = effectivePurpose == unifi.PurposeCorporate ||
			effectivePurpose == unifi.PurposeGuest
	}

	var configPref types.String
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("setting_preference"), &configPref)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !configPref.IsNull() {
		// The user set setting_preference explicitly: respect their choice,
		// but surface the unsatisfiable combination instead of letting apply
		// fail with a confusing inconsistent-result error.
		if configPref.ValueString() == "auto" && guardAtRisk {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("setting_preference"),
				"DHCP guarding is reset by the controller when setting_preference is \"auto\"",
				"The controller force-disables dhcpguard_enabled on any write to an "+
					"auto-managed network, so dhcp_guarding.enabled = true cannot take "+
					"effect and apply is likely to fail with \"Provider produced "+
					"inconsistent result after apply\". Set setting_preference = "+
					"\"manual\", or remove it so the provider manages it, to use DHCP "+
					"guarding (#419).",
			)
		}
		return
	}

	needManual := guardAtRisk

	var relay types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("dhcp_relay"), &relay)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !relay.IsNull() && !relay.IsUnknown() {
		var dr dhcpRelayModel
		resp.Diagnostics.Append(relay.As(ctx, &dr, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		needManual = needManual || dr.Enabled.ValueBool()
	}

	if !needManual {
		return
	}

	resp.Diagnostics.Append(
		resp.Plan.SetAttribute(
			ctx,
			path.Root("setting_preference"),
			types.StringValue("manual"),
		)...)
}

func (r *networkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data networkResourceModel

	// Read Terraform plan data into the model
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

	// Convert to unifi.Network
	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the network
	createdNetwork, err := r.client.CreateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating network",
			err.Error(),
		)
		return
	}

	// Convert back to model, passing the plan data to preserve null values
	var planData networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, createdNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	idModel := networkIdentityModel{ID: data.ID, Site: data.Site}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data networkResourceModel

	// Read Terraform prior state data into the model
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
	// identity support. When an identity comes in it must be passed through
	// unchanged: Terraform treats any modification of a non-null identity
	// (including filling a null attribute) as an error.
	haveIdentity := req.Identity != nil && !req.Identity.Raw.IsNull()
	var idModel networkIdentityModel
	if haveIdentity {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &idModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		idModel.ID = data.ID
		idModel.Site = data.Site
	}

	// Tolerate identity-only state (the refresh right after an identity-based
	// import): fill the missing lookup keys from identity.
	id := ""
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		id = data.ID.ValueString()
	}
	if id == "" {
		id = idModel.ID.ValueString()
	}

	site := data.Site.ValueString()
	if site == "" {
		site = idModel.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}

	var err error
	var network *unifi.Network

	if id != "" {
		// Get the network by ID
		network, err = r.client.GetNetwork(ctx, site, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading network",
				"Could not read network ID "+id+": "+err.Error(),
			)
			return
		}
	} else {
		// Get the network by name
		network, err = r.client.GetNetworkByName(ctx, site, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading network",
				"Could not read network name "+data.Name.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Convert to model, passing the current state to preserve null values
	var priorState networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.networkToModel(ctx, network, &data, site, &priorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state. A pre-existing identity is
	// re-set unchanged; a fresh one is derived from the refreshed state.
	if !haveIdentity {
		idModel.ID = data.ID
		idModel.Site = data.Site
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data networkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := data.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Convert to unifi.Network
	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	network.ID = data.ID.ValueString()

	// modelToNetwork zero-values the DHCP guarding fields when the
	// dhcp_guarding block is absent from configuration (and the DHCP server
	// option fields when dhcp_server is absent), and the controller treats
	// the resulting PUT literally — silently wiping settings configured
	// outside Terraform on every unrelated update. Preserve the controller's
	// current values instead.
	if data.DhcpGuarding.IsNull() || data.DhcpServer.IsNull() {
		current, err := r.client.GetNetwork(ctx, site, network.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating network",
				fmt.Sprintf(
					"Could not read the network to preserve its unmanaged DHCP settings: %s",
					err,
				),
			)
			return
		}
		preserveUnmanagedDhcpGuarding(data.DhcpGuarding, network, current)

		relayEnabled := false
		if !data.DhcpRelay.IsNull() && !data.DhcpRelay.IsUnknown() {
			var relay dhcpRelayModel
			resp.Diagnostics.Append(
				data.DhcpRelay.As(ctx, &relay, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}
			relayEnabled = relay.Enabled.ValueBool()
		}
		preserveUnmanagedDhcpServer(data.DhcpServer, relayEnabled, network, current)
	}

	// Update the network
	updatedNetwork, err := r.client.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating network",
			err.Error(),
		)
		return
	}

	// Convert back to model, passing the plan data to preserve null values
	var planData networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, updatedNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state. Identity is immutable once set:
	// carry the incoming identity through unchanged, deriving a fresh one from
	// state only when it was absent.
	idModel := networkIdentityModel{ID: data.ID, Site: data.Site}
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &idModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data networkResourceModel

	// Read Terraform prior state data into the model
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

	// Delete the network
	name := data.Name.ValueString()
	err := r.client.DeleteNetwork(ctx, site, data.ID.ValueString(), name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting network",
			err.Error(),
		)
		return
	}
}

func (r *networkResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import by ID string (terraform import CLI, or import block with id set).
	// Formats: "site:id", "name=<name>", a bare 24-hex controller ObjectID, or
	// a plain network name.
	if req.ID != "" {
		var site types.String

		idParts := strings.Split(req.ID, ":")
		if len(idParts) == 2 {
			site = types.StringValue(idParts[0])
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
			req.ID = idParts[1]
		}

		if strings.HasPrefix(req.ID, "name=") {
			req.ID = strings.TrimPrefix(req.ID, "name=")
			resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
		} else if regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(req.ID) {
			idModel := networkIdentityModel{ID: types.StringValue(req.ID), Site: site}
			resp.Diagnostics.Append(
				resp.State.SetAttribute(ctx, path.Root("id"), idModel.ID)...)
			resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
		} else {
			// Fall back to importing by name.
			resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
		}
		return
	}

	// Import by resource identity (import block with identity, Terraform 1.12+).
	var idModel networkIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &idModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idModel.ID)...)
	if !idModel.Site.IsNull() && idModel.Site.ValueString() != "" {
		resp.Diagnostics.Append(
			resp.State.SetAttribute(ctx, path.Root("site"), idModel.Site)...)
	}
}

// preserveUnmanagedDhcpGuarding copies the controller's current DHCP guarding
// fields onto an outgoing network update when the configuration does not
// manage the dhcp_guarding block (planned is null). Without this, an update
// built from the model alone carries dhcpguard_enabled=false and empty
// dhcpd_ip_1..3, disabling guarding that was configured outside Terraform.
// Returns true when the fields were preserved.
func preserveUnmanagedDhcpGuarding(
	planned types.Object,
	network *unifi.Network,
	current *unifi.Network,
) bool {
	if !planned.IsNull() {
		return false
	}
	network.DHCPguardEnabled = current.DHCPguardEnabled
	network.DHCPDIP1 = current.DHCPDIP1
	network.DHCPDIP2 = current.DHCPDIP2
	network.DHCPDIP3 = current.DHCPDIP3
	return true
}

// preserveUnmanagedDhcpServer copies the controller's current DHCP server
// option fields onto an outgoing network update when the configuration does
// not manage the dhcp_server block (planned is null). modelToNetwork
// zero-fills these fields in that case, and go-unifi serializes them for
// corporate/guest networks (since go-unifi#73 that includes empty
// dhcpd_dns_1..4 and pointer-to-empty dhcpd_ntp_1..2, so an empty value now
// actively clears the slot). Without preserving, any unrelated update would
// reset the controller's DHCP configuration — range, lease time, DNS, NTP,
// WINS, boot options — configured outside Terraform. Skipped when DHCP relay
// is enabled: relay requires the built-in DHCP server disabled, and
// modelToNetwork's zero-filling is intentional there. Returns true when the
// fields were preserved.
func preserveUnmanagedDhcpServer(
	planned types.Object,
	relayEnabled bool,
	network *unifi.Network,
	current *unifi.Network,
) bool {
	if !planned.IsNull() || relayEnabled {
		return false
	}
	network.DHCPDEnabled = current.DHCPDEnabled
	network.DHCPDStart = current.DHCPDStart
	network.DHCPDStop = current.DHCPDStop
	network.DHCPDLeaseTime = current.DHCPDLeaseTime
	network.DHCPDGatewayEnabled = current.DHCPDGatewayEnabled
	network.DHCPDConflictChecking = current.DHCPDConflictChecking
	network.DHCPDBootEnabled = current.DHCPDBootEnabled
	network.DHCPDBootServer = current.DHCPDBootServer
	network.DHCPDBootFilename = current.DHCPDBootFilename
	network.DHCPDTimeOffsetEnabled = current.DHCPDTimeOffsetEnabled
	network.DHCPDDNSEnabled = current.DHCPDDNSEnabled
	network.DHCPDDNS1 = current.DHCPDDNS1
	network.DHCPDDNS2 = current.DHCPDDNS2
	network.DHCPDDNS3 = current.DHCPDDNS3
	network.DHCPDDNS4 = current.DHCPDDNS4
	network.DHCPDNtpEnabled = current.DHCPDNtpEnabled
	network.DHCPDNtp1 = current.DHCPDNtp1
	network.DHCPDNtp2 = current.DHCPDNtp2
	network.DHCPDWinsEnabled = current.DHCPDWinsEnabled
	network.DHCPDWins1 = current.DHCPDWins1
	network.DHCPDWins2 = current.DHCPDWins2
	network.DHCPDWPAdUrl = current.DHCPDWPAdUrl
	network.DHCPDTFTPServer = current.DHCPDTFTPServer
	network.DHCPDUnifiController = current.DHCPDUnifiController
	return true
}

// networkIPv6FromAPI builds the ipv6 object from the controller's values. An
// omitted interface_type normalizes to the schema default "none" so an
// imported network does not perpetually plan null -> none (#414), an empty
// pd.prefixid reads as null, and aliases is always null because go-unifi has
// no field to carry it (#413; ModifyPlan rejects a configured value).
func networkIPv6FromAPI(
	ctx context.Context,
	network *unifi.Network,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	ra, d := types.ObjectValueFrom(ctx, networkIPv6RAAttrTypes(), networkIPv6RAModel{
		Enabled:  types.BoolValue(network.IPV6RaEnabled),
		Priority: types.StringPointerValue(network.IPV6RaPriority),
		PreferredLifetime: util.DurationPtrValue(
			network.IPV6RaPreferredLifetime,
			time.Second,
		),
		ValidLifetime: util.DurationPtrValue(network.IPV6RaValidLifetime, time.Second),
	})
	diags.Append(d...)
	pd, d := types.ObjectValueFrom(ctx, networkIPv6PDAttrTypes(), networkIPv6PDModel{
		Interface:           types.StringPointerValue(network.IPV6PDInterface),
		Prefixid:            stringOrNull(network.IPV6PDPrefixid),
		Start:               types.StringPointerValue(network.IPV6PDStart),
		Stop:                types.StringPointerValue(network.IPV6PDStop),
		AutoPrefixidEnabled: types.BoolValue(network.IPV6PDAutoPrefixidEnabled),
	})
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(networkIPv6AttrTypes()), diags
	}
	interfaceType := types.StringValue("none")
	if network.IPV6InterfaceType != nil && *network.IPV6InterfaceType != "" {
		interfaceType = types.StringPointerValue(network.IPV6InterfaceType)
	}
	obj, d := types.ObjectValueFrom(ctx, networkIPv6AttrTypes(), networkIPv6Model{
		InterfaceType:           interfaceType,
		ClientAddressAssignment: types.StringPointerValue(network.IPV6ClientAddressAssignment),
		StaticSubnet:            types.StringPointerValue(network.IPV6Subnet),
		Aliases:                 types.ListNull(types.StringType),
		RA:                      ra,
		PD:                      pd,
	})
	diags.Append(d...)
	return obj, diags
}

// ipv6ToFramework builds the ipv6 object for the model. Corporate and guest
// networks reflect the controller. For vlan-only networks (vlanOnly) the
// controller omits the IPv6 fields, so previous (the plan or prior state) is
// preserved leaf by leaf, the way the flat attributes were: only unknown
// leaves resolve to the controller's value (the Computed +
// UseStateForUnknown leaves are unknown on Create, and copying them verbatim
// would trip "invalid result object after apply"), an absent interface_type
// normalizes to the schema default "none" (#414), and aliases stays null.
func (r *networkResource) ipv6ToFramework(
	ctx context.Context,
	network *unifi.Network,
	previous types.Object,
	vlanOnly bool,
) (types.Object, diag.Diagnostics) {
	api, diags := networkIPv6FromAPI(ctx, network)
	if !vlanOnly || diags.HasError() {
		return api, diags
	}
	if previous.IsNull() || previous.IsUnknown() {
		var d diag.Diagnostics
		previous, d = objectOfLeaves(ctx, networkIPv6AttrTypes(), previous.IsUnknown())
		diags.Append(d...)
		if diags.HasError() {
			return api, diags
		}
	}
	var prev, out networkIPv6Model
	diags.Append(previous.As(ctx, &prev, basetypes.ObjectAsOptions{})...)
	diags.Append(
		util.ResolveUnknownObject(ctx, previous, api).As(ctx, &out, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return api, diags
	}
	if prev.InterfaceType.IsNull() || prev.InterfaceType.IsUnknown() ||
		prev.InterfaceType.ValueString() == "" {
		out.InterfaceType = types.StringValue("none")
	}
	out.Aliases = types.ListNull(types.StringType)
	obj, d := types.ObjectValueFrom(ctx, networkIPv6AttrTypes(), out)
	diags.Append(d...)
	return obj, diags
}

// stringListOrNull builds a Terraform list from values. When values is empty,
// it mirrors previous's null-ness instead of always collapsing to null: these
// list attributes are Optional but not Computed, so an empty-list plan (e.g.
// dns.servers = []) must read back as an empty list, not null, or Terraform
// reports "provider produced inconsistent result after apply". A previous
// value of null (attribute never configured) is preserved as null.
func stringListOrNull(
	ctx context.Context,
	values []string,
	previous types.List,
) (types.List, diag.Diagnostics) {
	if len(values) > 0 {
		return types.ListValueFrom(ctx, types.StringType, values)
	}
	if !previous.IsNull() && !previous.IsUnknown() {
		return types.ListValueMust(types.StringType, []attr.Value{}), nil
	}
	return types.ListNull(types.StringType), nil
}

// modelToNetwork converts from Terraform model to unifi.Network.
func (r *networkResource) modelToNetwork(
	ctx context.Context,
	model *networkResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:                    model.Name.ValueStringPointer(),
		Purpose:                 unifi.PurposeCorporate,
		NetworkGroup:            new("LAN"),
		AutoScaleEnabled:        model.AutoScale.ValueBool(),
		IPSubnet:                model.Subnet.ValueStringPointer(),
		NetworkIsolationEnabled: model.NetworkIsolation.ValueBool(),
		SettingPreference:       model.SettingPreference.ValueStringPointer(),
		InternetAccessEnabled:   model.InternetAccess.ValueBool(),
		MdnsEnabled:             model.MulticastDNS.ValueBool(),
		GatewayType:             model.GatewayType.ValueStringPointer(),
		LteLanEnabled:           model.LteLan.ValueBool(),
		VLANEnabled:             !model.Vlan.IsNull() && !model.Vlan.IsUnknown(),
		Enabled:                 model.Enabled.ValueBool(),
		IGMPSnooping:            model.IgmpSnooping.ValueBool(),
		IPAliases:               []string{},
	}

	// IPv6: a null or unknown object (or sub-object) contributes nothing,
	// exactly as the unset flat attributes did.
	if v6, ok, d := util.ObjectAs[networkIPv6Model](ctx, model.IPv6); ok {
		network.IPV6InterfaceType = v6.InterfaceType.ValueStringPointer()
		network.IPV6ClientAddressAssignment = optStr(v6.ClientAddressAssignment)
		network.IPV6Subnet = v6.StaticSubnet.ValueStringPointer()
		if ra, ok, d := util.ObjectAs[networkIPv6RAModel](ctx, v6.RA); ok {
			network.IPV6RaEnabled = ra.Enabled.ValueBool()
			network.IPV6RaPriority = optStr(ra.Priority)
			network.IPV6RaPreferredLifetime = util.DurationUnitsPtr(
				ra.PreferredLifetime,
				time.Second,
			)
			network.IPV6RaValidLifetime = util.DurationUnitsPtr(ra.ValidLifetime, time.Second)
		} else {
			diags.Append(d...)
		}
		if pd, ok, d := util.ObjectAs[networkIPv6PDModel](ctx, v6.PD); ok {
			network.IPV6PDInterface = optStr(pd.Interface)
			network.IPV6PDPrefixid = pd.Prefixid.ValueString()
			network.IPV6PDStart = optStr(pd.Start)
			network.IPV6PDStop = optStr(pd.Stop)
			network.IPV6PDAutoPrefixidEnabled = pd.AutoPrefixidEnabled.ValueBool()
		} else {
			diags.Append(d...)
		}
	} else {
		diags.Append(d...)
	}

	// Purpose: default corporate, honor an explicitly configured value (guest,
	// vlan-only, corporate). third_party_gateway is the legacy way to request
	// vlan-only and takes precedence so existing configs keep working.
	if !model.Purpose.IsNull() && !model.Purpose.IsUnknown() &&
		model.Purpose.ValueString() != "" {
		network.Purpose = model.Purpose.ValueString()
	}
	if model.ThirdPartyGateway.ValueBool() {
		network.Purpose = unifi.PurposeVLANOnly
	}

	// Handle DHCP guarding configuration
	if !model.DhcpGuarding.IsNull() && !model.DhcpGuarding.IsUnknown() {
		var dhcpGuarding dhcpGuardingModel
		d := model.DhcpGuarding.As(ctx, &dhcpGuarding, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.DHCPguardEnabled = dhcpGuarding.Enabled.ValueBool()

			// Map servers to dhcpd_ip_1..3
			if !dhcpGuarding.Servers.IsNull() && !dhcpGuarding.Servers.IsUnknown() {
				var servers []string
				d := dhcpGuarding.Servers.ElementsAs(ctx, &servers, false)
				diags.Append(d...)
				if !diags.HasError() {
					if len(servers) > 0 {
						network.DHCPDIP1 = servers[0]
					}
					if len(servers) > 1 {
						network.DHCPDIP2 = servers[1]
					}
					if len(servers) > 2 {
						network.DHCPDIP3 = servers[2]
					}
				}
			}
		}
	}

	// Handle domain name - set to empty string if null
	network.DomainName = model.DomainName.ValueStringPointer()

	// Handle optional int64 pointer fields
	network.VLAN = model.Vlan.ValueInt64Pointer()

	// Handle NAT outbound IP addresses
	if !model.NatOutboundIPAddresses.IsNull() && !model.NatOutboundIPAddresses.IsUnknown() {
		var natIPs []natOutboundIPAddressesModel
		d := model.NatOutboundIPAddresses.ElementsAs(ctx, &natIPs, true)
		diags.Append(d...)
		if !diags.HasError() {
			for _, natIP := range natIPs {
				v := unifi.NetworkNATOutboundIPAddresses{
					IPAddress:       natIP.IPAddress.ValueString(),
					Mode:            natIP.Mode.ValueStringPointer(),
					WANNetworkGroup: natIP.WANNetworkGroup.ValueStringPointer(),
				}
				network.NATOutboundIPAddresses = append(network.NATOutboundIPAddresses, v)
			}
		}
	}

	// Handle IP aliases
	if !model.IPAliases.IsNull() && !model.IPAliases.IsUnknown() {
		var ipAliases []string
		d := model.IPAliases.ElementsAs(ctx, &ipAliases, false)
		diags.Append(d...)
		if !diags.HasError() {
			network.IPAliases = ipAliases
		}
	}

	// ipv6.aliases: go-unifi's Network struct has no field to send this to the
	// API (#413), so there is nothing to map here. ModifyPlan rejects a
	// non-empty configured value before modelToNetwork ever runs.

	// A DHCP server and DHCP relay cannot coexist on a network: with relay on,
	// emitting DHCPDEnabled=true (as the default branch below would) makes the
	// controller reject the request. We therefore skip the DHCP-server defaults
	// when relay is enabled. ModifyPlan additionally pins setting_preference to
	// "manual" so the controller honors the relay instead of auto-managing it.
	relayEnabled := false
	if !model.DhcpRelay.IsNull() && !model.DhcpRelay.IsUnknown() {
		var dr dhcpRelayModel
		if d := model.DhcpRelay.As(ctx, &dr, basetypes.ObjectAsOptions{}); !d.HasError() {
			relayEnabled = dr.Enabled.ValueBool()
		}
	}

	// Handle DHCP server configuration
	if !model.DhcpServer.IsNull() && !model.DhcpServer.IsUnknown() {
		var dhcpServer dhcpServerModel
		d := model.DhcpServer.As(ctx, &dhcpServer, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			// Handle DHCP boot configuration
			if !dhcpServer.Boot.IsNull() && !dhcpServer.Boot.IsUnknown() {
				var dhcpBoot dhcpBootModel
				d := dhcpServer.Boot.As(ctx, &dhcpBoot, basetypes.ObjectAsOptions{})
				diags.Append(d...)
				if !diags.HasError() {
					network.DHCPDBootEnabled = dhcpBoot.Enabled.ValueBool()
					if dhcpBoot.Server.IsNull() || dhcpBoot.Server.IsUnknown() {
						network.DHCPDBootServer = ""
					} else {
						network.DHCPDBootServer = dhcpBoot.Server.ValueString()
					}
					if dhcpBoot.Filename.IsNull() || dhcpBoot.Filename.IsUnknown() {
						network.DHCPDBootFilename = new("")
					} else {
						network.DHCPDBootFilename = dhcpBoot.Filename.ValueStringPointer()
					}
				}
			} else {
				network.DHCPDBootEnabled = false
				network.DHCPDBootServer = ""
				network.DHCPDBootFilename = new("")
			}
			network.DHCPDEnabled = dhcpServer.Enabled.ValueBool()
			network.DHCPDStart = dhcpServer.Start.ValueStringPointer()
			network.DHCPDStop = dhcpServer.Stop.ValueStringPointer()
			network.DHCPDGatewayEnabled = dhcpServer.GatewayEnabled.ValueBool()
			network.DHCPDConflictChecking = dhcpServer.ConflictChecking.ValueBool()
			// Handle NTP servers. A null or unknown ntp object behaves like the
			// unset flat attributes: disabled, with both server slots cleared.
			var ntp dhcpServerOptionModel
			if v, ok, d := util.ObjectAs[dhcpServerOptionModel](ctx, dhcpServer.Ntp); ok {
				ntp = v
			} else {
				diags.Append(d...)
			}
			network.DHCPDNtpEnabled = ntp.Enabled.ValueBool()
			if !ntp.Servers.IsNull() && !ntp.Servers.IsUnknown() {
				var ntpServers []string
				d := ntp.Servers.ElementsAs(ctx, &ntpServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					for i, ntp := range ntpServers {
						if i >= 2 {
							break
						}
						switch i {
						case 0:
							network.DHCPDNtp1 = new(ntp)
						case 1:
							network.DHCPDNtp2 = new(ntp)
						}
					}
					// Set remaining NTP servers to empty
					for i := len(ntpServers); i < 2; i++ {
						switch i {
						case 0:
							network.DHCPDNtp1 = new("")
						case 1:
							network.DHCPDNtp2 = new("")
						}
					}
				}
			} else {
				// Set all NTP servers to empty string when not configured
				network.DHCPDNtp1 = new("")
				network.DHCPDNtp2 = new("")
			}

			network.DHCPDTimeOffsetEnabled = dhcpServer.TimeOffsetEnabled.ValueBool()
			network.DHCPDLeaseTime = util.DurationUnitsPtr(dhcpServer.Leasetime, time.Second)

			// Handle WINS configuration
			if !dhcpServer.Wins.IsNull() && !dhcpServer.Wins.IsUnknown() {
				var wins winsModel
				d := dhcpServer.Wins.As(ctx, &wins, basetypes.ObjectAsOptions{})
				diags.Append(d...)
				if !diags.HasError() {
					network.DHCPDWinsEnabled = wins.Enabled.ValueBool()
					if !wins.Addresses.IsNull() && !wins.Addresses.IsUnknown() {
						var addresses []string
						d := wins.Addresses.ElementsAs(ctx, &addresses, false)
						diags.Append(d...)
						if !diags.HasError() {
							for i, addr := range addresses {
								if i >= 2 {
									break
								}
								switch i {
								case 0:
									network.DHCPDWins1 = new(addr)
								case 1:
									network.DHCPDWins2 = new(addr)
								}
							}
							// Set remaining WINS servers to empty string
							for i := len(addresses); i < 2; i++ {
								switch i {
								case 0:
									network.DHCPDWins1 = new("")
								case 1:
									network.DHCPDWins2 = new("")
								}
							}
						}
					} else {
						network.DHCPDWins1 = new("")
						network.DHCPDWins2 = new("")
					}
				}
			} else {
				network.DHCPDWinsEnabled = false
				network.DHCPDWins1 = new("")
				network.DHCPDWins2 = new("")
			}

			if dhcpServer.WpadUrl.IsNull() || dhcpServer.WpadUrl.IsUnknown() {
				network.DHCPDWPAdUrl = new("")
			} else {
				network.DHCPDWPAdUrl = dhcpServer.WpadUrl.ValueStringPointer()
			}

			if dhcpServer.TftpServer.IsNull() || dhcpServer.TftpServer.IsUnknown() {
				network.DHCPDTFTPServer = new("")
			} else {
				network.DHCPDTFTPServer = dhcpServer.TftpServer.ValueStringPointer()
			}

			if dhcpServer.UnifiController.IsNull() || dhcpServer.UnifiController.IsUnknown() {
				network.DHCPDUnifiController = new("")
			} else {
				network.DHCPDUnifiController = dhcpServer.UnifiController.ValueStringPointer()
			}

			// Handle DNS servers. A null or unknown dns object behaves like the
			// unset flat attributes: disabled, with every server slot cleared.
			var dns dhcpServerOptionModel
			if v, ok, d := util.ObjectAs[dhcpServerOptionModel](ctx, dhcpServer.Dns); ok {
				dns = v
			} else {
				diags.Append(d...)
			}
			network.DHCPDDNSEnabled = dns.Enabled.ValueBool()
			if !dns.Servers.IsNull() && !dns.Servers.IsUnknown() {
				var dnsServers []string
				d := dns.Servers.ElementsAs(ctx, &dnsServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					for i, dns := range dnsServers {
						if i >= 4 {
							break
						}
						switch i {
						case 0:
							network.DHCPDDNS1 = new(dns)
						case 1:
							network.DHCPDDNS2 = new(dns)
						case 2:
							network.DHCPDDNS3 = new(dns)
						case 3:
							network.DHCPDDNS4 = new(dns)
						}
					}
					// Set remaining DNS servers to empty string
					for i := len(dnsServers); i < 4; i++ {
						switch i {
						case 0:
							network.DHCPDDNS1 = new("")
						case 1:
							network.DHCPDDNS2 = new("")
						case 2:
							network.DHCPDDNS3 = new("")
						case 3:
							network.DHCPDDNS4 = new("")
						}
					}
				}
			} else {
				// Set all DNS servers to empty string when not configured
				network.DHCPDDNS1 = new("")
				network.DHCPDDNS2 = new("")
				network.DHCPDDNS3 = new("")
				network.DHCPDDNS4 = new("")
			}
		}
	} else if !relayEnabled {
		// Set defaults when DHCP server is not configured (and relay is off).
		network.DHCPDBootEnabled = false
		network.DHCPDBootServer = ""
		network.DHCPDBootFilename = new("")
		network.DHCPDEnabled = true
		network.DHCPDGatewayEnabled = false
		network.DHCPDConflictChecking = true
		network.DHCPDNtpEnabled = false
		network.DHCPDNtp1 = new("")
		network.DHCPDNtp2 = new("")
		network.DHCPDTimeOffsetEnabled = false
		network.DHCPDDNSEnabled = false
		network.DHCPDLeaseTime = new(int64(86400))
		network.DHCPDWinsEnabled = false
		network.DHCPDWins1 = new("")
		network.DHCPDWins2 = new("")
		network.DHCPDWPAdUrl = new("")
		network.DHCPDTFTPServer = new("")
		network.DHCPDUnifiController = new("")
		network.DHCPDDNS1 = new("")
		network.DHCPDDNS2 = new("")
		network.DHCPDDNS3 = new("")
		network.DHCPDDNS4 = new("")
	}

	// Handle DHCPv6 server configuration
	if !model.DhcpV6Server.IsNull() && !model.DhcpV6Server.IsUnknown() {
		var dhcpV6Server dhcpV6ServerModel
		d := model.DhcpV6Server.As(ctx, &dhcpV6Server, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.DHCPDV6Enabled = dhcpV6Server.Enabled.ValueBool()
			network.DHCPDV6Start = dhcpV6Server.Start.ValueStringPointer()
			network.DHCPDV6Stop = dhcpV6Server.Stop.ValueStringPointer()
			network.DHCPDV6LeaseTime = dhcpV6Server.Lease.ValueInt64Pointer()

			// Handle DHCPv6 DNS. A null or unknown dns object behaves like the
			// unset flat attributes: auto off, with every server slot cleared.
			var dns dhcpV6DNSModel
			if v, ok, d := util.ObjectAs[dhcpV6DNSModel](ctx, dhcpV6Server.DNS); ok {
				dns = v
			} else {
				diags.Append(d...)
			}
			network.DHCPDV6DNSAuto = dns.Auto.ValueBool()
			if !dns.Servers.IsNull() && !dns.Servers.IsUnknown() {
				var dnsServers []string
				d := dns.Servers.ElementsAs(ctx, &dnsServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					for i, dns := range dnsServers {
						if i >= 4 {
							break
						}
						switch i {
						case 0:
							network.DHCPDV6DNS1 = new(dns)
						case 1:
							network.DHCPDV6DNS2 = new(dns)
						case 2:
							network.DHCPDV6DNS3 = new(dns)
						case 3:
							network.DHCPDV6DNS4 = new(dns)
						}
					}
					for i := len(dnsServers); i < 4; i++ {
						switch i {
						case 0:
							network.DHCPDV6DNS1 = new("")
						case 1:
							network.DHCPDV6DNS2 = new("")
						case 2:
							network.DHCPDV6DNS3 = new("")
						case 3:
							network.DHCPDV6DNS4 = new("")
						}
					}
				}
			} else {
				network.DHCPDV6DNS1 = new("")
				network.DHCPDV6DNS2 = new("")
				network.DHCPDV6DNS3 = new("")
				network.DHCPDV6DNS4 = new("")
			}
		}
	}

	// Handle DHCP relay configuration
	if !model.DhcpRelay.IsNull() && !model.DhcpRelay.IsUnknown() {
		var dhcpRelay dhcpRelayModel
		d := model.DhcpRelay.As(ctx, &dhcpRelay, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.DHCPRelayEnabled = dhcpRelay.Enabled.ValueBool()
			if !dhcpRelay.Servers.IsNull() && !dhcpRelay.Servers.IsUnknown() {
				var servers []string
				d := dhcpRelay.Servers.ElementsAs(ctx, &servers, false)
				diags.Append(d...)
				if !diags.HasError() {
					// The go-unifi client's marshalCorporate maps RemoteVPNSubnets → dhcp_relay_servers
					// JSON field. marshalGuest uses DHCPRelayServers directly. Setting both ensures
					// relay servers are serialized correctly for any network purpose.
					network.RemoteVPNSubnets = servers
					network.DHCPRelayServers = servers
				}
			}
		}
	} else {
		// Set defaults when DHCP relay is not configured
		network.DHCPRelayEnabled = false
	}

	if !model.FirewallZoneID.IsNull() && !model.FirewallZoneID.IsUnknown() &&
		model.FirewallZoneID.ValueString() != "" {
		zoneID := model.FirewallZoneID.ValueString()
		network.FirewallZoneID = &zoneID
	}

	return network, diags
}

// networkToModel converts from unifi.Network to Terraform model.
// previousModel is the model from the plan or previous state, used to preserve null values.
func (r *networkResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *networkResourceModel,
	site string,
	previousModel *networkResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringPointerValue(network.Name)
	model.Enabled = types.BoolValue(network.Enabled)
	model.IgmpSnooping = types.BoolValue(network.IGMPSnooping)
	model.NetworkIsolation = types.BoolValue(network.NetworkIsolationEnabled)

	// Set third_party_gateway based on API purpose
	isVLANOnly := network.Purpose == unifi.PurposeVLANOnly
	model.ThirdPartyGateway = types.BoolValue(isVLANOnly)

	// Reflect the controller's actual purpose. On ZBF controllers the purpose is
	// driven by the network's firewall zone (e.g. guest ⇄ Hotspot zone), so we
	// read it back rather than assume the configured value: an unset purpose
	// (Computed) resolves to whatever the controller reports, and a configured
	// value that the controller rejects surfaces as an inconsistent-result error
	// instead of silently drifting.
	if network.Purpose != "" {
		model.Purpose = types.StringValue(network.Purpose)
	} else {
		model.Purpose = types.StringValue(unifi.PurposeCorporate)
	}

	// For vlan-only networks, the API does not return fields like subnet, gateway_type,
	// setting_preference, etc. Preserve the plan/state values for these irrelevant fields
	// to avoid "inconsistent result after apply" errors.
	if isVLANOnly && previousModel != nil {
		model.Subnet = previousModel.Subnet
		model.AutoScale = previousModel.AutoScale
		model.SettingPreference = previousModel.SettingPreference
		model.InternetAccess = previousModel.InternetAccess
		// multicast_dns uses UseStateForUnknown, so it may be unknown during
		// Create. Resolve it from the API value (the controller does not honor
		// mDNS for vlan-only networks, so this is effectively false).
		if previousModel.MulticastDNS.IsUnknown() {
			model.MulticastDNS = types.BoolValue(network.MdnsEnabled)
		} else {
			model.MulticastDNS = previousModel.MulticastDNS
		}
		// Preserve configured values, but normalize import/read values that are
		// absent because the controller omits fields irrelevant to vlan-only
		// networks. Leaving these null/unknown would perpetually plan the schema
		// defaults (gateway_type="default", ipv6.interface_type="none") (#414).
		if previousModel.GatewayType.IsNull() || previousModel.GatewayType.IsUnknown() ||
			previousModel.GatewayType.ValueString() == "" {
			model.GatewayType = types.StringValue("default")
		} else {
			model.GatewayType = previousModel.GatewayType
		}
		// ipv6 is preserved leaf by leaf; see ipv6ToFramework for the rules.
		ipv6Obj, d := r.ipv6ToFramework(ctx, network, previousModel.IPv6, true)
		diags.Append(d...)
		model.IPv6 = ipv6Obj
		model.LteLan = previousModel.LteLan
		// domain_name uses UseStateForUnknown, so it may be unknown during Create.
		// Resolve unknown to null since the API doesn't return it for vlan-only.
		if previousModel.DomainName.IsUnknown() {
			model.DomainName = types.StringNull()
		} else {
			model.DomainName = previousModel.DomainName
		}
	} else {
		model.AutoScale = types.BoolValue(network.AutoScaleEnabled)
		if network.IPSubnet != nil {
			model.Subnet = cidrtypes.NewIPv4PrefixValue(*network.IPSubnet)
		} else {
			model.Subnet = cidrtypes.NewIPv4PrefixNull()
		}
		model.SettingPreference = types.StringPointerValue(network.SettingPreference)
		model.InternetAccess = types.BoolValue(network.InternetAccessEnabled)
		// Some controllers (notably UniFi OS gateways) ignore mdns_enabled
		// per-network and always store false, so a configured `true` would fail
		// the consistency check (#282; the vlan-only branch above already does
		// this). Preserve the configured/known value; fall back to the
		// controller's value only when it wasn't set by the user (unknown/null,
		// e.g. on Read or List).
		if previousModel != nil && !previousModel.MulticastDNS.IsNull() &&
			!previousModel.MulticastDNS.IsUnknown() {
			model.MulticastDNS = previousModel.MulticastDNS
		} else {
			model.MulticastDNS = types.BoolValue(network.MdnsEnabled)
		}
		// UniFi omits these fields when they have their implicit controller defaults.
		// Normalize the omitted values to the provider schema defaults so an imported
		// network does not perpetually plan null -> default/none changes (#414).
		if network.GatewayType == nil || *network.GatewayType == "" {
			model.GatewayType = types.StringValue("default")
		} else {
			model.GatewayType = types.StringPointerValue(network.GatewayType)
		}
		ipv6Obj, d := r.ipv6ToFramework(
			ctx,
			network,
			types.ObjectNull(networkIPv6AttrTypes()),
			false,
		)
		diags.Append(d...)
		model.IPv6 = ipv6Obj
		model.LteLan = types.BoolValue(network.LteLanEnabled)
		model.DomainName = types.StringPointerValue(network.DomainName)
	}

	// Determine if this is an import. On import only the ID/identity is seeded into
	// state, so the computed network_isolation field is still null; in every other
	// flow (create/read/update) networkToModel always assigns it above. We can no
	// longer use Subnet for this since it is now optional (e.g. vlan_only networks).
	isImport := previousModel != nil && previousModel.NetworkIsolation.IsNull()

	// Build dhcp_guarding from API fields
	shouldPopulateDhcpGuarding := false
	if previousModel != nil {
		shouldPopulateDhcpGuarding = !previousModel.DhcpGuarding.IsNull() ||
			(isImport && network.DHCPguardEnabled)
	} else {
		shouldPopulateDhcpGuarding = network.DHCPguardEnabled
	}

	if shouldPopulateDhcpGuarding {
		var serversList types.List
		servers := collectNonEmptyStrings(network.DHCPDIP1, network.DHCPDIP2, network.DHCPDIP3)
		if len(servers) > 0 {
			var d diag.Diagnostics
			serversList, d = types.ListValueFrom(ctx, types.StringType, servers)
			diags.Append(d...)
		} else {
			serversList = types.ListNull(types.StringType)
		}

		dhcpGuardingValue := dhcpGuardingModel{
			Enabled: types.BoolValue(network.DHCPguardEnabled),
			Servers: serversList,
		}
		dhcpGuardingObj, d := types.ObjectValueFrom(
			ctx,
			dhcpGuardingValue.AttributeTypes(),
			dhcpGuardingValue,
		)
		diags.Append(d...)
		model.DhcpGuarding = dhcpGuardingObj
	} else {
		model.DhcpGuarding = types.ObjectNull(dhcpGuardingModel{}.AttributeTypes())
	}

	model.Vlan = types.Int64PointerValue(network.VLAN)

	// nat_outbound_ip_addresses: ip_address, mode and wan_network_group round-trip
	// from the API. ip_address_pool is not wired end-to-end (ModifyPlan rejects
	// a non-null configured value before Create/Update run) so it is always null.
	// Only populate the field when:
	//   1. It was already non-null in the previous state (i.e., the user manages
	//      it), or
	//   2. This is an import (previousModel == nil) and the controller returned data.
	// This mirrors the dhcp_server "preserve null unless managed" pattern so that
	// a controller-side non-empty list doesn't cause unexpected plan changes for
	// users who haven't configured nat_outbound_ip_addresses.
	shouldPopulateNAT := previousModel == nil || !previousModel.NatOutboundIPAddresses.IsNull()
	if shouldPopulateNAT && len(network.NATOutboundIPAddresses) > 0 {
		natValues := make([]natOutboundIPAddressesModel, 0, len(network.NATOutboundIPAddresses))
		for _, nat := range network.NATOutboundIPAddresses {
			natValues = append(natValues, natOutboundIPAddressesModel{
				IPAddress:       types.StringValue(nat.IPAddress),
				IPAddressPool:   types.ListNull(types.StringType),
				Mode:            types.StringPointerValue(nat.Mode),
				WANNetworkGroup: types.StringPointerValue(nat.WANNetworkGroup),
			})
		}
		natList, d := types.ListValueFrom(
			ctx,
			types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			natValues,
		)
		diags.Append(d...)
		model.NatOutboundIPAddresses = natList
	} else if shouldPopulateNAT {
		// Managed but API returned nothing: write an empty list (not null) to
		// avoid drift between empty vs null when the user configures [].
		model.NatOutboundIPAddresses = types.ListValueMust(
			types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			[]attr.Value{},
		)
	} else {
		model.NatOutboundIPAddresses = types.ListNull(
			types.ObjectType{AttrTypes: natOutboundIPAddresses()},
		)
	}

	if len(network.IPAliases) > 0 {
		ipAliasesList, d := types.ListValueFrom(ctx, types.StringType, network.IPAliases)
		diags.Append(d...)
		model.IPAliases = ipAliasesList
	} else if previousModel != nil && !previousModel.IPAliases.IsNull() &&
		!previousModel.IPAliases.IsUnknown() {
		// Managed but the API returned nothing: keep a known empty list (not
		// null) so a configured `ip_aliases = []` doesn't fail apply with an
		// inconsistent-result error (planned [] vs applied null).
		model.IPAliases = types.ListValueMust(types.StringType, []attr.Value{})
	} else {
		model.IPAliases = types.ListNull(types.StringType)
	}

	// Only populate dhcp_server if:
	// 1. It was configured in the previous state (not null), OR
	// 2. This is an import and DHCP is enabled (populate everything during import)
	shouldPopulateDhcp := false
	if previousModel != nil {
		shouldPopulateDhcp = !previousModel.DhcpServer.IsNull() ||
			(isImport && network.DHCPDEnabled)
	}

	if shouldPopulateDhcp {
		// Helper function to convert empty strings to null
		strPtrToType := func(ptr *string) types.String {
			if ptr == nil || *ptr == "" {
				return types.StringNull()
			}
			return types.StringValue(*ptr)
		}

		// Extract the previous dhcp_server value (from plan or prior state) so
		// list attributes below can distinguish "never configured" (null) from
		// "configured empty" (empty list) when the API reports no values.
		var previousDhcpServer dhcpServerModel
		var previousWins winsModel
		var previousDNS, previousNTP dhcpServerOptionModel
		if previousModel != nil && !previousModel.DhcpServer.IsNull() &&
			!previousModel.DhcpServer.IsUnknown() {
			d := previousModel.DhcpServer.As(ctx, &previousDhcpServer, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if !previousDhcpServer.Wins.IsNull() && !previousDhcpServer.Wins.IsUnknown() {
				d := previousDhcpServer.Wins.As(ctx, &previousWins, basetypes.ObjectAsOptions{})
				diags.Append(d...)
			}
			if v, ok, d := util.ObjectAs[dhcpServerOptionModel](ctx, previousDhcpServer.Dns); ok {
				previousDNS = v
			} else {
				diags.Append(d...)
			}
			if v, ok, d := util.ObjectAs[dhcpServerOptionModel](ctx, previousDhcpServer.Ntp); ok {
				previousNTP = v
			} else {
				diags.Append(d...)
			}
		}

		bootServer := types.StringNull()
		if network.DHCPDBootServer != "" {
			bootServer = types.StringValue(network.DHCPDBootServer)
		}
		dhcpBootValue := dhcpBootModel{
			Enabled:  types.BoolValue(network.DHCPDBootEnabled),
			Server:   bootServer,
			Filename: strPtrToType(network.DHCPDBootFilename),
		}

		dhcpBootObj, d := types.ObjectValueFrom(
			ctx,
			dhcpBootValue.AttributeTypes(),
			dhcpBootValue,
		)
		diags.Append(d...)

		// Build DNS servers list from DHCPDDNS1-4
		dnsServers := collectNonEmptyStringPointers(
			network.DHCPDDNS1, network.DHCPDDNS2, network.DHCPDDNS3, network.DHCPDDNS4,
		)

		dnsServersList, d := stringListOrNull(ctx, dnsServers, previousDNS.Servers)
		diags.Append(d...)
		dnsObj, d := types.ObjectValueFrom(
			ctx,
			dhcpServerOptionModel{}.AttributeTypes(),
			dhcpServerOptionModel{
				Enabled: types.BoolValue(network.DHCPDDNSEnabled),
				Servers: dnsServersList,
			},
		)
		diags.Append(d...)

		// Build NTP servers list from DHCPDNtp1-2
		var ntpServers []string
		if network.DHCPDNtp1 != nil && *network.DHCPDNtp1 != "" {
			ntpServers = append(ntpServers, *network.DHCPDNtp1)
		}
		if network.DHCPDNtp2 != nil && *network.DHCPDNtp2 != "" {
			ntpServers = append(ntpServers, *network.DHCPDNtp2)
		}

		ntpServersList, d := stringListOrNull(ctx, ntpServers, previousNTP.Servers)
		diags.Append(d...)
		ntpObj, d := types.ObjectValueFrom(
			ctx,
			dhcpServerOptionModel{}.AttributeTypes(),
			dhcpServerOptionModel{
				Enabled: types.BoolValue(network.DHCPDNtpEnabled),
				Servers: ntpServersList,
			},
		)
		diags.Append(d...)

		// Build WINS addresses list from DHCPDWins1-2
		var winsAddresses []string
		if network.DHCPDWins1 != nil && *network.DHCPDWins1 != "" {
			winsAddresses = append(winsAddresses, *network.DHCPDWins1)
		}
		if network.DHCPDWins2 != nil && *network.DHCPDWins2 != "" {
			winsAddresses = append(winsAddresses, *network.DHCPDWins2)
		}

		winsAddressesList, d := stringListOrNull(ctx, winsAddresses, previousWins.Addresses)
		diags.Append(d...)

		winsValue := winsModel{
			Enabled:   types.BoolValue(network.DHCPDWinsEnabled),
			Addresses: winsAddressesList,
		}

		winsObj, d := types.ObjectValueFrom(ctx, winsValue.AttributeTypes(), winsValue)
		diags.Append(d...)

		dhcpServerValue := dhcpServerModel{
			Boot:              dhcpBootObj,
			Enabled:           types.BoolValue(network.DHCPDEnabled),
			GatewayEnabled:    types.BoolValue(network.DHCPDGatewayEnabled),
			ConflictChecking:  types.BoolValue(network.DHCPDConflictChecking),
			Ntp:               ntpObj,
			TimeOffsetEnabled: types.BoolValue(network.DHCPDTimeOffsetEnabled),
			Dns:               dnsObj,
			Leasetime:         util.DurationPtrValue(network.DHCPDLeaseTime, time.Second),
			Wins:              winsObj,
			WpadUrl:           strPtrToType(network.DHCPDWPAdUrl),
			Start:             types.StringPointerValue(network.DHCPDStart),
			Stop:              types.StringPointerValue(network.DHCPDStop),
			TftpServer:        strPtrToType(network.DHCPDTFTPServer),
			UnifiController:   strPtrToType(network.DHCPDUnifiController),
		}

		dhcpServerObj, d := types.ObjectValueFrom(
			ctx,
			dhcpServerValue.AttributeTypes(),
			dhcpServerValue,
		)
		diags.Append(d...)
		model.DhcpServer = dhcpServerObj
	} else {
		// Keep dhcp_server null if it wasn't in the plan/state
		model.DhcpServer = types.ObjectNull(dhcpServerModel{}.AttributeTypes())
	}

	// Only populate dhcp_v6_server if:
	// 1. It was configured in the previous state (not null), OR
	// 2. This is an import and DHCPv6 is enabled
	shouldPopulateDhcpV6 := false
	if previousModel != nil {
		shouldPopulateDhcpV6 = !previousModel.DhcpV6Server.IsNull() ||
			(isImport && network.DHCPDV6Enabled)
	}

	if shouldPopulateDhcpV6 {
		dhcpv6DNS := collectNonEmptyStringPointers(
			network.DHCPDV6DNS1, network.DHCPDV6DNS2,
			network.DHCPDV6DNS3, network.DHCPDV6DNS4,
		)
		var dhcpv6DNSList types.List
		if len(dhcpv6DNS) > 0 {
			var d diag.Diagnostics
			dhcpv6DNSList, d = types.ListValueFrom(ctx, types.StringType, dhcpv6DNS)
			diags.Append(d...)
		} else {
			dhcpv6DNSList = types.ListNull(types.StringType)
		}

		dhcpv6DNSObj, d := types.ObjectValueFrom(
			ctx,
			dhcpV6DNSModel{}.AttributeTypes(),
			dhcpV6DNSModel{
				Auto:    types.BoolValue(network.DHCPDV6DNSAuto),
				Servers: dhcpv6DNSList,
			},
		)
		diags.Append(d...)

		dhcpV6ServerValue := dhcpV6ServerModel{
			Enabled: types.BoolValue(network.DHCPDV6Enabled),
			DNS:     dhcpv6DNSObj,
			Lease:   types.Int64PointerValue(network.DHCPDV6LeaseTime),
			Start:   types.StringPointerValue(network.DHCPDV6Start),
			Stop:    types.StringPointerValue(network.DHCPDV6Stop),
		}
		dhcpV6ServerObj, d := types.ObjectValueFrom(
			ctx,
			dhcpV6ServerValue.AttributeTypes(),
			dhcpV6ServerValue,
		)
		diags.Append(d...)
		model.DhcpV6Server = dhcpV6ServerObj
	} else {
		model.DhcpV6Server = types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes())
	}

	// Only populate dhcp_relay if:
	// 1. It was configured in the previous state (not null), OR
	// 2. This is an import and DHCP relay is enabled (populate everything during import)
	shouldPopulateRelay := false
	if previousModel != nil {
		shouldPopulateRelay = !previousModel.DhcpRelay.IsNull() ||
			(isImport && network.DHCPRelayEnabled)
	}

	if shouldPopulateRelay {
		var relayServersVal types.List
		if len(network.DHCPRelayServers) > 0 {
			var d diag.Diagnostics
			relayServersVal, d = types.ListValueFrom(
				ctx,
				types.StringType,
				network.DHCPRelayServers,
			)
			diags.Append(d...)
		} else {
			relayServersVal = types.ListNull(types.StringType)
		}
		dhcpRelayValue := dhcpRelayModel{
			Enabled: types.BoolValue(network.DHCPRelayEnabled),
			Servers: relayServersVal,
		}

		dhcpRelayObj, d := types.ObjectValueFrom(
			ctx,
			dhcpRelayValue.AttributeTypes(),
			dhcpRelayValue,
		)
		diags.Append(d...)
		model.DhcpRelay = dhcpRelayObj
	} else {
		// Keep dhcp_relay null if it wasn't in the plan/state
		model.DhcpRelay = types.ObjectNull(dhcpRelayModel{}.AttributeTypes())
	}

	if network.FirewallZoneID != nil {
		model.FirewallZoneID = types.StringPointerValue(network.FirewallZoneID)
	} else {
		model.FirewallZoneID = types.StringNull()
	}

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *networkResource) ListResourceConfigSchema(
	ctx context.Context,
	req list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List networks in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list networks from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`.",
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
func (r *networkResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config networkListConfigModel

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
	var filters []networkListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	networks, err := r.client.ListNetwork(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Networks", "Could not list networks: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, network := range networks {
			// Filter by purpose: only corporate, guest and vlan-only networks.
			if network.Purpose != unifi.PurposeCorporate &&
				network.Purpose != unifi.PurposeGuest &&
				network.Purpose != unifi.PurposeVLANOnly {
				continue
			}

			// Apply name filter if specified.
			if nameFilter, ok := postFilters["name"]; ok {
				if network.Name == nil || *network.Name != nameFilter {
					continue
				}
			}

			result := req.NewListResult(ctx)
			if network.Name != nil {
				result.DisplayName = *network.Name
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.Set(ctx, networkIdentityModel{
					ID:   types.StringValue(network.ID),
					Site: types.StringValue(site),
				})...,
			)

			// Convert to model.
			var model networkResourceModel
			result.Diagnostics.Append(
				r.networkToModel(ctx, &network, &model, site, &networkResourceModel{})...)
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
