package wsbus

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const defaultSendBuffer = 16

// Client is one WebSocket connection for a citizen.
type Client struct {
	hub       *Hub
	citizenID uuid.UUID
	conn      *websocket.Conn
	send      chan []byte

	closeOnce sync.Once

	writeTimeout time.Duration
	pingInterval time.Duration
	pongWait     time.Duration
	readLimit    int64
}

// NewClient builds a client; caller must Register then Run.
func NewClient(hub *Hub, citizenID uuid.UUID, conn *websocket.Conn, writeTimeout, pingInterval, pongWait time.Duration, readLimit int64) *Client {
	if writeTimeout <= 0 {
		writeTimeout = 10 * time.Second
	}
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	if pongWait <= 0 {
		pongWait = 60 * time.Second
	}
	if readLimit <= 0 {
		readLimit = 1 << 20
	}
	return &Client{
		hub:          hub,
		citizenID:    citizenID,
		conn:         conn,
		send:         make(chan []byte, defaultSendBuffer),
		writeTimeout: writeTimeout,
		pingInterval: pingInterval,
		pongWait:     pongWait,
		readLimit:    readLimit,
	}
}

func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		c.hub.Unregister(c)
		close(c.send)
		_ = c.conn.Close()
	})
}

// Run starts read and write pumps; blocks until the connection ends.
func (c *Client) Run() {
	c.hub.Register(c)
	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer c.shutdown()
	c.conn.SetReadLimit(c.readLimit)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()
	defer c.shutdown()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(c.writeTimeout)); err != nil {
				return
			}
		}
	}
}
