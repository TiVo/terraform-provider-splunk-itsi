package util

import (
	"sort"
	"testing"
)

func TestSet_Add(t *testing.T) {
	s := NewSet[int]()
	s.Add(1, 2, 3)

	if !s.Contains(1) {
		t.Errorf("Expected set to contain 1")
	}

	if !s.Contains(2) {
		t.Errorf("Expected set to contain 2")
	}

	if !s.Contains(3) {
		t.Errorf("Expected set to contain 3")
	}
}

func TestSet_Contains(t *testing.T) {
	s := NewSet("apple", "banana", "orange")

	if !s.Contains("apple") {
		t.Errorf("Expected set to contain 'apple'")
	}

	if !s.Contains("banana") {
		t.Errorf("Expected set to contain 'banana'")
	}

	if !s.Contains("orange") {
		t.Errorf("Expected set to contain 'orange'")
	}

	if s.Contains("grape") {
		t.Errorf("Expected set to not contain 'grape'")
	}
}

func TestSet_ToSlice(t *testing.T) {
	s := NewSet(1, 2, 3)
	list := s.ToSlice()

	if len(list) != 3 {
		t.Errorf("Expected slice length to be 3")
	}

	sort.Ints(list)

	if len(list) != 3 {
		t.Errorf("Expected list length to be 3")
	}

	if list[0] != 1 {
		t.Errorf("Expected first element to be 1")
	}

	if list[1] != 2 {
		t.Errorf("Expected second element to be 2")
	}

	if list[2] != 3 {
		t.Errorf("Expected third element to be 3")
	}
}

func TestNewSetFromSlice(t *testing.T) {
	slice := []string{"apple", "banana", "orange"}
	set := NewSetFromSlice(slice)

	if !set.Contains("apple") {
		t.Errorf("Expected set to contain 'apple'")
	}

	if !set.Contains("banana") {
		t.Errorf("Expected set to contain 'banana'")
	}

	if !set.Contains("orange") {
		t.Errorf("Expected set to contain 'orange'")
	}

	if set.Contains("grape") {
		t.Errorf("Expected set to not contain 'grape'")
	}
}

func TestUnique(t *testing.T) {
	list := []int{1, 2, 2, 3, 3, 3}
	uniqueList := Unique(list)

	sort.Ints(uniqueList)

	if len(uniqueList) != 3 {
		t.Errorf("Expected unique list length to be 3")
	}

	if uniqueList[0] != 1 {
		t.Errorf("Expected first element to be 1")
	}

	if uniqueList[1] != 2 {
		t.Errorf("Expected second element to be 2")
	}

	if uniqueList[2] != 3 {
		t.Errorf("Expected third element to be 3")
	}
}
