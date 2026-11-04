package service

import (
	"context"
	"errors"
)

type Adapter interface {
	Capabilities() AdapterStatus
	Connect(context.Context) error
	Synchronize(context.Context) error
	Send(context.Context, []byte) error
}

type UnavailableError struct {
	Protocol string
}

func (e *UnavailableError) Error() string { return "FEATURE_UNAVAILABLE: " + e.Protocol }

type unavailableAdapter struct {
	protocol string
}

func (a unavailableAdapter) Capabilities() AdapterStatus {
	return AdapterStatus{ContractVersion: 1, Protocol: a.protocol, Available: false}
}

func (a unavailableAdapter) Connect(context.Context) error {
	return &UnavailableError{Protocol: a.protocol}
}

func (a unavailableAdapter) Synchronize(context.Context) error {
	return &UnavailableError{Protocol: a.protocol}
}

func (a unavailableAdapter) Send(context.Context, []byte) error {
	return &UnavailableError{Protocol: a.protocol}
}

func NewAdapters() map[string]Adapter {
	adapters := make(map[string]Adapter)
	for _, protocol := range []string{"imap", "smtp", "jmap", "caldav", "carddav"} {
		adapters[protocol] = unavailableAdapter{protocol: protocol}
	}
	return adapters
}

func IsUnavailable(err error) bool {
	var unavailable *UnavailableError
	return errors.As(err, &unavailable)
}
