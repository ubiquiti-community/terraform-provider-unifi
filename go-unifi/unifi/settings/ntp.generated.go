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

type Ntp struct {
	BaseSetting

	NtpServer1        string `json:"ntp_server_1,omitempty"`
	NtpServer2        string `json:"ntp_server_2,omitempty"`
	NtpServer3        string `json:"ntp_server_3,omitempty"`
	NtpServer4        string `json:"ntp_server_4,omitempty"`
	SettingPreference string `json:"setting_preference,omitempty"` // auto|manual
}

func (dst *Ntp) UnmarshalJSON(b []byte) error {
	type Alias Ntp
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
