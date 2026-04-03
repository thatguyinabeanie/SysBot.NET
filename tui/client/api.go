// Package client provides a REST API client for the SysBot.Pokemon.Web backend.
// It wraps every endpoint exposed by the BotController, StatusController, and
// ConfigController into typed Go methods.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client holds the base URL and underlying HTTP client used to talk to the
// SysBot.Pokemon.Web REST API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a Client pointed at the given base URL (e.g. "http://localhost:5000").
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// ── Private helpers ────────────────────────────────────────────────────

// get performs a GET request to the given path and JSON-decodes the response
// into result.
func (c *Client) get(path string, result any) error {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readErrorResponse(resp)
	}

	// Decode the JSON body into the caller's target.
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("GET %s: decode: %w", path, err)
		}
	}
	return nil
}

// post performs a POST request with a JSON body and decodes the response.
// body may be nil for POST requests with no payload.
func (c *Client) post(path string, body any, result any) error {
	var payload io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("POST %s: marshal: %w", path, err)
		}
		payload = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, payload)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readErrorResponse(resp)
	}

	// Decode if the caller expects a result.
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("POST %s: decode: %w", path, err)
		}
	}
	return nil
}

// delete performs a DELETE request. It does not attempt to decode a body.
func (c *Client) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readErrorResponse(resp)
	}
	return nil
}

// patch performs a PATCH request with a JSON body and decodes the response.
func (c *Client) patch(path string, body any, result any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("PATCH %s: marshal: %w", path, err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("PATCH %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("PATCH %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readErrorResponse(resp)
	}

	// Decode if the caller expects a result.
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("PATCH %s: decode: %w", path, err)
		}
	}
	return nil
}

// readErrorResponse reads the response body and returns a formatted error
// including the HTTP status code and server message.
func readErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// ── Public API methods ─────────────────────────────────────────────────

// ListBots returns every bot registered with the runner.
// Calls GET /api/bots.
func (c *Client) ListBots() ([]BotDto, error) {
	var bots []BotDto
	err := c.get("/api/bots", &bots)
	return bots, err
}

// AddBot registers a new bot from connection details.
// Calls POST /api/bots.
func (c *Client) AddBot(req AddBotRequest) (*BotDto, error) {
	var bot BotDto
	err := c.post("/api/bots", req, &bot)
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

// RemoveBot removes a bot by its connection-name ID.
// Calls DELETE /api/bots/{id}.
func (c *Client) RemoveBot(id string) error {
	path := "/api/bots/" + url.PathEscape(id)
	return c.delete(path)
}

// BotAction performs a lifecycle action (start, stop, pause, resume, restart)
// on the bot identified by id.
// Calls POST /api/bots/{id}/{action}.
func (c *Client) BotAction(id, action string) (*BotDto, error) {
	path := "/api/bots/" + url.PathEscape(id) + "/" + url.PathEscape(action)
	var bot BotDto
	err := c.post(path, nil, &bot)
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

// StartAll starts every registered bot.
// Calls POST /api/bots/start-all.
func (c *Client) StartAll() error {
	return c.post("/api/bots/start-all", nil, nil)
}

// StopAll stops every registered bot.
// Calls POST /api/bots/stop-all.
func (c *Client) StopAll() error {
	return c.post("/api/bots/stop-all", nil, nil)
}

// GetMeta returns high-level metadata about the running instance: game mode,
// supported routines, available protocols, and running state.
// Calls GET /api/meta.
func (c *Client) GetMeta() (*MetaInfo, error) {
	var meta MetaInfo
	err := c.get("/api/meta", &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// GetQueues returns the current queue depths for each trade routine.
// Calls GET /api/queues.
func (c *Client) GetQueues() (*QueueStatus, error) {
	var qs QueueStatus
	err := c.get("/api/queues", &qs)
	if err != nil {
		return nil, err
	}
	return &qs, nil
}

// GetHubConfig returns the full PokeTradeHubConfig as a free-form map.
// Calls GET /api/config/hub.
func (c *Client) GetHubConfig() (map[string]any, error) {
	var cfg map[string]any
	err := c.get("/api/config/hub", &cfg)
	return cfg, err
}

// PatchHubConfig applies a partial update to the PokeTradeHubConfig.
// Calls PATCH /api/config/hub.
func (c *Client) PatchHubConfig(patch map[string]any) error {
	return c.patch("/api/config/hub", patch, nil)
}

// GetConfigSchema returns the introspected JSON schema for PokeTradeHubConfig,
// grouped by category with type info, descriptions, and enum values.
// Calls GET /api/config/hub/schema.
func (c *Client) GetConfigSchema() (*ConfigSchema, error) {
	var schema ConfigSchema
	err := c.get("/api/config/hub/schema", &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}
