package runner

import (
	"strings"
	"testing"
)

func TestRingBufferBasicWrite(t *testing.T) {
	rb := NewRingBuffer(10)
	n, err := rb.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected 5 bytes written, got %d", n)
	}
	if rb.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", rb.String())
	}
	if rb.TotalBytesWritten() != 5 {
		t.Fatalf("expected 5 total bytes, got %d", rb.TotalBytesWritten())
	}
}

func TestRingBufferWrapAround(t *testing.T) {
	rb := NewRingBuffer(5)
	_, _ = rb.Write([]byte("abcde"))
	if rb.String() != "abcde" {
		t.Fatalf("expected 'abcde', got %q", rb.String())
	}

	// Write 2 more bytes; 'ab' should be dropped
	_, _ = rb.Write([]byte("fg"))
	if rb.String() != "cdefg" {
		t.Fatalf("expected 'cdefg', got %q", rb.String())
	}
	if rb.TotalBytesWritten() != 7 {
		t.Fatalf("expected 7 total bytes, got %d", rb.TotalBytesWritten())
	}
}

func TestRingBufferLargeWrite(t *testing.T) {
	rb := NewRingBuffer(5)
	// Write string larger than buffer capacity
	_, _ = rb.Write([]byte("0123456789"))
	if rb.String() != "56789" {
		t.Fatalf("expected '56789', got %q", rb.String())
	}
	if rb.TotalBytesWritten() != 10 {
		t.Fatalf("expected 10 total bytes, got %d", rb.TotalBytesWritten())
	}
}

func TestRingBufferReset(t *testing.T) {
	rb := NewRingBuffer(5)
	_, _ = rb.Write([]byte("test"))
	rb.Reset()
	if rb.String() != "" {
		t.Fatalf("expected empty string after reset, got %q", rb.String())
	}
	if rb.TotalBytesWritten() != 0 {
		t.Fatalf("expected 0 total bytes, got %d", rb.TotalBytesWritten())
	}
}

func TestRingBufferLongLogStream(t *testing.T) {
	rb := NewRingBuffer(1024)
	chunk := strings.Repeat("A", 200) + "\n"
	for i := 0; i < 20; i++ {
		_, _ = rb.Write([]byte(chunk))
	}
	if len(rb.Bytes()) != 1024 {
		t.Fatalf("expected buffer to be exactly 1024 bytes, got %d", len(rb.Bytes()))
	}
	if rb.TotalBytesWritten() != 20*int64(len(chunk)) {
		t.Fatalf("expected %d total bytes, got %d", 20*len(chunk), rb.TotalBytesWritten())
	}
}

