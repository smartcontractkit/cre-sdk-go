package cre

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// OrderedEntries returns a sequence that iterates over the entries in the map in the natural sorted order of the keys.
// For keys which do not implement cmp.Ordered, use OrderedEntriesFunc.
func OrderedEntries[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	keys := slices.Collect(maps.Keys(m))
	slices.Sort(keys)
	return func(yield func(K, V) bool) {
		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}

// OrderedEntriesFunc returns a sequence that iterates over the entries in the map based on sorting keys with the cmp func.
// For naturally cmp.Ordered keys, use OrderedEntries.
func OrderedEntriesFunc[K comparable, V any](m map[K]V, cmp func(a, b K) int) iter.Seq2[K, V] {
	keys := slices.Collect(maps.Keys(m))
	slices.SortFunc(keys, cmp)
	return func(yield func(K, V) bool) {
		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
