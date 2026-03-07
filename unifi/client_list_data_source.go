package unifi

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	gounifi "github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var _ datasource.DataSource = &clientListDataSource{}

func NewClientListDataSource() datasource.DataSource {
	return &clientListDataSource{}
}

type clientListDataSource struct {
	client *Client

	// Cache group name → ID lookups per site to avoid repeated API calls.
	groupCacheMu sync.Mutex
	groupCache   map[string]map[string]string // site → (name → id)
}

type clientListDataSourceModel struct {
	Site    types.String `tfsdk:"site"`
	Group   types.String `tfsdk:"group"`
	Wired   types.Bool   `tfsdk:"wired"`
	Blocked types.Bool   `tfsdk:"blocked"`
	OUI     types.String `tfsdk:"oui"`
	Clients types.List   `tfsdk:"clients"`
}

func clientListEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		// From Client (user REST API)
		"id":                        types.StringType,
		"mac":                       types.StringType,
		"name":                      types.StringType,
		"display_name":              types.StringType,
		"group_id":                  types.StringType,
		"note":                      types.StringType,
		"fixed_ip":                  types.StringType,
		"fixed_ap_mac":              types.StringType,
		"network_id":                types.StringType,
		"network_members_group_ids": types.ListType{ElemType: types.StringType},
		"blocked":                   types.BoolType,
		"local_dns_record":          types.StringType,
		"hostname":                  types.StringType,

		// From ClientInfo (active/history enrichment)
		"ip":                           types.StringType,
		"status":                       types.StringType,
		"uptime":                       types.Int64Type,
		"first_seen":                   types.Int64Type,
		"last_seen":                    types.Int64Type,
		"is_wired":                     types.BoolType,
		"is_guest":                     types.BoolType,
		"authorized":                   types.BoolType,
		"oui":                          types.StringType,
		"ap_mac":                       types.StringType,
		"channel":                      types.Int64Type,
		"radio":                        types.StringType,
		"radio_name":                   types.StringType,
		"essid":                        types.StringType,
		"bssid":                        types.StringType,
		"signal":                       types.Int64Type,
		"rssi":                         types.Int64Type,
		"noise":                        types.Int64Type,
		"tx_rate":                      types.Int64Type,
		"rx_rate":                      types.Int64Type,
		"tx_bytes":                     types.Int64Type,
		"rx_bytes":                     types.Int64Type,
		"wired_rate_mbps":              types.Int64Type,
		"sw_port":                      types.Int64Type,
		"last_uplink_mac":              types.StringType,
		"last_uplink_name":             types.StringType,
		"network_name":                 types.StringType,
		"last_connection_network_id":   types.StringType,
		"last_connection_network_name": types.StringType,
	}
}

func clientListEntrySchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		// From Client (user REST API)
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the client.",
			Computed:            true,
		},
		"mac": schema.StringAttribute{
			MarkdownDescription: "The MAC address of the client.",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the client.",
			Computed:            true,
		},
		"display_name": schema.StringAttribute{
			MarkdownDescription: "The display name of the client.",
			Computed:            true,
		},
		"group_id": schema.StringAttribute{
			MarkdownDescription: "The user group ID for the client.",
			Computed:            true,
		},
		"note": schema.StringAttribute{
			MarkdownDescription: "A note with additional information for the client.",
			Computed:            true,
		},
		"fixed_ip": schema.StringAttribute{
			MarkdownDescription: "A fixed IPv4 address for this client.",
			Computed:            true,
		},
		"fixed_ap_mac": schema.StringAttribute{
			MarkdownDescription: "The MAC address of the access point to which this client is fixed.",
			Computed:            true,
		},
		"network_id": schema.StringAttribute{
			MarkdownDescription: "The network ID for this client.",
			Computed:            true,
		},
		"network_members_group_ids": schema.ListAttribute{
			MarkdownDescription: "List of network member group IDs for this client.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"blocked": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is blocked from the network.",
			Computed:            true,
		},
		"local_dns_record": schema.StringAttribute{
			MarkdownDescription: "The local DNS record for this client.",
			Computed:            true,
		},
		"hostname": schema.StringAttribute{
			MarkdownDescription: "The hostname of the client.",
			Computed:            true,
		},

		// From ClientInfo (active/history enrichment)
		"ip": schema.StringAttribute{
			MarkdownDescription: "The IP address of the client.",
			Computed:            true,
		},
		"status": schema.StringAttribute{
			MarkdownDescription: "The connection status of the client.",
			Computed:            true,
		},
		"uptime": schema.Int64Attribute{
			MarkdownDescription: "The uptime of the client in seconds.",
			Computed:            true,
		},
		"first_seen": schema.Int64Attribute{
			MarkdownDescription: "Unix timestamp when the client was first seen.",
			Computed:            true,
		},
		"last_seen": schema.Int64Attribute{
			MarkdownDescription: "Unix timestamp when the client was last seen.",
			Computed:            true,
		},
		"is_wired": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is connected via wired connection.",
			Computed:            true,
		},
		"is_guest": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is a guest.",
			Computed:            true,
		},
		"authorized": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is authorized.",
			Computed:            true,
		},
		"oui": schema.StringAttribute{
			MarkdownDescription: "The OUI (vendor) of the client's MAC address.",
			Computed:            true,
		},
		"ap_mac": schema.StringAttribute{
			MarkdownDescription: "The MAC address of the access point the client is connected to.",
			Computed:            true,
		},
		"channel": schema.Int64Attribute{
			MarkdownDescription: "The WiFi channel the client is connected on.",
			Computed:            true,
		},
		"radio": schema.StringAttribute{
			MarkdownDescription: "The radio type (e.g., na, ng).",
			Computed:            true,
		},
		"radio_name": schema.StringAttribute{
			MarkdownDescription: "The radio name (e.g., wifi0, wifi1).",
			Computed:            true,
		},
		"essid": schema.StringAttribute{
			MarkdownDescription: "The ESSID (network name) the client is connected to.",
			Computed:            true,
		},
		"bssid": schema.StringAttribute{
			MarkdownDescription: "The BSSID of the access point.",
			Computed:            true,
		},
		"signal": schema.Int64Attribute{
			MarkdownDescription: "The signal strength in dBm.",
			Computed:            true,
		},
		"rssi": schema.Int64Attribute{
			MarkdownDescription: "The RSSI value.",
			Computed:            true,
		},
		"noise": schema.Int64Attribute{
			MarkdownDescription: "The noise level in dBm.",
			Computed:            true,
		},
		"tx_rate": schema.Int64Attribute{
			MarkdownDescription: "The transmit rate in kbps.",
			Computed:            true,
		},
		"rx_rate": schema.Int64Attribute{
			MarkdownDescription: "The receive rate in kbps.",
			Computed:            true,
		},
		"tx_bytes": schema.Int64Attribute{
			MarkdownDescription: "Total bytes transmitted.",
			Computed:            true,
		},
		"rx_bytes": schema.Int64Attribute{
			MarkdownDescription: "Total bytes received.",
			Computed:            true,
		},
		"wired_rate_mbps": schema.Int64Attribute{
			MarkdownDescription: "The wired connection rate in Mbps.",
			Computed:            true,
		},
		"sw_port": schema.Int64Attribute{
			MarkdownDescription: "The switch port number the client is connected to.",
			Computed:            true,
		},
		"last_uplink_mac": schema.StringAttribute{
			MarkdownDescription: "The MAC address of the last uplink device.",
			Computed:            true,
		},
		"last_uplink_name": schema.StringAttribute{
			MarkdownDescription: "The name of the last uplink device.",
			Computed:            true,
		},
		"network_name": schema.StringAttribute{
			MarkdownDescription: "The network name for this client.",
			Computed:            true,
		},
		"last_connection_network_id": schema.StringAttribute{
			MarkdownDescription: "The network ID of the last connection.",
			Computed:            true,
		},
		"last_connection_network_name": schema.StringAttribute{
			MarkdownDescription: "The network name of the last connection.",
			Computed:            true,
		},
	}
}

func (d *clientListDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client_list"
}

func (d *clientListDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of clients (users) on the network with optional filtering. " +
			"Merges client configuration data with active and historical connection information " +
			"for network discovery within Terraform.",

		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to retrieve clients from.",
				Optional:            true,
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "Filter clients by network members group name.",
				Optional:            true,
			},
			"wired": schema.BoolAttribute{
				MarkdownDescription: "Filter clients by wired connection status.",
				Optional:            true,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Filter clients by blocked status.",
				Optional:            true,
			},
			"oui": schema.StringAttribute{
				MarkdownDescription: "Filter clients by OUI (vendor prefix).",
				Optional:            true,
			},
			"clients": schema.ListNestedAttribute{
				MarkdownDescription: "List of clients matching the specified filters.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: clientListEntrySchemaAttributes(),
				},
			},
		},
	}
}

func (d *clientListDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*Client); ok {
		d.client = client
	} else {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
	}
}

