package unifi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func Test_settingResource_radiusSecretWriteOnlySchema(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var resp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("schema implementation diagnostics: %v", diags)
	}

	secretWO, ok := resp.Schema.Attributes["radius_secret_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatal("radius_secret_wo is not a string attribute")
	}
	if !secretWO.WriteOnly || !secretWO.Sensitive || !secretWO.Optional || secretWO.Computed {
		t.Fatalf(
			"radius_secret_wo flags = write-only:%t sensitive:%t optional:%t computed:%t",
			secretWO.WriteOnly,
			secretWO.Sensitive,
			secretWO.Optional,
			secretWO.Computed,
		)
	}
	version, ok := resp.Schema.Attributes["radius_secret_wo_version"].(schema.Int64Attribute)
	if !ok || !version.Optional || version.WriteOnly {
		t.Fatal("radius_secret_wo_version is not an optional, stored int64 attribute")
	}

	// The existing radius block and its secret are unchanged.
	radius, ok := resp.Schema.Attributes["radius"].(schema.SingleNestedAttribute)
	if !ok || !radius.Optional || !radius.Computed {
		t.Fatal("radius is no longer an Optional+Computed single nested attribute")
	}
	secret, ok := radius.Attributes["secret"].(schema.StringAttribute)
	if !ok || !secret.Optional || !secret.Computed || !secret.Sensitive || secret.WriteOnly {
		t.Fatal("radius.secret is no longer an Optional+Computed sensitive attribute")
	}
}

// radiusSecretWOController is a fake controller serving the radius setting
// document. It records the body of every radius PUT and echoes it back.
type radiusSecretWOController struct {
	mu   sync.Mutex
	puts []map[string]any
}

func (c *radiusSecretWOController) lastPut(t *testing.T) map[string]any {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.puts) == 0 {
		t.Fatal("no radius setting was written")
	}
	return c.puts[len(c.puts)-1]
}

