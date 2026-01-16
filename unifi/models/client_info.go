package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var (
	_ basetypes.ObjectTypable  = ClientInfoObjectType{}
	_ basetypes.ObjectValuable = ClientInfoObjectValue{}
)

// ClientInfoObjectType is a custom object type for client information.
type ClientInfoObjectType struct {
	basetypes.ObjectType
}

// Equal returns true if the given type is equivalent.
func (t ClientInfoObjectType) Equal(o attr.Type) bool {
	other, ok := o.(ClientInfoObjectType)
	if !ok {
		return false
	}
	return t.ObjectType.Equal(other.ObjectType)
}

// String returns a human-readable string representation.
func (t ClientInfoObjectType) String() string {
	return "ClientInfoObjectType"
}

// ValueFromObject creates a ClientInfoObjectValue from an ObjectValue.
func (t ClientInfoObjectType) ValueFromObject(
	ctx context.Context,
	in basetypes.ObjectValue,
) (basetypes.ObjectValuable, diag.Diagnostics) {
	return ClientInfoObjectValue{
		Object: in,
	}, nil
}

// ClientInfoObjectValue is a custom object value for client information.
type ClientInfoObjectValue struct {
	types.Object
}

// Type returns the custom object type.
func (v ClientInfoObjectValue) Type(ctx context.Context) attr.Type {
	return ClientInfoObjectType{
		ObjectType: basetypes.ObjectType{
			AttrTypes: AttributeTypes(),
		},
	}
}

// Type returns the custom object type.
func (v ClientInfoObjectValue) ValueType(ctx context.Context) attr.Type {
	return ClientInfoObjectType{
		ObjectType: basetypes.ObjectType{
			AttrTypes: AttributeTypes(),
		},
	}
}

// Equal returns true if the given value is equivalent.
func (v ClientInfoObjectValue) Equal(o attr.Value) bool {
	other, ok := o.(ClientInfoObjectValue)
	if !ok {
		return false
	}
	return v.Object.Equal(other.Object)
}

// NewClientInfoObjectType creates a new instance of the custom object type.
func NewClientInfoObjectType() ClientInfoObjectType {
	return ClientInfoObjectType{
		ObjectType: basetypes.ObjectType{
			AttrTypes: AttributeTypes(),
		},
	}
}

func NewClientInfoObjectTypeFromData(data unifi.ClientInfo) {
}

// ClientInfoAttrTypes returns the attribute types for the client info object.
func AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                           types.StringType,
		"mac":                          types.StringType,
		"name":                         types.StringType,
		"display_name":                 types.StringType,
		"hostname":                     types.StringType,
		"ip":                           types.StringType,
		"fixed_ip":                     types.StringType,
		"network_id":                   types.StringType,
		"network_name":                 types.StringType,
		"usergroup_id":                 types.StringType,
		"blocked":                      types.BoolType,
		"is_guest":                     types.BoolType,
		"is_wired":                     types.BoolType,
		"authorized":                   types.BoolType,
		"status":                       types.StringType,
		"uptime":                       types.Int64Type,
		"first_seen":                   types.Int64Type,
		"last_seen":                    types.Int64Type,
		"oui":                          types.StringType,
		"local_dns_record":             types.StringType,
		"local_dns_record_enabled":     types.BoolType,
		"use_fixedip":                  types.BoolType,
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
		"last_uplink_remote_port":      types.Int64Type,
		"last_connection_network_id":   types.StringType,
		"last_connection_network_name": types.StringType,
	}
}

func Attributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
		"hostname": schema.StringAttribute{
			MarkdownDescription: "The hostname of the client.",
			Computed:            true,
		},
		"ip": schema.StringAttribute{
			MarkdownDescription: "The IP address of the client.",
			Computed:            true,
		},
		"fixed_ip": schema.StringAttribute{
			MarkdownDescription: "Fixed IPv4 address set for this client.",
			Computed:            true,
		},
		"network_id": schema.StringAttribute{
			MarkdownDescription: "The network ID for this client.",
			Computed:            true,
		},
		"network_name": schema.StringAttribute{
			MarkdownDescription: "The network name for this client.",
			Computed:            true,
		},
		"usergroup_id": schema.StringAttribute{
			MarkdownDescription: "The user group ID for the client.",
			Computed:            true,
		},
		"blocked": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is blocked.",
			Computed:            true,
		},
		"is_guest": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is a guest.",
			Computed:            true,
		},
		"is_wired": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is connected via wired connection.",
			Computed:            true,
		},
		"authorized": schema.BoolAttribute{
			MarkdownDescription: "Whether the client is authorized.",
			Computed:            true,
		},
		"status": schema.StringAttribute{
			MarkdownDescription: "The status of the client.",
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
		"oui": schema.StringAttribute{
			MarkdownDescription: "The OUI (vendor) of the client's MAC address.",
			Computed:            true,
		},
		"local_dns_record": schema.StringAttribute{
			MarkdownDescription: "The local DNS record for this client.",
			Computed:            true,
		},
		"local_dns_record_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether local DNS record is enabled for this client.",
			Computed:            true,
		},
		"use_fixedip": schema.BoolAttribute{
			MarkdownDescription: "Whether this client uses a fixed IP.",
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
		"last_uplink_remote_port": schema.Int64Attribute{
			MarkdownDescription: "The remote port of the last uplink device.",
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

// ClientInfoSchemaAttributes returns the schema attributes for client info.
func ClientInfoDataSourceSchema() map[string]schema.Attribute {
	attrs := Attributes()
	attrs["site"] = schema.StringAttribute{
		MarkdownDescription: "The name of the site to retrieve the client from.",
		Optional:            true,
		Computed:            true,
	}
	attrs["mac"] = schema.StringAttribute{
		MarkdownDescription: "The MAC address of the client to retrieve.",
		Required:            true,
	}
	return attrs
}

// ClientInfoSchemaAttributes returns the schema attributes for client info.
func ClientInfoListAttribute() schema.Attribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "List of active clients.",
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: Attributes(),
			CustomType: ClientInfoObjectType{
				ObjectType: types.ObjectType{
					AttrTypes: AttributeTypes(),
				},
			},
		},
	}
}

