package unifi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// newIntegrationSitesTestClient returns a Client backed by a stub new-style
// controller that serves the Integration API site listing. The counter reports
// how many times the listing was requested.
func newIntegrationSitesTestClient(t *testing.T, sitesBody string) (*Client, *int32) {
	t.Helper()
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.WriteHeader(http.StatusOK) // 200 => new-style (UniFi OS) API
		case r.URL.Path == "/proxy/network/status":
			_, _ = w.Write([]byte(`{"meta":{"server_version":"8.0.0"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/auth/login":
			w.Header().Set("X-Csrf-Token", "tok")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/proxy/network/integration/v1/sites":
			atomic.AddInt32(&hits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sitesBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL:  srv.URL,
		Username: "admin",
		Password: "admin",
	})
	if err != nil {
		t.Fatalf("client init: %v", err)
	}
	return &Client{ApiClient: apiClient, Site: "default"}, &hits
}

const integrationSitesBody = `{
	"offset": 0, "limit": 200, "count": 1, "totalCount": 1,
	"data": [
		{"id":"11111111-1111-1111-1111-111111111111","internalReference":"default","name":"Default"}
	]
}`

// TestClient_IntegrationSiteID is the regression test for sending the legacy site
// name ("default") where the Integration API requires a UUID.
func TestClient_IntegrationSiteID(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves the legacy site name to a UUID", func(t *testing.T) {
		c, _ := newIntegrationSitesTestClient(t, integrationSitesBody)

		got, err := c.IntegrationSiteID(ctx, "default")
		if err != nil {
			t.Fatalf("IntegrationSiteID: %v", err)
		}
		if want := unifi.IntegrationSiteID("11111111-1111-1111-1111-111111111111"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("caches the resolution", func(t *testing.T) {
		c, hits := newIntegrationSitesTestClient(t, integrationSitesBody)

		for range 3 {
			if _, err := c.IntegrationSiteID(ctx, "default"); err != nil {
				t.Fatalf("IntegrationSiteID: %v", err)
			}
		}
		if n := atomic.LoadInt32(hits); n != 1 {
			t.Errorf("site listing requests = %d, want 1 (cached after first resolve)", n)
		}
	})

	t.Run("does not cache failures", func(t *testing.T) {
		c, hits := newIntegrationSitesTestClient(t, integrationSitesBody)

		if _, err := c.IntegrationSiteID(ctx, "missing"); err == nil {
			t.Fatal("expected an error for an unknown site")
		}
		if _, err := c.IntegrationSiteID(ctx, "missing"); err == nil {
			t.Fatal("expected an error for an unknown site")
		}
		if n := atomic.LoadInt32(hits); n != 2 {
			t.Errorf("site listing requests = %d, want 2 (failures are retried)", n)
		}
	})

	t.Run("a UUID is used as-is without a request", func(t *testing.T) {
		c, hits := newIntegrationSitesTestClient(t, integrationSitesBody)

		const uuid = "99999999-9999-9999-9999-999999999999"
		got, err := c.IntegrationSiteID(ctx, uuid)
		if err != nil {
			t.Fatalf("IntegrationSiteID: %v", err)
		}
		if got != unifi.IntegrationSiteID(uuid) {
			t.Errorf("got %q, want %q", got, uuid)
		}
		if n := atomic.LoadInt32(hits); n != 0 {
			t.Errorf("site listing requests = %d, want 0 for a UUID", n)
		}
	})
}

func Test_siteResolveError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantSummary string
		wantInside  []string
		wantAbsent  []string
	}{
		{
			name: "site not found points at the site setting",
			err: &unifi.NotFoundError{
				Type: "IntegrationSite", Attr: "name", Value: "default",
			},
			wantSummary: "Site Not Found",
			wantInside:  []string{`"default"`, "`site`", "UUID"},
		},
		{
			name: "ambiguous site asks for a UUID",
			err: fmt.Errorf(
				"%w: site %q is ambiguous: 2 Integration API sites share that name; use the site UUID",
				unifi.ErrAmbiguousIntegrationSite,
				"Office",
			),
			wantSummary: "Ambiguous Site",
			wantInside:  []string{"ambiguous", "`site`", "UUID"},
		},
		{
			name: "network failure is not blamed on the site",
			err: errors.New(
				"unable to perform request: GET integration/v1/sites Get " +
					`"https://api.ui.com/v1/connector/consoles/X/proxy/network/integration/v1/sites?limit=200&offset=0": ` +
					"dial tcp: lookup api.ui.com: no such host",
			),
			wantSummary: "Error Looking Up Site",
			wantInside:  []string{"no such host", `"default"`},
			wantAbsent:  []string{"could not be resolved", "Set `site`", "not found", "UUID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, detail := siteResolveError("default", tt.err)
			if summary != tt.wantSummary {
				t.Errorf("summary = %q, want %q", summary, tt.wantSummary)
			}
			for _, want := range tt.wantInside {
				if !strings.Contains(detail, want) {
					t.Errorf("detail %q is missing %q", detail, want)
				}
			}
			for _, bad := range tt.wantAbsent {
				if strings.Contains(detail, bad) {
					t.Errorf("detail %q must not contain %q", detail, bad)
				}
			}
		})
	}
}

func TestNewTrafficMatchingListResource(t *testing.T) {
	if NewTrafficMatchingListResource() == nil {
		t.Fatal("NewTrafficMatchingListResource returned nil")
	}
}

func TestNewTrafficMatchingListListResource(t *testing.T) {
	if NewTrafficMatchingListListResource() == nil {
		t.Fatal("NewTrafficMatchingListListResource returned nil")
	}
}

func Test_trafficMatchingListResource_Metadata(t *testing.T) {
	r := &trafficMatchingListResource{}
	resp := &fwresource.MetadataResponse{}
	r.Metadata(
		context.Background(),
		fwresource.MetadataRequest{ProviderTypeName: "unifi"},
		resp,
	)
	if resp.TypeName != "unifi_traffic_matching_list" {
		t.Errorf("TypeName = %q, want unifi_traffic_matching_list", resp.TypeName)
	}
}

func Test_trafficMatchingListResource_Schema(t *testing.T) {
	r := &trafficMatchingListResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
}

func Test_trafficMatchingListResource_modelToAPI(t *testing.T) {
	ctx := context.Background()
	r := &trafficMatchingListResource{}

	t.Run("addresses", func(t *testing.T) {
		items, d := types.ListValueFrom(
			ctx,
			types.StringType,
			[]string{"192.0.2.4", "198.51.100.53"},
		)
		if d.HasError() {
			t.Fatalf("items: %v", d)
		}
		model := &trafficMatchingListResourceModel{
			Name:  types.StringValue("DNS Resolvers"),
			Type:  types.StringValue(trafficMatchingListTypeAddresses),
			Items: items,
		}

		api, diags := r.modelToAPI(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToAPI: %v", diags)
		}
		if api.Type != trafficMatchingListTypeAddresses || api.Name != "DNS Resolvers" {
			t.Errorf("api = %q/%q", api.Type, api.Name)
		}
		if len(api.Items) != 2 {
			t.Fatalf("items = %d, want 2", len(api.Items))
		}
		if api.Items[0].Type != trafficMatchingListItemIPAddress ||
			api.Items[0].Value != "192.0.2.4" {
			t.Errorf("item[0] = %+v, want {IP_ADDRESS 192.0.2.4}", api.Items[0])
		}
	})

	t.Run("ports", func(t *testing.T) {
		items, _ := types.ListValueFrom(ctx, types.StringType, []string{"80", "443"})
		model := &trafficMatchingListResourceModel{
			Name:  types.StringValue("HTTP(S)"),
			Type:  types.StringValue(trafficMatchingListTypePorts),
			Items: items,
		}

		api, diags := r.modelToAPI(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToAPI: %v", diags)
		}
		if api.Items[0].Type != trafficMatchingListItemPort || api.Items[0].Value != 80 {
			t.Errorf("item[0] = %+v, want {PORT_NUMBER 80}", api.Items[0])
		}
		if api.Items[1].Value != 443 {
			t.Errorf("item[1].Value = %v, want 443", api.Items[1].Value)
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		items, _ := types.ListValueFrom(ctx, types.StringType, []string{"not-a-port"})
		model := &trafficMatchingListResourceModel{
			Name:  types.StringValue("bad"),
			Type:  types.StringValue(trafficMatchingListTypePorts),
			Items: items,
		}

		_, diags := r.modelToAPI(ctx, model)
		if !diags.HasError() {
			t.Error("expected error for non-numeric port item")
		}
	})
}

func Test_trafficMatchingListResource_apiToModel(t *testing.T) {
	ctx := context.Background()
	r := &trafficMatchingListResource{}

	api := &unifi.TrafficMatchingList{
		ID:   "177b5b16",
		Type: trafficMatchingListTypePorts,
		Name: "HTTP(S)",
		Items: []unifi.TrafficMatchingListItem{
			// Numbers decode through encoding/json as float64.
			{Type: trafficMatchingListItemPort, Value: float64(80)},
			{Type: trafficMatchingListItemPort, Value: float64(443)},
		},
	}

	var model trafficMatchingListResourceModel
	if diags := r.apiToModel(ctx, api, &model, "site-uuid"); diags.HasError() {
		t.Fatalf("apiToModel: %v", diags)
	}

	if model.ID.ValueString() != "177b5b16" {
		t.Errorf("ID = %q", model.ID.ValueString())
	}
	if model.Site.ValueString() != "site-uuid" {
		t.Errorf("Site = %q, want site-uuid", model.Site.ValueString())
	}
	var items []string
	if d := model.Items.ElementsAs(ctx, &items, false); d.HasError() {
		t.Fatalf("items: %v", d)
	}
	if len(items) != 2 || items[0] != "80" || items[1] != "443" {
		t.Errorf("items = %v, want [80 443]", items)
	}
}

func Test_trafficMatchingListItemValueToString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string address", "192.0.2.4", "192.0.2.4"},
		{"float64 port", float64(443), "443"},
		{"int port", 80, "80"},
		{"int64 port", int64(8080), "8080"},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trafficMatchingListItemValueToString(tt.in); got != tt.want {
				t.Errorf(
					"trafficMatchingListItemValueToString(%v) = %q, want %q",
					tt.in,
					got,
					tt.want,
				)
			}
		})
	}
}
