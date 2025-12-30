package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

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
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util/retry"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &deviceResource{}
	_ resource.ResourceWithImportState = &deviceResource{}
)

func NewDeviceFrameworkResource() resource.Resource {
	return &deviceResource{}
}

// deviceResource defines the resource implementation.
type deviceResource struct {
	client *Client
}

// deviceResourceModel describes the resource data model.
type deviceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Site            types.String `tfsdk:"site"`
	MAC             types.String `tfsdk:"mac"`
	Name            types.String `tfsdk:"name"`
	Disabled        types.Bool   `tfsdk:"disabled"`
	PortOverride    types.Set    `tfsdk:"port_override"`
	AllowAdoption   types.Bool   `tfsdk:"allow_adoption"`
	ForgetOnDestroy types.Bool   `tfsdk:"forget_on_destroy"`

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
	StpPriority       types.String `tfsdk:"stp_priority"`
	Locked            types.Bool   `tfsdk:"locked"`

	// PoE settings
	PoeMode types.String `tfsdk:"poe_mode"`

	// VLAN
	SwitchVLANEnabled types.Bool `tfsdk:"switch_vlan_enabled"`

	// Radio settings
	RadioTable types.List `tfsdk:"radio_table"`

	// Advanced features
	OutdoorModeOverride types.String `tfsdk:"outdoor_mode_override"`
	Volume              types.Int64  `tfsdk:"volume"`
	XBaresipPassword    types.String `tfsdk:"x_baresip_password"`

	// LCD/LCM settings
	LcmBrightness          types.Int64  `tfsdk:"lcm_brightness"`
	LcmBrightnessOverride  types.Bool   `tfsdk:"lcm_brightness_override"`
	LcmIDleTimeout         types.Int64  `tfsdk:"lcm_idle_timeout"`
	LcmIDleTimeoutOverride types.Bool   `tfsdk:"lcm_idle_timeout_override"`
	LcmNightModeBegins     types.String `tfsdk:"lcm_night_mode_begins"`
	LcmNightModeEnds       types.String `tfsdk:"lcm_night_mode_ends"`

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
}

