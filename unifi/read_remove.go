package unifi

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// removeResourceOnRead drops a resource that no longer exists on the
// controller from the state while keeping a resource identity in the read
// response.
//
// terraform-plugin-framework rejects any read response whose identity is
// fully null ("Missing Resource Identity After Read"), including the response
// that removes the resource. resp.Identity is pre-populated with the stored
// identity, so it is only null when the CLI never persisted one: OpenTofu
// (no resource identity support yet), Terraform < 1.12, or a state last
// written by a provider release that predates identity support. Without this
// helper an out-of-band deletion broke every plan on those CLIs instead of
// being planned as a re-creation.
//
// identity is the identity model rebuilt from the state lookup keys, the same
// value the success path writes when no identity was stored.
func removeResourceOnRead(ctx context.Context, resp *resource.ReadResponse, identity any) {
	if resp.Identity != nil && resp.Identity.Raw.IsFullyNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
	}

	resp.State.RemoveResource(ctx)
}

// isNotFound reports whether err says the object is not on the controller,
// which answers a GET of a deleted ID with rc "ok" and no data.
func isNotFound(err error) bool {
	var notFound *unifi.NotFoundError
	return errors.As(err, &notFound)
}
