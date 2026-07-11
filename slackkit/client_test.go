package slackkit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestWebhookSender_Send(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		td.Cmp(t, r.Method, http.MethodPost)
		td.Cmp(t, r.Header.Get("Content-Type"), "application/json")

		body, err := io.ReadAll(r.Body)
		td.CmpNoError(t, err)

		var got any
		err = json.Unmarshal(body, &got)
		td.CmpNoError(t, err)
		td.Cmp(t, got, td.JSON(`{
			"text": "Fallback text.",
			"blocks": [
				{
					"type": "section",
					"text": {
						"type": "mrkdwn",
						"text": "*Hello* from servekit"
					}
				}
			]
		}`))

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("ok"))
		td.CmpNoError(t, err)
	}))
	defer server.Close()

	message, err := NewMessage("Fallback text.", NewMarkdownSection("*Hello* from servekit"))
	td.CmpNoError(t, err)

	sender := NewWebhookSender(server.URL, WithHTTPClient(server.Client()))

	err = sender.Send(context.Background(), *message)

	td.CmpNoError(t, err)
	td.Cmp(t, requestCount, 1)
}

func TestWebhookSender_SendReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid_payload", http.StatusBadRequest)
	}))
	defer server.Close()

	message, err := NewMessage("Fallback text.", NewDivider())
	td.CmpNoError(t, err)

	sender := NewWebhookSender(server.URL, WithHTTPClient(server.Client()))

	err = sender.Send(context.Background(), *message)

	td.Cmp(t, err, td.NotNil())
	td.Cmp(t, err.Error(), td.Contains("slack webhook: unexpected status 400"))
	td.Cmp(t, err.Error(), td.Contains("invalid_payload"))
}

func TestClient_PostMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		td.Cmp(t, r.Method, http.MethodPost)
		td.Cmp(t, r.URL.Path, "/api/chat.postMessage")
		td.Cmp(t, r.Header.Get("Authorization"), "Bearer xoxb-token")
		td.Cmp(t, r.Header.Get("Content-Type"), "application/json")

		body, err := io.ReadAll(r.Body)
		td.CmpNoError(t, err)

		var got any
		err = json.Unmarshal(body, &got)
		td.CmpNoError(t, err)
		td.Cmp(t, got, td.JSON(`{
			"channel": "C123",
			"text": "Fallback text.",
			"blocks": [
				{
					"type": "section",
					"text": {
						"type": "mrkdwn",
						"text": "*Hello* from servekit"
					}
				}
			]
		}`))

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1710000000.000100"}`))
		td.CmpNoError(t, err)
	}))
	defer server.Close()

	message, err := NewMessage("Fallback text.", NewMarkdownSection("*Hello* from servekit"))
	td.CmpNoError(t, err)

	client := NewClient("xoxb-token", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	response, err := client.PostMessage(context.Background(), "C123", *message)

	td.CmpNoError(t, err)
	td.Cmp(t, response.OK, true)
	td.Cmp(t, response.Channel, "C123")
	td.Cmp(t, response.Timestamp, "1710000000.000100")
}

func TestClient_PostMessageReturnsSlackAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
		td.CmpNoError(t, err)
	}))
	defer server.Close()

	message, err := NewMessage("Fallback text.", NewDivider())
	td.CmpNoError(t, err)

	client := NewClient("xoxb-token", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	_, err = client.PostMessage(context.Background(), "C123", *message)

	var apiErr *APIError
	td.Cmp(t, errors.As(err, &apiErr), true)
	td.Cmp(t, apiErr.Code, "invalid_auth")
	td.Cmp(t, err.Error(), "slack api: invalid_auth")
}

func TestClient_PostMessageRejectsEmptyChannel(t *testing.T) {
	message, err := NewMessage("Fallback text.", NewDivider())
	td.CmpNoError(t, err)

	client := NewClient("xoxb-token")

	_, err = client.PostMessage(context.Background(), "", *message)

	td.Cmp(t, errors.Is(err, ErrNilChannel), true)
}
