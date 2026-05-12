package cre

import (
	"cmp"
	"fmt"
)

func ExampleOrderedEntries() {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}
	for _, v := range OrderedEntries(m) {
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}

func ExampleOrderedEntriesFunc() {
	type k struct{ f string }
	m := map[k]int{{"a"}: 1, {"b"}: 2, {"c"}: 3, {"d"}: 4, {"e"}: 5}
	for _, v := range OrderedEntriesFunc(m, func(a, b k) int {
		return cmp.Compare(a.f, b.f)
	}) {
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}