// resolveGroupID looks up a network members group by name and returns its ID.
// Results are cached per site to avoid repeated API calls.
func (d *clientListDataSource) resolveGroupID(
	ctx context.Context,
	site, groupName string,
) (string, error) {
	d.groupCacheMu.Lock()
	defer d.groupCacheMu.Unlock()

	if d.groupCache == nil {
		d.groupCache = make(map[string]map[string]string)
	}

	if siteCache, ok := d.groupCache[site]; ok {
		if id, ok := siteCache[groupName]; ok {
			return id, nil
		}
	}

	// Fetch all groups for this site and populate the cache.
	groups, err := d.client.ListNetworkMembersGroups(ctx, site)
	if err != nil {
		return "", fmt.Errorf("listing network members groups: %w", err)
	}

	siteCache := make(map[string]string, len(groups))
	for _, g := range groups {
		siteCache[g.Name] = g.ID
	}
	d.groupCache[site] = siteCache

	id, ok := siteCache[groupName]
	if !ok {
		return "", fmt.Errorf("network members group %q not found", groupName)
	}
	return id, nil
}

func (d *clientListDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data clientListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	// Build query parameters from filters.
	filters := make(map[string]string)

	if !data.Group.IsNull() {
		groupID, err := d.resolveGroupID(ctx, site, data.Group.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Resolving Group",
				err.Error(),
			)
			return
		}
		filters["network_members_group_ids"] = groupID
	}

	if !data.Wired.IsNull() {
		val := "false"
		if data.Wired.ValueBool() {
			val = "true"
		}
		filters["is_wired"] = val
	}

	if !data.Blocked.IsNull() {
		val := "false"
		if data.Blocked.ValueBool() {
			val = "true"
		}
		filters["blocked"] = val
	}

	if !data.OUI.IsNull() {
		filters["oui"] = data.OUI.ValueString()
	}

	// Fetch clients (users) from REST API.
	clients, err := d.client.ListClientFiltered(ctx, site, filters)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Clients",
			"Could not read clients: "+err.Error(),
		)
		return
	}

	// Fetch active client info and history for enrichment.
	// Build lookup maps keyed by user_id (which maps to Client.ID).
	infoByUserID := make(map[string]*gounifi.ClientInfo)

	activeClients, err := d.client.ListClientInfo(ctx, site)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Unable to Fetch Active Client Info",
			"Client info enrichment will be skipped: "+err.Error(),
		)
	} else {
		for i := range activeClients {
			ci := &activeClients[i]
			if ci.UserId != "" {
				infoByUserID[ci.UserId] = ci
			}
		}
	}

	// Also fetch history to cover offline clients.
	historyClients, err := d.client.ListClientHistory(ctx, site, 0)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Unable to Fetch Client History",
			"Historical client enrichment will be skipped: "+err.Error(),
		)
	} else {
		for i := range historyClients {
			ci := &historyClients[i]
			// Active data takes precedence — only add if not already present.
			if ci.UserId != "" {
				if _, exists := infoByUserID[ci.UserId]; !exists {
					infoByUserID[ci.UserId] = ci
				}
			}
		}
	}

	// Build result list merging Client + ClientInfo data.
	clientObjects := make([]basetypes.ObjectValue, 0, len(clients))
	for _, c := range clients {
		info := infoByUserID[c.ID]
		v := clientListEntryValues(&c, info)

		o, diags := types.ObjectValue(clientListEntryAttrTypes(), v)
		if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
			return
		}
		clientObjects = append(clientObjects, o)
	}

	clist, diags := types.ListValueFrom(
		ctx,
		types.ObjectType{AttrTypes: clientListEntryAttrTypes()},
		clientObjects,
	)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	data.Clients = clist
	data.Site = types.StringValue(site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// clientListEntryValues builds the attribute value map for a single client entry,
// merging Client data with optional ClientInfo enrichment.
func clientListEntryValues(c *gounifi.Client, info *gounifi.ClientInfo) map[string]attr.Value {
	v := map[string]attr.Value{
		// Client fields
		"id":           types.StringValue(c.ID),
		"mac":          types.StringValue(c.MAC),
		"name":         util.StringValueOrNull(c.Name),
		"display_name": util.StringValueOrNull(c.DisplayName),
		"group_id":     util.StringValueOrNull(c.UserGroupID),
		"note":         util.StringValueOrNull(c.Note),
		"fixed_ip":     util.StringValueOrNull(c.FixedIP),
		"fixed_ap_mac": util.StringValueOrNull(c.FixedApMAC),
		"hostname":     util.StringValueOrNull(c.Hostname),
		"blocked":      types.BoolPointerValue(c.Blocked),

		// Network ID: prefer virtual network override
		"network_id":       networkIDValue(c),
		"local_dns_record": util.StringValueOrNull(c.LocalDNSRecord),
	}

	// Network members group IDs
	v["network_members_group_ids"] = stringSliceToList(c.NetworkMembersGroupIDs)

	// ClientInfo enrichment fields — null when no info available
	if info != nil {
		v["ip"] = util.StringValueOrNull(info.IP)
		v["status"] = util.StringValueOrNull(info.Status)
		v["uptime"] = int64PointerValueOrNull(info.Uptime)
		v["first_seen"] = int64PointerValueOrNull(info.FirstSeen)
		v["last_seen"] = int64PointerValueOrNull(info.LastSeen)
		v["is_wired"] = types.BoolValue(info.IsWired)
		v["is_guest"] = types.BoolValue(info.IsGuest)
		v["authorized"] = types.BoolValue(info.Authorized)
		v["oui"] = util.StringValueOrNull(info.Oui)
		v["ap_mac"] = util.StringValueOrNull(info.ApMac)
		v["channel"] = int64PointerValueOrNull(info.Channel)
		v["radio"] = util.StringValueOrNull(info.Radio)
		v["radio_name"] = util.StringValueOrNull(info.RadioName)
		v["essid"] = util.StringValueOrNull(info.Essid)
		v["bssid"] = util.StringValueOrNull(info.Bssid)
		v["signal"] = int64PointerValueOrNull(info.Signal)
		v["rssi"] = int64PointerValueOrNull(info.Rssi)
		v["noise"] = int64PointerValueOrNull(info.Noise)
		v["tx_rate"] = int64PointerValueOrNull(info.TxRate)
		v["rx_rate"] = int64PointerValueOrNull(info.RxRate)
		v["tx_bytes"] = int64PointerValueOrNull(info.TxBytes)
		v["rx_bytes"] = int64PointerValueOrNull(info.RxBytes)
		v["wired_rate_mbps"] = int64PointerValueOrNull(info.WiredRateMbps)
		v["sw_port"] = int64PointerValueOrNull(info.SwPort)
		v["last_uplink_mac"] = util.StringValueOrNull(info.LastUplinkMac)
		v["last_uplink_name"] = util.StringValueOrNull(info.LastUplinkName)
		v["network_name"] = util.StringValueOrNull(info.NetworkName)
		v["last_connection_network_id"] = util.StringValueOrNull(info.LastConnectionNetworkId)
		v["last_connection_network_name"] = util.StringValueOrNull(info.LastConnectionNetworkName)
	} else {
		v["ip"] = types.StringNull()
		v["status"] = types.StringNull()
		v["uptime"] = types.Int64Null()
		v["first_seen"] = types.Int64Null()
		v["last_seen"] = types.Int64Null()
		v["is_wired"] = types.BoolNull()
		v["is_guest"] = types.BoolNull()
		v["authorized"] = types.BoolNull()
		v["oui"] = types.StringNull()
		v["ap_mac"] = types.StringNull()
		v["channel"] = types.Int64Null()
		v["radio"] = types.StringNull()
		v["radio_name"] = types.StringNull()
		v["essid"] = types.StringNull()
		v["bssid"] = types.StringNull()
		v["signal"] = types.Int64Null()
		v["rssi"] = types.Int64Null()
		v["noise"] = types.Int64Null()
		v["tx_rate"] = types.Int64Null()
		v["rx_rate"] = types.Int64Null()
		v["tx_bytes"] = types.Int64Null()
		v["rx_bytes"] = types.Int64Null()
		v["wired_rate_mbps"] = types.Int64Null()
		v["sw_port"] = types.Int64Null()
		v["last_uplink_mac"] = types.StringNull()
		v["last_uplink_name"] = types.StringNull()
		v["network_name"] = types.StringNull()
		v["last_connection_network_id"] = types.StringNull()
		v["last_connection_network_name"] = types.StringNull()
	}

	return v
}

func networkIDValue(c *gounifi.Client) types.String {
	if c.VirtualNetworkOverrideID != "" {
		return types.StringValue(c.VirtualNetworkOverrideID)
	}
	return util.StringValueOrNull(c.NetworkID)
}

func stringSliceToList(s []string) basetypes.ListValue {
	if len(s) == 0 {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, len(s))
	for i, v := range s {
		elements[i] = types.StringValue(v)
	}
	list, diags := types.ListValue(types.StringType, elements)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return list
}

func int64PointerValueOrNull(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
