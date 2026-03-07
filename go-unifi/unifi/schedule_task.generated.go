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

type ScheduleTask struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Action          string                       `json:"action,omitempty"` // upgrade
	CronExpr        string                       `json:"cron_expr,omitempty"`
	ExecuteOnlyOnce bool                         `json:"execute_only_once"`
	Name            string                       `json:"name,omitempty"`
	UpgradeTargets  []ScheduleTaskUpgradeTargets `json:"upgrade_targets,omitempty"`
}

func (dst *ScheduleTask) UnmarshalJSON(b []byte) error {
	type Alias ScheduleTask
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

type ScheduleTaskUpgradeTargets struct {
	MAC string `json:"mac,omitempty"` // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
}

func (dst *ScheduleTaskUpgradeTargets) UnmarshalJSON(b []byte) error {
	type Alias ScheduleTaskUpgradeTargets
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

func (c *ApiClient) listScheduleTask(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]ScheduleTask, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []ScheduleTask `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/scheduletask", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getScheduleTask(
	ctx context.Context,
	site string,
	id string,
) (*ScheduleTask, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []ScheduleTask `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/scheduletask/%s", site, id),
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

func (c *ApiClient) deleteScheduleTask(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/scheduletask/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createScheduleTask(
	ctx context.Context,
	site string,
	d *ScheduleTask,
) (*ScheduleTask, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []ScheduleTask `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/scheduletask", site),
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

func (c *ApiClient) updateScheduleTask(
	ctx context.Context,
	site string,
	d *ScheduleTask,
) (*ScheduleTask, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []ScheduleTask `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/scheduletask/%s", site, d.ID),
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
