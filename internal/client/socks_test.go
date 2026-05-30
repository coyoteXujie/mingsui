package client

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

func TestReadSOCKS5TargetDomain(t *testing.T) {
	var req bytes.Buffer
	req.Write([]byte{socksVersion5, 1, socksMethodNoAuth})
	req.Write([]byte{socksVersion5, socksCommandConnect, 0, socksATYPDomain})
	req.WriteByte(byte(len("example.com")))
	req.WriteString("example.com")
	_ = binary.Write(&req, binary.BigEndian, uint16(443))

	rw := &recordingReadWriter{read: req.Bytes()}
	target, reply, err := readSOCKS5Target(rw)
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

	_, reply, err := readSOCKS5Target(rw)
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
