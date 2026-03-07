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

type Netflow struct {
	BaseSetting

	AutoEngineIDEnabled bool     `json:"auto_engine_id_enabled"`
	Enabled             bool     `json:"enabled"`
	EngineID            *int64   `json:"engine_id,omitempty"` // ^$|[1-9][0-9]*
	ExportFrequency     *int64   `json:"export_frequency,omitempty"`
	NetworkIDs          []string `json:"network_ids,omitempty"`
	Port                *int64   `json:"port,omitempty"` // 102[4-9]|10[3-9][0-9]|1[1-9][0-9]{2}|[2-9][0-9]{3}|[1-5][0-9]{4}|[6][0-4][0-9]{3}|[6][5][0-4][0-9]{2}|[6][5][5][0-2][0-9]|[6][5][5][3][0-5]
	RefreshRate         *int64   `json:"refresh_rate,omitempty"`
	SamplingMode        string   `json:"sampling_mode,omitempty"` // off|hash|random|deterministic
	SamplingRate        *int64   `json:"sampling_rate,omitempty"` // [2-9]|[1-9][0-9]{1,3}|1[0-5][0-9]{3}|16[0-2][0-9]{2}|163[0-7][0-9]|1638[0-3]|^$
	Server              string   `json:"server,omitempty"`        // .{0,252}[^\.]$
	Version             *int64   `json:"version,omitempty"`       // 5|9|10
}

func (dst *Netflow) UnmarshalJSON(b []byte) error {
	type Alias Netflow
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
