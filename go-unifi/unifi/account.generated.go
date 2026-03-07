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

type Account struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	FilterIDs        []string `json:"filter_ids,omitempty"`
	IP               string   `json:"ip,omitempty"`   // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	Name             string   `json:"name,omitempty"` // ^[^"' ]+$
	NetworkID        string   `json:"networkconf_id,omitempty"`
	TunnelConfigType string   `json:"tunnel_config_type,omitempty"` // vpn|802.1x|custom
	TunnelMediumType *int64   `json:"tunnel_medium_type,omitempty"` // [1-9]|1[0-5]|^$
	TunnelType       *int64   `json:"tunnel_type,omitempty"`        // [1-9]|1[0-3]|^$
	UlpUserID        string   `json:"ulp_user_id,omitempty"`
	VLAN             *int64   `json:"vlan,omitempty"` // [2-9]|[1-9][0-9]{1,2}|[1-3][0-9]{3}|400[0-9]|^$
	XPassword        string   `json:"x_password,omitempty"`
}

func (dst *Account) UnmarshalJSON(b []byte) error {
	type Alias Account
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

func (c *ApiClient) listAccount(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]Account, error) {
	var respBody struct {
		Meta meta      `json:"meta"`
		Data []Account `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/account", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getAccount(
	ctx context.Context,
	site string,
	id string,
) (*Account, error) {
	var respBody struct {
		Meta meta      `json:"meta"`
		Data []Account `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/account/%s", site, id),
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

func (c *ApiClient) deleteAccount(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/account/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createAccount(
	ctx context.Context,
	site string,
	d *Account,
) (*Account, error) {
	var respBody struct {
		Meta meta      `json:"meta"`
		Data []Account `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/account", site),
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

func (c *ApiClient) updateAccount(
	ctx context.Context,
	site string,
	d *Account,
) (*Account, error) {
	var respBody struct {
		Meta meta      `json:"meta"`
		Data []Account `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/account/%s", site, d.ID),
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
