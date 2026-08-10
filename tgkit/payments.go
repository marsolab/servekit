package tgkit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	maxInvoiceTitleLength       = 32
	maxInvoiceDescriptionLength = 255
	maxInvoicePayloadBytes      = 128
	maxSuggestedTips            = 4

	// StarsCurrency is Telegram's ISO-like currency code for Telegram Stars.
	StarsCurrency = "XTR"
)

// ErrInvalidInvoice is returned when invoice settings violate Telegram's
// payment constraints.
var ErrInvalidInvoice = errors.New("invalid telegram invoice")

// LabeledPrice is one component of an invoice price.
type LabeledPrice = models.LabeledPrice

// SendInvoiceParams configures a one-time invoice sent to a Telegram chat.
type SendInvoiceParams = telegram.SendInvoiceParams

// CreateInvoiceLinkParams configures a one-time or recurring invoice link.
type CreateInvoiceLinkParams = telegram.CreateInvoiceLinkParams

// PreCheckoutQuery is Telegram's final checkout confirmation request.
type PreCheckoutQuery = models.PreCheckoutQuery

// SuccessfulPayment is Telegram's successful payment payload.
type SuccessfulPayment = models.SuccessfulPayment

// RefundedPayment is Telegram's refunded payment payload.
type RefundedPayment = models.RefundedPayment

// PaymentKind identifies whether a successful payment is one-time, an initial
// subscription payment, or a subscription renewal.
type PaymentKind string

const (
	// PaymentOneTime identifies a non-recurring payment.
	PaymentOneTime PaymentKind = "one_time"

	// PaymentSubscriptionStarted identifies the first subscription charge.
	PaymentSubscriptionStarted PaymentKind = "subscription_started"

	// PaymentSubscriptionRenewed identifies a later subscription charge.
	PaymentSubscriptionRenewed PaymentKind = "subscription_renewed"
)

// PaymentEvent is a successful payment extracted from a Telegram update.
// Payment.TelegramPaymentChargeID is the stable idempotency key applications
// should persist before granting an entitlement.
type PaymentEvent struct {
	Kind      PaymentKind
	UserID    int64
	ChatID    int64
	MessageID int
	Payment   *models.SuccessfulPayment
}

// RefundEvent is a refunded Stars payment extracted from a Telegram update.
type RefundEvent struct {
	UserID    int64
	ChatID    int64
	MessageID int
	Refund    *models.RefundedPayment
}

// Payments provides validated Telegram invoice and subscription operations.
type Payments struct {
	bot *telegram.Bot
}

// Payments creates a payment helper for the bot.
func (b *Bot) Payments() *Payments {
	return &Payments{bot: b.Bot}
}

// SendInvoice sends a one-time invoice after validating Telegram's payment
// constraints.
func (p *Payments) SendInvoice(ctx context.Context, params SendInvoiceParams) (*models.Message, error) {
	if err := validateInvoice(invoiceFieldsFromSend(params), invoiceOneTime); err != nil {
		return nil, err
	}

	message, err := p.bot.SendInvoice(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("send Telegram invoice: %w", err)
	}

	return message, nil
}

// CreateInvoiceLink creates a one-time invoice link after validating Telegram's
// payment constraints.
func (p *Payments) CreateInvoiceLink(ctx context.Context, params CreateInvoiceLinkParams) (string, error) {
	if params.SubscriptionPeriod != 0 {
		return "", fmt.Errorf("%w: use CreateSubscriptionLink for recurring invoices", ErrInvalidInvoice)
	}

	if err := validateInvoice(invoiceFieldsFromLink(params), invoiceOneTime); err != nil {
		return "", err
	}

	link, err := p.bot.CreateInvoiceLink(ctx, &params)
	if err != nil {
		return "", fmt.Errorf("create Telegram invoice link: %w", err)
	}

	return link, nil
}

