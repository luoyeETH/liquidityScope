package indexer

import (
	"reflect"
	"testing"
)

func TestSplitRange(t *testing.T) {
	got, err := SplitRange(100, 105, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []BlockRange{
		{From: 100, To: 101},
		{From: 102, To: 103},
		{From: 104, To: 105},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranges mismatch: %+v != %+v", got, want)
	}
}

func TestSplitRangeSingle(t *testing.T) {
	got, err := SplitRange(5, 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []BlockRange{{From: 5, To: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranges mismatch: %+v != %+v", got, want)
	}
}

func TestSplitRangeInvalid(t *testing.T) {
	if _, err := SplitRange(10, 9, 1); err == nil {
		t.Fatalf("expected error for invalid range")
	}
	if _, err := SplitRange(1, 10, 0); err == nil {
		t.Fatalf("expected error for zero batch size")
	}
}
