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

type ChannelPlan struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Date       string                  `json:"date"` // ^$|^(20[0-9]{2}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9])Z?$
	RadioTable []ChannelPlanRadioTable `json:"radio_table,omitempty"`
}

func (dst *ChannelPlan) UnmarshalJSON(b []byte) error {
	type Alias ChannelPlan
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

type ChannelPlanRadioTable struct {
	Channel     string `json:"channel,omitempty"`       // [0-9]|[1][0-4]|16|34|36|38|40|42|44|46|48|52|56|60|64|100|104|108|112|116|120|124|128|132|136|140|144|149|153|157|161|165|183|184|185|187|188|189|192|196|auto
	DeviceMAC   string `json:"device_mac,omitempty"`    // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	Name        string `json:"name,omitempty"`          // [a-z]*[0-9]*
	TxPower     string `json:"tx_power,omitempty"`      // [\d]+|auto
	TxPowerMode string `json:"tx_power_mode,omitempty"` // auto|medium|high|low|custom
	Width       *int64 `json:"width,omitempty"`         // 20|40|80|160
}

func (dst *ChannelPlanRadioTable) UnmarshalJSON(b []byte) error {
	type Alias ChannelPlanRadioTable
	aux := &struct {
		Channel types.Number `json:"channel"`
		TxPower types.Number `json:"tx_power"`

		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	if val, err := aux.Channel.Int64(); err == nil {
		dst.Channel = strconv.FormatInt(val, 10)
	}
	if val, err := aux.TxPower.Int64(); err == nil {
		dst.TxPower = strconv.FormatInt(val, 10)
	}

	return nil
}

func (c *ApiClient) listChannelPlan(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]ChannelPlan, error) {
	var respBody struct {
		Meta meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/channelplan", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getChannelPlan(
	ctx context.Context,
	site string,
	id string,
) (*ChannelPlan, error) {
	var respBody struct {
		Meta meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/channelplan/%s", site, id),
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

func (c *ApiClient) deleteChannelPlan(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/channelplan/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createChannelPlan(
	ctx context.Context,
	site string,
	d *ChannelPlan,
) (*ChannelPlan, error) {
	var respBody struct {
		Meta meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/channelplan", site),
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

func (c *ApiClient) updateChannelPlan(
	ctx context.Context,
	site string,
	d *ChannelPlan,
) (*ChannelPlan, error) {
	var respBody struct {
		Meta meta          `json:"meta"`
		Data []ChannelPlan `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/channelplan/%s", site, d.ID),
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
