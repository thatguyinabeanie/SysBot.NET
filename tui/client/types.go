// Package client provides API types and HTTP/WebSocket client for the SysBot.NET web API.
package client

import "time"

// --- Bot types ---

// BotDto represents a single bot as returned by the /api/bots endpoint.
type BotDto struct {
	ID             string    `json:"id"`
	IP             string    `json:"ip"`
	Port           int       `json:"port"`
	Protocol       string    `json:"protocol"`       // "WiFi" or "USB"
	InitialRoutine string    `json:"initialRoutine"`  // e.g. "FlexTrade"
	CurrentRoutine string    `json:"currentRoutine"`  // e.g. "Idle"
	NextRoutine    string    `json:"nextRoutine"`     // e.g. "FlexTrade"
	IsRunning      bool      `json:"isRunning"`
	IsPaused       bool      `json:"isPaused"`
	IsConnected    bool      `json:"isConnected"`
	LastLog        string    `json:"lastLog"`         // most recent log message
	LastActive     time.Time `json:"lastActive"`
}

// AddBotRequest is the payload for POST /api/bots to add a new bot.
type AddBotRequest struct {
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`       // "WiFi" or "USB"
	InitialRoutine string `json:"initialRoutine"`
}

// --- Meta types ---

// MetaInfo is returned by GET /api/meta and describes the hub's current mode
// and capabilities.
type MetaInfo struct {
	Mode              string   `json:"mode"`              // e.g. "LZA", "SV", "SWSH"
	SupportedRoutines []string `json:"supportedRoutines"` // available routine types
	Protocols         []string `json:"protocols"`         // available connection protocols
	IsRunning         bool     `json:"isRunning"`         // whether the hub is running
}

// --- Queue types ---

// QueueCount holds the count for a single queue category.
type QueueCount struct {
	Count int `json:"count"`
}

// QueueStatus is returned by GET /api/queues and shows the state of all
// trade queues.
type QueueStatus struct {
	CanQueue   bool                    `json:"canQueue"`
	Queues     map[string]*QueueCount  `json:"queues"`     // keyed by queue name (trade, seedCheck, clone, dump)
	TotalCount int                     `json:"totalCount"`
}

// --- Config / schema types ---

// SchemaProperty describes a single configuration property from the
// GET /api/config/schema endpoint.
type SchemaProperty struct {
	Description string                     `json:"description,omitempty"`
	Type        string                     `json:"type"`                   // "boolean", "string", "integer", "object", "enum", etc.
	Value       any                        `json:"value,omitempty"`        // current value
	EnumValues  []string                   `json:"enumValues,omitempty"`   // allowed values when Type is "enum"
	Properties  map[string]*SchemaProperty `json:"properties,omitempty"`   // nested properties for type "object"
}

// ConfigSchema is the top-level response from GET /api/config/schema.
// Categories is a nested map: category -> subcategory -> SchemaProperty.
type ConfigSchema struct {
	Categories map[string]map[string]*SchemaProperty `json:"categories"`
}

// --- SignalR / WebSocket types ---

// LogEntry represents a structured log line from SignalR streaming.
// Fields are strings because they arrive as string arguments from SignalR.
type LogEntry struct {
	Timestamp string
	Identity  string
	Message   string
}
