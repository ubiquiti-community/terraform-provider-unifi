package unifi

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ParseImportID parses import IDs supporting both "id" and "site:id" formats
func ParseImportID(id string, minParts int, maxParts int) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	parts := strings.SplitN(id, ":", maxParts)

	if len(parts) < minParts || len(parts) > maxParts {
		diags.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in the format 'id' or 'site:id', got: %s", id),
		)
		return nil, diags
	}

	return parts, diags
}
