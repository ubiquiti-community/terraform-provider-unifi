// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
	_ strings.Builder
)

type Hotspot2Conf struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	AnqpDomainID            *int64                              `json:"anqp_domain_id,omitempty"` // ^0|[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5]|$
	Capab                   []Hotspot2ConfCapab                 `json:"capab,omitempty"`
	CellularNetworkList     []Hotspot2ConfCellularNetworkList   `json:"cellular_network_list,omitempty"`
	DeauthReqTimeout        *int64                              `json:"deauth_req_timeout,omitempty"` // [1-9][0-9]|[1-9][0-9][0-9]|[1-2][0-9][0-9][0-9]|3[0-5][0-9][0-9]|3600
	DisableDgaf             bool                                `json:"disable_dgaf"`
	DomainNameList          []string                            `json:"domain_name_list,omitempty"` // .{1,128}
	FriendlyName            []Hotspot2ConfFriendlyName          `json:"friendly_name,omitempty"`
	GasAdvanced             bool                                `json:"gas_advanced"`
	GasComebackDelay        *int64                              `json:"gas_comeback_delay,omitempty"`
	GasFragLimit            *int64                              `json:"gas_frag_limit,omitempty"`
	Hessid                  string                              `json:"hessid"` // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$|^$
	HessidUsed              bool                                `json:"hessid_used"`
	IPaddrTypeAvailV4       *int64                              `json:"ipaddr_type_avail_v4,omitempty"` // 0|1|2|3|4|5|6|7
	IPaddrTypeAvailV6       *int64                              `json:"ipaddr_type_avail_v6,omitempty"` // 0|1|2
	Icons                   []Hotspot2ConfIcons                 `json:"icons,omitempty"`
	MetricsDownlinkLoad     *int64                              `json:"metrics_downlink_load,omitempty"`
	MetricsDownlinkLoadSet  bool                                `json:"metrics_downlink_load_set"`
	MetricsDownlinkSpeed    *int64                              `json:"metrics_downlink_speed,omitempty"`
	MetricsDownlinkSpeedSet bool                                `json:"metrics_downlink_speed_set"`
	MetricsInfoAtCapacity   bool                                `json:"metrics_info_at_capacity"`
	MetricsInfoLinkStatus   string                              `json:"metrics_info_link_status,omitempty"` // up|down|test
	MetricsInfoSymmetric    bool                                `json:"metrics_info_symmetric"`
	MetricsMeasurement      *int64                              `json:"metrics_measurement,omitempty"`
	MetricsMeasurementSet   bool                                `json:"metrics_measurement_set"`
	MetricsStatus           bool                                `json:"metrics_status"`
	MetricsUplinkLoad       *int64                              `json:"metrics_uplink_load,omitempty"`
	MetricsUplinkLoadSet    bool                                `json:"metrics_uplink_load_set"`
	MetricsUplinkSpeed      *int64                              `json:"metrics_uplink_speed,omitempty"`
	MetricsUplinkSpeedSet   bool                                `json:"metrics_uplink_speed_set"`
	NaiRealmList            []Hotspot2ConfNaiRealmList          `json:"nai_realm_list,omitempty"`
	Name                    string                              `json:"name,omitempty"` // .{1,128}
	NetworkAccessAsra       bool                                `json:"network_access_asra"`
	NetworkAccessEsr        bool                                `json:"network_access_esr"`
	NetworkAccessInternet   bool                                `json:"network_access_internet"`
	NetworkAccessUesa       bool                                `json:"network_access_uesa"`
	NetworkAuthType         *int64                              `json:"network_auth_type,omitempty"` // -1|0|1|2|3
	NetworkAuthUrl          string                              `json:"network_auth_url,omitempty"`
	NetworkType             *int64                              `json:"network_type,omitempty"` // 0|1|2|3|4|5|14|15
	Osu                     []Hotspot2ConfOsu                   `json:"osu,omitempty"`
	OsuSSID                 string                              `json:"osu_ssid,omitempty"`
	QOSMapDcsp              []Hotspot2ConfQOSMapDcsp            `json:"qos_map_dcsp,omitempty"`
	QOSMapExceptions        []Hotspot2ConfQOSMapExceptions      `json:"qos_map_exceptions,omitempty"`
	QOSMapStatus            bool                                `json:"qos_map_status"`
	RoamingConsortiumList   []Hotspot2ConfRoamingConsortiumList `json:"roaming_consortium_list,omitempty"`
	SaveTimestamp           string                              `json:"save_timestamp,omitempty"`
	TCFilename              string                              `json:"t_c_filename,omitempty"` // .{1,256}
	TCTimestamp             *int64                              `json:"t_c_timestamp,omitempty"`
	VenueGroup              *int64                              `json:"venue_group,omitempty"` // 0|1|2|3|4|5|6|7|8|9|10|11
	VenueName               []Hotspot2ConfVenueName             `json:"venue_name,omitempty"`
	VenueType               *int64                              `json:"venue_type,omitempty"` // 0|1|2|3|4|5|6|7|8|9|10|11|12|13|14|15
}

