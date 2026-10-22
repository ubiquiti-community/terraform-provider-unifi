package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// crudResource is what the QoS rule and content filter resources have in
// common, so one lifecycle test drives both.
type crudResource interface {
	fwresource.ResourceWithImportState
	fwresource.ResourceWithIdentity
	fwresource.ResourceWithConfigure
}

type crudCase struct {
	name     string
	new      func() crudResource
	coll     string
	create   map[string]any // planned attributes for create
	update   map[string]any // planned attributes for update
	errLabel string
}

func crudCases() []crudCase {
	return []crudCase{
		{
			name: "qos_rule",
			new:  func() crudResource { return &qosRuleResource{} },
			coll: "qos-rules",
			create: map[string]any{
				"name":                "Limit the TV",
				"enabled":             true,
				"objective":           "LIMIT",
				"download_limit_kbps": 40000,
				"upload_limit_kbps":   10000,
				"download_burst":      "OFF",
				"upload_burst":        "OFF",
				"source": map[string]any{
					"matching_target": "CLIENT",
					"client_macs":     []any{"9c:6b:00:39:f7:a6"},
				},
				"destination": map[string]any{
					"matching_target":      "IP",
					"matching_target_type": "SPECIFIC",
					"ips":                  []any{"10.0.0.0/8"},
					"port_matching_type":   "SPECIFIC",
					"port":                 "80,443",
				},
				"schedule": map[string]any{
					"mode":             "EVERY_WEEK",
					"repeat_on_days":   []any{"mon", "fri"},
					"time_range_start": "18:00",
					"time_range_end":   "23:00",
				},
			},
			update: map[string]any{
				"name":           "Prioritize Online Gaming",
				"enabled":        false,
				"objective":      "PRIORITIZE",
				"download_burst": "OFF",
				"upload_burst":   "OFF",
				"destination": map[string]any{
					"matching_target":  "APP_CATEGORY",
					"app_category_ids": []any{8},
				},
			},
			errLabel: "QoS Rule",
		},
		{
			name: "content_filter",
			new:  func() crudResource { return &contentFilterResource{} },
			coll: "content-filtering",
			create: map[string]any{
				"name":        "adblocking",
				"enabled":     true,
				"network_ids": []any{"66ec38ab34dbcb19dce4a67f"},
				"categories":  []any{"ADVERTISEMENT"},
				"block_list":  []any{"ads.example.com"},
				"safe_search": []any{"GOOGLE"},
			},
			update: map[string]any{
				"name":        "kids",
				"enabled":     true,
				"client_macs": []any{"9c:6b:00:39:f7:a6"},
				"categories":  []any{"ADVERTISEMENT", "GAMBLING"},
				"allow_list":  []any{"school.example.com"},
				"schedule": map[string]any{
					"mode":         "EVERY_DAY",
					"time_all_day": true,
				},
			},
			errLabel: "Content Filter",
		},
	}
}

func configured(t *testing.T, tc crudCase) (crudResource, *v2FakeController) {
	t.Helper()
	client, fake := newV2FakeController(t)
	r := tc.new()
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: client}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure: %v", resp.Diagnostics)
	}
	return r, fake
}

func doCreate(t *testing.T, r crudResource, attrs map[string]any) *fwresource.CreateResponse {
	t.Helper()
	s := resourceSchema(t, r)
	resp := &fwresource.CreateResponse{State: tfState(t, s, nil), Identity: tfIdentity(t, r)}
	r.Create(context.Background(), fwresource.CreateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs),
	}, resp)
	return resp
}

func doRead(
	t *testing.T,
	r crudResource,
	state tfsdk.State,
	identity *tfsdk.ResourceIdentity,
) *fwresource.ReadResponse {
	t.Helper()
	resp := &fwresource.ReadResponse{State: state, Identity: tfIdentity(t, r)}
	r.Read(context.Background(), fwresource.ReadRequest{State: state, Identity: identity}, resp)
	return resp
}

