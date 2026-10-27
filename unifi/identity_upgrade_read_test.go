package unifi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestRefreshWithoutStoredIdentity reproduces the first refresh on Terraform
// 1.12 or newer of a state written by Terraform < 1.12 (or migrated from
// paultyng/unifi), which carries no resource identity. Terraform still asks
// the provider to upgrade the identity from version 0, with no stored JSON,
// and sends the result to Read. The refresh must succeed and derive the
// identity from the state instead of failing with "Unexpected Identity
// Change" (unifi_firewall_group, unifi_firewall_rule) or keeping a null id
// (unifi_network, unifi_wlan).
//
// It also covers the state v0.60.0 left behind after such a refresh: a
// version 1 identity with a null id, which must be repaired rather than kept.
func TestRefreshWithoutStoredIdentity(t *testing.T) {
	ctx := context.Background()

	// rest wraps a controller object in the classic REST response envelope.
	rest := func(object string) string {
		return `{"meta":{"rc":"ok"},"data":[` + object + `]}`
	}

	// The cases cover the three ways Read treats an incoming identity:
	// rebuilt from state (firewall_group, firewall_rule), passed through
	// unchanged (network, static_route, and wlan, which already treated a
	// fully null identity as absent), and only set when none is stored
	// (dns_record).
	cases := []struct {
		typeName string
		id       string
		path     string
		body     string
		state    map[string]tftypes.Value
	}{
		{
			typeName: "unifi_firewall_group",
			id:       "fg1",
			path:     "/api/s/default/rest/firewallgroup/fg1",
			body: rest(
				`{"_id":"fg1","name":"group","group_type":"address-group","group_members":["192.168.1.10"]}`,
			),
			state: map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "group"),
				"type": tftypes.NewValue(tftypes.String, "address-group"),
			},
		},
		{
			typeName: "unifi_firewall_rule",
			id:       "fr1",
			path:     "/api/s/default/rest/firewallrule/fr1",
			body: rest(
				`{"_id":"fr1","name":"rule","ruleset":"LAN_IN","rule_index":2000,"action":"accept","protocol":"all","enabled":true}`,
			),
			state: map[string]tftypes.Value{
				"name":    tftypes.NewValue(tftypes.String, "rule"),
				"ruleset": tftypes.NewValue(tftypes.String, "LAN_IN"),
			},
		},
		{
			typeName: "unifi_network",
			id:       "net1",
			path:     "/api/s/default/rest/networkconf/net1",
			body: rest(
				`{"_id":"net1","name":"Main","purpose":"corporate","vlan":10,"vlan_enabled":true,"ip_subnet":"10.0.10.1/24"}`,
			),
			state: map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "Main"),
			},
		},
		{
			typeName: "unifi_static_route",
			id:       "sr1",
			path:     "/api/s/default/rest/routing/sr1",
			body: rest(
				`{"_id":"sr1","name":"route","type":"static-route","static-route_type":"nexthop-route","static-route_network":"10.1.0.0/24","static-route_nexthop":"10.0.0.1","enabled":true}`,
			),
			state: map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "route"),
			},
		},
		{
			typeName: "unifi_wlan",
			id:       "wlan1",
			path:     "/api/s/default/rest/wlanconf/wlan1",
			body:     rest(`{"_id":"wlan1","name":"ssid","security":"wpapsk","enabled":true}`),
			state: map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "ssid"),
			},
		},
		{
			typeName: "unifi_dns_record",
			id:       "dns1",
			path:     "/v2/api/site/default/static-dns",
			body:     `[{"_id":"dns1","key":"host.example.com","record_type":"A","value":"10.0.0.5","enabled":true}]`,
			state: map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "host.example.com"),
			},
		},
	}

	responses := make(map[string]string, len(cases))
	for _, tc := range cases {
		responses[tc.path] = tc.body
	}
	srv := newConfiguredProviderServer(t, newFakeController(t, responses))

	schemas, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	identitySchemas, err := srv.GetResourceIdentitySchemas(
		ctx,
		&tfprotov6.GetResourceIdentitySchemasRequest{},
	)
	if err != nil {
		t.Fatalf("GetResourceIdentitySchemas: %v", err)
	}

	stored := map[string]struct {
		version int64
		raw     *tfprotov6.RawState
	}{
		"none":                 {0, &tfprotov6.RawState{}},
		"null id from v0.60.0": {1, &tfprotov6.RawState{JSON: []byte(`{"id":null,"site":"default"}`)}},
	}

	for _, tc := range cases {
		for storedName, si := range stored {
			t.Run(tc.typeName+"/"+storedName, func(t *testing.T) {
				upgraded, err := srv.UpgradeResourceIdentity(
					ctx,
					&tfprotov6.UpgradeResourceIdentityRequest{
						TypeName:    tc.typeName,
						Version:     si.version,
						RawIdentity: si.raw,
					},
				)
				if err != nil {
					t.Fatalf("UpgradeResourceIdentity: %v", err)
				}
				failOnErrorDiagnostics(t, "upgrade", upgraded.Diagnostics)

				stateType := schemas.ResourceSchemas[tc.typeName].ValueType()
				values := map[string]tftypes.Value{
					"id":   tftypes.NewValue(tftypes.String, tc.id),
					"site": tftypes.NewValue(tftypes.String, "default"),
				}
				for k, v := range tc.state {
					values[k] = v
				}
				state, err := tfprotov6.NewDynamicValue(
					stateType,
					objectWithNulls(t, stateType, values),
				)
				if err != nil {
					t.Fatalf("state: %v", err)
				}

				read, err := srv.ReadResource(ctx, &tfprotov6.ReadResourceRequest{
					TypeName:        tc.typeName,
					CurrentState:    &state,
					CurrentIdentity: upgraded.UpgradedIdentity,
				})
				if err != nil {
					t.Fatalf("ReadResource: %v", err)
				}
				failOnErrorDiagnostics(t, "read", read.Diagnostics)

				identityType := identitySchemas.IdentitySchemas[tc.typeName].ValueType()
				got, err := read.NewIdentity.IdentityData.Unmarshal(identityType)
				if err != nil {
					t.Fatalf("unmarshal identity: %v", err)
				}
				var attrs map[string]tftypes.Value
				if err := got.As(&attrs); err != nil {
					t.Fatalf("identity is not an object: %v", err)
				}
				for attr, want := range map[string]string{"id": tc.id, "site": "default"} {
					var s string
					if err := attrs[attr].As(&s); err != nil || s != want {
						t.Errorf("identity %s = %v, want %q", attr, attrs[attr], want)
					}
				}
			})
		}
	}
}

