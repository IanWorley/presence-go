package pkg

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

func readFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 8)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(header[4:8])
	body := make([]byte, length)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}
	if len(body) != int(length) {
		return nil, fmt.Errorf("expected %d bytes, got %d", length, len(body))
	}

	msg := make([]byte, length+8)
	copy(msg, header)
	copy(msg[8:], body)

	return msg, nil
}

type Connection interface {
	Connect() error
	Disconnect() error
	Send(data []byte) error
	Receive() ([]byte, error)
	GetPath() string
}