func stateString(t *testing.T, s tfsdk.State, attr string) string {
	t.Helper()
	var v types.String
	if d := s.GetAttribute(context.Background(), path.Root(attr), &v); d.HasError() {
		t.Fatalf("reading %s: %v", attr, d)
	}
	return v.ValueString()
}

// Create, read, update, read, delete: the bodies sent are the configuration,
// and what the controller answers ends up in state.
func Test_qosContentFilter_lifecycle(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r, fake := configured(t, tc)
			s := resourceSchema(t, r)

			created := doCreate(t, r, tc.create)
			if created.Diagnostics.HasError() {
				t.Fatalf("Create: %v", created.Diagnostics)
			}
			id := stateString(t, created.State, "id")
			if id == "" || stateString(t, created.State, "site") != "default" {
				t.Fatalf("created state: id=%q site=%q", id, stateString(t, created.State, "site"))
			}
			if got := fake.bodies["POST"]["name"]; got != tc.create["name"] {
				t.Errorf("create body name = %v", got)
			}

			read := doRead(t, r, created.State, created.Identity)
			if read.Diagnostics.HasError() {
				t.Fatalf("Read: %v", read.Diagnostics)
			}
			if !read.State.Raw.Equal(created.State.Raw) {
				t.Errorf(
					"read after create changed state:\n%v\nwant\n%v",
					read.State.Raw,
					created.State.Raw,
				)
			}

			upd := &fwresource.UpdateResponse{State: created.State, Identity: created.Identity}
			r.Update(context.Background(), fwresource.UpdateRequest{
				Plan:   tfPlan(t, s, tc.update),
				Config: tfConfig(t, s, tc.update),
				State:  created.State,
			}, upd)
			if upd.Diagnostics.HasError() {
				t.Fatalf("Update: %v", upd.Diagnostics)
			}
			if got := fake.bodies["PUT"]["_id"]; got != id {
				t.Errorf("update body _id = %v, want %s", got, id)
			}
			if got := stateString(t, upd.State, "name"); got != tc.update["name"] {
				t.Errorf("updated name = %q", got)
			}
			read = doRead(t, r, upd.State, upd.Identity)
			if !read.State.Raw.Equal(upd.State.Raw) {
				t.Errorf(
					"read after update changed state:\n%v\nwant\n%v",
					read.State.Raw,
					upd.State.Raw,
				)
			}

			del := &fwresource.DeleteResponse{}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: upd.State}, del)
			if del.Diagnostics.HasError() {
				t.Fatalf("Delete: %v", del.Diagnostics)
			}
			if n := fake.count(tc.coll); n != 0 {
				t.Errorf("%d items left after delete", n)
			}
		})
	}
}

// Deleted outside Terraform: Read drops it from state, Delete succeeds.
func Test_qosContentFilter_goneFromController(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r, fake := configured(t, tc)
			created := doCreate(t, r, tc.create)
			if created.Diagnostics.HasError() {
				t.Fatalf("Create: %v", created.Diagnostics)
			}
			fake.remove(tc.coll, stateString(t, created.State, "id"))

			read := doRead(t, r, created.State, created.Identity)
			if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
				t.Errorf(
					"Read of a deleted item: diags=%v state=%v",
					read.Diagnostics,
					read.State.Raw,
				)
			}
			del := &fwresource.DeleteResponse{}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: created.State}, del)
			if del.Diagnostics.HasError() {
				t.Errorf("Delete of a deleted item: %v", del.Diagnostics)
			}
		})
	}
}

