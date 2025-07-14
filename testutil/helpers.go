package testutil

import (
	"testing"
)

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotNil fails the test if value is nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Fatal("Expected non-nil value, got nil")
	}
}

// AssertNotEqual fails the test if expected == actual
func AssertNotEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected == actual {
		t.Fatalf("Expected %v to not equal %v", expected, actual)
	}
}
