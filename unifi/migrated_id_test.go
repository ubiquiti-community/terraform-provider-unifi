package unifi

import "testing"

func Test_adoptMigrated(t *testing.T) {
	const (
		oldID = "0123456789abcdef01234567"
		newID = "864031f8-6747-498b-9509-7fb704a144a4"
	)
	type obj struct{ id, key string }
	for name, tc := range map[string]struct {
		storedID   string
		candidates []obj
		want       string
	}{
		"adopts the re-keyed object": {
			oldID, []obj{{"aaaaaaaa-0000-0000-0000-000000000000", "other"}, {newID, "key"}}, newID,
		},
		"stored ID is a UUID": {
			"bbbbbbbb-0000-0000-0000-000000000000", []obj{{newID, "key"}}, "",
		},
		"stored ID still exists": {
			oldID, []obj{{oldID, "other"}, {newID, "key"}}, "",
		},
		"match is an ObjectID (deleted on an older controller)": {
			oldID, []obj{{"89abcdef0123456789abcdef", "key"}}, "",
		},
		"two matches": {
			oldID, []obj{{newID, "key"}, {"cccccccc-0000-0000-0000-000000000000", "key"}}, "",
		},
		"no match": {
			oldID, []obj{{newID, "other"}}, "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := adoptMigrated(tc.storedID,
				func() ([]obj, error) { return tc.candidates, nil },
				func(o obj) string { return o.id },
				func(o obj) bool { return o.key == "key" })
			if err != nil {
				t.Fatal(err)
			}
			gotID := ""
			if got != nil {
				gotID = got.id
			}
			if gotID != tc.want {
				t.Errorf("adoptMigrated = %q, want %q", gotID, tc.want)
			}
		})
	}
}
