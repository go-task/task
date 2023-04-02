package orderedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFromMap(t *testing.T) {
	m := map[int]string{3: "three", 1: "one", 2: "two"}
	om := FromMap(m)
	assert.Len(t, om.m, 3)
	assert.Len(t, om.s, 3)
	assert.ElementsMatch(t, []int{1, 2, 3}, om.s)
	for key, value := range m {
		assert.Equal(t, om.Get(key), value)
	}
}

func TestSetGetExists(t *testing.T) {
	om := New[int, string]()
	assert.False(t, om.Exists(1))
	assert.Equal(t, "", om.Get(1))
	om.Set(1, "one")
	assert.True(t, om.Exists(1))
	assert.Equal(t, "one", om.Get(1))
}

func TestSort(t *testing.T) {
	om := New[int, string]()
	om.Set(3, "three")
	om.Set(1, "one")
	om.Set(2, "two")
	om.Sort()
	assert.Equal(t, []int{1, 2, 3}, om.s)
}

func TestSortFunc(t *testing.T) {
	om := New[int, string]()
	om.Set(3, "three")
	om.Set(1, "one")
	om.Set(2, "two")
	om.SortFunc(func(i, j int) bool {
		return i > j
	})
	assert.Equal(t, []int{3, 2, 1}, om.s)
}

func TestKeysValues(t *testing.T) {
	om := New[int, string]()
	om.Set(3, "three")
	om.Set(1, "one")
	om.Set(2, "two")
	assert.Equal(t, []int{3, 1, 2}, om.Keys())
	assert.Equal(t, []string{"three", "one", "two"}, om.Values())
}

func Range(t *testing.T) {
	om := New[int, string]()
	om.Set(3, "three")
	om.Set(1, "one")
	om.Set(2, "two")

	expectedKeys := []int{3, 1, 2}
	expectedValues := []string{"three", "one", "two"}

	keys := make([]int, 0, len(expectedKeys))
	values := make([]string, 0, len(expectedValues))

	err := om.Range(func(key int, value string) error {
		keys = append(keys, key)
		values = append(values, value)
		return nil
	})

	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedKeys, keys)
	assert.ElementsMatch(t, expectedValues, values)
}

func TestOrderedMapMerge(t *testing.T) {
	om1 := New[string, int]()
	om1.Set("a", 1)
	om1.Set("b", 2)

	om2 := New[string, int]()
	om2.Set("b", 3)
	om2.Set("c", 4)

	om1.Merge(om2)

	expectedKeys := []string{"a", "b", "c"}
	expectedValues := []int{1, 3, 4}

	assert.Equal(t, len(expectedKeys), len(om1.s))
	assert.Equal(t, len(expectedKeys), len(om1.m))

	for i, key := range expectedKeys {
		assert.True(t, om1.Exists(key))
		assert.Equal(t, expectedValues[i], om1.Get(key))
	}
}

func TestUnmarshalYAML(t *testing.T) {
	yamlString := `
3: three
1: one
2: two
`
	var om OrderedMap[int, string]
	err := yaml.Unmarshal([]byte(yamlString), &om)
	require.NoError(t, err)

	expectedKeys := []int{3, 1, 2}
	expectedValues := []string{"three", "one", "two"}

	assert.Equal(t, expectedKeys, om.Keys())
	assert.Equal(t, expectedValues, om.Values())
}
