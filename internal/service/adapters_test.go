package service

import (
	"context"
	"testing"
)

func TestFutureProtocolAdaptersExplicitlyRejectOperations(t *testing.T) {
	for protocol, adapter := range NewAdapters() {
		if adapter.Capabilities().Available {
			t.Fatalf("%s unexpectedly reports available", protocol)
		}
		for operation, err := range map[string]error{
			"connect":     adapter.Connect(context.Background()),
			"synchronize": adapter.Synchronize(context.Background()),
			"send":        adapter.Send(context.Background(), []byte("synthetic")),
		} {
			if !IsUnavailable(err) {
				t.Errorf("%s %s did not fail as unavailable: %v", protocol, operation, err)
			}
		}
	}
}
