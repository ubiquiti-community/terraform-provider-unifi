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

type Doh struct {
	BaseSetting

	CustomServers []SettingDohCustomServers `json:"custom_servers,omitempty"`
	ServerNames   []string                  `json:"server_names,omitempty"`
	State         string                    `json:"state,omitempty"` // off|auto|manual|custom
}

func (dst *Doh) UnmarshalJSON(b []byte) error {
	type Alias Doh
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

type SettingDohCustomServers struct {
	Enabled    bool   `json:"enabled"`
	SdnsStamp  string `json:"sdns_stamp,omitempty"`
	ServerName string `json:"server_name,omitempty"`
}

func (dst *SettingDohCustomServers) UnmarshalJSON(b []byte) error {
	type Alias SettingDohCustomServers
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}
