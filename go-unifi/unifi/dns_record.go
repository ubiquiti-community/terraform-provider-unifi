// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListDNSRecord(ctx context.Context, site string) ([]DNSRecord, error) {
	return c.listDNSRecord(ctx, site)
}

func (c *ApiClient) GetDNSRecord(ctx context.Context, site, id string) (*DNSRecord, error) {
	return c.getDNSRecord(ctx, site, id)
}

func (c *ApiClient) DeleteDNSRecord(ctx context.Context, site, id string) error {
	return c.deleteDNSRecord(ctx, site, id)
}

func (c *ApiClient) CreateDNSRecord(
	ctx context.Context,
	site string,
	d *DNSRecord,
) (*DNSRecord, error) {
	return c.createDNSRecord(ctx, site, d)
}

func (c *ApiClient) UpdateDNSRecord(
	ctx context.Context,
	site string,
	d *DNSRecord,
) (*DNSRecord, error) {
	return c.updateDNSRecord(ctx, site, d)
}