// Every controller error surfaces as a diagnostic naming the operation.
func Test_qosContentFilter_controllerErrors(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r, fake := configured(t, tc)
			s := resourceSchema(t, r)

			fake.fail["POST"] = true
			if resp := doCreate(
				t,
				r,
				tc.create,
			); !hasError(
				resp.Diagnostics,
				"Error Creating "+tc.errLabel,
			) {
				t.Errorf("Create: %v", resp.Diagnostics)
			}
			fake.fail["POST"] = false
			created := doCreate(t, r, tc.create)

			fake.fail["GET"] = true
			if resp := doRead(
				t,
				r,
				created.State,
				created.Identity,
			); !hasError(
				resp.Diagnostics,
				"Error Reading "+tc.errLabel,
			) {
				t.Errorf("Read: %v", resp.Diagnostics)
			}
			fake.fail["GET"] = false

			fake.fail["PUT"] = true
			upd := &fwresource.UpdateResponse{State: created.State}
			r.Update(context.Background(), fwresource.UpdateRequest{
				Plan: tfPlan(
					t,
					s,
					tc.update,
				),
				Config: tfConfig(t, s, tc.update),
				State:  created.State,
			}, upd)
			if !hasError(upd.Diagnostics, "Error Updating "+tc.errLabel) {
				t.Errorf("Update: %v", upd.Diagnostics)
			}

			fake.fail["DELETE"] = true
			del := &fwresource.DeleteResponse{}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: created.State}, del)
			if !hasError(del.Diagnostics, "Error Deleting "+tc.errLabel) {
				t.Errorf("Delete: %v", del.Diagnostics)
			}
		})
	}
}

// hasError reports whether diags holds an error with the given summary.
func hasError(diags diag.Diagnostics, want string) bool {
	for _, d := range diags.Errors() {
		if d.Summary() == want {
			return true
		}
	}
	return false
}

// Import by id, by site:id and by identity, then a read by identity alone.
func Test_qosContentFilter_import(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := configured(t, tc)
			s := resourceSchema(t, r)
			created := doCreate(t, r, tc.create)
			if created.Diagnostics.HasError() {
				t.Fatalf("Create: %v", created.Diagnostics)
			}
			id := stateString(t, created.State, "id")

			for _, in := range []struct {
				id, wantID, wantSite string
			}{
				{id, id, "default"},
				{"other:" + id, id, "other"},
			} {
				resp := &fwresource.ImportStateResponse{
					State:    tfState(t, s, map[string]any{}),
					Identity: tfIdentity(t, r),
				}
				r.ImportState(context.Background(), fwresource.ImportStateRequest{ID: in.id}, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("import %q: %v", in.id, resp.Diagnostics)
				}
				if got := stateString(t, resp.State, "id"); got != in.wantID {
					t.Errorf("import %q: id = %q", in.id, got)
				}
				if got := stateString(t, resp.State, "site"); got != in.wantSite {
					t.Errorf("import %q: site = %q", in.id, got)
				}
			}

			identity := tfIdentity(t, r)
			identity.Raw = tfValue(
				t,
				identity.Raw.Type(),
				map[string]any{"id": id, "site": "default"},
			)
			resp := &fwresource.ImportStateResponse{
				State:    tfState(t, s, map[string]any{}),
				Identity: tfIdentity(t, r),
			}
			r.ImportState(
				context.Background(),
				fwresource.ImportStateRequest{Identity: identity},
				resp,
			)
			if resp.Diagnostics.HasError() || stateString(t, resp.State, "id") != id {
				t.Errorf(
					"import by identity: %v, id = %q",
					resp.Diagnostics,
					stateString(t, resp.State, "id"),
				)
			}

			// The state an import by identity starts from carries nothing but the identity.
			read := doRead(t, r, tfState(t, s, map[string]any{}), identity)
			if read.Diagnostics.HasError() ||
				stateString(t, read.State, "name") != tc.create["name"] {
				t.Errorf(
					"read by identity: %v, name = %q",
					read.Diagnostics,
					stateString(t, read.State, "name"),
				)
			}

			empty := tfIdentity(t, r)
			empty.Raw = tfValue(t, empty.Raw.Type(), map[string]any{})
			resp = &fwresource.ImportStateResponse{
				State:    tfState(t, s, map[string]any{}),
				Identity: tfIdentity(t, r),
			}
			r.ImportState(
				context.Background(),
				fwresource.ImportStateRequest{Identity: empty},
				resp,
			)
			if !hasError(resp.Diagnostics, "Invalid Import ID") {
				t.Errorf("import without id: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_qosContentFilter_configure(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.new()
			resp := &fwresource.ConfigureResponse{}
			r.Configure(context.Background(), fwresource.ConfigureRequest{}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("Configure without provider data: %v", resp.Diagnostics)
			}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: "nope"},
				resp,
			)
			if !hasError(resp.Diagnostics, "Unexpected Resource Configure Type") {
				t.Errorf("Configure with the wrong type: %v", resp.Diagnostics)
			}

			mresp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				mresp,
			)
			if want := "unifi_" + tc.name; mresp.TypeName != want {
				t.Errorf("type name = %q, want %q", mresp.TypeName, want)
			}
			if !reflect.DeepEqual(resourceSchema(t, r).Attributes["id"].IsComputed(), true) {
				t.Error("id is not computed")
			}
		})
	}
}

