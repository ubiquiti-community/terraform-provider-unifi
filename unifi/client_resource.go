package unifi

import (
	"context"
	"errors"
	"fmt"
	"net"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &clientResource{}
	_ resource.ResourceWithImportState = &clientResource{}
	_ resource.ResourceWithIdentity    = &clientResource{}
	_ resource.ResourceWithImportState = &clientResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &clientResource{}
	_ list.ListResourceWithConfigure = &clientResource{}
)

const (
	defaultSkipForgetOnDestroy = false
	defaultAllowExisting       = true
)

func NewClientResource() resource.Resource {
	return &clientResource{}
}

func NewClientListResource() list.ListResource {
	return &clientResource{}
}

// clientResource defines the resource implementation.
type clientResource struct {
	client *Client
}

// clientResourceModel describes the resource data model.
type clientResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Site                types.String `tfsdk:"site"`
	MAC                 types.String `tfsdk:"mac"`
	Name                types.String `tfsdk:"name"`
	GroupID             types.String `tfsdk:"group_id"`
	Note                types.String `tfsdk:"note"`
	FixedIP             types.String `tfsdk:"fixed_ip"`
	NetworkID           types.String `tfsdk:"network_id"`
	Blocked             types.Bool   `tfsdk:"blocked"`
	DevIDOverride       types.Int64  `tfsdk:"dev_id_override"`
	LocalDNSRecord      types.String `tfsdk:"local_dns_record"`
	AllowExisting       types.Bool   `tfsdk:"allow_existing"`
	SkipForgetOnDestroy types.Bool   `tfsdk:"skip_forget_on_destroy"`

	// Computed attributes
	Hostname            types.String `tfsdk:"hostname"`
	IP                  types.String `tfsdk:"ip"`
	Anomalies           types.Int64  `tfsdk:"anomalies"`
	AssocTime           types.Int64  `tfsdk:"assoc_time"`
	Authorized          types.Bool   `tfsdk:"authorized"`
	DisconnectTimestamp types.Int64  `tfsdk:"disconnect_timestamp"`
	EagerlyDiscovered   types.Bool   `tfsdk:"eagerly_discovered"`
	FingerprintOverride types.Bool   `tfsdk:"fingerprint_override"`
	FirstSeen           types.Int64  `tfsdk:"first_seen"`
	HostnameSource      types.String `tfsdk:"hostname_source"`
	IPv6Addresses       types.List   `tfsdk:"ipv6_addresses"`
	IsWired             types.Bool   `tfsdk:"is_wired"`
	LatestAssocTime     types.Int64  `tfsdk:"latest_assoc_time"`
	Network             types.String `tfsdk:"network"`
	Noted               types.Bool   `tfsdk:"noted"`
	OUI                 types.String `tfsdk:"oui"`
	QOSPolicyApplied    types.Bool   `tfsdk:"qos_policy_applied"`
	Satisfaction        types.Int64  `tfsdk:"satisfaction"`
	TxRetries           types.Int64  `tfsdk:"tx_retries"`
	Uptime              types.Int64  `tfsdk:"uptime"`
	UserID              types.String `tfsdk:"user_id"`
	VLAN                types.Int64  `tfsdk:"vlan"`

	// Nested computed attributes
	Gateway     types.Object `tfsdk:"gateway"`
	GuestStatus types.Object `tfsdk:"guest_status"`
	Last        types.Object `tfsdk:"last"`
	Switch      types.Object `tfsdk:"switch"`
	UptimeStats types.Object `tfsdk:"uptime_stats"`
	WiFi        types.Object `tfsdk:"wifi"`
	Wired       types.Object `tfsdk:"wired"`
}

type clientGatewayModel struct {
	MAC  types.String `tfsdk:"mac"`
	VLAN types.Int64  `tfsdk:"vlan"`
}

type clientGuestStatusModel struct {
	IsGuest      types.Bool `tfsdk:"is_guest"`
	IsGuestByUGW types.Bool `tfsdk:"is_guest_by_ugw"`
	IsGuestByUSW types.Bool `tfsdk:"is_guest_by_usw"`
}

type clientLastModel struct {
	Identity1x            types.String `tfsdk:"identity_1x"`
	ConnectionNetworkID   types.String `tfsdk:"connection_network_id"`
	ConnectionNetworkName types.String `tfsdk:"connection_network_name"`
	IP                    types.String `tfsdk:"ip"`
	IPv6                  types.List   `tfsdk:"ipv6"`
	ReachableByGW         types.Int64  `tfsdk:"reachable_by_gw"`
	Seen                  types.Int64  `tfsdk:"seen"`
	SeenByUGW             types.Int64  `tfsdk:"seen_by_ugw"`
	SeenByUSW             types.Int64  `tfsdk:"seen_by_usw"`
	UplinkMAC             types.String `tfsdk:"uplink_mac"`
	UplinkName            types.String `tfsdk:"uplink_name"`
	UplinkRemotePort      types.Int64  `tfsdk:"uplink_remote_port"`
}