// newConfiguredProviderServer returns the real provider server configured
// against the controller at url.
func newConfiguredProviderServer(t *testing.T, url string) tfprotov6.ProviderServer {
	t.Helper()
	ctx := context.Background()

	srv, err := providerserver.NewProtocol6WithError(New())()
	if err != nil {
		t.Fatalf("provider server: %v", err)
	}
	schemas, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	configType := schemas.Provider.ValueType()
	config, err := tfprotov6.NewDynamicValue(configType, objectWithNulls(t, configType,
		map[string]tftypes.Value{
			"api_url":  tftypes.NewValue(tftypes.String, url),
			"username": tftypes.NewValue(tftypes.String, "admin"),
			"password": tftypes.NewValue(tftypes.String, "admin"),
		}))
	if err != nil {
		t.Fatalf("provider config: %v", err)
	}
	resp, err := srv.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: &config})
	if err != nil {
		t.Fatalf("ConfigureProvider: %v", err)
	}
	failOnErrorDiagnostics(t, "configure", resp.Diagnostics)
	return srv
}

// newFakeController starts a minimal old-style controller (302 on /, cookie
// login at /api/login) that answers a GET of each path in responses with its
// body, and returns its URL.
func newFakeController(t *testing.T, responses map[string]string) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/manage", http.StatusFound)
			return
		}
		body, ok := responses[r.URL.Path]
		if !ok || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "unifises", Value: "fake-session", Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

// objectWithNulls builds a value of the object type typ with the given
// top-level attributes and every other attribute null.
func objectWithNulls(
	t *testing.T,
	typ tftypes.Type,
	values map[string]tftypes.Value,
) tftypes.Value {
	t.Helper()
	obj, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("type is %T, want object", typ)
	}
	attrs := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, attrType := range obj.AttributeTypes {
		if v, ok := values[name]; ok {
			attrs[name] = v
			continue
		}
		attrs[name] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(obj, attrs)
}

func failOnErrorDiagnostics(t *testing.T, step string, diags []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s diagnostic: %s: %s", step, d.Summary, d.Detail)
		}
	}
}
