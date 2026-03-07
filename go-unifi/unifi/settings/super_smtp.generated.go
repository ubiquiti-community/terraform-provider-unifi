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

type SuperSmtp struct {
	BaseSetting

	Enabled   bool   `json:"enabled"`
	Host      string `json:"host,omitempty"`
	Port      *int64 `json:"port,omitempty"` // [1-9][0-9]{0,3}|[1-5][0-9]{4}|[6][0-4][0-9]{3}|[6][5][0-4][0-9]{2}|[6][5][5][0-2][0-9]|[6][5][5][3][0-5]|^$
	Sender    string `json:"sender,omitempty"`
	UseAuth   bool   `json:"use_auth"`
	UseSender bool   `json:"use_sender"`
	UseSsl    bool   `json:"use_ssl"`
	Username  string `json:"username,omitempty"`
	XPassword string `json:"x_password,omitempty"`
}

func (dst *SuperSmtp) UnmarshalJSON(b []byte) error {
	type Alias SuperSmtp
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
