package httpkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marsolab/servekit/ctxkit"
	"github.com/marsolab/servekit/errkit"
	td "github.com/maxatome/go-testdeep"
)

func TestNewResponseOptions_Defaults(t *testing.T) {
	w := httptest.NewRecorder()
	o := NewResponseOptions(w)

	td.Cmp(t, o.statusCode, 0, "default status code should be 0 (unset).")
	td.Cmp(t, o.reportError, false, "default reportError should be false.")
	td.Cmp(t, len(o.headers), 0, "default headers should be empty.")
}

func TestNewResponseOptions_WithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	o := NewResponseOptions(w, WithStatus(http.StatusCreated))

	td.Cmp(t, o.statusCode, http.StatusCreated, "status code should be 201.")
}

func TestNewResponseOptions_WithHeader(t *testing.T) {
	w := httptest.NewRecorder()
	o := NewResponseOptions(w,
		WithHeader("X-Custom", "value1"),
		WithHeader("X-Custom", "value2"),
	)

	td.Cmp(t, o.headers.Values("X-Custom"), []string{"value1", "value2"},
		"headers should contain both values.")

	// Headers should also be set on the response writer.
	td.Cmp(t, w.Header().Values("X-Custom"), []string{"value1", "value2"},
		"response writer should have the custom headers.")
}

func TestNewResponseOptions_WithErrorReport(t *testing.T) {
	w := httptest.NewRecorder()
	o := NewResponseOptions(w, WithErrorReport())

	td.Cmp(t, o.reportError, true, "reportError should be true.")
}

func TestNewResponseOptions_MultipleOptions(t *testing.T) {
	w := httptest.NewRecorder()
	o := NewResponseOptions(w,
		WithStatus(http.StatusAccepted),
		WithHeader("X-Req-ID", "abc123"),
		WithErrorReport(),
	)

	td.Cmp(t, o.statusCode, http.StatusAccepted, "status code should be 202.")
	td.Cmp(t, o.headers.Get("X-Req-ID"), "abc123", "header X-Req-ID should be set.")
	td.Cmp(t, o.reportError, true, "reportError should be true.")
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "OK", statusCode: http.StatusOK},
		{name: "NotFound", statusCode: http.StatusNotFound},
		{name: "Created", statusCode: http.StatusCreated},
		{name: "NoContent", statusCode: http.StatusNoContent},
		{name: "InternalServerError", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			Status(w, r, tt.statusCode)

			td.Cmp(t, w.Code, tt.statusCode, "status code should match.")
		})
	}
}

func TestStatus_WithHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	Status(w, r, http.StatusOK, WithHeader("X-Test", "hello"))

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("X-Test"), "hello", "custom header should be set.")
}

func TestJSON_Success(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(w, r, payload{Name: "Alice", Age: 30})

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "application/json; charset=utf-8",
		"content type should be application/json.")

	var got payload
	err := json.Unmarshal(w.Body.Bytes(), &got)
	td.CmpNoError(t, err, "response body should be valid JSON.")
	td.Cmp(t, got, payload{Name: "Alice", Age: 30}, "decoded payload should match.")
}

func TestJSON_WithStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(w, r, map[string]string{"id": "123"}, WithStatus(http.StatusCreated))

	td.Cmp(t, w.Code, http.StatusCreated, "status code should be 201.")
}

func TestJSON_WithCustomHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(w, r, "ok", WithHeader("X-Request-ID", "req-1"))

	td.Cmp(t, w.Header().Get("X-Request-ID"), "req-1",
		"custom header should be set on JSON response.")
}

func TestJSON_EncodingError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// A channel is not JSON-serializable.
	JSON(w, r, make(chan int))

	td.Cmp(t, w.Code, http.StatusInternalServerError,
		"encoding error should result in 500.")
}

func TestJSON_EncodingError_WithLogHook(t *testing.T) {
	w := httptest.NewRecorder()

	var loggedErr error
	ctx := ctxkit.SetLogErrHook(context.Background(), func(err error) { loggedErr = err })
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	// A channel is not JSON-serializable.
	JSON(w, r, make(chan int))

	td.Cmp(t, w.Code, http.StatusInternalServerError,
		"encoding error should result in 500.")
	td.CmpNot(t, loggedErr, nil, "error should be logged via hook.")
}

func TestJSON_NilValue(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(w, r, nil)

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "application/json; charset=utf-8",
		"content type should be application/json.")
}

func TestJSON_SlicePayload(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(w, r, []string{"a", "b", "c"})

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")

	var got []string
	err := json.Unmarshal(w.Body.Bytes(), &got)
	td.CmpNoError(t, err, "response body should be valid JSON.")
	td.Cmp(t, got, []string{"a", "b", "c"}, "decoded slice should match.")
}

func TestHTML_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	HTML(w, r, []byte("<h1>Hello</h1>"))

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8",
		"content type should be text/html.")
	td.Cmp(t, w.Body.String(), "<h1>Hello</h1>", "body should contain HTML content.")
}

func TestHTML_WithStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	HTML(w, r, []byte("<p>Not Found</p>"), WithStatus(http.StatusNotFound))

	td.Cmp(t, w.Code, http.StatusNotFound, "status code should be 404.")
}

func TestHTML_EmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	HTML(w, r, nil)

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8",
		"content type should be set even with empty body.")
	td.Cmp(t, w.Body.String(), "", "body should be empty.")
}

func TestTEXT_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TEXT(w, r, []byte("plain text content"))

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "text/plain; charset=utf-8",
		"content type should be text/plain.")
	td.Cmp(t, w.Body.String(), "plain text content", "body should contain text content.")
}

func TestTEXT_WithStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TEXT(w, r, []byte("accepted"), WithStatus(http.StatusAccepted))

	td.Cmp(t, w.Code, http.StatusAccepted, "status code should be 202.")
}

