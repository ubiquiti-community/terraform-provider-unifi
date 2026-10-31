package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// fakeController serves one object of a rest collection the way the gate run
// measured the controller: a PUT merges its top-level keys into the stored
// document and replaces nested values whole, and a PUT that changes nothing
// returns empty data. Every PUT body is recorded. A POST stores its body.
// Every other rest collection is empty.
type fakeController struct {
	mu   sync.Mutex
	id   string
	doc  map[string]any
	puts []map[string]any
	// reject, when set, returns an api.err code that fails the PUT unwritten.
	reject func(body map[string]any) string
	// onPost, when set, edits a POST body before it is stored.
	onPost func(body map[string]any)
}

func newUpdateFakeController(t *testing.T, kind, id, document string) (*fakeController, *Client) {
	t.Helper()
	f := &fakeController{id: id, doc: decodeJSONObject(t, document)}

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
	mux.HandleFunc("/api/s/default/rest/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[]}`))
	})
	mux.HandleFunc("/api/s/default/rest/"+kind+"/"+id, f.serve(t))
	mux.HandleFunc("/api/s/default/rest/"+kind, f.serve(t))

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
	return f, &Client{ApiClient: apiClient, Site: "default"}
}

func (f *fakeController) serve(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")

		var body map[string]any
		if r.Method != http.MethodGet {
			dec := json.NewDecoder(r.Body)
			dec.UseNumber()
			if err := dec.Decode(&body); err != nil {
				t.Errorf("decoding %s body: %v", r.Method, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		if r.Method == http.MethodPost {
			body["_id"] = f.id
			if f.onPost != nil {
				f.onPost(body)
			}
			f.doc = body
		}
		data := []any{f.doc}
		if r.Method == http.MethodPut {
			f.puts = append(f.puts, body)
			if f.reject != nil {
				if code := f.reject(body); code != "" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprintf(w, `{"meta":{"rc":"error","msg":%q},"data":[]}`, code)
					return
				}
			}
			changed := false
			for k, v := range body {
				if !reflect.DeepEqual(f.doc[k], v) {
					f.doc[k] = v
					changed = true
				}
			}
			if !changed {
				data = []any{}
			}
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{"rc": "ok"},
			"data": data,
		}); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}
}

func (f *fakeController) putsSince(n int) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]map[string]any(nil), f.puts[n:]...)
}

func (f *fakeController) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.puts)
}

func (f *fakeController) stored(key string) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.doc[key]
}

func decodeJSONObject(t *testing.T, s string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("decoding %s: %v", s, err)
	}
	return m
}

type harnessResource interface {
	fwresource.ResourceWithImportState
	fwresource.ResourceWithIdentity
}

// updateHarness drives a resource the way Terraform does: import and refresh,
// then plan through the real provider server, Update, refresh and plan again.
type updateHarness struct {
	t        *testing.T
	ctx      context.Context
	r        harnessResource
	typeName string
	id       string
	fake     *fakeController
	schema   schema.Schema
	state    tfsdk.State
	identity *tfsdk.ResourceIdentity
}

// newUpdateHarness starts a fake controller holding document under
// rest/<kind>/<id>, with null state.
func newUpdateHarness(
	t *testing.T,
	typeName, kind, id, document string,
	newResource func(*Client) harnessResource,
) *updateHarness {
	t.Helper()
	ctx := context.Background()
	fake, client := newUpdateFakeController(t, kind, id, document)
	h := &updateHarness{
		t:        t,
		ctx:      ctx,
		r:        newResource(client),
		typeName: typeName,
		id:       id,
		fake:     fake,
	}

	var schemaResp fwresource.SchemaResponse
	h.r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	h.schema = schemaResp.Schema
	var identityResp fwresource.IdentitySchemaResponse
	h.r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)

	h.state = tfsdk.State{
		Schema: h.schema,
		Raw:    tftypes.NewValue(h.schema.Type().TerraformType(ctx), nil),
	}
	h.identity = &tfsdk.ResourceIdentity{
		Schema: identityResp.IdentitySchema,
		Raw:    tftypes.NewValue(identityResp.IdentitySchema.Type().TerraformType(ctx), nil),
	}
	return h
}

// importAndRead imports the object by its ID and refreshes it.
func (h *updateHarness) importAndRead() {
	h.t.Helper()
	imported := &fwresource.ImportStateResponse{State: h.state, Identity: h.identity}
	h.r.ImportState(h.ctx, fwresource.ImportStateRequest{ID: h.id}, imported)
	if imported.Diagnostics.HasError() {
		h.t.Fatalf("ImportState: %v", imported.Diagnostics)
	}
	h.state, h.identity = imported.State, imported.Identity
	h.state = h.read()
}

// config builds a configuration value. Unset attributes and blocks are null.
func (h *updateHarness) config(set map[string]attr.Value) tftypes.Value {
	h.t.Helper()
	objType, ok := h.schema.Type().TerraformType(h.ctx).(tftypes.Object)
	if !ok {
		h.t.Fatal("schema type is not an object")
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(typ, nil)
	}
	for name, v := range set {
		tv, err := v.ToTerraformValue(h.ctx)
		if err != nil {
			h.t.Fatalf("config %s: %v", name, err)
		}
		vals[name] = tv
	}
	return tftypes.NewValue(objType, vals)
}

