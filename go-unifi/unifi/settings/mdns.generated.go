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

type Mdns struct {
	BaseSetting

	CustomServices     []SettingMdnsCustomServices     `json:"custom_services,omitempty"`
	Mode               string                          `json:"mode,omitempty"` // all|auto|custom
	PredefinedServices []SettingMdnsPredefinedServices `json:"predefined_services,omitempty"`
}

func (dst *Mdns) UnmarshalJSON(b []byte) error {
	type Alias Mdns
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

type SettingMdnsCustomServices struct {
	Address string `json:"address,omitempty"` // ^_[a-zA-Z0-9._-]+\._(tcp|udp)(\.local)?$
	Name    string `json:"name,omitempty"`
}

func (dst *SettingMdnsCustomServices) UnmarshalJSON(b []byte) error {
	type Alias SettingMdnsCustomServices
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

type SettingMdnsPredefinedServices struct {
	Code string `json:"code,omitempty"` // amazon_devices|android_tv_remote|apple_airDrop|apple_airPlay|apple_file_sharing|apple_iChat|apple_iTunes|aqara|bose|dns_service_discovery|ftp_servers|google_chromecast|homeKit|matter_network|philips_hue|printers|roku|scanners|sonos|spotify_connect|ssh_servers|time_capsule|web_servers|windows_file_sharing_samba
}

func (dst *SettingMdnsPredefinedServices) UnmarshalJSON(b []byte) error {
	type Alias SettingMdnsPredefinedServices
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
