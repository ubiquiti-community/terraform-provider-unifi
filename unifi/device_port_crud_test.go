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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const (
	fakeSwitchID  = "66ec38ab34dbcb19dce4a6b0"
	fakeSwitchMAC = "00:27:22:00:00:01"
)

// fakeSwitch serves one connected switch with four ports. A PUT merges its
// top-level keys into the device (port_overrides is replaced whole, as the
// controller does) and every PUT body is recorded.
type fakeSwitch struct {
	mu     sync.Mutex
	device map[string]any
	gone   bool
	reject bool
	puts   []map[string]any
}

func newFakeSwitch(t *testing.T, overrides ...map[string]any) (*fakeSwitch, *Client) {
	t.Helper()
	ports := make([]any, 0, 4)
	for i := 1; i <= 4; i++ {
		ports = append(ports, map[string]any{"port_idx": i, "name": "Port", "up": true})
	}
	po := make([]any, 0, len(overrides))
	for _, o := range overrides {
		po = append(po, o)
	}
	f := &fakeSwitch{device: map[string]any{
		"_id":            fakeSwitchID,
		"mac":            fakeSwitchMAC,
		"name":           "sw",
		"type":           "usw",
		"model":          "US48P750",
		"adopted":        true,
		"state":          1,
		"port_table":     ports,
		"port_overrides": po,
	}}

	reply := func(w http.ResponseWriter, data []any) {
		w.Header().Set("Content-Type", "application/json")
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
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "unifises", Value: "fake-session", Path: "/"})
		reply(w, []any{})
	})
	mux.HandleFunc("/api/s/default/stat/device/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		k := strings.TrimPrefix(r.URL.Path, "/api/s/default/stat/device/")
		if f.gone || (k != fakeSwitchMAC && k != fakeSwitchID) {
			reply(w, []any{})
			return
		}
		reply(w, []any{f.device})
	})
	mux.HandleFunc("/api/s/default/rest/device/"+fakeSwitchID,
		func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			var body map[string]any
			if err := json.Unmarshal(b, &body); err != nil {
				t.Errorf("PUT body is not JSON: %v", err)
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			f.puts = append(f.puts, body)
			if f.reject {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"meta":{"rc":"error","msg":"api.err.Invalid"},"data":[]}`))
				return
			}
			for k, v := range body {
				f.device[k] = v
			}
			reply(w, []any{f.device})
		})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL: srv.URL, Username: "admin", Password: "admin",
	})
	if err != nil {
		t.Fatalf("creating client against fake controller: %v", err)
	}
	return f, &Client{ApiClient: apiClient, Site: "default"}
}

func (f *fakeSwitch) overrides() []any {
	f.mu.Lock()
	defer f.mu.Unlock()
	po, _ := f.device["port_overrides"].([]any)
	return po
}

func (f *fakeSwitch) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.puts)
}

func overrideAt(po []any, index float64) map[string]any {
	for _, e := range po {
		if m, ok := e.(map[string]any); ok && m["port_idx"] == index {
			return m
		}
	}
	return nil
}

// devicePortStateAttr reads a top-level string attribute of a state.
func devicePortStateAttr(t *testing.T, s tfsdk.State, name string) types.String {
	t.Helper()
	var v types.String
	if d := s.GetAttribute(context.Background(), path.Root(name), &v); d.HasError() {
		t.Fatalf("reading %s: %v", name, d)
	}
	return v
}

func devicePortCreate(
	t *testing.T,
	client *Client,
	planned map[string]any,
) *fwresource.CreateResponse {
	t.Helper()
	r := &devicePortResource{client: client}
	s := resourceSchema(t, r)
	resp := &fwresource.CreateResponse{State: tfState(t, s, nil), Identity: tfIdentity(t, r)}
	r.Create(context.Background(), fwresource.CreateRequest{
		Plan:   tfPlan(t, s, planned),
		Config: tfConfig(t, s, planned),
	}, resp)
	return resp
}

func devicePortRead(
	t *testing.T,
	client *Client,
	prior map[string]any,
	withIdentity bool,
) *fwresource.ReadResponse {
	t.Helper()
	r := &devicePortResource{client: client}
	s := resourceSchema(t, r)
	req := fwresource.ReadRequest{State: tfState(t, s, prior)}
	resp := &fwresource.ReadResponse{State: tfState(t, s, prior), Identity: tfIdentity(t, r)}
	if withIdentity {
		id := tfIdentity(t, r)
		d := id.Set(context.Background(), devicePortIdentity(fakeSwitchMAC, 2, "default"))
		if d.HasError() {
			t.Fatalf("identity: %v", d)
		}
		req.Identity = id
		resp.Identity = id
	}
	r.Read(context.Background(), req, resp)
	return resp
}

// Creating a port that has no override yet appends its entry and keeps the
// entries of the other ports.
func TestDevicePortCreate_appendsEntry(t *testing.T) {
	sw, client := newFakeSwitch(t, map[string]any{"port_idx": 1, "name": "other"})
	resp := devicePortCreate(t, client, map[string]any{
		"device_mac": fakeSwitchMAC,
		"index":      int64(3),
		"name":       "nas",
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	po := sw.overrides()
	if got := overrideAt(po, 3); got == nil || got["name"] != "nas" {
		t.Errorf("override of port 3 = %v, want name nas", got)
	}
	if got := overrideAt(po, 1); got == nil || got["name"] != "other" {
		t.Errorf("override of port 1 = %v, want it kept", got)
	}
	if got := devicePortStateAttr(t, resp.State, "id").ValueString(); got != fakeSwitchMAC+"/3" {
		t.Errorf("id = %q", got)
	}
	if got := devicePortStateAttr(t, resp.State, "site").ValueString(); got != "default" {
		t.Errorf("site = %q", got)
	}
	if got := devicePortStateAttr(t, resp.State, "name").ValueString(); got != "nas" {
		t.Errorf("name = %q", got)
	}
	var identity devicePortIdentityModel
	if d := resp.Identity.Get(context.Background(), &identity); d.HasError() {
		t.Fatalf("identity: %v", d)
	}
	if identity.Index.ValueInt64() != 3 || identity.Site.ValueString() != "default" {
		t.Errorf("identity = %+v", identity)
	}
}

func TestDevicePortCreate_errors(t *testing.T) {
	cases := []struct {
		name    string
		mac     string
		index   int64
		setup   func(*fakeSwitch)
		wantErr string
		wantPut bool
	}{
		{"port not on the device", fakeSwitchMAC, 9, nil, "Port Not Found", false},
		{"unknown device", "00:27:22:00:00:99", 1, nil, "Device Not Found", false},
		{
			"controller rejects the write", fakeSwitchMAC, 2,
			func(f *fakeSwitch) { f.reject = true }, "Error Updating Device Port", true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sw, client := newFakeSwitch(t)
			if tc.setup != nil {
				tc.setup(sw)
			}
			resp := devicePortCreate(t, client, map[string]any{
				"device_mac": tc.mac,
				"index":      tc.index,
				"name":       "x",
			})
			if !resp.Diagnostics.HasError() ||
				resp.Diagnostics.Errors()[0].Summary() != tc.wantErr {
				t.Fatalf("diagnostics = %v, want %q", resp.Diagnostics, tc.wantErr)
			}
			if got := sw.putCount() > 0; got != tc.wantPut {
				t.Errorf("PUT sent = %v, want %v", got, tc.wantPut)
			}
		})
	}
}

// An update rewrites only the managed port's entry.
func TestDevicePortUpdate(t *testing.T) {
	sw, client := newFakeSwitch(t,
		map[string]any{"port_idx": 1, "name": "other"},
		map[string]any{"port_idx": 2, "name": "old"},
	)
	r := &devicePortResource{client: client}
	s := resourceSchema(t, r)
	prior := map[string]any{
		"id":         fakeSwitchMAC + "/2",
		"site":       "default",
		"device_mac": fakeSwitchMAC,
		"index":      int64(2),
		"name":       "old",
	}
	planned := map[string]any{}
	for k, v := range prior {
		planned[k] = v
	}
	planned["name"] = "new"

	resp := &fwresource.UpdateResponse{State: tfState(t, s, prior), Identity: tfIdentity(t, r)}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan:   tfPlan(t, s, planned),
		Config: tfConfig(t, s, planned),
		State:  tfState(t, s, prior),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	po := sw.overrides()
	if got := overrideAt(po, 2); got == nil || got["name"] != "new" {
		t.Errorf("override of port 2 = %v, want name new", got)
	}
	if got := overrideAt(po, 1); got == nil || got["name"] != "other" {
		t.Errorf("override of port 1 = %v, want it kept", got)
	}
	if got := devicePortStateAttr(t, resp.State, "name").ValueString(); got != "new" {
		t.Errorf("name = %q", got)
	}
}

func TestDevicePortRead(t *testing.T) {
	managed := map[string]any{
		"id":         fakeSwitchMAC + "/2",
		"site":       "default",
		"device_mac": fakeSwitchMAC,
		"index":      int64(2),
		"name":       "configured",
	}
	imported := map[string]any{"device_mac": fakeSwitchMAC, "index": int64(2), "site": "default"}

	t.Run("drift on a configured attribute", func(t *testing.T) {
		_, client := newFakeSwitch(t, map[string]any{"port_idx": 2, "name": "changed"})
		resp := devicePortRead(t, client, managed, false)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		if got := devicePortStateAttr(t, resp.State, "name").ValueString(); got != "changed" {
			t.Errorf("name = %q, want the controller's value", got)
		}
		if resp.Identity.Raw.IsNull() {
			t.Error("identity not set")
		}
	})

	t.Run("override entry cleared on the controller", func(t *testing.T) {
		_, client := newFakeSwitch(t)
		resp := devicePortRead(t, client, managed, false)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		if got := devicePortStateAttr(t, resp.State, "name"); !got.IsNull() {
			t.Errorf("name = %v, want null so the plan writes it again", got)
		}
	})

	t.Run("device gone", func(t *testing.T) {
		sw, client := newFakeSwitch(t)
		sw.gone = true
		resp := devicePortRead(t, client, managed, false)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		if !resp.State.Raw.IsNull() {
			t.Error("state kept for a device that no longer exists")
		}
	})

	t.Run("first read after an import", func(t *testing.T) {
		_, client := newFakeSwitch(t, map[string]any{
			"port_idx": 2, "name": "from-controller", "poe_mode": "off",
		})
		resp := devicePortRead(t, client, imported, false)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		for attr, want := range map[string]string{
			"id":       fakeSwitchMAC + "/2",
			"name":     "from-controller",
			"poe_mode": "off",
			"op_mode":  "switch",
		} {
			if got := devicePortStateAttr(t, resp.State, attr).ValueString(); got != want {
				t.Errorf("%s = %q, want %q", attr, got, want)
			}
		}
	})

	t.Run("import of a port without overrides", func(t *testing.T) {
		_, client := newFakeSwitch(t)
		resp := devicePortRead(t, client, imported, false)
		if !resp.Diagnostics.HasError() ||
			resp.Diagnostics.Errors()[0].Summary() != "Port Has No Overrides" {
			t.Fatalf("diagnostics = %v, want Port Has No Overrides", resp.Diagnostics)
		}
	})

	t.Run("keys from the identity only", func(t *testing.T) {
		_, client := newFakeSwitch(t, map[string]any{"port_idx": 2, "name": "by-identity"})
		resp := devicePortRead(t, client, map[string]any{}, true)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		if got := devicePortStateAttr(t, resp.State, "name").ValueString(); got != "by-identity" {
			t.Errorf("name = %q", got)
		}
	})

	t.Run("no keys at all", func(t *testing.T) {
		_, client := newFakeSwitch(t)
		resp := devicePortRead(t, client, map[string]any{}, false)
		if !resp.Diagnostics.HasError() ||
			resp.Diagnostics.Errors()[0].Summary() != "Incomplete Device Port State" {
			t.Fatalf("diagnostics = %v, want Incomplete Device Port State", resp.Diagnostics)
		}
	})
}

// Destroy leaves the port's settings on the controller and says so.
func TestDevicePortDelete(t *testing.T) {
	sw, client := newFakeSwitch(t, map[string]any{"port_idx": 2, "name": "kept"})
	r := &devicePortResource{client: client}
	s := resourceSchema(t, r)
	prior := map[string]any{
		"id": fakeSwitchMAC + "/2", "site": "default",
		"device_mac": fakeSwitchMAC, "index": int64(2), "name": "kept",
	}
	resp := &fwresource.DeleteResponse{State: tfState(t, s, prior)}
	r.Delete(context.Background(), fwresource.DeleteRequest{State: tfState(t, s, prior)}, resp)
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("diagnostics = %v, want one warning", resp.Diagnostics)
	}
	if sw.putCount() != 0 {
		t.Error("destroy wrote to the controller")
	}
}

func TestDevicePortConfigure(t *testing.T) {
	r := &devicePortResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{}, resp)
	if resp.Diagnostics.HasError() || r.client != nil {
		t.Fatalf("nil provider data: %v", resp.Diagnostics)
	}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "nope"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("wrong provider data type accepted")
	}
	resp = &fwresource.ConfigureResponse{}
	c := &Client{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() || r.client != c {
		t.Fatalf("client not set: %v", resp.Diagnostics)
	}
}