// plan returns the planned state for config, with the proposed new state built
// as Terraform core builds it: a computed attribute left out of the
// configuration keeps its prior value. The framework hands an empty list block
// to the resource as null, so the planned state does the same.
func (h *updateHarness) plan(config tftypes.Value) tftypes.Value {
	h.t.Helper()
	proposed := proposedNewState(h.t, h.schema.Attributes, h.state.Raw, config)
	planned := planResourceChange(h.t, h.typeName, h.schema, h.state.Raw, proposed, config)
	var vals map[string]tftypes.Value
	if err := planned.As(&vals); err != nil {
		h.t.Fatalf("planned state as object: %v", err)
	}
	for name := range h.schema.Blocks {
		var elems []tftypes.Value
		if v := vals[name]; v.IsKnown() && !v.IsNull() && v.As(&elems) == nil && len(elems) == 0 {
			vals[name] = tftypes.NewValue(v.Type(), nil)
		}
	}
	return tftypes.NewValue(planned.Type(), vals)
}

func proposedNewState(
	t *testing.T,
	attrs map[string]schema.Attribute,
	prior, config tftypes.Value,
) tftypes.Value {
	t.Helper()
	var priorVals, configVals map[string]tftypes.Value
	if err := config.As(&configVals); err != nil {
		t.Fatalf("config as object: %v", err)
	}
	if err := prior.As(&priorVals); err != nil {
		t.Fatalf("prior as object: %v", err)
	}
	out := maps.Clone(configVals)
	for name, a := range attrs {
		c, p := configVals[name], priorVals[name]
		if prior.IsNull() {
			p = tftypes.NewValue(c.Type(), nil)
		}
		single, isSingle := a.(schema.SingleNestedAttribute)
		switch {
		case a.IsWriteOnly():
			out[name] = tftypes.NewValue(c.Type(), nil)
		case c.IsNull() && a.IsComputed():
			out[name] = p
		case isSingle && !c.IsNull() && !p.IsNull():
			out[name] = proposedNewState(t, single.Attributes, p, c)
		}
	}
	return tftypes.NewValue(config.Type(), out)
}

func (h *updateHarness) update(planned, config tftypes.Value) diag.Diagnostics {
	h.t.Helper()
	resp := &fwresource.UpdateResponse{
		State:    tfsdk.State{Schema: h.schema, Raw: planned.Copy()},
		Identity: &tfsdk.ResourceIdentity{Schema: h.identity.Schema, Raw: h.identity.Raw.Copy()},
	}
	h.r.Update(h.ctx, fwresource.UpdateRequest{
		Config:   tfsdk.Config{Schema: h.schema, Raw: config},
		Plan:     tfsdk.Plan{Schema: h.schema, Raw: planned},
		State:    h.state,
		Identity: h.identity,
	}, resp)
	if !resp.Diagnostics.HasError() {
		h.state = resp.State
	}
	return resp.Diagnostics
}

func (h *updateHarness) read() tfsdk.State {
	h.t.Helper()
	resp := &fwresource.ReadResponse{
		State:    tfsdk.State{Schema: h.schema, Raw: h.state.Raw.Copy()},
		Identity: &tfsdk.ResourceIdentity{Schema: h.identity.Schema, Raw: h.identity.Raw.Copy()},
	}
	h.r.Read(h.ctx, fwresource.ReadRequest{State: h.state, Identity: h.identity}, resp)
	if resp.Diagnostics.HasError() {
		h.t.Fatalf("Read: %v", resp.Diagnostics)
	}
	return resp.State
}

// apply plans config, runs Update and returns the PUT bodies it sent. It
// checks that the new state holds every known planned value, that a refresh
// returns the same state, and that the next plan is empty.
func (h *updateHarness) apply(config tftypes.Value) []map[string]any {
	h.t.Helper()
	planned := h.plan(config)
	sent := h.fake.putCount()
	if diags := h.update(planned, config); diags.HasError() {
		h.t.Fatalf("Update: %v", diags)
	}

	resolved, err := util.ResolveUnknowns(planned, h.state.Raw)
	if err != nil {
		h.t.Fatalf("resolving the plan from the new state: %v", err)
	}
	h.assertEqual("state after Update differs from the plan", resolved, h.state.Raw)
	h.assertEqual("refresh after Update changed the state", h.state.Raw, h.read().Raw)
	h.assertEqual("second plan is not empty", h.state.Raw, h.plan(config))
	return h.fake.putsSince(sent)
}

func (h *updateHarness) assertEqual(msg string, want, got tftypes.Value) {
	h.t.Helper()
	diffs, err := want.Diff(got)
	if err != nil {
		h.t.Fatalf("diff: %v", err)
	}
	if len(diffs) == 0 {
		return
	}
	lines := make([]string, 0, len(diffs))
	for _, d := range diffs {
		lines = append(lines, fmt.Sprintf("  %s: %v -> %v", d.Path, d.Value1, d.Value2))
	}
	sort.Strings(lines)
	h.t.Errorf("%s:\n%s", msg, strings.Join(lines, "\n"))
}

func assertBodies(t *testing.T, got []map[string]any, want ...string) {
	t.Helper()
	gotJSON := make([]string, len(got))
	for i, b := range got {
		out, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("encoding body: %v", err)
		}
		gotJSON[i] = string(out)
	}
	wantJSON := make([]string, len(want))
	for i, w := range want {
		out, err := json.Marshal(decodeJSONObject(t, w))
		if err != nil {
			t.Fatalf("encoding want: %v", err)
		}
		wantJSON[i] = string(out)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Errorf("PUT bodies = %v, want %v", gotJSON, wantJSON)
	}
}

func diagDetails(diags diag.Diagnostics) string {
	var b strings.Builder
	for _, d := range diags {
		b.WriteString(d.Summary() + ": " + d.Detail() + "\n")
	}
	return b.String()
}
