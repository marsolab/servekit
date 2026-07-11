package slackkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL       = "https://slack.com"
	maxResponseBodyBytes = 4096
)

// HTTPOption configures Slack HTTP clients.
type HTTPOption func(*httpConfig)

type httpConfig struct {
	httpClient *http.Client
	baseURL    string
}

// WithHTTPClient sets the HTTP client used for Slack requests.
func WithHTTPClient(httpClient *http.Client) HTTPOption {
	return func(config *httpConfig) {
		config.httpClient = httpClient
	}
}

// WithBaseURL sets the Slack Web API base URL.
func WithBaseURL(baseURL string) HTTPOption {
	return func(config *httpConfig) {
		config.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WebhookSender sends Slack messages through an incoming webhook URL.
type WebhookSender struct {
	webhookURL string
	httpClient *http.Client
}

// NewWebhookSender creates a Slack incoming webhook sender.
func NewWebhookSender(webhookURL string, options ...HTTPOption) *WebhookSender {
	config := newHTTPConfig(options...)

	return &WebhookSender{
		webhookURL: webhookURL,
		httpClient: config.httpClient,
	}
}

// Send sends a notification to a Slack incoming webhook.
func (s *WebhookSender) Send(ctx context.Context, message Notification) error {
	if err := message.validate(); err != nil {
		return fmt.Errorf("slack webhook: validate message: %w", err)
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("slack webhook: marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("slack webhook: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack webhook: send request: %w", err)
	}
	defer resp.Body.Close()

	return checkResponseStatus("slack webhook", resp)
}

// Client sends Slack messages through the Slack Web API.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Slack Web API client.
func NewClient(token string, options ...HTTPOption) *Client {
	config := newHTTPConfig(options...)

	return &Client{
		token:      token,
		baseURL:    config.baseURL,
		httpClient: config.httpClient,
	}
}

// PostMessageResponse is the Slack chat.postMessage response.
type PostMessageResponse struct {
	OK        bool   `json:"ok"`
	Channel   string `json:"channel,omitempty"`
	Timestamp string `json:"ts,omitempty"`
	Error     string `json:"error,omitempty"`
}

// APIError is returned when Slack returns an ok=false API response.
type APIError struct {
	Code string
}

func (e *APIError) Error() string {
	return "slack api: " + e.Code
}

//nolint:tagliatelle // Slack API uses snake_case JSON fields.
type postMessageRequest struct {
	Channel  string  `json:"channel"`
	Text     string  `json:"text,omitempty"`
	Blocks   []Block `json:"blocks,omitempty"`
	ThreadTS string  `json:"thread_ts,omitempty"`
}

// PostMessage sends a notification to a channel through chat.postMessage.
func (c *Client) PostMessage(ctx context.Context, channel string, message Notification) (PostMessageResponse, error) {
	var response PostMessageResponse

	payload, err := newPostMessageRequest(channel, message)
	if err != nil {
		return response, err
	}

	resp, err := c.doPostMessage(ctx, payload)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	if err := checkResponseStatus("slack api", resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, fmt.Errorf("slack api: decode response: %w", err)
	}

	if !response.OK {
		return response, &APIError{Code: response.Error}
	}

	return response, nil
}

func newPostMessageRequest(channel string, message Notification) ([]byte, error) {
	if channel == "" {
		return nil, fmt.Errorf("slack api: channel: %w", ErrNilChannel)
	}

	if err := message.validate(); err != nil {
		return nil, fmt.Errorf("slack api: validate message: %w", err)
	}

	requestPayload := postMessageRequest{
		Channel:  channel,
		Text:     message.Text,
		Blocks:   message.Blocks,
		ThreadTS: message.ThreadTS,
	}

	payload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("slack api: marshal message: %w", err)
	}

	return payload, nil
}

func (c *Client) doPostMessage(ctx context.Context, payload []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/chat.postMessage",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("slack api: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack api: send request: %w", err)
	}

	return resp, nil
}

func newHTTPConfig(options ...HTTPOption) httpConfig {
	config := httpConfig{
		httpClient: http.DefaultClient,
		baseURL:    defaultBaseURL,
	}

	for _, option := range options {
		option(&config)
	}

	if config.httpClient == nil {
		config.httpClient = http.DefaultClient
	}

	if config.baseURL == "" {
		config.baseURL = defaultBaseURL
	}

	return config
}

func readResponseBody(body io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseBodyBytes))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func checkResponseStatus(prefix string, resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: unexpected status %d: read response body: %w", prefix, resp.StatusCode, err)
	}

	return fmt.Errorf("%s: unexpected status %d: %s", prefix, resp.StatusCode, body)
}
