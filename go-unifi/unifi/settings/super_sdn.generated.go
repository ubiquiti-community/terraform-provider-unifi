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

type SuperSdn struct {
	BaseSetting

	AuthToken       string `json:"auth_token,omitempty"`
	DeviceID        string `json:"device_id,omitempty"`
	Enabled         bool   `json:"enabled"`
	Migrated        bool   `json:"migrated"`
	SsoLoginEnabled string `json:"sso_login_enabled,omitempty"`
	UbicUuid        string `json:"ubic_uuid,omitempty"`
}

func (dst *SuperSdn) UnmarshalJSON(b []byte) error {
	type Alias SuperSdn
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
