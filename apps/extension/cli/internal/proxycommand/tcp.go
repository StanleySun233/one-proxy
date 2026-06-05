package proxycommand

import (
	"io"
	"sync"
)

func RunTCP(cfg Config, input io.Reader, output io.Writer) error {
	conn, err := DialUpstream(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, input)
		if halfCloser, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = halfCloser.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(output, conn)
	}()
	wg.Wait()
	return nil
}
