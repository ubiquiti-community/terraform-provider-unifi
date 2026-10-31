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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const fakeWANID = "65f0c0ffee00000000000a01"

// fakeWANDocument is a WAN networkconf document as Network 10.6 returns it:
// there is no `enabled` key at all, and the IGMP proxy serves "some"
// downstream networks.
const fakeWANDocument = `{
	"_id": "` + fakeWANID + `",
	"site_id": "site",
	"attr_hidden_id": "WAN",
	"attr_no_delete": true,
	"name": "Internet 1",
	"purpose": "wan",
	"wan_networkgroup": "WAN",
	"wan_type": "dhcp",
	"wan_type_v6": "disabled",
	"wan_vlan_enabled": false,
	"wan_smartq_enabled": false,
	"wan_dhcpv6_pd_size_auto": true,
	"wan_dslite_remote_host_auto": true,
	"wan_dns_preference": "auto",
	"wan_ip_aliases": [],
	"wan_dhcp_options": [],
	"igmp_proxy_for": "some",
	"igmp_proxy_upstream": true,
	"igmp_proxy_downstream_networkconf_ids": ["lan-a", "lan-b"],
	"report_wan_event": false,
	"setting_preference": "auto",
	"mac_override_enabled": false
}`

// newWANFakeControllerClient serves fakeWANDocument from a minimal fake
// controller (old-style API: 302 on /, cookie login at /api/login) and records
// the body of every PUT to it.
func newWANFakeControllerClient(t *testing.T, puts *[]map[string]any) *Client {
	t.Helper()

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
	mux.HandleFunc(
		"/api/s/default/rest/networkconf/"+fakeWANID,
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
			// The controller answers both with the stored document, which
			// still has no `enabled` key.
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[` + fakeWANDocument + `]}`))
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

// TestWANUpdateKeepsOmittedEnabledOn covers #600. The controller returns no
// `enabled` key for a WAN, and the provider used to import `enabled = false`
// and then send it with every update, which switches the uplink off. An
// imported WAN must read as enabled, and an unrelated update must not send
// `"enabled": false`.
func TestWANUpdateKeepsOmittedEnabledOn(t *testing.T) {
	ctx := context.Background()
	var puts []map[string]any
	r := &wanResource{client: newWANFakeControllerClient(t, &puts)}

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
	r.ImportState(ctx, fwresource.ImportStateRequest{ID: fakeWANID}, importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", importResp.Diagnostics)
	}

	readResp := &fwresource.ReadResponse{State: importResp.State, Identity: newIdentity()}
	r.Read(ctx, fwresource.ReadRequest{State: importResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}

	var state wanResourceModel
	if d := readResp.State.Get(ctx, &state); d.HasError() {
		t.Fatalf("reading state: %v", d)
	}
	if !state.Enabled.Equal(types.BoolValue(true)) {
		t.Errorf("imported enabled = %v, want true for a WAN the controller reports without it",
			state.Enabled)
	}
	var igmp igmpProxyModel
	if d := state.IGMPProxy.As(ctx, &igmp, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("reading igmp_proxy: %v", d)
	}
	if igmp.Downstream.ValueString() != "some" {
		t.Errorf("igmp_proxy.downstream = %v, want some", igmp.Downstream)
	}

	// An unrelated change: only report_wan_event differs from the import.
	plan := state
	plan.ReportWANEvent = types.BoolValue(true)
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
	// The update is sparse: it sends the changed key and leaves enabled, which
	// did not change, to the controller rather than switching the uplink off.
	body := puts[0]
	if body["report_wan_event"] != true {
		t.Errorf("PUT report_wan_event = %v, want true", body["report_wan_event"])
	}
	if v, ok := body["enabled"]; ok && v != true {
		t.Errorf("PUT enabled = %v, want it left out or true so the uplink stays up", v)
	}
	if v, ok := body["igmp_proxy_for"]; ok && v != "some" {
		t.Errorf("PUT igmp_proxy_for = %v, want it left out or some", v)
	}
	if _, ok := body["igmp_proxy_downstream_networkconf_ids"]; ok {
		t.Error("PUT carries igmp_proxy_downstream_networkconf_ids, want it left to the controller")
	}

	var updated wanResourceModel
	if d := updateResp.State.Get(ctx, &updated); d.HasError() {
		t.Fatalf("reading updated state: %v", d)
	}
	if !updated.Enabled.Equal(types.BoolValue(true)) {
		t.Errorf("enabled after update = %v, want true", updated.Enabled)
	}
}

// A WAN the configuration explicitly disables keeps `enabled = false`: when
// the controller omits the key, a known prior value must not be flipped back
// to true.
func TestWANNetworkToModelKeepsExplicitEnabled(t *testing.T) {
	ctx := context.Background()
	r := &wanResource{}
	network := &unifi.Network{ID: fakeWANID, Purpose: unifi.PurposeWAN}

	for _, tc := range []struct {
		name  string
		prior types.Bool
		want  bool
	}{
		{"import", types.BoolNull(), true},
		{"enabled", types.BoolValue(true), true},
		{"explicitly disabled", types.BoolValue(false), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := &wanResourceModel{Enabled: tc.prior}
			applyWANDefaults(model)
			if d := r.networkToModel(ctx, network, model, "default"); d.HasError() {
				t.Fatalf("networkToModel: %v", d)
			}
			if !model.Enabled.Equal(types.BoolValue(tc.want)) {
				t.Errorf("enabled = %v, want %v", model.Enabled, tc.want)
			}
		})
	}
}

// An `enabled` key the controller does report wins over the prior value.
func TestWANNetworkToModelReadsReportedEnabled(t *testing.T) {
	ctx := context.Background()
	r := &wanResource{}

	for _, tc := range []struct {
		name     string
		reported bool
		prior    types.Bool
	}{
		{"reported false on import", false, types.BoolNull()},
		{"reported false over prior true", false, types.BoolValue(true)},
		{"reported true over prior false", true, types.BoolValue(false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			network := &unifi.Network{
				ID:      fakeWANID,
				Purpose: unifi.PurposeWAN,
				Enabled: new(tc.reported),
			}
			model := &wanResourceModel{Enabled: tc.prior}
			applyWANDefaults(model)
			if d := r.networkToModel(ctx, network, model, "default"); d.HasError() {
				t.Fatalf("networkToModel: %v", d)
			}
			if !model.Enabled.Equal(types.BoolValue(tc.reported)) {
				t.Errorf("enabled = %v, want %v", model.Enabled, tc.reported)
			}
		})
	}
}
