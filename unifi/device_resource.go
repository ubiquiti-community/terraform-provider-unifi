package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util/retry"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &deviceResource{}
	_ resource.ResourceWithImportState  = &deviceResource{}
	_ resource.ResourceWithIdentity     = &deviceResource{}
	_ resource.ResourceWithUpgradeState = &deviceResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &deviceResource{}
	_ list.ListResourceWithConfigure = &deviceResource{}
)

func NewDeviceFrameworkResource() resource.Resource {
	return &deviceResource{}
}

func NewDeviceListResource() list.ListResource {
	return &deviceResource{}
}

// deviceListConfigModel describes the list configuration model.
type deviceListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// deviceListFilterModel represents a single name/value filter entry.
type deviceListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// deviceResource defines the resource implementation.
type deviceResource struct {
	client *Client
}

// deviceResourceModel describes the resource data model.
type deviceResourceModel struct {
	ID              types.String       `tfsdk:"id"`
	Site            types.String       `tfsdk:"site"`
	MAC             hwtypes.MACAddress `tfsdk:"mac"`
	Name            types.String       `tfsdk:"name"`
	Disabled        types.Bool         `tfsdk:"disabled"`
	PortOverride    types.Set          `tfsdk:"port_override"`
	AllowAdoption   types.Bool         `tfsdk:"allow_adoption"`
	ForgetOnDestroy types.Bool         `tfsdk:"forget_on_destroy"`

	// Network configuration
	ConfigNetwork types.Object `tfsdk:"config_network"`

	// LED settings
	LedOverride                types.String `tfsdk:"led_override"`
	LedOverrideColor           types.String `tfsdk:"led_override_color"`
	LedOverrideColorBrightness types.Int64  `tfsdk:"led_override_color_brightness"`

	// Device features
	BandsteeringMode  types.String `tfsdk:"bandsteering_mode"`
	FlowctrlEnabled   types.Bool   `tfsdk:"flowctrl_enabled"`
	JumboframeEnabled types.Bool   `tfsdk:"jumboframe_enabled"`
	StpVersion        types.String `tfsdk:"stp_version"`
	StpPriority       types.Int64  `tfsdk:"stp_priority"`
	Locked            types.Bool   `tfsdk:"locked"`

	// PoE settings
	PoeMode types.String `tfsdk:"poe_mode"`

	// VLAN
	SwitchVLANEnabled types.Bool `tfsdk:"switch_vlan_enabled"`

	// Mesh
	MeshStaVapEnabled types.Bool `tfsdk:"mesh_sta_vap_enabled"`

	// Radio settings
	RadioTable types.List `tfsdk:"radio_table"`

	// Advanced features
	OutdoorModeOverride types.String `tfsdk:"outdoor_mode_override"`
	Volume              types.Int64  `tfsdk:"volume"`
	BaresipPassword     types.String `tfsdk:"x_baresip_password"`

	// LCD/LCM settings
	LcmBrightness          types.Int64          `tfsdk:"lcm_brightness"`
	LcmBrightnessOverride  types.Bool           `tfsdk:"lcm_brightness_override"`
	LcmIDleTimeout         timetypes.GoDuration `tfsdk:"lcm_idle_timeout"`
	LcmIDleTimeoutOverride types.Bool           `tfsdk:"lcm_idle_timeout_override"`
	LcmNightModeBegins     types.String         `tfsdk:"lcm_night_mode_begins"`
	LcmNightModeEnds       types.String         `tfsdk:"lcm_night_mode_ends"`

	// Outlet settings
	OutletOverrides types.List `tfsdk:"outlet_overrides"`
	OutletEnabled   types.Bool `tfsdk:"outlet_enabled"`

	// Management
	MgmtNetworkID types.String `tfsdk:"mgmt_network_id"`

	// Computed attributes
	Adopted types.Bool   `tfsdk:"adopted"`
	Model   types.String `tfsdk:"model"`
	Type    types.String `tfsdk:"type"`
	State   types.Int64  `tfsdk:"state"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// portOverrideModel describes the port override data model.
type portOverrideModel struct {
	Index                      types.Int64          `tfsdk:"index"`
	Name                       types.String         `tfsdk:"name"`
	PortProfileID              types.String         `tfsdk:"port_profile_id"`
	OpMode                     types.String         `tfsdk:"op_mode"`
	PoeMode                    types.String         `tfsdk:"poe_mode"`
	AggregateMembers           types.List           `tfsdk:"aggregate_members"`
	Autoneg                    types.Bool           `tfsdk:"autoneg"`
	Dot1XCtrl                  types.String         `tfsdk:"dot1x_ctrl"`
	Dot1XIDleTimeout           timetypes.GoDuration `tfsdk:"dot1x_idle_timeout"`
	EgressRateLimitKbps        types.Int64          `tfsdk:"egress_rate_limit_kbps"`
	EgressRateLimitKbpsEnabled types.Bool           `tfsdk:"egress_rate_limit_kbps_enabled"`
	ExcludedNetworkIDs         types.List           `tfsdk:"excluded_networkconf_ids"`
	FecMode                    types.String         `tfsdk:"fec_mode"`
	FlowControlEnabled         types.Bool           `tfsdk:"flow_control_enabled"`
	Forward                    types.String         `tfsdk:"forward"`
	FullDuplex                 types.Bool           `tfsdk:"full_duplex"`
	Isolation                  types.Bool           `tfsdk:"isolation"`
	LldpmedEnabled             types.Bool           `tfsdk:"lldpmed_enabled"`
	LldpmedNotifyEnabled       types.Bool           `tfsdk:"lldpmed_notify_enabled"`
	MirrorPortIDX              types.Int64          `tfsdk:"mirror_port_idx"`
	MulticastRouterNetworkIDs  types.List           `tfsdk:"multicast_router_networkconf_ids"`
	NativeNetworkID            types.String         `tfsdk:"native_networkconf_id"`
	PortKeepaliveEnabled       types.Bool           `tfsdk:"port_keepalive_enabled"`
	PortSecurityEnabled        types.Bool           `tfsdk:"port_security_enabled"`
	PortSecurityMACAddress     types.List           `tfsdk:"port_security_mac_address"`
	PriorityQueue1Level        types.Int64          `tfsdk:"priority_queue1_level"`
	PriorityQueue2Level        types.Int64          `tfsdk:"priority_queue2_level"`
	PriorityQueue3Level        types.Int64          `tfsdk:"priority_queue3_level"`
	PriorityQueue4Level        types.Int64          `tfsdk:"priority_queue4_level"`
	SettingPreference          types.String         `tfsdk:"setting_preference"`
	Speed                      types.Int64          `tfsdk:"speed"`
	StormctrlBroadcastEnabled  types.Bool           `tfsdk:"stormctrl_bcast_enabled"`
	StormctrlBroadcastLevel    types.Int64          `tfsdk:"stormctrl_bcast_level"`
	StormctrlBroadcastRate     types.Int64          `tfsdk:"stormctrl_bcast_rate"`
	StormctrlMcastEnabled      types.Bool           `tfsdk:"stormctrl_mcast_enabled"`
	StormctrlMcastLevel        types.Int64          `tfsdk:"stormctrl_mcast_level"`
	StormctrlMcastRate         types.Int64          `tfsdk:"stormctrl_mcast_rate"`
	StormctrlType              types.String         `tfsdk:"stormctrl_type"`
	StormctrlUcastEnabled      types.Bool           `tfsdk:"stormctrl_ucast_enabled"`
	StormctrlUcastLevel        types.Int64          `tfsdk:"stormctrl_ucast_level"`
	StormctrlUcastRate         types.Int64          `tfsdk:"stormctrl_ucast_rate"`
	StpPortMode                types.Bool           `tfsdk:"stp_port_mode"`
	TaggedNetworkIDs           types.List           `tfsdk:"tagged_networkconf_ids"`
	TaggedVLANMgmt             types.String         `tfsdk:"tagged_vlan_mgmt"`
	VoiceNetworkID             types.String         `tfsdk:"voice_networkconf_id"`
}

func (m portOverrideModel) AttributeTypes() map[string]attr.Type {
	return portOverrideAttrTypes()
}

// configNetworkModel describes the config network data model.
type configNetworkModel struct {
	Type           types.String `tfsdk:"type"`
	IP             types.String `tfsdk:"ip"`
	Netmask        types.String `tfsdk:"netmask"`
	Gateway        types.String `tfsdk:"gateway"`
	DNS1           types.String `tfsdk:"dns1"`
	DNS2           types.String `tfsdk:"dns2"`
	DNSsuffix      types.String `tfsdk:"dnssuffix"`
	BondingEnabled types.Bool   `tfsdk:"bonding_enabled"`
}

// radioTableModel describes the radio table data model.
type radioTableModel struct {
	Radio                  types.String `tfsdk:"radio"`
	Channel                types.String `tfsdk:"channel"`
	Ht                     types.Int64  `tfsdk:"ht"`
	TxPower                types.String `tfsdk:"tx_power"`
	TxPowerMode            types.String `tfsdk:"tx_power_mode"`
	MinRssiEnabled         types.Bool   `tfsdk:"min_rssi_enabled"`
	MinRssi                types.Int64  `tfsdk:"min_rssi"`
	AntennaGain            types.Int64  `tfsdk:"antenna_gain"`
	AntennaID              types.Int64  `tfsdk:"antenna_id"`
	AssistedRoamingEnabled types.Bool   `tfsdk:"assisted_roaming_enabled"`
	AssistedRoamingRssi    types.Int64  `tfsdk:"assisted_roaming_rssi"`
	Dfs                    types.Bool   `tfsdk:"dfs"`
	HardNoiseFloorEnabled  types.Bool   `tfsdk:"hard_noise_floor_enabled"`
	LoadbalanceEnabled     types.Bool   `tfsdk:"loadbalance_enabled"`
	Maxsta                 types.Int64  `tfsdk:"maxsta"`
	Name                   types.String `tfsdk:"name"`
	SensLevel              types.Int64  `tfsdk:"sens_level"`
	SensLevelEnabled       types.Bool   `tfsdk:"sens_level_enabled"`
	VwireEnabled           types.Bool   `tfsdk:"vwire_enabled"`
}

// outletOverrideModel describes the outlet override data model.
type outletOverrideModel struct {
	Index        types.Int64  `tfsdk:"index"`
	Name         types.String `tfsdk:"name"`
	RelayState   types.Bool   `tfsdk:"relay_state"`
	CycleEnabled types.Bool   `tfsdk:"cycle_enabled"`
}

func (r *deviceResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *deviceResource) IdentitySchema(
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

func (r *deviceResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		// v1: lcm_idle_timeout and port_override.dot1x_idle_timeout changed from
		// Int64 (seconds) to GoDuration strings. See UpgradeState.
		Version: 1,
		Description: "`unifi_device` manages a device of the network.\n\n" +
			"Devices are adopted by the controller, so it is not possible for this resource to be created through " +
			"Terraform, the create operation instead will simply start managing the device specified by MAC address. " +
			"It's safer to start this process with an explicit import of the device.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the device.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "The name of the site to associate the device with.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				Description: "The MAC address of the device. This can be specified so that the provider can take control of a device (since devices are created through adoption).",
				Optional:    true,
				Computed:    true,
				CustomType:  hwtypes.MACAddressType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the device.",
				Optional:    true,
				Computed:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Specifies whether this device should be disabled.",
				Optional:    true,
				Computed:    true,
			},
			"allow_adoption": schema.BoolAttribute{
				Description: "Specifies whether this resource should tell the controller to adopt the device on create.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"forget_on_destroy": schema.BoolAttribute{
				Description: "Specifies whether this resource should tell the controller to forget the device on destroy.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			// Network configuration
			"config_network": schema.SingleNestedAttribute{
				Description: "Network configuration for the device.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Network configuration type (dhcp or static).",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("dhcp", "static"),
						},
					},
					"ip": schema.StringAttribute{
						Description: "IP address (for static configuration).",
						Optional:    true,
						Computed:    true,
					},
					"netmask": schema.StringAttribute{
						Description: "Network mask (for static configuration).",
						Optional:    true,
						Computed:    true,
					},
					"gateway": schema.StringAttribute{
						Description: "Gateway address (for static configuration).",
						Optional:    true,
						Computed:    true,
					},
					"dns1": schema.StringAttribute{
						Description: "Primary DNS server.",
						Optional:    true,
						Computed:    true,
					},
					"dns2": schema.StringAttribute{
						Description: "Secondary DNS server.",
						Optional:    true,
						Computed:    true,
					},
					"dnssuffix": schema.StringAttribute{
						Description: "DNS suffix.",
						Optional:    true,
						Computed:    true,
					},
					"bonding_enabled": schema.BoolAttribute{
						Description: "Enable network bonding.",
						Optional:    true,
						Computed:    true,
					},
				},
			},

			// LED settings
			"led_override": schema.StringAttribute{
				Description: "LED override setting; valid values are `default`, `on`, and `off`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("default", "on", "off"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"led_override_color": schema.StringAttribute{
				Description: "LED color override (hex color code).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"led_override_color_brightness": schema.Int64Attribute{
				Description: "LED brightness (0-100).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},

			// Device features
			"bandsteering_mode": schema.StringAttribute{
				Description: "Band steering mode; valid values are `off`, `equal`, and `prefer_5g`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("off", "equal", "prefer_5g"),
				},
			},
			"flowctrl_enabled": schema.BoolAttribute{
				Description: "Enable flow control.",
				Optional:    true,
				Computed:    true,
			},
			"jumboframe_enabled": schema.BoolAttribute{
				Description: "Enable jumbo frames.",
				Optional:    true,
				Computed:    true,
			},
			"stp_version": schema.StringAttribute{
				Description: "STP version; valid values are `stp`, `rstp`, and `disabled`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("stp", "rstp", "disabled"),
				},
			},
			"stp_priority": schema.Int64Attribute{
				Description: "STP priority.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						0, 4096, 8192, 12288, 16384, 20480,
						24576, 28672, 32768, 36864, 40960,
						45056, 49152, 53248, 57344, 61440,
					),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Specifies whether the device is locked.",
				Optional:    true,
				Computed:    true,
			},

			// PoE settings
			"poe_mode": schema.StringAttribute{
				Description: "PoE mode; valid values are `auto`, `pasv24`, `passthrough`, and `off`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "pasv24", "passthrough", "off"),
				},
			},

			// VLAN
			"switch_vlan_enabled": schema.BoolAttribute{
				Description: "Enable VLAN support on the switch.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			// Mesh
			"mesh_sta_vap_enabled": schema.BoolAttribute{
				Description: "Enable the mesh station VAP (the UI \"Mesh Connect\" toggle), letting this AP uplink wirelessly to a mesh parent.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			// Advanced features
			"outdoor_mode_override": schema.StringAttribute{
				Description: "Outdoor mode override; valid values are `default`, `on`, and `off`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("default", "on", "off"),
				},
			},
			"volume": schema.Int64Attribute{
				Description: "Volume level (0-100).",
				Optional:    true,
				Computed:    true,
			},
			"x_baresip_password": schema.StringAttribute{
				Description: "Baresip password.",
				Optional:    true,
				Sensitive:   true,
				Computed:    true,
			},

			// LCD/LCM settings
			"lcm_brightness": schema.Int64Attribute{
				Description: "LCM brightness (1-100).",
				Optional:    true,
				Computed:    true,
			},
			"lcm_brightness_override": schema.BoolAttribute{
				Description: "Override LCM brightness.",
				Optional:    true,
				Computed:    true,
			},
			"lcm_idle_timeout": schema.StringAttribute{
				Description: "LCM idle timeout, as a Go duration string (e.g. `10m`, `600s`).",
				CustomType:  timetypes.GoDurationType{},
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validators.GoDurationBetween(10*time.Second, 3600*time.Second),
					validators.GoDurationMultipleOf(time.Second),
				},
			},
			"lcm_idle_timeout_override": schema.BoolAttribute{
				Description: "Override LCM idle timeout.",
				Optional:    true,
				Computed:    true,
			},
			"lcm_night_mode_begins": schema.StringAttribute{
				Description: "LCM night mode begin time (HH:MM format).",
				Optional:    true,
				Computed:    true,
			},
			"lcm_night_mode_ends": schema.StringAttribute{
				Description: "LCM night mode end time (HH:MM format).",
				Optional:    true,
				Computed:    true,
			},

			// Outlet settings
			"outlet_enabled": schema.BoolAttribute{
				Description: "Enable outlet control.",
				Optional:    true,
				Computed:    true,
			},

			// Management
			"mgmt_network_id": schema.StringAttribute{
				Description: "Management network ID. The network this device uses for its own management traffic (the UI's Network Override). When set, the device tags its management onto this network's VLAN, so that VLAN must already be tagged on the device's upstream switch port(s) before this attribute is applied. Otherwise the device loses its management path, drops off, and the apply fails with an inconsistent-result error. Apply in two steps: tag the VLAN on the uplink (a port_override tagged_networkconf_ids entry) first, then set mgmt_network_id. Leave unset to manage on the uplink's native (untagged) network.",
				Optional:    true,
				Computed:    true,
			},

			// Computed attributes
			"adopted": schema.BoolAttribute{
				Description: "Whether the device is adopted.",
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "Device model.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Device type.",
				Computed:    true,
			},
			"state": schema.Int64Attribute{
				Description: "Device state.",
				Computed:    true,
			},

			// Radio table
			"radio_table": schema.ListNestedAttribute{
				Description: "Radio configuration table.",
				Optional:    true,
				Computed:    true,
				// Controller-managed radio config: keep the prior value when the plan
				// leaves it unknown, so editing an unrelated device field doesn't replan
				// the whole table to "(known after apply)".
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"radio": schema.StringAttribute{
							Description: "Radio band (ng, na, ad, 6e).",
							Optional:    true,
							Computed:    true,
						},
						"channel": schema.StringAttribute{
							Description: "Channel number or 'auto'.",
							Optional:    true,
							Computed:    true,
						},
						"ht": schema.Int64Attribute{
							Description: "Channel width (20, 40, 80, 160).",
							Optional:    true,
							Computed:    true,
						},
						"tx_power": schema.StringAttribute{
							Description: "Transmit power or 'auto'.",
							Optional:    true,
							Computed:    true,
						},
						"tx_power_mode": schema.StringAttribute{
							Description: "Transmit power mode (auto, medium, high, low, custom).",
							Optional:    true,
							Computed:    true,
						},
						"min_rssi_enabled": schema.BoolAttribute{
							Description: "Enable minimum RSSI.",
							Optional:    true,
							Computed:    true,
						},
						"min_rssi": schema.Int64Attribute{
							Description: "Minimum RSSI value.",
							Optional:    true,
							Computed:    true,
						},
						"antenna_gain": schema.Int64Attribute{
							Description: "Antenna gain.",
							Optional:    true,
							Computed:    true,
						},
						"antenna_id": schema.Int64Attribute{
							Description: "Antenna ID.",
							Optional:    true,
							Computed:    true,
						},
						"assisted_roaming_enabled": schema.BoolAttribute{
							Description: "Enable assisted roaming.",
							Optional:    true,
							Computed:    true,
						},
						"assisted_roaming_rssi": schema.Int64Attribute{
							Description: "Assisted roaming RSSI threshold.",
							Optional:    true,
							Computed:    true,
						},
						"dfs": schema.BoolAttribute{
							Description: "Enable DFS (Dynamic Frequency Selection).",
							Optional:    true,
							Computed:    true,
						},
						"hard_noise_floor_enabled": schema.BoolAttribute{
							Description: "Enable hard noise floor.",
							Optional:    true,
							Computed:    true,
						},
						"loadbalance_enabled": schema.BoolAttribute{
							Description: "Enable load balancing.",
							Optional:    true,
							Computed:    true,
						},
						"maxsta": schema.Int64Attribute{
							Description: "Maximum number of stations.",
							Optional:    true,
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Radio name.",
							Optional:    true,
							Computed:    true,
						},
						"sens_level": schema.Int64Attribute{
							Description: "Sensitivity level.",
							Optional:    true,
							Computed:    true,
						},
						"sens_level_enabled": schema.BoolAttribute{
							Description: "Enable sensitivity level.",
							Optional:    true,
							Computed:    true,
						},
						"vwire_enabled": schema.BoolAttribute{
							Description: "Enable virtual wire.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},

			// Outlet overrides
			"outlet_overrides": schema.ListNestedAttribute{
				Description: "Outlet configuration overrides.",
				Optional:    true,
				Computed:    true,
				// Keep the prior value when the plan leaves it unknown, so editing an
				// unrelated device field doesn't replan every outlet to
				// "(known after apply)".
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index": schema.Int64Attribute{
							Description: "Outlet index.",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description: "Outlet name.",
							Optional:    true,
							Computed:    true,
						},
						"relay_state": schema.BoolAttribute{
							Description: "Relay state (on/off).",
							Optional:    true,
							Computed:    true,
						},
						"cycle_enabled": schema.BoolAttribute{
							Description: "Enable power cycle.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},

			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},

		Blocks: map[string]schema.Block{
			"port_override": schema.SetNestedBlock{
				Description: "Per-port settings overrides, applied only to the ports you " +
					"declare. Ports without a `port_override` block keep their existing " +
					"controller-side configuration — the provider merges your declared " +
					"ports (by `index`) into the device's current overrides rather than " +
					"replacing the whole set. Removing a block stops managing that port " +
					"but does not reset it; clear a port by overriding it back to the " +
					"defaults instead.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"index": schema.Int64Attribute{
							Description: "Switch port index.",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description: "Human-readable name of the port.",
							Optional:    true,
						},
						"port_profile_id": schema.StringAttribute{
							Description: "ID of the Port Profile used on this port.",
							Optional:    true,
						},
						"op_mode": schema.StringAttribute{
							Description: "Operating mode of the port: `switch` (default), `mirror`, or `aggregate`. " +
								"Set `aggregate` on the lead port of an SFP+/link-aggregation (LAG) group and list the member ports in `aggregate_members`. " +
								"Only written when not `switch`, as gateway devices (UDM) reject op_mode on update.",
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("switch"),
							Validators: []validator.String{
								stringvalidator.OneOf("switch", "mirror", "aggregate"),
							},
						},
						"poe_mode": schema.StringAttribute{
							Description: "PoE mode of the port; valid values are `auto`, `pasv24`, `passthrough`, and `off`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "pasv24", "passthrough", "off"),
							},
						},
						"aggregate_members": schema.ListAttribute{
							Description: "Port indices that make up this link-aggregation (LAG) group. " +
								"Only takes effect when `op_mode` is `aggregate` on this port.",
							Optional:    true,
							ElementType: types.Int64Type,
						},
						"autoneg": schema.BoolAttribute{
							Description: "Enable auto-negotiation for port speed.",
							Optional:    true,
							Computed:    true,
						},
						"dot1x_ctrl": schema.StringAttribute{
							Description: "802.1X control mode.",
							Optional:    true,
						},
						"dot1x_idle_timeout": schema.StringAttribute{
							Description: "802.1X idle timeout, as a Go duration string (e.g. `5m`, `300s`).",
							CustomType:  timetypes.GoDurationType{},
							Optional:    true,
							Validators: []validator.String{
								validators.GoDurationBetween(0, 65535*time.Second),
								validators.GoDurationMultipleOf(time.Second),
							},
						},
						"egress_rate_limit_kbps": schema.Int64Attribute{
							Description: "Egress rate limit in kbps.",
							Optional:    true,
						},
						"egress_rate_limit_kbps_enabled": schema.BoolAttribute{
							Description: "Enable egress rate limiting.",
							Optional:    true,
							Computed:    true,
						},
						"excluded_networkconf_ids": schema.ListAttribute{
							Description: "List of network IDs to exclude from this port.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"fec_mode": schema.StringAttribute{
							Description: "Forward Error Correction mode.",
							Optional:    true,
						},
						"flow_control_enabled": schema.BoolAttribute{
							Description: "Enable flow control.",
							Optional:    true,
							Computed:    true,
						},
						"forward": schema.StringAttribute{
							Description: "Forwarding mode.",
							Optional:    true,
						},
						"full_duplex": schema.BoolAttribute{
							Description: "Enable full duplex mode.",
							Optional:    true,
							Computed:    true,
						},
						"isolation": schema.BoolAttribute{
							Description: "Enable port isolation.",
							Optional:    true,
							Computed:    true,
						},
						"lldpmed_enabled": schema.BoolAttribute{
							Description: "Enable LLDP-MED.",
							Optional:    true,
							Computed:    true,
						},
						"lldpmed_notify_enabled": schema.BoolAttribute{
							Description: "Enable LLDP-MED notifications.",
							Optional:    true,
							Computed:    true,
						},
						"mirror_port_idx": schema.Int64Attribute{
							Description: "Mirror port index.",
							Optional:    true,
						},
						"multicast_router_networkconf_ids": schema.ListAttribute{
							Description: "List of network IDs for multicast router.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"native_networkconf_id": schema.StringAttribute{
							Description: "Native network ID (VLAN).",
							Optional:    true,
						},
						"port_keepalive_enabled": schema.BoolAttribute{
							Description: "Enable port keepalive.",
							Optional:    true,
							Computed:    true,
						},
						"port_security_enabled": schema.BoolAttribute{
							Description: "Enable port security.",
							Optional:    true,
							Computed:    true,
						},
						"port_security_mac_address": schema.ListAttribute{
							Description: "List of MAC addresses allowed when port security is enabled.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"priority_queue1_level": schema.Int64Attribute{
							Description: "Priority queue 1 level.",
							Optional:    true,
						},
						"priority_queue2_level": schema.Int64Attribute{
							Description: "Priority queue 2 level.",
							Optional:    true,
						},
						"priority_queue3_level": schema.Int64Attribute{
							Description: "Priority queue 3 level.",
							Optional:    true,
						},
						"priority_queue4_level": schema.Int64Attribute{
							Description: "Priority queue 4 level.",
							Optional:    true,
						},
						"setting_preference": schema.StringAttribute{
							Description: "Setting preference.",
							Optional:    true,
						},
						"speed": schema.Int64Attribute{
							Description: "Port speed in Mbps.",
							Optional:    true,
						},
						"stormctrl_bcast_enabled": schema.BoolAttribute{
							Description: "Enable broadcast storm control.",
							Optional:    true,
							Computed:    true,
						},
						"stormctrl_bcast_level": schema.Int64Attribute{
							Description: "Broadcast storm control level.",
							Optional:    true,
						},
						"stormctrl_bcast_rate": schema.Int64Attribute{
							Description: "Broadcast storm control rate.",
							Optional:    true,
						},
						"stormctrl_mcast_enabled": schema.BoolAttribute{
							Description: "Enable multicast storm control.",
							Optional:    true,
							Computed:    true,
						},
						"stormctrl_mcast_level": schema.Int64Attribute{
							Description: "Multicast storm control level.",
							Optional:    true,
						},
						"stormctrl_mcast_rate": schema.Int64Attribute{
							Description: "Multicast storm control rate.",
							Optional:    true,
						},
						"stormctrl_type": schema.StringAttribute{
							Description: "Storm control type.",
							Optional:    true,
						},
						"stormctrl_ucast_enabled": schema.BoolAttribute{
							Description: "Enable unicast storm control.",
							Optional:    true,
							Computed:    true,
						},
						"stormctrl_ucast_level": schema.Int64Attribute{
							Description: "Unicast storm control level.",
							Optional:    true,
						},
						"stormctrl_ucast_rate": schema.Int64Attribute{
							Description: "Unicast storm control rate.",
							Optional:    true,
						},
						"stp_port_mode": schema.BoolAttribute{
							Description: "STP port mode.",
							Optional:    true,
							Computed:    true,
						},
						"tagged_networkconf_ids": schema.ListAttribute{
							Description: "List of network IDs to tag on this port.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"tagged_vlan_mgmt": schema.StringAttribute{
							Description: "Tagged VLAN management.",
							Optional:    true,
						},
						"voice_networkconf_id": schema.StringAttribute{
							Description: "Voice network ID.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

// UpgradeState migrates v0 state to v1: lcm_idle_timeout and each
// port_override.dot1x_idle_timeout changed from integer seconds to GoDuration
// strings.
func (r *deviceResource) UpgradeState(
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
				dv, err := util.UpgradeDurationRawState(
					schemaType,
					req.RawState.JSON,
					func(state map[string]any) {
						util.SetDurationField(state, "lcm_idle_timeout", time.Second)
						if pos, ok := state["port_override"].([]any); ok {
							for _, p := range pos {
								if pm, ok := p.(map[string]any); ok {
									util.SetDurationField(pm, "dot1x_idle_timeout", time.Second)
								}
							}
						}
					},
				)
				if err != nil {
					resp.Diagnostics.AddError("Failed to upgrade device state", err.Error())
					return
				}
				resp.DynamicValue = dv
			},
		},
	}
}

func (r *deviceResource) Configure(
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

func (r *deviceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan deviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
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

	mac := plan.MAC.ValueString()
	if mac == "" {
		resp.Diagnostics.AddError(
			"MAC Address Required",
			"No MAC address specified, please import the device using terraform import",
		)
		return
	}

	mac = cleanMAC(mac)
	device := new(unifi.Device)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		d, err := r.client.GetDeviceByMAC(ctx, site, mac)
		if err != nil {
			return retry.RetryableError(err)
		}
		device = d
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device with MAC %s: %s", mac, err),
		)
		return
	}

	if device == nil {
		resp.Diagnostics.AddError(
			"Device Not Found",
			fmt.Sprintf("Device not found using mac %s", mac),
		)
		return
	}

	if !device.Adopted {
		allowAdoption := plan.AllowAdoption.ValueBool()
		if !allowAdoption {
			resp.Diagnostics.AddError(
				"Device Not Adopted",
				"Device must be adopted before it can be managed",
			)
			return
		}

		err := r.client.AdoptDevice(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Adopting Device",
				fmt.Sprintf("Could not adopt device with MAC %s: %s", mac, err),
			)
			return
		}

		d, err := r.waitForDeviceState(
			ctx,
			site, mac,
			unifi.DeviceStateConnected,
			[]unifi.DeviceState{
				unifi.DeviceStateAdopting,
				unifi.DeviceStatePending,
				unifi.DeviceStateProvisioning,
				unifi.DeviceStateUpgrading,
			},
			3*time.Minute,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Waiting for Device Adoption",
				fmt.Sprintf("Could not wait for device adoption: %s", err),
			)
			return
		}
		device = d
	}

	plan.ID = types.StringValue(device.ID)
	plan.Site = types.StringValue(site)
	plan.Adopted = types.BoolValue(true)

	// Save plan-only values before they get overwritten later.
	allowAdoption := plan.AllowAdoption
	forgetOnDestroy := plan.ForgetOnDestroy
	plannedPortOverride := plan.PortOverride

	// Set Type from the API so updateDevice can include it in the PUT body.
	// We deliberately do NOT call setResourceData here — it would fill the model
	// with ALL device fields (radio_table, outlet_overrides, etc.) which then get
	// serialized into the PUT body and cause "not found" errors from the API.
	plan.Type = types.StringValue(device.Type)
	plan.Model = types.StringValue(device.Model)

	if plan.ConfigNetwork.IsNull() || plan.ConfigNetwork.IsUnknown() {
		plan.ConfigNetwork = types.ObjectNull(configNetworkAttrTypes())
	}

	// Apply the update operation (sends only user-configured fields + type/model)
	diags = r.updateDevice(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-read the device from the API to get the actual state after create/update,
	// but preserve port_override from the plan. The API returns ALL ports (e.g. 32)
	// while the plan only contains the ports we manage (e.g. 27). Terraform's
	// post-apply consistency check requires the returned set to match the plan.
	freshDevice, _ := r.client.GetDeviceByMAC(ctx, site, plan.MAC.ValueString())
	if freshDevice == nil {
		freshDevice, _ = r.client.GetDevice(ctx, site, plan.ID.ValueString())
	}
	if freshDevice != nil {
		r.setResourceData(ctx, &resp.Diagnostics, freshDevice, &plan, site)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	// Restore port_override from plan. The API returns ALL ports (e.g. 32) but we
	// only manage a subset (e.g. 27). Terraform's post-apply consistency check
	// requires the set length to match the plan. On subsequent Read, the full
	// port state will be loaded, which may cause a one-time update on next apply.
	plan.PortOverride = plannedPortOverride

	// Restore plan-only flags
	plan.AllowAdoption = allowAdoption
	plan.ForgetOnDestroy = forgetOnDestroy

	// Set state
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *deviceResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state deviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
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

	// Preserve plan-only flags and port_override before reading API state.
	// Port_override is a Set where ALL fields contribute to identity. The API
	// returns all ports with all fields, but the user only configures a subset.
	// Keeping the user's minimal set in state prevents phantom drift on every plan.
	allowAdoption := state.AllowAdoption
	forgetOnDestroy := state.ForgetOnDestroy
	priorPortOverride := state.PortOverride

	id := state.ID.ValueString()
	mac := state.MAC.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Prefer GetDeviceByMAC (stat/device/{mac}) because stat/device/{id} doesn't
	// work through the cloud connector proxy. Fall back to GetDevice for local setups.
	var device *unifi.Device
	var err error
	if mac != "" {
		device, err = r.client.GetDeviceByMAC(ctx, site, mac)
	}
	if device == nil || err != nil {
		device, err = r.client.GetDevice(ctx, site, id)
	}
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device %s (mac %s): %s", id, mac, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, &resp.Diagnostics, device, &state, site)
	if resp.Diagnostics.HasError() {
		return
	}

	// Restore plan-only flags. These are write-only and never returned by the
	// API, so keep the prior value. After an import the prior value is null/unknown;
	// normalize it to the schema default (true) so a later apply does not see a
	// null and report an inconsistent result.
	if allowAdoption.IsNull() || allowAdoption.IsUnknown() {
		state.AllowAdoption = types.BoolValue(true)
	} else {
		state.AllowAdoption = allowAdoption
	}
	if forgetOnDestroy.IsNull() || forgetOnDestroy.IsUnknown() {
		state.ForgetOnDestroy = types.BoolValue(true)
	} else {
		state.ForgetOnDestroy = forgetOnDestroy
	}

	// Reconcile port_override: the API returns all ports with all fields, but
	// the user only configures a subset. Rebuild state from the API response
	// using only the ports/fields the user configured, so drift is detectable.
	// If the user configured no port_overrides, keep state null so Terraform
	// doesn't plan to remove ports it doesn't manage.
	if priorPortOverride.IsNull() || priorPortOverride.IsUnknown() {
		state.PortOverride = priorPortOverride
	} else {
		reconciled, reconcileDiags := r.reconcilePortOverrides(
			ctx,
			priorPortOverride,
			device.PortOverrides,
		)
		resp.Diagnostics.Append(reconcileDiags...)
		if !resp.Diagnostics.HasError() {
			state.PortOverride = reconciled
		}
	}

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), state.ID)...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *deviceResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan deviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
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

	var state deviceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current device state and merge with planned changes
	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	mac := state.MAC.ValueString()

	// Prefer MAC lookup (works through cloud connector). Fall back to ID lookup.
	var currentDevice *unifi.Device
	var err error
	if mac != "" {
		currentDevice, err = r.client.GetDeviceByMAC(ctx, site, mac)
	}
	if currentDevice == nil || err != nil {
		currentDevice, err = r.client.GetDevice(ctx, site, id)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Device for Update",
			fmt.Sprintf("Could not read device %s for update: %s", id, err),
		)
		return
	}

	// Set type/model on plan from current device (required by API for PUT).
	// We deliberately skip setResourceData to avoid filling the model with ALL
	// device fields which would bloat the PUT body and cause API errors.
	if plan.Type.IsNull() || plan.Type.IsUnknown() {
		plan.Type = types.StringValue(currentDevice.Type)
	}
	if plan.Model.IsNull() || plan.Model.IsUnknown() {
		plan.Model = types.StringValue(currentDevice.Model)
	}
	plan.ID = state.ID

	// Preserve plan-only flags. They are write-only (config or default); keep the
	// planned value and only fall back to prior state if the plan left them unset.
	// Do NOT unconditionally overwrite with state — a null state (e.g. after an
	// import) would clobber the planned `true` and produce an inconsistent result.
	if plan.AllowAdoption.IsNull() || plan.AllowAdoption.IsUnknown() {
		plan.AllowAdoption = state.AllowAdoption
	}
	if plan.ForgetOnDestroy.IsNull() || plan.ForgetOnDestroy.IsUnknown() {
		plan.ForgetOnDestroy = state.ForgetOnDestroy
	}

	if plan.ConfigNetwork.IsNull() || plan.ConfigNetwork.IsUnknown() {
		plan.ConfigNetwork = types.ObjectNull(configNetworkAttrTypes())
	}

	// Save planned port overrides for post-update restore
	plannedPortOverride := plan.PortOverride

	// Save planned LED overrides too. The controller applies these to APs
	// asynchronously, so the immediate post-update read can still report the old
	// values, which would conflict with the plan (#337). Re-assert the planned
	// value (when known) after the read; the next refresh reconciles state with
	// the controller once the AP has applied it.
	plannedLedOverride := plan.LedOverride
	plannedLedOverrideColor := plan.LedOverrideColor
	plannedLedOverrideColorBrightness := plan.LedOverrideColorBrightness

	// Update the device with only user-configured fields
	diags = r.updateDevice(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-read the full device state after update
	freshDevice, _ := r.client.GetDeviceByMAC(ctx, site, mac)
	if freshDevice == nil {
		freshDevice, _ = r.client.GetDevice(ctx, site, id)
	}
	if freshDevice != nil {
		r.setResourceData(ctx, &resp.Diagnostics, freshDevice, &plan, site)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Restore port_override from plan (API returns all ports, plan has subset)
	plan.PortOverride = plannedPortOverride

	// Re-assert the planned LED values when the user configured them, so an
	// asynchronously-applied controller value doesn't trip the consistency
	// check (#337). The next Read converges state with the controller.
	if !plannedLedOverride.IsNull() && !plannedLedOverride.IsUnknown() {
		plan.LedOverride = plannedLedOverride
	}
	if !plannedLedOverrideColor.IsNull() && !plannedLedOverrideColor.IsUnknown() {
		plan.LedOverrideColor = plannedLedOverrideColor
	}
	if !plannedLedOverrideColorBrightness.IsNull() &&
		!plannedLedOverrideColorBrightness.IsUnknown() {
		plan.LedOverrideColorBrightness = plannedLedOverrideColorBrightness
	}
	// allow_adoption / forget_on_destroy were resolved before the update and are
	// not touched by setResourceData; ensure a concrete value (default true)
	// rather than overwriting the planned value with prior state.
	if plan.AllowAdoption.IsNull() || plan.AllowAdoption.IsUnknown() {
		plan.AllowAdoption = types.BoolValue(true)
	}
	if plan.ForgetOnDestroy.IsNull() || plan.ForgetOnDestroy.IsUnknown() {
		plan.ForgetOnDestroy = types.BoolValue(true)
	}

	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), plan.ID)...)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *deviceResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state deviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
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

	if !state.ForgetOnDestroy.ValueBool() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	mac := state.MAC.ValueString()
	err := r.client.ForgetDevice(ctx, site, mac)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Forgetting Device",
			fmt.Sprintf("Could not forget device with MAC %s: %s", mac, err),
		)
		return
	}

	_, err = r.waitForDeviceState(
		ctx,
		site, mac,
		unifi.DeviceStatePending,
		[]unifi.DeviceState{unifi.DeviceStateConnected, unifi.DeviceStateDeleting},
		3*time.Minute,
	)
	if _, ok := err.(*unifi.NotFoundError); !ok && err != nil {
		resp.Diagnostics.AddError(
			"Error Waiting for Device Forget",
			fmt.Sprintf("Could not wait for device forget: %s", err),
		)
		return
	}
}

func (r *deviceResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	importID := req.ID
	mac := cleanMAC(importID)
	site := r.client.Site

	normalizeMAC := func(m string) string {
		m = strings.ToLower(m)
		m = strings.ReplaceAll(m, ":", "")
		m = strings.ReplaceAll(m, "-", "")
		return m
	}

	var device *unifi.Device
	var getErr, listErr error
	var deviceCount int

	// The import identifier may be either a MAC address or the controller's
	// internal device ID (the value of the `id` attribute, which is what
	// ImportStateVerify replays). Try a MAC lookup first, then fall back to an
	// ID lookup, then to scanning the device list matching on either.
	device, getErr = r.client.GetDeviceByMAC(ctx, site, mac)
	if (getErr != nil || device == nil || device.ID == "") && importID != "" {
		if d, idErr := r.client.GetDevice(ctx, site, importID); idErr == nil && d != nil &&
			d.ID != "" {
			device = d
			getErr = nil
		}
	}
	if getErr != nil || device == nil || device.ID == "" {
		if getErr != nil {
			tflog.Warn(ctx, "GetDeviceByMAC failed, falling back to device list",
				map[string]any{"mac": mac, "error": getErr.Error()})
		}

		// Fallback: list all devices and match by MAC or internal ID. This also
		// avoids the full JSON deserialization that can fail on some device types.
		devices, err := r.client.ListDevice(ctx, site)
		listErr = err
		if listErr != nil {
			resp.Diagnostics.AddError(
				"Error Listing Devices",
				fmt.Sprintf(
					"Could not list devices to find %s: %s (original error: %v)",
					importID,
					listErr,
					getErr,
				),
			)
			return
		}

		deviceCount = len(devices)
		normalizedImport := normalizeMAC(mac)
		for _, d := range devices {
			if normalizeMAC(d.MAC) == normalizedImport || d.ID == importID {
				device = &d
				break
			}
		}
	}

	if device == nil || device.ID == "" {
		var macList []string
		if deviceCount > 0 {
			devices, _ := r.client.ListDevice(ctx, site)
			for _, d := range devices {
				macList = append(macList, fmt.Sprintf("%s (id=%s)", d.MAC, d.ID))
			}
		}
		resp.Diagnostics.AddError(
			"Device Not Found",
			fmt.Sprintf(
				"No device found matching %q (tried MAC and internal ID) on site %s. GetDeviceByMAC error: %v. ListDevice found %d device(s): %v",
				importID,
				site,
				getErr,
				deviceCount,
				macList,
			),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), device.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mac"), device.MAC)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
}

// Helper methods

// buildMinimalUpdateDevice assembles the Device sent in an update PUT. The full
// Device struct carries computed fields (adopted, state, …) whose Go zero-values
// the API rejects, so only the user-configured / API-required fields are sent.
// LED overrides MUST be included: omitting them makes the controller keep the
// old values, so the post-apply read conflicts with the plan (#337). All of the
// LED fields are `omitempty`, so unset ones are dropped from the body.
//
// mgmt_network_id (the UI "Network Override") is likewise user-configurable and
// must be carried here: the earlier hand-listed body dropped it, so the
// controller never received the value and the per-device management VLAN could
// not be set through the provider (#329). modelToAPIDevice only sets it when
// configured, and it is `omitempty`, so a null value stays off the wire and
// never reintroduces the #177 zero-value rejection.
//
// switch_vlan_enabled (the UI "Port VLAN" toggle, needed on APs with a built-in
// switch to make VLAN tagging take effect on the built-in ports) is the same
// story: the hand-listed body dropped it, so the controller kept its old value
// and the post-apply read conflicted with a configured `true`. It is `omitempty`,
// so a `false` stays off the wire and doesn't disturb the controller default.
//
// radio_table and mesh_sta_vap_enabled are the same bug class for the mesh
// toggles. radio_table[].vwire_enabled (the UI "Mesh Parent" toggle) is fully
// wired through the schema and converters, but the hand-listed body never copied
// radio_table across, so every radio_table sub-field — vwire_enabled included —
// was dropped from the PUT. mesh_sta_vap_enabled (the top-level "Mesh Connect"
// toggle) is likewise carried here. Both are `omitempty` (radio_table at the
// device level, and every DeviceRadioTable sub-field), so an unset table or a
// false/zero sub-field stays off the wire. Note that when radio_table is in the
// plan (user-configured, or state-inherited via the list's UseStateForUnknown),
// its non-zero sub-fields (channel, tx_power, …) also travel in the PUT; sending
// back the values the controller already returned is idempotent.
func buildMinimalUpdateDevice(
	deviceReq, currentDevice *unifi.Device,
	portOverrides []unifi.DevicePortOverrides,
) *unifi.Device {
	minimalDevice := &unifi.Device{
		ID:                         deviceReq.ID,
		Type:                       deviceReq.Type,
		MAC:                        deviceReq.MAC,
		Name:                       deviceReq.Name,
		PortOverrides:              portOverrides,
		MgmtNetworkID:              deviceReq.MgmtNetworkID,
		LedOverride:                deviceReq.LedOverride,
		LedOverrideColor:           deviceReq.LedOverrideColor,
		LedOverrideColorBrightness: deviceReq.LedOverrideColorBrightness,
		SwitchVLANEnabled:          deviceReq.SwitchVLANEnabled,
		MeshStaVapEnabled:          deviceReq.MeshStaVapEnabled,
		RadioTable:                 deviceReq.RadioTable,
	}
	if currentDevice != nil {
		minimalDevice.State = currentDevice.State
		minimalDevice.Adopted = currentDevice.Adopted
	}
	return minimalDevice
}

func (r *deviceResource) updateDevice(
	ctx context.Context,
	model *deviceResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	site := model.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert model to API request
	deviceReq, convDiags := r.modelToAPIDevice(ctx, model)
	diags.Append(convDiags...)
	if diags.HasError() {
		return diags
	}

	deviceReq.ID = model.ID.ValueString()

	// Fetch the current device once. We need it for two reasons:
	//   1. 'type' is a computed field the API requires in the PUT body.
	//   2. UpdateDevice sends a diff against the existing device. The Device
	//      struct marshals `state` and `adopted` without omitempty, so leaving
	//      them at their Go zero-values makes the diff try to reset state to 0
	//      and adopted to false — which UDM/Dream Machine gateways reject with
	//      api.err.Invalid (issue #177). Echo the current values so they don't
	//      appear in the diff.
	// Prefer MAC lookup (works through cloud connector); fall back to ID (local).
	var currentDevice *unifi.Device
	if deviceReq.MAC != "" {
		currentDevice, _ = r.client.GetDeviceByMAC(ctx, site, deviceReq.MAC)
	}
	if currentDevice == nil && deviceReq.ID != "" {
		currentDevice, _ = r.client.GetDevice(ctx, site, deviceReq.ID)
	}
	if currentDevice == nil {
		// Everything below (state/adopted echo, port_overrides echo) depends on
		// currentDevice. Proceeding without it would leave state/adopted at their Go
		// zero-values in the PUT body — precisely the condition that triggers
		// api.err.InvalidPayload (400) on UDM/Dream Machine gateways that this PR
		// exists to fix (#177/#150). Fail fast with a clear diagnostic instead of
		// silently reintroducing that failure mode (review feedback on PR #378).
		diags.AddError(
			"Error Updating Device",
			fmt.Sprintf(
				"Could not fetch the current device (mac=%q, id=%q) on site %q before updating — "+
					"refusing to send an update with state/adopted left unset, which UDM/Dream "+
					"Machine gateways reject. Verify the device still exists on this site and that "+
					"its MAC/ID are correct.",
				deviceReq.MAC, deviceReq.ID, site,
			),
		)
		return diags
	}
	if deviceReq.Type == "" {
		deviceReq.Type = currentDevice.Type
	}

	// The UniFi PUT treats port_overrides as a full-replace array. Sending only the
	// user-declared subset would wipe every other port's override (#266). When the
	// user declared at least one port_override, merge the declared blocks (by
	// port_idx) onto the device's current overrides so undeclared ports keep their
	// existing controller-side config — i.e. partial management of just the declared
	// ports. With no override declared we echo the controller's current overrides
	// (below) so the diff never emits `port_overrides: null`, which UDM/Dream Machine
	// gateways reject.
	portOverrides := deviceReq.PortOverrides
	if len(deviceReq.PortOverrides) > 0 {
		portOverrides = mergePortOverridesByIndex(
			currentDevice.PortOverrides,
			deviceReq.PortOverrides,
		)
	} else {
		// No port_override blocks are managed in config (e.g. gateways/APs and
		// switches we only touch for name/LED/radio). Echo the controller's current
		// overrides so the diff doesn't emit `port_overrides: null`, which UDM/Dream
		// Machine gateways reject with api.err.InvalidPayload (400) (#177).
		portOverrides = currentDevice.PortOverrides
	}

	minimalDevice := buildMinimalUpdateDevice(deviceReq, currentDevice, portOverrides)

	if reqJSON, jsonErr := json.Marshal(minimalDevice); jsonErr == nil {
		tflog.Info(ctx, "Sending device update", map[string]any{
			"id": minimalDevice.ID, "type": minimalDevice.Type, "mac": minimalDevice.MAC,
			"body_length": len(reqJSON),
		})
	}

	device, err := r.client.UpdateDevice(ctx, site, minimalDevice)
	if err != nil {
		diags.AddError(
			"Error Updating Device",
			fmt.Sprintf("Could not update device: %s", err),
		)
		return diags
	}

	// Wait for device to be in connected state
	if d, err := r.waitForDeviceState(
		ctx,
		site, device.MAC,
		unifi.DeviceStateConnected,
		[]unifi.DeviceState{unifi.DeviceStateAdopting, unifi.DeviceStateProvisioning},
		3*time.Minute,
	); err != nil {
		diags.AddError(
			"Error Waiting for Device Update",
			fmt.Sprintf("Could not wait for device update: %s", err),
		)
		return diags
	} else {
		device = d
	}

	// Update state from API response
	r.setResourceData(ctx, &diags, device, model, site)
	return diags
}

func (r *deviceResource) setResourceData(
	ctx context.Context,
	diags *diag.Diagnostics,
	device *unifi.Device,
	model *deviceResourceModel,
	site string,
) {
	// Set core identity fields
	model.ID = types.StringValue(device.ID)
	model.Site = types.StringValue(site)

	if device.MAC == "" {
		model.MAC = hwtypes.NewMACAddressNull()
	} else {
		model.MAC = hwtypes.NewMACAddressValue(device.MAC)
	}

	if device.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(device.Name)
	}

	model.Disabled = types.BoolValue(device.Disabled)
	model.Adopted = types.BoolValue(device.Adopted)

	if device.Model == "" {
		model.Model = types.StringNull()
	} else {
		model.Model = types.StringValue(device.Model)
	}

	if device.Type == "" {
		model.Type = types.StringNull()
	} else {
		model.Type = types.StringValue(device.Type)
	}

	// State is always present as int64
	model.State = types.Int64Value(int64(device.State))

	// LED settings — write-only in practice: the controller frequently does not
	// echo these back. When the API returns an empty value, preserve the
	// configured/known value from the model instead of nulling it (which would
	// trigger "inconsistent result after apply"). Only resolve unknowns to null.
	if device.LedOverride != "" {
		model.LedOverride = types.StringValue(device.LedOverride)
	} else if model.LedOverride.IsUnknown() {
		model.LedOverride = types.StringNull()
	}

	if device.LedOverrideColor != "" {
		model.LedOverrideColor = types.StringValue(device.LedOverrideColor)
	} else if model.LedOverrideColor.IsUnknown() {
		model.LedOverrideColor = types.StringNull()
	}

	if device.LedOverrideColorBrightness != nil {
		model.LedOverrideColorBrightness = types.Int64PointerValue(
			device.LedOverrideColorBrightness,
		)
	} else if model.LedOverrideColorBrightness.IsUnknown() {
		model.LedOverrideColorBrightness = types.Int64Null()
	}

	// Device features
	if device.BandsteeringMode == "" {
		model.BandsteeringMode = types.StringNull()
	} else {
		model.BandsteeringMode = types.StringValue(device.BandsteeringMode)
	}

	model.FlowctrlEnabled = types.BoolValue(device.FlowctrlEnabled)
	model.JumboframeEnabled = types.BoolValue(device.JumboframeEnabled)

	if device.StpVersion == "" {
		model.StpVersion = types.StringNull()
	} else {
		model.StpVersion = types.StringValue(device.StpVersion)
	}

	model.StpPriority = types.Int64PointerValue(device.StpPriority)

	model.Locked = types.BoolValue(device.Locked)

	// PoE settings
	if device.PoeMode == "" {
		model.PoeMode = types.StringNull()
	} else {
		model.PoeMode = types.StringValue(device.PoeMode)
	}

	// VLAN
	model.SwitchVLANEnabled = types.BoolValue(device.SwitchVLANEnabled)

	// Mesh
	model.MeshStaVapEnabled = types.BoolValue(device.MeshStaVapEnabled)

	// Advanced features
	if device.OutdoorModeOverride == "" {
		model.OutdoorModeOverride = types.StringNull()
	} else {
		model.OutdoorModeOverride = types.StringValue(device.OutdoorModeOverride)
	}

	model.Volume = types.Int64PointerValue(device.Volume)

	if device.BaresipPassword == "" {
		model.BaresipPassword = types.StringNull()
	} else {
		model.BaresipPassword = types.StringValue(device.BaresipPassword)
	}

	// LCD/LCM settings
	model.LcmBrightness = types.Int64PointerValue(device.LcmBrightness)

	model.LcmBrightnessOverride = types.BoolValue(device.LcmBrightnessOverride)

	model.LcmIDleTimeout = util.DurationPtrValue(device.LcmIDleTimeout, time.Second)

	model.LcmIDleTimeoutOverride = types.BoolValue(device.LcmIDleTimeoutOverride)

	if device.LcmNightModeBegins == "" {
		model.LcmNightModeBegins = types.StringNull()
	} else {
		model.LcmNightModeBegins = types.StringValue(device.LcmNightModeBegins)
	}

	if device.LcmNightModeEnds == "" {
		model.LcmNightModeEnds = types.StringNull()
	} else {
		model.LcmNightModeEnds = types.StringValue(device.LcmNightModeEnds)
	}

	// Outlet settings
	model.OutletEnabled = types.BoolValue(device.OutletEnabled)

	// Management
	if device.MgmtNetworkID == "" {
		model.MgmtNetworkID = types.StringNull()
	} else {
		model.MgmtNetworkID = types.StringValue(device.MgmtNetworkID)
	}

	// Convert config network
	configNetwork, convDiags := r.configNetworkToFramework(ctx, device.ConfigNetwork)
	diags.Append(convDiags...)
	if !diags.HasError() {
		model.ConfigNetwork = configNetwork
	}

	// Convert port overrides
	portOverrides, convDiags := r.portOverridesToFramework(ctx, device.PortOverrides)
	diags.Append(convDiags...)
	if !diags.HasError() {
		model.PortOverride = portOverrides
	}

	// Convert radio table
	radioTable, convDiags := r.radioTableToFramework(ctx, device.RadioTable)
	diags.Append(convDiags...)
	if !diags.HasError() {
		model.RadioTable = radioTable
	}

	// Convert outlet overrides
	outletOverrides, convDiags := r.outletOverridesToFramework(ctx, device.OutletOverrides)
	diags.Append(convDiags...)
	if !diags.HasError() {
		model.OutletOverrides = outletOverrides
	}
}

func (r *deviceResource) modelToAPIDevice(
	ctx context.Context,
	model *deviceResourceModel,
) (*unifi.Device, diag.Diagnostics) {
	var diags diag.Diagnostics

	device := &unifi.Device{
		MAC:  model.MAC.ValueString(),
		Name: model.Name.ValueString(),
	}

	if !model.Type.IsNull() && !model.Type.IsUnknown() {
		device.Type = model.Type.ValueString()
	}
	if !model.Model.IsNull() && !model.Model.IsUnknown() {
		device.Model = model.Model.ValueString()
	}

	// Only set Disabled if it's explicitly configured
	if !model.Disabled.IsNull() && !model.Disabled.IsUnknown() {
		device.Disabled = model.Disabled.ValueBool()
	}

	// LED settings
	if !model.LedOverride.IsNull() {
		device.LedOverride = model.LedOverride.ValueString()
	}
	if !model.LedOverrideColor.IsNull() {
		device.LedOverrideColor = model.LedOverrideColor.ValueString()
	}
	if !model.LedOverrideColorBrightness.IsNull() && !model.LedOverrideColorBrightness.IsUnknown() {
		device.LedOverrideColorBrightness = model.LedOverrideColorBrightness.ValueInt64Pointer()
	}

	// Device features
	if !model.BandsteeringMode.IsNull() {
		device.BandsteeringMode = model.BandsteeringMode.ValueString()
	}
	device.FlowctrlEnabled = model.FlowctrlEnabled.ValueBool()
	device.JumboframeEnabled = model.JumboframeEnabled.ValueBool()
	if !model.StpVersion.IsNull() {
		device.StpVersion = model.StpVersion.ValueString()
	}
	if !model.StpPriority.IsNull() && !model.StpPriority.IsUnknown() {
		device.StpPriority = model.StpPriority.ValueInt64Pointer()
	}
	device.Locked = model.Locked.ValueBool()

	// PoE settings
	if !model.PoeMode.IsNull() {
		device.PoeMode = model.PoeMode.ValueString()
	}

	// VLAN
	device.SwitchVLANEnabled = model.SwitchVLANEnabled.ValueBool()

	// Mesh
	device.MeshStaVapEnabled = model.MeshStaVapEnabled.ValueBool()

	// Advanced features
	if !model.OutdoorModeOverride.IsNull() {
		device.OutdoorModeOverride = model.OutdoorModeOverride.ValueString()
	}
	if !model.Volume.IsNull() && !model.Volume.IsUnknown() {
		device.Volume = model.Volume.ValueInt64Pointer()
	}
	if !model.BaresipPassword.IsNull() {
		device.BaresipPassword = model.BaresipPassword.ValueString()
	}

	// LCD/LCM settings
	if !model.LcmBrightness.IsNull() && !model.LcmBrightness.IsUnknown() {
		device.LcmBrightness = model.LcmBrightness.ValueInt64Pointer()
	}
	device.LcmBrightnessOverride = model.LcmBrightnessOverride.ValueBool()
	if !model.LcmIDleTimeout.IsNull() && !model.LcmIDleTimeout.IsUnknown() {
		device.LcmIDleTimeout = util.DurationUnitsPtr(model.LcmIDleTimeout, time.Second)
	}
	device.LcmIDleTimeoutOverride = model.LcmIDleTimeoutOverride.ValueBool()
	if !model.LcmNightModeBegins.IsNull() {
		device.LcmNightModeBegins = model.LcmNightModeBegins.ValueString()
	}
	if !model.LcmNightModeEnds.IsNull() {
		device.LcmNightModeEnds = model.LcmNightModeEnds.ValueString()
	}

	// Outlet settings
	device.OutletEnabled = model.OutletEnabled.ValueBool()

	// Management
	if !model.MgmtNetworkID.IsNull() {
		device.MgmtNetworkID = model.MgmtNetworkID.ValueString()
	}

	// Convert config network
	if !model.ConfigNetwork.IsNull() && !model.ConfigNetwork.IsUnknown() {
		configNetwork, convDiags := r.frameworkToConfigNetwork(ctx, model.ConfigNetwork)
		diags.Append(convDiags...)
		if !diags.HasError() {
			device.ConfigNetwork = configNetwork
		}
	}

	// Convert port overrides
	if !model.PortOverride.IsNull() && !model.PortOverride.IsUnknown() {
		portOverrides, convDiags := r.frameworkToPortOverrides(ctx, model.PortOverride)
		diags.Append(convDiags...)
		if !diags.HasError() {
			device.PortOverrides = portOverrides
		}
	}

	// Convert radio table
	if !model.RadioTable.IsNull() && !model.RadioTable.IsUnknown() {
		radioTable, convDiags := r.frameworkToRadioTable(ctx, model.RadioTable)
		diags.Append(convDiags...)
		if !diags.HasError() {
			device.RadioTable = radioTable
		}
	}

	// Convert outlet overrides
	if !model.OutletOverrides.IsNull() && !model.OutletOverrides.IsUnknown() {
		outletOverrides, convDiags := r.frameworkToOutletOverrides(ctx, model.OutletOverrides)
		diags.Append(convDiags...)
		if !diags.HasError() {
			device.OutletOverrides = outletOverrides
		}
	}

	if !model.Adopted.IsNull() && !model.Adopted.IsUnknown() {
		device.Adopted = model.Adopted.ValueBool()
	}

	return device, diags
}

// mergePortOverridesByIndex overlays the user-declared port overrides onto the
// device's current overrides, keyed by port_idx. The UniFi PUT replaces the whole
// port_overrides array, so to manage only a subset of ports without clobbering the
// rest (#266) we start from what the controller already has and replace just the
// declared ports. Ports present only in the current set are preserved; ports
// declared but not yet present are appended. Declared order is preserved for the
// appended entries so the result is deterministic.
func mergePortOverridesByIndex(
	current, declared []unifi.DevicePortOverrides,
) []unifi.DevicePortOverrides {
	if len(declared) == 0 {
		return current
	}

	declaredByIdx := make(map[int64]int, len(declared))
	for i, po := range declared {
		if po.PortIDX != nil {
			declaredByIdx[*po.PortIDX] = i
		}
	}

	merged := make([]unifi.DevicePortOverrides, 0, len(current)+len(declared))
	used := make([]bool, len(declared))
	for _, po := range current {
		if po.PortIDX != nil {
			if i, ok := declaredByIdx[*po.PortIDX]; ok {
				merged = append(merged, declared[i])
				used[i] = true
				continue
			}
		}
		merged = append(merged, po)
	}
	// Append declared ports not already merged: newly-managed ports, or any entry
	// without a port_idx (which we cannot key on).
	for i, po := range declared {
		if !used[i] {
			merged = append(merged, po)
		}
	}
	return merged
}

// reconcilePortOverrides rebuilds the port_override Set from the API response,
// but only for ports and fields that the user explicitly configured. This lets
// Terraform detect drift (e.g. tagged VLANs not applied) without the phantom
// drift caused by computed fields the API adds for every port.
func (r *deviceResource) reconcilePortOverrides(
	ctx context.Context,
	prior types.Set,
	apiOverrides []unifi.DevicePortOverrides,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build a map from port index → API port override for fast lookup.
	apiByIndex := make(map[int64]unifi.DevicePortOverrides, len(apiOverrides))
	for _, po := range apiOverrides {
		if po.PortIDX != nil {
			apiByIndex[*po.PortIDX] = po
		}
	}

	// Iterate over the user-configured (prior) port overrides and rebuild each
	// one using values from the API response for the same port index.
	var priorModels []portOverrideModel
	diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
	if diags.HasError() {
		return prior, diags
	}

	elements := make([]attr.Value, 0, len(priorModels))
	for _, pm := range priorModels {
		idx := pm.Index.ValueInt64()
		apiPO, found := apiByIndex[idx]
		if !found {
			// Port not in API response — keep prior value unchanged.
			objVal, objDiags := types.ObjectValueFrom(ctx, pm.AttributeTypes(), pm)
			diags.Append(objDiags...)
			elements = append(elements, objVal)
			continue
		}

		// Build a new model seeded from the prior (user config), then update
		// only the fields that were explicitly set (non-null in prior) with
		// the actual API value so drift is visible.
		updated := pm

		if !pm.Name.IsNull() {
			if apiPO.Name == "" {
				updated.Name = types.StringNull()
			} else {
				updated.Name = types.StringValue(apiPO.Name)
			}
		}
		if !pm.NativeNetworkID.IsNull() {
			if apiPO.NATiveNetworkID == "" {
				updated.NativeNetworkID = types.StringNull()
			} else {
				updated.NativeNetworkID = types.StringValue(apiPO.NATiveNetworkID)
			}
		}
		if !pm.Forward.IsNull() {
			if apiPO.Forward == "" {
				updated.Forward = types.StringNull()
			} else {
				updated.Forward = types.StringValue(apiPO.Forward)
			}
		}
		if !pm.TaggedVLANMgmt.IsNull() {
			if apiPO.TaggedVLANMgmt == "" {
				updated.TaggedVLANMgmt = types.StringNull()
			} else {
				updated.TaggedVLANMgmt = types.StringValue(apiPO.TaggedVLANMgmt)
			}
		}
		if !pm.ExcludedNetworkIDs.IsNull() {
			if len(apiPO.ExcludedNetworkIDs) > 0 {
				sorted := make([]string, len(apiPO.ExcludedNetworkIDs))
				copy(sorted, apiPO.ExcludedNetworkIDs)
				sort.Strings(sorted)
				vals := make([]attr.Value, len(sorted))
				for i, id := range sorted {
					vals[i] = types.StringValue(id)
				}
				listVal, listDiags := types.ListValue(types.StringType, vals)
				diags.Append(listDiags...)
				updated.ExcludedNetworkIDs = listVal
			} else {
				emptyList, listDiags := types.ListValue(types.StringType, []attr.Value{})
				diags.Append(listDiags...)
				updated.ExcludedNetworkIDs = emptyList
			}
		}
		if !pm.PortProfileID.IsNull() {
			if apiPO.PortProfileID == "" {
				updated.PortProfileID = types.StringNull()
			} else {
				updated.PortProfileID = types.StringValue(apiPO.PortProfileID)
			}
		}

		objVal, objDiags := types.ObjectValueFrom(ctx, updated.AttributeTypes(), updated)
		diags.Append(objDiags...)
		elements = append(elements, objVal)
	}

	if diags.HasError() {
		return prior, diags
	}

	setValue, setDiags := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		elements,
	)
	diags.Append(setDiags...)
	if diags.HasError() {
		return prior, diags
	}
	return setValue, diags
}

func (r *deviceResource) portOverridesToFramework(
	ctx context.Context,
	pos []unifi.DevicePortOverrides,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(pos) == 0 {
		return types.SetNull(types.ObjectType{
			AttrTypes: portOverrideAttrTypes(),
		}), diags
	}

	elements := make([]attr.Value, 0, len(pos))
	for _, po := range pos {
		model := portOverrideModel{
			Index: types.Int64PointerValue(po.PortIDX),
		}

		// String attributes
		if po.Name == "" {
			model.Name = types.StringNull()
		} else {
			model.Name = types.StringValue(po.Name)
		}

		if po.PortProfileID == "" {
			model.PortProfileID = types.StringNull()
		} else {
			model.PortProfileID = types.StringValue(po.PortProfileID)
		}

		if po.OpMode == "" {
			model.OpMode = types.StringNull()
		} else {
			model.OpMode = types.StringValue(po.OpMode)
		}

		if po.PoeMode == "" {
			model.PoeMode = types.StringNull()
		} else {
			model.PoeMode = types.StringValue(po.PoeMode)
		}

		if po.Dot1XCtrl == "" {
			model.Dot1XCtrl = types.StringNull()
		} else {
			model.Dot1XCtrl = types.StringValue(po.Dot1XCtrl)
		}

		if po.FecMode == "" {
			model.FecMode = types.StringNull()
		} else {
			model.FecMode = types.StringValue(po.FecMode)
		}

		if po.Forward == "" {
			model.Forward = types.StringNull()
		} else {
			model.Forward = types.StringValue(po.Forward)
		}

		if po.NATiveNetworkID == "" {
			model.NativeNetworkID = types.StringNull()
		} else {
			model.NativeNetworkID = types.StringValue(po.NATiveNetworkID)
		}

		if po.SettingPreference == "" {
			model.SettingPreference = types.StringNull()
		} else {
			model.SettingPreference = types.StringValue(po.SettingPreference)
		}

		if po.StormctrlType == "" {
			model.StormctrlType = types.StringNull()
		} else {
			model.StormctrlType = types.StringValue(po.StormctrlType)
		}

		if po.TaggedVLANMgmt == "" {
			model.TaggedVLANMgmt = types.StringNull()
		} else {
			model.TaggedVLANMgmt = types.StringValue(po.TaggedVLANMgmt)
		}

		if po.VoiceNetworkID == "" {
			model.VoiceNetworkID = types.StringNull()
		} else {
			model.VoiceNetworkID = types.StringValue(po.VoiceNetworkID)
		}

		// Boolean attributes
		model.Autoneg = types.BoolValue(po.Autoneg)
		model.EgressRateLimitKbpsEnabled = types.BoolValue(po.EgressRateLimitKbpsEnabled)
		model.FlowControlEnabled = types.BoolValue(po.FlowControlEnabled)
		model.FullDuplex = types.BoolValue(po.FullDuplex)
		model.Isolation = types.BoolValue(po.Isolation)
		model.LldpmedEnabled = types.BoolValue(po.LldpmedEnabled)
		model.LldpmedNotifyEnabled = types.BoolValue(po.LldpmedNotifyEnabled)
		model.PortKeepaliveEnabled = types.BoolValue(po.PortKeepaliveEnabled)
		model.PortSecurityEnabled = types.BoolValue(po.PortSecurityEnabled)
		model.StormctrlBroadcastEnabled = types.BoolValue(po.StormctrlBroadcastastEnabled)
		model.StormctrlMcastEnabled = types.BoolValue(po.StormctrlMcastEnabled)
		model.StormctrlUcastEnabled = types.BoolValue(po.StormctrlUcastEnabled)
		model.StpPortMode = types.BoolValue(po.StpPortMode)

		// Int64 attributes
		model.Dot1XIDleTimeout = util.DurationPtrValue(po.Dot1XIDleTimeout, time.Second)

		model.EgressRateLimitKbps = types.Int64PointerValue(po.EgressRateLimitKbps)

		model.MirrorPortIDX = types.Int64PointerValue(po.MirrorPortIDX)

		model.PriorityQueue1Level = types.Int64PointerValue(po.PriorityQueue1Level)

		model.PriorityQueue2Level = types.Int64PointerValue(po.PriorityQueue2Level)
		model.PriorityQueue3Level = types.Int64PointerValue(po.PriorityQueue3Level)
		model.PriorityQueue4Level = types.Int64PointerValue(po.PriorityQueue4Level)
		model.Speed = types.Int64PointerValue(po.Speed)

		model.StormctrlBroadcastLevel = types.Int64PointerValue(po.StormctrlBroadcastastLevel)

		model.StormctrlBroadcastRate = types.Int64PointerValue(po.StormctrlBroadcastastRate)

		model.StormctrlMcastLevel = types.Int64PointerValue(po.StormctrlMcastLevel)

		model.StormctrlMcastRate = types.Int64PointerValue(po.StormctrlMcastRate)

		model.StormctrlUcastLevel = types.Int64PointerValue(po.StormctrlUcastLevel)

		model.StormctrlUcastRate = types.Int64PointerValue(po.StormctrlUcastRate)

		// List attributes
		if len(po.AggregateMembers) == 0 {
			model.AggregateMembers = types.ListNull(types.Int64Type)
		} else {
			aggrMemberValues := make([]attr.Value, 0, len(po.AggregateMembers))
			for _, member := range po.AggregateMembers {
				aggrMemberValues = append(aggrMemberValues, types.Int64Value(member))
			}
			listVal, listDiags := types.ListValue(types.Int64Type, aggrMemberValues)
			diags.Append(listDiags...)
			if diags.HasError() {
				continue
			}
			model.AggregateMembers = listVal
		}

		if len(po.ExcludedNetworkIDs) == 0 {
			model.ExcludedNetworkIDs = types.ListNull(types.StringType)
		} else {
			sortedExcluded := make([]string, len(po.ExcludedNetworkIDs))
			copy(sortedExcluded, po.ExcludedNetworkIDs)
			sort.Strings(sortedExcluded)
			excludedValues := make([]attr.Value, 0, len(sortedExcluded))
			for _, id := range sortedExcluded {
				excludedValues = append(excludedValues, types.StringValue(id))
			}
			listVal, listDiags := types.ListValue(types.StringType, excludedValues)
			diags.Append(listDiags...)
			if diags.HasError() {
				continue
			}
			model.ExcludedNetworkIDs = listVal
		}

		// FIX (#235): the pinned go-unifi SDK has no TaggedNetworkIDs field, so
		// nothing populates it below. Without this assignment the model field
		// stays an untyped zero-value types.List, which makes ObjectValueFrom
		// emit a "types.ListType[!!! MISSING TYPE !!!]" Value Conversion Error
		// against the schema's ListAttribute{ElementType: types.StringType}.
		model.TaggedNetworkIDs = types.ListNull(types.StringType)

		if len(po.MulticastRouterNetworkIDs) == 0 {
			model.MulticastRouterNetworkIDs = types.ListNull(types.StringType)
		} else {
			multicastValues := make([]attr.Value, 0, len(po.MulticastRouterNetworkIDs))
			for _, id := range po.MulticastRouterNetworkIDs {
				multicastValues = append(multicastValues, types.StringValue(id))
			}
			listVal, listDiags := types.ListValue(types.StringType, multicastValues)
			diags.Append(listDiags...)
			if diags.HasError() {
				continue
			}
			model.MulticastRouterNetworkIDs = listVal
		}

		if len(po.PortSecurityMACAddress) == 0 {
			model.PortSecurityMACAddress = types.ListNull(types.StringType)
		} else {
			macValues := make([]attr.Value, 0, len(po.PortSecurityMACAddress))
			for _, mac := range po.PortSecurityMACAddress {
				macValues = append(macValues, types.StringValue(mac))
			}
			listVal, listDiags := types.ListValue(types.StringType, macValues)
			diags.Append(listDiags...)
			if diags.HasError() {
				continue
			}
			model.PortSecurityMACAddress = listVal
		}

		// Convert model to object
		objVal, objDiags := types.ObjectValueFrom(ctx, model.AttributeTypes(), model)
		diags.Append(objDiags...)
		if diags.HasError() {
			continue
		}
		elements = append(elements, objVal)
	}

	if diags.HasError() {
		return types.SetNull(types.ObjectType{
			AttrTypes: portOverrideAttrTypes(),
		}), diags
	}

	setValue, setDiags := types.SetValue(types.ObjectType{
		AttrTypes: portOverrideAttrTypes(),
	}, elements)
	diags.Append(setDiags...)
	return setValue, diags
}

func (r *deviceResource) frameworkToPortOverrides(
	ctx context.Context,
	portOverrideSet types.Set,
) ([]unifi.DevicePortOverrides, diag.Diagnostics) {
	var diags diag.Diagnostics

	elements := portOverrideSet.Elements()
	overrideMap := make(map[int64]unifi.DevicePortOverrides)

	for _, elem := range elements {
		var model portOverrideModel
		if elemObj, ok := elem.(types.Object); ok {
			diags.Append(elemObj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return nil, diags
			}

			idx := model.Index.ValueInt64()
			po := unifi.DevicePortOverrides{
				PortIDX: model.Index.ValueInt64Pointer(),
			}

			// String attributes
			if !model.Name.IsNull() {
				po.Name = model.Name.ValueString()
			}
			if !model.PortProfileID.IsNull() {
				po.PortProfileID = model.PortProfileID.ValueString()
			}
			// op_mode is only written when the port runs in a non-default mode
			// (aggregate/mirror). Sending op_mode on a PUT for gateway devices
			// (UDM) is rejected — see #213 — but those ports never use
			// aggregate/mirror, so they stay at the "switch" default and we skip
			// it. Writing it for the non-default cases is required to form an
			// SFP+ link aggregation (#177), which otherwise never engages because
			// aggregate_members is sent without ever switching op_mode.
			if !model.OpMode.IsNull() && !model.OpMode.IsUnknown() &&
				model.OpMode.ValueString() != "" && model.OpMode.ValueString() != "switch" {
				po.OpMode = model.OpMode.ValueString()
			}
			if !model.PoeMode.IsNull() {
				po.PoeMode = model.PoeMode.ValueString()
			}
			if !model.Dot1XCtrl.IsNull() {
				po.Dot1XCtrl = model.Dot1XCtrl.ValueString()
			}
			if !model.FecMode.IsNull() {
				po.FecMode = model.FecMode.ValueString()
			}
			if !model.Forward.IsNull() {
				po.Forward = model.Forward.ValueString()
			}
			if !model.NativeNetworkID.IsNull() {
				po.NATiveNetworkID = model.NativeNetworkID.ValueString()
			}
			if !model.SettingPreference.IsNull() {
				po.SettingPreference = model.SettingPreference.ValueString()
			}
			if !model.StormctrlType.IsNull() {
				po.StormctrlType = model.StormctrlType.ValueString()
			}
			if !model.TaggedVLANMgmt.IsNull() {
				po.TaggedVLANMgmt = model.TaggedVLANMgmt.ValueString()
			}
			if !model.VoiceNetworkID.IsNull() {
				po.VoiceNetworkID = model.VoiceNetworkID.ValueString()
			}

			// Boolean attributes
			po.Autoneg = model.Autoneg.ValueBool()
			po.EgressRateLimitKbpsEnabled = model.EgressRateLimitKbpsEnabled.ValueBool()
			po.FlowControlEnabled = model.FlowControlEnabled.ValueBool()
			po.FullDuplex = model.FullDuplex.ValueBool()
			po.Isolation = model.Isolation.ValueBool()
			po.LldpmedEnabled = model.LldpmedEnabled.ValueBool()
			po.LldpmedNotifyEnabled = model.LldpmedNotifyEnabled.ValueBool()
			po.PortKeepaliveEnabled = model.PortKeepaliveEnabled.ValueBool()
			po.PortSecurityEnabled = model.PortSecurityEnabled.ValueBool()
			po.StormctrlBroadcastastEnabled = model.StormctrlBroadcastEnabled.ValueBool()
			po.StormctrlMcastEnabled = model.StormctrlMcastEnabled.ValueBool()
			po.StormctrlUcastEnabled = model.StormctrlUcastEnabled.ValueBool()
			po.StpPortMode = model.StpPortMode.ValueBool()

			// Int64 attributes
			if !model.Dot1XIDleTimeout.IsNull() {
				po.Dot1XIDleTimeout = util.DurationUnitsPtr(model.Dot1XIDleTimeout, time.Second)
			}
			if !model.EgressRateLimitKbps.IsNull() {
				po.EgressRateLimitKbps = model.EgressRateLimitKbps.ValueInt64Pointer()
			}
			if !model.MirrorPortIDX.IsNull() {
				po.MirrorPortIDX = model.MirrorPortIDX.ValueInt64Pointer()
			}
			if !model.PriorityQueue1Level.IsNull() {
				po.PriorityQueue1Level = model.PriorityQueue1Level.ValueInt64Pointer()
			}
			if !model.PriorityQueue2Level.IsNull() {
				po.PriorityQueue2Level = model.PriorityQueue2Level.ValueInt64Pointer()
			}
			if !model.PriorityQueue3Level.IsNull() {
				po.PriorityQueue3Level = model.PriorityQueue3Level.ValueInt64Pointer()
			}
			if !model.PriorityQueue4Level.IsNull() {
				po.PriorityQueue4Level = model.PriorityQueue4Level.ValueInt64Pointer()
			}
			if !model.Speed.IsNull() {
				po.Speed = model.Speed.ValueInt64Pointer()
			}
			if !model.StormctrlBroadcastLevel.IsNull() {
				po.StormctrlBroadcastastLevel = model.StormctrlBroadcastLevel.ValueInt64Pointer()
			}
			if !model.StormctrlBroadcastRate.IsNull() {
				po.StormctrlBroadcastastRate = model.StormctrlBroadcastRate.ValueInt64Pointer()
			}
			if !model.StormctrlMcastLevel.IsNull() {
				po.StormctrlMcastLevel = model.StormctrlMcastLevel.ValueInt64Pointer()
			}
			if !model.StormctrlMcastRate.IsNull() {
				po.StormctrlMcastRate = model.StormctrlMcastRate.ValueInt64Pointer()
			}
			if !model.StormctrlUcastLevel.IsNull() {
				po.StormctrlUcastLevel = model.StormctrlUcastLevel.ValueInt64Pointer()
			}
			if !model.StormctrlUcastRate.IsNull() {
				po.StormctrlUcastRate = model.StormctrlUcastRate.ValueInt64Pointer()
			}

			// List attributes
			if !model.AggregateMembers.IsNull() {
				var aggrMembers []int64
				diags.Append(model.AggregateMembers.ElementsAs(ctx, &aggrMembers, true)...)
				if diags.HasError() {
					return nil, diags
				}
				po.AggregateMembers = aggrMembers
			}

			if !model.ExcludedNetworkIDs.IsNull() {
				var excludedIDs []string
				diags.Append(model.ExcludedNetworkIDs.ElementsAs(ctx, &excludedIDs, true)...)
				if diags.HasError() {
					return nil, diags
				}
				po.ExcludedNetworkIDs = excludedIDs
			}

			if !model.MulticastRouterNetworkIDs.IsNull() {
				var multicastIDs []string
				diags.Append(
					model.MulticastRouterNetworkIDs.ElementsAs(ctx, &multicastIDs, true)...)
				if diags.HasError() {
					return nil, diags
				}
				po.MulticastRouterNetworkIDs = multicastIDs
			}

			if !model.PortSecurityMACAddress.IsNull() {
				var macAddresses []string
				diags.Append(model.PortSecurityMACAddress.ElementsAs(ctx, &macAddresses, true)...)
				if diags.HasError() {
					return nil, diags
				}
				po.PortSecurityMACAddress = macAddresses
			}

			overrideMap[idx] = po
		} else {
			diags.Append(
				diag.NewErrorDiagnostic(
					"Invalid port override model",
					"Error casting `portOverrideModel` to `types.Object`",
				),
			)
		}
	}

	pos := make([]unifi.DevicePortOverrides, 0, len(overrideMap))
	for _, po := range overrideMap {
		pos = append(pos, po)
	}

	return pos, diags
}

func (r *deviceResource) waitForDeviceState(
	ctx context.Context,
	site, mac string,
	targetState unifi.DeviceState,
	pendingStates []unifi.DeviceState,
	timeout time.Duration,
) (*unifi.Device, error) {
	// Always consider unknown to be a pending state.
	pendingStates = append(pendingStates, unifi.DeviceStateUnknown)

	var pending []string
	for _, state := range pendingStates {
		pending = append(pending, state.String())
	}

	wait := retry.StateChangeConf{
		Pending: pending,
		Target:  []string{targetState.String()},
		Refresh: func() (any, string, error) {
			device, err := r.client.GetDeviceByMAC(ctx, site, mac)

			if _, ok := err.(*unifi.NotFoundError); ok {
				err = nil
			}

			// When a device is forgotten, it will disappear from the UI for a few seconds before reappearing.
			// During this time, `device.GetDeviceByMAC` will return a 400.
			if err != nil && strings.Contains(err.Error(), "api.err.UnknownDevice") {
				err = nil
			}

			var state string
			if device != nil {
				state = device.State.String()
			}

			if device == nil {
				return nil, state, err
			}

			return device, state, err
		},
		Timeout:        timeout,
		NotFoundChecks: 30,
	}

	outputRaw, err := wait.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*unifi.Device); ok {
		return output, err
	}

	return nil, err
}

// cleanMAC normalizes MAC address format.
func cleanMAC(mac string) string {
	mac = strings.ReplaceAll(mac, "-", ":")
	mac = strings.ToLower(mac)
	return mac
}

// portOverrideAttrTypes returns the attribute types for port override objects.
func portOverrideAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"index":                            types.Int64Type,
		"name":                             types.StringType,
		"port_profile_id":                  types.StringType,
		"op_mode":                          types.StringType,
		"poe_mode":                         types.StringType,
		"aggregate_members":                types.ListType{ElemType: types.Int64Type},
		"autoneg":                          types.BoolType,
		"dot1x_ctrl":                       types.StringType,
		"dot1x_idle_timeout":               timetypes.GoDurationType{},
		"egress_rate_limit_kbps":           types.Int64Type,
		"egress_rate_limit_kbps_enabled":   types.BoolType,
		"excluded_networkconf_ids":         types.ListType{ElemType: types.StringType},
		"fec_mode":                         types.StringType,
		"flow_control_enabled":             types.BoolType,
		"forward":                          types.StringType,
		"full_duplex":                      types.BoolType,
		"isolation":                        types.BoolType,
		"lldpmed_enabled":                  types.BoolType,
		"lldpmed_notify_enabled":           types.BoolType,
		"mirror_port_idx":                  types.Int64Type,
		"multicast_router_networkconf_ids": types.ListType{ElemType: types.StringType},
		"native_networkconf_id":            types.StringType,
		"port_keepalive_enabled":           types.BoolType,
		"port_security_enabled":            types.BoolType,
		"port_security_mac_address":        types.ListType{ElemType: types.StringType},
		"priority_queue1_level":            types.Int64Type,
		"priority_queue2_level":            types.Int64Type,
		"priority_queue3_level":            types.Int64Type,
		"priority_queue4_level":            types.Int64Type,
		"setting_preference":               types.StringType,
		"speed":                            types.Int64Type,
		"stormctrl_bcast_enabled":          types.BoolType,
		"stormctrl_bcast_level":            types.Int64Type,
		"stormctrl_bcast_rate":             types.Int64Type,
		"stormctrl_mcast_enabled":          types.BoolType,
		"stormctrl_mcast_level":            types.Int64Type,
		"stormctrl_mcast_rate":             types.Int64Type,
		"stormctrl_type":                   types.StringType,
		"stormctrl_ucast_enabled":          types.BoolType,
		"stormctrl_ucast_level":            types.Int64Type,
		"stormctrl_ucast_rate":             types.Int64Type,
		"stp_port_mode":                    types.BoolType,
		"tagged_networkconf_ids":           types.ListType{ElemType: types.StringType},
		"tagged_vlan_mgmt":                 types.StringType,
		"voice_networkconf_id":             types.StringType,
	}
}

// configNetworkAttrTypes returns the attribute types for config network objects.
func configNetworkAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":            types.StringType,
		"ip":              types.StringType,
		"netmask":         types.StringType,
		"gateway":         types.StringType,
		"dns1":            types.StringType,
		"dns2":            types.StringType,
		"dnssuffix":       types.StringType,
		"bonding_enabled": types.BoolType,
	}
}

// radioTableAttrTypes returns the attribute types for radio table objects.
func radioTableAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"radio":                    types.StringType,
		"channel":                  types.StringType,
		"ht":                       types.Int64Type,
		"tx_power":                 types.StringType,
		"tx_power_mode":            types.StringType,
		"min_rssi_enabled":         types.BoolType,
		"min_rssi":                 types.Int64Type,
		"antenna_gain":             types.Int64Type,
		"antenna_id":               types.Int64Type,
		"assisted_roaming_enabled": types.BoolType,
		"assisted_roaming_rssi":    types.Int64Type,
		"dfs":                      types.BoolType,
		"hard_noise_floor_enabled": types.BoolType,
		"loadbalance_enabled":      types.BoolType,
		"maxsta":                   types.Int64Type,
		"name":                     types.StringType,
		"sens_level":               types.Int64Type,
		"sens_level_enabled":       types.BoolType,
		"vwire_enabled":            types.BoolType,
	}
}

// outletOverrideAttrTypes returns the attribute types for outlet override objects.
func outletOverrideAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"index":         types.Int64Type,
		"name":          types.StringType,
		"relay_state":   types.BoolType,
		"cycle_enabled": types.BoolType,
	}
}

// stringOrNull returns a types.String with the value or null if empty.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// int64OrNull returns a types.Int64 with the value or null if zero.
func int64OrNull(i int64) types.Int64 { //nolint:unused
	if i == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(i)
}

// configNetworkToFramework converts API ConfigNetwork to Framework types.
func (r *deviceResource) configNetworkToFramework(
	ctx context.Context,
	cn *unifi.DeviceConfigNetwork,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if cn == nil || (cn.Type == "" && cn.IP == "" && cn.Gateway == "" && cn.Netmask == "") {
		return types.ObjectNull(configNetworkAttrTypes()), diags
	}

	model := configNetworkModel{
		Type:           stringOrNull(cn.Type),
		IP:             stringOrNull(cn.IP),
		Netmask:        stringOrNull(cn.Netmask),
		Gateway:        stringOrNull(cn.Gateway),
		DNS1:           stringOrNull(cn.DNS1),
		DNS2:           stringOrNull(cn.DNS2),
		DNSsuffix:      stringOrNull(cn.DNSsuffix),
		BondingEnabled: types.BoolValue(cn.BondingEnabled),
	}

	objVal, objDiags := types.ObjectValueFrom(ctx, configNetworkAttrTypes(), model)
	diags.Append(objDiags...)
	return objVal, diags
}

// radioTableToFramework converts API RadioTable to Framework types.
func (r *deviceResource) radioTableToFramework(
	ctx context.Context,
	radios []unifi.DeviceRadioTable,
) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrType := types.ObjectType{AttrTypes: radioTableAttrTypes()}

	if len(radios) == 0 {
		return types.ListNull(attrType), diags
	}

	elements := make([]attr.Value, 0, len(radios))
	for _, radio := range radios {
		model := radioTableModel{
			Radio:                  stringOrNull(radio.Radio),
			Channel:                stringOrNull(radio.Channel),
			Ht:                     types.Int64PointerValue(radio.Ht),
			TxPower:                stringOrNull(radio.TxPower),
			TxPowerMode:            stringOrNull(radio.TxPowerMode),
			MinRssiEnabled:         types.BoolValue(radio.MinRssiEnabled),
			MinRssi:                types.Int64PointerValue(radio.MinRssi),
			AntennaGain:            types.Int64PointerValue(radio.AntennaGain),
			AntennaID:              types.Int64PointerValue(radio.AntennaID),
			AssistedRoamingEnabled: types.BoolValue(radio.AssistedRoamingEnabled),
			AssistedRoamingRssi:    types.Int64PointerValue(radio.AssistedRoamingRssi),
			Dfs:                    types.BoolValue(radio.Dfs),
			HardNoiseFloorEnabled:  types.BoolValue(radio.HardNoiseFloorEnabled),
			LoadbalanceEnabled:     types.BoolValue(radio.LoadbalanceEnabled),
			Maxsta:                 types.Int64PointerValue(radio.Maxsta),
			Name:                   stringOrNull(radio.Name),
			SensLevel:              types.Int64PointerValue(radio.SensLevel),
			SensLevelEnabled:       types.BoolValue(radio.SensLevelEnabled),
			VwireEnabled:           types.BoolValue(radio.VwireEnabled),
		}

		objVal, objDiags := types.ObjectValueFrom(ctx, radioTableAttrTypes(), model)
		diags.Append(objDiags...)
		if diags.HasError() {
			continue
		}
		elements = append(elements, objVal)
	}

	if diags.HasError() {
		return types.ListNull(attrType), diags
	}

	listVal, listDiags := types.ListValue(attrType, elements)
	diags.Append(listDiags...)
	return listVal, diags
}

// outletOverridesToFramework converts API OutletOverrides to Framework types.
func (r *deviceResource) outletOverridesToFramework(
	ctx context.Context,
	outlets []unifi.DeviceOutletOverrides,
) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrType := types.ObjectType{AttrTypes: outletOverrideAttrTypes()}

	if len(outlets) == 0 {
		return types.ListNull(attrType), diags
	}

	elements := make([]attr.Value, 0, len(outlets))
	for _, outlet := range outlets {
		model := outletOverrideModel{
			Index:        types.Int64PointerValue(outlet.Index),
			Name:         stringOrNull(outlet.Name),
			RelayState:   types.BoolValue(outlet.RelayState),
			CycleEnabled: types.BoolValue(outlet.CycleEnabled),
		}

		objVal, objDiags := types.ObjectValueFrom(ctx, outletOverrideAttrTypes(), model)
		diags.Append(objDiags...)
		if diags.HasError() {
			continue
		}
		elements = append(elements, objVal)
	}

	if diags.HasError() {
		return types.ListNull(attrType), diags
	}

	listVal, listDiags := types.ListValue(attrType, elements)
	diags.Append(listDiags...)
	return listVal, diags
}

// frameworkToConfigNetwork converts Framework types to API ConfigNetwork.
func (r *deviceResource) frameworkToConfigNetwork(
	ctx context.Context,
	configNetworkObj types.Object,
) (*unifi.DeviceConfigNetwork, diag.Diagnostics) {
	var diags diag.Diagnostics

	if configNetworkObj.IsNull() || configNetworkObj.IsUnknown() {
		return nil, diags
	}

	var model configNetworkModel
	diags.Append(configNetworkObj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	cn := &unifi.DeviceConfigNetwork{
		Type:           model.Type.ValueString(),
		IP:             model.IP.ValueString(),
		Netmask:        model.Netmask.ValueString(),
		Gateway:        model.Gateway.ValueString(),
		DNS1:           model.DNS1.ValueString(),
		DNS2:           model.DNS2.ValueString(),
		DNSsuffix:      model.DNSsuffix.ValueString(),
		BondingEnabled: model.BondingEnabled.ValueBool(),
	}

	return cn, diags
}

// frameworkToRadioTable converts Framework types to API RadioTable.
// sanitizeRadioForUpdate drops numeric radio fields whose zero/out-of-range values
// the controller rejects with api.err.InvalidPayload (400) on UDM/Dream Machine
// gateways (#150/#177, same class as #303). Leaving the pointer nil lets `omitempty`
// drop the JSON key entirely. Valid ranges (from the go-unifi schema, low..high):
// min_rssi -90..-67 (only when enabled), maxsta 1..200, sens_level -90..-50 (only
// when enabled), assisted_roaming_rssi -80..-60 (only when enabled).
//
// When a field is ENABLED but its declared value is out of range, the value is
// still dropped (the controller would reject the whole request otherwise) but a
// warning diagnostic is returned so the silent no-op is visible to the user
// instead of their configured value just quietly failing to apply (review
// feedback on PR #378: "user-provided configuration can be silently ignored").
// Disabled fields, or fields simply left unset, drop silently as before — that's
// the normal/expected case, not something worth warning about.
func sanitizeRadioForUpdate(radioName string, radio *unifi.DeviceRadioTable) diag.Diagnostics {
	var diags diag.Diagnostics
	// Ranges are the go-unifi schema regexes: min_rssi [-90,-67], maxsta [1,200],
	// sens_level [-90,-50], assisted_roaming_rssi [-80,-60].
	inRange := func(v *int64, lo, hi int64) bool { return v != nil && *v >= lo && *v <= hi }
	warnDropped := func(field string, v int64, lo, hi int64) {
		diags.AddWarning(
			"Radio field out of range — not applied",
			fmt.Sprintf(
				"radio %q: %s=%d is outside the controller's valid range [%d,%d] and was dropped "+
					"from the update (the controller rejects out-of-range values with "+
					"api.err.InvalidPayload). The declared value will not take effect — adjust it "+
					"to be within range.",
				radioName, field, v, lo, hi,
			),
		)
	}

	if radio.MinRssiEnabled && radio.MinRssi != nil && !inRange(radio.MinRssi, -90, -67) {
		warnDropped("min_rssi", *radio.MinRssi, -90, -67)
	}
	if !radio.MinRssiEnabled || !inRange(radio.MinRssi, -90, -67) {
		radio.MinRssi = nil
	}
	if radio.Maxsta != nil && !inRange(radio.Maxsta, 1, 200) {
		warnDropped("maxsta", *radio.Maxsta, 1, 200)
	}
	if !inRange(radio.Maxsta, 1, 200) {
		radio.Maxsta = nil
	}
	if radio.SensLevelEnabled && radio.SensLevel != nil && !inRange(radio.SensLevel, -90, -50) {
		warnDropped("sens_level", *radio.SensLevel, -90, -50)
	}
	if !radio.SensLevelEnabled || !inRange(radio.SensLevel, -90, -50) {
		radio.SensLevel = nil
	}
	if radio.AssistedRoamingEnabled && radio.AssistedRoamingRssi != nil && !inRange(radio.AssistedRoamingRssi, -80, -60) {
		warnDropped("assisted_roaming_rssi", *radio.AssistedRoamingRssi, -80, -60)
	}
	if !radio.AssistedRoamingEnabled || !inRange(radio.AssistedRoamingRssi, -80, -60) {
		radio.AssistedRoamingRssi = nil
	}

	return diags
}

func (r *deviceResource) frameworkToRadioTable(
	ctx context.Context,
	radioList types.List,
) ([]unifi.DeviceRadioTable, diag.Diagnostics) {
	var diags diag.Diagnostics

	if radioList.IsNull() || radioList.IsUnknown() {
		return nil, diags
	}

	elements := radioList.Elements()
	radios := make([]unifi.DeviceRadioTable, 0, len(elements))

	for _, elem := range elements {
		obj, ok := elem.(types.Object)
		if !ok {
			continue
		}

		var model radioTableModel
		diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			continue
		}

		radio := unifi.DeviceRadioTable{
			Radio:                  model.Radio.ValueString(),
			Channel:                model.Channel.ValueString(),
			Ht:                     model.Ht.ValueInt64Pointer(),
			TxPower:                model.TxPower.ValueString(),
			TxPowerMode:            model.TxPowerMode.ValueString(),
			MinRssiEnabled:         model.MinRssiEnabled.ValueBool(),
			MinRssi:                model.MinRssi.ValueInt64Pointer(),
			AntennaGain:            model.AntennaGain.ValueInt64Pointer(),
			AntennaID:              model.AntennaID.ValueInt64Pointer(),
			AssistedRoamingEnabled: model.AssistedRoamingEnabled.ValueBool(),
			AssistedRoamingRssi:    model.AssistedRoamingRssi.ValueInt64Pointer(),
			Dfs:                    model.Dfs.ValueBool(),
			HardNoiseFloorEnabled:  model.HardNoiseFloorEnabled.ValueBool(),
			LoadbalanceEnabled:     model.LoadbalanceEnabled.ValueBool(),
			Maxsta:                 model.Maxsta.ValueInt64Pointer(),
			Name:                   model.Name.ValueString(),
			SensLevel:              model.SensLevel.ValueInt64Pointer(),
			SensLevelEnabled:       model.SensLevelEnabled.ValueBool(),
			VwireEnabled:           model.VwireEnabled.ValueBool(),
		}

		diags.Append(sanitizeRadioForUpdate(radio.Radio, &radio)...)

		radios = append(radios, radio)
	}

	return radios, diags
}

// frameworkToOutletOverrides converts Framework types to API OutletOverrides.
func (r *deviceResource) frameworkToOutletOverrides(
	ctx context.Context,
	outletList types.List,
) ([]unifi.DeviceOutletOverrides, diag.Diagnostics) {
	var diags diag.Diagnostics

	if outletList.IsNull() || outletList.IsUnknown() {
		return nil, diags
	}

	elements := outletList.Elements()
	outlets := make([]unifi.DeviceOutletOverrides, 0, len(elements))

	for _, elem := range elements {
		obj, ok := elem.(types.Object)
		if !ok {
			continue
		}

		var model outletOverrideModel
		diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			continue
		}

		outlet := unifi.DeviceOutletOverrides{
			Index:        model.Index.ValueInt64Pointer(),
			Name:         model.Name.ValueString(),
			RelayState:   model.RelayState.ValueBool(),
			CycleEnabled: model.CycleEnabled.ValueBool(),
		}

		outlets = append(outlets, outlet)
	}

	return outlets, diags
}

// ---------------------------------------------------------------------------
// List resource
// ---------------------------------------------------------------------------

// deviceListToModel populates the model's schema fields directly from the API
// struct for listing. It reuses the existing setResourceData flatten (which
// relies on nil-safe configNetwork/radioTable/outletOverrides helpers) for a
// faithful representation, then nulls out the write-only plan-only flags and
// the large port_override set — listing does not need full per-port detail.
func (r *deviceResource) deviceListToModel(
	ctx context.Context,
	api *unifi.Device,
	model *deviceResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	r.setResourceData(ctx, &diags, api, model, site)

	// port_override is a Set where every field contributes to identity; the API
	// returns all ports with all fields. A faithful-but-sparse listing sets it
	// null rather than reproducing the full controller-managed detail.
	model.PortOverride = types.SetNull(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
	)

	// Write-only plan flags are never returned by the API.
	model.AllowAdoption = types.BoolNull()
	model.ForgetOnDestroy = types.BoolNull()

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *deviceResource) ListResourceConfigSchema(
	_ context.Context,
	_ list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List devices in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list devices from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`, `mac`, `model`, `type`.",
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
func (r *deviceResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config deviceListConfigModel

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
	var filters []deviceListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	devices, err := r.client.ListDevice(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing Devices", "Could not list devices: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, device := range devices {
			// Apply name filter.
			if val, ok := postFilters["name"]; ok {
				if device.Name != val {
					continue
				}
			}

			// Apply mac filter.
			if val, ok := postFilters["mac"]; ok {
				if device.MAC != val {
					continue
				}
			}

			// Apply model filter.
			if val, ok := postFilters["model"]; ok {
				if device.Model != val {
					continue
				}
			}

			// Apply type filter.
			if val, ok := postFilters["type"]; ok {
				if device.Type != val {
					continue
				}
			}

			result := req.NewListResult(ctx)

			// Display name: prefer name, fall back to MAC.
			if device.Name != "" {
				result.DisplayName = device.Name
			} else {
				result.DisplayName = device.MAC
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.SetAttribute(
					ctx,
					path.Root("id"),
					types.StringValue(device.ID),
				)...,
			)

			// Convert to model.
			d := device
			var model deviceResourceModel
			result.Diagnostics.Append(r.deviceListToModel(ctx, &d, &model, site)...)
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
