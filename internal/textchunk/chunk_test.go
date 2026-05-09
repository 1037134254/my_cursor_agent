package textchunk

import "testing"

func TestSplitOverlap(t *testing.T) {
	chunks := Split("abcdefghij", 4, 1)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %v", chunks)
	}
}

func TestSplitEmpty(t *testing.T) {
	if Split("   ", 100, 0) != nil {
		t.Fatal("expected nil")
	}
}