// CreateSubscriptionLink creates a monthly Telegram Stars subscription invoice
// link. Telegram currently supports only a 30-day recurring period.
func (p *Payments) CreateSubscriptionLink(ctx context.Context, params CreateInvoiceLinkParams) (string, error) {
	params.SubscriptionPeriod = int(MonthlySubscriptionPeriod / time.Second)
	if err := validateInvoice(invoiceFieldsFromLink(params), invoiceSubscription); err != nil {
		return "", err
	}

	link, err := p.bot.CreateInvoiceLink(ctx, &params)
	if err != nil {
		return "", fmt.Errorf("create Telegram subscription link: %w", err)
	}

	return link, nil
}

// ApprovePreCheckout approves a Telegram pre-checkout query. Telegram requires
// the answer within ten seconds.
func (p *Payments) ApprovePreCheckout(ctx context.Context, queryID string) error {
	return p.answerPreCheckout(ctx, queryID, true, "")
}

// RejectPreCheckout rejects a Telegram pre-checkout query with a user-facing
// explanation.
func (p *Payments) RejectPreCheckout(ctx context.Context, queryID, message string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("%w: empty pre-checkout rejection message", ErrInvalidInvoice)
	}

	return p.answerPreCheckout(ctx, queryID, false, message)
}

// RefundStars refunds a successful Telegram Stars payment.
func (p *Payments) RefundStars(ctx context.Context, userID int64, chargeID string) error {
	if userID == 0 || strings.TrimSpace(chargeID) == "" {
		return fmt.Errorf("%w: refund requires a user and charge ID", ErrInvalidInvoice)
	}

	ok, err := p.bot.RefundStarPayment(ctx, &telegram.RefundStarPaymentParams{
		UserID:                  userID,
		TelegramPaymentChargeID: chargeID,
	})

	return checkTelegramResult("refund Telegram Stars payment", telegramResult(ok), err)
}

// CancelSubscription prevents a Telegram Stars subscription from renewing. The
// current paid period remains active.
func (p *Payments) CancelSubscription(ctx context.Context, userID int64, chargeID string) error {
	return p.setSubscriptionCanceled(ctx, userID, chargeID, true)
}

// ResumeSubscription allows a previously canceled Telegram Stars subscription
// to renew again.
func (p *Payments) ResumeSubscription(ctx context.Context, userID int64, chargeID string) error {
	return p.setSubscriptionCanceled(ctx, userID, chargeID, false)
}

// PaymentFromUpdate extracts a successful payment and classifies recurring
// subscription charges.
func PaymentFromUpdate(update *models.Update) (PaymentEvent, bool) {
	message := paymentMessage(update)
	if message == nil || message.SuccessfulPayment == nil {
		return PaymentEvent{}, false
	}

	event := PaymentEvent{
		Kind:      classifyPayment(message.SuccessfulPayment),
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		Payment:   message.SuccessfulPayment,
	}
	if message.From != nil {
		event.UserID = message.From.ID
	}

	return event, true
}

// RefundFromUpdate extracts a refunded payment from a Telegram update.
func RefundFromUpdate(update *models.Update) (RefundEvent, bool) {
	message := paymentMessage(update)
	if message == nil || message.RefundedPayment == nil {
		return RefundEvent{}, false
	}

	event := RefundEvent{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		Refund:    message.RefundedPayment,
	}
	if message.From != nil {
		event.UserID = message.From.ID
	}

	return event, true
}

// IdempotencyKey returns Telegram's unique charge identifier for this payment.
func (e PaymentEvent) IdempotencyKey() string {
	if e.Payment == nil {
		return ""
	}

	return e.Payment.TelegramPaymentChargeID
}

// SubscriptionExpiresAt returns the paid subscription period's expiration.
func (e PaymentEvent) SubscriptionExpiresAt() (time.Time, bool) {
	if e.Payment == nil || e.Payment.SubscriptionExpirationDate == 0 {
		return time.Time{}, false
	}

	return time.Unix(int64(e.Payment.SubscriptionExpirationDate), 0), true
}

