package unifi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

func TestAccountMarshalJSON(t *testing.T) {
	for n, c := range map[string]struct {
		expectedJSON string
		acc          unifi.Account
	}{
		"empty strings": {
			`{"vlan":"","tunnel_type":"","tunnel_medium_type":""}`,
			unifi.Account{},
		},
		"response": {
			`{"vlan":10,"tunnel_type":1,"tunnel_medium_type":1}`,
			unifi.Account{
				VLAN:             util.Ptr[int64](10),
				TunnelType:       util.Ptr[int64](1),
				TunnelMediumType: util.Ptr[int64](1),
			},
		},
	} {
		t.Run(n, func(t *testing.T) {
			actual, err := json.Marshal(&c.acc)
			if err != nil {
				t.Fatal(err)
			}
			assert.JSONEq(t, c.expectedJSON, string(actual))
		})
	}
}
