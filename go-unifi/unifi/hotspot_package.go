// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListHotspotPackage(ctx context.Context, site string) ([]HotspotPackage, error) {
	return c.listHotspotPackage(ctx, site)
}

func (c *ApiClient) GetHotspotPackage(ctx context.Context, site, id string) (*HotspotPackage, error) {
	return c.getHotspotPackage(ctx, site, id)
}

func (c *ApiClient) DeleteHotspotPackage(ctx context.Context, site, id string) error {
	return c.deleteHotspotPackage(ctx, site, id)
}

func (c *ApiClient) CreateHotspotPackage(
	ctx context.Context,
	site string,
	d *HotspotPackage,
) (*HotspotPackage, error) {
	return c.createHotspotPackage(ctx, site, d)
}

func (c *ApiClient) UpdateHotspotPackage(
	ctx context.Context,
	site string,
	d *HotspotPackage,
) (*HotspotPackage, error) {
	return c.updateHotspotPackage(ctx, site, d)
}
