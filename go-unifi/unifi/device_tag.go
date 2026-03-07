package unifi

import (
	"context"
	"fmt"
)

// just to fix compile issues with the import.
var (
	_ fmt.Formatter
	_ context.Context
)

// This is a v2 API object, manually coded.

type DeviceTag struct {
	ID               string   `json:"_id,omitempty"`
	Name             string   `json:"name"`
	MemberDeviceMacs []string `json:"member_device_macs"`
}

type deviceTagAssignment struct {
	Additions []string `json:"device_tag_additions"`
	Removals  []string `json:"device_tag_removals"`
}

func (c *ApiClient) ListDeviceTags(ctx context.Context, site string) ([]DeviceTag, error) {
	var respBody []DeviceTag

	err := c.do(ctx, "GET", fmt.Sprintf("v2/api/site/%s/device-tags", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

// AssignDeviceTag adds and removes device tag IDs for the given device MAC address.
// Returns the updated list of all device tags.
func (c *ApiClient) AssignDeviceTag(ctx context.Context, site string, mac string, additions []string, removals []string) ([]DeviceTag, error) {
	var respBody []DeviceTag

	body := deviceTagAssignment{
		Additions: additions,
		Removals:  removals,
	}

	err := c.do(ctx, "POST", fmt.Sprintf("v2/api/site/%s/device-tags/device-tag-assignment/%s", site, mac), body, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}
