package unifi

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var (
	_ resource.Resource                 = &settingResource{}
	_ resource.ResourceWithImportState  = &settingResource{}
	_ resource.ResourceWithUpgradeState = &settingResource{}
)

func NewSettingResource() resource.Resource {
	return &settingResource{}
}

type settingResource struct {
	client *Client
}

type sshKeyModel struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Key     types.String `tfsdk:"key"`
	Comment types.String `tfsdk:"comment"`
}

type settingMgmtModel struct {
	AutoUpgrade            types.Bool   `tfsdk:"auto_upgrade"`
	AutoUpgradeHour        types.Int64  `tfsdk:"auto_upgrade_hour"`
	SSHEnabled             types.Bool   `tfsdk:"ssh_enabled"`
	SSHKeys                types.List   `tfsdk:"ssh_keys"`
	AdvancedFeatureEnabled types.Bool   `tfsdk:"advanced_feature_enabled"`
	DebugToolsEnabled      types.Bool   `tfsdk:"debug_tools_enabled"`
	DirectConnectEnabled   types.Bool   `tfsdk:"direct_connect_enabled"`
	UnifiIdpEnabled        types.Bool   `tfsdk:"unifi_idp_enabled"`
	WifimanEnabled         types.Bool   `tfsdk:"wifiman_enabled"`
	SSHUsername            types.String `tfsdk:"ssh_username"`
	SSHPassword            types.String `tfsdk:"ssh_password"`
	SSHAuthPasswordEnabled types.Bool   `tfsdk:"ssh_auth_password_enabled"`
}

type settingRadiusModel struct {
	AccountingEnabled     types.Bool           `tfsdk:"accounting_enabled"`
	AcctPort              types.Int64          `tfsdk:"acct_port"`
	AuthPort              types.Int64          `tfsdk:"auth_port"`
	InterimUpdateInterval timetypes.GoDuration `tfsdk:"interim_update_interval"`
	Secret                types.String         `tfsdk:"secret"`
}

type dnsVerificationModel struct {
	Domain             types.String `tfsdk:"domain"`
	PrimaryDNSServer   types.String `tfsdk:"primary_dns_server"`
	SecondaryDNSServer types.String `tfsdk:"secondary_dns_server"`
	SettingPreference  types.String `tfsdk:"setting_preference"`
}

type settingUSGModel struct {
	BroadcastPing                  types.Bool           `tfsdk:"broadcast_ping"`
	DNSVerification                types.Object         `tfsdk:"dns_verification"`
	FtpModule                      types.Bool           `tfsdk:"ftp_module"`
	GeoIPFilteringBlock            types.String         `tfsdk:"geo_ip_filtering_block"`
	GeoIPFilteringCountries        types.String         `tfsdk:"geo_ip_filtering_countries"`
	GeoIPFilteringEnabled          types.Bool           `tfsdk:"geo_ip_filtering_enabled"`
	GeoIPFilteringTrafficDirection types.String         `tfsdk:"geo_ip_filtering_traffic_direction"`
	GreModule                      types.Bool           `tfsdk:"gre_module"`
	H323Module                     types.Bool           `tfsdk:"h323_module"`
	ICMPTimeout                    timetypes.GoDuration `tfsdk:"icmp_timeout"`
	MssClamp                       types.String         `tfsdk:"mss_clamp"`
	OffloadAccounting              types.Bool           `tfsdk:"offload_accounting"`
	OffloadL2Blocking              types.Bool           `tfsdk:"offload_l2_blocking"`
	OffloadSch                     types.Bool           `tfsdk:"offload_sch"`
	OtherTimeout                   timetypes.GoDuration `tfsdk:"other_timeout"`
	PptpModule                     types.Bool           `tfsdk:"pptp_module"`
	ReceiveRedirects               types.Bool           `tfsdk:"receive_redirects"`
	SendRedirects                  types.Bool           `tfsdk:"send_redirects"`
	SipModule                      types.Bool           `tfsdk:"sip_module"`
	SynCookies                     types.Bool           `tfsdk:"syn_cookies"`
	TCPCloseTimeout                timetypes.GoDuration `tfsdk:"tcp_close_timeout"`
	TCPCloseWaitTimeout            timetypes.GoDuration `tfsdk:"tcp_close_wait_timeout"`
	TCPEstablishedTimeout          timetypes.GoDuration `tfsdk:"tcp_established_timeout"`
	TCPFinWaitTimeout              timetypes.GoDuration `tfsdk:"tcp_fin_wait_timeout"`
	TCPLastAckTimeout              timetypes.GoDuration `tfsdk:"tcp_last_ack_timeout"`
	TCPSynRecvTimeout              timetypes.GoDuration `tfsdk:"tcp_syn_recv_timeout"`
	TCPSynSentTimeout              timetypes.GoDuration `tfsdk:"tcp_syn_sent_timeout"`
	TCPTimeWaitTimeout             timetypes.GoDuration `tfsdk:"tcp_time_wait_timeout"`
	TFTPModule                     types.Bool           `tfsdk:"tftp_module"`
	TimeoutSettingPreference       types.String         `tfsdk:"timeout_setting_preference"`
	UDPOtherTimeout                timetypes.GoDuration `tfsdk:"udp_other_timeout"`
	UDPStreamTimeout               timetypes.GoDuration `tfsdk:"udp_stream_timeout"`
	UnbindWANMonitors              types.Bool           `tfsdk:"unbind_wan_monitors"`
	UPnPEnabled                    types.Bool           `tfsdk:"upnp_enabled"`
	UPnPNATPmpEnabled              types.Bool           `tfsdk:"upnp_nat_pmp_enabled"`
	UPnPSecureMode                 types.Bool           `tfsdk:"upnp_secure_mode"`
	UPnPWANInterface               types.String         `tfsdk:"upnp_wan_interface"`
}

type settingDohCustomServerModel struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	SDNSStamp  types.String `tfsdk:"sdns_stamp"`
	ServerName types.String `tfsdk:"server_name"`
}

type settingAutoSpeedtestModel struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	CronExpr types.String `tfsdk:"cron_expr"`
}

type settingCountryModel struct {
	Code types.Int64 `tfsdk:"code"`
}

type settingDpiModel struct {
	Enabled               types.Bool `tfsdk:"enabled"`
	FingerprintingEnabled types.Bool `tfsdk:"fingerprinting_enabled"`
}

type settingLcmModel struct {
	Enabled     types.Bool  `tfsdk:"enabled"`
	Brightness  types.Int64 `tfsdk:"brightness"`
	IdleTimeout types.Int64 `tfsdk:"idle_timeout"`
	Sync        types.Bool  `tfsdk:"sync"`
	TouchEvent  types.Bool  `tfsdk:"touch_event"`
}

type settingNetworkOptimizationModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type settingNtpModel struct {
	NtpServer1        types.String `tfsdk:"ntp_server_1"`
	NtpServer2        types.String `tfsdk:"ntp_server_2"`
	NtpServer3        types.String `tfsdk:"ntp_server_3"`
	NtpServer4        types.String `tfsdk:"ntp_server_4"`
	SettingPreference types.String `tfsdk:"setting_preference"`
}

type settingSyslogModel struct {
	Enabled                     types.Bool   `tfsdk:"enabled"`
	Contents                    types.List   `tfsdk:"contents"`
	Debug                       types.Bool   `tfsdk:"debug"`
	IP                          types.String `tfsdk:"ip"`
	Port                        types.Int64  `tfsdk:"port"`
	LogAllContents              types.Bool   `tfsdk:"log_all_contents"`
	NetconsoleEnabled           types.Bool   `tfsdk:"netconsole_enabled"`
	NetconsoleHost              types.String `tfsdk:"netconsole_host"`
	NetconsolePort              types.Int64  `tfsdk:"netconsole_port"`
	ThisController              types.Bool   `tfsdk:"this_controller"`
	ThisControllerEncryptedOnly types.Bool   `tfsdk:"this_controller_encrypted_only"`
}

type settingDohModel struct {
	CustomServers types.List   `tfsdk:"custom_servers"`
	ServerNames   types.List   `tfsdk:"server_names"`
	State         types.String `tfsdk:"state"`
}

type settingIpsHoneypotModel struct {
	IPAddress types.String `tfsdk:"ip_address"`
	NetworkID types.String `tfsdk:"network_id"`
	Version   types.String `tfsdk:"version"`
}

type settingIpsWhitelistModel struct {
	Direction types.String `tfsdk:"direction"`
	Mode      types.String `tfsdk:"mode"`
	Value     types.String `tfsdk:"value"`
}

type settingIpsTrackingModel struct {
	Direction types.String `tfsdk:"direction"`
	Mode      types.String `tfsdk:"mode"`
	Value     types.String `tfsdk:"value"`
}

type settingIpsAlertModel struct {
	Category  types.String `tfsdk:"category"`
	Gid       types.Int64  `tfsdk:"gid"`
	ID        types.Int64  `tfsdk:"id"`
	Signature types.String `tfsdk:"signature"`
	Type      types.String `tfsdk:"type"`
	Tracking  types.List   `tfsdk:"tracking"`
}

type settingIpsModel struct {
	AdvancedFilteringPreference         types.String `tfsdk:"advanced_filtering_preference"`
	ContentFilteringBlockingPageEnabled types.Bool   `tfsdk:"content_filtering_blocking_page_enabled"`
	EnabledCategories                   types.List   `tfsdk:"enabled_categories"`
	EnabledNetworks                     types.List   `tfsdk:"enabled_networks"`
	Honeypot                            types.List   `tfsdk:"honeypot"`
	HoneypotEnabled                     types.Bool   `tfsdk:"honeypot_enabled"`
	IPSMode                             types.String `tfsdk:"ips_mode"`
	MemoryOptimized                     types.Bool   `tfsdk:"memory_optimized"`
	RestrictTorrents                    types.Bool   `tfsdk:"restrict_torrents"`
	SuppressionWhitelist                types.List   `tfsdk:"suppression_whitelist"`
	SuppressionAlerts                   types.List   `tfsdk:"suppression_alerts"`
}

type settingResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Site          types.String   `tfsdk:"site"`
	AutoSpeedtest types.Object   `tfsdk:"auto_speedtest"`
	Country       types.Object   `tfsdk:"country"`
	Dpi           types.Object   `tfsdk:"dpi"`
	Lcm           types.Object   `tfsdk:"lcm"`
	NetworkOpt    types.Object   `tfsdk:"network_optimization"`
	Ntp           types.Object   `tfsdk:"ntp"`
	Syslog        types.Object   `tfsdk:"syslog"`
	Doh           types.Object   `tfsdk:"doh"`
	Ips           types.Object   `tfsdk:"ips"`
	Mgmt          types.Object   `tfsdk:"mgmt"`
	Radius        types.Object   `tfsdk:"radius"`
	USG           types.Object   `tfsdk:"usg"`
	IgmpSnooping  types.Object   `tfsdk:"igmp_snooping"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

// settingIgmpSnoopingModel is the nested igmp_snooping block. On UniFi 10.3.x the
// effective IGMP snooping toggle moved from the per-network object to this site
// setting (#164). Only the common fields are exposed; advanced querier/flood
// fields are preserved across updates via a read-modify-write merge.
type settingIgmpSnoopingModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	NetworkIDs types.List `tfsdk:"network_ids"`
}

// Shared attribute-type maps for the doh/ips nested objects and lists. These
// are referenced from both readSettings and the *SettingToModel conversion
// helpers, so they live at package level to avoid drift between the two.
var (
	autoSpeedtestAttrTypes = map[string]attr.Type{
		"enabled":   types.BoolType,
		"cron_expr": types.StringType,
	}
	mgmtSSHKeyAttrTypes = map[string]attr.Type{
		"name":    types.StringType,
		"type":    types.StringType,
		"key":     types.StringType,
		"comment": types.StringType,
	}
	mgmtAttrTypes = map[string]attr.Type{
		"auto_upgrade":      types.BoolType,
		"auto_upgrade_hour": types.Int64Type,
		"ssh_enabled":       types.BoolType,
		"ssh_keys": types.ListType{
			ElemType: types.ObjectType{AttrTypes: mgmtSSHKeyAttrTypes},
		},
		"advanced_feature_enabled":  types.BoolType,
		"debug_tools_enabled":       types.BoolType,
		"direct_connect_enabled":    types.BoolType,
		"unifi_idp_enabled":         types.BoolType,
		"wifiman_enabled":           types.BoolType,
		"ssh_username":              types.StringType,
		"ssh_password":              types.StringType,
		"ssh_auth_password_enabled": types.BoolType,
	}
	countryAttrTypes = map[string]attr.Type{
		"code": types.Int64Type,
	}
	dpiAttrTypes = map[string]attr.Type{
		"enabled":                types.BoolType,
		"fingerprinting_enabled": types.BoolType,
	}
	lcmAttrTypes = map[string]attr.Type{
		"enabled":      types.BoolType,
		"brightness":   types.Int64Type,
		"idle_timeout": types.Int64Type,
		"sync":         types.BoolType,
		"touch_event":  types.BoolType,
	}
	networkOptimizationAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
	}
	ntpAttrTypes = map[string]attr.Type{
		"ntp_server_1":       types.StringType,
		"ntp_server_2":       types.StringType,
		"ntp_server_3":       types.StringType,
		"ntp_server_4":       types.StringType,
		"setting_preference": types.StringType,
	}
	syslogAttrTypes = map[string]attr.Type{
		"enabled":                        types.BoolType,
		"contents":                       types.ListType{ElemType: types.StringType},
		"debug":                          types.BoolType,
		"ip":                             types.StringType,
		"port":                           types.Int64Type,
		"log_all_contents":               types.BoolType,
		"netconsole_enabled":             types.BoolType,
		"netconsole_host":                types.StringType,
		"netconsole_port":                types.Int64Type,
		"this_controller":                types.BoolType,
		"this_controller_encrypted_only": types.BoolType,
	}
	dohCustomServerAttrTypes = map[string]attr.Type{
		"enabled":     types.BoolType,
		"sdns_stamp":  types.StringType,
		"server_name": types.StringType,
	}
	dohAttrTypes = map[string]attr.Type{
		"state":        types.StringType,
		"server_names": types.ListType{ElemType: types.StringType},
		"custom_servers": types.ListType{
			ElemType: types.ObjectType{AttrTypes: dohCustomServerAttrTypes},
		},
	}
	ipsHoneypotAttrTypes = map[string]attr.Type{
		"ip_address": types.StringType,
		"network_id": types.StringType,
		"version":    types.StringType,
	}
	ipsWhitelistAttrTypes = map[string]attr.Type{
		"direction": types.StringType,
		"mode":      types.StringType,
		"value":     types.StringType,
	}
	ipsTrackingAttrTypes = map[string]attr.Type{
		"direction": types.StringType,
		"mode":      types.StringType,
		"value":     types.StringType,
	}
	ipsAlertAttrTypes = map[string]attr.Type{
		"category":  types.StringType,
		"gid":       types.Int64Type,
		"id":        types.Int64Type,
		"signature": types.StringType,
		"type":      types.StringType,
		"tracking":  types.ListType{ElemType: types.ObjectType{AttrTypes: ipsTrackingAttrTypes}},
	}
	ipsAttrTypes = map[string]attr.Type{
		"advanced_filtering_preference":           types.StringType,
		"content_filtering_blocking_page_enabled": types.BoolType,
		"enabled_categories":                      types.ListType{ElemType: types.StringType},
		"enabled_networks":                        types.ListType{ElemType: types.StringType},
		"honeypot_enabled":                        types.BoolType,
		"honeypot": types.ListType{
			ElemType: types.ObjectType{AttrTypes: ipsHoneypotAttrTypes},
		},
		"ips_mode":          types.StringType,
		"memory_optimized":  types.BoolType,
		"restrict_torrents": types.BoolType,
		"suppression_whitelist": types.ListType{
			ElemType: types.ObjectType{AttrTypes: ipsWhitelistAttrTypes},
		},
		"suppression_alerts": types.ListType{
			ElemType: types.ObjectType{AttrTypes: ipsAlertAttrTypes},
		},
	}
	igmpSnoopingAttrTypes = map[string]attr.Type{
		"enabled":     types.BoolType,
		"network_ids": types.ListType{ElemType: types.StringType},
	}
)

func (r *settingResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_setting"
}

func (r *settingResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		// v1: radius.interim_update_interval and the usg conntrack timeouts
		// changed from Int64 (seconds) to GoDuration strings. See UpgradeState.
		Version:             1,
		MarkdownDescription: "Manages settings for a UniFi site. Configure only the settings you need by providing the corresponding nested object.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the settings.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the settings with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_speedtest": schema.SingleNestedAttribute{
				MarkdownDescription: "Periodic automated internet speed test settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether periodic automated speed tests are enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"cron_expr": schema.StringAttribute{
						MarkdownDescription: "Cron expression controlling when the speed test runs (e.g. `0 * * * *`).",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"country": schema.SingleNestedAttribute{
				MarkdownDescription: "Regulatory country settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"code": schema.Int64Attribute{
						MarkdownDescription: "Regulatory country code (ISO 3166-1 numeric).",
						Required:            true,
					},
				},
			},
			"dpi": schema.SingleNestedAttribute{
				MarkdownDescription: "Deep Packet Inspection (DPI) settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether DPI is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"fingerprinting_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether device fingerprinting is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
			"lcm": schema.SingleNestedAttribute{
				MarkdownDescription: "LCD/display (LCM) settings for devices with a screen.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether the device display is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"brightness": schema.Int64Attribute{
						MarkdownDescription: "Display brightness (1-100).",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.Int64{int64validator.Between(1, 100)},
					},
					"idle_timeout": schema.Int64Attribute{
						MarkdownDescription: "Seconds of inactivity before the display turns off (10-3600).",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.Int64{int64validator.Between(10, 3600)},
					},
					"sync": schema.BoolAttribute{
						MarkdownDescription: "Sync display settings across devices.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"touch_event": schema.BoolAttribute{
						MarkdownDescription: "Whether touch events on the display are enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
				},
			},
			"network_optimization": schema.SingleNestedAttribute{
				MarkdownDescription: "Automated network optimization settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether automated network optimization is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
			"ntp": schema.SingleNestedAttribute{
				MarkdownDescription: "NTP (time server) settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"setting_preference": schema.StringAttribute{
						MarkdownDescription: "Configuration mode: `auto` or `manual`.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("auto", "manual"),
						},
					},
					"ntp_server_1": schema.StringAttribute{
						MarkdownDescription: "Primary NTP server.",
						Optional:            true,
						Computed:            true,
					},
					"ntp_server_2": schema.StringAttribute{
						MarkdownDescription: "Second NTP server.",
						Optional:            true,
						Computed:            true,
					},
					"ntp_server_3": schema.StringAttribute{
						MarkdownDescription: "Third NTP server.",
						Optional:            true,
						Computed:            true,
					},
					"ntp_server_4": schema.StringAttribute{
						MarkdownDescription: "Fourth NTP server.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"syslog": schema.SingleNestedAttribute{
				MarkdownDescription: "Remote syslog (rsyslogd) settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether remote syslog is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"ip": schema.StringAttribute{
						MarkdownDescription: "Remote syslog server IP address.",
						Optional:            true,
						Computed:            true,
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "Remote syslog server port (1-65535).",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.Int64{int64validator.Between(1, 65535)},
					},
					"contents": schema.ListAttribute{
						MarkdownDescription: "Logged facilities (e.g. `device`, `client`, `firewall_default_policy`, `triggers`, `updates`, `admin_activity`, `critical`, `security_detections`, `vpn`).",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
					},
					"log_all_contents": schema.BoolAttribute{
						MarkdownDescription: "Log all available facilities.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"debug": schema.BoolAttribute{
						MarkdownDescription: "Enable debug logging.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"this_controller": schema.BoolAttribute{
						MarkdownDescription: "Also log this controller's events.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"this_controller_encrypted_only": schema.BoolAttribute{
						MarkdownDescription: "Only send this controller's logs over an encrypted channel.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"netconsole_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether netconsole logging is enabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"netconsole_host": schema.StringAttribute{
						MarkdownDescription: "Netconsole host.",
						Optional:            true,
						Computed:            true,
					},
					"netconsole_port": schema.Int64Attribute{
						MarkdownDescription: "Netconsole port (1-65535).",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.Int64{int64validator.Between(1, 65535)},
					},
				},
			},
			"doh": schema.SingleNestedAttribute{
				MarkdownDescription: "Encrypted DNS (DNS-over-HTTPS) settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"state": schema.StringAttribute{
						MarkdownDescription: "Encrypted DNS state: off, auto, manual, or custom.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("off", "auto", "manual", "custom"),
						},
					},
					"server_names": schema.ListAttribute{
						MarkdownDescription: "Predefined DNS provider names (e.g. \"cloudflare\", \"google\").",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
					},
					"custom_servers": schema.ListNestedAttribute{
						MarkdownDescription: "Custom DNS servers specified via DNS stamp.",
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									MarkdownDescription: "Enable this custom server. Defaults to true.",
									Optional:            true,
									Computed:            true,
									Default:             booldefault.StaticBool(true),
								},
								"sdns_stamp": schema.StringAttribute{
									MarkdownDescription: "DNS stamp (sdns://) for the custom resolver.",
									Required:            true,
								},
								"server_name": schema.StringAttribute{
									MarkdownDescription: "Human-readable name for this custom server.",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"ips": schema.SingleNestedAttribute{
				MarkdownDescription: "Intrusion Prevention System (IPS/IDS) and threat management settings. Basic IDS/IPS uses the built-in Emerging Threats ruleset and is free. A UniFi CyberSecure subscription adds enhanced threat intelligence from Proofpoint and Cloudflare on top of the base ruleset.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"ips_mode": schema.StringAttribute{
						MarkdownDescription: "IPS operating mode: ids (detect only), ips (detect and block), ipsInline, or disabled.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("ids", "ips", "ipsInline", "disabled"),
						},
					},
					"enabled_categories": schema.ListAttribute{
						MarkdownDescription: "Emerging Threats ruleset categories to enable (e.g. \"emerging-malware\", \"tor\", \"phishing\").",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
					},
					"enabled_networks": schema.ListAttribute{
						MarkdownDescription: "Network IDs to apply IPS inspection to.",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
					},
					"honeypot_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable honeypot to detect internal port scans.",
						Optional:            true,
						Computed:            true,
					},
					"honeypot": schema.ListNestedAttribute{
						MarkdownDescription: "Honeypot IP addresses per network.",
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"ip_address": schema.StringAttribute{
									MarkdownDescription: "IP address to use as a honeypot.",
									Required:            true,
								},
								"network_id": schema.StringAttribute{
									MarkdownDescription: "Network ID this honeypot IP belongs to.",
									Required:            true,
								},
								"version": schema.StringAttribute{
									MarkdownDescription: "IP version: v4 or v6.",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("v4", "v6"),
									},
								},
							},
						},
					},
					"restrict_torrents": schema.BoolAttribute{
						MarkdownDescription: "Block BitTorrent traffic.",
						Optional:            true,
						Computed:            true,
					},
					"content_filtering_blocking_page_enabled": schema.BoolAttribute{
						MarkdownDescription: "Show a blocking page when content filtering blocks a request.",
						Optional:            true,
						Computed:            true,
					},
					"memory_optimized": schema.BoolAttribute{
						MarkdownDescription: "Use memory-optimized IPS ruleset (reduced rule set for low-memory devices).",
						Optional:            true,
						Computed:            true,
					},
					"advanced_filtering_preference": schema.StringAttribute{
						MarkdownDescription: "Advanced filtering mode: manual or disabled.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("manual", "disabled"),
						},
					},
					"suppression_alerts": schema.ListNestedAttribute{
						MarkdownDescription: "IPS signature alert suppression entries — silence specific signatures or categories.",
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"category": schema.StringAttribute{
									MarkdownDescription: "Alert suppression signature category.",
									Optional:            true,
									Computed:            true,
								},
								"gid": schema.Int64Attribute{
									MarkdownDescription: "Signature Generator ID (GID).",
									Optional:            true,
									Computed:            true,
								},
								"id": schema.Int64Attribute{
									MarkdownDescription: "Signature ID.",
									Optional:            true,
									Computed:            true,
								},
								"signature": schema.StringAttribute{
									MarkdownDescription: "Suppression signature name.",
									Optional:            true,
									Computed:            true,
								},
								"type": schema.StringAttribute{
									MarkdownDescription: "Suppression type: `all` (everywhere) or `track` (only the tracked sources/destinations).",
									Optional:            true,
									Computed:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("all", "track"),
									},
								},
								"tracking": schema.ListNestedAttribute{
									MarkdownDescription: "Tracking specifications (used when `type` is `track`).",
									Optional:            true,
									Computed:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"direction": schema.StringAttribute{
												MarkdownDescription: "Match direction: both, src, or dest.",
												Required:            true,
												Validators: []validator.String{
													stringvalidator.OneOf("both", "src", "dest"),
												},
											},
											"mode": schema.StringAttribute{
												MarkdownDescription: "Match mode: ip, subnet, or network.",
												Required:            true,
												Validators: []validator.String{
													stringvalidator.OneOf(
														"ip",
														"subnet",
														"network",
													),
												},
											},
											"value": schema.StringAttribute{
												MarkdownDescription: "IP address, CIDR subnet, or network ID to match.",
												Required:            true,
											},
										},
									},
								},
							},
						},
					},
					"suppression_whitelist": schema.ListNestedAttribute{
						MarkdownDescription: "IPS suppression whitelist entries — sources/destinations to exclude from inspection.",
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"direction": schema.StringAttribute{
									MarkdownDescription: "Match direction: both, src, or dest.",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("both", "src", "dest"),
									},
								},
								"mode": schema.StringAttribute{
									MarkdownDescription: "Match mode: ip, subnet, or network.",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("ip", "subnet", "network"),
									},
								},
								"value": schema.StringAttribute{
									MarkdownDescription: "IP address, CIDR subnet, or network ID to whitelist.",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"mgmt": schema.SingleNestedAttribute{
				MarkdownDescription: "Management settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"auto_upgrade": schema.BoolAttribute{
						MarkdownDescription: "Automatically upgrade device firmware.",
						Optional:            true,
						Computed:            true,
					},
					"ssh_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable SSH authentication.",
						Optional:            true,
						Computed:            true,
					},
					"auto_upgrade_hour": schema.Int64Attribute{
						MarkdownDescription: "Hour of day (0-23) for automatic firmware upgrades.",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.Int64{int64validator.Between(0, 23)},
					},
					"advanced_feature_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable advanced features.",
						Optional:            true,
						Computed:            true,
					},
					"debug_tools_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable debug tools.",
						Optional:            true,
						Computed:            true,
					},
					"direct_connect_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable Direct Connect (remote access).",
						Optional:            true,
						Computed:            true,
					},
					"unifi_idp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable the UniFi Identity Provider.",
						Optional:            true,
						Computed:            true,
					},
					"wifiman_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable WiFiman.",
						Optional:            true,
						Computed:            true,
					},
					"ssh_username": schema.StringAttribute{
						MarkdownDescription: "SSH username for device access.",
						Optional:            true,
						Computed:            true,
					},
					"ssh_password": schema.StringAttribute{
						MarkdownDescription: "SSH password for device access. Sensitive — the controller " +
							"stores only a hash, so this value is kept from configuration and not read back.",
						Optional:  true,
						Sensitive: true,
					},
					"ssh_auth_password_enabled": schema.BoolAttribute{
						MarkdownDescription: "Allow SSH password authentication (in addition to keys).",
						Optional:            true,
						Computed:            true,
					},
					"ssh_keys": schema.ListNestedAttribute{
						MarkdownDescription: "SSH keys.",
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									MarkdownDescription: "Name of SSH key.",
									Required:            true,
								},
								"type": schema.StringAttribute{
									MarkdownDescription: "Type of SSH key, e.g. ssh-rsa.",
									Required:            true,
								},
								"key": schema.StringAttribute{
									MarkdownDescription: "Public SSH key.",
									Optional:            true,
									Computed:            true,
								},
								"comment": schema.StringAttribute{
									MarkdownDescription: "Comment.",
									Optional:            true,
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"radius": schema.SingleNestedAttribute{
				MarkdownDescription: "RADIUS settings.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"accounting_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable RADIUS accounting.",
						Optional:            true,
						Computed:            true,
					},
					"acct_port": schema.Int64Attribute{
						MarkdownDescription: "RADIUS accounting port.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"auth_port": schema.Int64Attribute{
						MarkdownDescription: "RADIUS authentication port.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"interim_update_interval": schema.StringAttribute{
						MarkdownDescription: "Interim update interval, as a Go duration string " +
							"(e.g. `1h`, `3600s`).",
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
					},
					"secret": schema.StringAttribute{
						MarkdownDescription: "RADIUS shared secret.",
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 48),
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[^\\\ "']+$`),
								"must not contain backslashes, spaces, single quotes, or double quotes",
							),
						},
					},
				},
			},
			"usg": schema.SingleNestedAttribute{
				MarkdownDescription: "USG settings.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"broadcast_ping": schema.BoolAttribute{
						MarkdownDescription: "Enable broadcast ping.",
						Optional:            true,
						Computed:            true,
					},
					"dns_verification": schema.SingleNestedAttribute{
						MarkdownDescription: "DNS verification settings.",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"domain": schema.StringAttribute{
								MarkdownDescription: "Domain for DNS verification.",
								Optional:            true,
								Computed:            true,
							},
							"primary_dns_server": schema.StringAttribute{
								MarkdownDescription: "Primary DNS server.",
								Optional:            true,
								Computed:            true,
							},
							"secondary_dns_server": schema.StringAttribute{
								MarkdownDescription: "Secondary DNS server.",
								Optional:            true,
								Computed:            true,
							},
							"setting_preference": schema.StringAttribute{
								MarkdownDescription: "Setting preference: auto or manual.",
								Optional:            true,
								Computed:            true,
							},
						},
					},
					"ftp_module": schema.BoolAttribute{
						MarkdownDescription: "Enable FTP module.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_block": schema.StringAttribute{
						MarkdownDescription: "Geo IP filtering action: block or allow.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_countries": schema.StringAttribute{
						MarkdownDescription: "Comma-separated list of country codes for geo IP filtering.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable geo IP filtering.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_traffic_direction": schema.StringAttribute{
						MarkdownDescription: "Geo IP filtering traffic direction: both, ingress, or egress.",
						Optional:            true,
						Computed:            true,
					},
					"gre_module": schema.BoolAttribute{
						MarkdownDescription: "Enable GRE module.",
						Optional:            true,
						Computed:            true,
					},
					"h323_module": schema.BoolAttribute{
						MarkdownDescription: "Enable H.323 module.",
						Optional:            true,
						Computed:            true,
					},
					"icmp_timeout": schema.StringAttribute{
						MarkdownDescription: "ICMP connection timeout, as a Go duration string (e.g. `30s`, `1m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"mss_clamp": schema.StringAttribute{
						MarkdownDescription: "MSS clamping mode: auto, custom, or disabled.",
						Optional:            true,
						Computed:            true,
					},
					"offload_accounting": schema.BoolAttribute{
						MarkdownDescription: "Enable hardware offload for accounting.",
						Optional:            true,
						Computed:            true,
					},
					"offload_l2_blocking": schema.BoolAttribute{
						MarkdownDescription: "Enable hardware offload for L2 blocking.",
						Optional:            true,
						Computed:            true,
					},
					"offload_sch": schema.BoolAttribute{
						MarkdownDescription: "Enable hardware offload for scheduling.",
						Optional:            true,
						Computed:            true,
					},
					"other_timeout": schema.StringAttribute{
						MarkdownDescription: "Other connections timeout, as a Go duration string (e.g. `600s`, `10m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"pptp_module": schema.BoolAttribute{
						MarkdownDescription: "Enable PPTP module.",
						Optional:            true,
						Computed:            true,
					},
					"receive_redirects": schema.BoolAttribute{
						MarkdownDescription: "Accept ICMP redirects.",
						Optional:            true,
						Computed:            true,
					},
					"send_redirects": schema.BoolAttribute{
						MarkdownDescription: "Send ICMP redirects.",
						Optional:            true,
						Computed:            true,
					},
					"sip_module": schema.BoolAttribute{
						MarkdownDescription: "Enable SIP module.",
						Optional:            true,
						Computed:            true,
					},
					"syn_cookies": schema.BoolAttribute{
						MarkdownDescription: "Enable SYN cookies.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_close_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP close timeout, as a Go duration string (e.g. `10s`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_close_wait_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP close wait timeout, as a Go duration string (e.g. `60s`, `1m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_established_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP established connection timeout, as a Go duration string (e.g. `7440s`, `2h4m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_fin_wait_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP fin wait timeout, as a Go duration string (e.g. `120s`, `2m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_last_ack_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP last ACK timeout, as a Go duration string (e.g. `30s`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_syn_recv_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP SYN received timeout, as a Go duration string (e.g. `60s`, `1m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_syn_sent_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP SYN sent timeout, as a Go duration string (e.g. `120s`, `2m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tcp_time_wait_timeout": schema.StringAttribute{
						MarkdownDescription: "TCP time wait timeout, as a Go duration string (e.g. `120s`, `2m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"tftp_module": schema.BoolAttribute{
						MarkdownDescription: "Enable TFTP module.",
						Optional:            true,
						Computed:            true,
					},
					"timeout_setting_preference": schema.StringAttribute{
						MarkdownDescription: "Timeout setting preference: auto or manual.",
						Optional:            true,
						Computed:            true,
					},
					"udp_other_timeout": schema.StringAttribute{
						MarkdownDescription: "UDP other timeout, as a Go duration string (e.g. `30s`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"udp_stream_timeout": schema.StringAttribute{
						MarkdownDescription: "UDP stream timeout, as a Go duration string (e.g. `180s`, `3m`).",
						CustomType:          timetypes.GoDurationType{},
						Optional:            true,
						Computed:            true,
					},
					"unbind_wan_monitors": schema.BoolAttribute{
						MarkdownDescription: "Unbind WAN monitors.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable UPnP.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_nat_pmp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable UPnP NAT-PMP.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_secure_mode": schema.BoolAttribute{
						MarkdownDescription: "Enable UPnP secure mode.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_wan_interface": schema.StringAttribute{
						MarkdownDescription: "UPnP WAN interface (e.g., WAN, WAN2).",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"igmp_snooping": schema.SingleNestedAttribute{
				MarkdownDescription: "Site-level IGMP snooping setting. On UniFi Network 10.3.x+ the effective IGMP snooping toggle lives here rather than on each network. Advanced querier/flood options configured in the UI are preserved across updates.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether IGMP snooping is enabled for the site.",
						Optional:            true,
						Computed:            true,
					},
					"network_ids": schema.ListAttribute{
						MarkdownDescription: "IDs of the networks IGMP snooping applies to.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

// UpgradeState migrates v0 state to v1: radius.interim_update_interval and the
// usg conntrack timeouts changed from integer seconds to GoDuration strings.
func (r *settingResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	conntrack := []string{
		"icmp_timeout", "other_timeout",
		"tcp_close_timeout", "tcp_close_wait_timeout", "tcp_established_timeout",
		"tcp_fin_wait_timeout", "tcp_last_ack_timeout", "tcp_syn_recv_timeout",
		"tcp_syn_sent_timeout", "tcp_time_wait_timeout",
		"udp_other_timeout", "udp_stream_timeout",
	}

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
						if radius, ok := state["radius"].(map[string]any); ok {
							util.SetDurationField(radius, "interim_update_interval", time.Second)
						}
						if usg, ok := state["usg"].(map[string]any); ok {
							for _, n := range conntrack {
								util.SetDurationField(usg, n, time.Second)
							}
						}
					},
				)
				if err != nil {
					resp.Diagnostics.AddError("Failed to upgrade settings state", err.Error())
					return
				}
				resp.DynamicValue = dv
			},
		},
	}
}

func (r *settingResource) Configure(
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

func (r *settingResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data settingResourceModel

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

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Update each configured setting type
	if !data.AutoSpeedtest.IsNull() && !data.AutoSpeedtest.IsUnknown() {
		var as settingAutoSpeedtestModel
		resp.Diagnostics.Append(data.AutoSpeedtest.As(ctx, &as, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.autoSpeedtestModelToSetting(&as)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Auto Speedtest Setting", err.Error())
			return
		}
	}

	if !data.Country.IsNull() && !data.Country.IsUnknown() {
		var m settingCountryModel
		resp.Diagnostics.Append(data.Country.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.countryModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Country Setting", err.Error())
			return
		}
	}

	if !data.Dpi.IsNull() && !data.Dpi.IsUnknown() {
		var m settingDpiModel
		resp.Diagnostics.Append(data.Dpi.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.dpiModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating DPI Setting", err.Error())
			return
		}
	}

	if !data.Lcm.IsNull() && !data.Lcm.IsUnknown() {
		var m settingLcmModel
		resp.Diagnostics.Append(data.Lcm.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.lcmModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating LCM Setting", err.Error())
			return
		}
	}

	if !data.NetworkOpt.IsNull() && !data.NetworkOpt.IsUnknown() {
		var m settingNetworkOptimizationModel
		resp.Diagnostics.Append(data.NetworkOpt.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.networkOptimizationModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Network Optimization Setting", err.Error())
			return
		}
	}

	if !data.Ntp.IsNull() && !data.Ntp.IsUnknown() {
		var m settingNtpModel
		resp.Diagnostics.Append(data.Ntp.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.ntpModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating NTP Setting", err.Error())
			return
		}
	}

	if !data.Syslog.IsNull() && !data.Syslog.IsUnknown() {
		var m settingSyslogModel
		resp.Diagnostics.Append(data.Syslog.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.syslogModelToSetting(ctx, &m, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Syslog Setting", err.Error())
			return
		}
	}

	if !data.Doh.IsNull() && !data.Doh.IsUnknown() {
		var doh settingDohModel
		resp.Diagnostics.Append(data.Doh.As(ctx, &doh, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.dohModelToSetting(ctx, &doh, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating DoH Setting", err.Error())
			return
		}
	}

	if !data.Ips.IsNull() && !data.Ips.IsUnknown() {
		var ips settingIpsModel
		resp.Diagnostics.Append(data.Ips.As(ctx, &ips, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.ipsModelToSetting(ctx, &ips, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating IPS Setting", err.Error())
			return
		}
	}

	if !data.Mgmt.IsNull() && !data.Mgmt.IsUnknown() {
		var mgmt settingMgmtModel
		resp.Diagnostics.Append(data.Mgmt.As(ctx, &mgmt, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Read current remote settings as the base so unset fields keep their values.
		_, currentMgmt, err := ui.GetSetting[*settings.Mgmt](r.client.ApiClient, ctx, site)
		if err != nil {
			var notFound *ui.NotFoundError
			if !errors.As(err, &notFound) {
				resp.Diagnostics.AddError("Error Reading Mgmt Setting", err.Error())
				return
			}
			currentMgmt = &settings.Mgmt{}
		}

		setting := r.mgmtModelToSetting(ctx, &mgmt, currentMgmt)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Mgmt Setting", err.Error())
			return
		}
	}

	if !data.Radius.IsNull() && !data.Radius.IsUnknown() {
		var radius settingRadiusModel
		resp.Diagnostics.Append(data.Radius.As(ctx, &radius, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Read current remote settings as the base so unset fields keep their remote values
		_, currentRadius, err := ui.GetSetting[*settings.Radius](r.client.ApiClient, ctx, site)
		if err != nil {
			var notFound *ui.NotFoundError
			if !errors.As(err, &notFound) {
				resp.Diagnostics.AddError("Error Reading Radius Setting", err.Error())
				return
			}
			currentRadius = &settings.Radius{}
		}

		setting := r.radiusModelToSetting(ctx, &radius, currentRadius)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Radius Setting", err.Error())
			return
		}
	}

	if !data.USG.IsNull() && !data.USG.IsUnknown() {
		var usg settingUSGModel
		resp.Diagnostics.Append(data.USG.As(ctx, &usg, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.usgModelToSetting(ctx, &usg)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating USG Setting", err.Error())
			return
		}
	}

	if !data.IgmpSnooping.IsNull() && !data.IgmpSnooping.IsUnknown() {
		var igmp settingIgmpSnoopingModel
		resp.Diagnostics.Append(data.IgmpSnooping.As(ctx, &igmp, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Read current remote setting as the base so advanced querier/flood
		// fields keep their remote values across the update.
		_, currentIgmp, err := ui.GetSetting[*settings.IgmpSnooping](r.client.ApiClient, ctx, site)
		if err != nil {
			var notFound *ui.NotFoundError
			if !errors.As(err, &notFound) {
				resp.Diagnostics.AddError("Error Reading IGMP Snooping Setting", err.Error())
				return
			}
			currentIgmp = &settings.IgmpSnooping{}
		}

		setting := r.igmpSnoopingModelToSetting(ctx, &igmp, currentIgmp, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating IGMP Snooping Setting", err.Error())
			return
		}
	}

	// Read back the settings
	r.readSettings(ctx, site, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data settingResourceModel

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

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	r.readSettings(ctx, site, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state settingResourceModel
	var plan settingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Update each configured setting type
	if !plan.AutoSpeedtest.IsNull() && !plan.AutoSpeedtest.IsUnknown() {
		var as settingAutoSpeedtestModel
		resp.Diagnostics.Append(plan.AutoSpeedtest.As(ctx, &as, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.autoSpeedtestModelToSetting(&as)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Auto Speedtest Setting", err.Error())
			return
		}
	}

	if !plan.Country.IsNull() && !plan.Country.IsUnknown() {
		var m settingCountryModel
		resp.Diagnostics.Append(plan.Country.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.countryModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Country Setting", err.Error())
			return
		}
	}

	if !plan.Dpi.IsNull() && !plan.Dpi.IsUnknown() {
		var m settingDpiModel
		resp.Diagnostics.Append(plan.Dpi.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.dpiModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating DPI Setting", err.Error())
			return
		}
	}

	if !plan.Lcm.IsNull() && !plan.Lcm.IsUnknown() {
		var m settingLcmModel
		resp.Diagnostics.Append(plan.Lcm.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.lcmModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating LCM Setting", err.Error())
			return
		}
	}

	if !plan.NetworkOpt.IsNull() && !plan.NetworkOpt.IsUnknown() {
		var m settingNetworkOptimizationModel
		resp.Diagnostics.Append(plan.NetworkOpt.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.networkOptimizationModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Network Optimization Setting", err.Error())
			return
		}
	}

	if !plan.Ntp.IsNull() && !plan.Ntp.IsUnknown() {
		var m settingNtpModel
		resp.Diagnostics.Append(plan.Ntp.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.ntpModelToSetting(&m)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating NTP Setting", err.Error())
			return
		}
	}

	if !plan.Syslog.IsNull() && !plan.Syslog.IsUnknown() {
		var m settingSyslogModel
		resp.Diagnostics.Append(plan.Syslog.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		setting := r.syslogModelToSetting(ctx, &m, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Syslog Setting", err.Error())
			return
		}
	}

	if !plan.Doh.IsNull() && !plan.Doh.IsUnknown() {
		var doh settingDohModel
		resp.Diagnostics.Append(plan.Doh.As(ctx, &doh, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.dohModelToSetting(ctx, &doh, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating DoH Setting", err.Error())
			return
		}
	}

	if !plan.Ips.IsNull() && !plan.Ips.IsUnknown() {
		var ips settingIpsModel
		resp.Diagnostics.Append(plan.Ips.As(ctx, &ips, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.ipsModelToSetting(ctx, &ips, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating IPS Setting", err.Error())
			return
		}
	}

	if !plan.Mgmt.IsNull() && !plan.Mgmt.IsUnknown() {
		var mgmt settingMgmtModel
		resp.Diagnostics.Append(plan.Mgmt.As(ctx, &mgmt, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Read current remote settings as the base so unset fields keep their values.
		_, currentMgmt, err := ui.GetSetting[*settings.Mgmt](r.client.ApiClient, ctx, site)
		if err != nil {
			var notFound *ui.NotFoundError
			if !errors.As(err, &notFound) {
				resp.Diagnostics.AddError("Error Reading Mgmt Setting", err.Error())
				return
			}
			currentMgmt = &settings.Mgmt{}
		}

		setting := r.mgmtModelToSetting(ctx, &mgmt, currentMgmt)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Mgmt Setting", err.Error())
			return
		}
	}

	if !plan.Radius.IsNull() && !plan.Radius.IsUnknown() {
		var radius settingRadiusModel
		resp.Diagnostics.Append(plan.Radius.As(ctx, &radius, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Read current remote settings as the base so unset fields keep their remote values
		_, currentRadius, err := ui.GetSetting[*settings.Radius](r.client.ApiClient, ctx, site)
		if err != nil {
			var notFound *ui.NotFoundError
			if !errors.As(err, &notFound) {
				resp.Diagnostics.AddError("Error Reading Radius Setting", err.Error())
				return
			}
			currentRadius = &settings.Radius{}
		}

		setting := r.radiusModelToSetting(ctx, &radius, currentRadius)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Radius Setting", err.Error())
			return
		}
	}

	if !plan.USG.IsNull() && !plan.USG.IsUnknown() {
		var usg settingUSGModel
		resp.Diagnostics.Append(plan.USG.As(ctx, &usg, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.usgModelToSetting(ctx, &usg)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating USG Setting", err.Error())
			return
		}
	}

	if !plan.IgmpSnooping.IsNull() && !plan.IgmpSnooping.IsUnknown() {
		var igmp settingIgmpSnoopingModel
		resp.Diagnostics.Append(plan.IgmpSnooping.As(ctx, &igmp, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		_, currentIgmp, err := ui.GetSetting[*settings.IgmpSnooping](r.client.ApiClient, ctx, site)
		if err != nil {
			var notFound *ui.NotFoundError
			if !errors.As(err, &notFound) {
				resp.Diagnostics.AddError("Error Reading IGMP Snooping Setting", err.Error())
				return
			}
			currentIgmp = &settings.IgmpSnooping{}
		}

		setting := r.igmpSnoopingModelToSetting(ctx, &igmp, currentIgmp, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating IGMP Snooping Setting", err.Error())
			return
		}
	}

	// Read back the settings
	r.readSettings(ctx, site, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *settingResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	// Settings cannot be deleted, only reset to defaults
	// Just remove from state
}

func (r *settingResource) ImportState(
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

func (r *settingResource) readSettings(
	ctx context.Context,
	site string,
	data *settingResourceModel,
	diags *diag.Diagnostics,
) {
	// Set the ID to the site since settings are site-level
	data.ID = types.StringValue(site)
	data.Site = types.StringValue(site)

	// Only read settings that were configured in the plan, set others to null

	// Auto speedtest settings
	if !data.AutoSpeedtest.IsNull() && !data.AutoSpeedtest.IsUnknown() {
		_, asSetting, err := ui.GetSetting[*settings.AutoSpeedtest](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Auto Speedtest Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(
			ctx, autoSpeedtestAttrTypes, r.autoSpeedtestSettingToModel(asSetting),
		)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.AutoSpeedtest = objValue
	} else {
		data.AutoSpeedtest = types.ObjectNull(autoSpeedtestAttrTypes)
	}

	// Country settings
	if !data.Country.IsNull() && !data.Country.IsUnknown() {
		_, s, err := ui.GetSetting[*settings.Country](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Country Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(ctx, countryAttrTypes, r.countrySettingToModel(s))
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Country = objValue
	} else {
		data.Country = types.ObjectNull(countryAttrTypes)
	}

	// DPI settings
	if !data.Dpi.IsNull() && !data.Dpi.IsUnknown() {
		_, s, err := ui.GetSetting[*settings.Dpi](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading DPI Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(ctx, dpiAttrTypes, r.dpiSettingToModel(s))
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Dpi = objValue
	} else {
		data.Dpi = types.ObjectNull(dpiAttrTypes)
	}

	// LCM settings
	if !data.Lcm.IsNull() && !data.Lcm.IsUnknown() {
		_, s, err := ui.GetSetting[*settings.Lcm](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading LCM Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(ctx, lcmAttrTypes, r.lcmSettingToModel(s))
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Lcm = objValue
	} else {
		data.Lcm = types.ObjectNull(lcmAttrTypes)
	}

	// Network optimization settings
	if !data.NetworkOpt.IsNull() && !data.NetworkOpt.IsUnknown() {
		_, s, err := ui.GetSetting[*settings.NetworkOptimization](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Network Optimization Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(
			ctx, networkOptimizationAttrTypes, r.networkOptimizationSettingToModel(s),
		)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.NetworkOpt = objValue
	} else {
		data.NetworkOpt = types.ObjectNull(networkOptimizationAttrTypes)
	}

	// NTP settings
	if !data.Ntp.IsNull() && !data.Ntp.IsUnknown() {
		_, s, err := ui.GetSetting[*settings.Ntp](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading NTP Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(ctx, ntpAttrTypes, r.ntpSettingToModel(s))
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Ntp = objValue
	} else {
		data.Ntp = types.ObjectNull(ntpAttrTypes)
	}

	// Syslog settings
	if !data.Syslog.IsNull() && !data.Syslog.IsUnknown() {
		_, s, err := ui.GetSetting[*settings.Rsyslogd](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Syslog Setting", err.Error())
			return
		}
		objValue, d := types.ObjectValueFrom(
			ctx, syslogAttrTypes, r.syslogSettingToModel(ctx, s, diags),
		)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Syslog = objValue
	} else {
		data.Syslog = types.ObjectNull(syslogAttrTypes)
	}

	// DoH settings
	if !data.Doh.IsNull() && !data.Doh.IsUnknown() {
		var planDoh settingDohModel
		diags.Append(data.Doh.As(ctx, &planDoh, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, dohSetting, err := ui.GetSetting[*settings.Doh](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading DoH Setting", err.Error())
			return
		}

		dohModel := r.dohSettingToModel(ctx, dohSetting, &planDoh, diags)
		objValue, d := types.ObjectValueFrom(ctx, dohAttrTypes, dohModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Doh = objValue
	} else {
		data.Doh = types.ObjectNull(dohAttrTypes)
	}

	// IPS settings
	if !data.Ips.IsNull() && !data.Ips.IsUnknown() {
		var planIps settingIpsModel
		diags.Append(data.Ips.As(ctx, &planIps, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, ipsSetting, err := ui.GetSetting[*settings.Ips](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading IPS Setting", err.Error())
			return
		}

		ipsModel := r.ipsSettingToModel(ctx, ipsSetting, &planIps, diags)
		objValue, d := types.ObjectValueFrom(ctx, ipsAttrTypes, ipsModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Ips = objValue
	} else {
		data.Ips = types.ObjectNull(ipsAttrTypes)
	}

	// Mgmt settings
	if !data.Mgmt.IsNull() && !data.Mgmt.IsUnknown() {
		// Get the current plan/state values
		var planMgmt settingMgmtModel
		diags.Append(data.Mgmt.As(ctx, &planMgmt, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, mgmtSetting, err := ui.GetSetting[*settings.Mgmt](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Mgmt Setting", err.Error())
			return
		}

		mgmtModel := r.mgmtSettingToModel(ctx, mgmtSetting, &planMgmt)
		objValue, d := types.ObjectValueFrom(ctx, mgmtAttrTypes, mgmtModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Mgmt = objValue
	} else {
		data.Mgmt = types.ObjectNull(mgmtAttrTypes)
	}

	// Radius settings
	if !data.Radius.IsNull() && !data.Radius.IsUnknown() {
		// Get the current plan/state values
		var planRadius settingRadiusModel
		diags.Append(data.Radius.As(ctx, &planRadius, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, radiusSetting, err := ui.GetSetting[*settings.Radius](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Radius Setting", err.Error())
			return
		}

		radiusModel := r.radiusSettingToModel(ctx, radiusSetting, &planRadius)
		objValue, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"accounting_enabled":      types.BoolType,
			"acct_port":               types.Int64Type,
			"auth_port":               types.Int64Type,
			"interim_update_interval": timetypes.GoDurationType{},
			"secret":                  types.StringType,
		}, radiusModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Radius = objValue
	} else {
		data.Radius = types.ObjectNull(map[string]attr.Type{
			"accounting_enabled":      types.BoolType,
			"acct_port":               types.Int64Type,
			"auth_port":               types.Int64Type,
			"interim_update_interval": timetypes.GoDurationType{},
			"secret":                  types.StringType,
		})
	}

	// USG settings
	if !data.USG.IsNull() && !data.USG.IsUnknown() {
		// Get the current plan/state values
		var planUSG settingUSGModel
		diags.Append(data.USG.As(ctx, &planUSG, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, usgSetting, err := ui.GetSetting[*settings.Usg](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading USG Setting", err.Error())
			return
		}

		usgModel := r.usgSettingToModel(ctx, usgSetting, &planUSG)
		objValue, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"broadcast_ping": types.BoolType,
			"dns_verification": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"domain":               types.StringType,
					"primary_dns_server":   types.StringType,
					"secondary_dns_server": types.StringType,
					"setting_preference":   types.StringType,
				},
			},
			"ftp_module":                         types.BoolType,
			"geo_ip_filtering_block":             types.StringType,
			"geo_ip_filtering_countries":         types.StringType,
			"geo_ip_filtering_enabled":           types.BoolType,
			"geo_ip_filtering_traffic_direction": types.StringType,
			"gre_module":                         types.BoolType,
			"h323_module":                        types.BoolType,
			"icmp_timeout":                       timetypes.GoDurationType{},
			"mss_clamp":                          types.StringType,
			"offload_accounting":                 types.BoolType,
			"offload_l2_blocking":                types.BoolType,
			"offload_sch":                        types.BoolType,
			"other_timeout":                      timetypes.GoDurationType{},
			"pptp_module":                        types.BoolType,
			"receive_redirects":                  types.BoolType,
			"send_redirects":                     types.BoolType,
			"sip_module":                         types.BoolType,
			"syn_cookies":                        types.BoolType,
			"tcp_close_timeout":                  timetypes.GoDurationType{},
			"tcp_close_wait_timeout":             timetypes.GoDurationType{},
			"tcp_established_timeout":            timetypes.GoDurationType{},
			"tcp_fin_wait_timeout":               timetypes.GoDurationType{},
			"tcp_last_ack_timeout":               timetypes.GoDurationType{},
			"tcp_syn_recv_timeout":               timetypes.GoDurationType{},
			"tcp_syn_sent_timeout":               timetypes.GoDurationType{},
			"tcp_time_wait_timeout":              timetypes.GoDurationType{},
			"tftp_module":                        types.BoolType,
			"timeout_setting_preference":         types.StringType,
			"udp_other_timeout":                  timetypes.GoDurationType{},
			"udp_stream_timeout":                 timetypes.GoDurationType{},
			"unbind_wan_monitors":                types.BoolType,
			"upnp_enabled":                       types.BoolType,
			"upnp_nat_pmp_enabled":               types.BoolType,
			"upnp_secure_mode":                   types.BoolType,
			"upnp_wan_interface":                 types.StringType,
		}, usgModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.USG = objValue
	} else {
		data.USG = types.ObjectNull(map[string]attr.Type{
			"broadcast_ping": types.BoolType,
			"dns_verification": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"domain":               types.StringType,
					"primary_dns_server":   types.StringType,
					"secondary_dns_server": types.StringType,
					"setting_preference":   types.StringType,
				},
			},
			"ftp_module":                         types.BoolType,
			"geo_ip_filtering_block":             types.StringType,
			"geo_ip_filtering_countries":         types.StringType,
			"geo_ip_filtering_enabled":           types.BoolType,
			"geo_ip_filtering_traffic_direction": types.StringType,
			"gre_module":                         types.BoolType,
			"h323_module":                        types.BoolType,
			"icmp_timeout":                       timetypes.GoDurationType{},
			"mss_clamp":                          types.StringType,
			"offload_accounting":                 types.BoolType,
			"offload_l2_blocking":                types.BoolType,
			"offload_sch":                        types.BoolType,
			"other_timeout":                      timetypes.GoDurationType{},
			"pptp_module":                        types.BoolType,
			"receive_redirects":                  types.BoolType,
			"send_redirects":                     types.BoolType,
			"sip_module":                         types.BoolType,
			"syn_cookies":                        types.BoolType,
			"tcp_close_timeout":                  timetypes.GoDurationType{},
			"tcp_close_wait_timeout":             timetypes.GoDurationType{},
			"tcp_established_timeout":            timetypes.GoDurationType{},
			"tcp_fin_wait_timeout":               timetypes.GoDurationType{},
			"tcp_last_ack_timeout":               timetypes.GoDurationType{},
			"tcp_syn_recv_timeout":               timetypes.GoDurationType{},
			"tcp_syn_sent_timeout":               timetypes.GoDurationType{},
			"tcp_time_wait_timeout":              timetypes.GoDurationType{},
			"tftp_module":                        types.BoolType,
			"timeout_setting_preference":         types.StringType,
			"udp_other_timeout":                  timetypes.GoDurationType{},
			"udp_stream_timeout":                 timetypes.GoDurationType{},
			"unbind_wan_monitors":                types.BoolType,
			"upnp_enabled":                       types.BoolType,
			"upnp_nat_pmp_enabled":               types.BoolType,
			"upnp_secure_mode":                   types.BoolType,
			"upnp_wan_interface":                 types.StringType,
		})
	}

	// IGMP snooping (site-level)
	if !data.IgmpSnooping.IsNull() && !data.IgmpSnooping.IsUnknown() {
		_, igmpSetting, err := ui.GetSetting[*settings.IgmpSnooping](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading IGMP Snooping Setting", err.Error())
			return
		}
		igmpModel := r.igmpSnoopingSettingToModel(ctx, igmpSetting, diags)
		objValue, d := types.ObjectValueFrom(ctx, igmpSnoopingAttrTypes, igmpModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.IgmpSnooping = objValue
	} else {
		data.IgmpSnooping = types.ObjectNull(igmpSnoopingAttrTypes)
	}
}

// Mgmt conversion functions.
func (r *settingResource) mgmtModelToSetting(
	ctx context.Context,
	model *settingMgmtModel,
	base *settings.Mgmt,
) *settings.Mgmt {
	setting := base

	if !model.AutoUpgrade.IsNull() && !model.AutoUpgrade.IsUnknown() {
		setting.AutoUpgrade = model.AutoUpgrade.ValueBool()
	}
	if !model.AutoUpgradeHour.IsNull() && !model.AutoUpgradeHour.IsUnknown() {
		setting.AutoUpgradeHour = model.AutoUpgradeHour.ValueInt64Pointer()
	}
	if !model.SSHEnabled.IsNull() && !model.SSHEnabled.IsUnknown() {
		setting.SSHEnabled = model.SSHEnabled.ValueBool()
	}
	if !model.AdvancedFeatureEnabled.IsNull() && !model.AdvancedFeatureEnabled.IsUnknown() {
		setting.AdvancedFeatureEnabled = model.AdvancedFeatureEnabled.ValueBool()
	}
	if !model.DebugToolsEnabled.IsNull() && !model.DebugToolsEnabled.IsUnknown() {
		setting.DebugToolsEnabled = model.DebugToolsEnabled.ValueBool()
	}
	if !model.DirectConnectEnabled.IsNull() && !model.DirectConnectEnabled.IsUnknown() {
		setting.DirectConnectEnabled = model.DirectConnectEnabled.ValueBool()
	}
	if !model.UnifiIdpEnabled.IsNull() && !model.UnifiIdpEnabled.IsUnknown() {
		setting.UniFiIdentityProviderEnabled = model.UnifiIdpEnabled.ValueBool()
	}
	if !model.WifimanEnabled.IsNull() && !model.WifimanEnabled.IsUnknown() {
		setting.WifimanEnabled = model.WifimanEnabled.ValueBool()
	}
	if !model.SSHUsername.IsNull() && !model.SSHUsername.IsUnknown() {
		setting.SSHUsername = model.SSHUsername.ValueString()
	}
	if !model.SSHPassword.IsNull() && !model.SSHPassword.IsUnknown() {
		setting.SSHPassword = model.SSHPassword.ValueString()
	}
	if !model.SSHAuthPasswordEnabled.IsNull() && !model.SSHAuthPasswordEnabled.IsUnknown() {
		setting.SSHAuthPasswordEnabled = model.SSHAuthPasswordEnabled.ValueBool()
	}

	if !model.SSHKeys.IsNull() && !model.SSHKeys.IsUnknown() {
		setting.SSHKeys = nil
		var sshKeys []sshKeyModel
		model.SSHKeys.ElementsAs(ctx, &sshKeys, false)
		for _, sshKey := range sshKeys {
			setting.SSHKeys = append(setting.SSHKeys, settings.SettingMgmtSSHKeys{
				Name:    sshKey.Name.ValueString(),
				KeyType: sshKey.Type.ValueString(),
				Key:     sshKey.Key.ValueString(),
				Comment: sshKey.Comment.ValueString(),
			})
		}
	}

	return setting
}

func (r *settingResource) mgmtSettingToModel(
	ctx context.Context,
	setting *settings.Mgmt,
	plan *settingMgmtModel,
) *settingMgmtModel {
	model := &settingMgmtModel{}

	// Only populate fields that were explicitly configured in the plan, so the
	// resource doesn't report drift on settings the user doesn't manage.
	boolOrNull := func(planVal types.Bool, apiVal bool) types.Bool {
		if !planVal.IsNull() && !planVal.IsUnknown() {
			return types.BoolValue(apiVal)
		}
		return types.BoolNull()
	}

	model.AutoUpgrade = boolOrNull(plan.AutoUpgrade, setting.AutoUpgrade)
	model.SSHEnabled = boolOrNull(plan.SSHEnabled, setting.SSHEnabled)
	model.AdvancedFeatureEnabled = boolOrNull(
		plan.AdvancedFeatureEnabled, setting.AdvancedFeatureEnabled,
	)
	model.DebugToolsEnabled = boolOrNull(plan.DebugToolsEnabled, setting.DebugToolsEnabled)
	model.DirectConnectEnabled = boolOrNull(plan.DirectConnectEnabled, setting.DirectConnectEnabled)
	model.UnifiIdpEnabled = boolOrNull(plan.UnifiIdpEnabled, setting.UniFiIdentityProviderEnabled)
	model.WifimanEnabled = boolOrNull(plan.WifimanEnabled, setting.WifimanEnabled)
	model.SSHAuthPasswordEnabled = boolOrNull(
		plan.SSHAuthPasswordEnabled, setting.SSHAuthPasswordEnabled,
	)

	if !plan.AutoUpgradeHour.IsNull() && !plan.AutoUpgradeHour.IsUnknown() {
		model.AutoUpgradeHour = types.Int64PointerValue(setting.AutoUpgradeHour)
	} else {
		model.AutoUpgradeHour = types.Int64Null()
	}

	if !plan.SSHUsername.IsNull() && !plan.SSHUsername.IsUnknown() {
		model.SSHUsername = util.StringValueOrNull(setting.SSHUsername)
	} else {
		model.SSHUsername = types.StringNull()
	}

	// The controller never returns the plaintext SSH password (only hashes), so
	// preserve the configured value to avoid a perpetual diff.
	model.SSHPassword = plan.SSHPassword

	if !plan.SSHKeys.IsNull() && !plan.SSHKeys.IsUnknown() {
		if len(setting.SSHKeys) > 0 {
			var sshKeys []sshKeyModel
			for _, sshKey := range setting.SSHKeys {
				sshKeys = append(sshKeys, sshKeyModel{
					Name:    types.StringValue(sshKey.Name),
					Type:    types.StringValue(sshKey.KeyType),
					Key:     types.StringValue(sshKey.Key),
					Comment: types.StringValue(sshKey.Comment),
				})
			}
			listValue, _ := types.ListValueFrom(
				ctx, types.ObjectType{AttrTypes: mgmtSSHKeyAttrTypes}, sshKeys,
			)
			model.SSHKeys = listValue
		} else {
			model.SSHKeys = types.ListNull(types.ObjectType{AttrTypes: mgmtSSHKeyAttrTypes})
		}
	} else {
		model.SSHKeys = types.ListNull(types.ObjectType{AttrTypes: mgmtSSHKeyAttrTypes})
	}

	return model
}

// Radius conversion functions.
func (r *settingResource) radiusModelToSetting(
	_ context.Context,
	model *settingRadiusModel,
	base *settings.Radius,
) *settings.Radius {
	setting := base

	if !model.AccountingEnabled.IsNull() && !model.AccountingEnabled.IsUnknown() {
		setting.AccountingEnabled = model.AccountingEnabled.ValueBool()
	}
	if !model.AcctPort.IsNull() && !model.AcctPort.IsUnknown() {
		setting.AcctPort = model.AcctPort.ValueInt64Pointer()
	}
	if !model.AuthPort.IsNull() && !model.AuthPort.IsUnknown() {
		setting.AuthPort = model.AuthPort.ValueInt64Pointer()
	}
	if !model.InterimUpdateInterval.IsNull() && !model.InterimUpdateInterval.IsUnknown() {
		setting.InterimUpdateInterval = util.DurationUnitsPtr(
			model.InterimUpdateInterval,
			time.Second,
		)
	}
	if !model.Secret.IsNull() && !model.Secret.IsUnknown() {
		setting.Secret = model.Secret.ValueString()
	}

	return setting
}

func (r *settingResource) radiusSettingToModel(
	_ context.Context,
	setting *settings.Radius,
	plan *settingRadiusModel,
) *settingRadiusModel {
	model := &settingRadiusModel{}

	// Only populate fields that were explicitly configured in the plan
	model.AccountingEnabled = types.BoolValue(setting.AccountingEnabled)

	model.AcctPort = types.Int64PointerValue(setting.AcctPort)

	model.AuthPort = types.Int64PointerValue(setting.AuthPort)

	model.InterimUpdateInterval = util.DurationPtrValue(setting.InterimUpdateInterval, time.Second)

	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		if setting.Secret != "" {
			model.Secret = types.StringValue(setting.Secret)
		} else {
			model.Secret = types.StringNull()
		}
	} else {
		model.Secret = types.StringNull()
	}

	return model
}

// USG conversion functions.
func (r *settingResource) usgModelToSetting(
	ctx context.Context,
	model *settingUSGModel,
) *settings.Usg {
	setting := &settings.Usg{}

	if !model.BroadcastPing.IsNull() {
		setting.BroadcastPing = model.BroadcastPing.ValueBool()
	}
	if !model.DNSVerification.IsNull() && !model.DNSVerification.IsUnknown() {
		var dnsVerif dnsVerificationModel
		model.DNSVerification.As(ctx, &dnsVerif, basetypes.ObjectAsOptions{})
		setting.DNSVerification = &settings.SettingUsgDNSVerification{
			Domain:             dnsVerif.Domain.ValueString(),
			PrimaryDNSServer:   dnsVerif.PrimaryDNSServer.ValueString(),
			SecondaryDNSServer: dnsVerif.SecondaryDNSServer.ValueString(),
			SettingPreference:  dnsVerif.SettingPreference.ValueString(),
		}
	}
	if !model.FtpModule.IsNull() {
		setting.FtpModule = model.FtpModule.ValueBool()
	}
	if !model.GeoIPFilteringBlock.IsNull() {
		setting.GeoIPFilteringBlock = model.GeoIPFilteringBlock.ValueString()
	}
	if !model.GeoIPFilteringCountries.IsNull() {
		setting.GeoIPFilteringCountries = model.GeoIPFilteringCountries.ValueString()
	}
	if !model.GeoIPFilteringEnabled.IsNull() {
		setting.GeoIPFilteringEnabled = model.GeoIPFilteringEnabled.ValueBool()
	}
	if !model.GeoIPFilteringTrafficDirection.IsNull() {
		setting.GeoIPFilteringTrafficDirection = model.GeoIPFilteringTrafficDirection.ValueString()
	}
	if !model.GreModule.IsNull() {
		setting.GreModule = model.GreModule.ValueBool()
	}
	if !model.H323Module.IsNull() {
		setting.H323Module = model.H323Module.ValueBool()
	}
	if !model.ICMPTimeout.IsNull() && !model.ICMPTimeout.IsUnknown() {
		setting.ICMPTimeout = util.DurationUnits(model.ICMPTimeout, time.Second)
	}
	if !model.MssClamp.IsNull() {
		setting.MssClamp = model.MssClamp.ValueString()
	}
	if !model.OffloadAccounting.IsNull() {
		setting.OffloadAccounting = model.OffloadAccounting.ValueBool()
	}
	if !model.OffloadL2Blocking.IsNull() {
		setting.OffloadL2Blocking = model.OffloadL2Blocking.ValueBool()
	}
	if !model.OffloadSch.IsNull() {
		setting.OffloadSch = model.OffloadSch.ValueBool()
	}
	if !model.OtherTimeout.IsNull() && !model.OtherTimeout.IsUnknown() {
		setting.OtherTimeout = util.DurationUnits(model.OtherTimeout, time.Second)
	}
	if !model.PptpModule.IsNull() {
		setting.PptpModule = model.PptpModule.ValueBool()
	}
	if !model.ReceiveRedirects.IsNull() {
		setting.ReceiveRedirects = model.ReceiveRedirects.ValueBool()
	}
	if !model.SendRedirects.IsNull() {
		setting.SendRedirects = model.SendRedirects.ValueBool()
	}
	if !model.SipModule.IsNull() {
		setting.SipModule = model.SipModule.ValueBool()
	}
	if !model.SynCookies.IsNull() {
		setting.SynCookies = model.SynCookies.ValueBool()
	}
	if !model.TCPCloseTimeout.IsNull() && !model.TCPCloseTimeout.IsUnknown() {
		setting.TCPCloseTimeout = util.DurationUnits(model.TCPCloseTimeout, time.Second)
	}
	if !model.TCPCloseWaitTimeout.IsNull() && !model.TCPCloseWaitTimeout.IsUnknown() {
		setting.TCPCloseWaitTimeout = util.DurationUnits(model.TCPCloseWaitTimeout, time.Second)
	}
	if !model.TCPEstablishedTimeout.IsNull() && !model.TCPEstablishedTimeout.IsUnknown() {
		setting.TCPEstablishedTimeout = util.DurationUnits(model.TCPEstablishedTimeout, time.Second)
	}
	if !model.TCPFinWaitTimeout.IsNull() && !model.TCPFinWaitTimeout.IsUnknown() {
		setting.TCPFinWaitTimeout = util.DurationUnits(model.TCPFinWaitTimeout, time.Second)
	}
	if !model.TCPLastAckTimeout.IsNull() && !model.TCPLastAckTimeout.IsUnknown() {
		setting.TCPLastAckTimeout = util.DurationUnits(model.TCPLastAckTimeout, time.Second)
	}
	if !model.TCPSynRecvTimeout.IsNull() && !model.TCPSynRecvTimeout.IsUnknown() {
		setting.TCPSynRecvTimeout = util.DurationUnits(model.TCPSynRecvTimeout, time.Second)
	}
	if !model.TCPSynSentTimeout.IsNull() && !model.TCPSynSentTimeout.IsUnknown() {
		setting.TCPSynSentTimeout = util.DurationUnits(model.TCPSynSentTimeout, time.Second)
	}
	if !model.TCPTimeWaitTimeout.IsNull() && !model.TCPTimeWaitTimeout.IsUnknown() {
		setting.TCPTimeWaitTimeout = util.DurationUnits(model.TCPTimeWaitTimeout, time.Second)
	}
	if !model.TFTPModule.IsNull() {
		setting.TFTPModule = model.TFTPModule.ValueBool()
	}
	if !model.TimeoutSettingPreference.IsNull() {
		setting.TimeoutSettingPreference = model.TimeoutSettingPreference.ValueString()
	}
	if !model.UDPOtherTimeout.IsNull() && !model.UDPOtherTimeout.IsUnknown() {
		setting.UDPOtherTimeout = util.DurationUnits(model.UDPOtherTimeout, time.Second)
	}
	if !model.UDPStreamTimeout.IsNull() && !model.UDPStreamTimeout.IsUnknown() {
		setting.UDPStreamTimeout = util.DurationUnits(model.UDPStreamTimeout, time.Second)
	}
	if !model.UnbindWANMonitors.IsNull() {
		setting.UnbindWANMonitors = model.UnbindWANMonitors.ValueBool()
	}
	if !model.UPnPEnabled.IsNull() {
		setting.UPnPEnabled = model.UPnPEnabled.ValueBool()
	}
	if !model.UPnPNATPmpEnabled.IsNull() {
		setting.UPnPNATPmpEnabled = model.UPnPNATPmpEnabled.ValueBool()
	}
	if !model.UPnPSecureMode.IsNull() {
		setting.UPnPSecureMode = model.UPnPSecureMode.ValueBool()
	}
	if !model.UPnPWANInterface.IsNull() {
		setting.UPnPWANInterface = model.UPnPWANInterface.ValueString()
	}

	return setting
}

func (r *settingResource) usgSettingToModel(
	ctx context.Context,
	setting *settings.Usg,
	plan *settingUSGModel,
) *settingUSGModel {
	model := &settingUSGModel{}

	// Only populate fields that were explicitly configured in the plan
	if !plan.BroadcastPing.IsNull() && !plan.BroadcastPing.IsUnknown() {
		model.BroadcastPing = types.BoolValue(setting.BroadcastPing)
	} else {
		model.BroadcastPing = types.BoolNull()
	}

	if !plan.DNSVerification.IsNull() && !plan.DNSVerification.IsUnknown() {
		dnsVerif := dnsVerificationModel{
			Domain:             types.StringValue(setting.DNSVerification.Domain),
			PrimaryDNSServer:   types.StringValue(setting.DNSVerification.PrimaryDNSServer),
			SecondaryDNSServer: types.StringValue(setting.DNSVerification.SecondaryDNSServer),
			SettingPreference:  types.StringValue(setting.DNSVerification.SettingPreference),
		}
		objValue, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"domain":               types.StringType,
			"primary_dns_server":   types.StringType,
			"secondary_dns_server": types.StringType,
			"setting_preference":   types.StringType,
		}, dnsVerif)
		model.DNSVerification = objValue
	} else {
		model.DNSVerification = types.ObjectNull(map[string]attr.Type{
			"domain":               types.StringType,
			"primary_dns_server":   types.StringType,
			"secondary_dns_server": types.StringType,
			"setting_preference":   types.StringType,
		})
	}

	if !plan.FtpModule.IsNull() && !plan.FtpModule.IsUnknown() {
		model.FtpModule = types.BoolValue(setting.FtpModule)
	} else {
		model.FtpModule = types.BoolNull()
	}

	if !plan.GeoIPFilteringBlock.IsNull() && !plan.GeoIPFilteringBlock.IsUnknown() {
		if setting.GeoIPFilteringBlock != "" {
			model.GeoIPFilteringBlock = types.StringValue(setting.GeoIPFilteringBlock)
		} else {
			model.GeoIPFilteringBlock = types.StringNull()
		}
	} else {
		model.GeoIPFilteringBlock = types.StringNull()
	}

	if !plan.GeoIPFilteringCountries.IsNull() && !plan.GeoIPFilteringCountries.IsUnknown() {
		if setting.GeoIPFilteringCountries != "" {
			model.GeoIPFilteringCountries = types.StringValue(setting.GeoIPFilteringCountries)
		} else {
			model.GeoIPFilteringCountries = types.StringNull()
		}
	} else {
		model.GeoIPFilteringCountries = types.StringNull()
	}

	if !plan.GeoIPFilteringEnabled.IsNull() && !plan.GeoIPFilteringEnabled.IsUnknown() {
		model.GeoIPFilteringEnabled = types.BoolValue(setting.GeoIPFilteringEnabled)
	} else {
		model.GeoIPFilteringEnabled = types.BoolNull()
	}

	if !plan.GeoIPFilteringTrafficDirection.IsNull() &&
		!plan.GeoIPFilteringTrafficDirection.IsUnknown() {
		if setting.GeoIPFilteringTrafficDirection != "" {
			model.GeoIPFilteringTrafficDirection = types.StringValue(
				setting.GeoIPFilteringTrafficDirection,
			)
		} else {
			model.GeoIPFilteringTrafficDirection = types.StringNull()
		}
	} else {
		model.GeoIPFilteringTrafficDirection = types.StringNull()
	}

	if !plan.GreModule.IsNull() && !plan.GreModule.IsUnknown() {
		model.GreModule = types.BoolValue(setting.GreModule)
	} else {
		model.GreModule = types.BoolNull()
	}

	if !plan.H323Module.IsNull() && !plan.H323Module.IsUnknown() {
		model.H323Module = types.BoolValue(setting.H323Module)
	} else {
		model.H323Module = types.BoolNull()
	}

	if !plan.ICMPTimeout.IsNull() && !plan.ICMPTimeout.IsUnknown() {
		model.ICMPTimeout = util.DurationValue(setting.ICMPTimeout, time.Second)
	} else {
		model.ICMPTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.MssClamp.IsNull() && !plan.MssClamp.IsUnknown() {
		if setting.MssClamp != "" {
			model.MssClamp = types.StringValue(setting.MssClamp)
		} else {
			model.MssClamp = types.StringNull()
		}
	} else {
		model.MssClamp = types.StringNull()
	}

	if !plan.OffloadAccounting.IsNull() && !plan.OffloadAccounting.IsUnknown() {
		model.OffloadAccounting = types.BoolValue(setting.OffloadAccounting)
	} else {
		model.OffloadAccounting = types.BoolNull()
	}

	if !plan.OffloadL2Blocking.IsNull() && !plan.OffloadL2Blocking.IsUnknown() {
		model.OffloadL2Blocking = types.BoolValue(setting.OffloadL2Blocking)
	} else {
		model.OffloadL2Blocking = types.BoolNull()
	}

	if !plan.OffloadSch.IsNull() && !plan.OffloadSch.IsUnknown() {
		model.OffloadSch = types.BoolValue(setting.OffloadSch)
	} else {
		model.OffloadSch = types.BoolNull()
	}

	if !plan.OtherTimeout.IsNull() && !plan.OtherTimeout.IsUnknown() {
		model.OtherTimeout = util.DurationValue(setting.OtherTimeout, time.Second)
	} else {
		model.OtherTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.PptpModule.IsNull() && !plan.PptpModule.IsUnknown() {
		model.PptpModule = types.BoolValue(setting.PptpModule)
	} else {
		model.PptpModule = types.BoolNull()
	}

	if !plan.ReceiveRedirects.IsNull() && !plan.ReceiveRedirects.IsUnknown() {
		model.ReceiveRedirects = types.BoolValue(setting.ReceiveRedirects)
	} else {
		model.ReceiveRedirects = types.BoolNull()
	}

	if !plan.SendRedirects.IsNull() && !plan.SendRedirects.IsUnknown() {
		model.SendRedirects = types.BoolValue(setting.SendRedirects)
	} else {
		model.SendRedirects = types.BoolNull()
	}

	if !plan.SipModule.IsNull() && !plan.SipModule.IsUnknown() {
		model.SipModule = types.BoolValue(setting.SipModule)
	} else {
		model.SipModule = types.BoolNull()
	}

	if !plan.SynCookies.IsNull() && !plan.SynCookies.IsUnknown() {
		model.SynCookies = types.BoolValue(setting.SynCookies)
	} else {
		model.SynCookies = types.BoolNull()
	}

	if !plan.TCPCloseTimeout.IsNull() && !plan.TCPCloseTimeout.IsUnknown() {
		model.TCPCloseTimeout = util.DurationValue(setting.TCPCloseTimeout, time.Second)
	} else {
		model.TCPCloseTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPCloseWaitTimeout.IsNull() && !plan.TCPCloseWaitTimeout.IsUnknown() {
		model.TCPCloseWaitTimeout = util.DurationValue(setting.TCPCloseWaitTimeout, time.Second)
	} else {
		model.TCPCloseWaitTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPEstablishedTimeout.IsNull() && !plan.TCPEstablishedTimeout.IsUnknown() {
		model.TCPEstablishedTimeout = util.DurationValue(setting.TCPEstablishedTimeout, time.Second)
	} else {
		model.TCPEstablishedTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPFinWaitTimeout.IsNull() && !plan.TCPFinWaitTimeout.IsUnknown() {
		model.TCPFinWaitTimeout = util.DurationValue(setting.TCPFinWaitTimeout, time.Second)
	} else {
		model.TCPFinWaitTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPLastAckTimeout.IsNull() && !plan.TCPLastAckTimeout.IsUnknown() {
		model.TCPLastAckTimeout = util.DurationValue(setting.TCPLastAckTimeout, time.Second)
	} else {
		model.TCPLastAckTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPSynRecvTimeout.IsNull() && !plan.TCPSynRecvTimeout.IsUnknown() {
		model.TCPSynRecvTimeout = util.DurationValue(setting.TCPSynRecvTimeout, time.Second)
	} else {
		model.TCPSynRecvTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPSynSentTimeout.IsNull() && !plan.TCPSynSentTimeout.IsUnknown() {
		model.TCPSynSentTimeout = util.DurationValue(setting.TCPSynSentTimeout, time.Second)
	} else {
		model.TCPSynSentTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TCPTimeWaitTimeout.IsNull() && !plan.TCPTimeWaitTimeout.IsUnknown() {
		model.TCPTimeWaitTimeout = util.DurationValue(setting.TCPTimeWaitTimeout, time.Second)
	} else {
		model.TCPTimeWaitTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.TFTPModule.IsNull() && !plan.TFTPModule.IsUnknown() {
		model.TFTPModule = types.BoolValue(setting.TFTPModule)
	} else {
		model.TFTPModule = types.BoolNull()
	}

	if !plan.TimeoutSettingPreference.IsNull() && !plan.TimeoutSettingPreference.IsUnknown() {
		if setting.TimeoutSettingPreference != "" {
			model.TimeoutSettingPreference = types.StringValue(setting.TimeoutSettingPreference)
		} else {
			model.TimeoutSettingPreference = types.StringNull()
		}
	} else {
		model.TimeoutSettingPreference = types.StringNull()
	}

	if !plan.UDPOtherTimeout.IsNull() && !plan.UDPOtherTimeout.IsUnknown() {
		model.UDPOtherTimeout = util.DurationValue(setting.UDPOtherTimeout, time.Second)
	} else {
		model.UDPOtherTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.UDPStreamTimeout.IsNull() && !plan.UDPStreamTimeout.IsUnknown() {
		model.UDPStreamTimeout = util.DurationValue(setting.UDPStreamTimeout, time.Second)
	} else {
		model.UDPStreamTimeout = timetypes.NewGoDurationNull()
	}

	if !plan.UnbindWANMonitors.IsNull() && !plan.UnbindWANMonitors.IsUnknown() {
		model.UnbindWANMonitors = types.BoolValue(setting.UnbindWANMonitors)
	} else {
		model.UnbindWANMonitors = types.BoolNull()
	}

	if !plan.UPnPEnabled.IsNull() && !plan.UPnPEnabled.IsUnknown() {
		model.UPnPEnabled = types.BoolValue(setting.UPnPEnabled)
	} else {
		model.UPnPEnabled = types.BoolNull()
	}

	if !plan.UPnPNATPmpEnabled.IsNull() && !plan.UPnPNATPmpEnabled.IsUnknown() {
		model.UPnPNATPmpEnabled = types.BoolValue(setting.UPnPNATPmpEnabled)
	} else {
		model.UPnPNATPmpEnabled = types.BoolNull()
	}

	if !plan.UPnPSecureMode.IsNull() && !plan.UPnPSecureMode.IsUnknown() {
		model.UPnPSecureMode = types.BoolValue(setting.UPnPSecureMode)
	} else {
		model.UPnPSecureMode = types.BoolNull()
	}

	if !plan.UPnPWANInterface.IsNull() && !plan.UPnPWANInterface.IsUnknown() {
		if setting.UPnPWANInterface != "" {
			model.UPnPWANInterface = types.StringValue(setting.UPnPWANInterface)
		} else {
			model.UPnPWANInterface = types.StringNull()
		}
	} else {
		model.UPnPWANInterface = types.StringNull()
	}

	return model
}

// IGMP snooping conversion functions.

// igmpSnoopingModelToSetting overlays the user-set fields (enabled, network_ids)
// onto the current remote setting (base) so advanced querier/flood options are
// preserved across updates.
func (r *settingResource) igmpSnoopingModelToSetting(
	ctx context.Context,
	model *settingIgmpSnoopingModel,
	base *settings.IgmpSnooping,
	diags *diag.Diagnostics,
) *settings.IgmpSnooping {
	setting := base
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		setting.Enabled = model.Enabled.ValueBool()
	}
	if !model.NetworkIDs.IsNull() && !model.NetworkIDs.IsUnknown() {
		var ids []string
		diags.Append(model.NetworkIDs.ElementsAs(ctx, &ids, false)...)
		setting.NetworkIDs = ids
	}
	return setting
}

func (r *settingResource) igmpSnoopingSettingToModel(
	ctx context.Context,
	setting *settings.IgmpSnooping,
	diags *diag.Diagnostics,
) *settingIgmpSnoopingModel {
	model := &settingIgmpSnoopingModel{
		Enabled: types.BoolValue(setting.Enabled),
	}
	ids, d := types.ListValueFrom(ctx, types.StringType, setting.NetworkIDs)
	diags.Append(d...)
	model.NetworkIDs = ids
	return model
}

// DoH conversion functions.
func (r *settingResource) autoSpeedtestModelToSetting(
	model *settingAutoSpeedtestModel,
) *settings.AutoSpeedtest {
	setting := &settings.AutoSpeedtest{}
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		setting.Enabled = model.Enabled.ValueBool()
	}
	if !model.CronExpr.IsNull() && !model.CronExpr.IsUnknown() {
		setting.CronExpr = model.CronExpr.ValueString()
	}
	return setting
}

func (r *settingResource) autoSpeedtestSettingToModel(
	setting *settings.AutoSpeedtest,
) settingAutoSpeedtestModel {
	return settingAutoSpeedtestModel{
		Enabled:  types.BoolValue(setting.Enabled),
		CronExpr: util.StringValueOrNull(setting.CronExpr),
	}
}

func (r *settingResource) countryModelToSetting(m *settingCountryModel) *settings.Country {
	return &settings.Country{Code: m.Code.ValueInt64Pointer()}
}

func (r *settingResource) countrySettingToModel(s *settings.Country) settingCountryModel {
	return settingCountryModel{Code: types.Int64PointerValue(s.Code)}
}

func (r *settingResource) dpiModelToSetting(m *settingDpiModel) *settings.Dpi {
	return &settings.Dpi{
		Enabled:               m.Enabled.ValueBool(),
		FingerprintingEnabled: m.FingerprintingEnabled.ValueBool(),
	}
}

func (r *settingResource) dpiSettingToModel(s *settings.Dpi) settingDpiModel {
	return settingDpiModel{
		Enabled:               types.BoolValue(s.Enabled),
		FingerprintingEnabled: types.BoolValue(s.FingerprintingEnabled),
	}
}

func (r *settingResource) lcmModelToSetting(m *settingLcmModel) *settings.Lcm {
	setting := &settings.Lcm{
		Enabled:    m.Enabled.ValueBool(),
		Sync:       m.Sync.ValueBool(),
		TouchEvent: m.TouchEvent.ValueBool(),
	}
	// Guard the optional ints: an unknown (unset Optional+Computed) value yields a
	// 0 pointer, which the controller rejects as out of range (cf. #288/#303).
	if !m.Brightness.IsNull() && !m.Brightness.IsUnknown() {
		setting.Brightness = m.Brightness.ValueInt64Pointer()
	}
	if !m.IdleTimeout.IsNull() && !m.IdleTimeout.IsUnknown() {
		setting.IDleTimeout = m.IdleTimeout.ValueInt64Pointer()
	}
	return setting
}

func (r *settingResource) lcmSettingToModel(s *settings.Lcm) settingLcmModel {
	return settingLcmModel{
		Enabled:     types.BoolValue(s.Enabled),
		Brightness:  types.Int64PointerValue(s.Brightness),
		IdleTimeout: types.Int64PointerValue(s.IDleTimeout),
		Sync:        types.BoolValue(s.Sync),
		TouchEvent:  types.BoolValue(s.TouchEvent),
	}
}

func (r *settingResource) networkOptimizationModelToSetting(
	m *settingNetworkOptimizationModel,
) *settings.NetworkOptimization {
	return &settings.NetworkOptimization{Enabled: m.Enabled.ValueBool()}
}

func (r *settingResource) networkOptimizationSettingToModel(
	s *settings.NetworkOptimization,
) settingNetworkOptimizationModel {
	return settingNetworkOptimizationModel{Enabled: types.BoolValue(s.Enabled)}
}

func (r *settingResource) ntpModelToSetting(m *settingNtpModel) *settings.Ntp {
	return &settings.Ntp{
		NtpServer1:        m.NtpServer1.ValueString(),
		NtpServer2:        m.NtpServer2.ValueString(),
		NtpServer3:        m.NtpServer3.ValueString(),
		NtpServer4:        m.NtpServer4.ValueString(),
		SettingPreference: m.SettingPreference.ValueString(),
	}
}

func (r *settingResource) ntpSettingToModel(s *settings.Ntp) settingNtpModel {
	return settingNtpModel{
		NtpServer1:        util.StringValueOrNull(s.NtpServer1),
		NtpServer2:        util.StringValueOrNull(s.NtpServer2),
		NtpServer3:        util.StringValueOrNull(s.NtpServer3),
		NtpServer4:        util.StringValueOrNull(s.NtpServer4),
		SettingPreference: util.StringValueOrNull(s.SettingPreference),
	}
}

func (r *settingResource) syslogModelToSetting(
	ctx context.Context,
	m *settingSyslogModel,
	diags *diag.Diagnostics,
) *settings.Rsyslogd {
	setting := &settings.Rsyslogd{
		Enabled:                     m.Enabled.ValueBool(),
		Debug:                       m.Debug.ValueBool(),
		IP:                          m.IP.ValueString(),
		LogAllContents:              m.LogAllContents.ValueBool(),
		NetconsoleEnabled:           m.NetconsoleEnabled.ValueBool(),
		NetconsoleHost:              m.NetconsoleHost.ValueString(),
		ThisController:              m.ThisController.ValueBool(),
		ThisControllerEncryptedOnly: m.ThisControllerEncryptedOnly.ValueBool(),
	}
	// Guard the optional ports: an unknown (unset Optional+Computed) value yields a
	// 0 pointer, which the controller rejects as an out-of-range port (#303, cf. #288).
	if !m.Port.IsNull() && !m.Port.IsUnknown() {
		setting.Port = m.Port.ValueInt64Pointer()
	}
	if !m.NetconsolePort.IsNull() && !m.NetconsolePort.IsUnknown() {
		setting.NetconsolePort = m.NetconsolePort.ValueInt64Pointer()
	}
	if !m.Contents.IsNull() && !m.Contents.IsUnknown() {
		diags.Append(m.Contents.ElementsAs(ctx, &setting.Contents, false)...)
	}
	return setting
}

func (r *settingResource) syslogSettingToModel(
	ctx context.Context,
	s *settings.Rsyslogd,
	diags *diag.Diagnostics,
) settingSyslogModel {
	contents, d := types.ListValueFrom(ctx, types.StringType, s.Contents)
	diags.Append(d...)
	return settingSyslogModel{
		Enabled:                     types.BoolValue(s.Enabled),
		Contents:                    contents,
		Debug:                       types.BoolValue(s.Debug),
		IP:                          util.StringValueOrNull(s.IP),
		Port:                        types.Int64PointerValue(s.Port),
		LogAllContents:              types.BoolValue(s.LogAllContents),
		NetconsoleEnabled:           types.BoolValue(s.NetconsoleEnabled),
		NetconsoleHost:              util.StringValueOrNull(s.NetconsoleHost),
		NetconsolePort:              types.Int64PointerValue(s.NetconsolePort),
		ThisController:              types.BoolValue(s.ThisController),
		ThisControllerEncryptedOnly: types.BoolValue(s.ThisControllerEncryptedOnly),
	}
}

func (r *settingResource) dohModelToSetting(
	ctx context.Context,
	model *settingDohModel,
	diags *diag.Diagnostics,
) *settings.Doh {
	setting := &settings.Doh{}

	if !model.State.IsNull() && !model.State.IsUnknown() {
		setting.State = model.State.ValueString()
	}
	if !model.ServerNames.IsNull() && !model.ServerNames.IsUnknown() {
		diags.Append(model.ServerNames.ElementsAs(ctx, &setting.ServerNames, false)...)
		if diags.HasError() {
			return setting
		}
	}
	if !model.CustomServers.IsNull() && !model.CustomServers.IsUnknown() {
		var servers []settingDohCustomServerModel
		diags.Append(model.CustomServers.ElementsAs(ctx, &servers, false)...)
		if diags.HasError() {
			return setting
		}
		for _, s := range servers {
			enabled := true
			if !s.Enabled.IsNull() && !s.Enabled.IsUnknown() {
				enabled = s.Enabled.ValueBool()
			}
			setting.CustomServers = append(setting.CustomServers, settings.SettingDohCustomServers{
				Enabled:    enabled,
				SdnsStamp:  s.SDNSStamp.ValueString(),
				ServerName: s.ServerName.ValueString(),
			})
		}
	}

	return setting
}

func (r *settingResource) dohSettingToModel(
	ctx context.Context,
	setting *settings.Doh,
	plan *settingDohModel,
	diags *diag.Diagnostics,
) *settingDohModel {
	model := &settingDohModel{}

	if !plan.State.IsNull() && !plan.State.IsUnknown() {
		if setting.State != "" {
			model.State = types.StringValue(setting.State)
		} else {
			model.State = types.StringNull()
		}
	} else {
		model.State = types.StringNull()
	}

	// When the attribute was configured (plan is known), mirror the remote
	// value as a list — including an empty list when the controller returns
	// none. Returning ListNull for a configured-but-empty list would differ
	// from the planned []value and trip the "inconsistent result after apply"
	// check. ListNull is reserved for the not-configured / unknown case.
	if !plan.ServerNames.IsNull() && !plan.ServerNames.IsUnknown() {
		listVal, d := types.ListValueFrom(ctx, types.StringType, setting.ServerNames)
		diags.Append(d...)
		model.ServerNames = listVal
	} else {
		model.ServerNames = types.ListNull(types.StringType)
	}

	customServersType := types.ObjectType{AttrTypes: dohCustomServerAttrTypes}
	if !plan.CustomServers.IsNull() && !plan.CustomServers.IsUnknown() {
		servers := make([]settingDohCustomServerModel, 0, len(setting.CustomServers))
		for _, s := range setting.CustomServers {
			servers = append(servers, settingDohCustomServerModel{
				Enabled:    types.BoolValue(s.Enabled),
				SDNSStamp:  types.StringValue(s.SdnsStamp),
				ServerName: types.StringValue(s.ServerName),
			})
		}
		listVal, d := types.ListValueFrom(ctx, customServersType, servers)
		diags.Append(d...)
		model.CustomServers = listVal
	} else {
		model.CustomServers = types.ListNull(customServersType)
	}

	return model
}

// IPS conversion functions.
func (r *settingResource) ipsModelToSetting(
	ctx context.Context,
	model *settingIpsModel,
	diags *diag.Diagnostics,
) *settings.Ips {
	setting := &settings.Ips{}

	if !model.IPSMode.IsNull() && !model.IPSMode.IsUnknown() {
		setting.IPsMode = model.IPSMode.ValueString()
	}
	if !model.HoneypotEnabled.IsNull() && !model.HoneypotEnabled.IsUnknown() {
		setting.HoneypotEnabled = model.HoneypotEnabled.ValueBool()
	}
	if !model.RestrictTorrents.IsNull() && !model.RestrictTorrents.IsUnknown() {
		setting.RestrictTorrents = model.RestrictTorrents.ValueBool()
	}
	if !model.ContentFilteringBlockingPageEnabled.IsNull() &&
		!model.ContentFilteringBlockingPageEnabled.IsUnknown() {
		setting.ContentFilteringBlockingPageEnabled = model.ContentFilteringBlockingPageEnabled.ValueBool()
	}
	if !model.MemoryOptimized.IsNull() && !model.MemoryOptimized.IsUnknown() {
		setting.MemoryOptimized = model.MemoryOptimized.ValueBool()
	}
	if !model.AdvancedFilteringPreference.IsNull() &&
		!model.AdvancedFilteringPreference.IsUnknown() {
		setting.AdvancedFilteringPreference = model.AdvancedFilteringPreference.ValueString()
	}
	if !model.EnabledCategories.IsNull() && !model.EnabledCategories.IsUnknown() {
		diags.Append(model.EnabledCategories.ElementsAs(ctx, &setting.EnabledCategories, false)...)
		if diags.HasError() {
			return setting
		}
	}
	if !model.EnabledNetworks.IsNull() && !model.EnabledNetworks.IsUnknown() {
		diags.Append(model.EnabledNetworks.ElementsAs(ctx, &setting.EnabledNetworks, false)...)
		if diags.HasError() {
			return setting
		}
	}
	if !model.Honeypot.IsNull() && !model.Honeypot.IsUnknown() {
		var honeypots []settingIpsHoneypotModel
		diags.Append(model.Honeypot.ElementsAs(ctx, &honeypots, false)...)
		if diags.HasError() {
			return setting
		}
		for _, h := range honeypots {
			setting.Honeypot = append(setting.Honeypot, settings.SettingIpsHoneypot{
				IPAddress: h.IPAddress.ValueString(),
				NetworkID: h.NetworkID.ValueString(),
				Version:   h.Version.ValueString(),
			})
		}
	}
	if !model.SuppressionWhitelist.IsNull() && !model.SuppressionWhitelist.IsUnknown() {
		var whitelist []settingIpsWhitelistModel
		diags.Append(model.SuppressionWhitelist.ElementsAs(ctx, &whitelist, false)...)
		if diags.HasError() {
			return setting
		}
		if setting.Suppression == nil {
			setting.Suppression = &settings.SettingIpsSuppression{}
		}
		for _, w := range whitelist {
			setting.Suppression.Whitelist = append(
				setting.Suppression.Whitelist,
				settings.SettingIpsWhitelist{
					Direction: w.Direction.ValueString(),
					Mode:      w.Mode.ValueString(),
					Value:     w.Value.ValueString(),
				},
			)
		}
	}
	if !model.SuppressionAlerts.IsNull() && !model.SuppressionAlerts.IsUnknown() {
		var alerts []settingIpsAlertModel
		diags.Append(model.SuppressionAlerts.ElementsAs(ctx, &alerts, false)...)
		if diags.HasError() {
			return setting
		}
		if setting.Suppression == nil {
			setting.Suppression = &settings.SettingIpsSuppression{}
		}
		for _, a := range alerts {
			alert := settings.SettingIpsAlerts{
				Category:  a.Category.ValueString(),
				Signature: a.Signature.ValueString(),
				Type:      a.Type.ValueString(),
			}
			// Omit gid/id when unset rather than sending 0 (cf. #303).
			if !a.Gid.IsNull() && !a.Gid.IsUnknown() {
				alert.Gid = a.Gid.ValueInt64Pointer()
			}
			if !a.ID.IsNull() && !a.ID.IsUnknown() {
				alert.ID = a.ID.ValueInt64Pointer()
			}
			if !a.Tracking.IsNull() && !a.Tracking.IsUnknown() {
				var tracking []settingIpsTrackingModel
				diags.Append(a.Tracking.ElementsAs(ctx, &tracking, false)...)
				for _, t := range tracking {
					alert.Tracking = append(alert.Tracking, settings.SettingIpsTracking{
						Direction: t.Direction.ValueString(),
						Mode:      t.Mode.ValueString(),
						Value:     t.Value.ValueString(),
					})
				}
			}
			setting.Suppression.Alerts = append(setting.Suppression.Alerts, alert)
		}
	}

	return setting
}

func (r *settingResource) ipsSettingToModel(
	ctx context.Context,
	setting *settings.Ips,
	plan *settingIpsModel,
	diags *diag.Diagnostics,
) *settingIpsModel {
	model := &settingIpsModel{}

	if !plan.IPSMode.IsNull() && !plan.IPSMode.IsUnknown() {
		if setting.IPsMode != "" {
			model.IPSMode = types.StringValue(setting.IPsMode)
		} else {
			model.IPSMode = types.StringNull()
		}
	} else {
		model.IPSMode = types.StringNull()
	}

	if !plan.HoneypotEnabled.IsNull() && !plan.HoneypotEnabled.IsUnknown() {
		model.HoneypotEnabled = types.BoolValue(setting.HoneypotEnabled)
	} else {
		model.HoneypotEnabled = types.BoolNull()
	}

	if !plan.RestrictTorrents.IsNull() && !plan.RestrictTorrents.IsUnknown() {
		model.RestrictTorrents = types.BoolValue(setting.RestrictTorrents)
	} else {
		model.RestrictTorrents = types.BoolNull()
	}

	if !plan.ContentFilteringBlockingPageEnabled.IsNull() &&
		!plan.ContentFilteringBlockingPageEnabled.IsUnknown() {
		model.ContentFilteringBlockingPageEnabled = types.BoolValue(
			setting.ContentFilteringBlockingPageEnabled,
		)
	} else {
		model.ContentFilteringBlockingPageEnabled = types.BoolNull()
	}

	if !plan.MemoryOptimized.IsNull() && !plan.MemoryOptimized.IsUnknown() {
		model.MemoryOptimized = types.BoolValue(setting.MemoryOptimized)
	} else {
		model.MemoryOptimized = types.BoolNull()
	}

	if !plan.AdvancedFilteringPreference.IsNull() && !plan.AdvancedFilteringPreference.IsUnknown() {
		if setting.AdvancedFilteringPreference != "" {
			model.AdvancedFilteringPreference = types.StringValue(
				setting.AdvancedFilteringPreference,
			)
		} else {
			model.AdvancedFilteringPreference = types.StringNull()
		}
	} else {
		model.AdvancedFilteringPreference = types.StringNull()
	}

	// Configured lists mirror the remote value (empty list included); ListNull
	// is reserved for the not-configured / unknown case. See dohSettingToModel
	// for why a configured-but-empty list must not become ListNull.
	if !plan.EnabledCategories.IsNull() && !plan.EnabledCategories.IsUnknown() {
		listVal, d := types.ListValueFrom(ctx, types.StringType, setting.EnabledCategories)
		diags.Append(d...)
		model.EnabledCategories = listVal
	} else {
		model.EnabledCategories = types.ListNull(types.StringType)
	}

	if !plan.EnabledNetworks.IsNull() && !plan.EnabledNetworks.IsUnknown() {
		listVal, d := types.ListValueFrom(ctx, types.StringType, setting.EnabledNetworks)
		diags.Append(d...)
		model.EnabledNetworks = listVal
	} else {
		model.EnabledNetworks = types.ListNull(types.StringType)
	}

	honeypotType := types.ObjectType{AttrTypes: ipsHoneypotAttrTypes}
	if !plan.Honeypot.IsNull() && !plan.Honeypot.IsUnknown() {
		honeypots := make([]settingIpsHoneypotModel, 0, len(setting.Honeypot))
		for _, h := range setting.Honeypot {
			honeypots = append(honeypots, settingIpsHoneypotModel{
				IPAddress: types.StringValue(h.IPAddress),
				NetworkID: types.StringValue(h.NetworkID),
				Version:   types.StringValue(h.Version),
			})
		}
		listVal, d := types.ListValueFrom(ctx, honeypotType, honeypots)
		diags.Append(d...)
		model.Honeypot = listVal
	} else {
		model.Honeypot = types.ListNull(honeypotType)
	}

	whitelistType := types.ObjectType{AttrTypes: ipsWhitelistAttrTypes}
	if !plan.SuppressionWhitelist.IsNull() && !plan.SuppressionWhitelist.IsUnknown() {
		var whitelist []settings.SettingIpsWhitelist
		if setting.Suppression != nil {
			whitelist = setting.Suppression.Whitelist
		}
		entries := make([]settingIpsWhitelistModel, 0, len(whitelist))
		for _, w := range whitelist {
			entries = append(entries, settingIpsWhitelistModel{
				Direction: types.StringValue(w.Direction),
				Mode:      types.StringValue(w.Mode),
				Value:     types.StringValue(w.Value),
			})
		}
		listVal, d := types.ListValueFrom(ctx, whitelistType, entries)
		diags.Append(d...)
		model.SuppressionWhitelist = listVal
	} else {
		model.SuppressionWhitelist = types.ListNull(whitelistType)
	}

	trackingType := types.ObjectType{AttrTypes: ipsTrackingAttrTypes}
	alertType := types.ObjectType{AttrTypes: ipsAlertAttrTypes}
	if !plan.SuppressionAlerts.IsNull() && !plan.SuppressionAlerts.IsUnknown() {
		var alerts []settings.SettingIpsAlerts
		if setting.Suppression != nil {
			alerts = setting.Suppression.Alerts
		}
		entries := make([]settingIpsAlertModel, 0, len(alerts))
		for _, a := range alerts {
			tracking := make([]settingIpsTrackingModel, 0, len(a.Tracking))
			for _, t := range a.Tracking {
				tracking = append(tracking, settingIpsTrackingModel{
					Direction: types.StringValue(t.Direction),
					Mode:      types.StringValue(t.Mode),
					Value:     types.StringValue(t.Value),
				})
			}
			trackingList, d := types.ListValueFrom(ctx, trackingType, tracking)
			diags.Append(d...)
			entries = append(entries, settingIpsAlertModel{
				Category:  util.StringValueOrNull(a.Category),
				Gid:       types.Int64PointerValue(a.Gid),
				ID:        types.Int64PointerValue(a.ID),
				Signature: util.StringValueOrNull(a.Signature),
				Type:      util.StringValueOrNull(a.Type),
				Tracking:  trackingList,
			})
		}
		listVal, d := types.ListValueFrom(ctx, alertType, entries)
		diags.Append(d...)
		model.SuppressionAlerts = listVal
	} else {
		model.SuppressionAlerts = types.ListNull(alertType)
	}

	return model
}
