package validators

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// wanInterfaceRegex is the controller's own pattern for a WAN interface name
// (`wan[2-9]?` in the field specs of dynamic DNS, port forwards and the VPN
// interfaces).
var wanInterfaceRegex = regexp.MustCompile(`^wan[2-9]?$`)

// WANInterface validates that a string names a WAN interface the way the
// controller does: `wan` for the first one, `wan2` through `wan9` for the
// others. A gateway with three WAN ports has a `wan3`.
func WANInterface() validator.String {
	return stringvalidator.RegexMatches(
		wanInterfaceRegex,
		"must be 'wan' or 'wan2' through 'wan9'",
	)
}
