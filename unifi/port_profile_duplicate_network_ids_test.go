package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// The controller can return the same network ID twice in a port profile's
// excluded_networkconf_ids: creating a network appends it to every
// Custom-trunk profile, and an update in the same apply that already lists the
// new ID adds it again. Building a set from that list failed every later plan
// with "Duplicate Set Element". The read now drops duplicates.
//
// Unit rather than acceptance test: the conversion is what failed, and it
// needs no controller.
func TestPortProfileDuplicateNetworkIDsCollapseOnRead(t *testing.T) {
	r := &portProfileResource{}
	api := &unifi.PortProfile{
		Name:                      "trunk",
		ExcludedNetworkIDs:        []string{"net-b", "net-a", "net-c", "net-b"},
		MulticastRouterNetworkIDs: []string{"net-x", "net-x"},
	}

	model := &portProfileResourceModel{Name: types.StringValue("trunk")}
	if d := r.portProfileToModel(context.Background(), api, model, "default"); d.HasError() {
		t.Fatalf("conversion: %v", d)
	}

	if got := setStrings(
		t,
		model.ExcludedNetworkConfIDs,
	); !reflect.DeepEqual(
		got,
		[]string{"net-a", "net-b", "net-c"},
	) {
		t.Errorf("excluded_networkconf_ids = %v, want [net-a net-b net-c]", got)
	}
	if got := setStrings(
		t,
		model.MulticastRouterNetworkIDs,
	); !reflect.DeepEqual(
		got,
		[]string{"net-x"},
	) {
		t.Errorf("multicast_router_networkconf_ids = %v, want [net-x]", got)
	}
}
