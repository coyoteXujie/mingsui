package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	Version        = 1
	MaxMessageSize = 64 * 1024
)

type ConnectRequest struct {
	Version int    `json:"version"`
	Token   string `json:"token"`
	Network string `json:"network"`
	Address string `json:"address"`
}

type ConnectResponse struct {
	Version int    `json:"version"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

func WriteJSON(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > MaxMessageSize {
		return fmt.Errorf("protocol message too large: %d bytes", len(data))
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if err := writeFull(w, header[:]); err != nil {
		return err
	}
	return writeFull(w, data)
}

func ReadJSON(r io.Reader, value any) error {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}

	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return fmt.Errorf("protocol message cannot be empty")
	}
	if size > MaxMessageSize {
		return fmt.Errorf("protocol message too large: %d bytes", size)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return err
	}
	return nil
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