// portOverrideModel describes the port override data model.
type portOverrideModel struct {
	Number                     types.Int64  `tfsdk:"number"`
	Name                       types.String `tfsdk:"name"`
	PortProfileID              types.String `tfsdk:"port_profile_id"`
	OpMode                     types.String `tfsdk:"op_mode"`
	PoeMode                    types.String `tfsdk:"poe_mode"`
	AggregateMembers           types.List   `tfsdk:"aggregate_members"`
	Autoneg                    types.Bool   `tfsdk:"autoneg"`
	Dot1XCtrl                  types.String `tfsdk:"dot1x_ctrl"`
	Dot1XIDleTimeout           types.Int64  `tfsdk:"dot1x_idle_timeout"`
	EgressRateLimitKbps        types.Int64  `tfsdk:"egress_rate_limit_kbps"`
	EgressRateLimitKbpsEnabled types.Bool   `tfsdk:"egress_rate_limit_kbps_enabled"`
	ExcludedNetworkIDs         types.List   `tfsdk:"excluded_networkconf_ids"`
	FecMode                    types.String `tfsdk:"fec_mode"`
	FlowControlEnabled         types.Bool   `tfsdk:"flow_control_enabled"`
	Forward                    types.String `tfsdk:"forward"`
	FullDuplex                 types.Bool   `tfsdk:"full_duplex"`
	Isolation                  types.Bool   `tfsdk:"isolation"`
	LldpmedEnabled             types.Bool   `tfsdk:"lldpmed_enabled"`
	LldpmedNotifyEnabled       types.Bool   `tfsdk:"lldpmed_notify_enabled"`
	MirrorPortIDX              types.Int64  `tfsdk:"mirror_port_idx"`
	MulticastRouterNetworkIDs  types.List   `tfsdk:"multicast_router_networkconf_ids"`
	NativeNetworkID            types.String `tfsdk:"native_networkconf_id"`
	PortKeepaliveEnabled       types.Bool   `tfsdk:"port_keepalive_enabled"`
	PortSecurityEnabled        types.Bool   `tfsdk:"port_security_enabled"`
	PortSecurityMACAddress     types.List   `tfsdk:"port_security_mac_address"`
	PriorityQueue1Level        types.Int64  `tfsdk:"priority_queue1_level"`
	PriorityQueue2Level        types.Int64  `tfsdk:"priority_queue2_level"`
	PriorityQueue3Level        types.Int64  `tfsdk:"priority_queue3_level"`
	PriorityQueue4Level        types.Int64  `tfsdk:"priority_queue4_level"`
	SettingPreference          types.String `tfsdk:"setting_preference"`
	Speed                      types.Int64  `tfsdk:"speed"`
	StormctrlBroadcastEnabled  types.Bool   `tfsdk:"stormctrl_bcast_enabled"`
	StormctrlBroadcastLevel    types.Int64  `tfsdk:"stormctrl_bcast_level"`
	StormctrlBroadcastRate     types.Int64  `tfsdk:"stormctrl_bcast_rate"`
	StormctrlMcastEnabled      types.Bool   `tfsdk:"stormctrl_mcast_enabled"`
	StormctrlMcastLevel        types.Int64  `tfsdk:"stormctrl_mcast_level"`
	StormctrlMcastRate         types.Int64  `tfsdk:"stormctrl_mcast_rate"`
	StormctrlType              types.String `tfsdk:"stormctrl_type"`
	StormctrlUcastEnabled      types.Bool   `tfsdk:"stormctrl_ucast_enabled"`
	StormctrlUcastLevel        types.Int64  `tfsdk:"stormctrl_ucast_level"`
	StormctrlUcastRate         types.Int64  `tfsdk:"stormctrl_ucast_rate"`
	StpPortMode                types.Bool   `tfsdk:"stp_port_mode"`
	TaggedVLANMgmt             types.String `tfsdk:"tagged_vlan_mgmt"`
	VoiceNetworkID             types.String `tfsdk:"voice_networkconf_id"`
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

var macAddressRegexp = regexp.MustCompile(`^([a-fA-F0-9]{2}[:-]){5}[a-fA-F0-9]{2}$`)

func (r *deviceResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *deviceResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
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
				Validators: []validator.String{
					validators.MACAddressValidator(),
				},
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
			},
			"forget_on_destroy": schema.BoolAttribute{
				Description: "Specifies whether this resource should tell the controller to forget the device on destroy.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
			},
			"led_override_color": schema.StringAttribute{
				Description: "LED color override (hex color code).",
				Optional:    true,
				Computed:    true,
			},
			"led_override_color_brightness": schema.Int64Attribute{
				Description: "LED brightness (0-100).",
				Optional:    true,
				Computed:    true,
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
			"stp_priority": schema.StringAttribute{
				Description: "STP priority.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"0", "4096", "8192", "12288", "16384", "20480",
						"24576", "28672", "32768", "36864", "40960",
						"45056", "49152", "53248", "57344", "61440",
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
			"lcm_idle_timeout": schema.Int64Attribute{
				Description: "LCM idle timeout in seconds (10-3600).",
				Optional:    true,
				Computed:    true,
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
				Description: "Management network ID.",
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
		},

		Blocks: map[string]schema.Block{
			"port_override": schema.SetNestedBlock{
				Description: "Settings overrides for specific switch ports.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"number": schema.Int64Attribute{
							Description: "Switch port number.",
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
							Description: "Operating mode of the port, valid values are `switch`, `mirror`, and `aggregate`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("switch"),
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
							Description: "Number of ports in the aggregate.",
							Optional:    true,
							ElementType: types.Int64Type,
						},
					},
				},
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
	device, err := r.client.GetDeviceByMAC(ctx, site, mac)
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

		device, err = r.waitForDeviceState(
			ctx,
			site, mac,
			unifi.DeviceStateConnected,
			[]unifi.DeviceState{
				unifi.DeviceStateAdopting,
				unifi.DeviceStatePending,
				unifi.DeviceStateProvisioning,
				unifi.DeviceStateUpgrading,
			},
			2*time.Minute,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Waiting for Device Adoption",
				fmt.Sprintf("Could not wait for device adoption: %s", err),
			)
			return
		}
	}

	plan.ID = types.StringValue(device.ID)
	plan.Site = types.StringValue(site)

	// Apply the update operation
	diags = r.updateDevice(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state
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

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	device, err := r.client.GetDevice(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, &resp.Diagnostics, device, &state, site)
	if resp.Diagnostics.HasError() {
		return
	}

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
	currentDevice, err := r.client.GetDevice(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Device for Update",
			fmt.Sprintf("Could not read device %s for update: %s", id, err),
		)
		return
	}

	// Apply current API values to state
	r.setResourceData(ctx, &resp.Diagnostics, currentDevice, &state, site)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply plan changes to the state
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Disabled.IsNull() && !plan.Disabled.IsUnknown() {
		state.Disabled = plan.Disabled
	}
	if !plan.PortOverride.IsNull() && !plan.PortOverride.IsUnknown() {
		state.PortOverride = plan.PortOverride
	}
	if !plan.ConfigNetwork.IsNull() && !plan.ConfigNetwork.IsUnknown() {
		state.ConfigNetwork = plan.ConfigNetwork
	}
	if !plan.LedOverride.IsNull() && !plan.LedOverride.IsUnknown() {
		state.LedOverride = plan.LedOverride
	}
	if !plan.LedOverrideColor.IsNull() && !plan.LedOverrideColor.IsUnknown() {
		state.LedOverrideColor = plan.LedOverrideColor
	}
	if !plan.LedOverrideColorBrightness.IsNull() && !plan.LedOverrideColorBrightness.IsUnknown() {
		state.LedOverrideColorBrightness = plan.LedOverrideColorBrightness
	}
	if !plan.BandsteeringMode.IsNull() && !plan.BandsteeringMode.IsUnknown() {
		state.BandsteeringMode = plan.BandsteeringMode
	}
	if !plan.FlowctrlEnabled.IsNull() && !plan.FlowctrlEnabled.IsUnknown() {
		state.FlowctrlEnabled = plan.FlowctrlEnabled
	}
	if !plan.JumboframeEnabled.IsNull() && !plan.JumboframeEnabled.IsUnknown() {
		state.JumboframeEnabled = plan.JumboframeEnabled
	}
	if !plan.StpVersion.IsNull() && !plan.StpVersion.IsUnknown() {
		state.StpVersion = plan.StpVersion
	}
	if !plan.StpPriority.IsNull() && !plan.StpPriority.IsUnknown() {
		state.StpPriority = plan.StpPriority
	}
	if !plan.Locked.IsNull() && !plan.Locked.IsUnknown() {
		state.Locked = plan.Locked
	}
	if !plan.PoeMode.IsNull() && !plan.PoeMode.IsUnknown() {
		state.PoeMode = plan.PoeMode
	}
	if !plan.SwitchVLANEnabled.IsNull() && !plan.SwitchVLANEnabled.IsUnknown() {
		state.SwitchVLANEnabled = plan.SwitchVLANEnabled
	}
	if !plan.OutdoorModeOverride.IsNull() && !plan.OutdoorModeOverride.IsUnknown() {
		state.OutdoorModeOverride = plan.OutdoorModeOverride
	}
	if !plan.Volume.IsNull() && !plan.Volume.IsUnknown() {
		state.Volume = plan.Volume
	}
	if !plan.XBaresipPassword.IsNull() && !plan.XBaresipPassword.IsUnknown() {
		state.XBaresipPassword = plan.XBaresipPassword
	}
	if !plan.LcmBrightness.IsNull() && !plan.LcmBrightness.IsUnknown() {
		state.LcmBrightness = plan.LcmBrightness
	}
	if !plan.LcmBrightnessOverride.IsNull() && !plan.LcmBrightnessOverride.IsUnknown() {
		state.LcmBrightnessOverride = plan.LcmBrightnessOverride
	}
	if !plan.LcmIDleTimeout.IsNull() && !plan.LcmIDleTimeout.IsUnknown() {
		state.LcmIDleTimeout = plan.LcmIDleTimeout
	}
	if !plan.LcmIDleTimeoutOverride.IsNull() && !plan.LcmIDleTimeoutOverride.IsUnknown() {
		state.LcmIDleTimeoutOverride = plan.LcmIDleTimeoutOverride
	}
	if !plan.LcmNightModeBegins.IsNull() && !plan.LcmNightModeBegins.IsUnknown() {
		state.LcmNightModeBegins = plan.LcmNightModeBegins
	}
	if !plan.LcmNightModeEnds.IsNull() && !plan.LcmNightModeEnds.IsUnknown() {
		state.LcmNightModeEnds = plan.LcmNightModeEnds
	}
	if !plan.OutletEnabled.IsNull() && !plan.OutletEnabled.IsUnknown() {
		state.OutletEnabled = plan.OutletEnabled
	}
	if !plan.OutletOverrides.IsNull() && !plan.OutletOverrides.IsUnknown() {
		state.OutletOverrides = plan.OutletOverrides
	}
	if !plan.MgmtNetworkID.IsNull() && !plan.MgmtNetworkID.IsUnknown() {
		state.MgmtNetworkID = plan.MgmtNetworkID
	}
	if !plan.RadioTable.IsNull() && !plan.RadioTable.IsUnknown() {
		state.RadioTable = plan.RadioTable
	}

	// Update the resource
	diags = r.updateDevice(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
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
		1*time.Minute,
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
	resource.ImportStatePassthroughID(
		ctx,
		path.Root("id"),
		req,
		resp,
	)
}

