package runner

import (
	"io"
	"sync"
)

// RingBuffer is a thread-safe, fixed-capacity circular byte buffer that retains the
// most recent N bytes written to it, preventing unbounded memory growth (OOM).
type RingBuffer struct {
	mu       sync.Mutex
	buf      []byte
	size     int
	start    int
	length   int
	isFull   bool
	totalIn  int64
}

// NewRingBuffer allocates a new RingBuffer with the given maximum byte capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 1024 * 1024 // 1 MB default
	}
	return &RingBuffer{
		buf:  make([]byte, capacity),
		size: capacity,
	}
}

// Write appends p to the circular buffer, dropping the oldest bytes if capacity is exceeded.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n = len(p)
	r.totalIn += int64(n)

	if n >= r.size {
		// Incoming slice is larger than entire capacity: keep only the trailing r.size bytes
		copy(r.buf, p[n-r.size:])
		r.start = 0
		r.length = r.size
		r.isFull = true
		return n, nil
	}

	for i := 0; i < n; i++ {
		pos := (r.start + r.length) % r.size
		r.buf[pos] = p[i]
		if r.isFull {
			r.start = (r.start + 1) % r.size
		} else {
			r.length++
			if r.length == r.size {
				r.isFull = true
			}
		}
	}

	return n, nil
}

// Bytes returns a contiguous snapshot of all retained bytes in chronological order.
func (r *RingBuffer) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]byte, r.length)
	if !r.isFull {
		copy(out, r.buf[:r.length])
		return out
	}

	// Two segments: from start to end of buffer, and from 0 to remainder
	n1 := r.size - r.start
	copy(out[:n1], r.buf[r.start:])
	copy(out[n1:], r.buf[:r.start])
	return out
}

// String returns the contents of the buffer as a string.
func (r *RingBuffer) String() string {
	return string(r.Bytes())
}

// TotalBytesWritten returns the cumulative number of bytes written before truncation.
func (r *RingBuffer) TotalBytesWritten() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.totalIn
}

// Reset clears the buffer.
func (r *RingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.start = 0
	r.length = 0
	r.isFull = false
	r.totalIn = 0
}

var _ io.Writer = (*RingBuffer)(nil)

