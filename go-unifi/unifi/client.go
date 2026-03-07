package unifi

import (
	"context"
	"fmt"
	"maps"
)

// GetClientByMAC returns slightly different information than GetClient, as they
// use separate endpoints for their lookups. Specifically IP is only returned
// by this method.
func (c *ApiClient) GetClientByMAC(ctx context.Context, site, mac string) (*Client, error) {
	resp, err := c.ListClient(ctx, site, map[string]string{"mac": mac})
	if err != nil {
		return nil, err
	}
	if len(resp) != 1 {
		return nil, &NotFoundError{}
	}
	d := resp[0]

	return &d, nil
}

func (c *ApiClient) stamgr(
	ctx context.Context,
	site, cmd string,
	data map[string]any,
) ([]Client, error) {
	reqBody := map[string]any{}

	maps.Copy(reqBody, data)

	reqBody["cmd"] = cmd

	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Client `json:"data"`
	}

	err := c.do(ctx, "POST", fmt.Sprintf("api/s/%s/cmd/stamgr", site), reqBody, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Data, nil
}

func (c *ApiClient) BlockClientByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "block-sta", map[string]any{
		"mac": mac,
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *ApiClient) UnblockClientByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "unblock-sta", map[string]any{
		"mac": mac,
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *ApiClient) DeleteClientByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "forget-sta", map[string]any{
		"macs": []string{mac},
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *ApiClient) KickClientByMAC(ctx context.Context, site, mac string) error {
	users, err := c.stamgr(ctx, site, "kick-sta", map[string]any{
		"mac": mac,
	})
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return &NotFoundError{}
	}
	return nil
}

func (c *ApiClient) OverrideClientFingerprint(
	ctx context.Context,
	site, mac string,
	devIdOveride int64,
) error {
	reqBody := map[string]any{
		"mac":             mac,
		"dev_id_override": devIdOveride,
		"search_query":    "",
	}

	var reqMethod string
	if devIdOveride == 0 {
		reqMethod = "DELETE"
	} else {
		reqMethod = "PUT"
	}

	var respBody struct {
		Mac           string `json:"mac"`
		DevIdOverride int64  `json:"dev_id_override"`
		SearchQuery   string `json:"search_query"`
	}

	err := c.do(
		ctx,
		reqMethod,
		fmt.Sprintf("v2/api/site/%s/station/%s/fingerprint_override", site, mac),
		reqBody,
		&respBody,
	)
	if err != nil {
		return err
	}

	return nil
}

// ListClient returns all clients, optionally filtered by query parameters.
// The query parameter can contain any field from the Client struct to filter results.
// For example: map[string]string{"network_id": "abc123", "blocked": "true"}.
func (c *ApiClient) ListClient(
	ctx context.Context,
	site string,
	params ...map[string]string,
) ([]Client, error) {
	return c.listClient(ctx, site, params...)
}

// ListClientFiltered returns clients filtered by the provided key-value parameters.
// This is the map-based variant of ListClient for use outside the unifi package,
// where the anonymous struct parameter type cannot be spread as variadic args.
func (c *ApiClient) ListClientFiltered(ctx context.Context, site string, filters map[string]string) ([]Client, error) {
	return c.listClient(ctx, site, filters)
}

func (c *ApiClient) CreateClient(ctx context.Context, site string, d *Client) (*Client, error) {
	return c.createClient(ctx, site, d)
}

// GetClient returns information about a user from the REST endpoint.
// The GetClientByMAC method returns slightly different information (for
// example the IP) as it uses a different endpoint.
func (c *ApiClient) GetClient(ctx context.Context, site, id string) (*Client, error) {
	return c.getClient(ctx, site, id)
}

func (c *ApiClient) UpdateClient(ctx context.Context, site string, d *Client) (*Client, error) {
	return c.updateClient(ctx, site, d)
}
