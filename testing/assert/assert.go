package assert

import "testing"

func NotNil(t testing.TB, v any) {
	t.Helper()

	if v == nil {
		t.Fatal("expected not nil")
	}
}

func NotEmpty(t testing.TB, v string) {
	t.Helper()

	if v == "" {
		t.Fatalf("expected not empty")
	}
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
