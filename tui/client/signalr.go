package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// recordSeparator is the SignalR JSON protocol message terminator (0x1E).
	recordSeparator = "\x1e"

	// pingInterval is how often we send a keep-alive ping to the server.
	pingInterval = 15 * time.Second
)

// negotiateResponse holds the fields returned by the SignalR negotiate endpoint.
type negotiateResponse struct {
	ConnectionID    string `json:"connectionId"`
	ConnectionToken string `json:"connectionToken"`
}

// signalRMessage is a local type for parsing SignalR frames with json.RawMessage arguments.
type signalRMessage struct {
	Type      int               `json:"type"`
	Target    string            `json:"target,omitempty"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
}

// SignalRClient manages a WebSocket connection to a SignalR hub and dispatches
// incoming messages to registered callbacks.
type SignalRClient struct {
	conn   *websocket.Conn
	OnLog  func(entry LogEntry)
	OnEcho func(timestamp, message string)
	OnBot  func(bot BotDto)
	done   chan struct{}
}

// NewSignalRClient negotiates a connection with the SignalR hub at baseURL,
// opens a WebSocket, and performs the JSON protocol handshake.
//
// baseURL should look like "http://localhost:5000" (no trailing slash).
func NewSignalRClient(baseURL string) (*SignalRClient, error) {
	// --- Step 1: Negotiate to get a connection ID / token ---
	negotiateURL := strings.TrimRight(baseURL, "/") + "/hubs/logs/negotiate?negotiateVersion=1"

	resp, err := http.Post(negotiateURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("negotiate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("negotiate returned status %d", resp.StatusCode)
	}

	var neg negotiateResponse
	if err := json.NewDecoder(resp.Body).Decode(&neg); err != nil {
		return nil, fmt.Errorf("failed to decode negotiate response: %w", err)
	}

	// Use connectionToken if present (newer SignalR), fall back to connectionId.
	connID := neg.ConnectionToken
	if connID == "" {
		connID = neg.ConnectionID
	}
	if connID == "" {
		return nil, fmt.Errorf("negotiate response contained no connectionId or connectionToken")
	}

	// --- Step 2: Build the WebSocket URL ---
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Replace http(s) scheme with ws(s).
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	default:
		parsed.Scheme = "ws"
	}
	parsed.Path = "/hubs/logs"
	q := parsed.Query()
	q.Set("id", connID)
	parsed.RawQuery = q.Encode()

	wsURL := parsed.String()

	// --- Step 3: Open WebSocket connection ---
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	// --- Step 4: Send the JSON protocol handshake ---
	hsBytes, err := json.Marshal(struct {
		Protocol string `json:"protocol"`
		Version  int    `json:"version"`
	}{"json", 1})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to marshal handshake: %w", err)
	}

	// Handshake message must be terminated by the record separator.
	if err := conn.WriteMessage(websocket.TextMessage, append(hsBytes, 0x1E)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	client := &SignalRClient{
		conn: conn,
		done: make(chan struct{}),
	}

	// Start the background ping loop to keep the connection alive.
	go client.pingLoop()

	return client, nil
}

// Listen reads frames from the WebSocket, splits them by the record separator,
// and dispatches each SignalR message. This method blocks until the connection
// is closed or an error occurs.
func (c *SignalRClient) Listen() {
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			// If the done channel is closed, this is an expected shutdown.
			select {
			case <-c.done:
				return
			default:
			}
			// Unexpected read error — stop listening.
			return
		}

		// A single WebSocket frame can contain multiple SignalR messages
		// separated by 0x1E.
		parts := strings.Split(string(raw), recordSeparator)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			var msg signalRMessage
			if err := json.Unmarshal([]byte(part), &msg); err != nil {
				// Skip malformed messages.
				continue
			}

			switch msg.Type {
			case 1:
				// Invocation message — dispatch based on target.
				c.handleInvocation(msg)
			case 6:
				// Ping — no action needed, the server is just keeping us alive.
			}
		}
	}
}

// Close shuts down the ping loop and closes the WebSocket connection.
func (c *SignalRClient) Close() {
	// Signal the ping loop to stop.
	select {
	case <-c.done:
		// Already closed.
	default:
		close(c.done)
	}

	// Send a graceful close frame, then close the connection.
	_ = c.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	_ = c.conn.Close()
}

// pingLoop sends a SignalR ping message every 15 seconds to keep the
// connection alive. It stops when the done channel is closed.
func (c *SignalRClient) pingLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	// Ping message: {"type":6} followed by the record separator.
	pingMsg := []byte(`{"type":6}` + recordSeparator)

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.TextMessage, pingMsg); err != nil {
				// Write failed — connection is likely dead. Stop pinging.
				return
			}
		}
	}
}

// handleInvocation dispatches an invocation message (type=1) to the
// appropriate callback based on its Target field.
func (c *SignalRClient) handleInvocation(msg signalRMessage) {
	switch msg.Target {
	case "ReceiveLog":
		// Expected arguments: [timestamp, identity, message]
		if c.OnLog == nil || len(msg.Arguments) < 3 {
			return
		}

		var timestamp, identity, message string
		if err := json.Unmarshal(msg.Arguments[0], &timestamp); err != nil {
			return
		}
		if err := json.Unmarshal(msg.Arguments[1], &identity); err != nil {
			return
		}
		if err := json.Unmarshal(msg.Arguments[2], &message); err != nil {
			return
		}

		c.OnLog(LogEntry{
			Timestamp: timestamp,
			Identity:  identity,
			Message:   message,
		})

	case "ReceiveEcho":
		// Expected arguments: [timestamp, message]
		if c.OnEcho == nil || len(msg.Arguments) < 2 {
			return
		}

		var timestamp, message string
		if err := json.Unmarshal(msg.Arguments[0], &timestamp); err != nil {
			return
		}
		if err := json.Unmarshal(msg.Arguments[1], &message); err != nil {
			return
		}

		c.OnEcho(timestamp, message)

	case "BotStatusChanged":
		// Expected arguments: [BotDto as JSON object]
		if c.OnBot == nil || len(msg.Arguments) < 1 {
			return
		}

		var bot BotDto
		if err := json.Unmarshal(msg.Arguments[0], &bot); err != nil {
			return
		}

		c.OnBot(bot)
	}
}
