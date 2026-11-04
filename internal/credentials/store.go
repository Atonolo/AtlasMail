package credentials

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/zalando/go-keyring"
)

const serviceName = "Atlas Mail"

type Backend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type Store struct {
	backend Backend
}

func New() *Store { return &Store{backend: systemBackend{}} }

func NewWithBackend(backend Backend) *Store { return &Store{backend: backend} }

func (s *Store) Put(accountID, secret string) error {
	if !model.ValidIdentifier(accountID) {
		return errors.New("INVALID_IDENTIFIER")
	}
	if strings.TrimSpace(secret) == "" || len(secret) > 2048 {
		return errors.New("INVALID_CREDENTIAL")
	}
	if err := s.backend.Set(serviceName, accountID, secret); err != nil {
		return fmt.Errorf("CREDENTIAL_STORE_UNAVAILABLE: %w", err)
	}
	return nil
}

func (s *Store) Get(accountID string) (string, error) {
	if !model.ValidIdentifier(accountID) {
		return "", errors.New("INVALID_IDENTIFIER")
	}
	value, err := s.backend.Get(serviceName, accountID)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("CREDENTIAL_READ_FAILED: %w", err)
	}
	return value, nil
}

func (s *Store) Delete(accountID string) error {
	if !model.ValidIdentifier(accountID) {
		return errors.New("INVALID_IDENTIFIER")
	}
	err := s.backend.Delete(serviceName, accountID)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("CREDENTIAL_DELETE_FAILED: %w", err)
	}
	return nil
}

type systemBackend struct{}

func (systemBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (systemBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}