type clientSwitchModel struct {
	Depth types.Int64  `tfsdk:"depth"`
	MAC   types.String `tfsdk:"mac"`
	Port  types.Int64  `tfsdk:"port"`
}

type clientUptimeStatsModel struct {
	Uptime      types.Int64 `tfsdk:"uptime"`
	UptimeByUGW types.Int64 `tfsdk:"uptime_by_ugw"`
	UptimeByUSW types.Int64 `tfsdk:"uptime_by_usw"`
}

type clientWiFiModel struct {
	TxAttempts          types.Int64   `tfsdk:"tx_attempts"`
	TxDropped           types.Int64   `tfsdk:"tx_dropped"`
	TxRetriesPercentage types.Float64 `tfsdk:"tx_retries_percentage"`
}

type clientWiredModel struct {
	RateMbps  types.Int64   `tfsdk:"rate_mbps"`
	RxBytes   types.Int64   `tfsdk:"rx_bytes"`
	RxBytesR  types.Float64 `tfsdk:"rx_bytes_r"`
	RxPackets types.Int64   `tfsdk:"rx_packets"`
	TxBytes   types.Int64   `tfsdk:"tx_bytes"`
	TxBytesR  types.Float64 `tfsdk:"tx_bytes_r"`
	TxPackets types.Int64   `tfsdk:"tx_packets"`
}

type clientIdentityModel struct {
	ID  types.String `tfsdk:"id"`
	MAC types.String `tfsdk:"mac"`
}

// clientListConfigModel describes the list configuration model.
type clientListConfigModel struct {
	Site        types.String `tfsdk:"site"`
	NetworkID   types.String `tfsdk:"network_id"`
	NetworkName types.String `tfsdk:"network_name"`
}

func (r *clientResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *clientResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				OptionalForImport: true,
			},
			"mac": identityschema.StringAttribute{
				OptionalForImport: true,
			},
		},
	}
}

