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

func NotNil(t testing.TB, v any, out ...any) {
	t.Helper()
	if v == nil {
		t.Fatal(output("expected not nil value", out))
	}
}

func Nil(t testing.TB, v any, out ...any) {
	t.Helper()

	if v != nil {
		t.Fatal(output("expected nil value", out))
	}
}

func isEmpty(v any) bool {
	if v == nil {
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
	case reflect.Ptr:
		if value.IsNil() {
			return true
		}
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

func NoError(t testing.TB, err, nwant error, out ...any) {
	t.Helper()

	if err == nwant {
		common := fmt.Sprintf("didn't want error %v, but got it", err)
		t.Fatal(output(common, out))
	}
}

func Error(t testing.TB, got, want error, out ...any) {
	t.Helper()

	if got == nil {
		t.Fatal(output("didn't get an error", out))
	}
	if got != want {
		common := fmt.Sprintf("got error %v, but want %v", got, want)
		t.Fatal(output(common, out))
	}
}
