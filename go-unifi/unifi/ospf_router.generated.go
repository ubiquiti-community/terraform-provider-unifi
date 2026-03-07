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

type OSPFRouter struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	AnnounceDefaultRoute                  bool                   `json:"announce_default_route"`
	Areas                                 []OSPFRouterAreas      `json:"areas,omitempty"`
	Enabled                               bool                   `json:"enabled"`
	Interfaces                            []OSPFRouterInterfaces `json:"interfaces,omitempty"`
	RedistributeConnectedRoutes           bool                   `json:"redistribute_connected_routes"`
	RedistributeConnectedRoutesMetricType string                 `json:"redistribute_connected_routes_metric_type,omitempty"`
	RedistributeStaticRoutes              bool                   `json:"redistribute_static_routes"`
	RedistributeStaticRoutesMetricType    string                 `json:"redistribute_static_routes_metric_type,omitempty"`
	RouterID                              string                 `json:"router_id,omitempty"`
}

func (dst *OSPFRouter) UnmarshalJSON(b []byte) error {
	type Alias OSPFRouter
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

type OSPFRouterAreas struct {
	AreaID     string   `json:"area_id,omitempty"`
	AreaType   string   `json:"area_type,omitempty"`
	Name       string   `json:"name,omitempty"`
	NetworkIDs []string `json:"network_ids,omitempty"`
}

func (dst *OSPFRouterAreas) UnmarshalJSON(b []byte) error {
	type Alias OSPFRouterAreas
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

type OSPFRouterInterfaces struct {
	AuthenticationType string `json:"authentication_type,omitempty"`
	Cost               string `json:"cost,omitempty"`
	DeadInterval       string `json:"dead_interval,omitempty"`
	HelloInterval      string `json:"hello_interval,omitempty"`
	NetworkID          string `json:"network_id,omitempty"`
	PassiveInterface   bool   `json:"passive_interface"`
	Priority           string `json:"priority,omitempty"`
}

func (dst *OSPFRouterInterfaces) UnmarshalJSON(b []byte) error {
	type Alias OSPFRouterInterfaces
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

func (c *ApiClient) listOSPFRouter(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]OSPFRouter, error) {
	var respBody []OSPFRouter

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/ospf/router", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (c *ApiClient) getOSPFRouter(
	ctx context.Context,
	site string,
	id string,
) (*OSPFRouter, error) {
	respBody, err := c.listOSPFRouter(ctx, site)
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

func (c *ApiClient) deleteOSPFRouter(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("v2/api/site/%s/ospf/router/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createOSPFRouter(
	ctx context.Context,
	site string,
	d *OSPFRouter,
) (*OSPFRouter, error) {
	var respBody OSPFRouter

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("v2/api/site/%s/ospf/router", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) updateOSPFRouter(
	ctx context.Context,
	site string,
	d *OSPFRouter,
) (*OSPFRouter, error) {
	var respBody OSPFRouter
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("v2/api/site/%s/ospf/router/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
