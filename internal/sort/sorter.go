package sort

import (
	"slices"
	"sort"
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
	sort.Slice(items, func(i, j int) bool {
		iContainsColon := strings.Contains(items[i], ":")
		jContainsColon := strings.Contains(items[j], ":")
		if iContainsColon == jContainsColon {
			return items[i] < items[j]
		}
		if !iContainsColon && jContainsColon {
			return true
		}
		return false
	})
	return items
}
