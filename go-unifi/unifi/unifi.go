package unifi //

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

//go:generate go run ../cmd/fields/ -output-dir=../unifi/ -latest

const (
	loginPath    = "/api/login"
	loginPathNew = "/api/auth/login"
)

// Config holds all configuration for creating a new ApiClient.
type Config struct {
	BaseURL        string
	APIKey         string
	Username       string
	Password       string
	AllowInsecure  bool
	CloudConnector bool
	HardwareID     string
	Logger         any
	TimeoutSeconds *int
	RetryMax       *int
}

// New creates a fully initialized ApiClient from the provided configuration.
// For cloud connector mode, set CloudConnector=true and optionally HardwareID.
// For direct connection, provide BaseURL and either APIKey or Username/Password.
func New(ctx context.Context, cfg *Config) (*ApiClient, error) {
	c := retryablehttp.NewClient()
	timeoutSeconds := 30
	if cfg.TimeoutSeconds != nil {
		timeoutSeconds = *cfg.TimeoutSeconds
	}
	c.HTTPClient.Timeout = time.Duration(timeoutSeconds) * time.Second

	if cfg.Logger != nil {
		c.Logger = cfg.Logger
	} else {
		c.Logger = nil
	}

	if cfg.RetryMax != nil {
		c.RetryMax = *cfg.RetryMax
	}

	if cfg.AllowInsecure {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: time.Duration(timeoutSeconds) * time.Second,
			}).DialContext,
		}
		c.HTTPClient.Transport = transport
	}

	jar, _ := cookiejar.New(nil)
	c.HTTPClient.Jar = jar

	client := &ApiClient{
		c:        c,
		apiKey:   cfg.APIKey,
		username: cfg.Username,
		password: cfg.Password,
	}

	if cfg.CloudConnector {
		var err error
		if cfg.HardwareID != "" {
			_, err = client.enableCloudConnectorByHardwareID(ctx, cfg.HardwareID)
		} else {
			_, err = client.enableCloudConnector(ctx, -1)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to enable cloud connector: %w", err)
		}
	} else {
		if err := client.setBaseURL(cfg.BaseURL); err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
	}

	// setAPIUrlStyle is best-effort; API key connections pre-configure the path.
	_ = client.setAPIUrlStyle(ctx)

	if cfg.Username != "" && cfg.Password != "" && client.apiKey == "" {
		if err := client.login(ctx); err != nil {
			return nil, fmt.Errorf("unable to login: %w", err)
		}
	}

	// setServerVersion is best-effort; version info is not required for operation.
	_ = client.setServerVersion(ctx)

	return client, nil
}

type ApiClient struct {
	loginMu sync.RWMutex // serializes non-apiKey requests for CSRF token propagation

	c       *retryablehttp.Client
	baseURL *url.URL

	apiKey    string
	username  string
	password  string
	loginPath string
	apiPath   string // path to API, e.g. "proxy/network" for new style API, "/api" for old style API

	csrf        string
	tokenExpiry time.Time
	loginErr    error // cached login error to prevent retry storms

	version string

	// Cloud Connector support
	cloudConsoleID string // Console ID for Cloud Connector API proxy
}

func (c *ApiClient) Version() string {
	return c.version
}

// isNewStyle returns true if the client is configured for UniFi OS authentication
// (TOKEN cookie + X-CSRF-Token via /api/auth/login). Returns false for standalone
// Network Application authentication (unifises cookie via /api/login).
func (c *ApiClient) isNewStyle() bool {
	return c.loginPath == loginPathNew
}

// isLoggedIn checks whether the client has a valid authentication session.
// For UniFi OS (new-style): checks for a non-empty CSRF token.
// For standalone (old-style): checks the cookiejar for a unifises session cookie.
func (c *ApiClient) isLoggedIn() bool {
	if c.isNewStyle() {
		return c.csrf != ""
	}
	return c.hasCookie("unifises")
}

// hasCookie checks if the cookiejar contains a cookie with the given name.
func (c *ApiClient) hasCookie(name string) bool {
	if c.baseURL == nil {
		return false
	}
	return slices.ContainsFunc(c.c.HTTPClient.Jar.Cookies(c.baseURL), func(cookie *http.Cookie) bool {
		return cookie.Name == name
	})
}

func (c *ApiClient) setBaseURL(base string) error {
	var err error
	c.baseURL, err = url.Parse(base)
	if err != nil {
		return err
	}

	// error for people who are still passing hard coded old paths
	if path := strings.TrimSuffix(c.baseURL.Path, "/"); path == "/api" {
		return fmt.Errorf("expected a base URL without the `/api`, got: %q", c.baseURL)
	}

	return nil
}

