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

type Nat struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Description           string                `json:"description,omitempty"`
	DestinationFilter     *NatDestinationFilter `json:"destination_filter,omitempty"`
	Enabled               bool                  `json:"enabled"`
	Exclude               bool                  `json:"exclude"`
	IPAddress             string                `json:"ip_address,omitempty"`
	IPVersion             string                `json:"ip_version,omitempty"` // IPV4|IPV6
	InInterface           string                `json:"in_interface,omitempty"`
	IsPredefined          bool                  `json:"is_predefined"`
	Logging               bool                  `json:"logging"`
	OutInterface          string                `json:"out_interface,omitempty"`
	Port                  *int64                `json:"port,omitempty"` // [1-9][0-9]{0,4}
	PppoeUseBaseInterface bool                  `json:"pppoe_use_base_interface"`
	Protocol              string                `json:"protocol,omitempty"` // all|tcp|udp|tcp_udp
	RuleIndex             *int64                `json:"rule_index,omitempty"`
	SettingPreference     string                `json:"setting_preference,omitempty"` // auto|manual
	SourceFilter          *NatSourceFilter      `json:"source_filter,omitempty"`
	Type                  string                `json:"type,omitempty"` // DNAT|SNAT|MASQUERADE
}

func (dst *Nat) UnmarshalJSON(b []byte) error {
	type Alias Nat
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

type NatDestinationFilter struct {
	Address          string   `json:"address,omitempty"`
	FilterType       string   `json:"filter_type,omitempty"` // NONE|ADDRESS_AND_PORT|FIREWALL_GROUPS|NETWORK_CONF
	FirewallGroupIDs []string `json:"firewall_group_ids,omitempty"`
	InvertAddress    bool     `json:"invert_address"`
	InvertPort       bool     `json:"invert_port"`
	NetworkConfID    string   `json:"network_conf_id,omitempty"`
	Port             *int64   `json:"port,omitempty"` // [1-9][0-9]{0,4}
}

func (dst *NatDestinationFilter) UnmarshalJSON(b []byte) error {
	type Alias NatDestinationFilter
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

type NatSourceFilter struct {
	Address          string   `json:"address,omitempty"`
	FilterType       string   `json:"filter_type,omitempty"` // NONE|ADDRESS_AND_PORT|FIREWALL_GROUPS|NETWORK_CONF
	FirewallGroupIDs []string `json:"firewall_group_ids,omitempty"`
	InvertAddress    bool     `json:"invert_address"`
	InvertPort       bool     `json:"invert_port"`
	NetworkConfID    string   `json:"network_conf_id,omitempty"`
	Port             *int64   `json:"port,omitempty"` // [1-9][0-9]{0,4}
}

func (dst *NatSourceFilter) UnmarshalJSON(b []byte) error {
	type Alias NatSourceFilter
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

func (c *ApiClient) listNat(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]Nat, error) {
	var respBody []Nat

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/nat", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (c *ApiClient) getNat(
	ctx context.Context,
	site string,
	id string,
) (*Nat, error) {
	respBody, err := c.listNat(ctx, site)
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

func (c *ApiClient) deleteNat(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("v2/api/site/%s/nat/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createNat(
	ctx context.Context,
	site string,
	d *Nat,
) (*Nat, error) {
	var respBody Nat

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("v2/api/site/%s/nat", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) updateNat(
	ctx context.Context,
	site string,
	d *Nat,
) (*Nat, error) {
	var respBody Nat
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("v2/api/site/%s/nat/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
