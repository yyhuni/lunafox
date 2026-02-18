package websocket

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"go.uber.org/zap"
)

const (
	defaultPingInterval = 30 * time.Second
	defaultPongWait     = 60 * time.Second
	defaultWriteWait    = 10 * time.Second
)

// Client maintains a WebSocket connection to the server.
type Client struct {
	wsURL        string
	apiKey       string
	dialer       *websocket.Dialer
	send         chan []byte
	onMessage    func([]byte)
	backoff      Backoff
	pingInterval time.Duration
	pongWait     time.Duration
	writeWait    time.Duration
}

// NewClient creates a WebSocket client for the agent.
func NewClient(wsURL, apiKey string) *Client {
	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return &Client{
		wsURL:        wsURL,
		apiKey:       apiKey,
		dialer:       &dialer,
		send:         make(chan []byte, 256),
		backoff:      NewBackoff(1*time.Second, 60*time.Second),
		pingInterval: defaultPingInterval,
		pongWait:     defaultPongWait,
		writeWait:    defaultWriteWait,
	}
}

// SetOnMessage registers a callback for incoming messages.
func (c *Client) SetOnMessage(fn func([]byte)) {
	c.onMessage = fn
}

// Send queues a message for sending. It returns false if the buffer is full.
func (c *Client) Send(payload []byte) bool {
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

// Run keeps the connection alive with reconnect backoff and keepalive pings.
func (c *Client) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		logger.Log.Info("websocket connect attempt", zap.String("url", c.wsURL))
		conn, err := c.connect(ctx)
		if err != nil {
			logger.Log.Warn("websocket connect failed", zap.Error(err))
			if !sleepWithContext(ctx, c.backoff.Next()) {
				return ctx.Err()
			}
			continue
		}

		c.backoff.Reset()
		logger.Log.Info("websocket connected")
		err = c.runConn(ctx, conn)
		if err != nil && ctx.Err() == nil {
			logger.Log.Warn("websocket connection closed", zap.Error(err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !sleepWithContext(ctx, c.backoff.Next()) {
			return ctx.Err()
		}
	}
}

func (c *Client) connect(ctx context.Context) (*websocket.Conn, error) {
	header := http.Header{}
	if c.apiKey != "" {
		header.Set("X-Agent-Key", c.apiKey)
	}
	conn, _, err := c.dialer.DialContext(ctx, c.wsURL, header)
	return conn, err
}

func (c *Client) runConn(ctx context.Context, conn *websocket.Conn) error {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(c.pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})

	errCh := make(chan error, 2)
	go c.readLoop(conn, errCh)
	go c.writeLoop(ctx, conn, errCh)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (c *Client) readLoop(conn *websocket.Conn, errCh chan<- error) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}
		if c.onMessage != nil {
			c.onMessage(message)
		}
	}
}

func (c *Client) writeLoop(ctx context.Context, conn *websocket.Conn, errCh chan<- error) {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		case payload := <-c.send:
			if err := c.writeMessage(conn, websocket.TextMessage, payload); err != nil {
				errCh <- err
				return
			}
		case <-ticker.C:
			if err := c.writeMessage(conn, websocket.PingMessage, nil); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (c *Client) writeMessage(conn *websocket.Conn, msgType int, payload []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	return conn.WriteMessage(msgType, payload)
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