func ClientListValue(
	ctx context.Context,
	clientInfoList unifi.ClientList,
	target *types.List,
) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// Build the list of client objects
	clientObjects := make([]ClientInfoObjectValue, 0, len(clientInfoList))

	for _, c := range clientInfoList {
		var clientObj ClientInfoObjectValue
		diags := ClientInfoValue(ctx, &c, &clientObj)
		diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}
		clientObjects = append(clientObjects, clientObj)
	}

	elementType := basetypes.ObjectType{
		AttrTypes: AttributeTypes(),
	}
	if clientsList, diags := types.ListValueFrom(ctx, elementType, clientObjects); diags.HasError() {
		diagnostics.Append(diags...)
	} else {
		*target = clientsList
	}

	return diagnostics
}

func ClientInfoAttrValues(
	ctx context.Context,
	clientInfo *unifi.ClientInfo,
) map[string]attr.Value {
	return map[string]attr.Value{
		"id":                         util.StringValueOrNull(clientInfo.Id),
		"mac":                        util.StringValueOrNull(clientInfo.Mac),
		"name":                       util.StringValueOrNull(clientInfo.Name),
		"display_name":               util.StringValueOrNull(clientInfo.DisplayName),
		"hostname":                   util.StringValueOrNull(clientInfo.Hostname),
		"ip":                         util.StringValueOrNull(clientInfo.IP),
		"fixed_ip":                   util.StringValueOrNull(clientInfo.FixedIP),
		"network_id":                 util.StringValueOrNull(clientInfo.NetworkId),
		"network_name":               util.StringValueOrNull(clientInfo.NetworkName),
		"usergroup_id":               util.StringValueOrNull(clientInfo.UsergroupId),
		"blocked":                    types.BoolValue(clientInfo.Blocked),
		"is_guest":                   types.BoolValue(clientInfo.IsGuest),
		"is_wired":                   types.BoolValue(clientInfo.IsWired),
		"authorized":                 types.BoolValue(clientInfo.Authorized),
		"status":                     util.StringValueOrNull(clientInfo.Status),
		"uptime":                     types.Int64PointerValue(clientInfo.Uptime),
		"first_seen":                 types.Int64PointerValue(clientInfo.FirstSeen),
		"last_seen":                  types.Int64PointerValue(clientInfo.LastSeen),
		"oui":                        util.StringValueOrNull(clientInfo.Oui),
		"local_dns_record":           util.StringValueOrNull(clientInfo.LocalDNSRecord),
		"local_dns_record_enabled":   types.BoolValue(clientInfo.LocalDNSRecordEnabled),
		"use_fixedip":                types.BoolValue(clientInfo.UseFixedip),
		"ap_mac":                     util.StringValueOrNull(clientInfo.ApMac),
		"channel":                    types.Int64PointerValue(clientInfo.Channel),
		"radio":                      util.StringValueOrNull(clientInfo.Radio),
		"radio_name":                 util.StringValueOrNull(clientInfo.RadioName),
		"essid":                      util.StringValueOrNull(clientInfo.Essid),
		"bssid":                      util.StringValueOrNull(clientInfo.Bssid),
		"signal":                     types.Int64PointerValue(clientInfo.Signal),
		"rssi":                       types.Int64PointerValue(clientInfo.Rssi),
		"noise":                      types.Int64PointerValue(clientInfo.Noise),
		"tx_rate":                    types.Int64PointerValue(clientInfo.TxRate),
		"rx_rate":                    types.Int64PointerValue(clientInfo.RxRate),
		"tx_bytes":                   types.Int64PointerValue(clientInfo.TxBytes),
		"rx_bytes":                   types.Int64PointerValue(clientInfo.RxBytes),
		"wired_rate_mbps":            types.Int64PointerValue(clientInfo.WiredRateMbps),
		"sw_port":                    types.Int64PointerValue(clientInfo.SwPort),
		"last_uplink_mac":            util.StringValueOrNull(clientInfo.LastUplinkMac),
		"last_uplink_name":           util.StringValueOrNull(clientInfo.LastUplinkName),
		"last_uplink_remote_port":    types.Int64PointerValue(clientInfo.LastUplinkRemotePort),
		"last_connection_network_id": util.StringValueOrNull(clientInfo.LastConnectionNetworkId),
		"last_connection_network_name": util.StringValueOrNull(
			clientInfo.LastConnectionNetworkName,
		),
	}
}

func ClientInfoValue(
	ctx context.Context,
	clientInfo *unifi.ClientInfo,
	target *ClientInfoObjectValue,
) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	clientObj := ClientInfoAttrValues(ctx, clientInfo)

	if objValue, diags := types.ObjectValue(AttributeTypes(), clientObj); diags.HasError() {
		diagnostics.Append(diags...)
	} else {
		*target = ClientInfoObjectValue{
			Object: objValue,
		}
	}

	return diagnostics
}