// Helper methods

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
	deviceReq.SiteID = site

	device, err := r.client.UpdateDevice(ctx, site, deviceReq)
	if err != nil {
		diags.AddError(
			"Error Updating Device",
			fmt.Sprintf("Could not update device: %s", err),
		)
		return diags
	}

	// Wait for device to be in connected state
	device, err = r.waitForDeviceState(
		ctx,
		site, device.MAC,
		unifi.DeviceStateConnected,
		[]unifi.DeviceState{unifi.DeviceStateAdopting, unifi.DeviceStateProvisioning},
		1*time.Minute,
	)
	if err != nil {
		diags.AddError(
			"Error Waiting for Device Update",
			fmt.Sprintf("Could not wait for device update: %s", err),
		)
		return diags
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
	model.Site = types.StringValue(site)

	if device.MAC == "" {
		model.MAC = types.StringNull()
	} else {
		model.MAC = types.StringValue(device.MAC)
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

	model.State = types.Int64Value(int64(device.State))

	// LED settings
	if device.LedOverride == "" {
		model.LedOverride = types.StringNull()
	} else {
		model.LedOverride = types.StringValue(device.LedOverride)
	}

	if device.LedOverrideColor == "" {
		model.LedOverrideColor = types.StringNull()
	} else {
		model.LedOverrideColor = types.StringValue(device.LedOverrideColor)
	}

	if device.LedOverrideColorBrightness == 0 {
		model.LedOverrideColorBrightness = types.Int64Null()
	} else {
		model.LedOverrideColorBrightness = types.Int64Value(int64(device.LedOverrideColorBrightness))
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

	if device.StpPriority == "" {
		model.StpPriority = types.StringNull()
	} else {
		model.StpPriority = types.StringValue(device.StpPriority)
	}

	model.Locked = types.BoolValue(device.Locked)

	// PoE settings
	if device.PoeMode == "" {
		model.PoeMode = types.StringNull()
	} else {
		model.PoeMode = types.StringValue(device.PoeMode)
	}

	// VLAN
	model.SwitchVLANEnabled = types.BoolValue(device.SwitchVLANEnabled)

	// Advanced features
	if device.OutdoorModeOverride == "" {
		model.OutdoorModeOverride = types.StringNull()
	} else {
		model.OutdoorModeOverride = types.StringValue(device.OutdoorModeOverride)
	}

	if device.Volume == 0 {
		model.Volume = types.Int64Null()
	} else {
		model.Volume = types.Int64Value(int64(device.Volume))
	}

	if device.XBaresipPassword == "" {
		model.XBaresipPassword = types.StringNull()
	} else {
		model.XBaresipPassword = types.StringValue(device.XBaresipPassword)
	}

	// LCD/LCM settings
	if device.LcmBrightness == 0 {
		model.LcmBrightness = types.Int64Null()
	} else {
		model.LcmBrightness = types.Int64Value(int64(device.LcmBrightness))
	}

	model.LcmBrightnessOverride = types.BoolValue(device.LcmBrightnessOverride)

	if device.LcmIDleTimeout == 0 {
		model.LcmIDleTimeout = types.Int64Null()
	} else {
		model.LcmIDleTimeout = types.Int64Value(int64(device.LcmIDleTimeout))
	}

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
	configNetwork, convDiags := r.configNetworkToFramework(ctx, &device.ConfigNetwork)
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
		MAC:      model.MAC.ValueString(),
		Name:     model.Name.ValueString(),
		Disabled: model.Disabled.ValueBool(),
	}

	// LED settings
	if !model.LedOverride.IsNull() {
		device.LedOverride = model.LedOverride.ValueString()
	}
	if !model.LedOverrideColor.IsNull() {
		device.LedOverrideColor = model.LedOverrideColor.ValueString()
	}
	if !model.LedOverrideColorBrightness.IsNull() {
		device.LedOverrideColorBrightness = int(model.LedOverrideColorBrightness.ValueInt64())
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
	if !model.StpPriority.IsNull() {
		device.StpPriority = model.StpPriority.ValueString()
	}
	device.Locked = model.Locked.ValueBool()

	// PoE settings
	if !model.PoeMode.IsNull() {
		device.PoeMode = model.PoeMode.ValueString()
	}

	// VLAN
	device.SwitchVLANEnabled = model.SwitchVLANEnabled.ValueBool()

	// Advanced features
	if !model.OutdoorModeOverride.IsNull() {
		device.OutdoorModeOverride = model.OutdoorModeOverride.ValueString()
	}
	if !model.Volume.IsNull() {
		device.Volume = int(model.Volume.ValueInt64())
	}
	if !model.XBaresipPassword.IsNull() {
		device.XBaresipPassword = model.XBaresipPassword.ValueString()
	}

	// LCD/LCM settings
	if !model.LcmBrightness.IsNull() {
		device.LcmBrightness = int(model.LcmBrightness.ValueInt64())
	}
	device.LcmBrightnessOverride = model.LcmBrightnessOverride.ValueBool()
	if !model.LcmIDleTimeout.IsNull() {
		device.LcmIDleTimeout = int(model.LcmIDleTimeout.ValueInt64())
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
			device.ConfigNetwork = *configNetwork
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

	return device, diags
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
			Number: types.Int64Value(int64(po.PortIDX)),
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
			model.OpMode = types.StringValue("switch")
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
		if po.Dot1XIDleTimeout == 0 {
			model.Dot1XIDleTimeout = types.Int64Null()
		} else {
			model.Dot1XIDleTimeout = types.Int64Value(int64(po.Dot1XIDleTimeout))
		}

		if po.EgressRateLimitKbps == 0 {
			model.EgressRateLimitKbps = types.Int64Null()
		} else {
			model.EgressRateLimitKbps = types.Int64Value(int64(po.EgressRateLimitKbps))
		}

		if po.MirrorPortIDX == 0 {
			model.MirrorPortIDX = types.Int64Null()
		} else {
			model.MirrorPortIDX = types.Int64Value(int64(po.MirrorPortIDX))
		}

		if po.PriorityQueue1Level == 0 {
			model.PriorityQueue1Level = types.Int64Null()
		} else {
			model.PriorityQueue1Level = types.Int64Value(int64(po.PriorityQueue1Level))
		}

		if po.PriorityQueue2Level == 0 {
			model.PriorityQueue2Level = types.Int64Null()
		} else {
			model.PriorityQueue2Level = types.Int64Value(int64(po.PriorityQueue2Level))
		}

		if po.PriorityQueue3Level == 0 {
			model.PriorityQueue3Level = types.Int64Null()
		} else {
			model.PriorityQueue3Level = types.Int64Value(int64(po.PriorityQueue3Level))
		}

		if po.PriorityQueue4Level == 0 {
			model.PriorityQueue4Level = types.Int64Null()
		} else {
			model.PriorityQueue4Level = types.Int64Value(int64(po.PriorityQueue4Level))
		}

		if po.Speed == 0 {
			model.Speed = types.Int64Null()
		} else {
			model.Speed = types.Int64Value(int64(po.Speed))
		}

		if po.StormctrlBroadcastastLevel == 0 {
			model.StormctrlBroadcastLevel = types.Int64Null()
		} else {
			model.StormctrlBroadcastLevel = types.Int64Value(int64(po.StormctrlBroadcastastLevel))
		}

		if po.StormctrlBroadcastastRate == 0 {
			model.StormctrlBroadcastRate = types.Int64Null()
		} else {
			model.StormctrlBroadcastRate = types.Int64Value(int64(po.StormctrlBroadcastastRate))
		}

		if po.StormctrlMcastLevel == 0 {
			model.StormctrlMcastLevel = types.Int64Null()
		} else {
			model.StormctrlMcastLevel = types.Int64Value(int64(po.StormctrlMcastLevel))
		}

		if po.StormctrlMcastRate == 0 {
			model.StormctrlMcastRate = types.Int64Null()
		} else {
			model.StormctrlMcastRate = types.Int64Value(int64(po.StormctrlMcastRate))
		}

		if po.StormctrlUcastLevel == 0 {
			model.StormctrlUcastLevel = types.Int64Null()
		} else {
			model.StormctrlUcastLevel = types.Int64Value(int64(po.StormctrlUcastLevel))
		}

		if po.StormctrlUcastRate == 0 {
			model.StormctrlUcastRate = types.Int64Null()
		} else {
			model.StormctrlUcastRate = types.Int64Value(int64(po.StormctrlUcastRate))
		}

		// List attributes
		if len(po.AggregateMembers) == 0 {
			model.AggregateMembers = types.ListNull(types.Int64Type)
		} else {
			aggrMemberValues := make([]attr.Value, 0, len(po.AggregateMembers))
			for _, member := range po.AggregateMembers {
				aggrMemberValues = append(aggrMemberValues, types.Int64Value(int64(member)))
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
			excludedValues := make([]attr.Value, 0, len(po.ExcludedNetworkIDs))
			for _, id := range po.ExcludedNetworkIDs {
				excludedValues = append(excludedValues, types.StringValue(id))
			}
			listVal, listDiags := types.ListValue(types.StringType, excludedValues)
			diags.Append(listDiags...)
			if diags.HasError() {
				continue
			}
			model.ExcludedNetworkIDs = listVal
		}

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
		objVal, objDiags := types.ObjectValueFrom(ctx, portOverrideAttrTypes(), model)
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
	overrideMap := make(map[int]unifi.DevicePortOverrides)

	for _, elem := range elements {
		var model portOverrideModel
		diags.Append(elem.(types.Object).As(ctx, &model, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		idx := int(model.Number.ValueInt64())
		po := unifi.DevicePortOverrides{
			PortIDX: idx,
		}

		// String attributes
		if !model.Name.IsNull() {
			po.Name = model.Name.ValueString()
		}
		if !model.PortProfileID.IsNull() {
			po.PortProfileID = model.PortProfileID.ValueString()
		}
		if !model.OpMode.IsNull() {
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
			po.Dot1XIDleTimeout = int(model.Dot1XIDleTimeout.ValueInt64())
		}
		if !model.EgressRateLimitKbps.IsNull() {
			po.EgressRateLimitKbps = int(model.EgressRateLimitKbps.ValueInt64())
		}
		if !model.MirrorPortIDX.IsNull() {
			po.MirrorPortIDX = int(model.MirrorPortIDX.ValueInt64())
		}
		if !model.PriorityQueue1Level.IsNull() {
			po.PriorityQueue1Level = int(model.PriorityQueue1Level.ValueInt64())
		}
		if !model.PriorityQueue2Level.IsNull() {
			po.PriorityQueue2Level = int(model.PriorityQueue2Level.ValueInt64())
		}
		if !model.PriorityQueue3Level.IsNull() {
			po.PriorityQueue3Level = int(model.PriorityQueue3Level.ValueInt64())
		}
		if !model.PriorityQueue4Level.IsNull() {
			po.PriorityQueue4Level = int(model.PriorityQueue4Level.ValueInt64())
		}
		if !model.Speed.IsNull() {
			po.Speed = int(model.Speed.ValueInt64())
		}
		if !model.StormctrlBroadcastLevel.IsNull() {
			po.StormctrlBroadcastastLevel = int(model.StormctrlBroadcastLevel.ValueInt64())
		}
		if !model.StormctrlBroadcastRate.IsNull() {
			po.StormctrlBroadcastastRate = int(model.StormctrlBroadcastRate.ValueInt64())
		}
		if !model.StormctrlMcastLevel.IsNull() {
			po.StormctrlMcastLevel = int(model.StormctrlMcastLevel.ValueInt64())
		}
		if !model.StormctrlMcastRate.IsNull() {
			po.StormctrlMcastRate = int(model.StormctrlMcastRate.ValueInt64())
		}
		if !model.StormctrlUcastLevel.IsNull() {
			po.StormctrlUcastLevel = int(model.StormctrlUcastLevel.ValueInt64())
		}
		if !model.StormctrlUcastRate.IsNull() {
			po.StormctrlUcastRate = int(model.StormctrlUcastRate.ValueInt64())
		}

		// List attributes
		if !model.AggregateMembers.IsNull() {
			var aggrMembers []int
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
			diags.Append(model.MulticastRouterNetworkIDs.ElementsAs(ctx, &multicastIDs, true)...)
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
		"number":                           types.Int64Type,
		"name":                             types.StringType,
		"port_profile_id":                  types.StringType,
		"op_mode":                          types.StringType,
		"poe_mode":                         types.StringType,
		"aggregate_members":                types.ListType{ElemType: types.Int64Type},
		"autoneg":                          types.BoolType,
		"dot1x_ctrl":                       types.StringType,
		"dot1x_idle_timeout":               types.Int64Type,
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
func int64OrNull(i int) types.Int64 {
	if i == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(i))
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
			Ht:                     int64OrNull(radio.Ht),
			TxPower:                stringOrNull(radio.TxPower),
			TxPowerMode:            stringOrNull(radio.TxPowerMode),
			MinRssiEnabled:         types.BoolValue(radio.MinRssiEnabled),
			MinRssi:                int64OrNull(radio.MinRssi),
			AntennaGain:            int64OrNull(radio.AntennaGain),
			AntennaID:              int64OrNull(radio.AntennaID),
			AssistedRoamingEnabled: types.BoolValue(radio.AssistedRoamingEnabled),
			AssistedRoamingRssi:    int64OrNull(radio.AssistedRoamingRssi),
			Dfs:                    types.BoolValue(radio.Dfs),
			HardNoiseFloorEnabled:  types.BoolValue(radio.HardNoiseFloorEnabled),
			LoadbalanceEnabled:     types.BoolValue(radio.LoadbalanceEnabled),
			Maxsta:                 int64OrNull(radio.Maxsta),
			Name:                   stringOrNull(radio.Name),
			SensLevel:              int64OrNull(radio.SensLevel),
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
			Index:        types.Int64Value(int64(outlet.Index)),
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
			Ht:                     int(model.Ht.ValueInt64()),
			TxPower:                model.TxPower.ValueString(),
			TxPowerMode:            model.TxPowerMode.ValueString(),
			MinRssiEnabled:         model.MinRssiEnabled.ValueBool(),
			MinRssi:                int(model.MinRssi.ValueInt64()),
			AntennaGain:            int(model.AntennaGain.ValueInt64()),
			AntennaID:              int(model.AntennaID.ValueInt64()),
			AssistedRoamingEnabled: model.AssistedRoamingEnabled.ValueBool(),
			AssistedRoamingRssi:    int(model.AssistedRoamingRssi.ValueInt64()),
			Dfs:                    model.Dfs.ValueBool(),
			HardNoiseFloorEnabled:  model.HardNoiseFloorEnabled.ValueBool(),
			LoadbalanceEnabled:     model.LoadbalanceEnabled.ValueBool(),
			Maxsta:                 int(model.Maxsta.ValueInt64()),
			Name:                   model.Name.ValueString(),
			SensLevel:              int(model.SensLevel.ValueInt64()),
			SensLevelEnabled:       model.SensLevelEnabled.ValueBool(),
			VwireEnabled:           model.VwireEnabled.ValueBool(),
		}

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
			Index:        int(model.Index.ValueInt64()),
			Name:         model.Name.ValueString(),
			RelayState:   model.RelayState.ValueBool(),
			CycleEnabled: model.CycleEnabled.ValueBool(),
		}

		outlets = append(outlets, outlet)
	}

	return outlets, diags
}
