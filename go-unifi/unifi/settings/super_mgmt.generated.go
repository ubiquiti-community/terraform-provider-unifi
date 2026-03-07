// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
)

type SuperMgmt struct {
	BaseSetting

	AnalyticsDisapprovedFor                  string   `json:"analytics_disapproved_for,omitempty"`
	AutoUpgrade                              bool     `json:"auto_upgrade"`
	AutobackupCronExpr                       string   `json:"autobackup_cron_expr,omitempty"`
	AutobackupDays                           *int64   `json:"autobackup_days,omitempty"`
	AutobackupEnabled                        bool     `json:"autobackup_enabled"`
	AutobackupGcsBucket                      string   `json:"autobackup_gcs_bucket,omitempty"`
	AutobackupGcsCertificatePath             string   `json:"autobackup_gcs_certificate_path,omitempty"`
	AutobackupLocalPath                      string   `json:"autobackup_local_path,omitempty"`
	AutobackupMaxFiles                       *int64   `json:"autobackup_max_files,omitempty"`
	AutobackupPostActions                    []string `json:"autobackup_post_actions,omitempty"` // copy_local|copy_gcs|copy_cloud
	AutobackupTimezone                       string   `json:"autobackup_timezone,omitempty"`
	BackupToCloudEnabled                     bool     `json:"backup_to_cloud_enabled"`
	ContactInfoCity                          string   `json:"contact_info_city,omitempty"`
	ContactInfoCompanyName                   string   `json:"contact_info_company_name,omitempty"`
	ContactInfoCountry                       string   `json:"contact_info_country,omitempty"`
	ContactInfoFullName                      string   `json:"contact_info_full_name,omitempty"`
	ContactInfoPhoneNumber                   string   `json:"contact_info_phone_number,omitempty"`
	ContactInfoShippingAddress1              string   `json:"contact_info_shipping_address_1,omitempty"`
	ContactInfoShippingAddress2              string   `json:"contact_info_shipping_address_2,omitempty"`
	ContactInfoState                         string   `json:"contact_info_state,omitempty"`
	ContactInfoZip                           string   `json:"contact_info_zip,omitempty"`
	DataRetentionSettingPreference           string   `json:"data_retention_setting_preference,omitempty"` // auto|manual
	DataRetentionTimeInHoursFor5MinutesScale *int64   `json:"data_retention_time_in_hours_for_5minutes_scale,omitempty"`
	DataRetentionTimeInHoursForDailyScale    *int64   `json:"data_retention_time_in_hours_for_daily_scale,omitempty"`
	DataRetentionTimeInHoursForHourlyScale   *int64   `json:"data_retention_time_in_hours_for_hourly_scale,omitempty"`
	DataRetentionTimeInHoursForMonthlyScale  *int64   `json:"data_retention_time_in_hours_for_monthly_scale,omitempty"`
	DataRetentionTimeInHoursForOthers        *int64   `json:"data_retention_time_in_hours_for_others,omitempty"`
	DefaultSiteDeviceAuthPasswordAlert       string   `json:"default_site_device_auth_password_alert,omitempty"` // false
	Discoverable                             bool     `json:"discoverable"`
	EnableAnalytics                          bool     `json:"enable_analytics"`
	GoogleMapsApiKey                         string   `json:"google_maps_api_key,omitempty"`
	ImageMapsUseGoogleEngine                 bool     `json:"image_maps_use_google_engine"`
	LedEnabled                               bool     `json:"led_enabled"`
	LiveChat                                 string   `json:"live_chat,omitempty"`    // disabled|super-only|everyone
	LiveUpdates                              string   `json:"live_updates,omitempty"` // disabled|live|auto
	MinimumUsableHdSpace                     *int64   `json:"minimum_usable_hd_space,omitempty"`
	MinimumUsableSdSpace                     *int64   `json:"minimum_usable_sd_space,omitempty"`
	MultipleSitesEnabled                     bool     `json:"multiple_sites_enabled"`
	OverrideInformHost                       bool     `json:"override_inform_host"`
	OverrideInformHostLocation               string   `json:"override_inform_host_location,omitempty"`
	SSHPassword                              string   `json:"x_ssh_password,omitempty"`
	SSHUsername                              string   `json:"x_ssh_username,omitempty"`
	StoreEnabled                             string   `json:"store_enabled,omitempty"` // disabled|super-only|everyone
	TimeSeriesPerClientStatsEnabled          bool     `json:"time_series_per_client_stats_enabled"`
}

func (dst *SuperMgmt) UnmarshalJSON(b []byte) error {
	type Alias SuperMgmt
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	// First unmarshal base setting
	if err := json.Unmarshal(b, &dst.BaseSetting); err != nil {
		return fmt.Errorf("unable to unmarshal base setting: %w", err)
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}
