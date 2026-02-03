package main

import "sort"

func Ternary[T any](condition bool, value1, value2 T) T {
	if condition {
		return value1
	}
	return value2
}

// KeyValuePair represents a key-value pair
type KeyValuePair[K comparable, V any] struct {
	Key   K
	Value V
}

// SortMapByKey sorts a map by its keys and returns a sorted slice of key-value pairs.
func SortMapByKey[K comparable, V any](m map[K]V, less func(a, b K) bool) []KeyValuePair[K, *V] {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	// Sort keys using the provided comparator
	sort.Slice(keys, func(i, j int) bool {
		return less(keys[i], keys[j])
	})

	// Create a sorted slice of key-value pairs
	sorted := make([]KeyValuePair[K, *V], 0, len(m))
	for _, k := range keys {
		H := m[k]
		sorted = append(sorted, KeyValuePair[K, *V]{Key: k, Value: &H})
	}

	return sorted
}

func SortMapByKeyDesc[K string | int, V any](m map[K]V) []KeyValuePair[K, *V] {
	return SortMapByKey(m, func(a, b K) bool { return a < b })
}
