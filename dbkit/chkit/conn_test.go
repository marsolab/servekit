package chkit

import (
	"context"
	"errors"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/maxatome/go-testdeep/td"
)

// mockConn implements driver.Conn for testing purposes.
type mockConn struct {
	driver.Conn
	pingErr error
}

func (m *mockConn) Ping(_ context.Context) error { return m.pingErr }

func TestWithAuth(t *testing.T) {
	tests := map[string]struct {
		database string
		username string
		password string
		wantAuth clickhouse.Auth
	}{
		"all fields": {
			database: "analytics",
			username: "admin",
			password: "secret",
			wantAuth: clickhouse.Auth{Database: "analytics", Username: "admin", Password: "secret"},
		},
		"empty fields": {
			database: "",
			username: "",
			password: "",
			wantAuth: clickhouse.Auth{Database: "", Username: "", Password: ""},
		},
		"only database": {
			database: "metrics",
			username: "",
			password: "",
			wantAuth: clickhouse.Auth{Database: "metrics", Username: "", Password: ""},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var opts clickhouse.Options
			WithAuth(tc.database, tc.username, tc.password)(&opts)
			td.Cmp(t, opts.Auth, tc.wantAuth)
		})
	}
}

func TestWithConnectionPool(t *testing.T) {
	tests := map[string]struct {
		maxOpen     int
		maxIdle     int
		wantOpen    int
		wantIdle    int
	}{
		"both positive": {
			maxOpen:  100,
			maxIdle:  50,
			wantOpen: 100,
			wantIdle: 50,
		},
		"zero values are ignored": {
			maxOpen:  0,
			maxIdle:  0,
			wantOpen: 0,
			wantIdle: 0,
		},
		"negative values are ignored": {
			maxOpen:  -1,
			maxIdle:  -5,
			wantOpen: 0,
			wantIdle: 0,
		},
		"only maxOpen": {
			maxOpen:  200,
			maxIdle:  0,
			wantOpen: 200,
			wantIdle: 0,
		},
		"only maxIdle": {
			maxOpen:  0,
			maxIdle:  75,
			wantOpen: 0,
			wantIdle: 75,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var opts clickhouse.Options
			WithConnectionPool(tc.maxOpen, tc.maxIdle)(&opts)
			td.Cmp(t, opts.MaxOpenConns, tc.wantOpen)
			td.Cmp(t, opts.MaxIdleConns, tc.wantIdle)
		})
	}
}

func TestWithAsyncInsert(t *testing.T) {
	tests := map[string]struct {
		opts         []AsyncInsertOption
		wantSettings clickhouse.Settings
	}{
		"all options": {
			opts: []AsyncInsertOption{
				WaitForInsert(),
				MaxDataSize(10485760),
				BusyTimeout(5000),
			},
			wantSettings: clickhouse.Settings{
				"async_insert":                 1,
				"wait_for_async_insert":        1,
				"async_insert_max_data_size":   10485760,
				"async_insert_busy_timeout_ms": 5000,
			},
		},
		"no options": {
			opts: nil,
			wantSettings: clickhouse.Settings{
				"async_insert":          1,
				"wait_for_async_insert": 0,
			},
		},
		"only wait for insert": {
			opts: []AsyncInsertOption{WaitForInsert()},
			wantSettings: clickhouse.Settings{
				"async_insert":          1,
				"wait_for_async_insert": 1,
			},
		},
		"only data size and timeout": {
			opts: []AsyncInsertOption{
				MaxDataSize(1048576),
				BusyTimeout(3000),
			},
			wantSettings: clickhouse.Settings{
				"async_insert":                 1,
				"wait_for_async_insert":        0,
				"async_insert_max_data_size":   1048576,
				"async_insert_busy_timeout_ms": 3000,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var opts clickhouse.Options
			WithAsyncInsert(tc.opts...)(&opts)
			td.Cmp(t, opts.Settings, tc.wantSettings)
		})
	}
}

