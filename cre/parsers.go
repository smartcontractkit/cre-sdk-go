package cre

import (
	"encoding/json"
)

// ParseJSON parses a JSON byte slice into a struct of type *T.
func ParseJSON[T any](bytes []byte) (*T, error) {
	var result T
	err := json.Unmarshal(bytes, &result)
	return &result, err
}
