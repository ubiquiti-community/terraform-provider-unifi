package unifi

import (
	"context"
	"errors"
	"fmt"
	"regexp"

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
)

var (
	_ resource.Resource                = &settingResource{}
	_ resource.ResourceWithImportState = &settingResource{}
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
	AutoUpgrade types.Bool `tfsdk:"auto_upgrade"`
	SSHEnabled  types.Bool `tfsdk:"ssh_enabled"`
	SSHKeys     types.List `tfsdk:"ssh_keys"`
}

type settingRadiusModel struct {
	AccountingEnabled     types.Bool   `tfsdk:"accounting_enabled"`
	AcctPort              types.Int64  `tfsdk:"acct_port"`
	AuthPort              types.Int64  `tfsdk:"auth_port"`
	InterimUpdateInterval types.Int64  `tfsdk:"interim_update_interval"`
	Secret                types.String `tfsdk:"secret"`
}

type dnsVerificationModel struct {
	Domain             types.String `tfsdk:"domain"`
	PrimaryDNSServer   types.String `tfsdk:"primary_dns_server"`
	SecondaryDNSServer types.String `tfsdk:"secondary_dns_server"`
	SettingPreference  types.String `tfsdk:"setting_preference"`
}

type settingUSGModel struct {
	BroadcastPing                  types.Bool   `tfsdk:"broadcast_ping"`
	DNSVerification                types.Object `tfsdk:"dns_verification"`
	FtpModule                      types.Bool   `tfsdk:"ftp_module"`
	GeoIPFilteringBlock            types.String `tfsdk:"geo_ip_filtering_block"`
	GeoIPFilteringCountries        types.String `tfsdk:"geo_ip_filtering_countries"`
	GeoIPFilteringEnabled          types.Bool   `tfsdk:"geo_ip_filtering_enabled"`
	GeoIPFilteringTrafficDirection types.String `tfsdk:"geo_ip_filtering_traffic_direction"`
	GreModule                      types.Bool   `tfsdk:"gre_module"`
	H323Module                     types.Bool   `tfsdk:"h323_module"`
	ICMPTimeout                    types.Int64  `tfsdk:"icmp_timeout"`
	MssClamp                       types.String `tfsdk:"mss_clamp"`
	OffloadAccounting              types.Bool   `tfsdk:"offload_accounting"`
	OffloadL2Blocking              types.Bool   `tfsdk:"offload_l2_blocking"`
	OffloadSch                     types.Bool   `tfsdk:"offload_sch"`
	OtherTimeout                   types.Int64  `tfsdk:"other_timeout"`
	PptpModule                     types.Bool   `tfsdk:"pptp_module"`
	ReceiveRedirects               types.Bool   `tfsdk:"receive_redirects"`
	SendRedirects                  types.Bool   `tfsdk:"send_redirects"`
	SipModule                      types.Bool   `tfsdk:"sip_module"`
	SynCookies                     types.Bool   `tfsdk:"syn_cookies"`
	TCPCloseTimeout                types.Int64  `tfsdk:"tcp_close_timeout"`
	TCPCloseWaitTimeout            types.Int64  `tfsdk:"tcp_close_wait_timeout"`
	TCPEstablishedTimeout          types.Int64  `tfsdk:"tcp_established_timeout"`
	TCPFinWaitTimeout              types.Int64  `tfsdk:"tcp_fin_wait_timeout"`
	TCPLastAckTimeout              types.Int64  `tfsdk:"tcp_last_ack_timeout"`
	TCPSynRecvTimeout              types.Int64  `tfsdk:"tcp_syn_recv_timeout"`
	TCPSynSentTimeout              types.Int64  `tfsdk:"tcp_syn_sent_timeout"`
	TCPTimeWaitTimeout             types.Int64  `tfsdk:"tcp_time_wait_timeout"`
	TFTPModule                     types.Bool   `tfsdk:"tftp_module"`
	TimeoutSettingPreference       types.String `tfsdk:"timeout_setting_preference"`
	UDPOtherTimeout                types.Int64  `tfsdk:"udp_other_timeout"`
	UDPStreamTimeout               types.Int64  `tfsdk:"udp_stream_timeout"`
	UnbindWANMonitors              types.Bool   `tfsdk:"unbind_wan_monitors"`
	UPnPEnabled                    types.Bool   `tfsdk:"upnp_enabled"`
	UPnPNATPmpEnabled              types.Bool   `tfsdk:"upnp_nat_pmp_enabled"`
	UPnPSecureMode                 types.Bool   `tfsdk:"upnp_secure_mode"`
	UPnPWANInterface               types.String `tfsdk:"upnp_wan_interface"`
}

