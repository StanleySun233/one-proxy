package tunnel

import (
	"errors"
	"io"
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

func TestStreamConnReadDrainsBufferedDataAfterRemoteClose(t *testing.T) {
	stream := &streamConn{
		readCh: make(chan []byte, 1),
		done:   make(chan struct{}),
	}
	stream.readCh <- []byte("complete\n")
	close(stream.readCh)
	stream.remoteClosed = true
	stream.closeErr = io.EOF

	buffer := make([]byte, 32)
	n, err := stream.Read(buffer)
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if string(buffer[:n]) != "complete\n" {
		t.Fatalf("Read = %q", string(buffer[:n]))
	}
	_, err = stream.Read(buffer)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Read error = %v, want %v", err, io.EOF)
	}
}
