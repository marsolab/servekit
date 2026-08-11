package tgkit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
)

func TestValidateInvoice(t *testing.T) {
	t.Parallel()

	validStars := invoiceFields{
		title:       "Pro plan",
		description: "One month of Pro access",
		payload:     "order_123",
		currency:    StarsCurrency,
		prices:      []models.LabeledPrice{{Label: "Pro", Amount: 250}},
	}
	validFiat := invoiceFields{
		title:         "T-shirt",
		description:   "Cotton T-shirt",
		payload:       "order_456",
		providerToken: "provider-token",
		currency:      "USD",
		prices: []models.LabeledPrice{
			{Label: "Item", Amount: 1000},
			{Label: "Discount", Amount: -100},
		},
		maxTipAmount:        300,
		suggestedTipAmounts: []int{100, 200},
	}

	tests := map[string]struct {
		invoice invoiceFields
		kind    invoiceKind
		wantErr error
	}{
		"valid Stars": {invoice: validStars},
		"valid fiat":  {invoice: validFiat},
		"empty title": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) { invoice.title = "" }),
			wantErr: ErrInvalidInvoice,
		},
		"blank title": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) { invoice.title = "   " }),
			wantErr: ErrInvalidInvoice,
		},
		"long description": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) {
				invoice.description = strings.Repeat("x", maxInvoiceDescriptionLength+1)
			}),
			wantErr: ErrInvalidInvoice,
		},
		"long payload": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) {
				invoice.payload = strings.Repeat("x", maxInvoicePayloadBytes+1)
			}),
			wantErr: ErrInvalidInvoice,
		},
		"lowercase currency": {
			invoice: mutateInvoice(validFiat, func(invoice *invoiceFields) { invoice.currency = "usd" }),
			wantErr: ErrInvalidInvoice,
		},
		"non-letter currency": {
			invoice: mutateInvoice(validFiat, func(invoice *invoiceFields) { invoice.currency = "U1D" }),
			wantErr: ErrInvalidInvoice,
		},
		"fiat without provider": {
			invoice: mutateInvoice(validFiat, func(invoice *invoiceFields) { invoice.providerToken = "" }),
			wantErr: ErrInvalidInvoice,
		},
		"Stars with provider": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) { invoice.providerToken = "provider" }),
			wantErr: ErrInvalidInvoice,
		},
		"Stars with multiple prices": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) {
				invoice.prices = append(invoice.prices, models.LabeledPrice{Label: "Tax", Amount: 1})
			}),
			wantErr: ErrInvalidInvoice,
		},
		"non-positive total": {
			invoice: mutateInvoice(validFiat, func(invoice *invoiceFields) {
				invoice.prices = []models.LabeledPrice{{Label: "Free", Amount: 0}}
			}),
			wantErr: ErrInvalidInvoice,
		},
		"empty price label": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) { invoice.prices[0].Label = "" }),
			wantErr: ErrInvalidInvoice,
		},
		"Stars with tips": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) { invoice.maxTipAmount = 10 }),
			wantErr: ErrInvalidInvoice,
		},
		"unordered tips": {
			invoice: mutateInvoice(validFiat, func(invoice *invoiceFields) {
				invoice.suggestedTipAmounts = []int{200, 100}
			}),
			wantErr: ErrInvalidInvoice,
		},
		"valid subscription": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) {
				invoice.subscriptionPeriod = int(MonthlySubscriptionPeriod / time.Second)
			}),
			kind: invoiceSubscription,
		},
		"subscription must use Stars": {
			invoice: mutateInvoice(validFiat, func(invoice *invoiceFields) {
				invoice.subscriptionPeriod = int(MonthlySubscriptionPeriod / time.Second)
			}),
			kind:    invoiceSubscription,
			wantErr: ErrInvalidSubscription,
		},
		"subscription price maximum": {
			invoice: mutateInvoice(validStars, func(invoice *invoiceFields) {
				invoice.subscriptionPeriod = int(MonthlySubscriptionPeriod / time.Second)
				invoice.prices[0].Amount = MaxStarsSubscriptionPrice + 1
			}),
			kind:    invoiceSubscription,
			wantErr: ErrInvalidSubscription,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateInvoice(test.invoice, test.kind)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("validateInvoice() error = %v, want %v", err, test.wantErr)
				}

				return
			}
			if err != nil {
				t.Fatalf("validateInvoice() error = %v", err)
			}
		})
	}
}

