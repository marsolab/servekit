package kinde

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/marsolab/servekit/errkit"
	"github.com/maxatome/go-testdeep/td"
)

func TestManagementClient_GetUser(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "mgmt-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})

	mux.HandleFunc("GET /api/v1/user", func(w http.ResponseWriter, r *http.Request) {
		td.Cmp(t, r.Header.Get("Authorization"), "Bearer mgmt-token")

		userID := r.URL.Query().Get("id")
		td.Cmp(t, userID, "kp_user123")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{ //nolint:errcheck // Test helper.
			ID:        "kp_user123",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := &Client{
		domain:       server.URL,
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		oidc: &OIDCConfig{
			TokenEndpoint: server.URL + "/oauth2/token",
		},
	}

	mc := client.NewManagementClient()

	user, err := mc.GetUser(context.Background(), "kp_user123")
	td.CmpNil(t, err)
	td.CmpNotNil(t, user)
	td.Cmp(t, user.ID, "kp_user123")
	td.Cmp(t, user.Email, "test@example.com")
	td.Cmp(t, user.FirstName, "Test")
}

func TestManagementClient_ListUsers(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "mgmt-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})

	mux.HandleFunc("GET /api/v1/users", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck // Test helper.
			"users": []User{
				{ID: "kp_user1", Email: "user1@example.com"},
				{ID: "kp_user2", Email: "user2@example.com"},
			},
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := &Client{
		domain:       server.URL,
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		oidc: &OIDCConfig{
			TokenEndpoint: server.URL + "/oauth2/token",
		},
	}

	mc := client.NewManagementClient()

	users, err := mc.ListUsers(context.Background())
	td.CmpNil(t, err)
	td.Cmp(t, len(users), 2)
	td.Cmp(t, users[0].ID, "kp_user1")
	td.Cmp(t, users[1].ID, "kp_user2")
}

func TestManagementClient_GetOrganization(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "mgmt-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})

	mux.HandleFunc("GET /api/v1/organization", func(w http.ResponseWriter, r *http.Request) {
		orgCode := r.URL.Query().Get("code")
		td.Cmp(t, orgCode, "org_abc")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck // Test helper.
			"organization": Organization{
				Code:   "org_abc",
				Name:   "Test Org",
				Handle: "test-org",
			},
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := &Client{
		domain:       server.URL,
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		oidc: &OIDCConfig{
			TokenEndpoint: server.URL + "/oauth2/token",
		},
	}

	mc := client.NewManagementClient()

	org, err := mc.GetOrganization(context.Background(), "org_abc")
	td.CmpNil(t, err)
	td.CmpNotNil(t, org)
	td.Cmp(t, org.Code, "org_abc")
	td.Cmp(t, org.Name, "Test Org")
}

func TestManagementClient_NotFound(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "mgmt-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})

	mux.HandleFunc("GET /api/v1/user", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := &Client{
		domain:       server.URL,
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		oidc: &OIDCConfig{
			TokenEndpoint: server.URL + "/oauth2/token",
		},
	}

	mc := client.NewManagementClient()

	_, err := mc.GetUser(context.Background(), "nonexistent")
	td.CmpNotNil(t, err)
	td.Cmp(t, err, td.ErrorIs(errkit.ErrNotFound))
}

func TestMapHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"not found", http.StatusNotFound, errkit.ErrNotFound},
		{"unauthorized", http.StatusUnauthorized, errkit.ErrUnauthenticated},
		{"forbidden", http.StatusForbidden, errkit.ErrUnauthorized},
		{"conflict", http.StatusConflict, errkit.ErrAlreadyExists},
		{"bad request", http.StatusBadRequest, errkit.ErrInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapHTTPError(tt.statusCode)
			td.Cmp(t, err, td.ErrorIs(tt.wantErr))
		})
	}

	t.Run("unknown status", func(t *testing.T) {
		err := mapHTTPError(http.StatusServiceUnavailable)
		td.CmpNotNil(t, err)
		td.CmpContains(t, err.Error(), "unexpected status 503")
	})
}
