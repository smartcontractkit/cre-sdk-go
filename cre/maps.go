package cre

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// MapSorted returns a sequence that iterates over the entries in the map in the natural sorted order of the keys.
// For keys which do not implement cmp.Ordered, use MapSortedFunc.
func MapSorted[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}

// MapSortedFunc returns a sequence that iterates over the entries in the map based on sorting keys with the cmp func.
// For naturally cmp.Ordered keys, use MapSorted.
func MapSortedFunc[K comparable, V any](m map[K]V, cmp func(a, b K) int) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range slices.SortedFunc(maps.Keys(m), cmp) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