// A state written before identities existed: no identity, no site. Read
// falls back to the provider's site and records the identity.
func Test_qosContentFilter_readLegacyState(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := configured(t, tc)
			s := resourceSchema(t, r)
			created := doCreate(t, r, tc.create)
			id := stateString(t, created.State, "id")

			read := doRead(t, r, tfState(t, s, map[string]any{"id": id}), nil)
			if read.Diagnostics.HasError() {
				t.Fatalf("Read: %v", read.Diagnostics)
			}
			if got := stateString(t, read.State, "site"); got != "default" {
				t.Errorf("site = %q, want the provider's", got)
			}
			if read.Identity.Raw.IsNull() {
				t.Error("identity was not recorded")
			}
		})
	}
}

// Invalid timeouts and a state that does not fit the schema stop every
// operation before it reaches the controller.
func Test_qosContentFilter_invalidInput(t *testing.T) {
	for _, tc := range crudCases() {
		t.Run(tc.name, func(t *testing.T) {
			r, fake := configured(t, tc)
			s := resourceSchema(t, r)
			bad := map[string]any{"id": "x", "timeouts": map[string]any{
				"create": "soon", "read": "soon", "update": "soon", "delete": "soon",
			}}
			for k, v := range tc.create {
				bad[k] = v
			}
			mismatched := tfsdk.State{
				Schema: s,
				Raw:    tfValue(t, tftypes.Object{}, map[string]any{}),
			}

			cr := &fwresource.CreateResponse{State: tfState(t, s, nil), Identity: tfIdentity(t, r)}
			r.Create(context.Background(), fwresource.CreateRequest{Plan: tfPlan(t, s, bad)}, cr)
			cr2 := &fwresource.CreateResponse{State: tfState(t, s, nil)}
			r.Create(context.Background(), fwresource.CreateRequest{
				Plan: tfsdk.Plan{Schema: s, Raw: mismatched.Raw},
			}, cr2)
			rr := doRead(t, r, tfState(t, s, bad), nil)
			rr2 := doRead(t, r, mismatched, nil)
			ur := &fwresource.UpdateResponse{State: tfState(t, s, bad)}
			r.Update(context.Background(), fwresource.UpdateRequest{
				Plan: tfPlan(t, s, bad), State: tfState(t, s, bad),
			}, ur)
			ur2 := &fwresource.UpdateResponse{State: mismatched}
			r.Update(context.Background(), fwresource.UpdateRequest{
				Plan: tfsdk.Plan{Schema: s, Raw: mismatched.Raw}, State: mismatched,
			}, ur2)
			dr := &fwresource.DeleteResponse{}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: tfState(t, s, bad)}, dr)
			dr2 := &fwresource.DeleteResponse{}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: mismatched}, dr2)

			for name, d := range map[string]diag.Diagnostics{
				"create": cr.Diagnostics, "create/mismatched": cr2.Diagnostics,
				"read": rr.Diagnostics, "read/mismatched": rr2.Diagnostics,
				"update": ur.Diagnostics, "update/mismatched": ur2.Diagnostics,
				"delete": dr.Diagnostics, "delete/mismatched": dr2.Diagnostics,
			} {
				if !d.HasError() {
					t.Errorf("%s: no error", name)
				}
			}
			if len(fake.bodies) != 0 || fake.count(tc.coll) != 0 {
				t.Errorf("the controller was written to: %v", fake.bodies)
			}
		})
	}
}
