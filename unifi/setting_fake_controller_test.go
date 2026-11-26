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

// fakeGlobalNetwork is the v2 global network settings object of a Network 11
// controller. There the mDNS network selection lives in this object: the mdns
// setting mirrors mdns_enabled_for and mdns_enabled_for_network_ids (with an
// empty list while it is "all"), and writes to the mdns setting's enabled_for
// and enabled_for_network_ids are acknowledged and ignored. Puts records each
// body PUT to the object.
type fakeGlobalNetwork struct {
	Stored map[string]any
	Puts   []map[string]any
}

func (g *fakeGlobalNetwork) mirror(mdns map[string]any) {
	if mdns == nil {
		return
	}
	mdns["enabled_for"] = g.Stored["mdns_enabled_for"]
	ids := g.Stored["mdns_enabled_for_network_ids"]
	if g.Stored["mdns_enabled_for"] == "all" {
		ids = []any{}
	}
	mdns["enabled_for_network_ids"] = ids
}

// newSettingsFakeController serves the given stored settings for site
// "default" and records every setting PUT body by key. Like the controller, it
// merges a PUT into the stored setting, so later reads see the write, and
// answers a read of a setting it has never stored with no data.
//
// A stored setting can fail on purpose with HTTP 400 (not retried by the
// client): "_fail": "read" fails every read of it, "write" every PUT, and
// "readback" the reads after it has been written.
//
// The returned function hands out the PUTs since its last call.
//
// Passing a fakeGlobalNetwork makes the fake behave like Network 11 for mDNS
// (see fakeGlobalNetwork). Without one, the v2 global network settings answer
// 404, as on earlier controllers.
func newSettingsFakeController(
	t *testing.T,
	stored []map[string]any,
	network11 ...*fakeGlobalNetwork,
) (*settingResource, func() map[string]map[string]any) {
	t.Helper()
	var g *fakeGlobalNetwork
	if len(network11) > 0 {
		g = network11[0]
	}
	var mu sync.Mutex
	puts := map[string]map[string]any{}
	byKey := map[string]map[string]any{}
	var order []string
	for _, s := range stored {
		k, _ := s["key"].(string)
		byKey[k] = s
		order = append(order, k)
	}
	written := map[string]bool{}
	fail := func(w http.ResponseWriter) {
		http.Error(w, `{"meta":{"rc":"error","msg":"api.err.Invalid"}}`, http.StatusBadRequest)
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
		case s["_fail"] == "read", s["_fail"] == "readback" && written[k]:
			fail(w)
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
		if byKey[k] != nil && byKey[k]["_fail"] == "write" {
			fail(w)
			return
		}
		puts[k] = body
		written[k] = true
		if byKey[k] == nil {
			byKey[k] = map[string]any{}
			order = append(order, k)
		}
		for bk, bv := range body {
			if g != nil && k == "mdns" &&
				(bk == "enabled_for" || bk == "enabled_for_network_ids") {
				continue // Network 11 acknowledges these and ignores them
			}
			byKey[k][bk] = bv
		}
		reply(w, []any{byKey[k]})
	})
	if g != nil {
		g.mirror(byKey["mdns"])
		mux.HandleFunc(
			"/v2/api/site/default/global/config/network",
			func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method == http.MethodPut {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("global network PUT body is not JSON: %v", err)
					}
					g.Puts = append(g.Puts, body)
					g.Stored = body
					g.mirror(byKey["mdns"])
				}
				_ = json.NewEncoder(w).Encode(g.Stored)
			},
		)
	}

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
