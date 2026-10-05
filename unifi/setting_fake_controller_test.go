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

	"github.com/ubiquiti-community/go-unifi/unifi"
)

// newSettingsFakeController serves the given stored settings for site
// "default" and records every setting PUT body by key. Like the controller, it
// merges a PUT into the stored setting, so later reads see the write, and
// answers a read of a setting it has never stored with no data. A stored
// setting with "_fail": true answers every read with HTTP 500.
//
// The returned function hands out the PUTs since its last call.
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
		s, ok := byKey[k]
		switch {
		case !ok:
			reply(w, []any{})
		case s["_fail"] == true:
			http.Error(
				w,
				`{"meta":{"rc":"error","msg":"api.err.Internal"}}`,
				http.StatusInternalServerError,
			)
		default:
			reply(w, []any{s})
		}
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
