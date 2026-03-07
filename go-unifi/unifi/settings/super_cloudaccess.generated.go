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

type SuperCloudaccess struct {
	BaseSetting

	DeviceAuth      string `json:"device_auth,omitempty"`
	DeviceID        string `json:"device_id,omitempty"`
	Enabled         bool   `json:"enabled"`
	UbicUuid        string `json:"ubic_uuid,omitempty"`
	XCertificateArn string `json:"x_certificate_arn,omitempty"`
	XCertificatePem string `json:"x_certificate_pem,omitempty"`
	XPrivateKey     string `json:"x_private_key,omitempty"`
}

func (dst *SuperCloudaccess) UnmarshalJSON(b []byte) error {
	type Alias SuperCloudaccess
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
