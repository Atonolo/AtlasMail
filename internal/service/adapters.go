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

type implementedAdapter struct {
	protocol string
}

func (a implementedAdapter) Capabilities() AdapterStatus {
	return AdapterStatus{ContractVersion: 1, Protocol: a.protocol, Available: true}
}

func (implementedAdapter) Connect(context.Context) error {
	return errors.New("ACCOUNT_CONFIGURATION_REQUIRED")
}

func (implementedAdapter) Synchronize(context.Context) error {
	return errors.New("ACCOUNT_CONFIGURATION_REQUIRED")
}

func (implementedAdapter) Send(context.Context, []byte) error {
	return errors.New("ACCOUNT_CONFIGURATION_REQUIRED")
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
	for _, protocol := range []string{"imap", "smtp", "jmap"} {
		adapters[protocol] = implementedAdapter{protocol: protocol}
	}
	for _, protocol := range []string{"caldav", "carddav"} {
		adapters[protocol] = unavailableAdapter{protocol: protocol}
	}
	return adapters
}

func AdapterStatuses() []AdapterStatus {
	adapters := NewAdapters()
	return []AdapterStatus{
		adapters["imap"].Capabilities(), adapters["smtp"].Capabilities(),
		adapters["jmap"].Capabilities(), adapters["caldav"].Capabilities(),
		adapters["carddav"].Capabilities(),
	}
}

func IsUnavailable(err error) bool {
	var unavailable *UnavailableError
	return errors.As(err, &unavailable)
}
