package utils

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("test error", 404)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}
	if Code(err) != 404 {
		t.Errorf("expected code 404, got %d", Code(err))
	}
}

func TestWrap(t *testing.T) {
	t.Run("wrap nil returns nil", func(t *testing.T) {
		err := Wrap(nil, "msg", 500)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("wrap with message", func(t *testing.T) {
		cause := errors.New("root cause")
		err := Wrap(cause, "wrapped", 500)
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if Code(err) != 500 {
			t.Errorf("expected code 500, got %d", Code(err))
		}
		if !containsString(err.Error(), "wrapped") || !containsString(err.Error(), "root cause") {
			t.Errorf("expected error to contain both messages, got '%s'", err.Error())
		}
	})

	t.Run("wrap with empty message", func(t *testing.T) {
		cause := errors.New("root cause")
		err := Wrap(cause, "", 503)
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if Code(err) != 503 {
			t.Errorf("expected code 503, got %d", Code(err))
		}
		if err.Error() != "root cause" {
			t.Errorf("expected 'root cause', got '%s'", err.Error())
		}
	})
}

func TestCode(t *testing.T) {
	t.Run("nil error returns 0", func(t *testing.T) {
		if Code(nil) != 0 {
			t.Errorf("expected 0 for nil error, got %d", Code(nil))
		}
	})

	t.Run("internal error returns code", func(t *testing.T) {
		err := New("test", 42)
		if Code(err) != 42 {
			t.Errorf("expected 42, got %d", Code(err))
		}
	})

	t.Run("standard error returns 10000", func(t *testing.T) {
		err := errors.New("standard")
		if Code(err) != 10000 {
			t.Errorf("expected 10000, got %d", Code(err))
		}
	})

	t.Run("nested wrapped error", func(t *testing.T) {
		inner := New("inner", 123)
		outer := Wrap(inner, "outer", 456)
		if Code(outer) != 456 {
			t.Errorf("expected 456 for outer, got %d", Code(outer))
		}
	})
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