func TestWithAsyncInsert_PreservesExistingSettings(t *testing.T) {
	var opts clickhouse.Options
	opts.Settings = clickhouse.Settings{"existing_key": "value"}

	WithAsyncInsert()(&opts)

	td.Cmp(t, opts.Settings["existing_key"], "value")
	td.Cmp(t, opts.Settings["async_insert"], 1)
}

func TestConn_MarkUnhealthy(t *testing.T) {
	c := &Conn{}
	testErr := errors.New("connection lost")

	c.MarkUnhealthy(testErr)

	v := c.health.Load()
	td.CmpNotNil(t, v)

	box, ok := v.(errBox)
	td.CmpTrue(t, ok)
	td.CmpString(t, box.err.Error(), "connection lost")
}

func TestConn_MarkHealthy(t *testing.T) {
	c := &Conn{}
	c.MarkUnhealthy(errors.New("temporary error"))

	c.MarkHealthy()

	v := c.health.Load()
	box, ok := v.(errBox)
	td.CmpTrue(t, ok)
	td.CmpNil(t, box.err)
}

func TestConn_Health_MarkedUnhealthy(t *testing.T) {
	c := &Conn{}
	c.MarkUnhealthy(errors.New("disk full"))

	err := c.Health(context.Background())

	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "health check failed")
	td.CmpContains(t, err.Error(), "disk full")
}

func TestConn_MarkHealthy_ThenHealth(t *testing.T) {
	// Verify that after marking unhealthy and then healthy, the health
	// check no longer returns an error from the stored value.
	c := &Conn{}
	c.MarkUnhealthy(errors.New("temporary failure"))

	// Confirm it is unhealthy first.
	err := c.Health(context.Background())
	td.CmpNotNil(t, err)

	// Mark healthy and check again. Health will still fail because
	// Conn.Conn is nil (Ping will fail), but the stored errBox should be clear.
	c.MarkHealthy()

	v := c.health.Load()
	td.CmpNotNil(t, v)

	box, ok := v.(errBox)
	td.CmpTrue(t, ok)
	td.CmpNil(t, box.err)
}

func TestConn_Health_NoStoredValue(t *testing.T) {
	// When no value has been stored in health, Load returns nil.
	// The health check should skip the stored-error branch.
	c := &Conn{}

	v := c.health.Load()
	td.CmpNil(t, v)
}

func TestWaitForInsert(t *testing.T) {
	var cfg AsyncInsertConfig
	WaitForInsert()(&cfg)
	td.CmpTrue(t, cfg.waitForInsert)
}

func TestMaxDataSize(t *testing.T) {
	var cfg AsyncInsertConfig
	MaxDataSize(10485760)(&cfg)
	td.Cmp(t, cfg.maxDataSize, 10485760)
}

func TestBusyTimeout(t *testing.T) {
	var cfg AsyncInsertConfig
	BusyTimeout(5000)(&cfg)
	td.Cmp(t, cfg.busyTimeout, 5000)
}

func TestAsyncInsertOptions_Combined(t *testing.T) {
	// Verify that multiple options compose correctly.
	var cfg AsyncInsertConfig
	WaitForInsert()(&cfg)
	MaxDataSize(2048)(&cfg)
	BusyTimeout(1000)(&cfg)

	td.CmpTrue(t, cfg.waitForInsert)
	td.Cmp(t, cfg.maxDataSize, 2048)
	td.Cmp(t, cfg.busyTimeout, 1000)
}

func TestAsyncInsertOptions_ZeroValues(t *testing.T) {
	// MaxDataSize and BusyTimeout with zero should set zero.
	var cfg AsyncInsertConfig
	MaxDataSize(0)(&cfg)
	BusyTimeout(0)(&cfg)

	td.Cmp(t, cfg.maxDataSize, 0)
	td.Cmp(t, cfg.busyTimeout, 0)
}

func TestWithAsyncInsert_OnlyMaxDataSize(t *testing.T) {
	var opts clickhouse.Options
	WithAsyncInsert(MaxDataSize(5242880))(&opts)

	td.Cmp(t, opts.Settings["async_insert"], 1)
	td.Cmp(t, opts.Settings["wait_for_async_insert"], 0)
	td.Cmp(t, opts.Settings["async_insert_max_data_size"], 5242880)
	// busy_timeout should not be set.
	_, hasBusyTimeout := opts.Settings["async_insert_busy_timeout_ms"]
	td.CmpFalse(t, hasBusyTimeout)
}

