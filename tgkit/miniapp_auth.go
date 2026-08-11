package tgkit

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	maxMiniAppInitDataBytes = 16 * 1024
	miniAppFutureClockSkew  = 30 * time.Second
	webAppDataKey           = "WebAppData"

	telegramMiniAppProductionKeyHex = "e7bf03a2fa4602af4580703d88dda5bb59f32ed8b02a56c187fe7d34caed242d"
	telegramMiniAppTestKeyHex       = "40055058a4ee38156a06562e52eece92a771bcd8346a8c4615cb7376eddf72ec"

	// MiniAppInitDataHeader is the default HTTP header read by Mini App
	// authentication middleware.
	MiniAppInitDataHeader = "X-Telegram-Init-Data"

	// DefaultMiniAppInitDataMaxAge is the recommended default validity window
	// for authenticated Mini App initialization data.
	DefaultMiniAppInitDataMaxAge = 5 * time.Minute
)

var (
	// ErrInvalidMiniAppInitData is returned for malformed, incomplete, or
	// unauthenticated Mini App initialization data.
	ErrInvalidMiniAppInitData = errors.New("invalid telegram mini app init data")

	// ErrExpiredMiniAppInitData is returned when authenticated initialization
	// data is older than the configured validity window.
	ErrExpiredMiniAppInitData = errors.New("expired telegram mini app init data")

	// ErrFutureMiniAppInitData is returned when initialization data has an
	// auth_date unreasonably far in the future.
	ErrFutureMiniAppInitData = errors.New("future telegram mini app init data")
)

// MiniAppEnvironment selects Telegram's Ed25519 signature-verification key.
type MiniAppEnvironment uint8

const (
	// MiniAppEnvironmentProduction validates data from Telegram production.
	MiniAppEnvironmentProduction MiniAppEnvironment = iota

	// MiniAppEnvironmentTest validates data from Telegram's test environment.
	MiniAppEnvironmentTest
)

// MiniAppUser contains user data authenticated from Mini App initData.
//
//nolint:tagliatelle // Telegram Mini Apps use snake_case fields.
type MiniAppUser struct {
	ID                    int64  `json:"id"`
	IsBot                 bool   `json:"is_bot,omitempty"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name,omitempty"`
	Username              string `json:"username,omitempty"`
	LanguageCode          string `json:"language_code,omitempty"`
	IsPremium             bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu bool   `json:"added_to_attachment_menu,omitempty"`
	AllowsWriteToPM       bool   `json:"allows_write_to_pm,omitempty"`
	PhotoURL              string `json:"photo_url,omitempty"`
}

// MiniAppChat contains chat data authenticated from Mini App initData.
//
//nolint:tagliatelle // Telegram Mini Apps use snake_case fields.
type MiniAppChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username,omitempty"`
	PhotoURL string `json:"photo_url,omitempty"`
}

// MiniAppInitData is parsed and authenticated Telegram.WebApp.initData.
type MiniAppInitData struct {
	QueryID      string
	User         *MiniAppUser
	Receiver     *MiniAppUser
	Chat         *MiniAppChat
	ChatType     string
	ChatInstance string
	StartParam   string
	CanSendAfter time.Duration
	AuthDate     time.Time
	Hash         string
	Signature    string
	Values       map[string]string
	Raw          string
}

// MiniAppValidator authenticates Mini App initData with a private bot token.
type MiniAppValidator struct {
	botToken string
	maxAge   time.Duration
}

// MiniAppSignatureValidator authenticates Mini App initData for a third party
// using Telegram's public Ed25519 key and a bot ID.
type MiniAppSignatureValidator struct {
	botID     int64
	publicKey ed25519.PublicKey
	maxAge    time.Duration
}

type miniAppInitDataContextKey struct{}

// NewMiniAppValidator creates a bot-token Mini App data validator.
func NewMiniAppValidator(botToken string, maxAge time.Duration) (*MiniAppValidator, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, fmt.Errorf("%w: empty bot token", ErrInvalidMiniAppInitData)
	}

	if maxAge <= 0 {
		return nil, fmt.Errorf("%w: max age must be positive", ErrInvalidMiniAppInitData)
	}

	return &MiniAppValidator{
		botToken: botToken,
		maxAge:   maxAge,
	}, nil
}

