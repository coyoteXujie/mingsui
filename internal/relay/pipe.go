package relay

import (
	"io"
	"net"
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
