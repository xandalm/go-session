package assert

import (
	"fmt"
	"reflect"
	"testing"
)

func output(common string, out []any) string {
	if len(out) == 0 {
		return common
	}
	if str, ok := out[0].(string); ok {
		return fmt.Sprintf(str, out[1:]...)
	}
	panic("output argument must be a fmt string")
}

func isNil(v any) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Pointer, reflect.UnsafePointer,
		reflect.Map, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

func NotNil(t testing.TB, v any, out ...any) {
	t.Helper()
	if isNil(v) {
		t.Fatal(output("expected not nil value", out))
	}
}

func Nil(t testing.TB, v any, out ...any) {
	t.Helper()
	if !isNil(v) {
		t.Fatal(output("expected nil value", out))
	}
}

func isEmpty(v any) bool {
	if isNil(v) {
		return true
	}

	value := reflect.ValueOf(v)

	switch value.Kind() {
	case reflect.Array,
		reflect.Chan,
		reflect.Map,
		reflect.Slice,
		reflect.String:
		return value.Len() == 0
	case reflect.Ptr, reflect.UnsafePointer:
		return isEmpty(value.Elem().Interface())
	default:
		zero := reflect.Zero(value.Type())
		return reflect.DeepEqual(value, zero)
	}
}

func Empty[T any](t testing.TB, v T, out ...any) {
	t.Helper()

	if !isEmpty(v) {
		common := fmt.Sprintf("expected empty, but got %v", v)
		t.Fatal(output(common, out))
	}
}

func NotEmpty[T any](t testing.TB, v T, out ...any) {
	t.Helper()

	if isEmpty(v) {
		common := fmt.Sprintf("expected not empty, but got %v", v)
		t.Fatal(output(common, out))
	}
}

func Equal[T any](t testing.TB, got, want T, out ...any) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		common := fmt.Sprintf("expected equal values, but got %v and want %v", got, want)
		t.Fatal(output(common, out))
	}
}

func AnError(t testing.TB, err error, out ...any) {
	t.Helper()
	if err == nil {
		t.Fatal(output("expected an error, but didn't get one", out))
	}
}

func NoError(t testing.TB, err error, out ...any) {
	t.Helper()
	if err != nil {
		t.Fatal(output("expected no error but got one", out))
	}
}

func Error(t testing.TB, got, want error, out ...any) {
	t.Helper()

	if got != want {
		common := fmt.Sprintf("expected error %v, but didn't get it", got)
		t.Fatal(output(common, out))
	}
}

func NotError(t testing.TB, got, nwant error, out ...any) {
	t.Helper()

	if got == nwant {
		common := fmt.Sprintf("didn't expected error %v, but got it", got)
		t.Fatal(output(common, out))
	}
}
