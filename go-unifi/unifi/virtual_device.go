// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListVirtualDevice(ctx context.Context, site string) ([]VirtualDevice, error) {
	return c.listVirtualDevice(ctx, site)
}

func (c *ApiClient) GetVirtualDevice(ctx context.Context, site, id string) (*VirtualDevice, error) {
	return c.getVirtualDevice(ctx, site, id)
}

func (c *ApiClient) DeleteVirtualDevice(ctx context.Context, site, id string) error {
	return c.deleteVirtualDevice(ctx, site, id)
}

func (c *ApiClient) CreateVirtualDevice(
	ctx context.Context,
	site string,
	d *VirtualDevice,
) (*VirtualDevice, error) {
	return c.createVirtualDevice(ctx, site, d)
}

func (c *ApiClient) UpdateVirtualDevice(
	ctx context.Context,
	site string,
	d *VirtualDevice,
) (*VirtualDevice, error) {
	return c.updateVirtualDevice(ctx, site, d)
}
