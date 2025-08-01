package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DebugClient provides access to the bot's debug API
type DebugClient struct {
	baseURL string
	client  *http.Client
}

// NewDebugClient creates a new debug API client
func NewDebugClient(port int) *DebugClient {
	return &DebugClient{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckConnection verifies the debug API is accessible
func (c *DebugClient) CheckConnection() error {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("debug API not accessible: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("debug API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// ReplayMessage sends a replay command to the bot
func (c *DebugClient) ReplayMessage(channelID, messageID string) error {
	cmd := map[string]interface{}{
		"action": "replay",
		"params": map[string]string{
			"channel_id": channelID,
			"message_id": messageID,
		},
	}
	
	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf(resp.Error)
	}
	
	return nil
}

// GetGuilds fetches the list of guilds from the bot
func (c *DebugClient) GetGuilds() ([]GuildInfo, error) {
	resp, err := c.client.Get(c.baseURL + "/debug/guilds")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guilds: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var debugResp struct {
		Success bool        `json:"success"`
		Error   string      `json:"error,omitempty"`
		Data    []GuildInfo `json:"data,omitempty"`
	}
	
	if err := json.Unmarshal(respBody, &debugResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !debugResp.Success {
		return nil, fmt.Errorf("failed to get guilds: %s", debugResp.Error)
	}
	
	return debugResp.Data, nil
}

// DebugResponse represents the API response
type DebugResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// GuildInfo represents basic guild information
type GuildInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// sendCommand sends a command to the debug API
func (c *DebugClient) sendCommand(cmd interface{}) (*DebugResponse, error) {
	body, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}
	
	resp, err := c.client.Post(c.baseURL+"/debug/command", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var debugResp DebugResponse
	if err := json.Unmarshal(respBody, &debugResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &debugResp, nil
}