// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
)

type TrafficFlow struct {
	BaseSetting

	EnabledAllowedTraffic        bool `json:"enabled_allowed_traffic"`
	GatewayDNSEnabled            bool `json:"gateway_dns_enabled"`
	UnifiDeviceManagementEnabled bool `json:"unifi_device_management_enabled"`
	UnifiServicesEnabled         bool `json:"unifi_services_enabled"`
}

func (dst *TrafficFlow) UnmarshalJSON(b []byte) error {
	type Alias TrafficFlow
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	// First unmarshal base setting
	if err := json.Unmarshal(b, &dst.BaseSetting); err != nil {
		return fmt.Errorf("unable to unmarshal base setting: %w", err)
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}
