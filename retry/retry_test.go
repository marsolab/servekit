package retry

import (
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
