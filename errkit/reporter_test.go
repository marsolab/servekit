package errkit

import (
	"errors"
	"sync"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

// mockReporter is a test double for ErrorReporter.
type mockReporter struct {
	mu       sync.Mutex
	reported []error
}

func (m *mockReporter) Report(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reported = append(m.reported, err)
}

func (m *mockReporter) calls() []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.reported
}

// TestReporter tests RegisterReporter and Report in the order they must
// execute within a single test binary (sync.Once allows only one registration).
func TestReporter(t *testing.T) {
	// Step 1: Report with nil reporter should be a no-op (no panic).
	t.Run("Report_with_nil_reporter", func(t *testing.T) {
		td.CmpNotPanic(t, func() {
			Report(errors.New("should be ignored"))
		})
	})

	// Step 2: Register a reporter successfully.
	mock := &mockReporter{}

	t.Run("RegisterReporter_first_call_succeeds", func(t *testing.T) {
		err := RegisterReporter(mock)
		td.CmpNoError(t, err)
	})

	// Step 3: Report should forward the error to the registered reporter.
	t.Run("Report_delegates_to_reporter", func(t *testing.T) {
		testErr := errors.New("test error")
		Report(testErr)

		calls := mock.calls()
		td.Cmp(t, len(calls), 1)
		td.Cmp(t, calls[0], testErr)
	})

	// Step 4: Report with multiple errors.
	t.Run("Report_multiple_errors", func(t *testing.T) {
		err1 := errors.New("error one")
		err2 := errors.New("error two")

		Report(err1)
		Report(err2)

		calls := mock.calls()
		td.Cmp(t, len(calls), 3) // 1 from previous test + 2 new.
	})

	// Step 5: Second registration attempt should fail.
	t.Run("RegisterReporter_second_call_returns_error", func(t *testing.T) {
		another := &mockReporter{}
		err := RegisterReporter(another)
		td.CmpError(t, err)
		td.Cmp(t, err.Error(), "reporter already registered")
	})
}