func TestPaymentsSendInvoice(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, map[string]any{"message_id": 77})
	message, err := bot.Payments().SendInvoice(context.Background(), SendInvoiceParams{
		ChatID:      int64(123),
		Title:       "Credits",
		Description: "100 account credits",
		Payload:     "order_123",
		Currency:    StarsCurrency,
		Prices:      []LabeledPrice{{Label: "Credits", Amount: 100}},
	})
	if err != nil {
		t.Fatalf("SendInvoice() error = %v", err)
	}
	if message.ID != 77 {
		t.Errorf("message ID = %d, want 77", message.ID)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/sendInvoice" {
		t.Errorf("request path = %q", request.path)
	}
	if got := request.values.Get("currency"); got != StarsCurrency {
		t.Errorf("currency = %q, want %q", got, StarsCurrency)
	}
	if got := request.values.Get("prices"); got != `[{"label":"Credits","amount":100}]` {
		t.Errorf("prices = %q", got)
	}
}

func TestPaymentsCreateSubscriptionLink(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, "https://t.me/$invoice")
	link, err := bot.Payments().CreateSubscriptionLink(context.Background(), CreateInvoiceLinkParams{
		Title:       "Pro",
		Description: "Monthly Pro access",
		Payload:     "subscription_pro",
		Currency:    StarsCurrency,
		Prices:      []LabeledPrice{{Label: "Pro", Amount: 500}},
	})
	if err != nil {
		t.Fatalf("CreateSubscriptionLink() error = %v", err)
	}
	if link != "https://t.me/$invoice" {
		t.Errorf("link = %q", link)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/createInvoiceLink" {
		t.Errorf("request path = %q", request.path)
	}
	if got := request.values.Get("subscription_period"); got != "2592000" {
		t.Errorf("subscription_period = %q, want 2592000", got)
	}
}

func TestPaymentsCreateInvoiceLink(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, "https://t.me/$one-time")
	link, err := bot.Payments().CreateInvoiceLink(context.Background(), CreateInvoiceLinkParams{
		Title:         "T-shirt",
		Description:   "Cotton T-shirt",
		Payload:       "order_456",
		ProviderToken: "provider-token",
		Currency:      "USD",
		Prices:        []LabeledPrice{{Label: "T-shirt", Amount: 2500}},
	})
	if err != nil {
		t.Fatalf("CreateInvoiceLink() error = %v", err)
	}
	if link != "https://t.me/$one-time" {
		t.Errorf("link = %q", link)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/createInvoiceLink" {
		t.Errorf("request path = %q", request.path)
	}
	if got := request.values.Get("provider_token"); got != "provider-token" {
		t.Errorf("provider_token = %q", got)
	}
	if got := request.values.Get("subscription_period"); got != "" {
		t.Errorf("subscription_period = %q, want empty", got)
	}
}

