package tgkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		token     string
		options   []Option
		wantError bool
		wantErrIs error
		mode      Mode
	}{
		"polling by default": {
			token:   testBotToken,
			options: []Option{WithSkipIdentityCheck()},
			mode:    ModePolling,
		},
		"webhook": {
			token:   testBotToken,
			options: []Option{WithSkipIdentityCheck(), WithWebhook("secret")},
			mode:    ModeWebhook,
		},
		"empty token": {
			options:   []Option{WithSkipIdentityCheck()},
			wantError: true,
		},
		"invalid mode": {
			token:     testBotToken,
			options:   []Option{WithSkipIdentityCheck(), WithMode(Mode(99))},
			wantError: true,
			wantErrIs: ErrInvalidMode,
		},
		"zero workers": {
			token:     testBotToken,
			options:   []Option{WithSkipIdentityCheck(), WithWorkers(0)},
			wantError: true,
			wantErrIs: ErrInvalidBotConfig,
		},
		"nil option": {
			token:     testBotToken,
			options:   []Option{WithSkipIdentityCheck(), nil},
			wantError: true,
			wantErrIs: ErrInvalidBotConfig,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bot, err := New(test.token, test.options...)
			if test.wantError {
				if err == nil {
					t.Fatal("New() error = nil, want an error")
				}
				if test.wantErrIs != nil && !errors.Is(err, test.wantErrIs) {
					t.Fatalf("New() error = %v, want errors.Is(%v)", err, test.wantErrIs)
				}

				return
			}
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if bot.Mode() != test.mode {
				t.Errorf("Mode() = %v, want %v", bot.Mode(), test.mode)
			}
			if bot.Telegram() == nil {
				t.Error("Telegram() = nil")
			}
		})
	}
}

func TestBotServeWebhookDispatchesUpdate(t *testing.T) {
	t.Parallel()

	updates := make(chan int64, 1)
	bot, err := New(
		testBotToken,
		WithSkipIdentityCheck(),
		WithWebhook("webhook-secret"),
		WithDefaultHandler(func(_ context.Context, _ *telegram.Bot, update *models.Update) {
			updates <- update.ID
		}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- bot.Serve(ctx)
	}()

	request := httptest.NewRequest(http.MethodPost, "/telegram", strings.NewReader(`{"update_id":42}`))
	request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "webhook-secret")
	response := httptest.NewRecorder()
	bot.WebhookHandler().ServeHTTP(response, request)

	select {
	case updateID := <-updates:
		if updateID != 42 {
			t.Errorf("update ID = %d, want 42", updateID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for webhook update")
	}

	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not stop after cancellation")
	}
}

func TestBotServePollingDispatchesUpdate(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if calls.Add(1) > 1 {
			<-request.Context().Done()

			return
		}

		response.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(response).Encode(map[string]any{
			"ok": true,
			"result": []map[string]any{
				{"update_id": 84},
			},
		}); err != nil {
			panic(fmt.Sprintf("encode polling response: %v", err))
		}
	}))
	t.Cleanup(server.Close)

	updates := make(chan int64, 1)
	bot, err := New(
		testBotToken,
		WithSkipIdentityCheck(),
		WithServerURL(server.URL),
		WithHTTPClient(2*time.Second, server.Client()),
		WithDefaultHandler(func(_ context.Context, _ *telegram.Bot, update *models.Update) {
			updates <- update.ID
		}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- bot.Serve(ctx)
	}()

	select {
	case updateID := <-updates:
		if updateID != 84 {
			t.Errorf("update ID = %d, want 84", updateID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for polling update")
	}

	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not stop after cancellation")
	}
}

func TestBotServeRejectsInvalidMode(t *testing.T) {
	t.Parallel()

	bot := &Bot{mode: Mode(99)}
	err := bot.Serve(context.Background())
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("Serve() error = %v, want %v", err, ErrInvalidMode)
	}
}