func (r *clientResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a client of the network, identified by unique MAC addresses.

Clients are created in the controller when observed on the network, so the resource defaults to allowing itself to just take over management of a MAC address, but this can be turned off.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the client with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the client.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The group ID to attach to the client (controls QoS and other group-based settings).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A note with additional information for the client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fixed_ip": schema.StringAttribute{
				MarkdownDescription: "A fixed IPv4 address for this client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID for this client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this client should be blocked from the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dev_id_override": schema.Int64Attribute{
				MarkdownDescription: "Override the device fingerprint.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseNonNullStateForUnknown(),
				},
			},
			"local_dns_record": schema.StringAttribute{
				MarkdownDescription: "Specifies the local DNS record for this client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_existing": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should just take over control of an existing client.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(defaultAllowExisting),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"skip_forget_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this resource should tell the controller to \"forget\" the client on destroy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(defaultSkipForgetOnDestroy),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"anomalies": schema.Int64Attribute{
				MarkdownDescription: "Number of anomalies detected for this client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"assoc_time": schema.Int64Attribute{
				MarkdownDescription: "Association time timestamp.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"authorized": schema.BoolAttribute{
				MarkdownDescription: "Whether the client is authorized.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disconnect_timestamp": schema.Int64Attribute{
				MarkdownDescription: "Timestamp of last disconnect.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"eagerly_discovered": schema.BoolAttribute{
				MarkdownDescription: "Whether the client was eagerly discovered.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint_override": schema.BoolAttribute{
				MarkdownDescription: "Whether device fingerprint is overridden.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"first_seen": schema.Int64Attribute{
				MarkdownDescription: "Timestamp when client was first seen.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"hostname_source": schema.StringAttribute{
				MarkdownDescription: "Source of the hostname.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_addresses": schema.ListAttribute{
				MarkdownDescription: "List of IPv6 addresses assigned to the client.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"is_wired": schema.BoolAttribute{
				MarkdownDescription: "Whether the client is connected via wired connection.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"latest_assoc_time": schema.Int64Attribute{
				MarkdownDescription: "Latest association time timestamp.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "Network name the client is connected to.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"noted": schema.BoolAttribute{
				MarkdownDescription: "Whether the client has a note.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"oui": schema.StringAttribute{
				MarkdownDescription: "Organizationally Unique Identifier from MAC address.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"qos_policy_applied": schema.BoolAttribute{
				MarkdownDescription: "Whether QoS policy is applied to this client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"satisfaction": schema.Int64Attribute{
				MarkdownDescription: "Client satisfaction score.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tx_retries": schema.Int64Attribute{
				MarkdownDescription: "Number of transmission retries.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uptime": schema.Int64Attribute{
				MarkdownDescription: "Client uptime in seconds.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "User ID associated with the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "VLAN ID the client is on.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"gateway": schema.SingleNestedAttribute{
				MarkdownDescription: "Gateway information for the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"mac": schema.StringAttribute{
						MarkdownDescription: "MAC address of the gateway.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"vlan": schema.Int64Attribute{
						MarkdownDescription: "VLAN ID on the gateway.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"guest_status": schema.SingleNestedAttribute{
				MarkdownDescription: "Guest status information for the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"is_guest": schema.BoolAttribute{
						MarkdownDescription: "Whether the client is a guest.",
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"is_guest_by_ugw": schema.BoolAttribute{
						MarkdownDescription: "Whether the client is a guest according to UGW.",
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"is_guest_by_usw": schema.BoolAttribute{
						MarkdownDescription: "Whether the client is a guest according to USW.",
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"last": schema.SingleNestedAttribute{
				MarkdownDescription: "Last known information about the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"identity_1x": schema.StringAttribute{
						MarkdownDescription: "Last 802.1X identity.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"connection_network_id": schema.StringAttribute{
						MarkdownDescription: "Last connection network ID.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"connection_network_name": schema.StringAttribute{
						MarkdownDescription: "Last connection network name.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ip": schema.StringAttribute{
						MarkdownDescription: "Last known IP address.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ipv6": schema.ListAttribute{
						MarkdownDescription: "Last known IPv6 addresses.",
						Computed:            true,
						ElementType:         types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"reachable_by_gw": schema.Int64Attribute{
						MarkdownDescription: "Timestamp when last reachable by gateway.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"seen": schema.Int64Attribute{
						MarkdownDescription: "Timestamp when client was last seen.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"seen_by_ugw": schema.Int64Attribute{
						MarkdownDescription: "Timestamp when last seen by UGW.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"seen_by_usw": schema.Int64Attribute{
						MarkdownDescription: "Timestamp when last seen by USW.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"uplink_mac": schema.StringAttribute{
						MarkdownDescription: "MAC address of last uplink.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"uplink_name": schema.StringAttribute{
						MarkdownDescription: "Name of last uplink.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"uplink_remote_port": schema.Int64Attribute{
						MarkdownDescription: "Remote port of last uplink.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"switch": schema.SingleNestedAttribute{
				MarkdownDescription: "Switch connection information for the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"depth": schema.Int64Attribute{
						MarkdownDescription: "Switch depth in the network topology.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"mac": schema.StringAttribute{
						MarkdownDescription: "MAC address of the connected switch.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "Switch port the client is connected to.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"uptime_stats": schema.SingleNestedAttribute{
				MarkdownDescription: "Uptime statistics for the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"uptime": schema.Int64Attribute{
						MarkdownDescription: "Client uptime in seconds.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"uptime_by_ugw": schema.Int64Attribute{
						MarkdownDescription: "Client uptime as reported by UGW.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"uptime_by_usw": schema.Int64Attribute{
						MarkdownDescription: "Client uptime as reported by USW.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"wifi": schema.SingleNestedAttribute{
				MarkdownDescription: "WiFi connection statistics for the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"tx_attempts": schema.Int64Attribute{
						MarkdownDescription: "Number of WiFi transmission attempts.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tx_dropped": schema.Int64Attribute{
						MarkdownDescription: "Number of dropped WiFi transmissions.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tx_retries_percentage": schema.Float64Attribute{
						MarkdownDescription: "Percentage of WiFi transmission retries.",
						Computed:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"wired": schema.SingleNestedAttribute{
				MarkdownDescription: "Wired connection statistics for the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"rate_mbps": schema.Int64Attribute{
						MarkdownDescription: "Wired connection rate in Mbps.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"rx_bytes": schema.Int64Attribute{
						MarkdownDescription: "Bytes received on wired connection.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"rx_bytes_r": schema.Float64Attribute{
						MarkdownDescription: "Bytes received rate on wired connection.",
						Computed:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.UseStateForUnknown(),
						},
					},
					"rx_packets": schema.Int64Attribute{
						MarkdownDescription: "Packets received on wired connection.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tx_bytes": schema.Int64Attribute{
						MarkdownDescription: "Bytes transmitted on wired connection.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tx_bytes_r": schema.Float64Attribute{
						MarkdownDescription: "Bytes transmitted rate on wired connection.",
						Computed:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.UseStateForUnknown(),
						},
					},
					"tx_packets": schema.Int64Attribute{
						MarkdownDescription: "Packets transmitted on wired connection.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *clientResource) Configure(
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

func (r *clientResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan clientResourceModel
	var id clientIdentityModel

	resp.Diagnostics.Append(req.Identity.Get(ctx, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if (id.MAC.IsNull() || id.MAC.IsUnknown()) && (!plan.MAC.IsNull() && !plan.MAC.IsUnknown()) {
		id.MAC = plan.MAC
	}

	if (id.ID.IsNull() || id.ID.IsUnknown()) && (!plan.ID.IsNull() && !plan.ID.IsUnknown()) {
		id.ID = plan.ID
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the plan to UniFi Client struct
	client, diags := r.planToClient(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowExisting := plan.AllowExisting.ValueBool()

	// Create the Client
	createdClient, err := r.client.CreateClient(ctx, site, client)
	if err != nil {
		var apiErr *unifi.APIError
		if !errors.As(err, &apiErr) || (apiErr.Message != "api.err.MacUsed" || !allowExisting) {
			resp.Diagnostics.AddError(
				"Error Creating Client",
				"Could not create client: "+err.Error(),
			)
			return
		}

		// MAC in use, just absorb the existing client
		mac := plan.MAC.ValueString()
		existingClient, err := r.client.GetClientByMAC(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Existing Client",
				"Could not get existing client with MAC "+mac+": "+err.Error(),
			)
			return
		}

		// Implement merge pattern for existing client
		mergedClient := r.mergeClient(existingClient, client)
		updatedClient, err := r.client.UpdateClient(ctx, site, mergedClient)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Existing Client",
				"Could not update existing client: "+err.Error(),
			)
			return
		}
		createdClient = updatedClient
	}

	// Convert response back to model
	diags = r.clientToModel(ctx, createdClient, &plan, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id.MAC.ValueString() != plan.MAC.ValueString() {
		id.MAC = plan.MAC
	}

	if id.ID.ValueString() != plan.ID.ValueString() {
		id.ID = plan.ID
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clientResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var id clientIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state clientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !id.MAC.IsNull() && !id.MAC.IsUnknown() {
		state.MAC = id.MAC
	}

	if !id.ID.IsNull() && !id.ID.IsUnknown() {
		state.ID = id.ID
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	mac := id.MAC.ValueString()

	// Get the Client from the API
	var client *unifi.Client
	var err error

	// If we have a MAC address, try to get by MAC first
	if mac != "" {
		client, err = r.client.GetClientByMAC(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Client",
				"Could not read client with MAC "+mac+": "+err.Error(),
			)
			return
		}
	} else if id.ID.ValueString() != "" {
		// Otherwise use ID
		client, err = r.client.GetClient(ctx, site, id.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Client",
				"Could not read client with ID "+id.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Invalid State",
			"Client must have either an ID or MAC address",
		)
		return
	}

	// Convert API response to model
	resp.Diagnostics.Append(r.clientToModel(ctx, client, &state, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id.MAC.IsNull() || id.MAC.IsUnknown() {
		id.MAC = state.MAC
	}

	if id.ID.IsNull() || id.ID.IsUnknown() {
		id.ID = state.ID
	}

	if state.AllowExisting.IsNull() || state.AllowExisting.IsUnknown() {
		state.AllowExisting = types.BoolValue(defaultAllowExisting)
	}

	if state.SkipForgetOnDestroy.IsNull() || state.SkipForgetOnDestroy.IsUnknown() {
		state.SkipForgetOnDestroy = types.BoolValue(defaultSkipForgetOnDestroy)
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, id)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clientResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state clientResourceModel
	var plan clientResourceModel
	var id clientIdentityModel

	resp.Diagnostics.Append(req.Identity.Get(ctx, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Read the current state (which already contains API values from previous reads)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if (id.MAC.IsNull() || id.MAC.IsUnknown()) && (!plan.MAC.IsNull() && !plan.MAC.IsUnknown()) {
		id.MAC = plan.MAC
	}

	if (id.ID.IsNull() || id.ID.IsUnknown()) && (!plan.ID.IsNull() && !plan.ID.IsUnknown()) {
		id.ID = plan.ID
	}

	// Step 2: Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Step 3: Convert the updated state to API format
	client, diags := r.planToClient(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 4: Send to API
	client.ID = state.ID.ValueString()
	updatedClient, err := r.client.UpdateClient(ctx, site, client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Client",
			"Could not update client with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	diags = r.clientToModel(ctx, updatedClient, &state, site)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id.MAC.ValueString() != state.MAC.ValueString() {
		id.MAC = state.MAC
	}

	if id.ID.ValueString() != state.ID.ValueString() {
		id.ID = state.ID
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *clientResource) applyPlanToState(
	_ context.Context,
	plan *clientResourceModel,
	state *clientResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.MAC.IsNull() && !plan.MAC.IsUnknown() {
		state.MAC = plan.MAC
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		state.GroupID = plan.GroupID
	}
	if !plan.Note.IsNull() && !plan.Note.IsUnknown() {
		state.Note = plan.Note
	}
	if !plan.FixedIP.IsNull() && !plan.FixedIP.IsUnknown() {
		state.FixedIP = plan.FixedIP
	}
	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		state.NetworkID = plan.NetworkID
	}
	if !plan.Blocked.IsNull() && !plan.Blocked.IsUnknown() {
		state.Blocked = plan.Blocked
	}
	if !plan.DevIDOverride.IsNull() && !plan.DevIDOverride.IsUnknown() {
		state.DevIDOverride = plan.DevIDOverride
	}
	if !plan.LocalDNSRecord.IsNull() && !plan.LocalDNSRecord.IsUnknown() {
		state.LocalDNSRecord = plan.LocalDNSRecord
	}
	if !plan.AllowExisting.IsNull() && !plan.AllowExisting.IsUnknown() {
		state.AllowExisting = plan.AllowExisting
	}
	if !plan.SkipForgetOnDestroy.IsNull() && !plan.SkipForgetOnDestroy.IsUnknown() {
		state.SkipForgetOnDestroy = plan.SkipForgetOnDestroy
	}
	// Note: Computed attributes (Hostname, IP) are not applied from plan
}

func (r *clientResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state clientResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	skipForget := state.SkipForgetOnDestroy.ValueBool()

	if skipForget {
		// Just remove from Terraform state without telling UniFi to forget
		return
	}

	// lookup MAC instead of trusting state
	c, err := r.client.GetClient(ctx, site, id)
	if _, ok := err.(*unifi.NotFoundError); ok {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client for Delete",
			"Could not read client with ID "+id+": "+err.Error(),
		)
		return
	}

	err = r.client.DeleteClientByMAC(ctx, site, c.MAC)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Client",
			"Could not delete client with MAC "+c.MAC+": "+err.Error(),
		)
		return
	}
}

func (r *clientResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	pathAttr := "id"

	var idModel clientIdentityModel
	if d := req.Identity.Get(ctx, &idModel); !d.HasError() {
		if !idModel.MAC.IsNull() && !idModel.MAC.IsUnknown() {
			pathAttr = "mac"
		}
	} else if req.ID != "" {
		if _, err := net.ParseMAC(req.ID); err == nil {
			pathAttr = "mac"
		}
	} else {
		resp.Diagnostics.Append(d...)
		return
	}

	if req.ID != "" {
		if pathAttr == "id" {
			idModel.ID = util.StringValueOrNull(req.ID)
		} else {
			idModel.MAC = util.StringValueOrNull(req.ID)
		}
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resource.ImportStatePassthroughWithIdentity(
		ctx,
		path.Root(pathAttr),
		path.Root(pathAttr),
		req,
		resp,
	)
}

// Helper functions for conversion and merging

func (r *clientResource) planToClient(
	_ context.Context,
	plan clientResourceModel,
) (*unifi.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.ID.IsNull() && plan.Name.IsNull() && plan.MAC.IsNull() {
		diags.AddError(
			"Invalid Client",
			"Client must have either an ID, Name, or MAC to be imported",
		)
		return nil, diags
	}

	client := &unifi.Client{
		ID:             plan.ID.ValueString(),
		MAC:            plan.MAC.ValueString(),
		Name:           plan.Name.ValueString(),
		UserGroupID:    plan.GroupID.ValueString(),
		Note:           plan.Note.ValueString(),
		FixedIP:        plan.FixedIP.ValueString(),
		NetworkID:      plan.NetworkID.ValueString(),
		Blocked:        plan.Blocked.ValueBool(),
		LocalDNSRecord: plan.LocalDNSRecord.ValueString(),
	}

	// Note: DevIDOverride is not available in the Client type
	// if !plan.DevIDOverride.IsNull() && !plan.DevIDOverride.IsUnknown() {
	// 	client.DevIdOverride = plan.DevIDOverride.ValueInt64()
	// }

	return client, diags
}

func (r *clientResource) clientToModel(
	ctx context.Context,
	client *unifi.Client,
	model *clientResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if client.ID == "" && client.Name == "" && client.MAC == "" {
		diags.AddError(
			"Invalid Client",
			"Client must have either an ID, Name, or MAC to be imported",
		)
		return diags
	}

	model.ID = util.StringValueOrNull(client.ID)
	model.Site = util.StringValueOrNull(site)
	model.MAC = util.StringValueOrNull(client.MAC)
	model.Name = util.StringValueOrNull(client.Name)
	model.GroupID = util.StringValueOrNull(client.UserGroupID)
	model.Note = util.StringValueOrNull(client.Note)
	model.FixedIP = util.StringValueOrNull(client.FixedIP)
	model.NetworkID = util.StringValueOrNull(client.NetworkID)

	model.Blocked = types.BoolValue(client.Blocked)
	model.DevIDOverride = types.Int64PointerValue(client.DevIdOverride)
	model.LocalDNSRecord = util.StringValueOrNull(client.LocalDNSRecord)

	// Computed attributes
	model.Hostname = util.StringValueOrNull(client.Hostname)

	model.Anomalies = types.Int64PointerValue(client.Anomalies)
	model.AssocTime = types.Int64PointerValue(client.AssocTime)
	model.Authorized = types.BoolValue(client.Authorized)
	model.DisconnectTimestamp = types.Int64PointerValue(client.DisconnectTimestamp)
	model.EagerlyDiscovered = types.BoolValue(client.EagerlyDiscovered)
	model.FingerprintOverride = types.BoolValue(client.FingerprintOverride)
	model.FirstSeen = types.Int64PointerValue(client.FirstSeen)
	model.HostnameSource = util.StringValueOrNull(client.HostnameSource)
	model.IsWired = types.BoolValue(client.IsWired)
	model.LatestAssocTime = types.Int64PointerValue(client.LatestAssocTime)
	model.Network = util.StringValueOrNull(client.Network)
	model.Noted = types.BoolValue(client.Noted)
	model.OUI = util.StringValueOrNull(client.OUI)
	model.QOSPolicyApplied = types.BoolValue(client.QOSPolicyApplied)
	model.Satisfaction = types.Int64PointerValue(client.Satisfaction)
	model.TxRetries = types.Int64PointerValue(client.TxRetries)
	model.Uptime = types.Int64PointerValue(client.Uptime)
	model.UserID = util.StringValueOrNull(client.UserID)
	model.VLAN = types.Int64PointerValue(client.VLAN)

	// IPv6 addresses list
	if len(client.IPv6Addresses) > 0 {
		ipv6Values := make([]attr.Value, len(client.IPv6Addresses))
		for i, addr := range client.IPv6Addresses {
			ipv6Values[i] = util.StringValueOrNull(addr)
		}
		var listDiags diag.Diagnostics
		model.IPv6Addresses, listDiags = types.ListValue(types.StringType, ipv6Values)
		diags.Append(listDiags...)
	} else {
		model.IPv6Addresses = types.ListNull(types.StringType)
	}

	// Gateway nested object
	gatewayAttrs := map[string]attr.Type{
		"mac":  types.StringType,
		"vlan": types.Int64Type,
	}
	if client.GWMAC != "" || client.GWVLAN != nil {
		gatewayValues := map[string]attr.Value{
			"mac":  util.StringValueOrNull(client.GWMAC),
			"vlan": types.Int64PointerValue(client.GWVLAN),
		}
		var objDiags diag.Diagnostics
		model.Gateway, objDiags = types.ObjectValue(gatewayAttrs, gatewayValues)
		diags.Append(objDiags...)
	} else {
		model.Gateway = types.ObjectNull(gatewayAttrs)
	}

	// Guest status nested object
	guestStatusAttrs := map[string]attr.Type{
		"is_guest":        types.BoolType,
		"is_guest_by_ugw": types.BoolType,
		"is_guest_by_usw": types.BoolType,
	}
	guestStatusValues := map[string]attr.Value{
		"is_guest":        types.BoolValue(client.IsGuest),
		"is_guest_by_ugw": types.BoolValue(client.IsGuestByUGW),
		"is_guest_by_usw": types.BoolValue(client.IsGuestByUSW),
	}
	var guestDiags diag.Diagnostics
	model.GuestStatus, guestDiags = types.ObjectValue(guestStatusAttrs, guestStatusValues)
	diags.Append(guestDiags...)

	// Last nested object
	lastAttrs := map[string]attr.Type{
		"identity_1x":             types.StringType,
		"connection_network_id":   types.StringType,
		"connection_network_name": types.StringType,
		"ip":                      types.StringType,
		"ipv6":                    types.ListType{ElemType: types.StringType},
		"reachable_by_gw":         types.Int64Type,
		"seen":                    types.Int64Type,
		"seen_by_ugw":             types.Int64Type,
		"seen_by_usw":             types.Int64Type,
		"uplink_mac":              types.StringType,
		"uplink_name":             types.StringType,
		"uplink_remote_port":      types.Int64Type,
	}

	var lastIPv6 basetypes.ListValue
	if len(client.LastIPv6) > 0 {
		lastIPv6Values := make([]attr.Value, len(client.LastIPv6))
		for i, addr := range client.LastIPv6 {
			lastIPv6Values[i] = util.StringValueOrNull(addr)
		}
		var listDiags diag.Diagnostics
		lastIPv6, listDiags = types.ListValue(types.StringType, lastIPv6Values)
		diags.Append(listDiags...)
	} else {
		lastIPv6 = types.ListNull(types.StringType)
	}

	lastValues := map[string]attr.Value{
		"identity_1x":             util.StringValueOrNull(client.Last1xIdentity),
		"connection_network_id":   util.StringValueOrNull(client.LastConnectionNetworkID),
		"connection_network_name": util.StringValueOrNull(client.LastConnectionNetworkName),
		"ip":                      util.StringValueOrNull(client.LastIP),
		"ipv6":                    lastIPv6,
		"reachable_by_gw":         types.Int64PointerValue(client.LastReachableByGW),
		"seen":                    types.Int64PointerValue(client.LastSeen),
		"seen_by_ugw":             types.Int64PointerValue(client.LastSeenByUGW),
		"seen_by_usw":             types.Int64PointerValue(client.LastSeenByUSW),
		"uplink_mac":              util.StringValueOrNull(client.LastUplinkMAC),
		"uplink_name":             util.StringValueOrNull(client.LastUplinkName),
		"uplink_remote_port":      types.Int64PointerValue(client.LastUplinkRemotePort),
	}
	var lastDiags diag.Diagnostics
	model.Last, lastDiags = types.ObjectValue(lastAttrs, lastValues)
	diags.Append(lastDiags...)

	// Switch nested object
	switchAttrs := map[string]attr.Type{
		"depth": types.Int64Type,
		"mac":   types.StringType,
		"port":  types.Int64Type,
	}
	if client.SwMAC != "" || client.SwDepth != nil || client.SwPort != nil {
		switchValues := map[string]attr.Value{
			"depth": types.Int64PointerValue(client.SwDepth),
			"mac":   util.StringValueOrNull(client.SwMAC),
			"port":  types.Int64PointerValue(client.SwPort),
		}
		var switchDiags diag.Diagnostics
		model.Switch, switchDiags = types.ObjectValue(switchAttrs, switchValues)
		diags.Append(switchDiags...)
	} else {
		model.Switch = types.ObjectNull(switchAttrs)
	}

	// Uptime stats nested object
	uptimeStatsAttrs := map[string]attr.Type{
		"uptime":        types.Int64Type,
		"uptime_by_ugw": types.Int64Type,
		"uptime_by_usw": types.Int64Type,
	}
	uptimeStatsValues := map[string]attr.Value{
		"uptime":        types.Int64PointerValue(client.Uptime),
		"uptime_by_ugw": types.Int64PointerValue(client.UptimeByUGW),
		"uptime_by_usw": types.Int64PointerValue(client.UptimeByUSW),
	}
	var uptimeDiags diag.Diagnostics
	model.UptimeStats, uptimeDiags = types.ObjectValue(uptimeStatsAttrs, uptimeStatsValues)
	diags.Append(uptimeDiags...)

	// WiFi nested object
	wifiAttrs := map[string]attr.Type{
		"tx_attempts":           types.Int64Type,
		"tx_dropped":            types.Int64Type,
		"tx_retries_percentage": types.Float64Type,
	}
	if client.WiFiTxAttempts != nil || client.WiFiTxDropped != nil ||
		client.WiFiTxRetriesPercentage != 0 {
		wifiValues := map[string]attr.Value{
			"tx_attempts":           types.Int64PointerValue(client.WiFiTxAttempts),
			"tx_dropped":            types.Int64PointerValue(client.WiFiTxDropped),
			"tx_retries_percentage": types.Float64Value(client.WiFiTxRetriesPercentage),
		}
		var wifiDiags diag.Diagnostics
		model.WiFi, wifiDiags = types.ObjectValue(wifiAttrs, wifiValues)
		diags.Append(wifiDiags...)
	} else {
		model.WiFi = types.ObjectNull(wifiAttrs)
	}

	// Wired nested object
	wiredAttrs := map[string]attr.Type{
		"rate_mbps":  types.Int64Type,
		"rx_bytes":   types.Int64Type,
		"rx_bytes_r": types.Float64Type,
		"rx_packets": types.Int64Type,
		"tx_bytes":   types.Int64Type,
		"tx_bytes_r": types.Float64Type,
		"tx_packets": types.Int64Type,
	}
	if client.WiredRateMbps != nil || client.WiredRxBytes != nil || client.WiredTxBytes != nil {
		wiredValues := map[string]attr.Value{
			"rate_mbps":  types.Int64PointerValue(client.WiredRateMbps),
			"rx_bytes":   types.Int64PointerValue(client.WiredRxBytes),
			"rx_bytes_r": types.Float64Value(client.WiredRxBytesR),
			"rx_packets": types.Int64PointerValue(client.WiredRxPackets),
			"tx_bytes":   types.Int64PointerValue(client.WiredTxBytes),
			"tx_bytes_r": types.Float64Value(client.WiredTxBytesR),
			"tx_packets": types.Int64PointerValue(client.WiredTxPackets),
		}
		var wiredDiags diag.Diagnostics
		model.Wired, wiredDiags = types.ObjectValue(wiredAttrs, wiredValues)
		diags.Append(wiredDiags...)
	} else {
		model.Wired = types.ObjectNull(wiredAttrs)
	}

	return diags
}

func (r *clientResource) mergeClient(
	existing *unifi.Client,
	planned *unifi.Client,
) *unifi.Client {
	// Start with the existing client to preserve all UniFi internal fields
	merged := *existing

	// Override with planned values
	merged.Name = planned.Name
	merged.UserGroupID = planned.UserGroupID
	merged.Note = planned.Note
	merged.FixedIP = planned.FixedIP
	merged.NetworkID = planned.NetworkID
	merged.Blocked = planned.Blocked
	merged.LocalDNSRecord = planned.LocalDNSRecord

	return &merged
}

func (r *clientResource) ListResourceConfigSchema(
	ctx context.Context,
	req list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List clients in a site, optionally filtered by network.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list clients from.",
				Optional:            true,
			},
			"network_id": listschema.StringAttribute{
				MarkdownDescription: "Filter clients by network ID.",
				Optional:            true,
			},
			"network_name": listschema.StringAttribute{
				MarkdownDescription: "Filter clients by network name.",
				Optional:            true,
			},
		},
	}
}

func (r *clientResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config clientListConfigModel

	// Read list config data into the model
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Resolve network_id if network_name is provided
	networkID := config.NetworkID.ValueString()
	if !config.NetworkName.IsNull() && !config.NetworkName.IsUnknown() {
		networkName := config.NetworkName.ValueString()
		network, err := r.client.GetNetworkByName(ctx, site, networkName)
		if err != nil {
			result := req.NewListResult(ctx)
			result.Diagnostics.AddError(
				"Error Getting Network by Name",
				"Could not get network with name "+networkName+": "+err.Error(),
			)
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}
		networkID = network.ID
	}

	// List all clients
	clients, err := r.client.ListClient(ctx, site)
	if err != nil {
		result := req.NewListResult(ctx)
		result.Diagnostics.AddError(
			"Error Listing Clients",
			"Could not list clients: "+err.Error(),
		)
		stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
		return
	}

	// Define the function that will push results into the stream
	stream.Results = func(push func(list.ListResult) bool) {
		for _, client := range clients {
			// Apply network filter if specified
			if networkID != "" && client.NetworkID != networkID {
				continue
			}

			// Initialize a new result object for each client
			result := req.NewListResult(ctx)

			// Set the user-friendly name of this client
			if client.Name != "" {
				result.DisplayName = client.Name
			} else if client.Hostname != "" {
				result.DisplayName = client.Hostname
			} else {
				result.DisplayName = client.MAC
			}

			// Set resource identity data on the result
			result.Diagnostics.Append(result.Identity.Set(ctx, client.ID)...)

			// Convert the client to the resource model
			var model clientResourceModel
			modelDiags := r.clientToModel(ctx, &client, &model, site)
			result.Diagnostics.Append(modelDiags...)

			// Set the resource information on the result
			if !result.Diagnostics.HasError() {
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			// Send the result to the stream.
			if !push(result) {
				return
			}
		}
	}
}
