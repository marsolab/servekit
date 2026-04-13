package httpkit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cep21/circuit/v3"
	"github.com/cep21/circuit/v3/closers/simplelogic"
	td "github.com/maxatome/go-testdeep"

	"github.com/marsolab/servekit/retry"
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

func newTestCircuit(t *testing.T, name string, errorThreshold int64) *circuit.Circuit {
	t.Helper()

	return circuit.NewCircuitFromConfig(name, circuit.Config{
		General: circuit.GeneralConfig{
			ClosedToOpenFactory: simplelogic.ConsecutiveErrOpenerFactory(simplelogic.ConfigConsecutiveErrOpener{
				ErrorThreshold: errorThreshold,
			}),
		},
		Execution: circuit.ExecutionConfig{
			Timeout:               5 * time.Second,
			MaxConcurrentRequests: 100,
		},
	})
}

func TestNewClient_WithCircuitBreaker(t *testing.T) {
	cb := newTestCircuit(t, "TestNewClient_WithCircuitBreaker", 1)

	client := NewClient(WithCircuitBreaker(cb))

	td.CmpNot(t, client, nil, "client with circuit breaker should not be nil.")
}

func TestRoundTripper_CircuitBreaker_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := newTestCircuit(t, "TestRoundTripper_CircuitBreaker_Success", 1)
	client := NewClient(WithCircuitBreaker(cb))

	resp, err := client.Get(server.URL)

	td.CmpNoError(t, err, "request should succeed through circuit breaker.")
	td.CmpNot(t, resp, nil, "response should not be nil.")
	td.Cmp(t, resp.StatusCode, http.StatusOK, "status code should be 200.")

	resp.Body.Close()
}

func TestRoundTripper_CircuitBreaker_ServerError_OpensCircuit(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cb := newTestCircuit(t, "TestRoundTripper_CircuitBreaker_ServerError_OpensCircuit", 1)
	client := NewClient(
		WithCircuitBreaker(cb),
		// Use a tiny retry config so the retry loop's backoff path has a Backoff.
		WithRetries(
			retry.WithMaxAttempts(0),
			retry.WithBackoff(retry.StaticBackoff(0)),
		),
	)

	// First request: server returns 500, circuit records a failure.
	resp1, err1 := client.Get(server.URL)
	td.CmpNoError(t, err1, "first request should return response even on 500.")
	td.Cmp(t, resp1.StatusCode, http.StatusInternalServerError, "first response should be 500.")
	resp1.Body.Close()

	// Second request: circuit should be open, request should fail fast
	// without reaching the server.
	beforeCount := atomic.LoadInt64(&requestCount)

	_, err2 := client.Get(server.URL)
	td.CmpError(t, err2, "second request should fail with circuit open error.")

	// Verify the server was not contacted on the second call.
	afterCount := atomic.LoadInt64(&requestCount)
	td.Cmp(t, afterCount, beforeCount, "server should not be contacted when circuit is open.")

	// Verify the error is a circuit error indicating open state.
	var cbErr circuit.Error
	td.CmpTrue(t, errors.As(err2, &cbErr), "error should be circuit.Error.")
	td.CmpTrue(t, cbErr.CircuitOpen(), "error should indicate circuit is open.")
}

func TestRoundTripper_CircuitBreaker_ClientError_NoOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Circuit opens after 1 consecutive failure. 4xx should not count as a failure.
	cb := newTestCircuit(t, "TestRoundTripper_CircuitBreaker_ClientError_NoOpen", 1)
	client := NewClient(WithCircuitBreaker(cb))

	// Make several 4xx requests.
	for range 5 {
		resp, err := client.Get(server.URL)
		td.CmpNoError(t, err, "4xx response should not produce circuit error.")
		td.Cmp(t, resp.StatusCode, http.StatusBadRequest, "response should be 400.")
		resp.Body.Close()
	}

	// Circuit should still be closed -- verify by making another request.
	resp, err := client.Get(server.URL)
	td.CmpNoError(t, err, "circuit should still be closed after 4xx responses.")
	td.Cmp(t, resp.StatusCode, http.StatusBadRequest, "response should be 400.")
	resp.Body.Close()
}

func TestRoundTripper_CircuitBreaker_OpenCircuit_StopsRetries(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cb := newTestCircuit(t, "TestRoundTripper_CircuitBreaker_OpenCircuit_StopsRetries", 1)
	client := NewClient(
		WithCircuitBreaker(cb),
		WithRetries(
			retry.WithMaxAttempts(5),
			retry.WithBackoff(retry.StaticBackoff(0)),
		),
	)

	resp, err := client.Get(server.URL)
	// We expect either a circuit-open error or a final 500 response.
	if err == nil {
		resp.Body.Close()
	}

	// The server should have been contacted at most once before the circuit opened.
	// With ErrorThreshold=1, after the first 500 the circuit opens and subsequent
	// retries fail fast without contacting the server.
	count := atomic.LoadInt64(&requestCount)
	td.CmpLt(t, count, int64(5), "retries should stop once circuit opens.")
}

func TestRoundTripper_CircuitBreaker_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := newTestCircuit(t, "TestRoundTripper_CircuitBreaker_ContextCanceled", 1)
	client := NewClient(WithCircuitBreaker(cb))

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
