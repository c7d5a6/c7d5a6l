package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
)

const (
	defaultBaseURL = "https://api.telegram.org"
	defaultTimeout = 30 * time.Second
	maxBodyBytes   = 1 << 20
)

// Client calls Telegram Bot API. Token is never logged.
type Client struct {
	HTTP     *http.Client
	BaseURL  string
	BotToken string
	GroupID  string
}

// New trims token and group id. Incomplete config is allowed; methods then return ErrNotConfigured.
func New(botToken, groupID string) *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: defaultTimeout,
		},
		BaseURL:  defaultBaseURL,
		BotToken: strings.TrimSpace(botToken),
		GroupID:  strings.TrimSpace(groupID),
	}
}

// Configured reports whether outbound group calls can run.
func (c *Client) Configured() bool {
	return c != nil && c.BotToken != "" && c.GroupID != ""
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: defaultTimeout}
}

func (c *Client) baseURL() string {
	if c != nil && c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultBaseURL
}

type apiEnvelope struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
	Result      json.RawMessage `json:"result"`
}

func (c *Client) call(ctx context.Context, method string, payload any, dest any) error {
	if c == nil || c.BotToken == "" {
		return ErrNotConfigured
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram %s: encode: %w", method, err)
	}

	endpoint := strings.TrimRight(c.baseURL(), "/") + "/bot" + c.BotToken + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	debuglog.Printf("telegram %s", method)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return fmt.Errorf("telegram %s: read: %w", method, err)
	}
	if len(raw) > maxBodyBytes {
		return fmt.Errorf("telegram %s: response too large", method)
	}

	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("telegram %s: http %d: invalid json", method, resp.StatusCode)
	}
	if !env.OK {
		return apiError(method, env)
	}
	if dest == nil || len(env.Result) == 0 || string(env.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Result, dest); err != nil {
		return fmt.Errorf("telegram %s: decode result: %w", method, err)
	}
	return nil
}

func apiError(method string, env apiEnvelope) error {
	desc := strings.TrimSpace(env.Description)
	if desc == "" {
		desc = "api error"
	}
	if env.ErrorCode != 0 {
		return fmt.Errorf("telegram %s: %s (%d)", method, desc, env.ErrorCode)
	}
	return fmt.Errorf("telegram %s: %s", method, desc)
}
