package sort

import (
	"cmp"
	"slices"
	"strings"
)

// A Sorter is any function that sorts a set of tasks.
type Sorter func(items []string, namespaces []string) []string

// NoSort leaves the tasks in the order they are defined.
func NoSort(items []string, namespaces []string) []string {
	return items
}

// AlphaNumeric sorts the JSON output so that tasks are in alpha numeric order
// by task name.
func AlphaNumeric(items []string, namespaces []string) []string {
	slices.Sort(items)
	return items
}

// AlphaNumericWithRootTasksFirst sorts the JSON output so that tasks are in
// alpha numeric order by task name. It will also ensure that tasks that are not
// namespaced will be listed before tasks that are. We detect this by searching
// for a ':' in the task name.
func AlphaNumericWithRootTasksFirst(items []string, namespaces []string) []string {
	if len(namespaces) > 0 {
		return AlphaNumeric(items, namespaces)
	}
	slices.SortFunc(items, func(a, b string) int {
		aContainsColon := strings.Contains(a, ":")
		bContainsColon := strings.Contains(b, ":")
		if aContainsColon == bContainsColon {
			return cmp.Compare(a, b)
		}
		if !aContainsColon && bContainsColon {
			return -1
		}
		return 1
	})
	return items
}
