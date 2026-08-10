package tgkit

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestNewMiniAppAndButtons(t *testing.T) {
	t.Parallel()

	app, err := NewMiniApp("https://app.example.com/telegram")
	if err != nil {
		t.Fatalf("NewMiniApp() error = %v", err)
	}
	if app.URL() != "https://app.example.com/telegram" {
		t.Errorf("URL() = %q", app.URL())
	}

	keyboard, err := app.KeyboardButton("Open")
	if err != nil {
		t.Fatalf("KeyboardButton() error = %v", err)
	}
	if keyboard.WebApp == nil || keyboard.WebApp.URL != app.URL() {
		t.Errorf("keyboard WebApp = %#v", keyboard.WebApp)
	}

	inline, err := app.InlineButton("Open")
	if err != nil {
		t.Fatalf("InlineButton() error = %v", err)
	}
	if inline.WebApp == nil || inline.WebApp.URL != app.URL() {
		t.Errorf("inline WebApp = %#v", inline.WebApp)
	}

	query, err := app.InlineQueryButton("Create")
	if err != nil {
		t.Fatalf("InlineQueryButton() error = %v", err)
	}
	if query.WebApp == nil || query.WebApp.URL != app.URL() {
		t.Errorf("inline-query WebApp = %#v", query.WebApp)
	}

	menu, err := app.MenuButton("Launch")
	if err != nil {
		t.Fatalf("MenuButton() error = %v", err)
	}
	if menu.Type != models.MenuButtonTypeWebApp || menu.WebApp.URL != app.URL() {
		t.Errorf("menu button = %#v", menu)
	}
}

func TestNewMiniAppRejectsInvalidURL(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"http://app.example.com",
		"/relative",
		"https://user:secret@app.example.com",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()

			_, err := NewMiniApp(rawURL)
			if !errors.Is(err, ErrInvalidMiniApp) {
				t.Fatalf("NewMiniApp() error = %v, want %v", err, ErrInvalidMiniApp)
			}
		})
	}
}

func TestMiniAppButtonsRejectEmptyText(t *testing.T) {
	t.Parallel()

	app, err := NewMiniApp("https://app.example.com")
	if err != nil {
		t.Fatalf("NewMiniApp() error = %v", err)
	}

	if _, err := app.KeyboardButton(" "); !errors.Is(err, ErrInvalidMiniApp) {
		t.Errorf("KeyboardButton() error = %v", err)
	}
	if _, err := app.InlineButton(""); !errors.Is(err, ErrInvalidMiniApp) {
		t.Errorf("InlineButton() error = %v", err)
	}
	if _, err := app.InlineQueryButton(""); !errors.Is(err, ErrInvalidMiniApp) {
		t.Errorf("InlineQueryButton() error = %v", err)
	}
	if _, err := app.MenuButton(""); !errors.Is(err, ErrInvalidMiniApp) {
		t.Errorf("MenuButton() error = %v", err)
	}
}

func TestMiniAppSetMenuButtonRequiresBot(t *testing.T) {
	t.Parallel()

	app, err := NewMiniApp("https://app.example.com")
	if err != nil {
		t.Fatalf("NewMiniApp() error = %v", err)
	}

	err = app.SetMenuButton(context.Background(), nil, "Launch")
	if !errors.Is(err, ErrMiniAppNotBound) {
		t.Fatalf("SetMenuButton() error = %v, want %v", err, ErrMiniAppNotBound)
	}
}

func TestMiniAppSetMenuButton(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, true)
	app, err := bot.MiniApp("https://app.example.com")
	if err != nil {
		t.Fatalf("MiniApp() error = %v", err)
	}

	err = app.SetMenuButton(context.Background(), int64(123), "Launch")
	if err != nil {
		t.Fatalf("SetMenuButton() error = %v", err)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/setChatMenuButton" {
		t.Errorf("request path = %q", request.path)
	}
	if got := request.values.Get("chat_id"); got != "123" {
		t.Errorf("chat_id = %q, want 123", got)
	}
	if got := request.values.Get("menu_button"); !strings.Contains(got, `"type":"web_app"`) ||
		!strings.Contains(got, `"url":"https://app.example.com"`) {
		t.Errorf("menu_button = %q", got)
	}
}

func TestMiniAppAnswerQuery(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, map[string]any{"inline_message_id": "inline_123"})
	app, err := bot.MiniApp("https://app.example.com")
	if err != nil {
		t.Fatalf("MiniApp() error = %v", err)
	}

	message, err := app.AnswerQuery(context.Background(), "query_123", &models.InlineQueryResultArticle{
		ID:    "result_123",
		Title: "Created",
		InputMessageContent: models.InputTextMessageContent{
			MessageText: "Created in the Mini App.",
		},
	})
	if err != nil {
		t.Fatalf("AnswerQuery() error = %v", err)
	}
	if message.InlineMessageID != "inline_123" {
		t.Errorf("inline message ID = %q", message.InlineMessageID)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/answerWebAppQuery" {
		t.Errorf("request path = %q", request.path)
	}
	if got := request.values.Get("web_app_query_id"); got != "query_123" {
		t.Errorf("web_app_query_id = %q", got)
	}
	if got := request.values.Get("result"); !strings.Contains(got, `"type":"article"`) ||
		!strings.Contains(got, `"message_text":"Created in the Mini App."`) {
		t.Errorf("result = %q", got)
	}
}

func TestMiniAppLink(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		username   string
		shortName  string
		startParam string
		mode       MiniAppMode
		want       string
		wantErr    bool
	}{
		"main app": {
			username: "@example_bot",
			want:     "https://t.me/example_bot?startapp",
		},
		"named compact app": {
			username:   "example_bot",
			shortName:  "store",
			startParam: "order 123",
			mode:       MiniAppModeCompact,
			want:       "https://t.me/example_bot/store?startapp=order+123&mode=compact",
		},
		"invalid username": {
			username: "bad/name",
			wantErr:  true,
		},
		"invalid short name": {
			username:  "example_bot",
			shortName: "bad/name",
			wantErr:   true,
		},
		"invalid mode": {
			username: "example_bot",
			mode:     MiniAppMode("full"),
			wantErr:  true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := MiniAppLink(test.username, test.shortName, test.startParam, test.mode)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidMiniApp) {
					t.Fatalf("MiniAppLink() error = %v, want %v", err, ErrInvalidMiniApp)
				}

				return
			}
			if err != nil {
				t.Fatalf("MiniAppLink() error = %v", err)
			}
			if got != test.want {
				t.Errorf("MiniAppLink() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMiniAppMiddlewareRejectsMissingData(t *testing.T) {
	t.Parallel()

	validator, err := NewMiniAppValidator(testBotToken, DefaultMiniAppInitDataMaxAge)
	if err != nil {
		t.Fatalf("NewMiniAppValidator() error = %v", err)
	}

	handler := validator.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("next handler was called")
	}))
	response := httptestResponse(handler, "")
	if response.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
