package tunnel

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestStreamConnReadDeadline(t *testing.T) {
	stream := &streamConn{
		readCh: make(chan []byte),
		done:   make(chan struct{}),
	}
	if err := stream.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error = %v", err)
	}

	_, err := stream.Read(make([]byte, 1))
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("Read error = %v, want %v", err, os.ErrDeadlineExceeded)
	}
}
