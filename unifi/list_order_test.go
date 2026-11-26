package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func stringList(vals ...string) types.List {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}

func Test_listInPriorOrder(t *testing.T) {
	for name, tc := range map[string]struct {
		prior types.List
		got   []string
		want  types.List
	}{
		"reordered keeps prior": {
			stringList("wan1", "wan2", "wan3"),
			[]string{"wan3", "wan1", "wan2"},
			stringList("wan1", "wan2", "wan3"),
		},
		"changed takes controller": {
			stringList("wan1", "wan2"),
			[]string{"wan2", "wan3"},
			stringList("wan2", "wan3"),
		},
		"same set, different counts takes controller": {
			stringList("wan1", "wan1", "wan2"),
			[]string{"wan2", "wan1", "wan2"},
			stringList("wan2", "wan1", "wan2"),
		},
		"added element takes controller": {
			stringList("wan1"),
			[]string{"wan2", "wan1"},
			stringList("wan2", "wan1"),
		},
		"null prior takes controller": {
			types.ListNull(types.StringType),
			[]string{"wan2", "wan1"},
			stringList("wan2", "wan1"),
		},
		"unknown prior takes controller": {
			types.ListUnknown(types.StringType),
			[]string{"wan2", "wan1"},
			stringList("wan2", "wan1"),
		},
		"empty": {
			stringList(),
			[]string{},
			stringList(),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, diags := listInPriorOrder(context.Background(), tc.prior, tc.got)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// The controller returns a client's groups in its own order (#719).
func TestClientToModel_keepsPriorGroupOrder(t *testing.T) {
	r := &clientResource{client: &Client{groupCache: map[string]map[string]string{
		"default": {"netman-managed": "g1", "iot-internet-only": "g2", "cameras": "g3"},
	}}}
	client := &unifi.Client{
		MAC:                    "02:00:00:de:ad:03",
		NetworkMembersGroupIDs: []string{"g1", "g2"},
	}

	model := clientResourceModel{Groups: stringList("iot-internet-only", "netman-managed")}
	if diags := r.clientToModel(context.Background(), client, &model, "default"); diags.HasError() {
		t.Fatal(diags)
	}
	if want := stringList("iot-internet-only", "netman-managed"); !model.Groups.Equal(want) {
		t.Errorf("groups = %v, want the prior order %v", model.Groups, want)
	}

	model = clientResourceModel{Groups: stringList("iot-internet-only", "cameras")}
	if diags := r.clientToModel(context.Background(), client, &model, "default"); diags.HasError() {
		t.Fatal(diags)
	}
	if want := stringList("netman-managed", "iot-internet-only"); !model.Groups.Equal(want) {
		t.Errorf("groups = %v, want the controller's %v", model.Groups, want)
	}
}
