package model

import (
	"encoding/json"
	"testing"
)

func TestSyntheticAccountRoundTripsInVersionedState(t *testing.T) {
	account, err := MakeAccount(AccountInput{
		Label:    "  Household  ",
		Address:  "  family@example.invalid ",
		Protocol: "imap-smtp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.Label != "Household" || account.Address != "family@example.invalid" {
		t.Fatalf("input was not normalized: %#v", account)
	}
	state := EmptyState()
	state.Accounts = append(state.Accounts, account)
	content, err := state.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := DecodeState(content)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Accounts[0] != account || restored.SchemaVersion != SchemaVersion {
		t.Fatalf("unexpected restored state: %#v", restored)
	}
}

func TestStateRejectsUnknownSchemaFieldsAndReferences(t *testing.T) {
	state := EmptyState()
	account, err := MakeAccount(AccountInput{Label: "Home", Address: "home@example.invalid", Protocol: "jmap"})
	if err != nil {
		t.Fatal(err)
	}
	state.Accounts = append(state.Accounts, account)

	content, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	withUnknownField := append(append([]byte(nil), content[:len(content)-1]...), []byte(`,"unexpected":true}`)...)
	if _, err := DecodeState(withUnknownField); err == nil {
		t.Fatal("unknown fields must be rejected")
	}

	state.Cache = append(state.Cache, CacheEntry{
		ID:        mustIdentifier(t),
		AccountID: account.ID,
		EntityID:  mustIdentifier(t),
		Kind:      "message",
		State:     "metadata",
	})
	if err := ValidateState(state); err == nil {
		t.Fatal("cache references to unknown entities must be rejected")
	}
}

func TestStateRejectsNullCollectionsAndDuplicateIdentifiers(t *testing.T) {
	if _, err := DecodeState([]byte(`{"schemaVersion":1,"accounts":null,"messages":[],"calendars":[],"events":[],"contacts":[],"tasks":[],"cache":[],"queue":[]}`)); err == nil {
		t.Fatal("null collections must be rejected")
	}
	account, err := MakeAccount(AccountInput{Label: "One", Address: "one@example.invalid", Protocol: "jmap"})
	if err != nil {
		t.Fatal(err)
	}
	state := EmptyState()
	state.Accounts = []Account{account, account}
	if err := ValidateState(state); err == nil {
		t.Fatal("duplicate identifiers must be rejected")
	}
}

func mustIdentifier(t *testing.T) string {
	t.Helper()
	id, err := NewIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
