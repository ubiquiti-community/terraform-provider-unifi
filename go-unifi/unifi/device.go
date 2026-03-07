package unifi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

//go:generate go tool golang.org/x/tools/cmd/stringer -trimprefix DeviceState -type DeviceState
type DeviceState int64

const (
	DeviceStateUnknown          DeviceState = 0
	DeviceStateConnected        DeviceState = 1
	DeviceStatePending          DeviceState = 2
	DeviceStateFirmwareMismatch DeviceState = 3
	DeviceStateUpgrading        DeviceState = 4
	DeviceStateProvisioning     DeviceState = 5
	DeviceStateHeartbeatMissed  DeviceState = 6
	DeviceStateAdopting         DeviceState = 7
	DeviceStateDeleting         DeviceState = 8
	DeviceStateInformError      DeviceState = 9
	DeviceStateAdoptFailed      DeviceState = 10
	DeviceStateIsolated         DeviceState = 11
)

type DeviceLastConnection struct {
	MAC      string `json:"mac,omitempty"`
	LastSeen int64  `json:"last_seen,omitempty"`
}
type DevicePortTable struct {
	PortIdx             int64                `json:"port_idx,omitempty"`
	Media               string               `json:"media,omitempty"`
	PortPoe             bool                 `json:"port_poe,omitempty"`
	PoeCaps             int64                `json:"poe_caps,omitempty"`
	SpeedCaps           int64                `json:"speed_caps,omitempty"`
	LastConnection      DeviceLastConnection `json:"last_connection,omitempty"`
	OpMode              string               `json:"op_mode,omitempty"`
	Forward             string               `json:"forward,omitempty"`
	PoeMode             string               `json:"poe_mode,omitempty"`
	Anomalies           int64                `json:"anomalies,omitempty"`
	Autoneg             bool                 `json:"autoneg,omitempty"`
	Dot1XMode           string               `json:"dot1x_mode,omitempty"`
	Dot1XStatus         string               `json:"dot1x_status,omitempty"`
	Enable              bool                 `json:"enable,omitempty"`
	FlowctrlRx          bool                 `json:"flowctrl_rx,omitempty"`
	FlowctrlTx          bool                 `json:"flowctrl_tx,omitempty"`
	FullDuplex          bool                 `json:"full_duplex,omitempty"`
	IsUplink            bool                 `json:"is_uplink,omitempty"`
	Jumbo               bool                 `json:"jumbo,omitempty"`
	MacTableCount       int64                `json:"mac_table_count,omitempty"`
	PoeClass            string               `json:"poe_class,omitempty"`
	PoeCurrent          string               `json:"poe_current,omitempty"`
	PoeEnable           bool                 `json:"poe_enable,omitempty"`
	PoeGood             bool                 `json:"poe_good,omitempty"`
	PoePower            string               `json:"poe_power,omitempty"`
	PoeVoltage          string               `json:"poe_voltage,omitempty"`
	RxBroadcast         int64                `json:"rx_broadcast,omitempty"`
	RxBytes             int64                `json:"rx_bytes,omitempty"`
	RxDropped           int64                `json:"rx_dropped,omitempty"`
	RxErrors            int64                `json:"rx_errors,omitempty"`
	RxMulticast         int64                `json:"rx_multicast,omitempty"`
	RxPackets           int64                `json:"rx_packets,omitempty"`
	Satisfaction        int64                `json:"satisfaction,omitempty"`
	SatisfactionReason  int64                `json:"satisfaction_reason,omitempty"`
	Speed               int64                `json:"speed,omitempty"`
	StpPathcost         int64                `json:"stp_pathcost,omitempty"`
	StpState            string               `json:"stp_state,omitempty"`
	TxBroadcast         int64                `json:"tx_broadcast,omitempty"`
	TxBytes             int64                `json:"tx_bytes,omitempty"`
	TxDropped           int64                `json:"tx_dropped,omitempty"`
	TxErrors            int64                `json:"tx_errors,omitempty"`
	TxMulticast         int64                `json:"tx_multicast,omitempty"`
	TxPackets           int64                `json:"tx_packets,omitempty"`
	Up                  bool                 `json:"up,omitempty"`
	TxBytesR            float64              `json:"tx_bytes-r,omitempty"`
	RxBytesR            float64              `json:"rx_bytes-r,omitempty"`
	BytesR              float64              `json:"bytes-r,omitempty"`
	FlowControlEnabled  bool                 `json:"flow_control_enabled,omitempty"`
	NativeNetworkconfID string               `json:"native_networkconf_id,omitempty"`
	Name                string               `json:"name,omitempty"`
	SettingPreference   string               `json:"setting_preference,omitempty"`
	StormctrlBcastRate  int64                `json:"stormctrl_bcast_rate,omitempty"`
	StormctrlMcastRate  int64                `json:"stormctrl_mcast_rate,omitempty"`
	StormctrlUcastRate  int64                `json:"stormctrl_ucast_rate,omitempty"`
	TaggedVlanMgmt      string               `json:"tagged_vlan_mgmt,omitempty"`
	Masked              bool                 `json:"masked,omitempty"`
	AggregatedBy        bool                 `json:"aggregated_by,omitempty"`
}

