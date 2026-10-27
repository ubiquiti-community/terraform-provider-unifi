package unifi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const (
	fakeFixedAPClientID  = "65f0c0ffee00000000000c01"
	fakeFixedAPClientMAC = "02:00:00:00:0c:01"
	fakeStaleFixedAPMAC  = "02:00:00:00:0a:01"
)

// fakeFixedAPClientDocument is a client whose fixed AP was turned off in the
// UI: the controller keeps echoing the last fixed_ap_mac.
const fakeFixedAPClientDocument = `{
	"_id": "` + fakeFixedAPClientID + `",
	"site_id": "site",
	"mac": "` + fakeFixedAPClientMAC + `",
	"name": "Test Client",
	"fixed_ap_enabled": false,
	"fixed_ap_mac": "` + fakeStaleFixedAPMAC + `",
	"use_fixedip": false,
	"local_dns_record_enabled": false
}`

// newFixedAPClientFakeControllerClient serves fakeFixedAPClientDocument from
// a minimal fake controller (old-style API: 302 on /, cookie login at
// /api/login) and records the body of every PUT to it.
func newFixedAPClientFakeControllerClient(t *testing.T, puts *[]map[string]any) *Client {
	t.Helper()

	reply := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(`{"meta":{"rc":"ok"},"data":[` + fakeFixedAPClientDocument + `]}`),
		)
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
	})
	mux.HandleFunc("/api/s/default/rest/user", func(w http.ResponseWriter, r *http.Request) {
		reply(w)
	})
	mux.HandleFunc(
		"/api/s/default/rest/user/"+fakeFixedAPClientID,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("reading PUT body: %v", err)
				}
				var body map[string]any
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Errorf("decoding PUT body: %v", err)
				}
				*puts = append(*puts, body)
			}
			reply(w)
		},
	)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL:  srv.URL,
		Username: "admin",
		Password: "admin",
	})
	if err != nil {
		t.Fatalf("creating client against fake controller: %v", err)
	}
	return &Client{ApiClient: apiClient, Site: "default"}
}

// TestClientUpdateLeavesDisabledFixedAPOff: the controller keeps echoing the
// last fixed_ap_mac after fixed AP is turned off, and the provider used to
// import that stale MAC. An unrelated update then sent it back with
// fixed_ap_enabled=true and pinned the client to the AP (the fixed_ip and
// local_dns_record echoes behind #386 and #387 are the same defect). An
// imported client must keep fixed_ap_mac null and an update must keep fixed
// AP off.
func TestClientUpdateLeavesDisabledFixedAPOff(t *testing.T) {
	ctx := context.Background()
	var puts []map[string]any
	r := &clientResource{client: newFixedAPClientFakeControllerClient(t, &puts)}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	var identityResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)
	newIdentity := func() *tfsdk.ResourceIdentity {
		return &tfsdk.ResourceIdentity{
			Raw: tftypes.NewValue(
				identityResp.IdentitySchema.Type().TerraformType(ctx), nil,
			),
			Schema: identityResp.IdentitySchema,
		}
	}

	importResp := &fwresource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			Schema: schemaResp.Schema,
		},
		Identity: newIdentity(),
	}
	r.ImportState(ctx, fwresource.ImportStateRequest{ID: fakeFixedAPClientMAC}, importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", importResp.Diagnostics)
	}

	readResp := &fwresource.ReadResponse{State: importResp.State, Identity: newIdentity()}
	r.Read(ctx, fwresource.ReadRequest{
		State:    importResp.State,
		Identity: importResp.Identity,
	}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}

	var state clientResourceModel
	if d := readResp.State.Get(ctx, &state); d.HasError() {
		t.Fatalf("reading state: %v", d)
	}
	if !state.FixedApMAC.IsNull() {
		t.Errorf("imported fixed_ap_mac = %v, want null while fixed AP is disabled",
			state.FixedApMAC)
	}

	// An unrelated change: only the name differs from the import.
	plan := state
	plan.Name = types.StringValue("Test Client (renamed)")
	planned := tfsdk.Plan{Schema: schemaResp.Schema}
	if d := planned.Set(ctx, &plan); d.HasError() {
		t.Fatalf("building plan: %v", d)
	}
	updateResp := &fwresource.UpdateResponse{State: readResp.State, Identity: newIdentity()}
	r.Update(ctx, fwresource.UpdateRequest{
		State:  readResp.State,
		Plan:   planned,
		Config: tfsdk.Config(planned),
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", updateResp.Diagnostics)
	}

	if len(puts) != 1 {
		t.Fatalf("PUT requests = %d, want 1", len(puts))
	}
	body := puts[0]
	if body["fixed_ap_enabled"] != false {
		t.Errorf("PUT fixed_ap_enabled = %v, want false so fixed AP stays off",
			body["fixed_ap_enabled"])
	}
	if mac, ok := body["fixed_ap_mac"]; ok {
		t.Errorf("PUT fixed_ap_mac = %v, want it left out", mac)
	}

	var updated clientResourceModel
	if d := updateResp.State.Get(ctx, &updated); d.HasError() {
		t.Fatalf("reading updated state: %v", d)
	}
	if !updated.FixedApMAC.IsNull() {
		t.Errorf("fixed_ap_mac after update = %v, want null", updated.FixedApMAC)
	}
}

// An enabled fixed AP still surfaces its MAC.
func TestClientToModel_FixedAPEnabledKeepsValue(t *testing.T) {
	r := &clientResource{}
	client := &unifi.Client{
		MAC:            fakeFixedAPClientMAC,
		FixedApEnabled: true,
		FixedApMAC:     fakeStaleFixedAPMAC,
	}

	var model clientResourceModel
	if diags := r.clientToModel(context.Background(), client, &model, "default"); diags.HasError() {
		t.Fatalf("clientToModel returned errors: %v", diags)
	}
	if model.FixedApMAC.ValueString() != fakeStaleFixedAPMAC {
		t.Errorf("fixed_ap_mac = %v, want %s", model.FixedApMAC, fakeStaleFixedAPMAC)
	}
}
