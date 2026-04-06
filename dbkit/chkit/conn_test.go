package chkit

import (
	"context"
	"errors"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/maxatome/go-testdeep/td"
)

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