// getHosts retrieves the list of UniFi hosts from the Site Manager API.
// This requires an API key and is typically the first step in using the Cloud Connector API.
func (c *ApiClient) getHosts(ctx context.Context) (*UnifiHostList, error) {
	if c.apiKey == "" {
		return nil, errors.New("API key required to fetch hosts from Site Manager API")
	}

	var hostList UnifiHostList
	err := c.do(ctx, "GET", "https://api.ui.com/v1/hosts", nil, &hostList)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch hosts: %w", err)
	}

	return &hostList, nil
}

// setCloudConsoleID configures the client to use the Cloud Connector API for all requests.
// When set, all API calls will be proxied through https://api.ui.com/v1/connector/consoles/{consoleId}/proxy/...
// This requires an API key and console firmware >= 5.0.3.
func (c *ApiClient) setCloudConsoleID(consoleID string) {
	c.cloudConsoleID = consoleID
	if consoleID != "" {
		// When using cloud connector, force the base URL to api.ui.com
		// and default to new-style paths (cloud connector is always UniFi OS).
		c.baseURL, _ = url.Parse("https://api.ui.com")
		c.apiPath = "/proxy/network"
		c.loginPath = loginPathNew
	}
}

// GetCloudConsoleID returns the currently configured cloud console ID.
func (c *ApiClient) GetCloudConsoleID() string {
	return c.cloudConsoleID
}

// enableCloudConnector fetches available hosts and configures the client to use
// the Cloud Connector API. Selection priority:
// 1. If hostIndex >= 0: uses the host at that index
// 2. If hostIndex < 0: defaults to the first host where owner=true
// 3. Falls back to the first host if no owner found
// Returns the selected console ID and any error encountered.
func (c *ApiClient) enableCloudConnector(ctx context.Context, hostIndex int) (string, error) {
	hosts, err := c.getHosts(ctx)
	if err != nil {
		return "", err
	}

	if len(hosts.Data) == 0 {
		return "", errors.New("no hosts found in Site Manager API")
	}

	var selectedHost *UnifiHost

	// If explicit index provided, use it
	if hostIndex >= 0 && hostIndex < len(hosts.Data) {
		selectedHost = &hosts.Data[hostIndex]
	} else {
		// Default to first owner host
		for i := range hosts.Data {
			if hosts.Data[i].Owner {
				selectedHost = &hosts.Data[i]
				break
			}
		}
		// Fallback to first host if no owner found
		if selectedHost == nil {
			selectedHost = &hosts.Data[0]
		}
	}

	c.setCloudConsoleID(selectedHost.ID)
	return selectedHost.ID, nil
}

// enableCloudConnectorByHardwareID fetches available hosts and configures the client
// to use the Cloud Connector API with the host matching the specified hardware ID.
// Returns the selected console ID and any error encountered.
func (c *ApiClient) enableCloudConnectorByHardwareID(ctx context.Context, hardwareID string) (string, error) {
	hosts, err := c.getHosts(ctx)
	if err != nil {
		return "", err
	}

	host := FindHostByHardwareID(hosts, hardwareID)
	if host == nil {
		return "", fmt.Errorf("no host found with hardware ID: %s", hardwareID)
	}

	c.setCloudConsoleID(host.ID)
	return host.ID, nil
}

// FindHostByHardwareID searches a host list for a specific hardware ID.
// Returns nil if not found.
func FindHostByHardwareID(hostList *UnifiHostList, hardwareID string) *UnifiHost {
	if hostList == nil {
		return nil
	}

	for i := range hostList.Data {
		if hostList.Data[i].HardwareID == hardwareID {
			return &hostList.Data[i]
		}
	}
	return nil
}

// FindOwnerHost returns the first host where owner=true.
// Returns nil if no owner host found.
func FindOwnerHost(hostList *UnifiHostList) *UnifiHost {
	if hostList == nil {
		return nil
	}

	for i := range hostList.Data {
		if hostList.Data[i].Owner {
			return &hostList.Data[i]
		}
	}
	return nil
}

