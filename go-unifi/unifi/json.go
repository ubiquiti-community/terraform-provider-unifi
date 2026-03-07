package unifi

func emptyBoolToTrue(b *bool) bool {
	if b == nil {
		return true
	}
	return *b
}
