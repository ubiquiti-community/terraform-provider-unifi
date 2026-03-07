package types

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Number extends types.Number to handle empty strings and string values.
// For example a field may contain a number, an empty string, or the string "auto".
type Number json.Number

func (n *Number) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	if s == `""` {
		*n = ""
		return nil
	}
	var err error
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s, err = strconv.Unquote(s)
		if err != nil {
			return err
		}
		*n = Number(s)
		return nil
	}
	// For numeric values, delegate to underlying types.Number
	*n = Number(string(b))
	return nil
}

// String returns the number as a string.
func (n Number) String() string {
	return string(n)
}

// Float64 returns the number as a float64.
func (n Number) Float64() (float64, error) {
	return json.Number(n).Float64()
}

// Int64 returns the number as an int64.
func (n Number) Int64() (int64, error) {
	return json.Number(n).Int64()
}

func (n Number) Int64Pointer() *int64 {
	val, err := n.Int64()
	if err != nil {
		return nil
	}
	return &val
}

func ToInt64Pointer(aux Number) *int64 {
	return aux.Int64Pointer()
}
