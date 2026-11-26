package unifi

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// portProfileJSONFields maps each JSON key of go-unifi's PortProfile to its
// struct field.
func portProfileJSONFields() map[string]reflect.StructField {
	out := map[string]reflect.StructField{}
	typ := reflect.TypeOf(unifi.PortProfile{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		out[name] = f
	}
	return out
}

// testPortProfile is a realistic controller profile the per-field tests start from.
func testPortProfile() unifi.PortProfile {
	return unifi.PortProfile{
		ID:                "pp-1",
		SiteID:            "site-1",
		Name:              "access",
		OpMode:            "switch",
		Forward:           "native",
		TaggedVLANMgmt:    "block_all",
		NATiveNetworkID:   "net-1",
		Autoneg:           true,
		Dot1XCtrl:         "force_authorized",
		LldpmedEnabled:    true,
		StpPortMode:       true,
		SettingPreference: "auto",
		StormctrlType:     "level",
	}
}

// portProfileUnmodelledFields are the PortProfile fields the resource does not
// model, on purpose: identity and controller bookkeeping, and the loop
// detection fields (ld_mode, dropped by Network 11, and ld_status, a status
// the controller reports). On update the controller keeps them: the
// omitempty ones are left out of the body, and ld_status is carried
// (carryUnexposedPortProfileFields); a create leaves them to the controller.
var portProfileUnmodelledFields = []string{
	"_id",
	"site_id",
	"attr_hidden",
	"attr_hidden_id",
	"attr_no_delete",
	"attr_no_edit",
	"ld_mode",
	"ld_status",
}

// TestPortProfile_modelsEveryField fails when go-unifi grows a PortProfile
// field that the resource neither models nor lists in
// portProfileUnmodelledFields: a value read from the controller must come back
// out of the model on create, too.
func TestPortProfile_modelsEveryField(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}
	unmodelled := map[string]bool{}
	for _, k := range portProfileUnmodelledFields {
		unmodelled[k] = true
	}
	var lost []string
	for name, f := range portProfileJSONFields() {
		current := testPortProfile()
		reflect.ValueOf(&current).Elem().FieldByIndex(f.Index).Set(nonZero(t, f.Type))
		var model portProfileResourceModel
		if d := r.portProfileToModel(ctx, &current, &model, "default"); d.HasError() {
			t.Fatalf("%s: portProfileToModel: %v", name, d)
		}
		got, d := r.modelToAPIPortProfile(ctx, &model)
		if d.HasError() {
			t.Fatalf("%s: modelToAPIPortProfile: %v", name, d)
		}
		same := reflect.DeepEqual(
			reflect.ValueOf(current).FieldByIndex(f.Index).Interface(),
			reflect.ValueOf(*got).FieldByIndex(f.Index).Interface(),
		)
		switch {
		case !same && !unmodelled[name]:
			lost = append(lost, name)
		case same && unmodelled[name]:
			t.Errorf(
				"%q round-trips through the model; remove it from portProfileUnmodelledFields",
				name,
			)
		}
	}
	sort.Strings(lost)
	for _, name := range lost {
		t.Errorf(
			"PortProfile.%s is not modelled by unifi_port_profile; expose it or list it in portProfileUnmodelledFields",
			name,
		)
	}
	for _, k := range portProfileUnmodelledFields {
		if _, ok := portProfileJSONFields()[k]; !ok {
			t.Errorf("listed field %q does not exist on PortProfile", k)
		}
	}
}

