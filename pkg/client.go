package pkg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

type Client struct {
	ID       string
	Conn     Connection
	sendChan chan []byte
	done     chan struct{}
	ready    chan struct{}
}

type RPC interface {
	Send(message []string) error
	Connect() error
	Disconnect() error
	EventLoop()
}

func (c *Client) Connect() error {
	message := Frame{
		V:        1,
		ClientID: c.ID,
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
	<-c.ready
	close(c.ready)

	fmt.Println("handshake response", response)
	return nil
}

func (c *Client) eventLoop() error {

	var readyOnce sync.Once
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
				case uint32(OpPong):
					slog.Info("Pong received")
					readyOnce.Do(func() {
						c.ready <- struct{}{}
						slog.Info("Ready")
					})
				case uint32(OpFrame):
					slog.Info("Frame received")
					var status map[string]any
					body := []byte(msg)
					json.Unmarshal(body[8:], &status)
					slog.Info("Status", "status", status)
				case uint32(OpClose):
					slog.Info("Close received")
					c.Disconnect()
					return
				}

			}

		}
	}()

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case <-ticker.C:
				pingMsg, err := c.buildMessage(Frame{V: 1, ClientID: c.ID, Cmd: "PING"}, OpPing)
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
	c.done <- struct{}{}
	return c.Conn.Disconnect()
}

func (c *Client) buildMessage(frame Frame, opcode Opcode) ([]byte, error) {

	defaultJSON, err := json.Marshal(frame)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int32(opcode))
	binary.Write(&buf, binary.LittleEndian, int32(len(defaultJSON)))
	buf.Write(defaultJSON)

	return buf.Bytes(), nil
}

func (c *Client) Send(frame Frame) error {
	completedMessage, err := c.buildMessage(frame, OpFrame)
	if err != nil {
		return err
	}

	c.sendChan <- completedMessage
	return nil
}

func NewClient(id string) *Client {
	conn := ConnectionFactory(runtime.GOOS)
	return &Client{ID: id, Conn: conn, sendChan: make(chan []byte), done: make(chan struct{}), ready: make(chan struct{})}
}

type Opcode uint32

const (
	OpHandshake Opcode = 0
	OpFrame     Opcode = 1
	OpClose     Opcode = 2
	OpPing      Opcode = 3
	OpPong      Opcode = 4
)

type Frame struct {
	V        int    `json:"v"`
	ClientID string `json:"client_id"`
	Cmd      string `json:"cmd,omitempty"`
	Args     Args   `json:"args,omitempty"`
	Nonce    string `json:"nonce,omitempty"`
}

type Args struct {
	PID      int       `json:"pid,omitempty"`
	Activity *Activity `json:"activity,omitempty"`
}

type Activity struct {
	Details    string      `json:"details,omitempty"`
	State      string      `json:"state,omitempty"`
	Assets     *Assets     `json:"assets,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Party      *Party      `json:"party,omitempty"`
	Buttons    *[]Button   `json:"buttons,omitempty"`
}

type Timestamps struct {
	Start int `json:"start,omitempty"`
	End   int `json:"end,omitempty"`
}

type Party struct {
	ID   string `json:"id,omitempty"`
	Size [2]int `json:"size,omitempty"` // corrected
}

type Button struct {
	Label string `json:"label,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}
