// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
	_ strings.Builder
)

type VirtualDevice struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	HeightInMeters float64 `json:"heightInMeters,omitempty"`
	Locked         bool    `json:"locked"`
	MapID          string  `json:"map_id,omitempty"`
	Type           string  `json:"type,omitempty"` // uap|usg|usw
	X              string  `json:"x,omitempty"`
	Y              string  `json:"y,omitempty"`
}

func (dst *VirtualDevice) UnmarshalJSON(b []byte) error {
	type Alias VirtualDevice
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

func (c *ApiClient) listVirtualDevice(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]VirtualDevice, error) {
	var respBody struct {
		Meta meta            `json:"meta"`
		Data []VirtualDevice `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/virtualdevice", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getVirtualDevice(
	ctx context.Context,
	site string,
	id string,
) (*VirtualDevice, error) {
	var respBody struct {
		Meta meta            `json:"meta"`
		Data []VirtualDevice `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/virtualdevice/%s", site, id),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}
	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *ApiClient) deleteVirtualDevice(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/virtualdevice/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createVirtualDevice(
	ctx context.Context,
	site string,
	d *VirtualDevice,
) (*VirtualDevice, error) {
	var respBody struct {
		Meta meta            `json:"meta"`
		Data []VirtualDevice `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/virtualdevice", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}

func (c *ApiClient) updateVirtualDevice(
	ctx context.Context,
	site string,
	d *VirtualDevice,
) (*VirtualDevice, error) {
	var respBody struct {
		Meta meta            `json:"meta"`
		Data []VirtualDevice `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/virtualdevice/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}
