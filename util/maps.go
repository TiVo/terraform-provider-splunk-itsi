package util

import "fmt"

// Given a mapping of items between two isomorphic sets,
// returns a new map with the keys and values swapped.
// Panics if the input map has duplicate values.
func ReverseMap[KeyType, ValueType comparable](m map[KeyType]ValueType) map[ValueType]KeyType {
	if m == nil {
		return nil
	}

	n := make(map[ValueType]KeyType, len(m))
	for k, v := range m {
		if _, ok := n[v]; ok {
			err := fmt.Sprintf(`ReverseMap: duplicate value in the original map: %#v`, m)
			panic(err)
		}
		n[v] = k
	}
	return n
}