func TestWithAsyncInsert_OnlyBusyTimeout(t *testing.T) {
	var opts clickhouse.Options
	WithAsyncInsert(BusyTimeout(2000))(&opts)

	td.Cmp(t, opts.Settings["async_insert"], 1)
	td.Cmp(t, opts.Settings["wait_for_async_insert"], 0)
	td.Cmp(t, opts.Settings["async_insert_busy_timeout_ms"], 2000)
	// max_data_size should not be set.
	_, hasMaxDataSize := opts.Settings["async_insert_max_data_size"]
	td.CmpFalse(t, hasMaxDataSize)
}

func TestErrBox(t *testing.T) {
	// Verify errBox wraps nil error correctly.
	box := errBox{}
	td.CmpNil(t, box.err)

	// Verify errBox wraps non-nil error correctly.
	testErr := errors.New("test error")
	box = errBox{err: testErr}
	td.Cmp(t, box.err.Error(), "test error")
}

func TestConn_Health_PingSuccess(t *testing.T) {
	// When no stored error and Ping succeeds, Health returns nil.
	c := &Conn{Conn: &mockConn{pingErr: nil}}

	err := c.Health(context.Background())
	td.CmpNil(t, err)
}

func TestConn_Health_PingError(t *testing.T) {
	// When Ping returns a generic error, Health wraps it.
	c := &Conn{Conn: &mockConn{pingErr: errors.New("connection refused")}}

	err := c.Health(context.Background())
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "health check failed")
	td.CmpContains(t, err.Error(), "connection refused")
}

func TestConn_Health_PingClickHouseException(t *testing.T) {
	// When Ping returns a ClickHouse exception, Health formats it.
	exc := &clickhouse.Exception{
		Code:       999,
		Message:    "test exception",
		StackTrace: "stack trace here",
	}
	c := &Conn{Conn: &mockConn{pingErr: exc}}

	err := c.Health(context.Background())
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "health check failed")
	td.CmpContains(t, err.Error(), "999")
	td.CmpContains(t, err.Error(), "test exception")
}

func TestConn_Health_MarkedHealthyThenPingSuccess(t *testing.T) {
	// After marking healthy, if Ping succeeds, Health returns nil.
	c := &Conn{Conn: &mockConn{pingErr: nil}}
	c.MarkUnhealthy(errors.New("temporary"))
	c.MarkHealthy()

	err := c.Health(context.Background())
	td.CmpNil(t, err)
}

func TestConn_Health_MarkedHealthyThenPingFails(t *testing.T) {
	// After marking healthy, if Ping fails, Health returns the ping error.
	c := &Conn{Conn: &mockConn{pingErr: errors.New("network down")}}
	c.MarkUnhealthy(errors.New("temporary"))
	c.MarkHealthy()

	err := c.Health(context.Background())
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "network down")
}

func TestConn_MarkUnhealthy_Overwrite(t *testing.T) {
	// Marking unhealthy multiple times should overwrite the previous error.
	c := &Conn{}
	c.MarkUnhealthy(errors.New("first error"))
	c.MarkUnhealthy(errors.New("second error"))

	v := c.health.Load()
	box, ok := v.(errBox)
	td.CmpTrue(t, ok)
	td.Cmp(t, box.err.Error(), "second error")
}

func TestNew_ConnectionFailure(t *testing.T) {
	// Connecting to a non-existent address should return an error.
	_, err := New("127.0.0.1:19999",
		WithAuth("default", "default", ""),
	)
	td.CmpNotNil(t, err)
}

func TestNewDBSQL_ConnectionFailure(t *testing.T) {
	// Connecting to a non-existent address should return an error.
	_, err := NewDBSQL("127.0.0.1:19999",
		WithAuth("default", "default", ""),
	)
	td.CmpNotNil(t, err)
}
