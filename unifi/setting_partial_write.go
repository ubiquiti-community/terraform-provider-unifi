package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// writeSetting sends only the keys of setting that change the stored setting.
//
// The typed go-unifi setting structs serialize most booleans without omitempty,
// so a key the controller has never stored (mgmt.led_enabled, boot_sound,
// usg.dhcpd_use_dnsmasq, radio_ai.useXY, ...) round-trips through the struct as
// false and a full PUT writes it. The controller merges a partial PUT into the
// stored setting (see persistGlobalSwitch), so the struct is diffed against the
// stored raw setting instead: a key is sent when its value differs from the
// stored one, or when it is absent and the new value is not a zero value. An
// absent key and a zero value are treated as equivalent. Unchanged secrets
// (x_ssh_*, x_mgmt_key, ...) are not re-sent either.
func (r *settingResource) writeSetting(ctx context.Context, site string, setting settings.Setting) error {
	key, err := settings.GetSettingKey(setting)
	if err != nil {
		return fmt.Errorf("failed to determine setting key: %w", err)
	}

	stored, err := r.storedSetting(ctx, site, key)
	if err != nil {
		return err
	}
	data, err := settingChanges(stored, setting)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return r.client.UpdateSetting(ctx, site, &settings.RawSetting{
		BaseSetting: settings.BaseSetting{Key: key},
		Data:        data,
	})
}

// storedSetting returns the raw stored setting for key, or nil if the
// controller has never stored it.
func (r *settingResource) storedSetting(ctx context.Context, site, key string) (map[string]any, error) {
	all, err := r.client.ListSettings(ctx, site)
	if err != nil {
		return nil, fmt.Errorf("reading stored %s setting: %w", key, err)
	}
	for _, s := range all {
		if s.Key == key {
			return s.Data, nil
		}
	}
	return nil, nil
}

// settingChanges returns the top-level keys of target that differ from stored.
func settingChanges(stored map[string]any, target settings.Setting) (map[string]any, error) {
	b, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	var want map[string]any
	if err := json.Unmarshal(b, &want); err != nil {
		return nil, err
	}

	changes := map[string]any{}
	for k, v := range want {
		switch k {
		case "_id", "site_id", "key":
			continue
		}
		cur, present := stored[k]
		if present {
			if !jsonValuesEqual(cur, v) {
				changes[k] = v
			}
			continue
		}
		if !isZeroJSON(v) {
			changes[k] = v
		}
	}
	return changes, nil
}

// jsonValuesEqual compares decoded JSON values, treating a number and its
// string form as equal: the controller stores some numeric fields as strings
// (country code "276", radio_ai channel lists) that go-unifi sends as numbers.
func jsonValuesEqual(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	switch av := a.(type) {
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonValuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k := range av {
			if !jsonValuesEqual(av[k], bv[k]) {
				return false
			}
		}
		return true
	case string, float64:
		switch b.(type) {
		case string, float64:
			return fmt.Sprint(a) == fmt.Sprint(b)
		}
	}
	return false
}

func isZeroJSON(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case bool:
		return !t
	case string:
		return t == ""
	case float64:
		return t == 0
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	}
	return false
}
