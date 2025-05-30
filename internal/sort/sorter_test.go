package sort

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlphaNumericWithRootTasksFirst_Sort(t *testing.T) {
	t.Parallel()

	item1 := "a-item1"
	item2 := "m-item2"
	item3 := "ns1:item3"
	item4 := "ns2:item4"
	item5 := "z-item5"
	item6 := "ns3:item6"

	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{
			name:  "no namespace items sorted alphabetically first",
			items: []string{item3, item2, item1},
			want:  []string{item1, item2, item3},
		},
		{
			name:  "namespace items sorted alphabetically after non-namespaced items",
			items: []string{item3, item4, item5},
			want:  []string{item5, item3, item4},
		},
		{
			name:  "all items sorted alphabetically with root items first",
			items: []string{item6, item5, item4, item3, item2, item1},
			want:  []string{item1, item2, item5, item3, item4, item6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			AlphaNumericWithRootTasksFirst(tt.items, nil)
			assert.Equal(t, tt.want, tt.items)
		})
	}
}

func TestAlphaNumeric_Sort(t *testing.T) {
	t.Parallel()

	item1 := "a-item1"
	item2 := "m-item2"
	item3 := "ns1:item3"
	item4 := "ns2:item4"
	item5 := "z-item5"
	item6 := "ns3:item6"

	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{
			name:  "all items sorted alphabetically",
			items: []string{item3, item2, item5, item1, item4, item6},
			want:  []string{item1, item2, item3, item4, item6, item5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			AlphaNumeric(tt.items, nil)
			assert.Equal(t, tt.want, tt.items)
		})
	}
}

func TestNoSort_Sort(t *testing.T) {
	t.Parallel()

	item1 := "a-item1"
	item2 := "m-item2"
	item3 := "ns1:item3"
	item4 := "ns2:item4"
	item5 := "z-item5"
	item6 := "ns3:item6"

	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{
			name:  "all items in order of definition",
			items: []string{item3, item2, item5, item1, item4, item6},
			want:  []string{item3, item2, item5, item1, item4, item6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			NoSort(tt.items, nil)
			assert.Equal(t, tt.want, tt.items)
		})
	}
}
