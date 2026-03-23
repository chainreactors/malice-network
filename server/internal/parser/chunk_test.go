package parser

import (
	"testing"
)

func TestCount(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		max     int
		want    int
	}{
		{"normal ceil division", make([]byte, 10), 3, 4},
		{"empty content positive max", []byte{}, 5, 0},
		{"non-empty content zero max", make([]byte, 5), 0, 1},
		{"non-empty content negative max", make([]byte, 5), -1, 1},
		{"empty content zero max", []byte{}, 0, 0},
		{"empty content negative max", []byte{}, -1, 0},
		{"exact multiple", make([]byte, 6), 3, 2},
		{"max equals length", make([]byte, 5), 5, 1},
		{"max greater than length", make([]byte, 3), 10, 1},
		{"single byte", make([]byte, 1), 1, 1},
		{"max of 1", make([]byte, 5), 1, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Count(tc.content, tc.max)
			if got != tc.want {
				t.Errorf("Count(len=%d, max=%d) = %d, want %d",
					len(tc.content), tc.max, got, tc.want)
			}
		})
	}
}

func TestChunked_CorrectSizes(t *testing.T) {
	content := []byte("abcdefghij") // 10 bytes
	max := 3
	ch := Chunked(content, max)

	expected := []string{"abc", "def", "ghi", "j"}
	var got []string
	for chunk := range ch {
		got = append(got, string(chunk))
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d chunks, got %d", len(expected), len(got))
	}
	for i, e := range expected {
		if got[i] != e {
			t.Errorf("chunk[%d] = %q, want %q", i, got[i], e)
		}
	}
}

func TestChunked_EmptyContent(t *testing.T) {
	ch := Chunked([]byte{}, 5)
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 chunks for empty content, got %d", count)
	}
}

func TestChunked_ZeroMax(t *testing.T) {
	content := []byte("hello")
	ch := Chunked(content, 0)

	var chunks [][]byte
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for zero max, got %d", len(chunks))
	}
	if string(chunks[0]) != "hello" {
		t.Fatalf("expected full content in single chunk, got %q", string(chunks[0]))
	}
}

func TestChunked_NegativeMax(t *testing.T) {
	content := []byte("world")
	ch := Chunked(content, -1)

	var chunks [][]byte
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for negative max, got %d", len(chunks))
	}
	if string(chunks[0]) != "world" {
		t.Fatalf("expected full content, got %q", string(chunks[0]))
	}
}

func TestChunked_ExactMultiple(t *testing.T) {
	content := []byte("abcdef") // 6 bytes
	ch := Chunked(content, 3)

	var chunks []string
	for chunk := range ch {
		chunks = append(chunks, string(chunk))
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != "abc" || chunks[1] != "def" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestChunked_EmptyContentZeroMax(t *testing.T) {
	// When max <= 0 and content is empty, nothing should be sent.
	ch := Chunked([]byte{}, 0)
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 chunks for empty content with zero max, got %d", count)
	}
}

func TestChunked_EmptyContentNegativeMax(t *testing.T) {
	ch := Chunked([]byte{}, -5)
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 chunks for empty content with negative max, got %d", count)
	}
}

func TestChunked_NilContent(t *testing.T) {
	// nil behaves like empty slice
	ch := Chunked(nil, 10)
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 chunks for nil content, got %d", count)
	}
}

func TestChunked_SingleByte(t *testing.T) {
	ch := Chunked([]byte{0x42}, 1)
	var chunks [][]byte
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0][0] != 0x42 {
		t.Fatalf("unexpected chunk content: %v", chunks[0])
	}
}

// TestCount_ConsistencyWithChunked verifies that Count returns the same number
// of chunks that Chunked actually produces.
func TestCount_ConsistencyWithChunked(t *testing.T) {
	cases := []struct {
		size int
		max  int
	}{
		{0, 5},
		{1, 1},
		{10, 3},
		{6, 3},
		{7, 7},
		{100, 13},
	}

	for _, tc := range cases {
		content := make([]byte, tc.size)
		expected := Count(content, tc.max)

		ch := Chunked(content, tc.max)
		actual := 0
		for range ch {
			actual++
		}

		if expected != actual {
			t.Errorf("Count(size=%d, max=%d)=%d but Chunked produced %d chunks",
				tc.size, tc.max, expected, actual)
		}
	}
}