func (p *Payments) answerPreCheckout(ctx context.Context, queryID string, approved bool, message string) error {
	if strings.TrimSpace(queryID) == "" {
		return fmt.Errorf("%w: empty pre-checkout query ID", ErrInvalidInvoice)
	}

	ok, err := p.bot.AnswerPreCheckoutQuery(ctx, &telegram.AnswerPreCheckoutQueryParams{
		PreCheckoutQueryID: queryID,
		OK:                 approved,
		ErrorMessage:       message,
	})

	return checkTelegramResult("answer Telegram pre-checkout query", telegramResult(ok), err)
}

func (p *Payments) setSubscriptionCanceled(
	ctx context.Context,
	userID int64,
	chargeID string,
	canceled bool,
) error {
	if userID == 0 || strings.TrimSpace(chargeID) == "" {
		return fmt.Errorf("%w: subscription update requires a user and charge ID", ErrInvalidSubscription)
	}

	ok, err := p.bot.EditUserStarSubscription(ctx, &telegram.EditUserStarSubscriptionParams{
		UserID:                  userID,
		TelegramPaymentChargeID: chargeID,
		IsCanceled:              canceled,
	})

	return checkTelegramResult("edit Telegram Stars subscription", telegramResult(ok), err)
}

type invoiceFields struct {
	title               string
	description         string
	payload             string
	providerToken       string
	currency            string
	prices              []models.LabeledPrice
	subscriptionPeriod  int
	maxTipAmount        int
	suggestedTipAmounts []int
}

type invoiceKind uint8

const (
	invoiceOneTime invoiceKind = iota
	invoiceSubscription
)

func invoiceFieldsFromSend(params SendInvoiceParams) invoiceFields {
	return invoiceFields{
		title:               params.Title,
		description:         params.Description,
		payload:             params.Payload,
		providerToken:       params.ProviderToken,
		currency:            params.Currency,
		prices:              params.Prices,
		maxTipAmount:        params.MaxTipAmount,
		suggestedTipAmounts: params.SuggestedTipAmounts,
	}
}

func invoiceFieldsFromLink(params CreateInvoiceLinkParams) invoiceFields {
	return invoiceFields{
		title:               params.Title,
		description:         params.Description,
		payload:             params.Payload,
		providerToken:       params.ProviderToken,
		currency:            params.Currency,
		prices:              params.Prices,
		subscriptionPeriod:  params.SubscriptionPeriod,
		maxTipAmount:        params.MaxTipAmount,
		suggestedTipAmounts: params.SuggestedTipAmounts,
	}
}

func validateInvoice(invoice invoiceFields, kind invoiceKind) error {
	if err := validateInvoiceText(invoice); err != nil {
		return err
	}

	if err := validateInvoiceCurrency(invoice); err != nil {
		return err
	}

	if err := validateInvoicePrices(invoice); err != nil {
		return err
	}

	if err := validateInvoiceTips(invoice); err != nil {
		return err
	}

	return validateInvoiceSubscription(invoice, kind)
}

func validateInvoiceText(invoice invoiceFields) error {
	titleLength := utf8.RuneCountInString(invoice.title)
	if titleLength > maxInvoiceTitleLength || strings.TrimSpace(invoice.title) == "" {
		return fmt.Errorf("%w: title must contain 1-%d characters", ErrInvalidInvoice, maxInvoiceTitleLength)
	}

	descriptionLength := utf8.RuneCountInString(invoice.description)
	if descriptionLength > maxInvoiceDescriptionLength || strings.TrimSpace(invoice.description) == "" {
		return fmt.Errorf(
			"%w: description must contain 1-%d characters",
			ErrInvalidInvoice,
			maxInvoiceDescriptionLength,
		)
	}

	payloadLength := len([]byte(invoice.payload))
	if payloadLength < 1 || payloadLength > maxInvoicePayloadBytes {
		return fmt.Errorf("%w: payload must contain 1-%d bytes", ErrInvalidInvoice, maxInvoicePayloadBytes)
	}

	return nil
}

