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

type RadioAi struct {
	BaseSetting

	AutoAdjustChannelsToCountry bool                                `json:"auto_adjust_channels_to_country"`
	AutoChannelPresetsType      string                              `json:"auto_channel_presets_type,omitempty"` // maximum_speed|conservative|custom
	Channels6E                  []int64                             `json:"channels_6e,omitempty"`               // [1-9]|[1-2][0-9]|3[3-9]|[4-5][0-9]|6[0-1]|6[5-9]|[7-8][0-9]|9[0-3]|9[7-9]|1[0-1][0-9]|12[0-5]|129|1[3-4][0-9]|15[0-7]|16[1-9]|1[7-8][0-9]|19[3-9]|2[0-1][0-9]|22[0-1]|22[5-9]|233
	ChannelsBlacklist           []SettingRadioAiChannelsBlacklist   `json:"channels_blacklist,omitempty"`
	ChannelsNa                  []int64                             `json:"channels_na,omitempty"` // 34|36|38|40|42|44|46|48|52|56|60|64|100|104|108|112|116|120|124|128|132|136|140|144|149|153|157|161|165|169
	ChannelsNg                  []int64                             `json:"channels_ng,omitempty"` // 1|2|3|4|5|6|7|8|9|10|11|12|13|14
	CronExpr                    string                              `json:"cron_expr,omitempty"`
	Default                     bool                                `json:"default"`
	Enabled                     bool                                `json:"enabled"`
	ExcludeDevices              []string                            `json:"exclude_devices,omitempty"`       // ([0-9a-z]{2}:){5}[0-9a-z]{2}
	HighPriorityDevices         []string                            `json:"high_priority_devices,omitempty"` // ([0-9a-z]{2}:){5}[0-9a-z]{2}
	HtModesNa                   []int64                             `json:"ht_modes_na,omitempty"`           // ^(20|40|80|160)$
	HtModesNg                   []int64                             `json:"ht_modes_ng,omitempty"`           // ^(20|40)$
	Optimize                    []string                            `json:"optimize,omitempty"`              // channel|power
	Radios                      []string                            `json:"radios,omitempty"`                // na|ng|6e
	RadiosConfiguration         []SettingRadioAiRadiosConfiguration `json:"radios_configuration,omitempty"`
	SettingPreference           string                              `json:"setting_preference,omitempty"` // auto|manual
	UseXy                       bool                                `json:"useXY"`
}

func (dst *RadioAi) UnmarshalJSON(b []byte) error {
	type Alias RadioAi
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

type SettingRadioAiChannelsBlacklist struct {
	Channel      *int64 `json:"channel,omitempty"`       // [1-9]|[1-9][0-9]|1[0-9][0-9]|2[0-9]|2[0-1][0-9]|22[0-1]|22[5-9]|233
	ChannelWidth *int64 `json:"channel_width,omitempty"` // 20|40|80|160|240|320
	Radio        string `json:"radio,omitempty"`         // na|ng|6e
}

func (dst *SettingRadioAiChannelsBlacklist) UnmarshalJSON(b []byte) error {
	type Alias SettingRadioAiChannelsBlacklist
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

type SettingRadioAiRadiosConfiguration struct {
	ChannelWidth *int64 `json:"channel_width,omitempty"` // 20|40|80|160|320
	Dfs          bool   `json:"dfs"`
	Radio        string `json:"radio,omitempty"` // na|ng|6e
}

func (dst *SettingRadioAiRadiosConfiguration) UnmarshalJSON(b []byte) error {
	type Alias SettingRadioAiRadiosConfiguration
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
