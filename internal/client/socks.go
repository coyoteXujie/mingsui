package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
)

const (
	socksVersion5       = 0x05
	socksMethodNoAuth   = 0x00
	socksMethodUserPass = 0x02
	socksMethodNoAccept = 0xff
	socksCommandConnect = 0x01

	socksReplySucceeded      = 0x00
	socksReplyGeneralFailure = 0x01
	socksReplyCommandNotSup  = 0x07
	socksReplyAddrNotSup     = 0x08

	socksATYPIPv4   = 0x01
	socksATYPDomain = 0x03
	socksATYPIPv6   = 0x04

	socksUserPassVersion = 0x01
	socksUserPassSuccess = 0x00
	socksUserPassFailure = 0x01
)

func (s *Service) handleSOCKS(conn net.Conn) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	target, reply, err := readSOCKS5Target(conn, s.cfg.LocalAuth)
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
	s.proxy(conn, relay)
}

func readSOCKS5Target(rw io.ReadWriter, auth config.ClientAuthConfig) (string, byte, error) {
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

	if err := negotiateSOCKS5Auth(rw, methods, auth); err != nil {
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

func negotiateSOCKS5Auth(rw io.ReadWriter, methods []byte, auth config.ClientAuthConfig) error {
	if !localAuthEnabled(auth) {
		if !hasSOCKS5Method(methods, socksMethodNoAuth) {
			_, _ = rw.Write([]byte{socksVersion5, socksMethodNoAccept})
			return fmt.Errorf("client offered no supported auth method")
		}
		_, err := rw.Write([]byte{socksVersion5, socksMethodNoAuth})
		return err
	}

	if !hasSOCKS5Method(methods, socksMethodUserPass) {
		_, _ = rw.Write([]byte{socksVersion5, socksMethodNoAccept})
		return fmt.Errorf("client offered no username/password auth method")
	}
	if _, err := rw.Write([]byte{socksVersion5, socksMethodUserPass}); err != nil {
		return err
	}
	return readSOCKS5UserPassAuth(rw, auth)
}

func hasSOCKS5Method(methods []byte, want byte) bool {
	for _, method := range methods {
		if method == want {
			return true
		}
	}
	return false
}

func readSOCKS5UserPassAuth(rw io.ReadWriter, auth config.ClientAuthConfig) error {
	var header [2]byte
	if _, err := io.ReadFull(rw, header[:]); err != nil {
		return err
	}
	if header[0] != socksUserPassVersion {
		_, _ = rw.Write([]byte{socksUserPassVersion, socksUserPassFailure})
		return fmt.Errorf("unsupported socks username/password auth version %d", header[0])
	}

	username := make([]byte, int(header[1]))
	if _, err := io.ReadFull(rw, username); err != nil {
		return err
	}
	var passwordLen [1]byte
	if _, err := io.ReadFull(rw, passwordLen[:]); err != nil {
		return err
	}
	password := make([]byte, int(passwordLen[0]))
	if _, err := io.ReadFull(rw, password); err != nil {
		return err
	}

	if !checkLocalCredentials(auth, string(username), string(password)) {
		_, _ = rw.Write([]byte{socksUserPassVersion, socksUserPassFailure})
		return fmt.Errorf("socks username/password auth failed")
	}
	_, err := rw.Write([]byte{socksUserPassVersion, socksUserPassSuccess})
	return err
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
