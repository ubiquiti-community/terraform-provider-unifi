// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListChannelPlan(ctx context.Context, site string) ([]ChannelPlan, error) {
	return c.listChannelPlan(ctx, site)
}

func (c *ApiClient) GetChannelPlan(ctx context.Context, site, id string) (*ChannelPlan, error) {
	return c.getChannelPlan(ctx, site, id)
}

func (c *ApiClient) DeleteChannelPlan(ctx context.Context, site, id string) error {
	return c.deleteChannelPlan(ctx, site, id)
}

func (c *ApiClient) CreateChannelPlan(
	ctx context.Context,
	site string,
	d *ChannelPlan,
) (*ChannelPlan, error) {
	return c.createChannelPlan(ctx, site, d)
}

func (c *ApiClient) UpdateChannelPlan(
	ctx context.Context,
	site string,
	d *ChannelPlan,
) (*ChannelPlan, error) {
	return c.updateChannelPlan(ctx, site, d)
}
