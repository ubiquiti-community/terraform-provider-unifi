package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Same failure as the port profile read: the controller can return a network
// ID twice in a port override's excluded_networkconf_ids (a new network is
// appended to every Custom-tagged port), and building a set from that list
// failed with "Duplicate Set Element". Both the read and the reconcile path
// now go through stringSetFromAPI, which drops the duplicate.

func TestPortOverridesToFramework_DuplicateNetworkIDsCollapse(t *testing.T) {
	r := &deviceResource{}
	set, diags := r.portOverridesToFramework(context.Background(), []unifi.DevicePortOverrides{
		{
			PortIDX:                   ptrInt64(5),
			ExcludedNetworkIDs:        []string{"net-b", "net-a", "net-b"},
			MulticastRouterNetworkIDs: []string{"net-x", "net-x"},
		},
	})
	if diags.HasError() {
		t.Fatalf("portOverridesToFramework errored: %v", diags)
	}
	var models []portOverrideModel
	if d := set.ElementsAs(context.Background(), &models, false); d.HasError() {
		t.Fatalf("reading set: %v", d)
	}
	if got := setStrings(
		t,
		models[0].ExcludedNetworkIDs,
	); !reflect.DeepEqual(
		got,
		[]string{"net-a", "net-b"},
	) {
		t.Errorf("excluded_networkconf_ids = %v, want [net-a net-b]", got)
	}
	if got := setStrings(
		t,
		models[0].MulticastRouterNetworkIDs,
	); !reflect.DeepEqual(
		got,
		[]string{"net-x"},
	) {
		t.Errorf("multicast_router_networkconf_ids = %v, want [net-x]", got)
	}
}

func TestReconcilePortOverrides_ExcludedNetworkIDsDuplicate(t *testing.T) {
	r := &deviceResource{}
	prior := stringSet(t, "net-a", "net-b")
	priorSet := portOverrideSetWith(t, map[string]attr.Value{
		"index":                    types.Int64Value(5),
		"excluded_networkconf_ids": prior,
	})
	got, diags := r.reconcilePortOverrides(
		context.Background(),
		priorSet,
		[]unifi.DevicePortOverrides{
			{PortIDX: ptrInt64(5), ExcludedNetworkIDs: []string{"net-b", "net-a", "net-b"}},
		},
	)
	if diags.HasError() {
		t.Fatalf("reconcilePortOverrides errored: %v", diags)
	}
	var models []portOverrideModel
	if d := got.ElementsAs(context.Background(), &models, false); d.HasError() {
		t.Fatalf("reading set: %v", d)
	}
	if !models[0].ExcludedNetworkIDs.Equal(prior) {
		t.Errorf("read-back %v != configured %v", models[0].ExcludedNetworkIDs, prior)
	}
}