func (c *ApiClient) setAPIUrlStyle(ctx context.Context) error {
	// API keys are a UniFi OS-only feature and only work with the new-style
	// /proxy/network path. Skip the unauthenticated HTTP probe for API key
	// connections because newer UniFi OS firmware returns 302 for GET /,
	// which would otherwise cause the probe to incorrectly set the old-style
	// apiPath ("/") instead of "/proxy/network".
	// This also covers cloud connector mode, which requires an API key and
	// has paths already configured by setCloudConsoleID.
	if c.apiKey != "" {
		c.apiPath = "/proxy/network"
		c.loginPath = loginPathNew
		return nil
	}

	// check if new style API
	// this is modified from the unifi-poller (https://github.com/unifi-poller/unifi) implementation.
	// see https://github.com/unifi-poller/unifi/blob/4dc44f11f61a2e08bf7ec5b20c71d5bced837b5d/unifi.go#L101-L104
	// and https://github.com/unifi-poller/unifi/commit/43a6b225031a28f2b358f52d03a7217c7b524143

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, c.baseURL.String(), nil)
	if err != nil {
		return err
	}

	// We can't share these cookies with other requests, so make a new client.
	// Checking the return code on the first request so don't follow a redirect.
	client := retryablehttp.NewClient()

	client.HTTPClient.Timeout = c.c.HTTPClient.Timeout
	client.HTTPClient.Transport = c.c.HTTPClient.Transport
	client.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusOK {
		// the new API returns a 200 for a / request
		c.apiPath = "/proxy/network"
		c.loginPath = loginPathNew
		return nil
	}

	if resp.StatusCode == http.StatusFound {
		// The old version returns a "302" (to /manage) for a / request
		c.apiPath = "/"
		c.loginPath = loginPath
		return nil
	}

	return errors.New("failed to get api url style")
}

func (c *ApiClient) login(ctx context.Context) error {
	// Clear stale session cookies before login; sending them causes errors.
	if c.baseURL != nil {
		c.c.HTTPClient.Jar.SetCookies(c.baseURL, []*http.Cookie{
			{Name: "TOKEN", MaxAge: -1},
			{Name: "unifises", MaxAge: -1},
		})
	}
	c.csrf = ""

	// Call doRequest directly to avoid login-check recursion and deadlock on loginMu.
	err := c.doRequest(ctx, http.MethodPost, c.loginPath, &struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: c.username,
		Password: c.password,
	}, nil)
	if err != nil {
		c.loginErr = err
		return err
	}

	c.loginErr = nil
	return nil
}

// ensureLoggedIn acquires an exclusive lock and performs login if needed.
// Only one goroutine will attempt login; others wait and reuse the result.
func (c *ApiClient) ensureLoggedIn(ctx context.Context) error {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()

	// Double-check: another goroutine may have already logged in while we waited.
	if c.isLoggedIn() && (c.tokenExpiry.IsZero() || time.Now().Before(c.tokenExpiry)) {
		return nil
	}

	// If not logged in and a previous login already failed, don't retry.
	if !c.isLoggedIn() && c.loginErr != nil {
		return c.loginErr
	}

	// Clear any stale loginErr before attempting (e.g. token-expiry refresh).
	c.loginErr = nil
	return c.login(ctx)
}

func (c *ApiClient) setServerVersion(ctx context.Context) (err error) {
	var status struct {
		Meta struct {
			ServerVersion string `json:"server_version"`
			UUID          string `json:"uuid"`
		} `json:"meta"`
	}

	err = c.do(ctx, http.MethodGet, "status", nil, &status)
	if err != nil {
		return err
	}

	if version := status.Meta.ServerVersion; version != "" {
		c.version = status.Meta.ServerVersion
		return nil
	}

	// newer version of 6.0 controller, use sysinfo to determine version
	// using default site since it must exist
	si, err := c.sysinfo(ctx, "default")
	if err != nil {
		return err
	}

	c.version = si.Version

	if c.version == "" {
		return errors.New("unable to determine controller version")
	}

	return nil
}

func (c *ApiClient) do(
	ctx context.Context,
	method, relativeURL string,
	reqBody any,
	respBody any,
	query ...map[string]string,
) error {
	// For username/password auth (no API key), ensure we are logged in before making requests.
	if c.apiKey == "" && c.username != "" && c.password != "" {
		// Wait for any in-progress login to complete, then check if we need to login.
		c.loginMu.RLock()
		needsLogin := !c.isLoggedIn() || (!c.tokenExpiry.IsZero() && time.Now().After(c.tokenExpiry))
		c.loginMu.RUnlock()

		if needsLogin {
			if err := c.ensureLoggedIn(ctx); err != nil {
				return err
			}
		}

		// Acquire read lock for the duration of the request (blocks while login is in progress).
		c.loginMu.RLock()
		defer c.loginMu.RUnlock()

		// Verify authentication after awaiting the login semaphore.
		if !c.isLoggedIn() {
			if c.loginErr != nil {
				return fmt.Errorf("login failed: %w", c.loginErr)
			}
			return &LoginRequiredError{}
		}
	}

	return c.doRequest(ctx, method, relativeURL, reqBody, respBody, query...)
}

