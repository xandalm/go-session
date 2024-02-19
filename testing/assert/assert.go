package assert

import (
	"reflect"
	"testing"
)

func NotNil(t testing.TB, v any) {
	t.Helper()

	if v == nil {
		t.Fatal("expected not nil")
	}
}

func NotEmpty[T any](t testing.TB, v T) {
	t.Helper()

	value := reflect.ValueOf(v)
	kind := value.Kind()

	if kind == reflect.Pointer {
		value = value.Elem()
		kind = value.Kind()
	}

	switch kind {
	case reflect.Array,
		reflect.Chan,
		reflect.Map,
		reflect.Slice,
		reflect.String:
		if value.Len() != 0 {
			return
		}
	default:
		zeroValue := reflect.Zero(reflect.TypeOf(value))
		if !reflect.DeepEqual(zeroValue, value) {
			return
		}
	}
	t.Fatalf("expected not empty")
}

func Equal[T comparable](t testing.TB, got, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("expected same values, but got %v and want %v", got, want)
	}
}

func NoError(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
}

func Error(t testing.TB, got, want error) {
	t.Helper()

	if got == nil {
		t.Fatalf("expect error, but didn't got one")
	}

	if got != want {
		t.Fatalf(`got error "%v" but want "%v"`, got, want)
	}
}
