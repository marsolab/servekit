package chkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// defaultDatabase is the default ClickHouse database and username.
const defaultDatabase = "default"

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

// AsyncInsertConfig holds configuration for server-side asynchronous inserts.
type AsyncInsertConfig struct {
	waitForInsert bool
	maxDataSize   int
	busyTimeout   int
}

// AsyncInsertOption configures the async insert behavior.
type AsyncInsertOption func(c *AsyncInsertConfig)

// WaitForInsert enables waiting for server-side flush, ensuring durability.
func WaitForInsert() AsyncInsertOption {
	return func(c *AsyncInsertConfig) { c.waitForInsert = true }
}

// MaxDataSize sets the buffer size in bytes before flush (e.g., 10485760 = 10MB).
func MaxDataSize(size int) AsyncInsertOption {
	return func(c *AsyncInsertConfig) { c.maxDataSize = size }
}

// BusyTimeout sets the max time in ms to wait before flush (e.g., 5000 = 5 seconds).
func BusyTimeout(timeout int) AsyncInsertOption {
	return func(c *AsyncInsertConfig) { c.busyTimeout = timeout }
}

// WithAsyncInsert enables asynchronous inserts on the server side.
// This reduces write IOPS by batching multiple insert operations on the ClickHouse server.
// Async inserts replace client-side batching — the server handles all buffering and flushing.
func WithAsyncInsert(opts ...AsyncInsertOption) Option {
	var cfg AsyncInsertConfig
	for _, o := range opts {
		o(&cfg)
	}

	return func(options *clickhouse.Options) {
		if options.Settings == nil {
			options.Settings = clickhouse.Settings{}
		}

		options.Settings["async_insert"] = 1

		if cfg.waitForInsert {
			options.Settings["wait_for_async_insert"] = 1
		} else {
			options.Settings["wait_for_async_insert"] = 0
		}

		if cfg.maxDataSize > 0 {
			options.Settings["async_insert_max_data_size"] = cfg.maxDataSize
		}

		if cfg.busyTimeout > 0 {
			options.Settings["async_insert_busy_timeout_ms"] = cfg.busyTimeout
		}
	}
}

// New establishes connection with Clickhouse server.
// Returns a pointer to a new instance of Conn struct.
func New(addr string, opts ...Option) (*Conn, error) {
	options := clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: defaultDatabase,
			Username: defaultDatabase,
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
			Database: defaultDatabase,
			Username: defaultDatabase,
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

// errBox wraps an error so that both nil and non-nil errors share the same
// concrete type when stored in atomic.Value (which requires type consistency).
type errBox struct{ err error }

// Conn represents connection to ClickHouse database.
type Conn struct {
	health atomic.Value
	driver.Conn
}

func (c *Conn) MarkUnhealthy(err error) { c.health.Store(errBox{err: err}) }
func (c *Conn) MarkHealthy()            { c.health.Store(errBox{}) }

func (c *Conn) Health(ctx context.Context) error {
	if v := c.health.Load(); v != nil {
		if box, ok := v.(errBox); ok && box.err != nil {
			return fmt.Errorf("clickhouse: health check failed: %w", box.err)
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
