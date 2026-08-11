// Package tgkit provides reusable Telegram bot, channel, Mini App, and payment
// building blocks.
//
// Bot wraps the complete Telegram Bot API client and implements servekit's
// Listener contract for long polling and webhook workers. Channel adds
// channel-scoped helpers, MiniApp handles launch controls and authenticated
// initialization data, and Payments covers one-time invoices, Telegram Stars
// subscriptions, refunds, and renewal management.
//
// Telegram bots and channels must still be provisioned in Telegram. The Bot API
// can operate a bot and administer channels to which it has been added, but it
// does not provision bot accounts or create channel chats.
package tgkit
