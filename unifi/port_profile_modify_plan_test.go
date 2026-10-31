package unifi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// newPortProfileFakeControllerClient serves two networks on the default site:
// "untagged" (no VLAN id, like the site default) and "tagged" (VLAN 20). Any
// other id reads as not found.
func newPortProfileFakeControllerClient(t *testing.T) *Client {
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
		"/api/s/default/rest/networkconf/{id}",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.PathValue("id") {
			case "untagged":
				_, _ = w.Write(
					[]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"untagged","name":"Default"}]}`),
				)
			case "tagged":
				_, _ = w.Write(
					[]byte(
						`{"meta":{"rc":"ok"},"data":[{"_id":"tagged","name":"Trunk","vlan":20}]}`,
					),
				)
			default:
				_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
			}
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

// portProfileRaw builds a port profile object with every attribute null except
// the given ones. A nil map gives a null object (no prior state).
func portProfileRaw(objType tftypes.Object, set map[string]any) tftypes.Value {
	if set == nil {
		return tftypes.NewValue(objType, nil)
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(at, nil)
	}
	for name, v := range set {
		vals[name] = tftypes.NewValue(objType.AttributeTypes[name], v)
	}
	return tftypes.NewValue(objType, vals)
}

// Test_portProfileResource_ModifyPlan_customize drives the customize guard
// (#496) through ModifyPlan, including a tagged_vlan_mgmt that config omits
// and that plans unknown because another attribute changed (#584).
func Test_portProfileResource_ModifyPlan_customize(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{client: newPortProfileFakeControllerClient(t)}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is not an object")
	}

	unknown := tftypes.UnknownValue

	tests := []struct {
		name      string
		forward   any
		nativeID  any
		planned   any
		config    any
		prior     map[string]any
		wantError bool
	}{
		{
			name:     "omitted, stored custom, untagged native: allowed",
			forward:  "customize",
			nativeID: "untagged",
			planned:  unknown,
			prior:    map[string]any{"tagged_vlan_mgmt": "custom"},
		},
		{
			name:      "omitted, stored auto, untagged native: refused",
			forward:   "customize",
			nativeID:  "untagged",
			planned:   unknown,
			prior:     map[string]any{"tagged_vlan_mgmt": "auto"},
			wantError: true,
		},
		{
			name:      "create with mode omitted, untagged native: refused",
			forward:   "customize",
			nativeID:  "untagged",
			planned:   unknown,
			wantError: true,
		},
		{
			name:     "unknown in config, stored auto: not judged",
			forward:  "customize",
			nativeID: "untagged",
			planned:  unknown,
			config:   unknown,
			prior:    map[string]any{"tagged_vlan_mgmt": "auto"},
		},
		{
			name:     "planned custom, untagged native: allowed",
			forward:  "customize",
			nativeID: "untagged",
			planned:  "custom",
			config:   "custom",
		},
		{
			name:     "planned auto, tagged native: allowed",
			forward:  "customize",
			nativeID: "tagged",
			planned:  "auto",
			config:   "auto",
		},
		{
			name:     "forward all: not judged",
			forward:  "all",
			nativeID: "untagged",
			planned:  "auto",
			config:   "auto",
		},
		{
			name:     "unknown native network: not judged",
			forward:  "customize",
			nativeID: unknown,
			planned:  "auto",
			config:   "auto",
		},
		{
			name:     "native network lookup fails: not judged",
			forward:  "customize",
			nativeID: "missing",
			planned:  "auto",
			config:   "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := tfsdk.Plan{
				Schema: schemaResp.Schema,
				Raw: portProfileRaw(objType, map[string]any{
					"forward":               tt.forward,
					"native_networkconf_id": tt.nativeID,
					"tagged_vlan_mgmt":      tt.planned,
				}),
			}
			config := tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw: portProfileRaw(objType, map[string]any{
					"forward":               tt.forward,
					"native_networkconf_id": tt.nativeID,
					"tagged_vlan_mgmt":      tt.config,
				}),
			}
			state := tfsdk.State{
				Schema: schemaResp.Schema,
				Raw:    portProfileRaw(objType, tt.prior),
			}

			req := fwresource.ModifyPlanRequest{Plan: plan, Config: config, State: state}
			resp := &fwresource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, req, resp)

			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error diagnostic, got none")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}
