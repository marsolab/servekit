package httpkit

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	td "github.com/maxatome/go-testdeep"
)

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient()

	td.CmpNot(t, client, nil, "client should not be nil.")
	td.CmpNot(t, client.Transport, nil, "transport should not be nil.")
}

func TestNewClient_WithTLSHandshakeTimeout(t *testing.T) {
	client := NewClient(WithTLSHandshakeTimeout(10 * time.Second))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithDialTimeout(t *testing.T) {
	client := NewClient(WithDialTimeout(15 * time.Second))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithKeepAliveDisabled(t *testing.T) {
	client := NewClient(WithKeepAliveDisabled(true))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithKeepAliveTimeout(t *testing.T) {
	client := NewClient(WithKeepAliveTimeout(60 * time.Second))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithMaxIdleConns(t *testing.T) {
	client := NewClient(WithMaxIdleConns(200))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithMaxIdleConnsPerHost(t *testing.T) {
	client := NewClient(WithMaxIdleConnsPerHost(10))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithResponseHeaderTimeout(t *testing.T) {
	client := NewClient(WithResponseHeaderTimeout(5 * time.Second))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithWriteBufferSize(t *testing.T) {
	client := NewClient(WithWriteBufferSize(4096))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithReadBufferSize(t *testing.T) {
	client := NewClient(WithReadBufferSize(4096))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_WithCustomDialer(t *testing.T) {
	dialer := &mockDialer{}
	client := NewClient(WithCustomDialer(dialer))

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_MultipleOptions(t *testing.T) {
	client := NewClient(
		WithTLSHandshakeTimeout(10*time.Second),
		WithDialTimeout(15*time.Second),
		WithKeepAliveDisabled(false),
		WithMaxIdleConns(50),
		WithMaxIdleConnsPerHost(5),
		WithResponseHeaderTimeout(3*time.Second),
		WithWriteBufferSize(8192),
		WithReadBufferSize(8192),
	)

	td.CmpNot(t, client, nil, "client should not be nil.")
}

func TestNewClient_CanMakeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()

	resp, err := client.Get(server.URL)

	td.CmpNoError(t, err, "GET request should not error.")
	td.CmpNot(t, resp, nil, "response should not be nil.")
	td.Cmp(t, resp.StatusCode, http.StatusOK, "status code should be 200.")

	resp.Body.Close()
}

func TestNewClient_WithRetries(t *testing.T) {
	// Just verify the client can be constructed with the retry option.
	client := NewClient(WithRetries())

	td.CmpNot(t, client, nil, "client with retries should not be nil.")
}

func TestConfig_Transport(t *testing.T) {
	cfg := Config{
		defaultDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		tlsHandshakeTimeout:   5 * time.Second,
		disableKeepAlives:     false,
		maxIdleConns:          100,
		maxIdleConnsPerHost:   2,
		idleConnTimeout:       90 * time.Second,
		expectContinueTimeout: time.Second,
	}

	transport := cfg.transport()

	td.CmpNot(t, transport, nil, "transport should not be nil.")
	td.Cmp(t, transport.TLSHandshakeTimeout, 5*time.Second,
		"TLS handshake timeout should match.")
	td.Cmp(t, transport.DisableKeepAlives, false,
		"keep alives should be enabled.")
	td.Cmp(t, transport.MaxIdleConns, 100,
		"max idle conns should match.")
	td.Cmp(t, transport.MaxIdleConnsPerHost, 2,
		"max idle conns per host should match.")
	td.Cmp(t, transport.IdleConnTimeout, 90*time.Second,
		"idle conn timeout should match.")
	td.Cmp(t, transport.ExpectContinueTimeout, time.Second,
		"expect continue timeout should match.")
}

func TestConfig_DialContext_Default(t *testing.T) {
	cfg := Config{
		defaultDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}

	dialFn := cfg.dialContext()
	td.CmpNot(t, dialFn, nil, "dial context function should not be nil.")
}

func TestConfig_DialContext_CustomDialer(t *testing.T) {
	dialer := &mockDialer{}
	cfg := Config{
		customDialer: dialer,
		defaultDialer: &net.Dialer{
			Timeout: 30 * time.Second,
		},
	}

	dialFn := cfg.dialContext()
	td.CmpNot(t, dialFn, nil, "dial context function should not be nil with custom dialer.")
}

func TestConfig_Client(t *testing.T) {
	cfg := Config{
		defaultDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		tlsHandshakeTimeout:   5 * time.Second,
		maxIdleConns:          100,
		maxIdleConnsPerHost:   2,
		idleConnTimeout:       90 * time.Second,
		expectContinueTimeout: time.Second,
	}

	client := cfg.client()
	td.CmpNot(t, client, nil, "client should not be nil.")
	td.CmpNot(t, client.Transport, nil, "client transport should not be nil.")
}

func TestRoundTripper_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	td.CmpNoError(t, err, "creating request should not error.")

	_, err = client.Do(req)
	td.CmpError(t, err, "request with canceled context should error.")
}

// mockDialer implements CustomDialer for testing.
type mockDialer struct{}

func (d *mockDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, address)
}
