package clickconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Option configures the ClickHouse client options.
type Option func(options *clickhouse.Options)

// WithAuth configures the authentication credentials for the ClickHouse client.
func WithAuth(database, username, password string) Option {
	return func(options *clickhouse.Options) {
		options.Auth = clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		}
	}
}

// WithConnectionPool configures the connection pool for high-concurrency scenarios.
// For high-throughput applications (1000+ concurrent requests), increase pool size accordingly.
// Recommended settings:
//   - maxOpenConns: Maximum number of open connections (e.g., 100-200 for high load)
//   - maxIdleConns: Maximum number of idle connections to keep (e.g., 50-100)
func WithConnectionPool(maxOpenConns, maxIdleConns int) Option {
	return func(options *clickhouse.Options) {
		if maxOpenConns > 0 {
			options.MaxOpenConns = maxOpenConns
		}

		if maxIdleConns > 0 {
			options.MaxIdleConns = maxIdleConns
		}
	}
}

// WithAsyncInsert enables asynchronous inserts on the server side.
// This reduces write IOPS by batching multiple insert operations on the ClickHouse server.
// Async inserts replace client-side batching - the server handles all buffering and flushing.
// Recommended settings:
//   - asyncInsert: true (enable feature)
//   - waitForAsyncInsert: true (wait for server-side flush, ensures durability)
//   - maxDataSize: buffer size in bytes before flush (e.g., 10485760 = 10MB)
//   - busyTimeout: max time in ms to wait before flush (e.g., 5000 = 5 seconds)
func WithAsyncInsert(asyncInsert, waitForAsyncInsert bool, maxDataSize, busyTimeout int) Option {
	return func(options *clickhouse.Options) {
		if options.Settings == nil {
			options.Settings = clickhouse.Settings{}
		}

		// Enable async insert feature.
		if asyncInsert {
			options.Settings["async_insert"] = 1
		} else {
			options.Settings["async_insert"] = 0
		}

		// Wait for async insert to complete (recommended for durability).
		if waitForAsyncInsert {
			options.Settings["wait_for_async_insert"] = 1
		} else {
			options.Settings["wait_for_async_insert"] = 0
		}

		// Set buffer size threshold for flush.
		if maxDataSize > 0 {
			options.Settings["async_insert_max_data_size"] = maxDataSize
		}

		// Set time threshold for flush.
		if busyTimeout > 0 {
			options.Settings["async_insert_busy_timeout_ms"] = busyTimeout
		}
	}
}

// New establishes connection with Clickhouse server.
// Returns a pointer to a new instance of Conn struct.
func New(addr string, opts ...Option) (*Conn, error) {
	options := clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	}

	for _, opt := range opts {
		opt(&options)
	}

	conn, err := clickhouse.Open(&options)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: open connection to database: %w", err)
	}

	ctx := context.Background()

	if err := conn.Ping(ctx); err != nil {
		var exception *clickhouse.Exception

		if errors.As(err, &exception) {
			return nil, fmt.Errorf("clickhouse: [%d] %s \n%s", exception.Code, exception.Message, exception.StackTrace)
		}

		return nil, fmt.Errorf("clickhouse: %w", err)
	}

	return &Conn{Conn: conn}, nil
}

// NewDBSQL establishes connection with Clickhouse server.
// Returns a pointer to a new instance of sql.DB.
// Slower than New. Used mostly for backward compatibility with
// sql.DB compatible interfaces.
func NewDBSQL(addr string, opts ...Option) (*sql.DB, error) {
	options := clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	}

	for _, opt := range opts {
		opt(&options)
	}

	db := clickhouse.OpenDB(&options)

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse: open connection to database: %w", err)
	}

	return db, nil
}

// Conn represents connection to ClickHouse database.
type Conn struct {
	atomic.Value
	driver.Conn
}

func (c *Conn) MarkUnhealthy(err error) { c.Store(err) }
func (c *Conn) MarkHealthy()            { c.Store(nil) }

func (c *Conn) Health(ctx context.Context) error {
	if v := c.Load(); v != nil {
		if err, ok := v.(error); ok && err != nil {
			return fmt.Errorf("clickhouse: health check failed: %w", err)
		}
	}

	if err := c.Ping(ctx); err != nil {
		var exception *clickhouse.Exception

		if errors.As(err, &exception) {
			return fmt.Errorf("clickhouse: health check failed: [%d] %s \n%s",
				exception.Code, exception.Message, exception.StackTrace,
			)
		}

		return fmt.Errorf("clickhouse: health check failed: %w", err)
	}

	return nil
}
