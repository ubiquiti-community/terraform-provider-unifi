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

type Rsyslogd struct {
	BaseSetting

	Contents                    []string `json:"contents,omitempty"` // device|client|firewall_default_policy|triggers|updates|admin_activity|critical|security_detections|vpn
	Debug                       bool     `json:"debug"`
	Enabled                     bool     `json:"enabled"`
	IP                          string   `json:"ip,omitempty"`
	LogAllContents              bool     `json:"log_all_contents"`
	NetconsoleEnabled           bool     `json:"netconsole_enabled"`
	NetconsoleHost              string   `json:"netconsole_host,omitempty"`
	NetconsolePort              *int64   `json:"netconsole_port,omitempty"` // [1-9][0-9]{0,3}|[1-5][0-9]{4}|[6][0-4][0-9]{3}|[6][5][0-4][0-9]{2}|[6][5][5][0-2][0-9]|[6][5][5][3][0-5]
	Port                        *int64   `json:"port,omitempty"`            // [1-9][0-9]{0,3}|[1-5][0-9]{4}|[6][0-4][0-9]{3}|[6][5][0-4][0-9]{2}|[6][5][5][0-2][0-9]|[6][5][5][3][0-5]
	ThisController              bool     `json:"this_controller"`
	ThisControllerEncryptedOnly bool     `json:"this_controller_encrypted_only"`
}

func (dst *Rsyslogd) UnmarshalJSON(b []byte) error {
	type Alias Rsyslogd
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