func (dst *Hotspot2Conf) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2Conf
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfCapab struct {
	Port     *int64 `json:"port,omitempty"`     // ^(0|[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])|$
	Protocol string `json:"protocol,omitempty"` // icmp|tcp_udp|tcp|udp|esp
	Status   string `json:"status,omitempty"`   // closed|open|unknown
}

func (dst *Hotspot2ConfCapab) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfCapab
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfCellularNetworkList struct {
	Mcc  *int64 `json:"mcc,omitempty"`
	Mnc  *int64 `json:"mnc,omitempty"`
	Name string `json:"name,omitempty"` // .{1,128}
}

func (dst *Hotspot2ConfCellularNetworkList) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfCellularNetworkList
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfDescription struct {
	Language string `json:"language,omitempty"` // [a-z]{3}
	Text     string `json:"text,omitempty"`     // .{1,128}
}

func (dst *Hotspot2ConfDescription) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfDescription
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfFriendlyName struct {
	Language string `json:"language,omitempty"` // [a-z]{3}
	Text     string `json:"text,omitempty"`     // .{1,128}
}

func (dst *Hotspot2ConfFriendlyName) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfFriendlyName
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfIcon struct {
	Name string `json:"name,omitempty"` // .{1,128}
}

func (dst *Hotspot2ConfIcon) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfIcon
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfIcons struct {
	Data     string `json:"data,omitempty"`
	Filename string `json:"filename,omitempty"` // .{1,256}
	Height   *int64 `json:"height,omitempty"`
	Language string `json:"language,omitempty"` // [a-z]{3}
	Media    string `json:"media,omitempty"`    // .{1,256}
	Name     string `json:"name,omitempty"`     // .{1,256}
	Size     *int64 `json:"size,omitempty"`
	Width    *int64 `json:"width,omitempty"`
}

func (dst *Hotspot2ConfIcons) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfIcons
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfNaiRealmList struct {
	AuthIDs   string `json:"auth_ids,omitempty"`
	AuthVals  string `json:"auth_vals,omitempty"`
	EapMethod *int64 `json:"eap_method,omitempty"` // 13|21|18|23|50
	Encoding  *int64 `json:"encoding,omitempty"`   // 0|1
	Name      string `json:"name,omitempty"`       // .{1,128}
	Status    bool   `json:"status"`
}

func (dst *Hotspot2ConfNaiRealmList) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfNaiRealmList
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfOsu struct {
	Description      []Hotspot2ConfDescription  `json:"description,omitempty"`
	FriendlyName     []Hotspot2ConfFriendlyName `json:"friendly_name,omitempty"`
	Icon             []Hotspot2ConfIcon         `json:"icon,omitempty"`
	MethodOmaDm      bool                       `json:"method_oma_dm"`
	MethodSoapXmlSpp bool                       `json:"method_soap_xml_spp"`
	Nai              string                     `json:"nai,omitempty"`
	Nai2             string                     `json:"nai2,omitempty"`
	OperatingClass   string                     `json:"operating_class,omitempty"` // [0-9A-Fa-f]{12}
	ServerUri        string                     `json:"server_uri,omitempty"`
}

func (dst *Hotspot2ConfOsu) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfOsu
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfQOSMapDcsp struct {
	High *int64 `json:"high,omitempty"`
	Low  *int64 `json:"low,omitempty"`
}

func (dst *Hotspot2ConfQOSMapDcsp) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfQOSMapDcsp
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfQOSMapExceptions struct {
	Dcsp *int64 `json:"dcsp,omitempty"`
	Up   *int64 `json:"up,omitempty"` // [0-7]
}

func (dst *Hotspot2ConfQOSMapExceptions) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfQOSMapExceptions
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfRoamingConsortiumList struct {
	Name string `json:"name,omitempty"` // .{1,128}
	Oid  string `json:"oid,omitempty"`  // .{1,128}
}

func (dst *Hotspot2ConfRoamingConsortiumList) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfRoamingConsortiumList
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type Hotspot2ConfVenueName struct {
	Language string `json:"language,omitempty"` // [a-z]{3}
	Name     string `json:"name,omitempty"`
	Url      string `json:"url,omitempty"`
}

func (dst *Hotspot2ConfVenueName) UnmarshalJSON(b []byte) error {
	type Alias Hotspot2ConfVenueName
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

func (c *ApiClient) listHotspot2Conf(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]Hotspot2Conf, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []Hotspot2Conf `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/hotspot2conf", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getHotspot2Conf(
	ctx context.Context,
	site string,
	id string,
) (*Hotspot2Conf, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []Hotspot2Conf `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/hotspot2conf/%s", site, id),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}
	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *ApiClient) deleteHotspot2Conf(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/hotspot2conf/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createHotspot2Conf(
	ctx context.Context,
	site string,
	d *Hotspot2Conf,
) (*Hotspot2Conf, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []Hotspot2Conf `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/hotspot2conf", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}

func (c *ApiClient) updateHotspot2Conf(
	ctx context.Context,
	site string,
	d *Hotspot2Conf,
) (*Hotspot2Conf, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []Hotspot2Conf `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/hotspot2conf/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}
