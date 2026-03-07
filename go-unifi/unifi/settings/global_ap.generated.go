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

type GlobalAp struct {
	BaseSetting

	ApExclusions    []string `json:"ap_exclusions,omitempty"`    // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	NaChannelSize   *int64   `json:"na_channel_size,omitempty"`  // 20|40|80|160
	NaTxPower       *int64   `json:"na_tx_power,omitempty"`      // [0-9]|[1-4][0-9]
	NaTxPowerMode   string   `json:"na_tx_power_mode,omitempty"` // auto|medium|high|low|custom
	NgChannelSize   *int64   `json:"ng_channel_size,omitempty"`  // 20|40
	NgTxPower       *int64   `json:"ng_tx_power,omitempty"`      // [0-9]|[1-4][0-9]
	NgTxPowerMode   string   `json:"ng_tx_power_mode,omitempty"` // auto|medium|high|low|custom
	SixEChannelSize *int64   `json:"6e_channel_size,omitempty"`  // 20|40|80|160
	SixETxPower     *int64   `json:"6e_tx_power,omitempty"`      // [0-9]|[1-4][0-9]
	SixETxPowerMode string   `json:"6e_tx_power_mode,omitempty"` // auto|medium|high|low|custom
}

func (dst *GlobalAp) UnmarshalJSON(b []byte) error {
	type Alias GlobalAp
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
