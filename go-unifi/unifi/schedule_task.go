// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListScheduleTask(ctx context.Context, site string) ([]ScheduleTask, error) {
	return c.listScheduleTask(ctx, site)
}

func (c *ApiClient) GetScheduleTask(ctx context.Context, site, id string) (*ScheduleTask, error) {
	return c.getScheduleTask(ctx, site, id)
}

func (c *ApiClient) DeleteScheduleTask(ctx context.Context, site, id string) error {
	return c.deleteScheduleTask(ctx, site, id)
}

func (c *ApiClient) CreateScheduleTask(
	ctx context.Context,
	site string,
	d *ScheduleTask,
) (*ScheduleTask, error) {
	return c.createScheduleTask(ctx, site, d)
}

func (c *ApiClient) UpdateScheduleTask(
	ctx context.Context,
	site string,
	d *ScheduleTask,
) (*ScheduleTask, error) {
	return c.updateScheduleTask(ctx, site, d)
}
