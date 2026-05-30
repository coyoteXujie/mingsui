package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	socksVersion5       = 0x05
	socksMethodNoAuth   = 0x00
	socksMethodNoAccept = 0xff
	socksCommandConnect = 0x01

	socksReplySucceeded      = 0x00
	socksReplyGeneralFailure = 0x01
	socksReplyCommandNotSup  = 0x07
	socksReplyAddrNotSup     = 0x08

	socksATYPIPv4   = 0x01
	socksATYPDomain = 0x03
	socksATYPIPv6   = 0x04
)

func (s *Service) handleSOCKS(conn net.Conn) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	target, reply, err := readSOCKS5Target(conn)
	if err != nil {
		_ = writeSOCKS5Reply(conn, reply)
		s.logger.Printf("socks handshake failed from %s: %v", conn.RemoteAddr(), err)
		return
	}

	relay, err := s.openRelay(target)
	if err != nil {
		_ = writeSOCKS5Reply(conn, socksReplyGeneralFailure)
		s.logger.Printf("relay connect failed target=%s: %v", target, err)
		return
	}
	defer relay.Close()

	if err := writeSOCKS5Reply(conn, socksReplySucceeded); err != nil {
		s.logger.Printf("socks reply failed target=%s: %v", target, err)
		return
	}
	_ = conn.SetDeadline(time.Time{})

	s.logger.Printf("proxy connected target=%s", target)
	proxyBidirectional(conn, relay)
}

func readSOCKS5Target(rw io.ReadWriter) (string, byte, error) {
	var hello [2]byte
	if _, err := io.ReadFull(rw, hello[:]); err != nil {
		return "", socksReplyGeneralFailure, err
	}
	if hello[0] != socksVersion5 {
		return "", socksReplyGeneralFailure, fmt.Errorf("unsupported socks version %d", hello[0])
	}

	methods := make([]byte, int(hello[1]))
	if _, err := io.ReadFull(rw, methods); err != nil {
		return "", socksReplyGeneralFailure, err
	}

	hasNoAuth := false
	for _, method := range methods {
		if method == socksMethodNoAuth {
			hasNoAuth = true
			break
		}
	}
	if !hasNoAuth {
		_, _ = rw.Write([]byte{socksVersion5, socksMethodNoAccept})
		return "", socksReplyGeneralFailure, fmt.Errorf("client offered no supported auth method")
	}
	if _, err := rw.Write([]byte{socksVersion5, socksMethodNoAuth}); err != nil {
		return "", socksReplyGeneralFailure, err
	}

	var req [4]byte
	if _, err := io.ReadFull(rw, req[:]); err != nil {
		return "", socksReplyGeneralFailure, err
	}
	if req[0] != socksVersion5 {
		return "", socksReplyGeneralFailure, fmt.Errorf("unsupported request version %d", req[0])
	}
	if req[1] != socksCommandConnect {
		return "", socksReplyCommandNotSup, fmt.Errorf("unsupported socks command %d", req[1])
	}

	host, reply, err := readSOCKS5Host(rw, req[3])
	if err != nil {
		return "", reply, err
	}

	var portBytes [2]byte
	if _, err := io.ReadFull(rw, portBytes[:]); err != nil {
		return "", socksReplyGeneralFailure, err
	}
	port := int(binary.BigEndian.Uint16(portBytes[:]))
	if port <= 0 {
		return "", socksReplyGeneralFailure, fmt.Errorf("invalid port %d", port)
	}

	return net.JoinHostPort(host, strconv.Itoa(port)), socksReplyGeneralFailure, nil
}

func readSOCKS5Host(r io.Reader, atyp byte) (string, byte, error) {
	switch atyp {
	case socksATYPIPv4:
		var ip [4]byte
		if _, err := io.ReadFull(r, ip[:]); err != nil {
			return "", socksReplyGeneralFailure, err
		}
		return net.IP(ip[:]).String(), socksReplyGeneralFailure, nil
	case socksATYPDomain:
		var lenBuf [1]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return "", socksReplyGeneralFailure, err
		}
		name := make([]byte, int(lenBuf[0]))
		if _, err := io.ReadFull(r, name); err != nil {
			return "", socksReplyGeneralFailure, err
		}
		if len(name) == 0 {
			return "", socksReplyGeneralFailure, fmt.Errorf("domain name is empty")
		}
		return string(name), socksReplyGeneralFailure, nil
	case socksATYPIPv6:
		var ip [16]byte
		if _, err := io.ReadFull(r, ip[:]); err != nil {
			return "", socksReplyGeneralFailure, err
		}
		return net.IP(ip[:]).String(), socksReplyGeneralFailure, nil
	default:
		return "", socksReplyAddrNotSup, fmt.Errorf("unsupported address type %d", atyp)
	}
}

func writeSOCKS5Reply(w io.Writer, reply byte) error {
	_, err := w.Write([]byte{
		socksVersion5,
		reply,
		0x00,
		socksATYPIPv4,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00,
	})
	return err
}
