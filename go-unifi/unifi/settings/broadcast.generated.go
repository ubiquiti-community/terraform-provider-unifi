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

type Broadcast struct {
	BaseSetting

	SoundAfterEnabled   bool   `json:"sound_after_enabled"`
	SoundAfterResource  string `json:"sound_after_resource,omitempty"`
	SoundAfterType      string `json:"sound_after_type,omitempty"` // sample|media
	SoundBeforeEnabled  bool   `json:"sound_before_enabled"`
	SoundBeforeResource string `json:"sound_before_resource,omitempty"`
	SoundBeforeType     string `json:"sound_before_type,omitempty"` // sample|media
}

func (dst *Broadcast) UnmarshalJSON(b []byte) error {
	type Alias Broadcast
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
