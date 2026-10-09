package models

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestClientInfoObjectType_Equal(t *testing.T) {
	type args struct {
		o attr.Type
	}
	tests := []struct {
		name string
		tr   ClientInfoObjectType
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.Equal(tt.args.o); got != tt.want {
				t.Errorf("ClientInfoObjectType.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectType_String(t *testing.T) {
	tests := []struct {
		name string
		tr   ClientInfoObjectType
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.String(); got != tt.want {
				t.Errorf("ClientInfoObjectType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectType_ValueFromObject(t *testing.T) {
	type args struct {
		ctx context.Context
		in  basetypes.ObjectValue
	}
	tests := []struct {
		name  string
		tr    ClientInfoObjectType
		args  args
		want  basetypes.ObjectValuable
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.tr.ValueFromObject(tt.args.ctx, tt.args.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoObjectType.ValueFromObject() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"ClientInfoObjectType.ValueFromObject() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func TestClientInfoObjectValue_Type(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    ClientInfoObjectValue
		args args
		want attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Type(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoObjectValue.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectValue_ValueType(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    ClientInfoObjectValue
		args args
		want attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.ValueType(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoObjectValue.ValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectValue_Equal(t *testing.T) {
	type args struct {
		o attr.Value
	}
	tests := []struct {
		name string
		v    ClientInfoObjectValue
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Equal(tt.args.o); got != tt.want {
				t.Errorf("ClientInfoObjectValue.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientInfoObjectType(t *testing.T) {
	tests := []struct {
		name string
		want ClientInfoObjectType
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientInfoObjectType(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientInfoObjectType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientInfoObjectTypeFromData(t *testing.T) {
	type args struct {
		data unifi.ClientInfo
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewClientInfoObjectTypeFromData(tt.args.data)
		})
	}
}

func TestAttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Attributes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Attributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoDataSourceSchema(t *testing.T) {
	tests := []struct {
		name string
		want map[string]schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoDataSourceSchema(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoDataSourceSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListAttribute(t *testing.T) {
	tests := []struct {
		name string
		want schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoListAttribute(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientListValue(t *testing.T) {
	type args struct {
		ctx            context.Context
		clientInfoList unifi.ClientList
		target         *types.List
	}
	tests := []struct {
		name string
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientListValue(
				tt.args.ctx,
				tt.args.clientInfoList,
				tt.args.target,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("ClientListValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAttributes_matchAttributeTypes guards the invariant that the schema map
// and the attr.Type map describe the same object, nested groups included.
func TestAttributes_matchAttributeTypes(t *testing.T) {
	attrs := Attributes()
	attrTypes := AttributeTypes()
	if len(attrs) != len(attrTypes) {
		t.Errorf("Attributes() has %d entries, AttributeTypes() has %d", len(attrs), len(attrTypes))
	}
	for name, a := range attrs {
		want, ok := attrTypes[name]
		if !ok {
			t.Errorf("schema attribute %q has no entry in AttributeTypes()", name)
			continue
		}
		if got := a.GetType(); !got.Equal(want) {
			t.Errorf("%s: schema type %s != AttributeTypes() %s", name, got, want)
		}
	}
	for _, old := range clientInfoFlattenedNames {
		if _, ok := attrs[old]; ok {
			t.Errorf("flat attribute %q still present in Attributes()", old)
		}
		if _, ok := attrTypes[old]; ok {
			t.Errorf("flat attribute %q still present in AttributeTypes()", old)
		}
	}
	for name, sub := range map[string]map[string]attr.Type{
		"network":                 ClientNetworkAttrTypes(),
		"last_uplink":             ClientLastUplinkAttrTypes(),
		"last_connection_network": ClientNetworkAttrTypes(),
		"tx":                      ClientTrafficAttrTypes(),
		"rx":                      ClientTrafficAttrTypes(),
	} {
		nested, ok := attrs[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Errorf("%s: is %T, want schema.SingleNestedAttribute", name, attrs[name])
			continue
		}
		if !nested.Computed || nested.Optional || nested.Required {
			t.Errorf("%s: want Computed-only object", name)
		}
		if len(nested.Attributes) != len(sub) {
			t.Errorf("%s: %d leaves, want %d", name, len(nested.Attributes), len(sub))
		}
		for leaf, leafType := range sub {
			la, ok := nested.Attributes[leaf]
			if !ok {
				t.Errorf("%s.%s: missing leaf", name, leaf)
				continue
			}
			if !la.IsComputed() || la.IsOptional() || la.IsRequired() {
				t.Errorf("%s.%s: want Computed-only leaf", name, leaf)
			}
			if !la.GetType().Equal(leafType) {
				t.Errorf("%s.%s: type %s, want %s", name, leaf, la.GetType(), leafType)
			}
		}
	}
}

// clientInfoFlattenedNames are the pre-nesting attribute names that must no
// longer appear anywhere in the client info object.
var clientInfoFlattenedNames = []string{
	"network_id", "network_name",
	"last_uplink_mac", "last_uplink_name", "last_uplink_remote_port",
	"last_connection_network_id", "last_connection_network_name",
	"tx_rate", "tx_bytes", "rx_rate", "rx_bytes",
}

// TestClientInfoAttrValues_nestedGroups checks API -> attribute map conversion
// places the prefixed API fields into their nested objects.
func TestClientInfoAttrValues_nestedGroups(t *testing.T) {
	ctx := context.Background()
	remotePort := int64(24)
	txRate, txBytes := int64(1000), int64(2000)
	rxRate, rxBytes := int64(3000), int64(4000)
	info := &unifi.ClientInfo{
		Id:                        "c1",
		Mac:                       "aa:bb:cc:dd:ee:ff",
		NetworkId:                 "net-1",
		NetworkName:               "LAN",
		LastUplinkMac:             "11:22:33:44:55:66",
		LastUplinkName:            "sw-1",
		LastUplinkRemotePort:      &remotePort,
		LastConnectionNetworkId:   "net-2",
		LastConnectionNetworkName: "IoT",
		TxRate:                    &txRate,
		TxBytes:                   &txBytes,
		RxRate:                    &rxRate,
		RxBytes:                   &rxBytes,
	}

	got, diags := ClientInfoAttrValues(ctx, info)
	if diags.HasError() {
		t.Fatalf("ClientInfoAttrValues: %v", diags)
	}
	if _, d := types.ObjectValue(AttributeTypes(), got); d.HasError() {
		t.Fatalf("value map does not satisfy AttributeTypes(): %v", d)
	}
	for _, old := range clientInfoFlattenedNames {
		if _, ok := got[old]; ok {
			t.Errorf("flat attribute %q still emitted", old)
		}
	}

	network := objectAttr(t, got, "network")
	wantLeaf(t, network, "id", types.StringValue("net-1"))
	wantLeaf(t, network, "name", types.StringValue("LAN"))

	lastUplink := objectAttr(t, got, "last_uplink")
	wantLeaf(t, lastUplink, "mac", types.StringValue("11:22:33:44:55:66"))
	wantLeaf(t, lastUplink, "name", types.StringValue("sw-1"))
	wantLeaf(t, lastUplink, "remote_port", types.Int64Value(24))

	lastConn := objectAttr(t, got, "last_connection_network")
	wantLeaf(t, lastConn, "id", types.StringValue("net-2"))
	wantLeaf(t, lastConn, "name", types.StringValue("IoT"))

	tx := objectAttr(t, got, "tx")
	wantLeaf(t, tx, "rate", types.Int64Value(1000))
	wantLeaf(t, tx, "bytes", types.Int64Value(2000))

	rx := objectAttr(t, got, "rx")
	wantLeaf(t, rx, "rate", types.Int64Value(3000))
	wantLeaf(t, rx, "bytes", types.Int64Value(4000))
}

// TestClientInfoAttrValues_emptyLeavesAreNull checks that, exactly as the flat
// attributes did, empty API values become null leaves while the objects
// themselves stay known.
func TestClientInfoAttrValues_emptyLeavesAreNull(t *testing.T) {
	got, diags := ClientInfoAttrValues(context.Background(), &unifi.ClientInfo{})
	if diags.HasError() {
		t.Fatalf("ClientInfoAttrValues: %v", diags)
	}
	for _, name := range []string{"network", "last_uplink", "last_connection_network", "tx", "rx"} {
		obj := objectAttr(t, got, name)
		if obj.IsNull() || obj.IsUnknown() {
			t.Errorf("%s: object should be known, got %v", name, obj)
			continue
		}
		for leaf, v := range obj.Attributes() {
			if !v.IsNull() {
				t.Errorf("%s.%s = %v, want null", name, leaf, v)
			}
		}
	}
}

// TestClientInfoValue_nestedGroups checks the custom object value carries the
// nested groups with the custom type's attribute types.
func TestClientInfoValue_nestedGroups(t *testing.T) {
	ctx := context.Background()
	var target ClientInfoObjectValue
	diags := ClientInfoValue(
		ctx,
		&unifi.ClientInfo{NetworkId: "net-1", NetworkName: "LAN"},
		&target,
	)
	if diags.HasError() {
		t.Fatalf("ClientInfoValue: %v", diags)
	}
	if !target.Type(ctx).Equal(NewClientInfoObjectType()) {
		t.Errorf("type = %s, want %s", target.Type(ctx), NewClientInfoObjectType())
	}
	network := objectAttr(t, target.Attributes(), "network")
	wantLeaf(t, network, "id", types.StringValue("net-1"))
	wantLeaf(t, network, "name", types.StringValue("LAN"))
}

// objectAttr is a checked lookup of a types.Object in an attribute map.
func objectAttr(t *testing.T, m map[string]attr.Value, name string) types.Object {
	t.Helper()
	v, ok := m[name]
	if !ok {
		t.Fatalf("attribute %q missing", name)
	}
	obj, ok := v.(types.Object)
	if !ok {
		t.Fatalf("attribute %q is %T, want types.Object", name, v)
	}
	return obj
}

// wantLeaf asserts obj[leaf] equals want.
func wantLeaf(t *testing.T, obj types.Object, leaf string, want attr.Value) {
	t.Helper()
	got, ok := obj.Attributes()[leaf]
	if !ok {
		t.Errorf("leaf %q missing from %v", leaf, obj)
		return
	}
	if !got.Equal(want) {
		t.Errorf("leaf %q = %v, want %v", leaf, got, want)
	}
}

func TestClientInfoValue(t *testing.T) {
	type args struct {
		ctx        context.Context
		clientInfo *unifi.ClientInfo
		target     *ClientInfoObjectValue
	}
	tests := []struct {
		name string
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoValue(
				tt.args.ctx,
				tt.args.clientInfo,
				tt.args.target,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("ClientInfoValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
