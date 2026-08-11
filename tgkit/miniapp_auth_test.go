package tgkit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	officialMiniAppBotID = int64(7342037359)
	officialMiniAppToken = "7342037359:AAHI25ES9xCOMPokpYoz-p8XVrZUdygo2J4"
	officialMiniAppData  = "user=%7B%22id%22%3A279058397%2C%22first_name%22%3A%22Vladislav%20%2B%20-%20%3F%20%5C%2F%22%2C%22last_name%22%3A%22Kibenko%22%2C%22username%22%3A%22vdkfrost%22%2C%22language_code%22%3A%22ru%22%2C%22is_premium%22%3Atrue%2C%22allows_write_to_pm%22%3Atrue%2C%22photo_url%22%3A%22https%3A%5C%2F%5C%2Ft.me%5C%2Fi%5C%2Fuserpic%5C%2F320%5C%2F4FPEE4tmP3ATHa57u6MqTDih13LTOiMoKoLDRG4PnSA.svg%22%7D&chat_instance=8134722200314281151&chat_type=private&auth_date=1733509682&signature=TYJxVcisqbWjtodPepiJ6ghziUL94-KNpG8Pau-X7oNNLNBM72APCpi_RKiUlBvcqo5L-LAxIc3dnTzcZX_PDg&hash=a433d8f9847bd6addcc563bff7cc82c89e97ea0d90c11fe5729cae6796a36d73"
)

func TestMiniAppValidatorOfficialVector(t *testing.T) {
	t.Parallel()

	validator, err := NewMiniAppValidator(officialMiniAppToken, time.Hour)
	if err != nil {
		t.Fatalf("NewMiniAppValidator() error = %v", err)
	}

	now := time.Unix(1733509682, 0).Add(time.Minute)
	data, err := validator.ValidateAt(officialMiniAppData, now)
	if err != nil {
		t.Fatalf("ValidateAt() error = %v", err)
	}
	if data.User == nil {
		t.Fatal("user = nil")
	}
	if data.User.ID != 279058397 || data.User.FirstName != "Vladislav + - ? /" {
		t.Errorf("user = %#v", data.User)
	}
	if data.ChatType != "private" || data.ChatInstance != "8134722200314281151" {
		t.Errorf("chat context = %q, %q", data.ChatType, data.ChatInstance)
	}
}

func TestMiniAppSignatureValidatorOfficialVector(t *testing.T) {
	t.Parallel()

	validator, err := NewMiniAppSignatureValidator(
		officialMiniAppBotID,
		MiniAppEnvironmentProduction,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("NewMiniAppSignatureValidator() error = %v", err)
	}

	now := time.Unix(1733509682, 0).Add(time.Minute)
	data, err := validator.ValidateAt(officialMiniAppData, now)
	if err != nil {
		t.Fatalf("ValidateAt() error = %v", err)
	}
	if data.User == nil || data.User.Username != "vdkfrost" {
		t.Errorf("user = %#v", data.User)
	}
}

func TestMiniAppValidatorRejectsTamperingAndStaleData(t *testing.T) {
	t.Parallel()

	validator, err := NewMiniAppValidator(officialMiniAppToken, time.Hour)
	if err != nil {
		t.Fatalf("NewMiniAppValidator() error = %v", err)
	}
	authDate := time.Unix(1733509682, 0)

	tests := map[string]struct {
		raw     string
		now     time.Time
		wantErr error
	}{
		"tampered": {
			raw:     strings.Replace(officialMiniAppData, "279058397", "279058398", 1),
			now:     authDate.Add(time.Minute),
			wantErr: ErrInvalidMiniAppInitData,
		},
		"expired": {
			raw:     officialMiniAppData,
			now:     authDate.Add(2 * time.Hour),
			wantErr: ErrExpiredMiniAppInitData,
		},
		"future": {
			raw:     officialMiniAppData,
			now:     authDate.Add(-time.Minute),
			wantErr: ErrFutureMiniAppInitData,
		},
		"duplicate": {
			raw:     officialMiniAppData + "&auth_date=1733509682",
			now:     authDate,
			wantErr: ErrInvalidMiniAppInitData,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := validator.ValidateAt(test.raw, test.now)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateAt() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestMiniAppValidatorMiddleware(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	raw := signMiniAppData(t, testBotToken, url.Values{
		"auth_date":      []string{strconv.FormatInt(now.Unix(), 10)},
		"can_send_after": []string{"15"},
		"query_id":       []string{"query_123"},
		"user":           []string{`{"id":123,"first_name":"Ada"}`},
	})
	validator, err := NewMiniAppValidator(testBotToken, DefaultMiniAppInitDataMaxAge)
	if err != nil {
		t.Fatalf("NewMiniAppValidator() error = %v", err)
	}

	handler := validator.Middleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		data, ok := MiniAppInitDataFromContext(request.Context())
		if !ok {
			t.Error("MiniAppInitDataFromContext() did not find data")
			return
		}
		if data.QueryID != "query_123" || data.User == nil || data.User.ID != 123 {
			t.Errorf("init data = %#v", data)
		}
		if data.CanSendAfter != 15*time.Second {
			t.Errorf("CanSendAfter = %v", data.CanSendAfter)
		}
		response.WriteHeader(http.StatusNoContent)
	}))

	response := httptestResponse(handler, raw)
	if response.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestMiniAppValidatorConstructors(t *testing.T) {
	t.Parallel()

	if _, err := NewMiniAppValidator("", time.Minute); !errors.Is(err, ErrInvalidMiniAppInitData) {
		t.Errorf("NewMiniAppValidator() error = %v", err)
	}
	if _, err := NewMiniAppValidator("token", 0); !errors.Is(err, ErrInvalidMiniAppInitData) {
		t.Errorf("NewMiniAppValidator() error = %v", err)
	}
	if _, err := NewMiniAppSignatureValidator(0, MiniAppEnvironmentProduction, time.Minute); !errors.Is(
		err,
		ErrInvalidMiniAppInitData,
	) {
		t.Errorf("NewMiniAppSignatureValidator() error = %v", err)
	}
	if _, err := NewMiniAppSignatureValidator(1, MiniAppEnvironment(99), time.Minute); !errors.Is(
		err,
		ErrInvalidMiniAppInitData,
	) {
		t.Errorf("NewMiniAppSignatureValidator() error = %v", err)
	}
}

func signMiniAppData(t *testing.T, token string, values url.Values) string {
	t.Helper()

	secret := hmac.New(sha256.New, []byte(webAppDataKey))
	if _, err := secret.Write([]byte(token)); err != nil {
		t.Fatalf("derive Mini App secret: %v", err)
	}
	digest := hmac.New(sha256.New, secret.Sum(nil))
	if _, err := digest.Write([]byte(miniAppDataCheckString(values, "hash"))); err != nil {
		t.Fatalf("sign Mini App data: %v", err)
	}
	values.Set("hash", hex.EncodeToString(digest.Sum(nil)))

	return values.Encode()
}

func httptestResponse(handler http.Handler, raw string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/mini-app", nil)
	if raw != "" {
		request.Header.Set(MiniAppInitDataHeader, raw)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}
