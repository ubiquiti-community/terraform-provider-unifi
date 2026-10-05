package unifi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// newLaggingAPController serves one connected access point. A PUT is accepted
// and answered with the written values, but reads keep reporting the LED and
// band steering values from before the write, as an AP does until it has
// applied them.
func newLaggingAPController(t *testing.T, ap map[string]any) (*Client, func() map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var lastPut map[string]any
	reply := func(w http.ResponseWriter, d map[string]any) {
		_ = json.NewEncoder(w).
			Encode(map[string]any{"meta": map[string]any{"rc": "ok"}, "data": []any{d}})
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
	mux.HandleFunc("/api/s/default/stat/device/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if k := strings.TrimPrefix(r.URL.Path, "/api/s/default/stat/device/"); k != ap["mac"] &&
			k != ap["_id"] {
			http.NotFound(w, r)
			return
		}
		reply(w, ap)
	})
	mux.HandleFunc("/api/s/default/rest/device/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(b, &body); err != nil {
			t.Errorf("PUT body is not JSON: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		lastPut = body
		applied := map[string]any{}
		for k, v := range ap {
			applied[k] = v
		}
		for k, v := range body {
			applied[k] = v
			switch k {
			case "led_override", "led_override_color", "led_override_color_brightness",
				"bandsteering_mode":
				// applied later by the AP; reads still show the old value
			default:
				ap[k] = v
			}
		}
		reply(w, applied)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL: srv.URL, Username: "admin", Password: "admin",
	})
	if err != nil {
		t.Fatalf("creating client against fake controller: %v", err)
	}
	return &Client{ApiClient: apiClient, Site: "default"}, func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return lastPut
	}
}

// Changing band steering and the LED of an AP: the read right after the PUT
// still reports the old values, and the state must hold the planned ones, or
// the apply fails with "Provider produced inconsistent result after apply".
func Test_deviceResource_updateKeepsAsyncAppliedValues(t *testing.T) {
	client, lastPut := newLaggingAPController(t, map[string]any{
		"_id":               "66ec38ab34dbcb19dce4a6a0",
		"mac":               "60:22:32:f0:ad:b5",
		"name":              "Gartenhaus",
		"type":              "uap",
		"model":             "U6M",
		"adopted":           true,
		"state":             1,
		"led_override":      "on",
		"bandsteering_mode": "equal",
	})
	r := &deviceResource{client: client}
	s := resourceSchema(t, r)

	prior := map[string]any{
		"id":                "66ec38ab34dbcb19dce4a6a0",
		"mac":               "60:22:32:f0:ad:b5",
		"site":              "default",
		"name":              "Gartenhaus",
		"led_override":      "on",
		"bandsteering_mode": "equal",
	}
	planned := map[string]any{}
	for k, v := range prior {
		planned[k] = v
	}
	planned["led_override"] = "off"
	planned["bandsteering_mode"] = "prefer_5g"

	resp := &fwresource.UpdateResponse{State: tfState(t, s, prior), Identity: tfIdentity(t, r)}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan:   tfPlan(t, s, planned),
		Config: tfConfig(t, s, planned),
		State:  tfState(t, s, prior),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}

	put := lastPut()
	if put["bandsteering_mode"] != "prefer_5g" || put["led_override"] != "off" {
		t.Errorf("PUT = %v, want the planned band steering and LED", put)
	}
	for attr, want := range map[string]string{
		"bandsteering_mode": "prefer_5g",
		"led_override":      "off",
	} {
		var got types.String
		resp.State.GetAttribute(context.Background(), path.Root(attr), &got)
		if got.ValueString() != want {
			t.Errorf("state %s = %q, want the planned %q", attr, got.ValueString(), want)
		}
	}
}
