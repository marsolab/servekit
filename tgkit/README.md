# tgkit

`tgkit` is the Telegram package for ServeKit. It combines the complete
[`go-telegram/bot`](https://github.com/go-telegram/bot) client with focused,
validated helpers for:

- long-polling and webhook bots that implement `servekit.Listener`;
- channel posts, metadata, membership, join requests, invite links, and paid
  subscription links;
- Mini App keyboard, inline, menu, inline-query, and direct-link launch points;
- server-side Mini App `initData` validation with either the bot token or
  Telegram's public Ed25519 keys;
- one-time provider or Telegram Stars invoices;
- recurring Telegram Stars invoices, renewal cancellation/resumption, refunds,
  and payment-event extraction.

## Telegram provisioning boundary

Telegram does not expose bot-account or channel-chat creation through the Bot
API. Create bots and configure their Mini Apps with
[@BotFather](https://t.me/BotFather), create channels in Telegram, and then add
the bot as an administrator with the rights needed by the operations below.
`tgkit` handles the application runtime and Bot API operations after that
provisioning step.

## Bot runtime

Long polling is the default. The bot can be registered directly with a
ServeKit server because it implements `Serve(context.Context) error`.

```go
bot, err := tgkit.New(
    os.Getenv("TELEGRAM_BOT_TOKEN"),
    tgkit.WithDefaultHandler(handleUpdate),
    tgkit.WithErrorHandler(reportTelegramError),
    tgkit.WithWorkers(4),
)
if err != nil {
    return err
}

server.RegisterListener("telegram", bot)
```

For webhooks, select webhook mode, mount the handler on an HTTPS endpoint, and
register that URL with Telegram using the embedded client.

```go
bot, err := tgkit.New(
    os.Getenv("TELEGRAM_BOT_TOKEN"),
    tgkit.WithWebhook(os.Getenv("TELEGRAM_WEBHOOK_SECRET")),
    tgkit.WithDefaultHandler(handleUpdate),
)
if err != nil {
    return err
}

router.Handle("/webhooks/telegram", bot.WebhookHandler())

_, err = bot.SetWebhook(ctx, &telegram.SetWebhookParams{
    URL:         "https://api.example.com/webhooks/telegram",
    SecretToken: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
})
```

The embedded `*bot.Bot` exposes handler registration and the complete Bot API,
so new or uncommon Telegram methods remain available without waiting for a
`tgkit` wrapper.

## Channels

Create a channel-scoped helper with a numeric chat ID or `@username`.

```go
channel, err := bot.Channel("@product_updates")
if err != nil {
    return err
}

_, err = channel.Send(ctx, telegram.SendMessageParams{
    Text: "Version 2.0 is live.",
})
if err != nil {
    return err
}

paidLink, err := channel.CreateSubscriptionInviteLink(ctx, tgkit.ChannelSubscription{
    Name:       "Members",
    PriceStars: 250,
})
```

Regular invite links, join approvals, channel metadata, administrators, member
lookups, pinning, and revocation are also channel-scoped methods.

## Mini Apps

Launch controls require an absolute HTTPS URL.

```go
app, err := bot.MiniApp("https://app.example.com")
if err != nil {
    return err
}

button, err := app.InlineButton("Open app")
if err != nil {
    return err
}

_, err = channel.Send(ctx, telegram.SendMessageParams{
    Text: "Open the app:",
    ReplyMarkup: models.InlineKeyboardMarkup{
        InlineKeyboard: [][]models.InlineKeyboardButton{{button}},
    },
})
```

Never trust `Telegram.WebApp.initDataUnsafe`. Send the raw
`Telegram.WebApp.initData` query string to the backend and authenticate it. The
built-in middleware reads it from `X-Telegram-Init-Data` and adds parsed data to
the request context.

```go
validator, err := tgkit.NewMiniAppValidator(
    os.Getenv("TELEGRAM_BOT_TOKEN"),
    tgkit.DefaultMiniAppInitDataMaxAge,
)
if err != nil {
    return err
}

router.With(validator.Middleware).Get("/mini-app/me", func(w http.ResponseWriter, r *http.Request) {
    initData, ok := tgkit.MiniAppInitDataFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // initData.User and the other fields are authenticated here.
})
```

Third parties that must not receive the bot token can use
`NewMiniAppSignatureValidator` with Telegram's production or test public key.

## Payments and subscriptions

Provider-backed fiat payments require the provider token configured through
BotFather. Telegram Stars payments use `XTR`, an empty provider token, and one
price component.

```go
payments := bot.Payments()

_, err := payments.SendInvoice(ctx, tgkit.SendInvoiceParams{
    ChatID:      userID,
    Title:       "100 credits",
    Description: "Account credit pack",
    Payload:     orderID,
    Currency:    tgkit.StarsCurrency,
    Prices: []tgkit.LabeledPrice{
        {Label: "Credits", Amount: 100},
    },
})
```

Recurring bot subscriptions currently use a 30-day Telegram Stars invoice
link. `CreateSubscriptionLink` sets that period and validates Telegram's current
price and currency rules.

```go
invoiceURL, err := payments.CreateSubscriptionLink(ctx, tgkit.CreateInvoiceLinkParams{
    Title:       "Pro plan",
    Description: "30 days of Pro access",
    Payload:     subscriptionID,
    Currency:    tgkit.StarsCurrency,
    Prices: []tgkit.LabeledPrice{
        {Label: "Pro", Amount: 500},
    },
})
```

Every checkout must be answered within Telegram's deadline using
`ApprovePreCheckout` or `RejectPreCheckout`. Grant entitlements only after
`PaymentFromUpdate` returns a successful payment. Persist
`PaymentEvent.IdempotencyKey()` before granting access so webhook retries cannot
apply a charge twice. Subscription events distinguish the first charge from
renewals and expose the paid expiration time.

Stars payments can be refunded with `RefundStars`. Recurring renewal can be
disabled or re-enabled with `CancelSubscription` and `ResumeSubscription`; a
cancellation leaves the already-paid period active.
