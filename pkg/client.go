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
	ID          string
	Conn        Connection
	sendChan    chan []byte
	done        chan struct{}
	ready       chan struct{}
	errChan     chan error
	receiveChan chan []byte
	OnError     func(error)
	OnReady     func()
	OnReceive   func([]byte)
	OnClose     func()
	closeOnce   sync.Once
}

type RPC interface {
	Send(message []string) error
	Connect() error
	Disconnect() error
	EventLoop()
}

func (c *Client) Connect() error {
	message := Status{
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

	fmt.Println("handshake sent")

	_, err = c.Conn.Receive()
	if err != nil {
		return err
	}

	fmt.Println("handshake received")

	go c.eventLoop()

	<-c.ready

	go c.errorHandler()
	go c.readyHandler()
	go c.receiveHandler()
	go c.closeHandler()

	close(c.ready)

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
				fmt.Println("msg", msg)
				if err != nil {
					c.errChan <- err
					c.Disconnect()

					return
				}
				opcode := binary.LittleEndian.Uint32([]byte(msg))
				switch opcode {
				case uint32(OpPong):
					fmt.Println("pong received")
					readyOnce.Do(func() {
						c.ready <- struct{}{}
					})
				case uint32(OpFrame):
					var status Status
					body := []byte(msg)
					json.Unmarshal(body[8:], &status)
				case uint32(OpClose):
					c.Disconnect()
					c = recreateClient(c)
					return
				}

			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case <-ticker.C:
				pingMsg, err := c.buildMessage(Status{V: 1, ClientID: c.ID, Cmd: "PING"}, OpPing)
				if err != nil {
					c.errChan <- err
					return
				}
				c.Conn.Send(pingMsg)
			case msg := <-c.sendChan:
				err := c.Conn.Send(msg)
				if err != nil {
					c.errChan <- err
					return
				}
			default:
				continue
			}

		}
	}()

	return nil
}

func (c *Client) Disconnect() error {
	c.closeOnce.Do(func() {
		close(c.sendChan)
		close(c.receiveChan)
		<-c.errChan
		close(c.errChan)

		c.done <- struct{}{}
	})
	return c.Conn.Disconnect()
}

func (c *Client) buildMessage(frame Status, opcode Opcode) ([]byte, error) {

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

func (c *Client) Send(frame Status) error {
	completedMessage, err := c.buildMessage(frame, OpFrame)
	if err != nil {
		return err
	}

	c.sendChan <- completedMessage
	return nil
}

func (c *Client) errorHandler() {
	for err := range c.errChan {
		if c.OnError != nil {
			c.OnError(err)
		} else {
			slog.Error("Client error", "error", err)
		}
	}
}

func (c *Client) readyHandler() {
	for range c.ready {
		if c.OnReady != nil {
			c.OnReady()
		}
	}
}

func (c *Client) receiveHandler() {
	for {
		msg := <-c.receiveChan

		if c.OnReceive != nil {
			c.OnReceive(msg)
		}
	}
}

func (c *Client) closeHandler() {
	for range c.done {
		fmt.Println("closeHandler")
		if c.OnClose != nil {
			c.OnClose()
		}
	}
}

func recreateClient(c *Client) *Client {
	conn := ConnectionFactory(runtime.GOOS)
	return &Client{ID: c.ID, Conn: conn, sendChan: make(chan []byte), done: make(chan struct{}), ready: make(chan struct{}), errChan: make(chan error), receiveChan: make(chan []byte), OnError: c.OnError, OnReady: c.OnReady, OnReceive: c.OnReceive, OnClose: c.OnClose}
}

func NewClient(id string) *Client {
	conn := ConnectionFactory(runtime.GOOS)
	return &Client{ID: id, Conn: conn, sendChan: make(chan []byte), done: make(chan struct{}), ready: make(chan struct{}), errChan: make(chan error), receiveChan: make(chan []byte), OnError: nil, OnReady: nil, OnReceive: nil, OnClose: nil}
}

type Opcode uint32

const (
	OpHandshake Opcode = 0
	OpFrame     Opcode = 1
	OpClose     Opcode = 2
	OpPing      Opcode = 3
	OpPong      Opcode = 4
)

type Status struct {
	V        int    `json:"v"`
	ClientID string `json:"client_id"`
	Cmd      string `json:"cmd,omitempty"`
	Args     Args   `json:"args,omitempty"`
	Nonce    string `json:"nonce,omitempty"` // TODO: make this a uuid
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
