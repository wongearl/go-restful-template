package sliceutil

import (
	"sort"
)

// RemoveString removes an item from a slice with a custom function
func RemoveString(slice []string, remove func(item string) bool) []string {
	for i := 0; i < len(slice); i++ {
		if remove(slice[i]) {
			slice = append(slice[:i], slice[i+1:]...)
			i--
		}
	}
	return slice
}

// RemoveSubSlice removes a sub-slice from another one
func RemoveSubSlice[T interface{}](slice []T, sub []int) []T {
	sort.Slice(sub, func(i, j int) bool {
		return i > j
	})

	for _, i := range sub {
		if i < 0 || i >= len(slice) {
			continue
		}
		slice = append(slice[0:i], slice[i+1:]...)
	}
	return slice
}

// SameItem returns a function to check if the item is same to
func SameItem(target string) func(item string) bool {
	return func(item string) bool {
		return target == item
	}
}

func HasString(slice []string, str string) bool {
	return HasAnyString(slice, str)
}

// AddToSlice adds an item to a slice without duplicated
func AddToSlice(item string, array []string) []string {
	if !HasString(array, item) {
		array = append(array, item)
	}
	return array
}

func HasAnyString(slice []string, items ...string) bool {
	for _, s := range slice {
		for _, i := range items {
			if s == i {
				return true
			}
		}
	}
	return false
}

func MapHasValue(annotations map[string]string, value string) (bool, string) {
	if annotations == nil {
		return false, ""
	}
	for k, v := range annotations {
		if v == value {
			return true, k
		}
	}
	return false, ""
}

func GetMapKeyFromValue(tmpMap map[string]string, value string) string {
	for k, v := range tmpMap {
		if v == value {
			return k
		}
	}
	return ""
}