// doRequest performs the actual HTTP request. It is separated from do so that
// login can call it directly without triggering login-check recursion.
func (c *ApiClient) doRequest(
	ctx context.Context,
	method, relativeURL string,
	reqBody any,
	respBody any,
	query ...map[string]string,
) error {
	var (
		reqReader io.Reader
		err       error
		reqBytes  []byte
	)
	if reqBody != nil {
		reqBytes, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("unable to marshal JSON: %s %s %w", method, relativeURL, err)
		}
		reqReader = bytes.NewReader(reqBytes)
	}

	reqURL, err := url.Parse(relativeURL)
	if err != nil {
		return fmt.Errorf("unable to parse URL: %s %s %w", method, relativeURL, err)
	}

	if len(query) > 0 {
		queryValues := reqURL.Query()
		for _, q := range query {
			for key, value := range q {
				queryValues.Set(key, value)
			}
		}
		reqURL.RawQuery = queryValues.Encode()
	}

	// Handle Cloud Connector API routing
	if c.cloudConsoleID != "" && !reqURL.IsAbs() && !strings.HasPrefix(relativeURL, "/v1/hosts") {
		// Route through Cloud Connector proxy: /v1/connector/consoles/{id}/proxy/...
		if !strings.HasPrefix(relativeURL, "/") {
			reqURL.Path = path.Join(c.apiPath, reqURL.Path)
		}
		reqURL.Path = path.Join("/v1/connector/consoles", c.cloudConsoleID, reqURL.Path)
	} else if !strings.HasPrefix(relativeURL, "/") && !reqURL.IsAbs() {
		// Regular API path handling
		reqURL.Path = path.Join(c.apiPath, reqURL.Path)
	}

	url := c.baseURL.ResolveReference(reqURL)
	req, err := retryablehttp.NewRequestWithContext(ctx, method, url.String(), reqReader)
	if err != nil {
		return fmt.Errorf("unable to create request: %s %s %w", method, relativeURL, err)
	}

	req.Header.Set("User-Agent", "terraform-provider-unifi/0.1")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	if c.csrf != "" {
		req.Header.Set("X-Csrf-Token", c.csrf)
	}

	resp, err := c.c.Do(req)
	if err != nil {
		return fmt.Errorf("unable to perform request: %s %s %w", method, relativeURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		var typeName string
		if t := reflect.TypeOf(respBody); t != nil {
			typeName = t.String()
		}
		return &NotFoundError{
			Type: typeName,
		}
	}

	if csrf := resp.Header.Get("X-Updated-Csrf-Token"); csrf != "" {
		c.csrf = csrf
	} else if csrf := resp.Header.Get("X-Csrf-Token"); csrf != "" {
		c.csrf = csrf
	}
	if exp := resp.Header.Get("X-Token-Expire-Time"); exp != "" {
		if ms, err := strconv.ParseInt(exp, 10, 64); err == nil {
			c.tokenExpiry = time.UnixMilli(ms)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return &LoginRequiredError{APIKey: c.apiKey != ""}
		}
		errBody := struct {
			Meta meta `json:"meta"`
			Data []struct {
				Meta meta `json:"meta"`
			} `json:"data"`
		}{}
		if err = json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
			return err
		}
		var apiErr error
		if len(errBody.Data) > 0 && errBody.Data[0].Meta.RC == "error" {
			// check first error in data, should we look for more than one?
			apiErr = errBody.Data[0].Meta.error()
		}
		if apiErr == nil {
			apiErr = errBody.Meta.error()
		}
		return fmt.Errorf(
			"%w (%s) for %s %s\npayload: %s",
			apiErr,
			strings.TrimSpace(resp.Status),
			method,
			url.String(),
			string(reqBytes),
		)
	}

	if respBody == nil || resp.ContentLength == 0 {
		return nil
	}

	// TODO: check rc in addition to status code?

	err = json.NewDecoder(resp.Body).Decode(respBody)
	if err != nil {
		return fmt.Errorf("unable to decode body: %s %s %w", method, relativeURL, err)
	}

	return nil
}

type meta struct {
	RC      string `json:"rc"`
	Message string `json:"msg"`
}

func (m *meta) error() error {
	if m.RC != "ok" {
		return &APIError{
			RC:      m.RC,
			Message: m.Message,
		}
	}

	return nil
}
