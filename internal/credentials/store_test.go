package credentials

import (
	"errors"
	"testing"
)

type fakeBackend struct {
	values map[string]string
	err    error
}

func (f *fakeBackend) Set(_ string, user, password string) error {
	if f.err != nil {
		return f.err
	}
	f.values[user] = password
	return nil
}

func (f *fakeBackend) Get(_ string, user string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	value, ok := f.values[user]
	if !ok {
		return "", errors.New("missing")
	}
	return value, nil
}

func (f *fakeBackend) Delete(_ string, user string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.values, user)
	return nil
}

func TestCredentialStoreUsesInjectedProtectedBackend(t *testing.T) {
	backend := &fakeBackend{values: make(map[string]string)}
	credentials := NewWithBackend(backend)
	const id = "550e8400-e29b-41d4-a716-446655440000"
	if err := credentials.Put(id, "synthetic-secret"); err != nil {
		t.Fatal(err)
	}
	value, err := credentials.Get(id)
	if err != nil || value != "synthetic-secret" {
		t.Fatalf("unexpected credential read: %q, %v", value, err)
	}
	if err := credentials.Delete(id); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialStoreRejectsInvalidInputAndBackendFailure(t *testing.T) {
	credentials := NewWithBackend(&fakeBackend{values: make(map[string]string), err: errors.New("keyring unavailable")})
	if err := credentials.Put("not-an-id", "secret"); err == nil {
		t.Fatal("invalid account identifier must be rejected")
	}
	const id = "550e8400-e29b-41d4-a716-446655440000"
	if err := credentials.Put(id, "secret"); err == nil {
		t.Fatal("unavailable system keyring must fail closed")
	}
}
