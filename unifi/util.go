package unifi

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ParseImportID parses import IDs supporting both "id" and "site:id" formats.
func ParseImportID(id string, minParts int, maxParts int) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	resp := map[string]string{}

	parts := strings.SplitN(id, ":", maxParts)

	if len(parts) < minParts || len(parts) > maxParts {
		diags.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in the format 'id' or 'site:id', got: %s", id),
		)
		return nil, diags
	}

	if len(parts) == 2 {
		slices.Reverse(parts)
		resp["site"] = parts[1]
	}

	resp["id"] = parts[0]

	return resp, diags
}