func (dst *DevicePortTable) UnmarshalJSON(b []byte) error {
	type Alias DevicePortTable
	aux := &struct {
		PortIdx            types.Number `json:"port_idx,omitempty"`
		PoeCaps            types.Number `json:"poe_caps,omitempty"`
		SpeedCaps          types.Number `json:"speed_caps,omitempty"`
		Anomalies          types.Number `json:"anomalies,omitempty"`
		MacTableCount      types.Number `json:"mac_table_count,omitempty"`
		RxBroadcast        types.Number `json:"rx_broadcast,omitempty"`
		RxBytes            types.Number `json:"rx_bytes,omitempty"`
		RxDropped          types.Number `json:"rx_dropped,omitempty"`
		RxErrors           types.Number `json:"rx_errors,omitempty"`
		RxMulticast        types.Number `json:"rx_multicast,omitempty"`
		RxPackets          types.Number `json:"rx_packets,omitempty"`
		Satisfaction       types.Number `json:"satisfaction,omitempty"`
		SatisfactionReason types.Number `json:"satisfaction_reason,omitempty"`
		Speed              types.Number `json:"speed,omitempty"`
		StpPathcost        types.Number `json:"stp_pathcost,omitempty"`
		TxBroadcast        types.Number `json:"tx_broadcast,omitempty"`
		TxBytes            types.Number `json:"tx_bytes,omitempty"`
		TxDropped          types.Number `json:"tx_dropped,omitempty"`
		TxErrors           types.Number `json:"tx_errors,omitempty"`
		TxMulticast        types.Number `json:"tx_multicast,omitempty"`
		TxPackets          types.Number `json:"tx_packets,omitempty"`
		StormctrlBcastRate types.Number `json:"stormctrl_bcast_rate,omitempty"`
		StormctrlMcastRate types.Number `json:"stormctrl_mcast_rate,omitempty"`
		StormctrlUcastRate types.Number `json:"stormctrl_ucast_rate,omitempty"`

		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	if portIdx, err := aux.PortIdx.Int64(); err != nil {
		dst.PortIdx = portIdx
	}
	if poeCaps, err := aux.PoeCaps.Int64(); err != nil {
		dst.PoeCaps = poeCaps
	}
	if speedCaps, err := aux.SpeedCaps.Int64(); err != nil {
		dst.SpeedCaps = speedCaps
	}
	if anomalies, err := aux.Anomalies.Int64(); err != nil {
		dst.Anomalies = anomalies
	}
	if macTableCount, err := aux.MacTableCount.Int64(); err != nil {
		dst.MacTableCount = macTableCount
	}
	if rxBroadcast, err := aux.RxBroadcast.Int64(); err != nil {
		dst.RxBroadcast = rxBroadcast
	}
	if rxBytes, err := aux.RxBytes.Int64(); err != nil {
		dst.RxBytes = rxBytes
	}
	if rxDropped, err := aux.RxDropped.Int64(); err != nil {
		dst.RxDropped = rxDropped
	}
	if rxErrors, err := aux.RxErrors.Int64(); err != nil {
		dst.RxErrors = rxErrors
	}
	if rxMulticast, err := aux.RxMulticast.Int64(); err != nil {
		dst.RxMulticast = rxMulticast
	}
	if rxPackets, err := aux.RxPackets.Int64(); err != nil {
		dst.RxPackets = rxPackets
	}
	if satisfaction, err := aux.Satisfaction.Int64(); err != nil {
		dst.Satisfaction = satisfaction
	}
	if satisfactionReason, err := aux.SatisfactionReason.Int64(); err != nil {
		dst.SatisfactionReason = satisfactionReason
	}
	if speed, err := aux.Speed.Int64(); err != nil {
		dst.Speed = speed
	}
	if stpPathcost, err := aux.StpPathcost.Int64(); err != nil {
		dst.StpPathcost = stpPathcost
	}
	if txBroadcast, err := aux.TxBroadcast.Int64(); err != nil {
		dst.TxBroadcast = txBroadcast
	}
	if txBytes, err := aux.TxBytes.Int64(); err != nil {
		dst.TxBytes = txBytes
	}
	if txDropped, err := aux.TxDropped.Int64(); err != nil {
		dst.TxDropped = txDropped
	}
	if txErrors, err := aux.TxErrors.Int64(); err != nil {
		dst.TxErrors = txErrors
	}
	if txMulticast, err := aux.TxMulticast.Int64(); err != nil {
		dst.TxMulticast = txMulticast
	}
	if txPackets, err := aux.TxPackets.Int64(); err != nil {
		dst.TxPackets = txPackets
	}
	if stormctrlBcastRate, err := aux.StormctrlBcastRate.Int64(); err != nil {
		dst.StormctrlBcastRate = stormctrlBcastRate
	}
	if stormctrlMcastRate, err := aux.StormctrlMcastRate.Int64(); err != nil {
		dst.StormctrlMcastRate = stormctrlMcastRate
	}
	if stormctrlUcastRate, err := aux.StormctrlUcastRate.Int64(); err != nil {
		dst.StormctrlUcastRate = stormctrlUcastRate
	}

	return nil
}

func (c *ApiClient) ListDevice(ctx context.Context, site string) ([]Device, error) {
	return c.listDevice(ctx, site)
}

func (c *ApiClient) GetDeviceByMAC(ctx context.Context, site, mac string) (*Device, error) {
	return c.getDevice(ctx, site, mac)
}

func (c *ApiClient) DeleteDevice(ctx context.Context, site, id string) error {
	return c.deleteDevice(ctx, site, id)
}

func (c *ApiClient) CreateDevice(ctx context.Context, site string, d *Device) (*Device, error) {
	return c.createDevice(ctx, site, d)
}

func (c *ApiClient) UpdateDevice(ctx context.Context, site string, d *Device) (*Device, error) {
	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Device `json:"data"`
	}

	// Get the existing device to compare
	existing, err := c.getDevice(ctx, site, d.MAC)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing device: %w", err)
	}

	// Create a patch with only changed fields
	patch, err := getDeviceDiff(existing, d)
	if err != nil {
		return nil, fmt.Errorf("failed to create device diff: %w", err)
	}

	err = c.do(
		ctx,
		"PUT",
		fmt.Sprintf("api/s/%s/rest/device/%s", site, d.ID),
		patch,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}

