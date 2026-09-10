package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/models"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var _ datasource.DataSource = &clientInfoDataSource{}

func NewClientInfoDataSource() datasource.DataSource {
	return &clientInfoDataSource{}
}

type clientInfoDataSource struct {
	client *Client
}

type clientInfoDataSourceModel struct {
	ID                    types.String         `tfsdk:"id"`
	Site                  types.String         `tfsdk:"site"`
	MAC                   types.String         `tfsdk:"mac"`
	Name                  types.String         `tfsdk:"name"`
	DisplayName           types.String         `tfsdk:"display_name"`
	Hostname              types.String         `tfsdk:"hostname"`
	IP                    types.String         `tfsdk:"ip"`
	FixedIP               types.String         `tfsdk:"fixed_ip"`
	Network               types.Object         `tfsdk:"network"`
	UsergroupID           types.String         `tfsdk:"usergroup_id"`
	Blocked               types.Bool           `tfsdk:"blocked"`
	IsGuest               types.Bool           `tfsdk:"is_guest"`
	IsWired               types.Bool           `tfsdk:"is_wired"`
	Authorized            types.Bool           `tfsdk:"authorized"`
	Status                types.String         `tfsdk:"status"`
	Uptime                timetypes.GoDuration `tfsdk:"uptime"`
	FirstSeen             types.Int64          `tfsdk:"first_seen"`
	LastSeen              types.Int64          `tfsdk:"last_seen"`
	Oui                   types.String         `tfsdk:"oui"`
	LocalDNSRecord        types.String         `tfsdk:"local_dns_record"`
	LocalDNSRecordEnabled types.Bool           `tfsdk:"local_dns_record_enabled"`
	UseFixedIP            types.Bool           `tfsdk:"use_fixedip"`
	APMAC                 types.String         `tfsdk:"ap_mac"`
	Channel               types.Int64          `tfsdk:"channel"`
	Radio                 types.String         `tfsdk:"radio"`
	RadioName             types.String         `tfsdk:"radio_name"`
	Essid                 types.String         `tfsdk:"essid"`
	BSSID                 types.String         `tfsdk:"bssid"`
	Signal                types.Int64          `tfsdk:"signal"`
	RSSI                  types.Int64          `tfsdk:"rssi"`
	Noise                 types.Int64          `tfsdk:"noise"`
	Tx                    types.Object         `tfsdk:"tx"`
	Rx                    types.Object         `tfsdk:"rx"`
	WiredRateMbps         types.Int64          `tfsdk:"wired_rate_mbps"`
	SwPort                types.Int64          `tfsdk:"sw_port"`
	LastUplink            types.Object         `tfsdk:"last_uplink"`
	LastConnectionNetwork types.Object         `tfsdk:"last_connection_network"`
	Timeouts              timeouts.Value       `tfsdk:"timeouts"`
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
	attributes := models.ClientInfoDataSourceSchema()
	attributes["timeouts"] = timeouts.Attributes(ctx)

	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a specific client by MAC address.",

		Attributes: attributes,
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

	readTimeout, timeoutDiags := data.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

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

	data.ID = util.StringValueOrNull(clientInfo.Id)
	data.Name = util.StringValueOrNull(clientInfo.Name)
	data.DisplayName = util.StringValueOrNull(clientInfo.DisplayName)
	data.Hostname = util.StringValueOrNull(clientInfo.Hostname)
	data.IP = util.StringValueOrNull(clientInfo.IP)
	data.FixedIP = util.StringValueOrNull(clientInfo.FixedIP)
	data.UsergroupID = util.StringValueOrNull(clientInfo.UsergroupId)
	data.Blocked = types.BoolValue(clientInfo.Blocked)
	data.IsGuest = types.BoolValue(clientInfo.IsGuest)
	data.IsWired = types.BoolValue(clientInfo.IsWired)
	data.Authorized = types.BoolValue(clientInfo.Authorized)
	data.Status = util.StringValueOrNull(clientInfo.Status)
	data.Uptime = util.DurationPtrValue(clientInfo.Uptime, time.Second)
	data.FirstSeen = types.Int64PointerValue(clientInfo.FirstSeen)
	data.LastSeen = types.Int64PointerValue(clientInfo.LastSeen)
	data.Oui = util.StringValueOrNull(clientInfo.Oui)
	data.LocalDNSRecord = util.StringValueOrNull(clientInfo.LocalDNSRecord)
	data.LocalDNSRecordEnabled = types.BoolValue(clientInfo.LocalDNSRecordEnabled)
	data.UseFixedIP = types.BoolValue(clientInfo.UseFixedip)
	data.APMAC = util.StringValueOrNull(clientInfo.ApMac)
	data.Channel = types.Int64PointerValue(clientInfo.Channel)
	data.Radio = util.StringValueOrNull(clientInfo.Radio)
	data.RadioName = util.StringValueOrNull(clientInfo.RadioName)
	data.Essid = util.StringValueOrNull(clientInfo.Essid)
	data.BSSID = util.StringValueOrNull(clientInfo.Bssid)
	data.Signal = types.Int64PointerValue(clientInfo.Signal)
	data.RSSI = types.Int64PointerValue(clientInfo.Rssi)
	data.Noise = types.Int64PointerValue(clientInfo.Noise)
	data.WiredRateMbps = types.Int64PointerValue(clientInfo.WiredRateMbps)
	data.SwPort = types.Int64PointerValue(clientInfo.SwPort)
	data.Site = util.StringValueOrNull(site)

	var diags diag.Diagnostics
	data.Network, diags = models.ClientNetworkValue(
		ctx,
		clientInfo.NetworkId,
		clientInfo.NetworkName,
	)
	resp.Diagnostics.Append(diags...)
	data.LastUplink, diags = models.ClientLastUplinkValue(
		ctx,
		clientInfo.LastUplinkMac,
		clientInfo.LastUplinkName,
		clientInfo.LastUplinkRemotePort,
	)
	resp.Diagnostics.Append(diags...)
	data.LastConnectionNetwork, diags = models.ClientNetworkValue(
		ctx,
		clientInfo.LastConnectionNetworkId,
		clientInfo.LastConnectionNetworkName,
	)
	resp.Diagnostics.Append(diags...)
	data.Tx, diags = models.ClientTrafficValue(ctx, clientInfo.TxRate, clientInfo.TxBytes)
	resp.Diagnostics.Append(diags...)
	data.Rx, diags = models.ClientTrafficValue(ctx, clientInfo.RxRate, clientInfo.RxBytes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
