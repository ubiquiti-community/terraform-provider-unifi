package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/models"
)

var _ datasource.DataSource = &clientInfoDataSource{}

func NewClientInfoDataSource() datasource.DataSource {
	return &clientInfoDataSource{}
}

type clientInfoDataSource struct {
	client *Client
}

type clientInfoDataSourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Site                      types.String `tfsdk:"site"`
	MAC                       types.String `tfsdk:"mac"`
	Name                      types.String `tfsdk:"name"`
	DisplayName               types.String `tfsdk:"display_name"`
	Hostname                  types.String `tfsdk:"hostname"`
	IP                        types.String `tfsdk:"ip"`
	FixedIP                   types.String `tfsdk:"fixed_ip"`
	NetworkID                 types.String `tfsdk:"network_id"`
	NetworkName               types.String `tfsdk:"network_name"`
	UsergroupID               types.String `tfsdk:"usergroup_id"`
	Blocked                   types.Bool   `tfsdk:"blocked"`
	IsGuest                   types.Bool   `tfsdk:"is_guest"`
	IsWired                   types.Bool   `tfsdk:"is_wired"`
	Authorized                types.Bool   `tfsdk:"authorized"`
	Status                    types.String `tfsdk:"status"`
	Uptime                    types.Int64  `tfsdk:"uptime"`
	FirstSeen                 types.Int64  `tfsdk:"first_seen"`
	LastSeen                  types.Int64  `tfsdk:"last_seen"`
	Oui                       types.String `tfsdk:"oui"`
	LocalDNSRecord            types.String `tfsdk:"local_dns_record"`
	LocalDNSRecordEnabled     types.Bool   `tfsdk:"local_dns_record_enabled"`
	UseFixedIP                types.Bool   `tfsdk:"use_fixedip"`
	APMAC                     types.String `tfsdk:"ap_mac"`
	Channel                   types.Int64  `tfsdk:"channel"`
	Radio                     types.String `tfsdk:"radio"`
	RadioName                 types.String `tfsdk:"radio_name"`
	Essid                     types.String `tfsdk:"essid"`
	BSSID                     types.String `tfsdk:"bssid"`
	Signal                    types.Int64  `tfsdk:"signal"`
	RSSI                      types.Int64  `tfsdk:"rssi"`
	Noise                     types.Int64  `tfsdk:"noise"`
	TxRate                    types.Int64  `tfsdk:"tx_rate"`
	RxRate                    types.Int64  `tfsdk:"rx_rate"`
	TxBytes                   types.Int64  `tfsdk:"tx_bytes"`
	RxBytes                   types.Int64  `tfsdk:"rx_bytes"`
	WiredRateMbps             types.Int64  `tfsdk:"wired_rate_mbps"`
	SwPort                    types.Int64  `tfsdk:"sw_port"`
	LastUplinkMAC             types.String `tfsdk:"last_uplink_mac"`
	LastUplinkName            types.String `tfsdk:"last_uplink_name"`
	LastConnectionNetworkID   types.String `tfsdk:"last_connection_network_id"`
	LastConnectionNetworkName types.String `tfsdk:"last_connection_network_name"`
}

func (d *clientInfoDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client_info"
}

func (d *clientInfoDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a specific client by MAC address.",

		Attributes: models.ClientInfoDataSourceSchema(),
	}
}

func (d *clientInfoDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	d.client = client
}

func (d *clientInfoDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data clientInfoDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	mac := data.MAC.ValueString()
	if mac == "" {
		resp.Diagnostics.AddError(
			"Missing MAC Address",
			"MAC address is required to retrieve client information.",
		)
		return
	}

	clientInfo, err := d.client.GetClientInfo(ctx, site, mac)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client Info",
			fmt.Sprintf("Could not read client info for MAC %s: %s", mac, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(clientInfo.Id)
	data.Name = types.StringValue(clientInfo.Name)
	data.DisplayName = types.StringValue(clientInfo.DisplayName)
	data.Hostname = types.StringValue(clientInfo.Hostname)
	data.IP = types.StringValue(clientInfo.IP)
	data.FixedIP = types.StringValue(clientInfo.FixedIP)
	data.NetworkID = types.StringValue(clientInfo.NetworkId)
	data.NetworkName = types.StringValue(clientInfo.NetworkName)
	data.UsergroupID = types.StringValue(clientInfo.UsergroupId)
	data.Blocked = types.BoolValue(clientInfo.Blocked)
	data.IsGuest = types.BoolValue(clientInfo.IsGuest)
	data.IsWired = types.BoolValue(clientInfo.IsWired)
	data.Authorized = types.BoolValue(clientInfo.Authorized)
	data.Status = types.StringValue(clientInfo.Status)
	data.Uptime = types.Int64Value(int64(clientInfo.Uptime))
	data.FirstSeen = types.Int64Value(int64(clientInfo.FirstSeen))
	data.LastSeen = types.Int64Value(int64(clientInfo.LastSeen))
	data.Oui = types.StringValue(clientInfo.Oui)
	data.LocalDNSRecord = types.StringValue(clientInfo.LocalDNSRecord)
	data.LocalDNSRecordEnabled = types.BoolValue(clientInfo.LocalDNSRecordEnabled)
	data.UseFixedIP = types.BoolValue(clientInfo.UseFixedip)
	data.APMAC = types.StringValue(clientInfo.ApMac)
	data.Channel = types.Int64Value(int64(clientInfo.Channel))
	data.Radio = types.StringValue(clientInfo.Radio)
	data.RadioName = types.StringValue(clientInfo.RadioName)
	data.Essid = types.StringValue(clientInfo.Essid)
	data.BSSID = types.StringValue(clientInfo.Bssid)
	data.Signal = types.Int64Value(int64(clientInfo.Signal))
	data.RSSI = types.Int64Value(int64(clientInfo.Rssi))
	data.Noise = types.Int64Value(int64(clientInfo.Noise))
	data.TxRate = types.Int64Value(int64(clientInfo.TxRate))
	data.RxRate = types.Int64Value(int64(clientInfo.RxRate))
	data.TxBytes = types.Int64Value(int64(clientInfo.TxBytes))
	data.RxBytes = types.Int64Value(int64(clientInfo.RxBytes))
	data.WiredRateMbps = types.Int64Value(int64(clientInfo.WiredRateMbps))
	data.SwPort = types.Int64Value(int64(clientInfo.SwPort))
	data.LastUplinkMAC = types.StringValue(clientInfo.LastUplinkMac)
	data.LastUplinkName = types.StringValue(clientInfo.LastUplinkName)
	data.LastConnectionNetworkID = types.StringValue(clientInfo.LastConnectionNetworkId)
	data.LastConnectionNetworkName = types.StringValue(clientInfo.LastConnectionNetworkName)
	data.Site = types.StringValue(site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
