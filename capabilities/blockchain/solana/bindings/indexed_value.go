package bindings

import (
	encbinary "encoding/binary"
	"fmt"
	"math"
	"reflect"
)

// EncodeIndexedValue converts a filter or decoded event field into the byte
// representation used in SubkeyConfig ValueComparator values.
//
// Supported types: []byte, string, integers, floats, and fixed-size byte arrays
// (e.g. PublicKey, [32]byte).
func EncodeIndexedValue(value any) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("subkey value cannot be nil")
	}

	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("subkey value cannot be nil")
		}
		return EncodeIndexedValue(rv.Elem().Interface())
	}

	if rv.Kind() == reflect.Array && rv.Type().Elem().Kind() == reflect.Uint8 {
		result := make([]byte, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = byte(rv.Index(i).Uint())
		}
		return result, nil
	}

	if rv.CanUint() {
		buf := make([]byte, 8)
		encbinary.BigEndian.PutUint64(buf, rv.Uint())
		return buf, nil
	}
	if rv.CanInt() {
		buf := make([]byte, 8)
		encbinary.BigEndian.PutUint64(buf, uint64(rv.Int())) //nolint:gosec // two's complement encoding
		return buf, nil
	}
	if rv.CanFloat() {
		buf := make([]byte, 8)
		f := rv.Float()
		if f > 0 {
			encbinary.BigEndian.PutUint64(buf, math.Float64bits(f)+math.MaxInt64+1)
		} else {
			encbinary.BigEndian.PutUint64(buf, math.MaxInt64+1-math.Float64bits(f))
		}
		return buf, nil
	}

	return nil, fmt.Errorf("unsupported subkey value type %T", value)
}
