package retry

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/maxatome/go-testdeep/td"
)

func TestNewConstantBackoff(t *testing.T) {
	td.NewT(t)

	got := NewConstantBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)
	td.Cmp(t, got, td.Struct(&ConstantBackoff{}, td.StructFields{
		"minBackoffInterval": float64(50),
		"maxBackoffInterval": float64(time.Second / time.Millisecond),
		"maxJitterInterval":  float64(50),
	}))

	var _ Backoff = got
}

func Test_constantBackoff_Next(t *testing.T) {
	td.NewT(t)

	backoff := NewConstantBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)
	result := backoff.Next(1)

	td.Cmp(t, result, td.Gte(50*time.Millisecond))
	td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
}

func TestNewLinearBackoff(t *testing.T) {
	td.NewT(t)

	got := NewLinearBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)
	td.Cmp(t, got, td.Struct(&LinearBackoff{}, td.StructFields{
		"minBackoffInterval": float64(50),
		"maxBackoffInterval": float64(time.Second / time.Millisecond),
		"maxJitterInterval":  float64(50),
	}))

	var _ Backoff = got
}

func Test_linearBackoff_Next(t *testing.T) {
	td.NewT(t)

	backoff := NewLinearBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)

	t.Run("1 Retry", func(t *testing.T) {
		result := backoff.Next(1)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})

	t.Run("2 Retry", func(t *testing.T) {
		result := backoff.Next(2)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})

	t.Run("3 Retry", func(t *testing.T) {
		result := backoff.Next(3)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})
}

func TestNewExponentialBackoff(t *testing.T) {
	td.NewT(t)

	got := NewExponentialBackoff(2, 50*time.Millisecond, time.Second, 50*time.Millisecond)
	td.Cmp(t, got, td.Struct(&ExponentialBackoff{}, td.StructFields{
		"exponentialFactor":  float64(2),
		"minBackoffInterval": float64(50),
		"maxBackoffInterval": float64(time.Second / time.Millisecond),
		"maxJitterInterval":  float64(50),
	}))

	var _ Backoff = got
}

func Test_exponentialBackoff_Next(t *testing.T) {
	td.NewT(t)

	backoff := NewExponentialBackoff(2, 50*time.Millisecond, time.Second, 50*time.Millisecond)

	t.Run("1 Retry", func(t *testing.T) {
		result := backoff.Next(1)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})

	t.Run("2 Retry", func(t *testing.T) {
		result := backoff.Next(2)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})

	t.Run("3 Retry", func(t *testing.T) {
		result := backoff.Next(3)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})

	t.Run("10 retry", func(t *testing.T) {
		result := backoff.Next(10)
		td.Cmp(t, result, td.Gte(50*time.Millisecond))
		td.Cmp(t, result, td.Lte(time.Second+50*time.Millisecond))
	})
}

// --- Error type tests ---

func TestError_Error(t *testing.T) {
	td.Cmp(t, ErrRetryLimitReached.Error(), "retry limit reached")

	custom := Error("custom error")
	td.Cmp(t, custom.Error(), "custom error")
}

func TestErrRetryLimitReached(t *testing.T) {
	var err error = ErrRetryLimitReached
	td.Cmp(t, err.Error(), "retry limit reached")

	// Verify it satisfies the error interface.
	td.Cmp(t, errors.Is(err, ErrRetryLimitReached), true)
}

// --- RetryableError tests ---

func TestRetryableError_Error(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		inner := errors.New("something failed")
		rErr := RetryableError{err: inner}
		td.Cmp(t, rErr.Error(), "retry: something failed")
	})

	t.Run("with nil error", func(t *testing.T) {
		rErr := RetryableError{err: nil}
		td.Cmp(t, rErr.Error(), "retry: <nil>")
	})
}

func TestRetryableError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	rErr := RetryableError{err: inner}
	td.Cmp(t, rErr.Unwrap(), inner)
}

func TestRetryableError_ErrorsAs(t *testing.T) {
	inner := errors.New("wrapped")
	rErr := &RetryableError{err: inner}

	var target *RetryableError
	td.Cmp(t, errors.As(rErr, &target), true)
	td.Cmp(t, target.Unwrap(), inner)
}

// --- MarkRetryable tests ---

func TestMarkRetryable(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		result := MarkRetryable(nil)
		td.Cmp(t, result, nil)
	})

	t.Run("non-nil error returns RetryableError", func(t *testing.T) {
		inner := errors.New("fail")
		result := MarkRetryable(inner)

		td.Cmp(t, result, td.NotNil())

		var rErr *RetryableError
		td.Cmp(t, errors.As(result, &rErr), true)
		td.Cmp(t, rErr.Unwrap(), inner)
	})
}

// --- StaticBackoff tests ---

func TestStaticBackoff_Next(t *testing.T) {
	t.Run("returns fixed duration", func(t *testing.T) {
		b := StaticBackoff(100 * time.Millisecond)
		td.Cmp(t, b.Next(0), 100*time.Millisecond)
		td.Cmp(t, b.Next(1), 100*time.Millisecond)
		td.Cmp(t, b.Next(10), 100*time.Millisecond)
	})

	t.Run("zero duration", func(t *testing.T) {
		b := StaticBackoff(0)
		td.Cmp(t, b.Next(0), time.Duration(0))
		td.Cmp(t, b.Next(5), time.Duration(0))
	})

	t.Run("implements Backoff interface", func(t *testing.T) {
		var _ Backoff = StaticBackoff(0)
	})
}

// --- Backoff zero-retry edge cases ---

