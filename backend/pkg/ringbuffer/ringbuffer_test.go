package ringbuffer

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

func TestRingBufferBasicWriteRead(t *testing.T) {
	rb := New(64)

	input := []byte("ORDER_ID,CUSTOMER_ID,GROSS_AMOUNT\nORD-1001,CUST-99,1499.00\n")
	n, err := rb.Write(input)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != len(input) {
		t.Fatalf("expected to write %d bytes, wrote %d", len(input), n)
	}

	buf := make([]byte, len(input))
	readBytes, err := rb.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if readBytes != len(input) {
		t.Fatalf("expected to read %d bytes, got %d", len(input), readBytes)
	}

	if !bytes.Equal(buf, input) {
		t.Fatalf("read content did not match written content")
	}
}

func TestRingBufferWrapAround(t *testing.T) {
	// Small buffer of 16 bytes to force wrap-around
	rb := New(16)

	// Step 1: Write 10 bytes
	_, _ = rb.Write([]byte("0123456789"))
	// Step 2: Read 6 bytes -> readPtr advances by 6
	out := make([]byte, 6)
	_, _ = rb.Read(out)
	if string(out) != "012345" {
		t.Fatalf("expected '012345', got '%s'", string(out))
	}

	// Step 3: Write 10 bytes -> writePtr wraps around index 16
	_, _ = rb.Write([]byte("abcdefghij"))

	// Step 4: Read all remaining bytes
	remaining := make([]byte, 14)
	n, err := rb.Read(remaining)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	expected := "6789abcdefghij"
	if string(remaining[:n]) != expected {
		t.Fatalf("expected '%s', got '%s'", expected, string(remaining[:n]))
	}
}

func TestRingBufferConcurrentStreaming(t *testing.T) {
	// 128 KB buffer streaming 5 MB of data concurrently
	rb := NewFromKB(128)

	dataSize := 1024 * 1024 // 1 MB
	sampleData := make([]byte, dataSize)
	for i := range sampleData {
		sampleData[i] = byte(i % 256)
	}

	var wg sync.WaitGroup
	var received bytes.Buffer

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		chunkSize := 4096
		for i := 0; i < len(sampleData); i += chunkSize {
			end := i + chunkSize
			if end > len(sampleData) {
				end = len(sampleData)
			}
			_, err := rb.Write(sampleData[i:end])
			if err != nil {
				t.Errorf("write error: %v", err)
				return
			}
		}
		_ = rb.Close()
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 8192)
		for {
			n, err := rb.Read(buf)
			if n > 0 {
				received.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("read error: %v", err)
				break
			}
		}
	}()

	wg.Wait()

	if !bytes.Equal(received.Bytes(), sampleData) {
		t.Fatalf("streamed data does not match original data (received %d bytes)", received.Len())
	}
}

func TestRingBufferCloseAndEOF(t *testing.T) {
	rb := New(32)
	_, _ = rb.Write([]byte("hello world"))
	_ = rb.Close()

	buf := make([]byte, 64)
	n, err := rb.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(buf[:n]) != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", string(buf[:n]))
	}

	// Subsequent read must return io.EOF
	_, err = rb.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	// Write on closed buffer must return ErrClosed
	_, err = rb.Write([]byte("test"))
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestRingBufferReset(t *testing.T) {
	rb := New(32)
	_, _ = rb.Write([]byte("abc"))
	if rb.Len() != 3 {
		t.Fatalf("expected length 3, got %d", rb.Len())
	}

	rb.Reset()
	if rb.Len() != 0 {
		t.Fatalf("expected length 0 after reset, got %d", rb.Len())
	}

	// Verify buffer is writable again after reset
	n, err := rb.Write([]byte("xyz"))
	if err != nil || n != 3 {
		t.Fatalf("failed to write after reset: %v", err)
	}
}

func TestRingBufferTimeoutProtection(t *testing.T) {
	rb := New(16)
	done := make(chan bool)

	go func() {
		buf := make([]byte, 8)
		// Will block since empty
		_, _ = rb.Read(buf)
		done <- true
	}()

	select {
	case <-done:
		t.Fatalf("read should have blocked on empty buffer")
	case <-time.After(50 * time.Millisecond):
		// Expected blocking
	}

	// Unblock by closing
	_ = rb.Close()
	select {
	case <-done:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("reader failed to unblock on close")
	}
}