type settingDohCustomServerModel struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	SDNSStamp  types.String `tfsdk:"sdns_stamp"`
	ServerName types.String `tfsdk:"server_name"`
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
}

type settingResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Site         types.String `tfsdk:"site"`
	Doh          types.Object `tfsdk:"doh"`
	Ips          types.Object `tfsdk:"ips"`
	Mgmt         types.Object `tfsdk:"mgmt"`
	Radius       types.Object `tfsdk:"radius"`
	USG          types.Object `tfsdk:"usg"`
	IgmpSnooping types.Object `tfsdk:"igmp_snooping"`
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
					"interim_update_interval": schema.Int64Attribute{
						MarkdownDescription: "Interim update interval in seconds.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.Int64{
							int64validator.Between(60, 86400),
						},
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
					"icmp_timeout": schema.Int64Attribute{
						MarkdownDescription: "ICMP connection timeout in seconds.",
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
					"other_timeout": schema.Int64Attribute{
						MarkdownDescription: "Other connections timeout in seconds.",
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
					"tcp_close_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP close timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_close_wait_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP close wait timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_established_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP established connection timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_fin_wait_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP fin wait timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_last_ack_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP last ACK timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_syn_recv_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP SYN received timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_syn_sent_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP SYN sent timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_time_wait_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP time wait timeout in seconds.",
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
					"udp_other_timeout": schema.Int64Attribute{
						MarkdownDescription: "UDP other timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"udp_stream_timeout": schema.Int64Attribute{
						MarkdownDescription: "UDP stream timeout in seconds.",
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

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Update each configured setting type
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

		setting := r.mgmtModelToSetting(ctx, &mgmt)
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

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Update each configured setting type
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

		setting := r.mgmtModelToSetting(ctx, &mgmt)
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
		objValue, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"auto_upgrade": types.BoolType,
			"ssh_enabled":  types.BoolType,
			"ssh_keys": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":    types.StringType,
						"type":    types.StringType,
						"key":     types.StringType,
						"comment": types.StringType,
					},
				},
			},
		}, mgmtModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Mgmt = objValue
	} else {
		data.Mgmt = types.ObjectNull(map[string]attr.Type{
			"auto_upgrade": types.BoolType,
			"ssh_enabled":  types.BoolType,
			"ssh_keys": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":    types.StringType,
						"type":    types.StringType,
						"key":     types.StringType,
						"comment": types.StringType,
					},
				},
			},
		})
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
			"interim_update_interval": types.Int64Type,
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
			"interim_update_interval": types.Int64Type,
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
			"icmp_timeout":                       types.Int64Type,
			"mss_clamp":                          types.StringType,
			"offload_accounting":                 types.BoolType,
			"offload_l2_blocking":                types.BoolType,
			"offload_sch":                        types.BoolType,
			"other_timeout":                      types.Int64Type,
			"pptp_module":                        types.BoolType,
			"receive_redirects":                  types.BoolType,
			"send_redirects":                     types.BoolType,
			"sip_module":                         types.BoolType,
			"syn_cookies":                        types.BoolType,
			"tcp_close_timeout":                  types.Int64Type,
			"tcp_close_wait_timeout":             types.Int64Type,
			"tcp_established_timeout":            types.Int64Type,
			"tcp_fin_wait_timeout":               types.Int64Type,
			"tcp_last_ack_timeout":               types.Int64Type,
			"tcp_syn_recv_timeout":               types.Int64Type,
			"tcp_syn_sent_timeout":               types.Int64Type,
			"tcp_time_wait_timeout":              types.Int64Type,
			"tftp_module":                        types.BoolType,
			"timeout_setting_preference":         types.StringType,
			"udp_other_timeout":                  types.Int64Type,
			"udp_stream_timeout":                 types.Int64Type,
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
			"icmp_timeout":                       types.Int64Type,
			"mss_clamp":                          types.StringType,
			"offload_accounting":                 types.BoolType,
			"offload_l2_blocking":                types.BoolType,
			"offload_sch":                        types.BoolType,
			"other_timeout":                      types.Int64Type,
			"pptp_module":                        types.BoolType,
			"receive_redirects":                  types.BoolType,
			"send_redirects":                     types.BoolType,
			"sip_module":                         types.BoolType,
			"syn_cookies":                        types.BoolType,
			"tcp_close_timeout":                  types.Int64Type,
			"tcp_close_wait_timeout":             types.Int64Type,
			"tcp_established_timeout":            types.Int64Type,
			"tcp_fin_wait_timeout":               types.Int64Type,
			"tcp_last_ack_timeout":               types.Int64Type,
			"tcp_syn_recv_timeout":               types.Int64Type,
			"tcp_syn_sent_timeout":               types.Int64Type,
			"tcp_time_wait_timeout":              types.Int64Type,
			"tftp_module":                        types.BoolType,
			"timeout_setting_preference":         types.StringType,
			"udp_other_timeout":                  types.Int64Type,
			"udp_stream_timeout":                 types.Int64Type,
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
) *settings.Mgmt {
	setting := &settings.Mgmt{}

	if !model.AutoUpgrade.IsNull() {
		setting.AutoUpgrade = model.AutoUpgrade.ValueBool()
	}
	if !model.SSHEnabled.IsNull() {
		setting.SSHEnabled = model.SSHEnabled.ValueBool()
	}

	if !model.SSHKeys.IsNull() && !model.SSHKeys.IsUnknown() {
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

	// Only populate fields that were explicitly configured in the plan
	if !plan.AutoUpgrade.IsNull() && !plan.AutoUpgrade.IsUnknown() {
		model.AutoUpgrade = types.BoolValue(setting.AutoUpgrade)
	} else {
		model.AutoUpgrade = types.BoolNull()
	}

	if !plan.SSHEnabled.IsNull() && !plan.SSHEnabled.IsUnknown() {
		model.SSHEnabled = types.BoolValue(setting.SSHEnabled)
	} else {
		model.SSHEnabled = types.BoolNull()
	}

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
			listValue, _ := types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":    types.StringType,
					"type":    types.StringType,
					"key":     types.StringType,
					"comment": types.StringType,
				},
			}, sshKeys)
			model.SSHKeys = listValue
		} else {
			model.SSHKeys = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":    types.StringType,
					"type":    types.StringType,
					"key":     types.StringType,
					"comment": types.StringType,
				},
			})
		}
	} else {
		model.SSHKeys = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":    types.StringType,
				"type":    types.StringType,
				"key":     types.StringType,
				"comment": types.StringType,
			},
		})
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
		setting.InterimUpdateInterval = model.InterimUpdateInterval.ValueInt64Pointer()
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

	model.InterimUpdateInterval = types.Int64PointerValue(setting.InterimUpdateInterval)

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
	if !model.ICMPTimeout.IsNull() {
		setting.ICMPTimeout = model.ICMPTimeout.ValueInt64()
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
	if !model.OtherTimeout.IsNull() {
		setting.OtherTimeout = model.OtherTimeout.ValueInt64()
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
	if !model.TCPCloseTimeout.IsNull() {
		setting.TCPCloseTimeout = model.TCPCloseTimeout.ValueInt64()
	}
	if !model.TCPCloseWaitTimeout.IsNull() {
		setting.TCPCloseWaitTimeout = model.TCPCloseWaitTimeout.ValueInt64()
	}
	if !model.TCPEstablishedTimeout.IsNull() {
		setting.TCPEstablishedTimeout = model.TCPEstablishedTimeout.ValueInt64()
	}
	if !model.TCPFinWaitTimeout.IsNull() {
		setting.TCPFinWaitTimeout = model.TCPFinWaitTimeout.ValueInt64()
	}
	if !model.TCPLastAckTimeout.IsNull() {
		setting.TCPLastAckTimeout = model.TCPLastAckTimeout.ValueInt64()
	}
	if !model.TCPSynRecvTimeout.IsNull() {
		setting.TCPSynRecvTimeout = model.TCPSynRecvTimeout.ValueInt64()
	}
	if !model.TCPSynSentTimeout.IsNull() {
		setting.TCPSynSentTimeout = model.TCPSynSentTimeout.ValueInt64()
	}
	if !model.TCPTimeWaitTimeout.IsNull() {
		setting.TCPTimeWaitTimeout = model.TCPTimeWaitTimeout.ValueInt64()
	}
	if !model.TFTPModule.IsNull() {
		setting.TFTPModule = model.TFTPModule.ValueBool()
	}
	if !model.TimeoutSettingPreference.IsNull() {
		setting.TimeoutSettingPreference = model.TimeoutSettingPreference.ValueString()
	}
	if !model.UDPOtherTimeout.IsNull() {
		setting.UDPOtherTimeout = model.UDPOtherTimeout.ValueInt64()
	}
	if !model.UDPStreamTimeout.IsNull() {
		setting.UDPStreamTimeout = model.UDPStreamTimeout.ValueInt64()
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
		model.ICMPTimeout = types.Int64Value(setting.ICMPTimeout)
	} else {
		model.ICMPTimeout = types.Int64Null()
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
		model.OtherTimeout = types.Int64Value(setting.OtherTimeout)
	} else {
		model.OtherTimeout = types.Int64Null()
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
		model.TCPCloseTimeout = types.Int64Value(setting.TCPCloseTimeout)
	} else {
		model.TCPCloseTimeout = types.Int64Null()
	}

	if !plan.TCPCloseWaitTimeout.IsNull() && !plan.TCPCloseWaitTimeout.IsUnknown() {
		model.TCPCloseWaitTimeout = types.Int64Value(setting.TCPCloseWaitTimeout)
	} else {
		model.TCPCloseWaitTimeout = types.Int64Null()
	}

	if !plan.TCPEstablishedTimeout.IsNull() && !plan.TCPEstablishedTimeout.IsUnknown() {
		model.TCPEstablishedTimeout = types.Int64Value(setting.TCPEstablishedTimeout)
	} else {
		model.TCPEstablishedTimeout = types.Int64Null()
	}

	if !plan.TCPFinWaitTimeout.IsNull() && !plan.TCPFinWaitTimeout.IsUnknown() {
		model.TCPFinWaitTimeout = types.Int64Value(setting.TCPFinWaitTimeout)
	} else {
		model.TCPFinWaitTimeout = types.Int64Null()
	}

	if !plan.TCPLastAckTimeout.IsNull() && !plan.TCPLastAckTimeout.IsUnknown() {
		model.TCPLastAckTimeout = types.Int64Value(setting.TCPLastAckTimeout)
	} else {
		model.TCPLastAckTimeout = types.Int64Null()
	}

	if !plan.TCPSynRecvTimeout.IsNull() && !plan.TCPSynRecvTimeout.IsUnknown() {
		model.TCPSynRecvTimeout = types.Int64Value(setting.TCPSynRecvTimeout)
	} else {
		model.TCPSynRecvTimeout = types.Int64Null()
	}

	if !plan.TCPSynSentTimeout.IsNull() && !plan.TCPSynSentTimeout.IsUnknown() {
		model.TCPSynSentTimeout = types.Int64Value(setting.TCPSynSentTimeout)
	} else {
		model.TCPSynSentTimeout = types.Int64Null()
	}

	if !plan.TCPTimeWaitTimeout.IsNull() && !plan.TCPTimeWaitTimeout.IsUnknown() {
		model.TCPTimeWaitTimeout = types.Int64Value(setting.TCPTimeWaitTimeout)
	} else {
		model.TCPTimeWaitTimeout = types.Int64Null()
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
		model.UDPOtherTimeout = types.Int64Value(setting.UDPOtherTimeout)
	} else {
		model.UDPOtherTimeout = types.Int64Null()
	}

	if !plan.UDPStreamTimeout.IsNull() && !plan.UDPStreamTimeout.IsUnknown() {
		model.UDPStreamTimeout = types.Int64Value(setting.UDPStreamTimeout)
	} else {
		model.UDPStreamTimeout = types.Int64Null()
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

	return model
}
