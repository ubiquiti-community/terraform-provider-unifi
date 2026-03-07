// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListTag(ctx context.Context, site string) ([]Tag, error) {
	return c.listTag(ctx, site)
}

func (c *ApiClient) GetTag(ctx context.Context, site, id string) (*Tag, error) {
	return c.getTag(ctx, site, id)
}

func (c *ApiClient) DeleteTag(ctx context.Context, site, id string) error {
	return c.deleteTag(ctx, site, id)
}

func (c *ApiClient) CreateTag(ctx context.Context, site string, d *Tag) (*Tag, error) {
	return c.createTag(ctx, site, d)
}

func (c *ApiClient) UpdateTag(ctx context.Context, site string, d *Tag) (*Tag, error) {
	return c.updateTag(ctx, site, d)
}
