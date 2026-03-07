// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListHeatMapPoint(ctx context.Context, site string) ([]HeatMapPoint, error) {
	return c.listHeatMapPoint(ctx, site)
}

func (c *ApiClient) GetHeatMapPoint(ctx context.Context, site, id string) (*HeatMapPoint, error) {
	return c.getHeatMapPoint(ctx, site, id)
}

func (c *ApiClient) DeleteHeatMapPoint(ctx context.Context, site, id string) error {
	return c.deleteHeatMapPoint(ctx, site, id)
}

func (c *ApiClient) CreateHeatMapPoint(
	ctx context.Context,
	site string,
	d *HeatMapPoint,
) (*HeatMapPoint, error) {
	return c.createHeatMapPoint(ctx, site, d)
}

func (c *ApiClient) UpdateHeatMapPoint(
	ctx context.Context,
	site string,
	d *HeatMapPoint,
) (*HeatMapPoint, error) {
	return c.updateHeatMapPoint(ctx, site, d)
}
