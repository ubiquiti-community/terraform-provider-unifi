// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListDHCPOption(ctx context.Context, site string) ([]DHCPOption, error) {
	return c.listDHCPOption(ctx, site)
}

func (c *ApiClient) GetDHCPOption(ctx context.Context, site, id string) (*DHCPOption, error) {
	return c.getDHCPOption(ctx, site, id)
}

func (c *ApiClient) DeleteDHCPOption(ctx context.Context, site, id string) error {
	return c.deleteDHCPOption(ctx, site, id)
}

func (c *ApiClient) CreateDHCPOption(
	ctx context.Context,
	site string,
	d *DHCPOption,
) (*DHCPOption, error) {
	return c.createDHCPOption(ctx, site, d)
}

func (c *ApiClient) UpdateDHCPOption(
	ctx context.Context,
	site string,
	d *DHCPOption,
) (*DHCPOption, error) {
	return c.updateDHCPOption(ctx, site, d)
}
