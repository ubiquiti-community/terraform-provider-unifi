package unifi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

type Setting struct {
	Id     string `json:"_id,omitempty"`
	SiteId string `json:"site_id,omitempty"`
	Key    string `json:"key"`
}

func GetSetting[T settings.Setting](c *ApiClient, ctx context.Context, site string) (*Setting, T, error) {
	// Create a zero value of T to determine the key
	var zeroValue T
	key, err := settings.GetSettingKey(zeroValue)
	if err != nil {
		return nil, zeroValue, fmt.Errorf("failed to determine setting key: %w", err)
	}

	var respBody struct {
		Meta meta              `json:"meta"`
		Data []json.RawMessage `json:"data"`
	}

	if err := c.do(
		ctx,
		"GET",
		fmt.Sprintf("api/s/%s/get/setting/%s", site, key),
		nil,
		&respBody,
	); err != nil {
		return nil, zeroValue, err
	}

	var raw json.RawMessage
	var setting *Setting
	for _, d := range respBody.Data {
		err = json.Unmarshal(d, &setting)
		if err != nil {
			return nil, zeroValue, err
		}
		if setting.Key == key {
			raw = d
			break
		}
	}
	if setting == nil {
		return nil, zeroValue, &NotFoundError{}
	}

	var result T
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return nil, zeroValue, err
	}

	return setting, result, nil
}

// ListSettings retrieves all settings for a site
// The endpoint returns an array of all setting objects identified by their 'key' attribute.
func (c *ApiClient) ListSettings(ctx context.Context, site string) ([]settings.RawSetting, error) {
	var respBody struct {
		Meta meta                  `json:"meta"`
		Data []settings.RawSetting `json:"data"`
	}

	err := c.do(
		ctx,
		"GET",
		fmt.Sprintf("api/s/%s/get/setting", site),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return respBody.Data, nil
}

// UpdateSetting updates a setting using the appropriate endpoint based on the setting type
// The setting's Key field will be automatically set based on the type.
func (c *ApiClient) UpdateSetting(ctx context.Context, site string, setting settings.Setting) error {
	// Determine the key from the setting type
	key, err := settings.GetSettingKey(setting)
	if err != nil {
		return fmt.Errorf("failed to determine setting key: %w", err)
	}

	// Set the key field
	setting.SetKey(key)

	var respBody struct {
		Meta meta              `json:"meta"`
		Data []json.RawMessage `json:"data"`
	}

	err = c.do(
		ctx,
		"PUT",
		fmt.Sprintf("api/s/%s/set/setting/%s", site, key),
		setting,
		&respBody,
	)
	if err != nil {
		return err
	}

	if len(respBody.Data) != 1 {
		return &NotFoundError{}
	}

	// Unmarshal the response back into the setting
	if err := json.Unmarshal(respBody.Data[0], setting); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}
