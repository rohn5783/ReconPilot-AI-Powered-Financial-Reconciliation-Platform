package ringbuffer

import (
	"errors"
	"io"
	"sync"
)

var (
	// ErrClosed is returned when an operation is performed on a closed ring buffer.
	ErrClosed = errors.New("ring buffer is closed")
	// ErrTooLarge is returned when writing a slice larger than the total buffer capacity.
	ErrTooLarge = errors.New("write exceeds maximum ring buffer capacity")
)

// RingBuffer implements a bounded circular buffer that satisfies io.Reader, io.Writer, and io.Closer.
// It guarantees zero OOM spikes when streaming massive CSV ingestion files by enforcing a hard upper limit.
type RingBuffer struct {
	buf      []byte
	size     int
	readPtr  int
	writePtr int
	count    int
	closed   bool
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
}

// New creates a new bounded RingBuffer with the specified capacity in bytes.
func New(capacityBytes int) *RingBuffer {
	if capacityBytes <= 0 {
		capacityBytes = 2048 * 1024 // 2048 KB default
	}

	rb := &RingBuffer{
		buf:  make([]byte, capacityBytes),
		size: capacityBytes,
	}
	rb.notEmpty = sync.NewCond(&rb.mu)
	rb.notFull = sync.NewCond(&rb.mu)
	return rb
}

// NewFromKB initializes a ring buffer specifying size in kilobytes.
func NewFromKB(kb int) *RingBuffer {
	if kb <= 0 {
		kb = 2048
	}
	return New(kb * 1024)
}

// Write writes data to the buffer. If the buffer is full, it blocks until space is freed.
// If the buffer is closed, ErrClosed is returned.
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return 0, ErrClosed
	}

	totalToWrite := len(p)
	if totalToWrite == 0 {
		return 0, nil
	}

	for totalToWrite > 0 {
		// Wait until there is space available to write
		for rb.count == rb.size && !rb.closed {
			rb.notFull.Wait()
		}

		if rb.closed {
			return n, ErrClosed
		}

		space := rb.size - rb.count
		toCopy := totalToWrite
		if toCopy > space {
			toCopy = space
		}

		// Calculate sequential space available before wrapping
		firstChunk := rb.size - rb.writePtr
		if firstChunk > toCopy {
			firstChunk = toCopy
		}

		copy(rb.buf[rb.writePtr:rb.writePtr+firstChunk], p[n:n+firstChunk])
		secondChunk := toCopy - firstChunk
		if secondChunk > 0 {
			copy(rb.buf[0:secondChunk], p[n+firstChunk:n+toCopy])
		}

		rb.writePtr = (rb.writePtr + toCopy) % rb.size
		rb.count += toCopy
		n += toCopy
		totalToWrite -= toCopy

		// Signal readers that new data is available
		rb.notEmpty.Signal()
	}

	return n, nil
}

// Read reads up to len(p) bytes into p. It blocks until at least 1 byte is available or EOF.
func (rb *RingBuffer) Read(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(p) == 0 {
		return 0, nil
	}

	// Wait until there is data or the buffer is closed
	for rb.count == 0 {
		if rb.closed {
			return 0, io.EOF
		}
		rb.notEmpty.Wait()
	}

	toRead := len(p)
	if toRead > rb.count {
		toRead = rb.count
	}

	// Sequential available before wrapping
	firstChunk := rb.size - rb.readPtr
	if firstChunk > toRead {
		firstChunk = toRead
	}

	copy(p[0:firstChunk], rb.buf[rb.readPtr:rb.readPtr+firstChunk])
	secondChunk := toRead - firstChunk
	if secondChunk > 0 {
		copy(p[firstChunk:toRead], rb.buf[0:secondChunk])
	}

	rb.readPtr = (rb.readPtr + toRead) % rb.size
	rb.count -= toRead
	n = toRead

	// Signal writers that space is now available
	rb.notFull.Signal()

	return n, nil
}

// Close closes the ring buffer for writing. Subsequent reads will drain remaining buffered data and return io.EOF.
func (rb *RingBuffer) Close() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.closed {
		rb.closed = true
		rb.notEmpty.Broadcast()
		rb.notFull.Broadcast()
	}
	return nil
}

// Reset clears the buffer pointers and state while retaining allocated memory.
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.readPtr = 0
	rb.writePtr = 0
	rb.count = 0
	rb.closed = false
	rb.notEmpty.Broadcast()
	rb.notFull.Broadcast()
}

// Len returns the current number of buffered bytes.
func (rb *RingBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

// Cap returns the total buffer capacity in bytes.
func (rb *RingBuffer) Cap() int {
	return rb.size
}

// Available returns remaining writable space before blocking.
func (rb *RingBuffer) Available() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.size - rb.count
}

// IsClosed checks if the buffer was closed.
func (rb *RingBuffer) IsClosed() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.closed
}

