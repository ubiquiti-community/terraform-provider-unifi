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

type GuestAccess struct {
	BaseSetting

	AllowedSubnet                          string   `json:"allowed_subnet_,omitempty"`
	Auth                                   string   `json:"auth,omitempty"` // none|hotspot|facebook_wifi|custom
	AuthUrl                                string   `json:"auth_url,omitempty"`
	AuthorizeUseSandbox                    bool     `json:"authorize_use_sandbox"`
	CustomIP                               string   `json:"custom_ip"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	EcEnabled                              bool     `json:"ec_enabled"`
	Expire                                 string   `json:"expire,omitempty"`        // [\d]+|custom
	ExpireNumber                           *int64   `json:"expire_number,omitempty"` // ^[1-9][0-9]{0,5}|1000000$
	ExpireUnit                             *int64   `json:"expire_unit,omitempty"`   // 1|60|1440
	FacebookAppID                          string   `json:"facebook_app_id,omitempty"`
	FacebookEnabled                        bool     `json:"facebook_enabled"`
	FacebookScopeEmail                     bool     `json:"facebook_scope_email"`
	FacebookWifiBlockHttps                 bool     `json:"facebook_wifi_block_https"`
	FacebookWifiGwID                       string   `json:"facebook_wifi_gw_id,omitempty"`
	FacebookWifiGwName                     string   `json:"facebook_wifi_gw_name,omitempty"`
	Gateway                                string   `json:"gateway,omitempty"` // paypal|stripe|authorize|quickpay|merchantwarrior|ippay
	GoogleClientID                         string   `json:"google_client_id,omitempty"`
	GoogleDomain                           string   `json:"google_domain,omitempty"`
	GoogleEnabled                          bool     `json:"google_enabled"`
	GoogleScopeEmail                       bool     `json:"google_scope_email"`
	IPpayUseSandbox                        bool     `json:"ippay_use_sandbox"`
	MerchantwarriorUseSandbox              bool     `json:"merchantwarrior_use_sandbox"`
	PasswordEnabled                        bool     `json:"password_enabled"`
	PaymentEnabled                         bool     `json:"payment_enabled"`
	PaypalUseSandbox                       bool     `json:"paypal_use_sandbox"`
	PortalCustomized                       bool     `json:"portal_customized"`
	PortalCustomizedAuthenticationText     string   `json:"portal_customized_authentication_text,omitempty"`
	PortalCustomizedBgColor                string   `json:"portal_customized_bg_color"` // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedBgImageEnabled         bool     `json:"portal_customized_bg_image_enabled"`
	PortalCustomizedBgImageFilename        string   `json:"portal_customized_bg_image_filename,omitempty"`
	PortalCustomizedBgImageTile            bool     `json:"portal_customized_bg_image_tile"`
	PortalCustomizedBgType                 string   `json:"portal_customized_bg_type,omitempty"`     // color|image|gallery
	PortalCustomizedBoxColor               string   `json:"portal_customized_box_color"`             // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedBoxLinkColor           string   `json:"portal_customized_box_link_color"`        // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedBoxOpacity             *int64   `json:"portal_customized_box_opacity,omitempty"` // ^[1-9][0-9]?$|^100$|^$
	PortalCustomizedBoxRADIUS              *int64   `json:"portal_customized_box_radius,omitempty"`  // [0-9]|[1-4][0-9]|50
	PortalCustomizedBoxTextColor           string   `json:"portal_customized_box_text_color"`        // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedButtonColor            string   `json:"portal_customized_button_color"`          // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedButtonText             string   `json:"portal_customized_button_text,omitempty"`
	PortalCustomizedButtonTextColor        string   `json:"portal_customized_button_text_color"`   // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedLanguages              []string `json:"portal_customized_languages,omitempty"` // ^[a-z]{2}([_-][a-zA-Z]{2,4})*$
	PortalCustomizedLinkColor              string   `json:"portal_customized_link_color"`          // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedLogoEnabled            bool     `json:"portal_customized_logo_enabled"`
	PortalCustomizedLogoFilename           string   `json:"portal_customized_logo_filename,omitempty"`
	PortalCustomizedLogoPosition           string   `json:"portal_customized_logo_position,omitempty"` // left|center|right
	PortalCustomizedLogoSize               *int64   `json:"portal_customized_logo_size,omitempty"`     // 6[4-9]|[7-9][0-9]|1[0-8][0-9]|19[0-2]
	PortalCustomizedSuccessText            string   `json:"portal_customized_success_text,omitempty"`
	PortalCustomizedTextColor              string   `json:"portal_customized_text_color"` // ^#[a-zA-Z0-9]{6}$|^#[a-zA-Z0-9]{3}$|^$
	PortalCustomizedTitle                  string   `json:"portal_customized_title,omitempty"`
	PortalCustomizedTos                    string   `json:"portal_customized_tos,omitempty"`
	PortalCustomizedTosEnabled             bool     `json:"portal_customized_tos_enabled"`
	PortalCustomizedUnsplashAuthorName     string   `json:"portal_customized_unsplash_author_name,omitempty"`
	PortalCustomizedUnsplashAuthorUsername string   `json:"portal_customized_unsplash_author_username,omitempty"`
	PortalCustomizedWelcomeText            string   `json:"portal_customized_welcome_text,omitempty"`
	PortalCustomizedWelcomeTextEnabled     bool     `json:"portal_customized_welcome_text_enabled"`
	PortalCustomizedWelcomeTextPosition    string   `json:"portal_customized_welcome_text_position,omitempty"` // under_logo|above_boxes
	PortalEnabled                          bool     `json:"portal_enabled"`
	PortalHostname                         string   `json:"portal_hostname"` // ^[a-zA-Z0-9.-]+$|^$
	PortalUseHostname                      bool     `json:"portal_use_hostname"`
	QuickpayTestmode                       bool     `json:"quickpay_testmode"`
	RADIUSAuthType                         string   `json:"radius_auth_type,omitempty"` // chap|mschapv2
	RADIUSDisconnectEnabled                bool     `json:"radius_disconnect_enabled"`
	RADIUSDisconnectPort                   *int64   `json:"radius_disconnect_port,omitempty"` // [1-9][0-9]{0,3}|[1-5][0-9]{4}|[6][0-4][0-9]{3}|[6][5][0-4][0-9]{2}|[6][5][5][0-2][0-9]|[6][5][5][3][0-5]
	RADIUSEnabled                          bool     `json:"radius_enabled"`
	RADIUSProfileID                        string   `json:"radiusprofile_id,omitempty"`
	RedirectEnabled                        bool     `json:"redirect_enabled"`
	RedirectHttps                          bool     `json:"redirect_https"`
	RedirectToHttps                        bool     `json:"redirect_to_https"`
	RedirectUrl                            string   `json:"redirect_url,omitempty"`
	RestrictedDNSEnabled                   bool     `json:"restricted_dns_enabled"`
	RestrictedDNSServers                   []string `json:"restricted_dns_servers,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	RestrictedSubnet                       string   `json:"restricted_subnet_,omitempty"`
	TemplateEngine                         string   `json:"template_engine,omitempty"` // jsp|angular
	VoucherCustomized                      bool     `json:"voucher_customized"`
	VoucherEnabled                         bool     `json:"voucher_enabled"`
	WechatAppID                            string   `json:"wechat_app_id,omitempty"`
	WechatEnabled                          bool     `json:"wechat_enabled"`
	WechatShopID                           string   `json:"wechat_shop_id,omitempty"`
	XAuthorizeLoginid                      string   `json:"x_authorize_loginid,omitempty"`
	XAuthorizeTransactionkey               string   `json:"x_authorize_transactionkey,omitempty"`
	XFacebookAppSecret                     string   `json:"x_facebook_app_secret,omitempty"`
	XFacebookWifiGwSecret                  string   `json:"x_facebook_wifi_gw_secret,omitempty"`
	XGoogleClientSecret                    string   `json:"x_google_client_secret,omitempty"`
	XIPpayTerminalid                       string   `json:"x_ippay_terminalid,omitempty"`
	XMerchantwarriorApikey                 string   `json:"x_merchantwarrior_apikey,omitempty"`
	XMerchantwarriorApipassphrase          string   `json:"x_merchantwarrior_apipassphrase,omitempty"`
	XMerchantwarriorMerchantuuid           string   `json:"x_merchantwarrior_merchantuuid,omitempty"`
	XPassword                              string   `json:"x_password,omitempty"`
	XPaypalPassword                        string   `json:"x_paypal_password,omitempty"`
	XPaypalSignature                       string   `json:"x_paypal_signature,omitempty"`
	XPaypalUsername                        string   `json:"x_paypal_username,omitempty"`
	XQuickpayAgreementid                   string   `json:"x_quickpay_agreementid,omitempty"`
	XQuickpayApikey                        string   `json:"x_quickpay_apikey,omitempty"`
	XQuickpayMerchantid                    string   `json:"x_quickpay_merchantid,omitempty"`
	XStripeApiKey                          string   `json:"x_stripe_api_key,omitempty"`
	XWechatAppSecret                       string   `json:"x_wechat_app_secret,omitempty"`
	XWechatSecretKey                       string   `json:"x_wechat_secret_key,omitempty"`
}

func (dst *GuestAccess) UnmarshalJSON(b []byte) error {
	type Alias GuestAccess
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
