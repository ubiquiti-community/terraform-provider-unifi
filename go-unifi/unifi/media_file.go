// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListMediaFile(ctx context.Context, site string) ([]MediaFile, error) {
	return c.listMediaFile(ctx, site)
}

func (c *ApiClient) GetMediaFile(ctx context.Context, site, id string) (*MediaFile, error) {
	return c.getMediaFile(ctx, site, id)
}

func (c *ApiClient) DeleteMediaFile(ctx context.Context, site, id string) error {
	return c.deleteMediaFile(ctx, site, id)
}

func (c *ApiClient) CreateMediaFile(
	ctx context.Context,
	site string,
	d *MediaFile,
) (*MediaFile, error) {
	return c.createMediaFile(ctx, site, d)
}

func (c *ApiClient) UpdateMediaFile(
	ctx context.Context,
	site string,
	d *MediaFile,
) (*MediaFile, error) {
	return c.updateMediaFile(ctx, site, d)
}
