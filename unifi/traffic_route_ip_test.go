package unifi

import (
	"reflect"
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestTrafficRouteIPRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		addresses []unifi.TrafficRouteIPAddresses
		ranges    []unifi.TrafficRouteIPRanges
	}{
		{
			name: "IPv4 address",
			addresses: []unifi.TrafficRouteIPAddresses{
				{Address: "192.0.2.1", Version: unifi.TrafficRouteIPVersionV4},
			},
		},
		{
			name: "IPv6 address",
			addresses: []unifi.TrafficRouteIPAddresses{
				{Address: "2001:db8::1", Version: unifi.TrafficRouteIPVersionV6},
			},
		},
		{
			name: "IPv4 prefix",
			addresses: []unifi.TrafficRouteIPAddresses{
				{Address: "192.0.2.0/24", Version: unifi.TrafficRouteIPVersionV4},
			},
		},
		{
			name: "IPv6 prefix",
			addresses: []unifi.TrafficRouteIPAddresses{
				{Address: "2001:db8::/32", Version: unifi.TrafficRouteIPVersionV6},
			},
		},
		{
			name: "IPv4 range",
			ranges: []unifi.TrafficRouteIPRanges{
				{Start: "192.0.2.1", Stop: "192.0.2.100", Version: unifi.TrafficRouteIPVersionV4},
			},
		},
		{
			name: "IPv6 range",
			ranges: []unifi.TrafficRouteIPRanges{
				{
					Start:   "2001:db8::1",
					Stop:    "2001:db8::100",
					Version: unifi.TrafficRouteIPVersionV6,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			r := &trafficRouteResource{}
			want := &unifi.TrafficRoute{
				NetworkID:      "net-1",
				MatchingTarget: "IP",
				IPAddresses:    append([]unifi.TrafficRouteIPAddresses{}, tt.addresses...),
				IPRanges:       append([]unifi.TrafficRouteIPRanges{}, tt.ranges...),
			}
			var model trafficRouteResourceModel
			if diags := r.apiToModel(ctx, want, &model, "default"); diags.HasError() {
				t.Fatalf("apiToModel: %v", diags)
			}
			got, diags := r.modelToAPI(ctx, &model, "default")
			if diags.HasError() {
				t.Fatalf("modelToAPI: %v", diags)
			}
			if got.MatchingTarget != want.MatchingTarget {
				t.Errorf("MatchingTarget = %q, want %q", got.MatchingTarget, want.MatchingTarget)
			}
			if !reflect.DeepEqual(got.IPAddresses, want.IPAddresses) {
				t.Errorf("IPAddresses = %#v, want %#v", got.IPAddresses, want.IPAddresses)
			}
			if !reflect.DeepEqual(got.IPRanges, want.IPRanges) {
				t.Errorf("IPRanges = %#v, want %#v", got.IPRanges, want.IPRanges)
			}
		})
	}
}
