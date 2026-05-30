package client

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestReadSOCKS5TargetDomain(t *testing.T) {
	var req bytes.Buffer
	req.Write([]byte{socksVersion5, 1, socksMethodNoAuth})
	req.Write([]byte{socksVersion5, socksCommandConnect, 0, socksATYPDomain})
	req.WriteByte(byte(len("example.com")))
	req.WriteString("example.com")
	_ = binary.Write(&req, binary.BigEndian, uint16(443))

	rw := &recordingReadWriter{read: req.Bytes()}
	target, reply, err := readSOCKS5Target(rw, config.ClientAuthConfig{})
	if err != nil {
		t.Fatalf("readSOCKS5Target() error = %v", err)
	}
	if target != "example.com:443" {
		t.Fatalf("target = %q, want example.com:443", target)
	}
	if reply != socksReplyGeneralFailure {
		t.Fatalf("reply = %d, want default failure reply for later errors", reply)
	}
	if got := rw.written.Bytes(); !bytes.Equal(got, []byte{socksVersion5, socksMethodNoAuth}) {
		t.Fatalf("method response = %v, want no-auth response", got)
	}
}

func TestReadSOCKS5TargetRejectsUnsupportedCommand(t *testing.T) {
	req := []byte{
		socksVersion5, 1, socksMethodNoAuth,
		socksVersion5, 0x02, 0, socksATYPDomain,
	}
	rw := &recordingReadWriter{read: req}

	_, reply, err := readSOCKS5Target(rw, config.ClientAuthConfig{})
	if err == nil {
		t.Fatal("readSOCKS5Target() error = nil, want unsupported command")
	}
	if reply != socksReplyCommandNotSup {
		t.Fatalf("reply = %d, want command-not-supported", reply)
	}
	if !strings.Contains(err.Error(), "unsupported socks command") {
		t.Fatalf("error = %v, want unsupported command", err)
	}
}

func TestReadSOCKS5TargetWithUserPassAuth(t *testing.T) {
	var req bytes.Buffer
	req.Write([]byte{socksVersion5, 1, socksMethodUserPass})
	req.Write([]byte{socksUserPassVersion, byte(len("user"))})
	req.WriteString("user")
	req.WriteByte(byte(len("pass")))
	req.WriteString("pass")
	req.Write([]byte{socksVersion5, socksCommandConnect, 0, socksATYPDomain})
	req.WriteByte(byte(len("example.com")))
	req.WriteString("example.com")
	_ = binary.Write(&req, binary.BigEndian, uint16(443))

	auth := config.ClientAuthConfig{Enabled: true, Username: "user", Password: "pass"}
	rw := &recordingReadWriter{read: req.Bytes()}
	target, _, err := readSOCKS5Target(rw, auth)
	if err != nil {
		t.Fatalf("readSOCKS5Target() error = %v", err)
	}
	if target != "example.com:443" {
		t.Fatalf("target = %q, want example.com:443", target)
	}
	wantPrefix := []byte{
		socksVersion5, socksMethodUserPass,
		socksUserPassVersion, socksUserPassSuccess,
	}
	if got := rw.written.Bytes(); !bytes.Equal(got, wantPrefix) {
		t.Fatalf("auth responses = %v, want %v", got, wantPrefix)
	}
}

func TestReadSOCKS5TargetRejectsBadUserPassAuth(t *testing.T) {
	var req bytes.Buffer
	req.Write([]byte{socksVersion5, 1, socksMethodUserPass})
	req.Write([]byte{socksUserPassVersion, byte(len("user"))})
	req.WriteString("user")
	req.WriteByte(byte(len("bad")))
	req.WriteString("bad")

	auth := config.ClientAuthConfig{Enabled: true, Username: "user", Password: "pass"}
	rw := &recordingReadWriter{read: req.Bytes()}
	_, _, err := readSOCKS5Target(rw, auth)
	if err == nil {
		t.Fatal("readSOCKS5Target() error = nil, want auth failure")
	}
	want := []byte{
		socksVersion5, socksMethodUserPass,
		socksUserPassVersion, socksUserPassFailure,
	}
	if got := rw.written.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("auth responses = %v, want %v", got, want)
	}
}

type recordingReadWriter struct {
	read    []byte
	written bytes.Buffer
}

func (rw *recordingReadWriter) Read(p []byte) (int, error) {
	if len(rw.read) == 0 {
		return 0, io.EOF
	}
	n := copy(p, rw.read)
	rw.read = rw.read[n:]
	return n, nil
}

func (rw *recordingReadWriter) Write(p []byte) (int, error) {
	return rw.written.Write(p)
}