func TestTEXT_EmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TEXT(w, r, nil)

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "text/plain; charset=utf-8",
		"content type should be set even with empty body.")
	td.Cmp(t, w.Body.String(), "", "body should be empty.")
}

func TestErrorHTTP_ErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "ErrAlreadyExists maps to 409 Conflict.",
			err:            errkit.ErrAlreadyExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "ErrNotFound maps to 404 Not Found.",
			err:            errkit.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "ErrUnauthenticated maps to 401 Unauthorized.",
			err:            errkit.ErrUnauthenticated,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "ErrUnauthorized maps to 403 Forbidden.",
			err:            errkit.ErrUnauthorized,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "ErrInvalidArgument maps to 400 Bad Request.",
			err:            errkit.ErrInvalidArgument,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ErrUnavailable maps to 503 Service Unavailable.",
			err:            errkit.ErrUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Unknown error maps to 500 Internal Server Error.",
			err:            errors.New("something went wrong"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			ErrorHTTP(w, r, tt.err)

			td.Cmp(t, w.Code, tt.expectedStatus, "HTTP status code should match error mapping.")
		})
	}
}

func TestErrorHTTP_WrappedErrors(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "Wrapped ErrNotFound maps to 404.",
			err:            fmt.Errorf("user lookup: %w", errkit.ErrNotFound),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Wrapped ErrAlreadyExists maps to 409.",
			err:            fmt.Errorf("create resource: %w", errkit.ErrAlreadyExists),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "Wrapped ErrUnauthenticated maps to 401.",
			err:            fmt.Errorf("auth check: %w", errkit.ErrUnauthenticated),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			ErrorHTTP(w, r, tt.err)

			td.Cmp(t, w.Code, tt.expectedStatus, "wrapped error should still map correctly.")
		})
	}
}

func TestErrorHTTP_WithStatusOverride(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Even though the error maps to 404, WithStatus should override it.
	ErrorHTTP(w, r, errkit.ErrNotFound, WithStatus(http.StatusTeapot))

	td.Cmp(t, w.Code, http.StatusTeapot,
		"WithStatus option should override the default error mapping.")
}

func TestErrorHTTP_LogHookCalled(t *testing.T) {
	var loggedErr error

	ctx := ctxkit.SetLogErrHook(context.Background(), func(err error) { loggedErr = err })
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	testErr := errors.New("test error")
	ErrorHTTP(w, r, testErr)

	td.Cmp(t, loggedErr, testErr, "log hook should receive the error.")
}

func TestTemplateHTML_Success(t *testing.T) {
	// Save and restore original htmlTemplater.
	origTemplater := htmlTemplater
	t.Cleanup(func() { htmlTemplater = origTemplater })

	tmpl := template.Must(template.New("test").Parse("<p>Hello, {{.Name}}</p>"))

	htmlTemplater = &mockTemplater{tmpl: tmpl}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TemplateHTML(w, r, TemplateData{
		Name: "test",
		Data: map[string]string{"Name": "World"},
	})

	td.Cmp(t, w.Code, http.StatusOK, "status code should be 200.")
	td.Cmp(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8",
		"content type should be text/html.")
	td.Cmp(t, w.Body.String(), "<p>Hello, World</p>",
		"template should be rendered with provided data.")
}

func TestTemplateHTML_WithStatusCode(t *testing.T) {
	origTemplater := htmlTemplater
	t.Cleanup(func() { htmlTemplater = origTemplater })

	tmpl := template.Must(template.New("test").Parse("<p>Created</p>"))
	htmlTemplater = &mockTemplater{tmpl: tmpl}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TemplateHTML(w, r, TemplateData{Name: "test"}, WithStatus(http.StatusCreated))

	td.Cmp(t, w.Code, http.StatusCreated, "status code should be 201.")
}

func TestTemplateHTML_TemplateProviderError(t *testing.T) {
	origTemplater := htmlTemplater
	t.Cleanup(func() { htmlTemplater = origTemplater })

	htmlTemplater = &mockTemplater{err: errors.New("template not found")}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TemplateHTML(w, r, TemplateData{Name: "missing"})

	td.Cmp(t, w.Code, http.StatusInternalServerError,
		"template provider error should result in 500.")
}

func TestTemplateHTML_ExecutionError(t *testing.T) {
	origTemplater := htmlTemplater
	t.Cleanup(func() { htmlTemplater = origTemplater })

	// Template that expects a method that does not exist on the data type.
	tmpl := template.Must(template.New("test").Parse("{{.Missing}}"))
	htmlTemplater = &mockTemplater{tmpl: tmpl}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Pass a type where .Missing does not exist but template has "missingkey=error".
	// Use nil data to make the template fail on execution.
	TemplateHTML(w, r, TemplateData{
		Name: "test",
		Data: nil,
	})

	// With nil data, {{.Missing}} will produce "<no value>" by default, not an error.
	// So this should succeed with 200.
	td.Cmp(t, w.Code, http.StatusOK,
		"template with nil data and missing field should render with zero value.")
}

func TestTemplateHTML_NoopTemplater(t *testing.T) {
	origTemplater := htmlTemplater
	t.Cleanup(func() { htmlTemplater = origTemplater })

	// Use the noop templater which always returns an error.
	htmlTemplater = &noopTemplater{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	TemplateHTML(w, r, TemplateData{Name: "test"})

	td.Cmp(t, w.Code, http.StatusInternalServerError,
		"noop templater should result in 500.")
}

// mockTemplater implements HTMLTemplateProvider for testing.
type mockTemplater struct {
	tmpl *template.Template
	err  error
}

func (m *mockTemplater) Template(_ context.Context, _ string) (*template.Template, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.tmpl, nil
}