func TestPaymentsCheckoutAndRefundOperations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		call       func(*Payments) error
		path       string
		wantValues map[string]string
	}{
		"approve checkout": {
			call: func(payments *Payments) error {
				return payments.ApprovePreCheckout(context.Background(), "checkout_123")
			},
			path: "/bot" + testBotToken + "/answerPreCheckoutQuery",
			wantValues: map[string]string{
				"pre_checkout_query_id": "checkout_123",
				"ok":                    "true",
			},
		},
		"reject checkout": {
			call: func(payments *Payments) error {
				return payments.RejectPreCheckout(context.Background(), "checkout_456", "Item unavailable")
			},
			path: "/bot" + testBotToken + "/answerPreCheckoutQuery",
			wantValues: map[string]string{
				"pre_checkout_query_id": "checkout_456",
				"ok":                    "false",
				"error_message":         "Item unavailable",
			},
		},
		"refund Stars": {
			call: func(payments *Payments) error {
				return payments.RefundStars(context.Background(), 123, "charge_refund")
			},
			path: "/bot" + testBotToken + "/refundStarPayment",
			wantValues: map[string]string{
				"user_id":                    "123",
				"telegram_payment_charge_id": "charge_refund",
			},
		},
		"resume subscription": {
			call: func(payments *Payments) error {
				return payments.ResumeSubscription(context.Background(), 456, "charge_resume")
			},
			path: "/bot" + testBotToken + "/editUserStarSubscription",
			wantValues: map[string]string{
				"user_id":                    "456",
				"telegram_payment_charge_id": "charge_resume",
				"is_canceled":                "false",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bot, requests := newTelegramTestBot(t, true)
			if err := test.call(bot.Payments()); err != nil {
				t.Fatalf("operation error = %v", err)
			}

			request := receiveTelegramRequest(t, requests)
			if request.path != test.path {
				t.Errorf("request path = %q, want %q", request.path, test.path)
			}
			for key, want := range test.wantValues {
				if got := request.values.Get(key); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestPaymentsRejectInvalidOperationInput(t *testing.T) {
	t.Parallel()

	payments := &Payments{}
	tests := map[string]struct {
		call    func() error
		wantErr error
	}{
		"approve without query": {
			call:    func() error { return payments.ApprovePreCheckout(context.Background(), "") },
			wantErr: ErrInvalidInvoice,
		},
		"reject without message": {
			call: func() error {
				return payments.RejectPreCheckout(context.Background(), "checkout", "")
			},
			wantErr: ErrInvalidInvoice,
		},
		"refund without user": {
			call:    func() error { return payments.RefundStars(context.Background(), 0, "charge") },
			wantErr: ErrInvalidInvoice,
		},
		"cancel without charge": {
			call:    func() error { return payments.CancelSubscription(context.Background(), 123, "") },
			wantErr: ErrInvalidSubscription,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := test.call(); !errors.Is(err, test.wantErr) {
				t.Fatalf("operation error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestPaymentsManageSubscription(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, true)
	err := bot.Payments().CancelSubscription(context.Background(), 123, "charge_123")
	if err != nil {
		t.Fatalf("CancelSubscription() error = %v", err)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/editUserStarSubscription" {
		t.Errorf("request path = %q", request.path)
	}
	wants := map[string]string{
		"user_id":                    "123",
		"telegram_payment_charge_id": "charge_123",
		"is_canceled":                "true",
	}
	for key, want := range wants {
		if got := request.values.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestPaymentFromUpdate(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().Add(MonthlySubscriptionPeriod).Truncate(time.Second)
	tests := map[string]struct {
		payment *models.SuccessfulPayment
		kind    PaymentKind
	}{
		"one-time": {
			payment: &models.SuccessfulPayment{TelegramPaymentChargeID: "one"},
			kind:    PaymentOneTime,
		},
		"subscription started": {
			payment: &models.SuccessfulPayment{
				TelegramPaymentChargeID:    "start",
				IsRecurring:                true,
				IsFirstRecurring:           true,
				SubscriptionExpirationDate: int(expiresAt.Unix()),
			},
			kind: PaymentSubscriptionStarted,
		},
		"subscription renewed": {
			payment: &models.SuccessfulPayment{
				TelegramPaymentChargeID: "renew",
				IsRecurring:             true,
			},
			kind: PaymentSubscriptionRenewed,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			update := &models.Update{Message: &models.Message{
				ID:                99,
				From:              &models.User{ID: 123},
				Chat:              models.Chat{ID: 456},
				SuccessfulPayment: test.payment,
			}}
			event, ok := PaymentFromUpdate(update)
			if !ok {
				t.Fatal("PaymentFromUpdate() did not find payment")
			}
			if event.Kind != test.kind {
				t.Errorf("kind = %q, want %q", event.Kind, test.kind)
			}
			if event.UserID != 123 || event.ChatID != 456 || event.MessageID != 99 {
				t.Errorf("event location = user %d, chat %d, message %d", event.UserID, event.ChatID, event.MessageID)
			}
			if event.IdempotencyKey() != test.payment.TelegramPaymentChargeID {
				t.Errorf("idempotency key = %q", event.IdempotencyKey())
			}
			if test.payment.SubscriptionExpirationDate != 0 {
				got, found := event.SubscriptionExpiresAt()
				if !found || !got.Equal(expiresAt) {
					t.Errorf("SubscriptionExpiresAt() = %v, %v", got, found)
				}
			}
		})
	}
}

func TestRefundFromUpdate(t *testing.T) {
	t.Parallel()

	update := &models.Update{BusinessMessage: &models.Message{
		ID:   12,
		From: &models.User{ID: 34},
		Chat: models.Chat{ID: 56},
		RefundedPayment: &models.RefundedPayment{
			TelegramPaymentChargeID: "refund_123",
		},
	}}
	event, ok := RefundFromUpdate(update)
	if !ok {
		t.Fatal("RefundFromUpdate() did not find refund")
	}
	if event.Refund.TelegramPaymentChargeID != "refund_123" {
		t.Errorf("charge ID = %q", event.Refund.TelegramPaymentChargeID)
	}
}

func TestPaymentFromUpdateWithoutPayment(t *testing.T) {
	t.Parallel()

	if _, ok := PaymentFromUpdate(nil); ok {
		t.Error("PaymentFromUpdate(nil) found a payment")
	}
	if _, ok := RefundFromUpdate(&models.Update{}); ok {
		t.Error("RefundFromUpdate(empty) found a refund")
	}
}

func mutateInvoice(invoice invoiceFields, mutate func(*invoiceFields)) invoiceFields {
	invoice.prices = append([]models.LabeledPrice(nil), invoice.prices...)
	invoice.suggestedTipAmounts = append([]int(nil), invoice.suggestedTipAmounts...)
	mutate(&invoice)

	return invoice
}
