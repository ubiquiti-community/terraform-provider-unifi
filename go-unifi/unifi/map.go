// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListMap(ctx context.Context, site string) ([]Map, error) {
	return c.listMap(ctx, site)
}

func (c *ApiClient) GetMap(ctx context.Context, site, id string) (*Map, error) {
	return c.getMap(ctx, site, id)
}

func (c *ApiClient) DeleteMap(ctx context.Context, site, id string) error {
	return c.deleteMap(ctx, site, id)
}

func (c *ApiClient) CreateMap(ctx context.Context, site string, d *Map) (*Map, error) {
	return c.createMap(ctx, site, d)
}

func (c *ApiClient) UpdateMap(ctx context.Context, site string, d *Map) (*Map, error) {
	return c.updateMap(ctx, site, d)
}
