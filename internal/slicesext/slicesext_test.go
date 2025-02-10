package slicesext

import (
	"math"
	"strconv"
	"testing"
)

func TestConvertIntToString(t *testing.T) {
	t.Parallel()
	input := []int{1, 2, 3, 4, 5}
	expected := []string{"1", "2", "3", "4", "5"}
	result := Convert(input, strconv.Itoa)

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("At index %d: expected %v, got %v", i, expected[i], result[i])
		}
	}
}

func TestConvertStringToInt(t *testing.T) {
	t.Parallel()
	input := []string{"1", "2", "3", "4", "5"}
	expected := []int{1, 2, 3, 4, 5}
	result := Convert(input, func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	})

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("At index %d: expected %v, got %v", i, expected[i], result[i])
		}
	}
}

func TestConvertFloatToInt(t *testing.T) {
	t.Parallel()
	input := []float64{1.1, 2.2, 3.7, 4.5, 5.9}
	expected := []int{1, 2, 4, 5, 6}
	result := Convert(input, func(f float64) int {
		return int(math.Round(f))
	})

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("At index %d: expected %v, got %v", i, expected[i], result[i])
		}
	}
}

func TestConvertEmptySlice(t *testing.T) {
	t.Parallel()
	input := []int{}
	result := Convert(input, strconv.Itoa)

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(result))
	}
}

func TestConvertNilSlice(t *testing.T) {
	t.Parallel()
	var input []int
	result := Convert(input, strconv.Itoa)

	if result == nil {
		t.Error("Expected non-nil empty slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(result))
	}
}
