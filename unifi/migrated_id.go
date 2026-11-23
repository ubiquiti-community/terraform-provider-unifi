package unifi

import "regexp"

var legacyObjectIDPattern = regexp.MustCompile(`^[0-9a-f]{24}$`)

// isLegacyObjectID reports whether id is a 24-hex ObjectID, the form every
// ID had before UniFi Network 11 re-keyed objects to UUIDs.
func isLegacyObjectID(id string) bool {
	return legacyObjectIDPattern.MatchString(id)
}

// adoptMigrated finds the object a stored ID named before Network 11 re-keyed
// it. Network 11 serves the v2 collections only under the new UUIDs, so a
// Read that looks the old ID up finds nothing and would drop an object that
// still exists, and the next apply creates a duplicate (#716, #742).
//
// It lists the candidates only when storedID is an ObjectID, and returns the
// one with a UUID that is the same object by its natural key; nil when there
// is none, or more than one, or storedID is still in the list. On controllers
// before Network 11 every ID is an ObjectID, so nothing is adopted there.
func adoptMigrated[T any](
	storedID string,
	list func() ([]T, error),
	id func(T) string,
	sameObject func(T) bool,
) (*T, error) {
	if !isLegacyObjectID(storedID) {
		return nil, nil
	}
	candidates, err := list()
	if err != nil {
		return nil, err
	}
	var adopted *T
	for i, c := range candidates {
		if id(c) == storedID {
			return nil, nil
		}
		if isLegacyObjectID(id(c)) || !sameObject(c) {
			continue
		}
		if adopted != nil {
			return nil, nil
		}
		adopted = &candidates[i]
	}
	return adopted, nil
}
