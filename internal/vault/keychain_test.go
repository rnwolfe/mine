package vault

import (
	"errors"
	"os"
	"testing"
)

func TestNoopKeychain_Get(t *testing.T) {
	s := &noopKeychain{}
	_, err := s.Get("any")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Get: got %v, want ErrNotSupported", err)
	}
}

func TestNoopKeychain_Set(t *testing.T) {
	s := &noopKeychain{}
	err := s.Set("any", "passphrase")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Set: got %v, want ErrNotSupported", err)
	}
}

func TestNoopKeychain_Delete(t *testing.T) {
	s := &noopKeychain{}
	err := s.Delete("any")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Delete: got %v, want ErrNotSupported", err)
	}
}

func TestIsKeychainMiss(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrNotSupported", ErrNotSupported, true},
		{"os.ErrNotExist", os.ErrNotExist, true},
		{"nil", nil, false},
		{"other error", errors.New("some other error"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsKeychainMiss(tc.err); got != tc.want {
				t.Errorf("IsKeychainMiss(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestNewPlatformStore_NonNil(t *testing.T) {
	s := NewPlatformStore()
	if s == nil {
		t.Fatal("NewPlatformStore() returned nil")
	}
}

// Compile-time check: noopKeychain satisfies PassphraseStore on all platforms.
var _ PassphraseStore = (*noopKeychain)(nil)
