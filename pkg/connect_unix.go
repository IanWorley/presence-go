//go:build darwin || linux
// +build darwin linux

package pkg

import (
	"fmt"
	"net"
	"os"
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

func (c *connectionUnixSocket) Receive() ([]byte, error) {
	frame, err := readFrame(c.conn)
	if err != nil {
		return nil, err
	}

	return frame, nil
}

func (c *connectionUnixSocket) GetPath() string {
	return c.path
}

func ConnectionFactory(platform string) Connection {
	switch platform {
	case "darwin":
		path, err := findDiscordIPC(os.Getenv("TMPDIR"))
		if err != nil {
			return nil
		}
		return &connectionUnixSocket{path: path}
	case "linux":
		path, err := findDiscordIPC(os.Getenv("XDG_RUNTIME_DIR"))
		if err != nil {
			return nil
		}
		return &connectionUnixSocket{path: path}
	default:
		return nil
	}
}

func findDiscordIPC(path string) (string, error) {
	for {
		for i := 0; i < 10; i++ {
			filePath := fmt.Sprintf("%s/discord-ipc-%d", path, i)
			conn, err := net.Dial("unix", filePath)
			if err != nil {
				continue
			}
			conn.Close()
			return filePath, nil
		}
		return "", fmt.Errorf("discord-ipc-0 not found")
	}

}
