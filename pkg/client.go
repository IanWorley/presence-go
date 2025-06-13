package pkg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type Client struct {
	ID       string
	Conn     Connection
	sendChan chan []byte
	done     chan struct{}
	activity map[string]any
}

type RPC interface {
	Send(message []string) error
	Connect() error
	Disconnect() error
	EventLoop() error
}

func (c *Client) Connect() error {

	// handShake

	message := map[string]any{
		"v":         1,
		"client_id": c.ID,
	}

	handshakeMessage, err := c.buildMessage(message, OpHandshake)
	if err != nil {
		return err
	}

	err = c.Conn.Connect()
	if err != nil {
		return err
	}

	err = c.Conn.Send(handshakeMessage)
	if err != nil {
		return err
	}

	response, err := c.Conn.Receive()
	if err != nil {
		return err
	}

	go c.eventLoop()

	fmt.Println("handshake response", response)
	return nil
}

func (c *Client) eventLoop() error {

	c.sendChan = make(chan []byte)
	c.done = make(chan struct{})

	// Receiver goroutine
	go func() {
		for {
			select {
			case <-c.done:
				return
			default:
				msg, err := c.Conn.Receive()
				if err != nil {
					// handle error, maybe close c.done
					return
				}
				opcode := binary.LittleEndian.Uint32([]byte(msg))
				switch opcode {
				case OpPong:
					slog.Info("Pong received")

				case OpFrame:
					slog.Info("Frame received")
					json.Unmarshal([]byte(msg), &c.activity)
					slog.Info("Activity", "activity", c.activity)
				case OpClose:
					slog.Info("Close received")
					c.Disconnect()
					return
				}

			}

		}
	}()

	// Sender and ping goroutine
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case <-ticker.C:
				pingMsg, err := c.buildMessage(map[string]any{}, OpPing)
				if err != nil {
					return
				}
				slog.Info("Ping sent")
				c.Conn.Send(pingMsg)
			case msg := <-c.sendChan:
				err := c.Conn.Send(msg)
				if err != nil {
					slog.Error("Error sending message", "error", err)
				}
			}
		}
	}()

	return nil
}

func (c *Client) Disconnect() error {
	return c.Conn.Disconnect()
}

func (c *Client) buildMessage(message map[string]any, opcode uint32) ([]byte, error) {

	defaultJSON, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int32(opcode))
	binary.Write(&buf, binary.LittleEndian, int32(len(defaultJSON)))
	buf.Write(defaultJSON)

	return buf.Bytes(), nil
}

func (c *Client) Send(message map[string]any) error {
	// Set default command if not provided
	if _, ok := message["cmd"]; !ok {
		message["cmd"] = "SET_ACTIVITY"
	}

	// Set default args
	defaultArgs := map[string]any{
		"pid": os.Getpid(),
		"activity": map[string]any{
			"details": "Testing",
			"state":   "Testing",
		},
	}

	// Merge user-provided args with defaults
	if userArgs, ok := message["args"].(map[string]any); ok {
		for k, v := range userArgs {
			defaultArgs[k] = v
		}
		message["args"] = defaultArgs
	} else {
		message["args"] = defaultArgs
	}

	completedMessage, err := c.buildMessage(message, OpFrame)
	if err != nil {
		return err
	}

	c.sendChan <- completedMessage
	return nil
}

func NewClient(id string) *Client {
	conn := ConnectionFactory(runtime.GOOS)
	return &Client{ID: id, Conn: conn, sendChan: make(chan []byte), done: make(chan struct{})}
}

const (
	OpHandshake = 0
	OpFrame     = 1
	OpClose     = 2
	OpPing      = 3
	OpPong      = 4
)
