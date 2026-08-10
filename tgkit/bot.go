package tgkit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var (
	// ErrInvalidMode is returned when a bot has an unsupported update mode.
	ErrInvalidMode = errors.New("invalid telegram update mode")

	// ErrInvalidBotConfig is returned when a bot option would produce an invalid
	// runtime configuration.
	ErrInvalidBotConfig = errors.New("invalid telegram bot configuration")
)

// Mode controls how a Bot receives Telegram updates.
type Mode uint8

const (
	// ModePolling receives updates through Telegram long polling.
	ModePolling Mode = iota

	// ModeWebhook receives updates submitted to Bot.WebhookHandler.
	ModeWebhook
)

// Handler handles a Telegram update.
type Handler = telegram.HandlerFunc

// Middleware decorates a Telegram update handler.
type Middleware = telegram.Middleware

// Update is a Telegram Bot API update.
type Update = models.Update

// HTTPClient executes Telegram Bot API requests.
type HTTPClient interface {
	Do(request *http.Request) (*http.Response, error)
}

// Option configures a Bot.
type Option func(*botConfig)

type botConfig struct {
	mode            Mode
	telegramOptions []telegram.Option
	err             error
}

// Bot is a ServeKit-compatible Telegram bot.
//
// The embedded Telegram client exposes the complete Bot API, including handler
// registration and methods not covered by tgkit's higher-level helpers.
type Bot struct {
	*telegram.Bot

	mode Mode
}

// New creates a Telegram bot. Long polling is used unless ModeWebhook is
// selected.
func New(token string, options ...Option) (*Bot, error) {
	config := botConfig{
		mode: ModePolling,
		telegramOptions: []telegram.Option{
			telegram.WithDefaultHandler(discardUpdate),
			telegram.WithErrorsHandler(discardError),
		},
	}

	for _, option := range options {
		if option == nil {
			config.setError(fmt.Errorf("%w: nil option", ErrInvalidBotConfig))

			continue
		}

		option(&config)
	}

	if config.err != nil {
		return nil, config.err
	}

	if config.mode != ModePolling && config.mode != ModeWebhook {
		return nil, fmt.Errorf("%w: %d", ErrInvalidMode, config.mode)
	}

	client, err := telegram.New(token, config.telegramOptions...)
	if err != nil {
		return nil, fmt.Errorf("create Telegram bot: %w", err)
	}

	return &Bot{
		Bot:  client,
		mode: config.mode,
	}, nil
}

// WithMode selects how the bot receives Telegram updates.
func WithMode(mode Mode) Option {
	return func(config *botConfig) {
		config.mode = mode
	}
}

// WithWebhook selects webhook mode and configures the secret Telegram must
// include with webhook requests.
func WithWebhook(secretToken string) Option {
	return func(config *botConfig) {
		config.mode = ModeWebhook
		config.telegramOptions = append(
			config.telegramOptions,
			telegram.WithWebhookSecretToken(secretToken),
		)
	}
}

// WithDefaultHandler configures the fallback update handler.
func WithDefaultHandler(handler Handler) Option {
	return func(config *botConfig) {
		if handler == nil {
			config.setError(fmt.Errorf("%w: nil default handler", ErrInvalidBotConfig))

			return
		}

		config.telegramOptions = append(config.telegramOptions, telegram.WithDefaultHandler(handler))
	}
}

// WithMiddlewares configures middleware applied to every update handler.
func WithMiddlewares(middlewares ...Middleware) Option {
	return func(config *botConfig) {
		for _, middleware := range middlewares {
			if middleware == nil {
				config.setError(fmt.Errorf("%w: nil middleware", ErrInvalidBotConfig))

				return
			}
		}

		config.telegramOptions = append(config.telegramOptions, telegram.WithMiddlewares(middlewares...))
	}
}

// WithErrorHandler configures handling for asynchronous polling, webhook, and
// update-processing errors.
func WithErrorHandler(handler func(error)) Option {
	return func(config *botConfig) {
		if handler == nil {
			config.setError(fmt.Errorf("%w: nil error handler", ErrInvalidBotConfig))

			return
		}

		config.telegramOptions = append(config.telegramOptions, telegram.WithErrorsHandler(handler))
	}
}

// WithWorkers configures the number of concurrent update workers.
func WithWorkers(workers int) Option {
	return func(config *botConfig) {
		if workers < 1 {
			config.setError(fmt.Errorf("%w: workers must be positive", ErrInvalidBotConfig))

			return
		}

		config.telegramOptions = append(config.telegramOptions, telegram.WithWorkers(workers))
	}
}

// WithAllowedUpdates restricts the update types Telegram should deliver.
func WithAllowedUpdates(updates ...string) Option {
	return withTelegramOption(telegram.WithAllowedUpdates(updates))
}

// WithServerURL configures a Telegram Bot API server URL. It supports both the
// official endpoint and self-hosted Bot API servers.
func WithServerURL(serverURL string) Option {
	return func(config *botConfig) {
		if serverURL == "" {
			config.setError(fmt.Errorf("%w: empty server URL", ErrInvalidBotConfig))

			return
		}

		config.telegramOptions = append(config.telegramOptions, telegram.WithServerURL(serverURL))
	}
}

// WithHTTPClient configures the HTTP client and long-poll timeout.
func WithHTTPClient(pollTimeout time.Duration, client HTTPClient) Option {
	return func(config *botConfig) {
		if pollTimeout <= 0 {
			config.setError(fmt.Errorf("%w: poll timeout must be positive", ErrInvalidBotConfig))

			return
		}

		if client == nil {
			config.setError(fmt.Errorf("%w: nil HTTP client", ErrInvalidBotConfig))

			return
		}

		config.telegramOptions = append(config.telegramOptions, telegram.WithHTTPClient(pollTimeout, client))
	}
}

// WithSkipIdentityCheck skips the getMe request made when the bot is created.
// It is intended for tests and specialized self-hosted Bot API setups.
func WithSkipIdentityCheck() Option {
	return withTelegramOption(telegram.WithSkipGetMe())
}

// WithTelegramOptions applies lower-level go-telegram/bot options. Options are
// applied in call order and provide an escape hatch for new client features.
func WithTelegramOptions(options ...telegram.Option) Option {
	return func(config *botConfig) {
		for _, option := range options {
			if option == nil {
				config.setError(fmt.Errorf("%w: nil Telegram option", ErrInvalidBotConfig))

				return
			}
		}

		config.telegramOptions = append(config.telegramOptions, options...)
	}
}

// Mode reports how the bot receives updates.
func (b *Bot) Mode() Mode {
	return b.mode
}

// Telegram returns the underlying complete Telegram Bot API client.
func (b *Bot) Telegram() *telegram.Bot {
	return b.Bot
}

// Serve receives and dispatches Telegram updates until the context is canceled.
// In webhook mode callers must mount WebhookHandler on an HTTP listener.
func (b *Bot) Serve(ctx context.Context) error {
	switch b.mode {
	case ModePolling:
		b.Start(ctx)
	case ModeWebhook:
		b.StartWebhook(ctx)
	default:
		return fmt.Errorf("%w: %d", ErrInvalidMode, b.mode)
	}

	return nil
}

func withTelegramOption(option telegram.Option) Option {
	return func(config *botConfig) {
		config.telegramOptions = append(config.telegramOptions, option)
	}
}

func (c *botConfig) setError(err error) {
	if c.err == nil {
		c.err = err
	}
}

func discardUpdate(context.Context, *telegram.Bot, *models.Update) {}

func discardError(error) {}
