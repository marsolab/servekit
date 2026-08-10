package tgkit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const testBotToken = "123456:test-token"

type telegramRequest struct {
	path   string
	values url.Values
	err    error
}

func newTelegramTestBot(t *testing.T, result any, options ...Option) (*Bot, <-chan telegramRequest) {
	t.Helper()

	requests := make(chan telegramRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := request.ParseMultipartForm(1 << 20)
		requests <- telegramRequest{
			path:   request.URL.Path,
			values: request.Form,
			err:    err,
		}

		response.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(response).Encode(map[string]any{
			"ok":     true,
			"result": result,
		}); err != nil {
			panic(fmt.Sprintf("encode Telegram test response: %v", err))
		}
	}))
	t.Cleanup(server.Close)

	botOptions := []Option{
		WithServerURL(server.URL),
		WithHTTPClient(time.Second, server.Client()),
		WithSkipIdentityCheck(),
	}
	botOptions = append(botOptions, options...)

	bot, err := New(testBotToken, botOptions...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return bot, requests
}

func receiveTelegramRequest(t *testing.T, requests <-chan telegramRequest) telegramRequest {
	t.Helper()

	select {
	case request := <-requests:
		if request.err != nil {
			t.Fatalf("parse Telegram request: %v", request.err)
		}

		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Telegram request")

		return telegramRequest{}
	}
}
