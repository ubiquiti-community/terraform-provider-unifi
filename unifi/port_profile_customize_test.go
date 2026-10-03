package unifi

import (
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

// networkName backs the #496 plan diagnostic, which has to name the offending
// native network. A network with no name must still produce something the
// operator can act on rather than an empty quote.
func TestNetworkNameForDiagnostics(t *testing.T) {
	name := "net-a"
	empty := ""

	for _, tc := range []struct {
		label string
		in    *unifi.Network
		want  string
	}{
		{"named network uses its name", &unifi.Network{ID: "id-1", Name: &name}, "net-a"},
		{"nil name falls back to the ID", &unifi.Network{ID: "id-1"}, "id-1"},
		{"empty name falls back to the ID", &unifi.Network{ID: "id-1", Name: &empty}, "id-1"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			if got := networkName(tc.in); got != tc.want {
				t.Errorf("networkName() = %q, want %q", got, tc.want)
			}
		})
	}
}
