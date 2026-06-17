package unifi

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// timeoutsNullValue returns a null timeouts.Value with the standard attribute types
// (create, read, update, delete). Use this when populating a resource model in a
// List context, where no timeout configuration is provided by the caller.
func timeoutsNullValue() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"delete": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
		}),
	}
}