func validateInvoiceCurrency(invoice invoiceFields) error {
	if !isUppercaseCurrencyCode(invoice.currency) {
		return fmt.Errorf("%w: currency must be an uppercase three-letter code", ErrInvalidInvoice)
	}

	if invoice.currency == StarsCurrency {
		if invoice.providerToken != "" {
			return fmt.Errorf("%w: Telegram Stars invoices cannot use a provider token", ErrInvalidInvoice)
		}

		return nil
	}

	if strings.TrimSpace(invoice.providerToken) == "" {
		return fmt.Errorf("%w: non-Stars invoices require a provider token", ErrInvalidInvoice)
	}

	return nil
}

func validateInvoicePrices(invoice invoiceFields) error {
	if len(invoice.prices) == 0 {
		return fmt.Errorf("%w: at least one price is required", ErrInvalidInvoice)
	}

	if invoice.currency == StarsCurrency && len(invoice.prices) != 1 {
		return fmt.Errorf("%w: Telegram Stars invoices require exactly one price", ErrInvalidInvoice)
	}

	var total int64

	for _, price := range invoice.prices {
		if strings.TrimSpace(price.Label) == "" {
			return fmt.Errorf("%w: price labels must not be empty", ErrInvalidInvoice)
		}

		total += int64(price.Amount)
	}

	if total <= 0 {
		return fmt.Errorf("%w: total price must be positive", ErrInvalidInvoice)
	}

	return nil
}

func validateInvoiceTips(invoice invoiceFields) error {
	if invoice.currency == StarsCurrency && (invoice.maxTipAmount != 0 || len(invoice.suggestedTipAmounts) != 0) {
		return fmt.Errorf("%w: Telegram Stars invoices do not support tips", ErrInvalidInvoice)
	}

	if invoice.maxTipAmount < 0 {
		return fmt.Errorf("%w: maximum tip must not be negative", ErrInvalidInvoice)
	}

	if len(invoice.suggestedTipAmounts) > maxSuggestedTips {
		return fmt.Errorf("%w: at most %d suggested tips are allowed", ErrInvalidInvoice, maxSuggestedTips)
	}

	previous := 0
	for _, tip := range invoice.suggestedTipAmounts {
		if tip <= previous || tip > invoice.maxTipAmount {
			return fmt.Errorf("%w: suggested tips must increase and not exceed the maximum", ErrInvalidInvoice)
		}

		previous = tip
	}

	return nil
}

func validateInvoiceSubscription(invoice invoiceFields, kind invoiceKind) error {
	if kind == invoiceOneTime {
		if invoice.subscriptionPeriod != 0 {
			return fmt.Errorf("%w: recurring period is not allowed for a one-time invoice", ErrInvalidInvoice)
		}

		return nil
	}

	periodSeconds := int(MonthlySubscriptionPeriod / time.Second)

	if invoice.currency != StarsCurrency {
		return fmt.Errorf("%w: recurring invoices must use Telegram Stars", ErrInvalidSubscription)
	}

	if invoice.subscriptionPeriod != periodSeconds {
		return fmt.Errorf("%w: recurring period must be %d seconds", ErrInvalidSubscription, periodSeconds)
	}

	if invoice.prices[0].Amount > MaxStarsSubscriptionPrice {
		return fmt.Errorf(
			"%w: price cannot exceed %d Stars",
			ErrInvalidSubscription,
			MaxStarsSubscriptionPrice,
		)
	}

	return nil
}

func classifyPayment(payment *models.SuccessfulPayment) PaymentKind {
	if !payment.IsRecurring {
		return PaymentOneTime
	}

	if payment.IsFirstRecurring {
		return PaymentSubscriptionStarted
	}

	return PaymentSubscriptionRenewed
}

func paymentMessage(update *models.Update) *models.Message {
	if update == nil {
		return nil
	}

	if update.Message != nil {
		return update.Message
	}

	if update.BusinessMessage != nil {
		return update.BusinessMessage
	}

	if update.ChannelPost != nil {
		return update.ChannelPost
	}

	return nil
}

func isUppercaseCurrencyCode(currency string) bool {
	if len(currency) != 3 {
		return false
	}

	for _, character := range currency {
		if character < 'A' || character > 'Z' {
			return false
		}
	}

	return true
}
