// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
)

func (c *ApiClient) GetBGPConfig(ctx context.Context, site string) (*BGPConfig, error) {
	return c.getBGPConfig(ctx, site)
}

func (c *ApiClient) CreateBGPConfig(
	ctx context.Context,
	site string,
	d *BGPConfig,
) (*BGPConfig, error) {
	return c.createBGPConfig(ctx, site, d)
}

func (c *ApiClient) UpdateBGPConfig(
	ctx context.Context,
	site string,
	d *BGPConfig,
) (*BGPConfig, error) {
	return c.createBGPConfig(ctx, site, d)
}

func (c *ApiClient) DeleteBGPConfig(ctx context.Context, site string) error {
	return c.deleteBGPConfig(ctx, site)
}
