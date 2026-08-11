package tgkit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var (
	// ErrInvalidMiniApp is returned when a Mini App URL, button, or direct link
	// is invalid.
	ErrInvalidMiniApp = errors.New("invalid telegram mini app")

	// ErrMiniAppNotBound is returned when an API operation is attempted on a
	// standalone MiniApp that is not associated with a Bot.
	ErrMiniAppNotBound = errors.New("telegram mini app is not bound to a bot")
)

// MiniAppMode controls the initial height of a Mini App opened by direct link.
type MiniAppMode string

const (
	// MiniAppModeDefault uses the Mini App's configured opening mode.
	MiniAppModeDefault MiniAppMode = ""

	// MiniAppModeCompact opens the Mini App at half-screen height.
	MiniAppModeCompact MiniAppMode = "compact"
)

const (
	minBotUsernameLength = 5
	maxBotUsernameLength = 32
	botUsernameSuffix    = "bot"
)

// MiniApp provides launch controls and Bot API operations for a Telegram Mini
// App.
type MiniApp struct {
	bot *telegram.Bot
	url string
}

// NewMiniApp creates standalone Mini App launch controls for an HTTPS URL.
func NewMiniApp(appURL string) (*MiniApp, error) {
	if err := validateMiniAppURL(appURL); err != nil {
		return nil, err
	}

	return &MiniApp{url: appURL}, nil
}

// MiniApp creates Mini App launch controls bound to this bot.
func (b *Bot) MiniApp(appURL string) (*MiniApp, error) {
	app, err := NewMiniApp(appURL)
	if err != nil {
		return nil, err
	}

	app.bot = b.Bot

	return app, nil
}

// URL returns the Mini App's HTTPS URL.
func (a *MiniApp) URL() string {
	return a.url
}

// KeyboardButton creates a reply-keyboard button that launches the Mini App.
func (a *MiniApp) KeyboardButton(text string) (models.KeyboardButton, error) {
	if strings.TrimSpace(text) == "" {
		return models.KeyboardButton{}, fmt.Errorf("%w: empty keyboard button text", ErrInvalidMiniApp)
	}

	return models.KeyboardButton{
		Text:   text,
		WebApp: &models.WebAppInfo{URL: a.url},
	}, nil
}

// InlineButton creates an inline-keyboard button that launches the Mini App.
func (a *MiniApp) InlineButton(text string) (models.InlineKeyboardButton, error) {
	if strings.TrimSpace(text) == "" {
		return models.InlineKeyboardButton{}, fmt.Errorf("%w: empty inline button text", ErrInvalidMiniApp)
	}

	return models.InlineKeyboardButton{
		Text:   text,
		WebApp: &models.WebAppInfo{URL: a.url},
	}, nil
}

// InlineQueryButton creates an inline-mode result button that launches the Mini
// App.
func (a *MiniApp) InlineQueryButton(text string) (models.InlineQueryResultsButton, error) {
	if strings.TrimSpace(text) == "" {
		return models.InlineQueryResultsButton{}, fmt.Errorf("%w: empty inline-query button text", ErrInvalidMiniApp)
	}

	return models.InlineQueryResultsButton{
		Text:   text,
		WebApp: &models.WebAppInfo{URL: a.url},
	}, nil
}

// MenuButton creates a bot menu button that launches the Mini App.
func (a *MiniApp) MenuButton(text string) (models.MenuButtonWebApp, error) {
	if strings.TrimSpace(text) == "" {
		return models.MenuButtonWebApp{}, fmt.Errorf("%w: empty menu button text", ErrInvalidMiniApp)
	}

	return models.MenuButtonWebApp{
		Type: models.MenuButtonTypeWebApp,
		Text: text,
		WebApp: models.WebAppInfo{
			URL: a.url,
		},
	}, nil
}

// SetMenuButton configures this Mini App as the bot menu button for a chat. A
// nil chatID configures the default menu button for all users.
func (a *MiniApp) SetMenuButton(ctx context.Context, chatID any, text string) error {
	if a.bot == nil {
		return ErrMiniAppNotBound
	}

	button, err := a.MenuButton(text)
	if err != nil {
		return err
	}

	ok, err := a.bot.SetChatMenuButton(ctx, &telegram.SetChatMenuButtonParams{
		ChatID:     chatID,
		MenuButton: button,
	})

	return checkTelegramResult("set Telegram Mini App menu button", telegramResult(ok), err)
}

// AnswerQuery sends an inline result on behalf of a Mini App user and closes
// the Mini App session.
func (a *MiniApp) AnswerQuery(
	ctx context.Context,
	queryID string,
	result models.InlineQueryResult,
) (*models.SentWebAppMessage, error) {
	if a.bot == nil {
		return nil, ErrMiniAppNotBound
	}

	if strings.TrimSpace(queryID) == "" {
		return nil, fmt.Errorf("%w: empty web app query ID", ErrInvalidMiniApp)
	}

	message, err := a.bot.AnswerWebAppQuery(ctx, &telegram.AnswerWebAppQueryParams{
		WebAppQueryID: queryID,
		Result:        result,
	})
	if err != nil {
		return nil, fmt.Errorf("answer Telegram Mini App query: %w", err)
	}

	return message, nil
}

// MiniAppLink builds a t.me direct link for a main or named Mini App. The bot
// username must contain 5-32 ASCII letters, digits, or underscores and end in
// "bot". Leave shortName empty to link to the bot's main Mini App.
func MiniAppLink(
	botUsername string,
	shortName string,
	startParameter string,
	mode MiniAppMode,
) (string, error) {
	username := strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if !validBotUsername(username) {
		return "", fmt.Errorf("%w: invalid bot username", ErrInvalidMiniApp)
	}

	if shortName != "" && !validMiniAppShortName(shortName) {
		return "", fmt.Errorf("%w: invalid short name", ErrInvalidMiniApp)
	}

	if mode != MiniAppModeDefault && mode != MiniAppModeCompact {
		return "", fmt.Errorf("%w: invalid opening mode %q", ErrInvalidMiniApp, mode)
	}

	link := &url.URL{
		Scheme: "https",
		Host:   "t.me",
		Path:   "/" + username,
	}
	if shortName != "" {
		link.Path += "/" + shortName
	}

	query := make([]string, 0, 2)
	if startParameter != "" {
		query = append(query, "startapp="+url.QueryEscape(startParameter))
	} else if shortName == "" {
		query = append(query, "startapp")
	}

	if mode == MiniAppModeCompact {
		query = append(query, "mode="+string(mode))
	}

	link.RawQuery = strings.Join(query, "&")

	return link.String(), nil
}

func validateMiniAppURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("%w: parse URL: %w", ErrInvalidMiniApp, err)
	}

	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%w: URL must be an absolute HTTPS URL without credentials", ErrInvalidMiniApp)
	}

	return nil
}

func validBotUsername(value string) bool {
	if len(value) < minBotUsernameLength || len(value) > maxBotUsernameLength ||
		!strings.EqualFold(value[len(value)-len(botUsernameSuffix):], botUsernameSuffix) {
		return false
	}

	for _, character := range value {
		if character != '_' && !isASCIILetterOrDigit(character) {
			return false
		}
	}

	return true
}

func validMiniAppShortName(value string) bool {
	if value == "" {
		return false
	}

	for _, character := range value {
		if character != '-' && character != '_' && !isASCIILetterOrDigit(character) {
			return false
		}
	}

	return true
}

func isASCIILetterOrDigit(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}