// NewMiniAppSignatureValidator creates a third-party Mini App data validator
// for Telegram's production or test environment.
func NewMiniAppSignatureValidator(
	botID int64,
	environment MiniAppEnvironment,
	maxAge time.Duration,
) (*MiniAppSignatureValidator, error) {
	if botID <= 0 {
		return nil, fmt.Errorf("%w: bot ID must be positive", ErrInvalidMiniAppInitData)
	}

	if maxAge <= 0 {
		return nil, fmt.Errorf("%w: max age must be positive", ErrInvalidMiniAppInitData)
	}

	publicKeyHex, err := miniAppPublicKeyHex(environment)
	if err != nil {
		return nil, err
	}

	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode Telegram Mini App public key: %w", err)
	}

	return &MiniAppSignatureValidator{
		botID:     botID,
		publicKey: ed25519.PublicKey(publicKey),
		maxAge:    maxAge,
	}, nil
}

// Validate authenticates and parses Mini App initData using the current time.
func (v *MiniAppValidator) Validate(raw string) (*MiniAppInitData, error) {
	return v.ValidateAt(raw, time.Now())
}

// ValidateAt authenticates and parses Mini App initData at a supplied time.
func (v *MiniAppValidator) ValidateAt(raw string, now time.Time) (*MiniAppInitData, error) {
	values, err := parseMiniAppValues(raw)
	if err != nil {
		return nil, err
	}

	if err := validateMiniAppHash(values, v.botToken); err != nil {
		return nil, err
	}

	return decodeMiniAppInitData(raw, values, now, v.maxAge)
}

// Middleware authenticates Mini App initData from MiniAppInitDataHeader and
// stores it in the request context. Invalid requests receive HTTP 401.
func (v *MiniAppValidator) Middleware(next http.Handler) http.Handler {
	return miniAppMiddleware(v.Validate, next)
}

// Validate authenticates and parses Mini App initData using Telegram's public
// signature and the current time.
func (v *MiniAppSignatureValidator) Validate(raw string) (*MiniAppInitData, error) {
	return v.ValidateAt(raw, time.Now())
}

// ValidateAt authenticates and parses Mini App initData using Telegram's public
// signature at a supplied time.
func (v *MiniAppSignatureValidator) ValidateAt(raw string, now time.Time) (*MiniAppInitData, error) {
	values, err := parseMiniAppValues(raw)
	if err != nil {
		return nil, err
	}

	if err := validateMiniAppSignature(values, v.botID, v.publicKey); err != nil {
		return nil, err
	}

	return decodeMiniAppInitData(raw, values, now, v.maxAge)
}

// Middleware authenticates third-party Mini App initData from
// MiniAppInitDataHeader and stores it in the request context. Invalid requests
// receive HTTP 401.
func (v *MiniAppSignatureValidator) Middleware(next http.Handler) http.Handler {
	return miniAppMiddleware(v.Validate, next)
}

// MiniAppInitDataFromContext returns authenticated Mini App data installed by
// the authentication middleware.
func MiniAppInitDataFromContext(ctx context.Context) (*MiniAppInitData, bool) {
	data, ok := ctx.Value(miniAppInitDataContextKey{}).(*MiniAppInitData)

	return data, ok
}

func parseMiniAppValues(raw string) (url.Values, error) {
	if raw == "" {
		return nil, fmt.Errorf("%w: empty value", ErrInvalidMiniAppInitData)
	}

	if len(raw) > maxMiniAppInitDataBytes {
		return nil, fmt.Errorf("%w: value exceeds %d bytes", ErrInvalidMiniAppInitData, maxMiniAppInitDataBytes)
	}

	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: parse query: %w", ErrInvalidMiniAppInitData, err)
	}

	for key, entries := range values {
		if len(entries) != 1 {
			return nil, fmt.Errorf("%w: duplicate field %q", ErrInvalidMiniAppInitData, key)
		}
	}

	return values, nil
}

func validateMiniAppHash(values url.Values, botToken string) error {
	receivedHash, err := hex.DecodeString(values.Get("hash"))
	if err != nil || len(receivedHash) != sha256.Size {
		return fmt.Errorf("%w: malformed hash", ErrInvalidMiniAppInitData)
	}

	secret := hmac.New(sha256.New, []byte(webAppDataKey))
	if _, err := secret.Write([]byte(botToken)); err != nil {
		return fmt.Errorf("%w: derive secret: %w", ErrInvalidMiniAppInitData, err)
	}

	digest := hmac.New(sha256.New, secret.Sum(nil))
	if _, err := digest.Write([]byte(miniAppDataCheckString(values, "hash"))); err != nil {
		return fmt.Errorf("%w: calculate hash: %w", ErrInvalidMiniAppInitData, err)
	}

	if !hmac.Equal(receivedHash, digest.Sum(nil)) {
		return fmt.Errorf("%w: hash mismatch", ErrInvalidMiniAppInitData)
	}

	return nil
}

