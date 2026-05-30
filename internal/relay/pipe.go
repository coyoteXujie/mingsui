package relay

import (
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

func (s *Server) proxy(clientConn, targetConn net.Conn) {
	if s.metrics == nil {
		proxyBidirectional(clientConn, targetConn)
		return
	}

	s.metrics.OpenConnection()
	defer s.metrics.CloseConnection()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := copyAndCloseWrite(targetConn, clientConn)
		s.metrics.AddUploadBytes(n)
	}()
	go func() {
		defer wg.Done()
		n, _ := copyAndCloseWrite(clientConn, targetConn)
		s.metrics.AddDownloadBytes(n)
	}()

	wg.Wait()
}
