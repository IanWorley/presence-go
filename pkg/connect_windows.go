//go:build windows
// +build windows

package pkg

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

type connectionWindows struct {
	conn net.Conn // active connection from Accept() or DialPipe()
	path string   // pipe path, e.g. \\.\pipe\mypipe
}

func (c *connectionWindows) Connect() error {
	var err error
	c.conn, err = winio.DialPipe(c.path, nil)
	if err != nil {
		return fmt.Errorf("error connecting to %s: %w", c.path, err)
	}
	return nil
}

func (c *connectionWindows) Disconnect() error {
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *connectionWindows) Receive() ([]byte, error) {
	frame, err := readFrame(c.conn)
	if err != nil {
		return nil, err
	}

	return frame, nil
}

func (c *connectionWindows) Send(data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}
	_, err := c.conn.Write(data)
	if err != nil {
		return fmt.Errorf("error sending data: %w", err)
	}
	return nil
}

func (c *connectionWindows) GetPath() string {

	return c.path
}

// Not a factory, just a helper function to create a new connectionWindows
func ConnectionFactory(path string) Connection {
	switch path {
	case "windows":
		return &connectionWindows{path: `\\.\pipe\discord-ipc-0`}
	default:
		return nil
	}
}