func validateMiniAppSignature(values url.Values, botID int64, publicKey ed25519.PublicKey) error {
	signature, err := decodeMiniAppSignature(values.Get("signature"))
	if err != nil {
		return err
	}

	prefix := strconv.FormatInt(botID, 10) + ":" + webAppDataKey + "\n"

	checkString := prefix + miniAppDataCheckString(values, "hash", "signature")
	if !ed25519.Verify(publicKey, []byte(checkString), signature) {
		return fmt.Errorf("%w: signature mismatch", ErrInvalidMiniAppInitData)
	}

	return nil
}

func decodeMiniAppSignature(value string) ([]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("%w: missing signature", ErrInvalidMiniAppInitData)
	}

	signature, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		signature, err = base64.URLEncoding.DecodeString(value)
	}

	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("%w: malformed signature", ErrInvalidMiniAppInitData)
	}

	return signature, nil
}

func miniAppDataCheckString(values url.Values, excludedKeys ...string) string {
	excluded := make(map[string]struct{}, len(excludedKeys))
	for _, key := range excludedKeys {
		excluded[key] = struct{}{}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		if _, exclude := excluded[key]; !exclude {
			keys = append(keys, key)
		}
	}

	slices.Sort(keys)

	fields := make([]string, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, key+"="+values.Get(key))
	}

	return strings.Join(fields, "\n")
}

func decodeMiniAppInitData(
	raw string,
	values url.Values,
	now time.Time,
	maxAge time.Duration,
) (*MiniAppInitData, error) {
	authDate, err := parseMiniAppAuthDate(values.Get("auth_date"))
	if err != nil {
		return nil, err
	}

	if err := validateMiniAppAuthDate(authDate, now, maxAge); err != nil {
		return nil, err
	}

	canSendAfter, err := parseOptionalSeconds(values.Get("can_send_after"))
	if err != nil {
		return nil, err
	}

	data := &MiniAppInitData{
		QueryID:      values.Get("query_id"),
		ChatType:     values.Get("chat_type"),
		ChatInstance: values.Get("chat_instance"),
		StartParam:   values.Get("start_param"),
		CanSendAfter: canSendAfter,
		AuthDate:     authDate,
		Hash:         values.Get("hash"),
		Signature:    values.Get("signature"),
		Values:       flattenMiniAppValues(values),
		Raw:          raw,
	}

	if err := decodeOptionalJSON(values.Get("user"), &data.User); err != nil {
		return nil, fmt.Errorf("%w: decode user: %w", ErrInvalidMiniAppInitData, err)
	}

	if err := decodeOptionalJSON(values.Get("receiver"), &data.Receiver); err != nil {
		return nil, fmt.Errorf("%w: decode receiver: %w", ErrInvalidMiniAppInitData, err)
	}

	if err := decodeOptionalJSON(values.Get("chat"), &data.Chat); err != nil {
		return nil, fmt.Errorf("%w: decode chat: %w", ErrInvalidMiniAppInitData, err)
	}

	return data, nil
}

func parseMiniAppAuthDate(value string) (time.Time, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}, fmt.Errorf("%w: invalid auth_date", ErrInvalidMiniAppInitData)
	}

	return time.Unix(seconds, 0), nil
}

func parseOptionalSeconds(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}

	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds < 0 {
		return 0, fmt.Errorf("%w: invalid can_send_after", ErrInvalidMiniAppInitData)
	}

	return time.Duration(seconds) * time.Second, nil
}

func validateMiniAppAuthDate(authDate, now time.Time, maxAge time.Duration) error {
	if authDate.After(now.Add(miniAppFutureClockSkew)) {
		return ErrFutureMiniAppInitData
	}

	if now.Sub(authDate) > maxAge {
		return ErrExpiredMiniAppInitData
	}

	return nil
}

func decodeOptionalJSON[T any](value string, destination **T) error {
	if value == "" {
		return nil
	}

	decoded := new(T)
	if err := json.Unmarshal([]byte(value), decoded); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	*destination = decoded

	return nil
}

func flattenMiniAppValues(values url.Values) map[string]string {
	flattened := make(map[string]string, len(values))
	for key := range values {
		flattened[key] = values.Get(key)
	}

	return flattened
}

func miniAppPublicKeyHex(environment MiniAppEnvironment) (string, error) {
	switch environment {
	case MiniAppEnvironmentProduction:
		return telegramMiniAppProductionKeyHex, nil
	case MiniAppEnvironmentTest:
		return telegramMiniAppTestKeyHex, nil
	default:
		return "", fmt.Errorf("%w: invalid environment %d", ErrInvalidMiniAppInitData, environment)
	}
}

func miniAppMiddleware(
	validate func(string) (*MiniAppInitData, error),
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		data, err := validate(request.Header.Get(MiniAppInitDataHeader))
		if err != nil {
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

			return
		}

		ctx := context.WithValue(request.Context(), miniAppInitDataContextKey{}, data)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}