func TestBackoff_ZeroRetry(t *testing.T) {
	t.Run("ExponentialBackoff returns 0 for retry=0", func(t *testing.T) {
		b := NewExponentialBackoff(2, 50*time.Millisecond, time.Second, 50*time.Millisecond)
		td.Cmp(t, b.Next(0), 0*time.Millisecond)
	})

	t.Run("ConstantBackoff returns 0 for retry=0", func(t *testing.T) {
		b := NewConstantBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)
		td.Cmp(t, b.Next(0), 0*time.Millisecond)
	})

	t.Run("LinearBackoff returns 0 for retry=0", func(t *testing.T) {
		b := NewLinearBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)
		td.Cmp(t, b.Next(0), 0*time.Millisecond)
	})
}

// --- Option function tests ---

func TestWithMaxAttempts(t *testing.T) {
	o := Options{}
	WithMaxAttempts(5)(&o)
	td.Cmp(t, o.MaxRetries(), uint(5))
}

func TestWithBackoff(t *testing.T) {
	o := Options{}
	b := StaticBackoff(42 * time.Millisecond)
	WithBackoff(b)(&o)
	td.Cmp(t, o.Backoff(), b)
}

func TestWithLogger(t *testing.T) {
	o := Options{}
	logger := slog.Default()
	WithLogger(logger)(&o)
	td.Cmp(t, o.logger, logger)
}

func TestOptions_Getters(t *testing.T) {
	b := StaticBackoff(10 * time.Millisecond)
	o := Options{
		maxRetries: 7,
		backoff:    b,
	}
	td.Cmp(t, o.MaxRetries(), uint(7))
	td.Cmp(t, o.Backoff(), b)
}

// --- Do() tests ---

func TestDo_SuccessOnFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	})

	td.CmpNoError(t, err)
	td.Cmp(t, calls, 1)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return MarkRetryable(errors.New("transient"))
		}
		return nil
	})

	td.CmpNoError(t, err)
	td.Cmp(t, calls, 3)
}

func TestDo_NonRetryableErrorStopsImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("permanent failure")

	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return sentinel
	})

	td.Cmp(t, errors.Is(err, sentinel), true)
	td.Cmp(t, calls, 1)
}

func TestDo_RetryableErrorRetries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return MarkRetryable(errors.New("try again"))
	}, WithMaxAttempts(5))

	td.Cmp(t, errors.Is(err, ErrRetryLimitReached), true)
	td.Cmp(t, calls, 5)
}

func TestDo_ErrRetryLimitReached(t *testing.T) {
	err := Do(context.Background(), func(_ context.Context) error {
		return MarkRetryable(errors.New("always fails"))
	}, WithMaxAttempts(2))

	td.Cmp(t, errors.Is(err, ErrRetryLimitReached), true)
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Run("already cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := Do(ctx, func(_ context.Context) error {
			return MarkRetryable(errors.New("should not matter"))
		})

		td.Cmp(t, errors.Is(err, context.Canceled), true)
	})

	t.Run("context cancelled during backoff", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0

		err := Do(ctx, func(_ context.Context) error {
			calls++
			if calls == 1 {
				// Cancel during the backoff wait.
				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()
			}
			return MarkRetryable(errors.New("transient"))
		}, WithMaxAttempts(10), WithBackoff(StaticBackoff(5*time.Second)))

		td.Cmp(t, errors.Is(err, context.Canceled), true)
	})
}

func TestDo_DefaultOptions(t *testing.T) {
	// Default is 3 retries with StaticBackoff(0).
	calls := 0
	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return MarkRetryable(errors.New("fail"))
	})

	td.Cmp(t, errors.Is(err, ErrRetryLimitReached), true)
	td.Cmp(t, calls, 3)
}

func TestDo_WithCustomBackoff(t *testing.T) {
	calls := 0
	start := time.Now()

	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return MarkRetryable(errors.New("transient"))
		}
		return nil
	}, WithBackoff(StaticBackoff(10*time.Millisecond)), WithMaxAttempts(5))

	elapsed := time.Since(start)

	td.CmpNoError(t, err)
	td.Cmp(t, calls, 3)
	// Two retries with 10ms backoff each, so at least ~20ms should have elapsed.
	td.Cmp(t, elapsed, td.Gte(15*time.Millisecond))
}

func TestDo_WithLogger(t *testing.T) {
	// Ensure WithLogger does not cause a panic or error.
	logger := slog.Default()
	calls := 0

	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	}, WithLogger(logger))

	td.CmpNoError(t, err)
	td.Cmp(t, calls, 1)
}

func TestDo_SingleAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return MarkRetryable(errors.New("fail"))
	}, WithMaxAttempts(1))

	td.Cmp(t, errors.Is(err, ErrRetryLimitReached), true)
	td.Cmp(t, calls, 1)
}

func TestDo_ContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Do(ctx, func(_ context.Context) error {
		return MarkRetryable(errors.New("slow"))
	}, WithMaxAttempts(100), WithBackoff(StaticBackoff(time.Second)))

	td.Cmp(t, errors.Is(err, context.DeadlineExceeded), true)
}

func TestDo_ValueRetryableError(t *testing.T) {
	// Callers that construct RetryableError{...} value literals directly
	// should also be retried by Do.
	calls := 0
	err := Do(context.Background(), func(_ context.Context) error {
		calls++
		return RetryableError{err: errors.New("value retryable")}
	}, WithMaxAttempts(3))

	td.Cmp(t, errors.Is(err, ErrRetryLimitReached), true)
	td.Cmp(t, calls, 3)
}
