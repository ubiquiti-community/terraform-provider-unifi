package unifi

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// A public key is shared with the peer by design. Marking it sensitive hid it
// from plan diffs and made `-generate-config-out` write `public_key = null`,
// which fails validation because the attribute is required. The secrets in the
// same block stay sensitive.
func Test_vpnClientResource_peerPublicKeyNotSensitive(t *testing.T) {
	var resp fwresource.SchemaResponse
	(&vpnClientResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	wg, ok := resp.Schema.Attributes["wireguard"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("wireguard is not a single nested attribute")
	}
	peer, ok := wg.Attributes["peer"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("wireguard.peer is not a single nested attribute")
	}
	publicKey, ok := peer.Attributes["public_key"].(schema.StringAttribute)
	if !ok {
		t.Fatal("wireguard.peer.public_key is not a string attribute")
	}
	if publicKey.Sensitive {
		t.Error("wireguard.peer.public_key must not be sensitive")
	}
	if !publicKey.Required {
		t.Error("wireguard.peer.public_key must stay required")
	}

	for _, name := range []string{"private_key", "private_key_wo"} {
		secret, ok := wg.Attributes[name].(schema.StringAttribute)
		if !ok || !secret.Sensitive {
			t.Errorf("wireguard.%s must stay sensitive", name)
		}
	}
}
