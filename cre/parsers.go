package cre

import (
	"encoding/json"
)

// ParseJSON parses a JSON byte slice into a struct of type *T.
//
// ParseJSON is a thin typed wrapper around [encoding/json.Unmarshal]; it allocates
// a new T, unmarshals into it, and returns a pointer to the result. All semantics
// of json.Unmarshal apply, including how it handles JSON "null":
//   - For non-pointer types (e.g. int), JSON "null" unmarshals to the zero value
//     and a valid pointer to that zero value is returned.
//   - For pointer types (e.g. *Foo), JSON "null" unmarshals to nil, so the
//     returned **Foo points to a nil *Foo. Callers that dereference the result
//     must account for this case.
//
// See [encoding/json.Unmarshal] for the full set of decoding rules.
func ParseJSON[T any](bytes []byte) (*T, error) {
	var result T
	err := json.Unmarshal(bytes, &result)
	return &result, err
}
