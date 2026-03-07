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

type DynamicDNS struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	CustomService string   `json:"custom_service,omitempty"` // ^[^"' ]+$
	HostName      string   `json:"host_name,omitempty"`      // ^[^"' ]+$
	Interface     string   `json:"interface,omitempty"`      // wan[2-9]?
	Login         string   `json:"login,omitempty"`          // ^[^"' ]+$
	Options       []string `json:"options,omitempty"`        // ^[^"' ]+$
	Server        string   `json:"server"`                   // ^[^"' ]+$|^$
	Service       string   `json:"service,omitempty"`        // afraid|changeip|cloudflare|cloudxns|ddnss|dhis|dnsexit|dnsomatic|dnspark|dnspod|dslreports|dtdns|duckdns|duiadns|dyn|dyndns|dynv6|easydns|freemyip|googledomains|loopia|namecheap|noip|nsupdate|ovh|sitelutions|spdyn|strato|tunnelbroker|zoneedit|cloudflare|custom
	XPassword     string   `json:"x_password,omitempty"`     // ^[^"' ]+$
}

func (dst *DynamicDNS) UnmarshalJSON(b []byte) error {
	type Alias DynamicDNS
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

func (c *ApiClient) listDynamicDNS(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]DynamicDNS, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DynamicDNS `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/dynamicdns", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getDynamicDNS(
	ctx context.Context,
	site string,
	id string,
) (*DynamicDNS, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DynamicDNS `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/dynamicdns/%s", site, id),
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

func (c *ApiClient) deleteDynamicDNS(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/dynamicdns/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createDynamicDNS(
	ctx context.Context,
	site string,
	d *DynamicDNS,
) (*DynamicDNS, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DynamicDNS `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/dynamicdns", site),
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

func (c *ApiClient) updateDynamicDNS(
	ctx context.Context,
	site string,
	d *DynamicDNS,
) (*DynamicDNS, error) {
	var respBody struct {
		Meta meta         `json:"meta"`
		Data []DynamicDNS `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/dynamicdns/%s", site, d.ID),
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