func TestPortProfile_newSettingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}
	debounce := int64(500)
	current := testPortProfile()
	current.StpBpduGuardEnabled = true
	current.StpEdgeState = "enabled"
	current.StpUplink = false
	current.MulticastRouterMode = "CUSTOM"
	current.MulticastRouterNetworkIDs = []string{"net-1"}
	current.EeeEnabled = true
	current.FlowControlEnabled = true
	current.LinkDebounce = &debounce
	current.LinkDebounceAuto = false
	current.StablePortEnabled = true
	current.PrecisionTimeProtocolEnabled = true
	current.QOSProfile = &unifi.PortProfileQOSProfile{
		QOSProfileMode: "custom",
		QOSPolicies: []unifi.PortProfileQOSPolicies{{
			QOSMatching: &unifi.PortProfileQOSMatching{
				DscpCode: ptrInt64(46), Protocol: "udp", DstPort: ptrInt64(5004),
			},
			QOSMarking: &unifi.PortProfileQOSMarking{Queue: ptrInt64(6), CosCode: ptrInt64(5)},
		}},
	}

	var model portProfileResourceModel
	if d := r.portProfileToModel(ctx, &current, &model, "default"); d.HasError() {
		t.Fatalf("portProfileToModel: %v", d)
	}
	if !model.STPBPDUGuardEnabled.ValueBool() || model.STPEdgeState.ValueString() != "enabled" ||
		model.MulticastRouterMode.ValueString() != "CUSTOM" || !model.EEEEnabled.ValueBool() ||
		!model.FlowControlEnabled.ValueBool() || !model.StablePortEnabled.ValueBool() ||
		!model.PTPEnabled.ValueBool() || model.STPUplink.ValueBool() {
		t.Errorf("flat settings read back wrong: %+v", model)
	}
	ld := model.LinkDebounce.Attributes()
	if d := attrAs[timetypes.GoDuration](t, ld["delay"]); d.ValueString() != "500ms" {
		t.Errorf("link_debounce.delay = %s, want 500ms", d.ValueString())
	}
	if attrAs[types.Bool](t, ld["auto"]).ValueBool() {
		t.Errorf("link_debounce.auto read back true")
	}
	qos := model.QOSProfile.Attributes()
	if attrAs[types.String](t, qos["mode"]).ValueString() != "custom" {
		t.Errorf("qos_profile.mode = %v", qos["mode"])
	}
	if n := len(attrAs[types.List](t, qos["policies"]).Elements()); n != 1 {
		t.Fatalf("qos_profile.policies has %d elements, want 1", n)
	}

	// Create path: the model alone reproduces the controller's values.
	got, d := r.modelToAPIPortProfile(ctx, &model)
	if d.HasError() {
		t.Fatalf("modelToAPIPortProfile: %v", d)
	}
	wantJSON, _ := json.Marshal(current)
	got.ID, got.SiteID = current.ID, current.SiteID
	gotJSON, _ := json.Marshal(got)
	if string(wantJSON) != string(gotJSON) {
		t.Errorf("round trip changed the profile:\n want %s\n  got %s", wantJSON, gotJSON)
	}
}

func TestPortQOSProfile_emptyAndCleared(t *testing.T) {
	ctx := context.Background()
	policyType := types.ObjectType{AttrTypes: portQOSPolicyAttrTypes()}

	// None on the controller: null object.
	obj, d := portQOSProfileObject(ctx, nil, types.ObjectNull(portQOSProfileAttrTypes()))
	if d.HasError() || !obj.IsNull() {
		t.Fatalf("nil profile: %v %v", obj, d)
	}

	// qos_profile = {} clears: it is sent as an empty object, which the
	// controller stores as {} and reads back as an object of nulls.
	cleared := types.ObjectValueMust(portQOSProfileAttrTypes(), map[string]attr.Value{
		"mode":     types.StringNull(),
		"policies": types.ListNull(policyType),
	})
	q, ok, d := portQOSProfileFromObject(ctx, cleared)
	if !ok || d.HasError() {
		t.Fatalf("cleared: ok=%v %v", ok, d)
	}
	body, _ := json.Marshal(portProfileQOSToAPI(q))
	if string(body) != "{}" {
		t.Errorf("qos_profile = {} marshals to %s, want {}", body)
	}
	back, d := portQOSProfileObject(
		ctx,
		portProfileQOSFromAPI(&unifi.PortProfileQOSProfile{}),
		cleared,
	)
	if d.HasError() || !back.Equal(cleared) {
		t.Errorf("cleared read back %v, want %v", back, cleared)
	}

	// mode = "custom", policies = []: the empty list is dropped from the body
	// (omitempty), so the read keeps the configured empty list from prior.
	emptyPolicies := types.ObjectValueMust(portQOSProfileAttrTypes(), map[string]attr.Value{
		"mode":     types.StringValue("custom"),
		"policies": types.ListValueMust(policyType, []attr.Value{}),
	})
	back, d = portQOSProfileObject(
		ctx,
		portProfileQOSFromAPI(
			&unifi.PortProfileQOSProfile{QOSProfileMode: "custom"},
		),
		emptyPolicies,
	)
	if d.HasError() || !back.Equal(emptyPolicies) {
		t.Errorf("empty policies read back %v, want %v", back, emptyPolicies)
	}

	// An unknown object (not configured) sends nothing.
	if _, ok, _ := portQOSProfileFromObject(
		ctx,
		types.ObjectUnknown(portQOSProfileAttrTypes()),
	); ok {
		t.Errorf("unknown qos_profile was converted")
	}
}

