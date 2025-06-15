//go:build windows
// +build windows

package pkg

import (
	"fmt"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

type connectionWindows struct {
	conn net.Conn
	path string
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
func ConnectionFactory(platform string) Connection {
	switch platform {
	case "windows":
		path, err := findDiscordIPC(os.Getenv("TMPDIR"))
		if err != nil {
			return nil
		}
		return &connectionWindows{path: path}
	default:
		return nil
	}
}

func findDiscordIPC(path string) (string, error) {
	for {
		for i := 0; i < 10; i++ {
			filePath := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
			conn, err := winio.DialPipe(filePath, nil)
			if err != nil {
				continue
			}
			conn.Close()
			return filePath, nil
		}
		return "", fmt.Errorf("discord-ipc-0 not found")
	}
}
