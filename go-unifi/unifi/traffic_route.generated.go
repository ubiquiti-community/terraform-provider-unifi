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

type TrafficRoute struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Description       string                      `json:"description,omitempty"` // .{0,128}
	Domains           []string                    `json:"domains"`
	Enabled           bool                        `json:"enabled"`
	IPAddresses       []TrafficRouteIPAddresses   `json:"ip_addresses,omitempty"`
	IPRanges          []TrafficRouteIPRanges      `json:"ip_ranges,omitempty"`
	KillSwitchEnabled bool                        `json:"kill_switch_enabled"`
	MatchingTarget    string                      `json:"matching_target,omitempty"`
	NetworkID         string                      `json:"network_id,omitempty"`
	NextHop           string                      `json:"next_hop,omitempty"`
	Regions           []string                    `json:"regions,omitempty"`
	TargetDevices     []TrafficRouteTargetDevices `json:"target_devices,omitempty"`
}

func (dst *TrafficRoute) UnmarshalJSON(b []byte) error {
	type Alias TrafficRoute
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

type TrafficRouteIPAddresses struct {
	IPOrSubnet string   `json:"ip_or_subnet,omitempty"`
	IPVersion  string   `json:"ip_version,omitempty"` // BOTH|IPV4|IPV6
	PortRanges []string `json:"port_ranges,omitempty"`
	Ports      []int64  `json:"ports,omitempty"` // [1-9][0-9]{0,4}
}

func (dst *TrafficRouteIPAddresses) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteIPAddresses
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

type TrafficRouteIPRanges struct {
	IPStart   string `json:"ip_start,omitempty"`
	IPStop    string `json:"ip_stop,omitempty"`
	IPVersion string `json:"ip_version,omitempty"` // BOTH|IPV4|IPV6
}

func (dst *TrafficRouteIPRanges) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteIPRanges
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

type TrafficRouteTargetDevices struct {
	ClientMAC string `json:"client_mac,omitempty"`
	Type      string `json:"type,omitempty"`
}

func (dst *TrafficRouteTargetDevices) UnmarshalJSON(b []byte) error {
	type Alias TrafficRouteTargetDevices
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

func (c *ApiClient) listTrafficRoute(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]TrafficRoute, error) {
	var respBody []TrafficRoute

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/trafficroutes", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (c *ApiClient) getTrafficRoute(
	ctx context.Context,
	site string,
	id string,
) (*TrafficRoute, error) {
	respBody, err := c.listTrafficRoute(ctx, site)
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

func (c *ApiClient) deleteTrafficRoute(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("v2/api/site/%s/trafficroutes/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createTrafficRoute(
	ctx context.Context,
	site string,
	d *TrafficRoute,
) (*TrafficRoute, error) {
	var respBody TrafficRoute

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("v2/api/site/%s/trafficroutes", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) updateTrafficRoute(
	ctx context.Context,
	site string,
	d *TrafficRoute,
) (*TrafficRoute, error) {
	var respBody TrafficRoute
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("v2/api/site/%s/trafficroutes/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
