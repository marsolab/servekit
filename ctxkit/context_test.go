package ctxkit

import (
	"context"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestGetLogErrHook(t *testing.T) {
	want := func(error) {}
	ctx := context.WithValue(context.Background(), logErrHook, want)
	got := GetLogErrHook(ctx)
	td.Cmp(t, got, td.Shallow(want))
}

func TestGetRequestID(t *testing.T) {
	want := "testid"
	ctx := context.WithValue(context.Background(), requestID, want)
	got := GetRequestID(ctx)
	td.Cmp(t, got, want)
}

func TestSetLogErrHook(t *testing.T) {
	want := func(error) {}
	ctx := SetLogErrHook(context.Background(), want)
	got, _ := ctx.Value(logErrHook).(func(error))
	td.Cmp(t, got, td.Shallow(want))
}

func TestSetRequestID(t *testing.T) {
	want := "testid"
	ctx := SetRequestID(context.Background(), want)
	got := ctx.Value(requestID)
	td.Cmp(t, got, want)
}

func TestSet(t *testing.T) {
	want := "test"
	ctx := Set[string](context.Background(), "ctx.str", want)
	got := ctx.Value(Key("ctx.str"))
	td.Cmp(t, got, want)
}

func TestGet(t *testing.T) {
	want := "test"
	ctx := context.WithValue(context.Background(), Key("ctx.str"), want)
	got := Get[string](ctx, "ctx.str")
	td.Cmp(t, got, want)
}

func TestGetLogErrHook_NoHookSet(t *testing.T) {
	// Getting a log error hook from a context that has no hook set should return nil.
	ctx := context.Background()
	got := GetLogErrHook(ctx)
	td.CmpNil(t, got)
}

func TestGetLogErrHook_WrongType(t *testing.T) {
	// Getting a log error hook when the context value has the wrong type should return nil.
	ctx := context.WithValue(context.Background(), logErrHook, "not-a-func")
	got := GetLogErrHook(ctx)
	td.CmpNil(t, got)
}

func TestGetRequestID_NoValueSet(t *testing.T) {
	// Getting a request ID from a context that has no value set should return an empty string.
	ctx := context.Background()
	got := GetRequestID(ctx)
	td.Cmp(t, got, "")
}

func TestGetRequestID_EmptyString(t *testing.T) {
	// Getting a request ID when the stored value is an empty string should return an empty string.
	ctx := context.WithValue(context.Background(), requestID, "")
	got := GetRequestID(ctx)
	td.Cmp(t, got, "")
}

func TestGetRequestID_WrongType(t *testing.T) {
	// Getting a request ID when the context value has the wrong type should return an empty string.
	ctx := context.WithValue(context.Background(), requestID, 12345)
	got := GetRequestID(ctx)
	td.Cmp(t, got, "")
}

func TestGet_WrongType(t *testing.T) {
	// Getting a value with a mismatched type should return the zero value for that type.
	ctx := context.WithValue(context.Background(), Key("ctx.int"), "not-an-int")
	got := Get[int](ctx, "ctx.int")
	td.Cmp(t, got, 0)
}

func TestGet_MissingKey(t *testing.T) {
	// Getting a value for a key that was never set should return the zero value.
	ctx := context.Background()
	got := Get[string](ctx, "ctx.missing")
	td.Cmp(t, got, "")
}

func TestGet_ZeroValueTypes(t *testing.T) {
	// Getting missing values for various types should return appropriate zero values.
	ctx := context.Background()

	td.Cmp(t, Get[int](ctx, "ctx.int"), 0)
	td.Cmp(t, Get[bool](ctx, "ctx.bool"), false)
	td.Cmp(t, Get[float64](ctx, "ctx.float"), float64(0))
	td.CmpNil(t, Get[[]string](ctx, "ctx.slice"))
	td.CmpNil(t, Get[map[string]string](ctx, "ctx.map"))
}

func TestSet_OverwriteValue(t *testing.T) {
	// Setting a value with the same key should overwrite the previous value.
	ctx := Set[string](context.Background(), "ctx.val", "first")
	ctx = Set[string](ctx, "ctx.val", "second")
	got := Get[string](ctx, "ctx.val")
	td.Cmp(t, got, "second")
}
