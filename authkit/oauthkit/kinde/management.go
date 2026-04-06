package kinde

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/marsolab/servekit/errkit"
)

// ManagementClient provides access to the Kinde Management API.
// It uses the M2M client credentials flow for authentication.
type ManagementClient struct {
	client  *Client
	m2m     *M2MClient
	baseURL string
}

// NewManagementClient creates a management API client. It initializes
// an M2M client targeting the Kinde management API audience.
func (c *Client) NewManagementClient() *ManagementClient {
	return &ManagementClient{
		client:  c,
		m2m:     c.NewM2MClient(),
		baseURL: c.domain + "/api/v1",
	}
}

// User represents a Kinde user.
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`   //nolint:tagliatelle // Kinde API uses snake_case.
	LastName     string `json:"last_name"`    //nolint:tagliatelle // Kinde API uses snake_case.
	IsSuspended  bool   `json:"is_suspended"` //nolint:tagliatelle // Kinde API uses snake_case.
	Picture      string `json:"picture"`
	TotalSignIns int    `json:"total_sign_ins"` //nolint:tagliatelle // Kinde API uses snake_case.
	CreatedOn    string `json:"created_on"`     //nolint:tagliatelle // Kinde API uses snake_case.
}

// Organization represents a Kinde organization.
type Organization struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Handle string `json:"handle"`
}

type apiRequest struct {
	method string
	path   string
	body   any
	result any
}

// GetUser retrieves a user by their Kinde user ID.
func (mc *ManagementClient) GetUser(ctx context.Context, userID string) (*User, error) {
	var resp struct {
		User
	}

	path := "/user?id=" + url.QueryEscape(userID)
	if err := mc.do(ctx, apiRequest{method: http.MethodGet, path: path, result: &resp}); err != nil {
		return nil, err
	}

	return &resp.User, nil
}

// ListUsers retrieves a list of users.
func (mc *ManagementClient) ListUsers(ctx context.Context) ([]User, error) {
	var resp struct {
		Users []User `json:"users"`
	}

	if err := mc.do(ctx, apiRequest{method: http.MethodGet, path: "/users", result: &resp}); err != nil {
		return nil, err
	}

	return resp.Users, nil
}

// GetOrganization retrieves an organization by its code.
func (mc *ManagementClient) GetOrganization(ctx context.Context, orgCode string) (*Organization, error) {
	var resp struct {
		Organization Organization `json:"organization"`
	}

	r := apiRequest{method: http.MethodGet, path: "/organization?code=" + url.QueryEscape(orgCode), result: &resp}
	if err := mc.do(ctx, r); err != nil {
		return nil, err
	}

	return &resp.Organization, nil
}

// ListOrganizations retrieves a list of organizations.
func (mc *ManagementClient) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var resp struct {
		Organizations []Organization `json:"organizations"`
	}

	if err := mc.do(ctx, apiRequest{method: http.MethodGet, path: "/organizations", result: &resp}); err != nil {
		return nil, err
	}

	return resp.Organizations, nil
}

// do performs an authenticated HTTP request to the management API.
func (mc *ManagementClient) do(ctx context.Context, r apiRequest) error {
	token, err := mc.m2m.Token(ctx)
	if err != nil {
		return fmt.Errorf("kinde management: %w", err)
	}

	body, err := encodeBody(r.body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, r.method, mc.baseURL+r.path, body)
	if err != nil {
		return fmt.Errorf("kinde management: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := mc.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kinde management: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return mapHTTPError(resp.StatusCode)
	}

	if r.result != nil {
		if err := json.NewDecoder(resp.Body).Decode(r.result); err != nil {
			return fmt.Errorf("kinde management: decode response: %w", err)
		}
	}

	return nil
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return http.NoBody, nil
	}

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("kinde management: encode request body: %w", err)
	}

	return &buf, nil
}

func mapHTTPError(statusCode int) error {
	switch statusCode {
	case http.StatusNotFound:
		return errors.Join(errkit.ErrNotFound, fmt.Errorf("kinde management: status %d", statusCode))
	case http.StatusUnauthorized:
		return errors.Join(errkit.ErrUnauthenticated, fmt.Errorf("kinde management: status %d", statusCode))
	case http.StatusForbidden:
		return errors.Join(errkit.ErrUnauthorized, fmt.Errorf("kinde management: status %d", statusCode))
	case http.StatusConflict:
		return errors.Join(errkit.ErrAlreadyExists, fmt.Errorf("kinde management: status %d", statusCode))
	case http.StatusBadRequest:
		return errors.Join(errkit.ErrInvalidArgument, fmt.Errorf("kinde management: status %d", statusCode))
	default:
		return fmt.Errorf("kinde management: unexpected status %d", statusCode)
	}
}
