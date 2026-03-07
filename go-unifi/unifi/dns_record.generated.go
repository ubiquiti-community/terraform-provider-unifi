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

type DNSRecord struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Enabled    bool   `json:"enabled"`
	Key        string `json:"key,omitempty"`         // .{1,128}
	Port       *int64 `json:"port,omitempty"`        // [1-9][0-9]{0,4}
	Priority   int64  `json:"priority,omitempty"`    // .{1,128}
	RecordType string `json:"record_type,omitempty"` // A|AAAA|CNAME|MX|NS|PTR|SOA|SRV|TXT
	Ttl        int64  `json:"ttl,omitempty"`
	Value      string `json:"value,omitempty"` // .{1,256}
	Weight     int64  `json:"weight,omitempty"`
}

func (dst *DNSRecord) UnmarshalJSON(b []byte) error {
	type Alias DNSRecord
	aux := &struct {
		Priority types.Number `json:"priority"`
		Ttl      types.Number `json:"ttl"`
		Weight   types.Number `json:"weight"`

		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	if val, err := aux.Priority.Int64(); err == nil {
		dst.Priority = val
	}
	if val, err := aux.Ttl.Int64(); err == nil {
		dst.Ttl = val
	}
	if val, err := aux.Weight.Int64(); err == nil {
		dst.Weight = val
	}

	return nil
}

func (c *ApiClient) listDNSRecord(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]DNSRecord, error) {
	var respBody []DNSRecord

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/static-dns", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (c *ApiClient) getDNSRecord(
	ctx context.Context,
	site string,
	id string,
) (*DNSRecord, error) {
	respBody, err := c.listDNSRecord(ctx, site)
	if err != nil {
		return nil, err
	}

	if len(respBody) == 0 {
		return nil, &NotFoundError{}
	}

	for _, val := range respBody {
		if val.ID == id {
			return &val, nil
		}
	}

	return nil, &NotFoundError{}
}

func (c *ApiClient) deleteDNSRecord(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("v2/api/site/%s/static-dns/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createDNSRecord(
	ctx context.Context,
	site string,
	d *DNSRecord,
) (*DNSRecord, error) {
	var respBody DNSRecord

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("v2/api/site/%s/static-dns", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) updateDNSRecord(
	ctx context.Context,
	site string,
	d *DNSRecord,
) (*DNSRecord, error) {
	var respBody DNSRecord
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("v2/api/site/%s/static-dns/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
