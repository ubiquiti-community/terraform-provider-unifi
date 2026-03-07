// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListRADIUSProfile(ctx context.Context, site string) ([]RADIUSProfile, error) {
	return c.listRADIUSProfile(ctx, site)
}

func (c *ApiClient) GetRADIUSProfile(ctx context.Context, site, id string) (*RADIUSProfile, error) {
	return c.getRADIUSProfile(ctx, site, id)
}

func (c *ApiClient) DeleteRADIUSProfile(ctx context.Context, site, id string) error {
	return c.deleteRADIUSProfile(ctx, site, id)
}

func (c *ApiClient) CreateRADIUSProfile(
	ctx context.Context,
	site string,
	d *RADIUSProfile,
) (*RADIUSProfile, error) {
	return c.createRADIUSProfile(ctx, site, d)
}

func (c *ApiClient) UpdateRADIUSProfile(
	ctx context.Context,
	site string,
	d *RADIUSProfile,
) (*RADIUSProfile, error) {
	return c.updateRADIUSProfile(ctx, site, d)
}