// portProfileAttr returns a schema attribute by path.
func portProfileAttr(t *testing.T, p ...string) schema.Attribute {
	t.Helper()
	var resp fwresource.SchemaResponse
	(&portProfileResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	a := resp.Schema.Attributes[p[0]]
	for _, name := range p[1:] {
		switch n := a.(type) {
		case schema.SingleNestedAttribute:
			a = n.Attributes[name]
		case schema.ListNestedAttribute:
			a = n.NestedObject.Attributes[name]
		default:
			t.Fatalf("%v: %T has no attribute %q", p, a, name)
		}
	}
	if a == nil {
		t.Fatalf("no attribute %v", p)
	}
	return a
}

func validString(t *testing.T, a schema.Attribute, v string) bool {
	t.Helper()
	var vs []validator.String
	switch s := a.(type) {
	case schema.StringAttribute:
		vs = s.Validators
	default:
		t.Fatalf("%T is not a string attribute", a)
	}
	return validStringValue(t, vs, types.StringValue(v))
}

func validStringValue(t *testing.T, vs []validator.String, v types.String) bool {
	t.Helper()
	for _, x := range vs {
		resp := &validator.StringResponse{}
		x.ValidateString(context.Background(), validator.StringRequest{
			Path: path.Root("test"), ConfigValue: v,
		}, resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}

func validInt64(t *testing.T, a schema.Attribute, v int64) bool {
	t.Helper()
	s, ok := a.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("%T is not an int64 attribute", a)
	}
	for _, x := range s.Validators {
		resp := &validator.Int64Response{}
		x.ValidateInt64(context.Background(), validator.Int64Request{
			Path: path.Root("test"), ConfigValue: types.Int64Value(v),
		}, resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}

// TestPortProfile_newSettingsValidation pins the validators to the
// controller's field patterns (api/fields/PortConf.json, Network 11.0).
func TestPortProfile_newSettingsValidation(t *testing.T) {
	delay := portProfileAttr(t, "link_debounce", "delay")
	for v, want := range map[string]bool{
		"0s": true, "100ms": true, "500ms": true, "4.9s": true, "5s": true,
		"150ms": false, "5.1s": false, "10s": false,
	} {
		if got := validString(t, delay, v); got != want {
			t.Errorf("link_debounce.delay %q valid = %v, want %v", v, got, want)
		}
	}
	edge := portProfileAttr(t, "stp_edge_state")
	for v, want := range map[string]bool{"enabled": true, "disabled": true, "auto": false, "": false} {
		if got := validString(t, edge, v); got != want {
			t.Errorf("stp_edge_state %q valid = %v, want %v", v, got, want)
		}
	}
	mode := portProfileAttr(t, "multicast_router_mode")
	for v, want := range map[string]bool{"ALL": true, "CUSTOM": true, "NONE": true, "all": false} {
		if got := validString(t, mode, v); got != want {
			t.Errorf("multicast_router_mode %q valid = %v, want %v", v, got, want)
		}
	}
	qmode := portProfileAttr(t, "qos_profile", "mode")
	for v, want := range map[string]bool{"custom": true, "dante_audio": true, "shure_audio": true, "dante": false} {
		if got := validString(t, qmode, v); got != want {
			t.Errorf("qos_profile.mode %q valid = %v, want %v", v, got, want)
		}
	}
	proto := portProfileAttr(t, "qos_profile", "policies", "matching", "protocol")
	for v, want := range map[string]bool{
		"udp": true, "tcp": true, "ipv6-icmp": true, "ax.25": true, "0": true, "17": true, "255": true,
		"256": false, "UDP": false, "axx25": false, "017": false,
	} {
		if got := validString(t, proto, v); got != want {
			t.Errorf("matching.protocol %q valid = %v, want %v", v, got, want)
		}
	}
	markDSCP := portProfileAttr(t, "qos_profile", "policies", "marking", "dscp_code")
	for v, want := range map[int64]bool{0: true, 46: true, 34: true, 44: true, 7: false, 63: false} {
		if got := validInt64(t, markDSCP, v); got != want {
			t.Errorf("marking.dscp_code %d valid = %v, want %v", v, got, want)
		}
	}
	matchDSCP := portProfileAttr(t, "qos_profile", "policies", "matching", "dscp_code")
	for v, want := range map[int64]bool{0: true, 7: true, 63: true, 64: false} {
		if got := validInt64(t, matchDSCP, v); got != want {
			t.Errorf("matching.dscp_code %d valid = %v, want %v", v, got, want)
		}
	}
	queue := portProfileAttr(t, "qos_profile", "policies", "marking", "queue")
	for v, want := range map[int64]bool{0: true, 7: true, 8: false} {
		if got := validInt64(t, queue, v); got != want {
			t.Errorf("marking.queue %d valid = %v, want %v", v, got, want)
		}
	}
}

// TestPortQOSProfileSchema_everyLeafComputed guards the qos_profile schema:
// Terraform keeps the prior value of an unset Optional+Computed nested
// attribute only when every set leaf in it is Computed. A single Optional-only
// leaf makes it propose null for a profile that configuration leaves out, and
// a QoS profile set in the UI then plans a change on every run.
func TestPortQOSProfileSchema_everyLeafComputed(t *testing.T) {
	var walk func(p string, a schema.Attribute)
	walk = func(p string, a schema.Attribute) {
		if !a.IsComputed() {
			t.Errorf("%s is not Computed", p)
		}
		switch n := a.(type) {
		case schema.SingleNestedAttribute:
			for name, c := range n.Attributes {
				walk(p+"."+name, c)
			}
		case schema.ListNestedAttribute:
			for name, c := range n.NestedObject.Attributes {
				walk(p+"[*]."+name, c)
			}
		}
	}
	walk("qos_profile", portProfileAttr(t, "qos_profile"))
}

// TestPortProfileModifyPlan_configuredQOSProfileIsExact: a configured
// qos_profile is planned exactly as written, so a leaf configuration leaves
// out is not kept from the prior profile and `qos_profile = {}` clears it.
func TestPortProfileModifyPlan_configuredQOSProfileIsExact(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{} // no client: ModifyPlan stops after the plan-only rules
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is not an object")
	}
	policyType := types.ObjectType{AttrTypes: portQOSPolicyAttrTypes()}

	qos := func(mode attr.Value, policies attr.Value) types.Object {
		return types.ObjectValueMust(portQOSProfileAttrTypes(), map[string]attr.Value{
			"mode": mode, "policies": policies,
		})
	}
	matching := func(protocol string, dstPort attr.Value) attr.Value {
		return types.ObjectValueMust(portQOSMatchingAttrTypes(), map[string]attr.Value{
			"cos_code": types.Int64Null(), "dscp_code": types.Int64Null(), "dst_port": dstPort,
			"ip_precedence_code": types.Int64Null(), "protocol": types.StringValue(protocol),
			"src_port": types.Int64Null(),
		})
	}
	marking := types.ObjectValueMust(portQOSMarkingAttrTypes(), map[string]attr.Value{
		"cos_code": types.Int64Null(), "dscp_code": types.Int64Null(),
		"ip_precedence_code": types.Int64Null(), "queue": types.Int64Value(3),
	})
	policy := func(m attr.Value) attr.Value {
		return types.ObjectValueMust(portQOSPolicyAttrTypes(), map[string]attr.Value{
			"matching": m, "marking": marking,
		})
	}
	prior := qos(types.StringValue("custom"), types.ListValueMust(policyType,
		[]attr.Value{policy(matching("udp", types.Int64Value(5004)))}))

	withQOS := func(v types.Object) tftypes.Value {
		st := tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    portProfileRaw(objType, map[string]any{"name": "av"}),
		}
		if d := st.SetAttribute(ctx, path.Root("qos_profile"), v); d.HasError() {
			t.Fatalf("SetAttribute: %v", d)
		}
		return st.Raw
	}

	tests := []struct {
		name             string
		config, proposed types.Object
		want             types.Object
	}{
		{
			name: "dst_port removed from the policy",
			config: qos(types.StringValue("custom"), types.ListValueMust(policyType,
				[]attr.Value{policy(matching("udp", types.Int64Null()))})),
			// What Terraform proposes for a Computed leaf that config leaves out: the prior value.
			proposed: prior,
			want: qos(types.StringValue("custom"), types.ListValueMust(policyType,
				[]attr.Value{policy(matching("udp", types.Int64Null()))})),
		},
		{
			name:     "qos_profile = {} clears it",
			config:   qos(types.StringNull(), types.ListNull(policyType)),
			proposed: prior,
			want:     qos(types.StringNull(), types.ListNull(policyType)),
		},
		{
			name:     "not configured: left to the plan modifiers",
			config:   types.ObjectNull(portQOSProfileAttrTypes()),
			proposed: prior,
			want:     prior,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fwresource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: withQOS(tt.config)},
				Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: withQOS(tt.proposed)},
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: withQOS(prior)},
			}
			resp := &fwresource.ModifyPlanResponse{Plan: req.Plan}
			r.ModifyPlan(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
			}
			var got types.Object
			resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("qos_profile"), &got)...)
			if !got.Equal(tt.want) {
				t.Errorf("planned qos_profile\n got %v\nwant %v", got, tt.want)
			}
		})
	}
}
