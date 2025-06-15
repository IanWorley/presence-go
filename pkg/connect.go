package pkg

import (
	"fmt"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

type connectionUnixSocket struct {
	path string
	conn net.Conn
}

func (c *connectionUnixSocket) Connect() error {
	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return fmt.Errorf("error connecting to %s: %w", c.path, err)
	}
	c.conn = conn
	fmt.Println("Connected to", c.path)
	return nil
}

func (c *connectionUnixSocket) Disconnect() error {
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *connectionUnixSocket) Send(data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("connection not established")
	}
	_, err := c.conn.Write(data)
	if err != nil {
		return fmt.Errorf("error sending data: %w", err)
	}
	return nil
}

func (c *connectionUnixSocket) Receive() (string, error) {

	// TOOD: Get the length and abastract away logic of parsing request :)

	buf := make([]byte, 1024*1024)
	n, err := c.conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func (c *connectionUnixSocket) GetPath() string {
	return c.path
}

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

func (c *connectionWindows) Receive() (string, error) {

	buf := make([]byte, 1024*1024)
	n, err := c.conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
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

type Connection interface {
	Connect() error
	Disconnect() error
	Send(data []byte) error
	Receive() (string, error)
	GetPath() string
}

func ConnectionFactory(platform string) Connection {
	switch platform {
	case "darwin":
		return &connectionUnixSocket{path: os.Getenv("TMPDIR") + "/discord-ipc-0"}
	case "linux":
		return &connectionUnixSocket{path: os.Getenv("XDG_RUNTIME_DIR") + "/discord-ipc-0"}
	case "windows":
		return &connectionWindows{path: `\\.\pipe\discord-ipc-0`}
	default:
		return nil
	}

}