func newRadiusSecretWOFakeControllerClient(
	t *testing.T,
) (*Client, *radiusSecretWOController) {
	t.Helper()

	ctrl := &radiusSecretWOController{}
	current := map[string]any{
		"_id":                "radius-id",
		"key":                "radius",
		"accounting_enabled": false,
		"acct_port":          1813,
		"auth_port":          1812,
		"x_secret":           "controller-secret",
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
	writeDoc := func(w http.ResponseWriter, doc map[string]any) {
		payload, err := json.Marshal(map[string]any{
			"meta": map[string]any{"rc": "ok"},
			"data": []any{doc},
		})
		if err != nil {
			t.Errorf("marshaling radius setting: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}
	mux.HandleFunc(
		"/api/s/default/get/setting/radius",
		func(w http.ResponseWriter, _ *http.Request) {
			ctrl.mu.Lock()
			defer ctrl.mu.Unlock()
			writeDoc(w, current)
		},
	)
	mux.HandleFunc(
		"/api/s/default/set/setting/radius",
		func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading radius PUT body: %v", err)
			}
			var doc map[string]any
			if err := json.Unmarshal(body, &doc); err != nil {
				t.Errorf("decoding radius PUT body: %v", err)
			}
			ctrl.mu.Lock()
			defer ctrl.mu.Unlock()
			ctrl.puts = append(ctrl.puts, doc)
			current = doc
			writeDoc(w, doc)
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
	return &Client{ApiClient: apiClient, Site: "default"}, ctrl
}

// radiusSecretWOValue builds a full resource object with every attribute null
// except site, an empty radius block and the two write-only attributes.
func radiusSecretWOValue(
	t *testing.T,
	ctx context.Context,
	s schema.Schema,
	secretWO any,
	version any,
) tftypes.Value {
	t.Helper()
	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(typ, nil)
	}
	radiusType, ok := objType.AttributeTypes["radius"].(tftypes.Object)
	if !ok {
		t.Fatal("radius type is not tftypes.Object")
	}
	radiusVals := make(map[string]tftypes.Value, len(radiusType.AttributeTypes))
	for name, typ := range radiusType.AttributeTypes {
		radiusVals[name] = tftypes.NewValue(typ, nil)
	}
	vals["site"] = tftypes.NewValue(tftypes.String, "default")
	vals["radius"] = tftypes.NewValue(radiusType, radiusVals)
	vals["radius_secret_wo"] = tftypes.NewValue(tftypes.String, secretWO)
	vals["radius_secret_wo_version"] = tftypes.NewValue(tftypes.Number, version)
	return tftypes.NewValue(objType, vals)
}

// assertRadiusSecretNotInState checks that neither the write-only secret nor
// the controller's secret entered state, and that the version was stored.
func assertRadiusSecretNotInState(
	t *testing.T,
	ctx context.Context,
	state tfsdk.State,
	wantVersion int64,
) {
	t.Helper()
	var secretWO types.String
	if d := state.GetAttribute(ctx, path.Root("radius_secret_wo"), &secretWO); d.HasError() {
		t.Fatalf("reading radius_secret_wo: %v", d)
	}
	if !secretWO.IsNull() {
		t.Fatalf("radius_secret_wo entered state: %q", secretWO.ValueString())
	}
	var version types.Int64
	if d := state.GetAttribute(ctx, path.Root("radius_secret_wo_version"), &version); d.HasError() {
		t.Fatalf("reading radius_secret_wo_version: %v", d)
	}
	if version.ValueInt64() != wantVersion {
		t.Fatalf("radius_secret_wo_version = %v, want %d", version, wantVersion)
	}
	var radiusObj types.Object
	if d := state.GetAttribute(ctx, path.Root("radius"), &radiusObj); d.HasError() {
		t.Fatalf("reading radius: %v", d)
	}
	var radius settingRadiusModel
	if d := radiusObj.As(ctx, &radius, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("decoding radius: %v", d)
	}
	if !radius.Secret.IsNull() {
		t.Fatalf("radius.secret entered state: %q", radius.Secret.ValueString())
	}
}

// TestSettingRadiusSecretWO checks that radius_secret_wo is sent as the RADIUS
// shared secret on create and update, and that state never holds it.
func TestSettingRadiusSecretWO(t *testing.T) {
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	(&settingResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	var idResp fwresource.IdentitySchemaResponse
	(&settingResource{}).IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &idResp)
	nullState := func() tfsdk.State {
		return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	}
	nullIdentity := func() *tfsdk.ResourceIdentity {
		return &tfsdk.ResourceIdentity{
			Schema: idResp.IdentitySchema,
			Raw:    tftypes.NewValue(idResp.IdentitySchema.Type().TerraformType(ctx), nil),
		}
	}

	t.Run("create sends the write-only secret", func(t *testing.T) {
		client, ctrl := newRadiusSecretWOFakeControllerClient(t)
		r := &settingResource{client: client}
		// Terraform always sends a write-only value as null in the plan.
		plan := radiusSecretWOValue(t, ctx, s, nil, 1)
		config := radiusSecretWOValue(t, ctx, s, "wo-secret", 1)
		resp := &fwresource.CreateResponse{State: nullState(), Identity: nullIdentity()}
		r.Create(ctx, fwresource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: s, Raw: plan},
			Config: tfsdk.Config{Schema: s, Raw: config},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Create diagnostics: %v", resp.Diagnostics)
		}
		if got := ctrl.lastPut(t)["x_secret"]; got != "wo-secret" {
			t.Fatalf("x_secret in request body = %v, want wo-secret", got)
		}
		assertRadiusSecretNotInState(t, ctx, resp.State, 1)
	})

	t.Run("update sends the write-only secret", func(t *testing.T) {
		client, ctrl := newRadiusSecretWOFakeControllerClient(t)
		r := &settingResource{client: client}
		prior := radiusSecretWOValue(t, ctx, s, nil, 1)
		plan := radiusSecretWOValue(t, ctx, s, nil, 2)
		config := radiusSecretWOValue(t, ctx, s, "rotated-secret", 2)
		resp := &fwresource.UpdateResponse{State: nullState(), Identity: nullIdentity()}
		r.Update(ctx, fwresource.UpdateRequest{
			State:  tfsdk.State{Schema: s, Raw: prior},
			Plan:   tfsdk.Plan{Schema: s, Raw: plan},
			Config: tfsdk.Config{Schema: s, Raw: config},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update diagnostics: %v", resp.Diagnostics)
		}
		if got := ctrl.lastPut(t)["x_secret"]; got != "rotated-secret" {
			t.Fatalf("x_secret in request body = %v, want rotated-secret", got)
		}
		assertRadiusSecretNotInState(t, ctx, resp.State, 2)
	})

	t.Run("unset write-only secret keeps the controller secret", func(t *testing.T) {
		client, ctrl := newRadiusSecretWOFakeControllerClient(t)
		r := &settingResource{client: client}
		plan := radiusSecretWOValue(t, ctx, s, nil, nil)
		resp := &fwresource.CreateResponse{State: nullState(), Identity: nullIdentity()}
		r.Create(ctx, fwresource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: s, Raw: plan},
			Config: tfsdk.Config{Schema: s, Raw: plan},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Create diagnostics: %v", resp.Diagnostics)
		}
		if got := ctrl.lastPut(t)["x_secret"]; got != "controller-secret" {
			t.Fatalf("x_secret in request body = %v, want controller-secret", got)
		}
	})
}