// getDiff compares two values of the same type and returns a map containing only changed fields.
// It skips read-only fields specified in skipFields.
func getDiff[T any](original, target *T, skipFields ...string) (map[string]any, error) {
	// Marshal both to JSON then unmarshal to maps for comparison
	origJSON, err := json.Marshal(original)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal original: %w", err)
	}

	targetJSON, err := json.Marshal(target)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal target: %w", err)
	}

	var origMap map[string]any
	var targetMap map[string]any

	if err := json.Unmarshal(origJSON, &origMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal original: %w", err)
	}

	if err := json.Unmarshal(targetJSON, &targetMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal target: %w", err)
	}

	// Create skip set for O(1) lookup
	skipSet := make(map[string]bool, len(skipFields))
	for _, field := range skipFields {
		skipSet[field] = true
	}

	// Build patch with only changed fields
	patch := make(map[string]any)

	for key, targetValue := range targetMap {
		// Skip read-only fields
		if skipSet[key] {
			continue
		}

		origValue, exists := origMap[key]

		// Include if field doesn't exist in original or value changed
		if !exists || !deepEqualJSON(origValue, targetValue) {
			patch[key] = targetValue
		}
	}

	return patch, nil
}

// getDeviceDiff compares two Device objects and returns a map containing only changed fields.
func getDeviceDiff(original, target *Device) (map[string]any, error) {
	return getDiff(original, target, "_id", "site_id")
}

// deepEqualJSON compares two values for deep equality by comparing their JSON representations.
func deepEqualJSON(a, b any) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

func (c *ApiClient) GetDevice(ctx context.Context, site, id string) (*Device, error) {
	devices, err := c.ListDevice(ctx, site)
	if err != nil {
		return nil, err
	}

	for _, d := range devices {
		if d.ID == id {
			return &d, nil
		}
	}

	return nil, &NotFoundError{}
}

func (c *ApiClient) AdoptDevice(ctx context.Context, site, mac string) error {
	reqBody := struct {
		Cmd string `json:"cmd"`
		MAC string `json:"mac"`
	}{
		Cmd: "adopt",
		MAC: mac,
	}

	var respBody struct {
		Meta meta `json:"meta"`
	}

	err := c.do(ctx, "POST", fmt.Sprintf("api/s/%s/cmd/devmgr", site), reqBody, &respBody)
	if err != nil {
		return err
	}

	return nil
}

func (c *ApiClient) ForgetDevice(ctx context.Context, site, mac string) error {
	reqBody := struct {
		Cmd  string   `json:"cmd"`
		MACs []string `json:"macs"`
	}{
		Cmd:  "delete-device",
		MACs: []string{mac},
	}

	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Device `json:"data"`
	}

	err := c.do(ctx, "POST", fmt.Sprintf("api/s/%s/cmd/sitemgr", site), reqBody, &respBody)
	if err != nil {
		return err
	}

	return nil
}
