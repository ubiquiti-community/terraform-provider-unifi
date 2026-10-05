package unifi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// A setting as UniFi Network 10.6.106 stores it: most optional booleans are
// absent rather than false.
func storedMgmt() map[string]any {
	return map[string]any{
		"_id":                         "66ec38ab34dbcb19dce4a66c",
		"site_id":                     "66ec388934dbcb19dce4a667",
		"key":                         "mgmt",
		"auto_upgrade":                true,
		"auto_upgrade_hour":           float64(3),
		"advanced_feature_enabled":    true,
		"debug_tools_enabled":         false,
		"unifi_idp_enabled":           true,
		"wifiman_enabled":             true,
		"x_ssh_enabled":               true,
		"x_ssh_auth_password_enabled": true,
		"x_ssh_password":              "secret",
	}
}

func Test_settingChanges_absentKeysAreNotWrittenAsFalse(t *testing.T) {
	// What the typed struct round-trips to: led_enabled, boot_sound, ... come
	// back as false because the struct has no way to say "absent".
	target := &settings.Mgmt{
		AutoUpgrade:                  true,
		AutoUpgradeHour:              ptrInt64(3),
		AdvancedFeatureEnabled:       true,
		UniFiIdentityProviderEnabled: true,
		WifimanEnabled:               true,
		SSHEnabled:                   true,
		SSHPassword:                  "secret",
	}
	got, err := settingChanges(storedMgmt(), target)
	if err != nil {
		t.Fatal(err)
	}
	// Only the real change: password auth switched off.
	want := map[string]any{"x_ssh_auth_password_enabled": false}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
}

func Test_settingChanges_unchangedSettingSendsNothing(t *testing.T) {
	target := &settings.Mgmt{
		AutoUpgrade:                  true,
		AutoUpgradeHour:              ptrInt64(3),
		AdvancedFeatureEnabled:       true,
		UniFiIdentityProviderEnabled: true,
		WifimanEnabled:               true,
		SSHEnabled:                   true,
		SSHAuthPasswordEnabled:       true,
		SSHPassword:                  "secret",
	}
	got, err := settingChanges(storedMgmt(), target)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("changes = %v, want none", got)
	}
}

func Test_settingChanges_newNonZeroKeyIsSent(t *testing.T) {
	target := &settings.Mgmt{AutoUpgrade: true, AutoUpgradeHour: ptrInt64(3), LedEnabled: true}
	got, err := settingChanges(storedMgmt(), target)
	if err != nil {
		t.Fatal(err)
	}
	if got["led_enabled"] != true {
		t.Errorf("led_enabled = %v, want true sent", got["led_enabled"])
	}
}

func Test_settingChanges_numberAndStringFormsAreEqual(t *testing.T) {
	// The controller stores country.code as "276"; go-unifi sends 276.
	got, err := settingChanges(
		map[string]any{"key": "country", "code": "276"},
		&settings.Country{Code: ptrInt64(276)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("changes = %v, want none", got)
	}
}

func Test_jsonValuesEqual(t *testing.T) {
	for _, tt := range []struct {
		a, b any
		want bool
	}{
		{[]any{"36", "40"}, []any{float64(36), float64(40)}, true},
		{[]any{"36"}, []any{float64(40)}, false},
		{map[string]any{"mode": "auto"}, map[string]any{"mode": "auto"}, true},
		{true, false, false},
		{"", float64(0), false},
		{float64(276), "276", true},
		{[]any{"36"}, []any{"36", "40"}, false},
		{[]any{"36"}, "36", false},
		{map[string]any{"a": "1"}, map[string]any{"a": float64(1)}, true},
		{map[string]any{"a": "1"}, map[string]any{"a": "1", "b": "2"}, false},
		{map[string]any{"a": "1"}, map[string]any{"a": float64(2)}, false},
		{map[string]any{"a": "1"}, []any{"1"}, false},
		{true, "true", false},
	} {
		if got := jsonValuesEqual(tt.a, tt.b); got != tt.want {
			t.Errorf("jsonValuesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// newSettingsFakeController serves the given stored settings and records every
// setting PUT body by key. Like the controller, it merges a PUT into the stored
// setting, so later reads see the write.
func newSettingsFakeController(
	t *testing.T,
	stored []map[string]any,
) (*settingResource, func() map[string]map[string]any) {
	t.Helper()
	var mu sync.Mutex
	puts := map[string]map[string]any{}
	byKey := map[string]map[string]any{}
	var order []string
	for _, s := range stored {
		k, _ := s["key"].(string)
		byKey[k] = s
		order = append(order, k)
	}
	reply := func(w http.ResponseWriter, data []any) {
		_ = json.NewEncoder(w).
			Encode(map[string]any{"meta": map[string]any{"rc": "ok"}, "data": data})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/manage", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "unifises", Value: "fake-session", Path: "/"})
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
	})
	mux.HandleFunc("/api/s/default/get/setting", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		data := []any{}
		for _, k := range order {
			data = append(data, byKey[k])
		}
		reply(w, data)
	})
	mux.HandleFunc("/api/s/default/get/setting/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		k := strings.TrimPrefix(r.URL.Path, "/api/s/default/get/setting/")
		if s, ok := byKey[k]; ok {
			reply(w, []any{s})
			return
		}
		reply(w, []any{map[string]any{"key": k}})
	})
	mux.HandleFunc("/api/s/default/set/setting/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(b, &body); err != nil {
			t.Errorf("PUT body is not JSON: %v", err)
		}
		k := strings.TrimPrefix(r.URL.Path, "/api/s/default/set/setting/")
		mu.Lock()
		defer mu.Unlock()
		puts[k] = body
		if byKey[k] == nil {
			byKey[k] = map[string]any{}
			order = append(order, k)
		}
		for bk, bv := range body {
			byKey[k][bk] = bv
		}
		reply(w, []any{byKey[k]})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL: srv.URL, Username: "admin", Password: "admin",
	})
	if err != nil {
		t.Fatalf("creating client against fake controller: %v", err)
	}
	r := &settingResource{client: &Client{ApiClient: apiClient, Site: "default"}}
	return r, func() map[string]map[string]any {
		mu.Lock()
		defer mu.Unlock()
		got := puts
		puts = map[string]map[string]any{}
		return got
	}
}

