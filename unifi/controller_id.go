package unifi

import "regexp"

// controllerIDPattern matches the IDs the controller gives its objects: a
// 24-hex MongoDB ObjectID before UniFi Network 11, and a lowercase UUID on
// objects Network 11 created or migrated.
var controllerIDPattern = regexp.MustCompile(
	`^([0-9a-f]{24}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`,
)

// isControllerID reports whether an import ID names an object by its ID
// rather than by its name.
func isControllerID(s string) bool {
	return controllerIDPattern.MatchString(s)
}
