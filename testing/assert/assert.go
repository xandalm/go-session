package assert

import "testing"

func AssertNotNil(t testing.TB, v any) {
	t.Helper()

	if v == nil {
		t.Fatal("expected not nil")
	}
}

func AssertNotEmpty(t testing.TB, v string) {
	t.Helper()

	if v == "" {
		t.Fatalf("expected not empty")
	}
}

func AssertEqual(t testing.TB, got, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("expected same values, but got %q and want %q", got, want)
	}
}

func AssertNoError(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
}

func AssertError(t testing.TB, got, want error) {
	t.Helper()

	if got == nil {
		t.Fatalf("expect error, but didn't got one")
	}

	if got != want {
		t.Fatalf(`got error "%v" but want "%v"`, got, want)
	}
}