func Test_writeSetting(t *testing.T) {
	ctx := context.Background()
	r, puts := newSettingsFakeController(t, []map[string]any{
		storedMgmt(),
		{"key": "country", "code": "276"},
	})

	// Unchanged: no request at all.
	if err := r.writeSetting(ctx, "default", &settings.Country{Code: ptrInt64(276)}); err != nil {
		t.Fatal(err)
	}
	// Changed: only the changed key, plus the setting key itself.
	if err := r.writeSetting(ctx, "default", &settings.Mgmt{
		AutoUpgrade:                  true,
		AutoUpgradeHour:              ptrInt64(3),
		AdvancedFeatureEnabled:       true,
		UniFiIdentityProviderEnabled: true,
		WifimanEnabled:               true,
		SSHEnabled:                   true,
		SSHPassword:                  "secret",
	}); err != nil {
		t.Fatal(err)
	}
	// Never stored: non-zero keys are sent, zero values are not.
	if err := r.writeSetting(ctx, "default", &settings.Dpi{Enabled: true}); err != nil {
		t.Fatal(err)
	}

	got := puts()
	if _, ok := got["country"]; ok {
		t.Errorf("unchanged country setting was written: %v", got["country"])
	}
	wantMgmt := map[string]any{"key": "mgmt", "x_ssh_auth_password_enabled": false}
	if !reflect.DeepEqual(got["mgmt"], wantMgmt) {
		t.Errorf("mgmt PUT = %v, want %v", got["mgmt"], wantMgmt)
	}
	wantDpi := map[string]any{"key": "dpi", "enabled": true}
	if !reflect.DeepEqual(got["dpi"], wantDpi) {
		t.Errorf("dpi PUT = %v, want %v", got["dpi"], wantDpi)
	}
}

func Test_isZeroJSON(t *testing.T) {
	for _, tt := range []struct {
		v    any
		want bool
	}{
		{nil, true},
		{false, true},
		{"", true},
		{float64(0), true},
		{[]any{}, true},
		{map[string]any{}, true},
		{true, false},
		{"x", false},
		{float64(1), false},
		{[]any{"x"}, false},
		{map[string]any{"a": 1}, false},
		{struct{}{}, false},
	} {
		if got := isZeroJSON(tt.v); got != tt.want {
			t.Errorf("isZeroJSON(%#v) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

// rawJSONSetting marshals to a fixed JSON text, to exercise settingChanges on
// bodies the typed settings never produce.
type rawJSONSetting struct {
	settings.BaseSetting
	json string
}

func (s *rawJSONSetting) MarshalJSON() ([]byte, error) {
	if s.json == "" {
		return nil, errors.New("cannot marshal")
	}
	return []byte(s.json), nil
}

func Test_settingChanges_errors(t *testing.T) {
	if _, err := settingChanges(nil, &rawJSONSetting{}); err == nil {
		t.Error("marshal failure: want error")
	}
	if _, err := settingChanges(nil, &rawJSONSetting{json: `["not", "an", "object"]`}); err == nil {
		t.Error("non-object body: want error")
	}
}

func Test_writeSetting_errors(t *testing.T) {
	ctx := context.Background()

	r, _ := newSettingsFakeController(t, nil)
	if err := r.writeSetting(ctx, "default", &rawJSONSetting{json: `{}`}); err == nil {
		t.Error("setting without a known key: want error")
	}
	if err := r.writeSetting(ctx, "default", &settings.Country{Code: ptrInt64(276)}); err != nil {
		t.Errorf("baseline write failed: %v", err)
	}
	if err := r.writeSetting(
		ctx,
		"other-site",
		&settings.Country{Code: ptrInt64(276)},
	); err == nil {
		t.Error("unreadable stored settings: want error")
	}
}
