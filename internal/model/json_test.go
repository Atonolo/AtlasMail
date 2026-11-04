package model

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestV1ReferenceFixturePreservesAllCollections(t *testing.T) {
	data, err := os.ReadFile("testdata/state-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := DecodeState(data)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := state.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := DecodeState(encoded)
	if err != nil || len(restored.Queue) != 1 || len(restored.Cache) != 1 ||
		len(restored.Events) != 1 || restored.Accounts[0].ID != state.Accounts[0].ID {
		t.Fatalf("reference fixture did not round-trip: %v", err)
	}
}

func TestStateDecodingRejectsLossyJSON(t *testing.T) {
	fixture, err := os.ReadFile("testdata/state-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"missing optional-looking field": bytes.Replace(fixture, []byte(`, "remoteId": "contact-example"`), nil, 1),
		"null string":                    bytes.Replace(fixture, []byte(`"remoteId": "contact-example"`), []byte(`"remoteId": null`), 1),
		"case alias":                     bytes.Replace(fixture, []byte(`"address":`), []byte(`"Address":`), 1),
		"duplicate key":                  bytes.Replace(fixture, []byte(`"schemaVersion": 1,`), []byte(`"schemaVersion": 1, "schemaVersion": 1,`), 1),
		"trailing JSON":                  append(bytes.Clone(fixture), []byte(` {}`)...),
		"invalid UTF-8":                  bytes.Replace(fixture, []byte("Synthetic account"), []byte{0xff}, 1),
		"unpaired high surrogate":        bytes.Replace(fixture, []byte("Synthetic account"), []byte(`\ud800`), 1),
		"unpaired low surrogate":         bytes.Replace(fixture, []byte("Synthetic account"), []byte(`\udfff`), 1),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeState(data); err == nil {
				t.Fatal("invalid JSON contract accepted")
			}
		})
	}
}

func TestAccountInputRequiresExactNonNullFields(t *testing.T) {
	for _, data := range []string{
		`{"label":"Test","address":"test@example.invalid"}`,
		`{"Label":"Test","address":"test@example.invalid","protocol":"jmap"}`,
		`{"label":"Test","label":"Test","address":"test@example.invalid","protocol":"jmap"}`,
		`{"label":null,"address":"test@example.invalid","protocol":"jmap"}`,
		`{"label":"Test","address":"test@example.invalid","protocol":"jmap","secret":"no"}`,
	} {
		if _, err := DecodeAccountInput([]byte(data)); err == nil {
			t.Errorf("accepted invalid account input: %s", data)
		}
	}
}

func TestUnicodeEscapesAreNeverSilentlyReplaced(t *testing.T) {
	for _, item := range []struct {
		encoded string
		want    string
	}{
		{`"\ud83d\ude00"`, "\U0001f600"},
		{`"\\ud800"`, `\ud800`},
		{`"\ufffd"`, "\ufffd"},
	} {
		data := []byte(`{"label":` + item.encoded + `,"address":"test@example.invalid","protocol":"jmap"}`)
		input, err := DecodeAccountInput(data)
		if err != nil || input.Label != item.want {
			t.Fatalf("Unicode round trip %s: %#v, %v", item.encoded, input, err)
		}
	}
	if _, err := MakeAccount(AccountInput{Label: string([]byte{0xff}), Address: "test@example.invalid", Protocol: "jmap"}); err == nil {
		t.Fatal("invalid Go UTF-8 must not reach lossy JSON serialization")
	}
}

func TestV1TextLimitsRemainUTF16Compatible(t *testing.T) {
	for _, item := range []struct {
		label string
		valid bool
	}{{strings.Repeat("x", 80), true}, {strings.Repeat("\U0001f600", 40), true}, {strings.Repeat("\U0001f600", 41), false}} {
		_, err := MakeAccount(AccountInput{Label: item.label, Address: "test@example.invalid", Protocol: "jmap"})
		if (err == nil) != item.valid {
			t.Fatalf("incorrect UTF-16 length validation: %v", err)
		}
	}
}

func TestFixtureRejectsWrongKindCrossAccountAndDuplicateIDs(t *testing.T) {
	data, err := os.ReadFile("testdata/state-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, modify := range []func(*State){
		func(s *State) { s.Cache[0].Kind = "task" },
		func(s *State) { s.Queue[0].ID = s.Cache[0].ID },
		func(s *State) {
			account, err := MakeAccount(AccountInput{Label: "Other", Address: "other@example.invalid", Protocol: "jmap"})
			if err != nil {
				t.Fatal(err)
			}
			s.Accounts = append(s.Accounts, account)
			s.Cache[0].AccountID = account.ID
		},
		func(s *State) { s.Queue[0].CreatedAt = "2026-02-30T12:00:00.000Z" },
	} {
		state, err := DecodeState(data)
		if err != nil {
			t.Fatal(err)
		}
		modify(&state)
		if err := ValidateState(state); err == nil {
			t.Fatal("invalid reference fixture accepted")
		}
	}
}
