// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

type BGPConfig struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Config           string `json:"frr_bgpd_config,omitempty"`
	Description      string `json:"description,omitempty"` // .{0,128}
	Enabled          bool   `json:"enabled"`
	UploadedFileName string `json:"uploaded_file_name,omitempty"` // .{0,256}
}

func (dst *BGPConfig) UnmarshalJSON(b []byte) error {
	type Alias BGPConfig
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

func (c *ApiClient) getBGPConfig(
	ctx context.Context,
	site string,
) (*BGPConfig, error) {
	var respBody []BGPConfig
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/bgp/config", site),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}
	if len(respBody) != 1 {
		return nil, &NotFoundError{}
	}

	d := respBody[0]
	return &d, nil
}

func (c *ApiClient) deleteBGPConfig(
	ctx context.Context,
	site string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("v2/api/site/%s/bgp/config", site),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createBGPConfig(
	ctx context.Context,
	site string,
	d *BGPConfig,
) (*BGPConfig, error) {
	var respBody BGPConfig

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("v2/api/site/%s/bgp/config", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
