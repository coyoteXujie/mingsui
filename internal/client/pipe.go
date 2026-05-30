package client

import (
	"bufio"
	"io"
	"net"
	"sync"
)

type closeWriter interface {
	CloseWrite() error
}

func copyAndCloseWrite(dst, src net.Conn) (int64, error) {
	n, err := io.Copy(dst, src)
	if cw, ok := dst.(closeWriter); ok {
		_ = cw.CloseWrite()
	} else {
		_ = dst.Close()
	}
	return n, err
}

func proxyBidirectional(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = copyAndCloseWrite(b, a)
	}()
	go func() {
		defer wg.Done()
		_, _ = copyAndCloseWrite(a, b)
	}()

	wg.Wait()
}

func (s *Service) proxy(clientConn, relayConn net.Conn) {
	if s.metrics == nil {
		proxyBidirectional(clientConn, relayConn)
		return
	}

	s.metrics.OpenConnection()
	defer s.metrics.CloseConnection()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := copyAndCloseWrite(relayConn, clientConn)
		s.metrics.AddUploadBytes(n)
	}()
	go func() {
		defer wg.Done()
		n, _ := copyAndCloseWrite(clientConn, relayConn)
		s.metrics.AddDownloadBytes(n)
	}()

	wg.Wait()
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil && c.reader.Buffered() > 0 {
		return c.reader.Read(p)
	}
	return c.Conn.Read(p)
}

func (c bufferedConn) CloseWrite() error {
	if cw, ok := c.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return c.Conn.Close()
}
