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

type DHCPOption struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Code   string `json:"code,omitempty"` // ^(?!(?:15|42|43|44|51|66|67|252)$)([7-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-4])$
	Name   string `json:"name,omitempty"` // ^[A-Za-z0-9-_]{1,25}$
	Signed bool   `json:"signed"`
	Type   string `json:"type,omitempty"`  // ^(boolean|hexarray|integer|ipaddress|macaddress|text)$
	Width  *int64 `json:"width,omitempty"` // ^(8|16|32)$
}

func (dst *DHCPOption) UnmarshalJSON(b []byte) error {
	type Alias DHCPOption
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

func (c *ApiClient) listDHCPOption(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]DHCPOption, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DHCPOption `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/dhcpoption", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getDHCPOption(
	ctx context.Context,
	site string,
	id string,
) (*DHCPOption, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DHCPOption `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/dhcpoption/%s", site, id),
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

func (c *ApiClient) deleteDHCPOption(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/dhcpoption/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createDHCPOption(
	ctx context.Context,
	site string,
	d *DHCPOption,
) (*DHCPOption, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DHCPOption `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/dhcpoption", site),
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

func (c *ApiClient) updateDHCPOption(
	ctx context.Context,
	site string,
	d *DHCPOption,
) (*DHCPOption, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DHCPOption `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/dhcpoption/%s", site, d.ID),
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
