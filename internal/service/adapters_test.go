package service

import (
	"context"
	"testing"
)

func TestMailAdaptersAreAvailableAndDAVRemainsExplicitlyUnavailable(t *testing.T) {
	for protocol, adapter := range NewAdapters() {
		available := protocol == "imap" || protocol == "smtp" || protocol == "jmap"
		if adapter.Capabilities().Available != available {
			t.Fatalf("%s availability = %v, want %v", protocol, adapter.Capabilities().Available, available)
		}
		for operation, err := range map[string]error{
			"connect":     adapter.Connect(context.Background()),
			"synchronize": adapter.Synchronize(context.Background()),
			"send":        adapter.Send(context.Background(), []byte("synthetic")),
		} {
			if available && err == nil {
				t.Errorf("%s %s succeeded without account configuration", protocol, operation)
			}
			if !available && !IsUnavailable(err) {
				t.Errorf("%s %s did not fail as unavailable: %v", protocol, operation, err)
			}
		}
	}
}
