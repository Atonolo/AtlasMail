package service

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Atonolo/AtlasMail/internal/credentials"
	"github.com/Atonolo/AtlasMail/internal/mail"
	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/Atonolo/AtlasMail/internal/store"
)

type memoryCredentials struct {
	mu      sync.Mutex
	values  map[string]string
	failSet bool
}

func newMemoryCredentials() *memoryCredentials {
	return &memoryCredentials{values: make(map[string]string)}
}

func (m *memoryCredentials) Set(_ string, user, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failSet {
		return errors.New("keyring unavailable")
	}
	m.values[user] = value
	return nil
}

func (m *memoryCredentials) Get(_, user string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.values[user], nil
}

func (m *memoryCredentials) Delete(_, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, user)
	return nil
}

func TestConfigureAndRemoveAccountKeepsCredentialsOutOfExports(t *testing.T) {
	directory := t.TempDir()
	stateStore := store.New(directory)
	keyring := newMemoryCredentials()
	mailService := NewWithCredentials(stateStore, credentials.NewWithBackend(keyring))
	account, err := mailService.ConfigureAccount(model.AccountInput{
		Label: "Personal", Address: "person@example.org", Protocol: "imap-smtp",
	}, mail.Profile{
		Username: "person@example.org", AuthMechanism: "password",
		IMAP: mail.Endpoint{Host: "imap.example.org", Port: 993, TLSMode: "implicit"},
		SMTP: mail.Endpoint{Host: "smtp.example.org", Port: 465, TLSMode: "implicit"},
	}, "sensitive-test-password")
	if err != nil {
		t.Fatal(err)
	}
	status, err := mailService.Status()
	if err != nil || len(status.Accounts) != 1 || !status.Accounts[0].Configured ||
		status.Accounts[0].ID != account.ID {
		t.Fatalf("configured account was not reported: %#v %v", status, err)
	}
	connectionData, err := stateStore.ReadConnections()
	if err != nil {
		t.Fatal(err)
	}
	backup, err := stateStore.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{connectionData, backup} {
		if strings.Contains(string(data), "sensitive-test-password") {
			t.Fatal("account credential was persisted outside the OS keyring")
		}
	}
	stored, err := keyring.Get("Atlas Mail", account.ID)
	if err != nil || stored != "sensitive-test-password" {
		t.Fatalf("credential was not stored in the credential backend: %q %v", stored, err)
	}
	if err := mailService.RemoveAccount(account.ID); err != nil {
		t.Fatal(err)
	}
	removed, err := keyring.Get("Atlas Mail", account.ID)
	if err != nil || removed != "" {
		t.Fatalf("account removal did not revoke credentials: %q %v", removed, err)
	}
	state, err := stateStore.Read()
	if err != nil || len(state.Accounts) != 0 {
		t.Fatalf("account remained after removal: %#v %v", state.Accounts, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "connections-v1.json")); !os.IsNotExist(err) {
		t.Fatalf("connection sidecar was not removed: %v", err)
	}
}

func TestMultipleAccountsKeepProfilesAndCredentialsIndependent(t *testing.T) {
	keyring := newMemoryCredentials()
	mailService := NewWithCredentials(store.New(t.TempDir()), credentials.NewWithBackend(keyring))
	accounts := make([]model.Account, 0, 2)
	secrets := []string{"first-test-password", "second-test-password"}
	for index, address := range []string{"first@example.org", "second@example.org"} {
		account, err := mailService.ConfigureAccount(model.AccountInput{
			Label: address, Address: address, Protocol: "imap-smtp",
		}, mail.Profile{
			Username: address, AuthMechanism: "password",
			IMAP: mail.Endpoint{Host: "imap.example.org", Port: 993, TLSMode: "implicit"},
			SMTP: mail.Endpoint{Host: "smtp.example.org", Port: 465, TLSMode: "implicit"},
		}, secrets[index])
		if err != nil {
			t.Fatal(err)
		}
		accounts = append(accounts, account)
	}
	status, err := mailService.Status()
	if err != nil || len(status.Accounts) != 2 || !status.Accounts[0].Configured || !status.Accounts[1].Configured {
		t.Fatalf("both accounts should be configured independently: %#v %v", status, err)
	}
	for index, account := range accounts {
		secret, err := keyring.Get("Atlas Mail", account.ID)
		if err != nil || secret != secrets[index] {
			t.Fatalf("account %s has the wrong credential: %q %v", account.Address, secret, err)
		}
	}
	if err := mailService.RemoveAccount(accounts[0].ID); err != nil {
		t.Fatal(err)
	}
	status, err = mailService.Status()
	if err != nil || len(status.Accounts) != 1 || status.Accounts[0].ID != accounts[1].ID || !status.Accounts[0].Configured {
		t.Fatalf("removing one account affected the other: %#v %v", status, err)
	}
	remainingSecret, err := keyring.Get("Atlas Mail", accounts[1].ID)
	if err != nil || remainingSecret != secrets[1] {
		t.Fatalf("remaining account credential was changed: %q %v", remainingSecret, err)
	}
}

func TestOAuthAccountUsesPKCEAndDoesNotNeedProviderSDK(t *testing.T) {
	mailService := NewWithCredentials(store.New(t.TempDir()), credentials.NewWithBackend(newMemoryCredentials()))
	account, err := mailService.ConfigureAccount(model.AccountInput{
		Label: "OAuth", Address: "person@example.org", Protocol: "imap-smtp",
	}, mail.Profile{
		Username: "person@example.org", AuthMechanism: "oauth2",
		IMAP: mail.Endpoint{Host: "imap.example.org", Port: 993, TLSMode: "implicit"},
		SMTP: mail.Endpoint{Host: "smtp.example.org", Port: 465, TLSMode: "implicit"},
		OAuth: &mail.OAuthSettings{
			AuthorizationURL: "https://provider.example/oauth/authorize",
			TokenURL:         "https://provider.example/oauth/token", ClientID: "public-client",
			Scope: "mail.read mail.send",
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := mailService.BeginOAuth(account.ID, "https://mail.example.org/oauth/callback")
	if err != nil {
		t.Fatal(err)
	}
	url, err := url.Parse(authorization)
	if err != nil {
		t.Fatal(err)
	}
	query := url.Query()
	if query.Get("response_type") != "code" || query.Get("client_id") != "public-client" ||
		query.Get("code_challenge_method") != "S256" || len(query.Get("code_challenge")) != 43 ||
		len(query.Get("state")) < 32 || strings.Contains(authorization, "code_verifier") {
		t.Fatalf("authorization request lacks PKCE/state protection: %s", authorization)
	}
	if err := mailService.CompleteOAuth(context.Background(), "invalid-state-value-000000000000000000", "code", "https://mail.example.org/oauth/callback"); err == nil {
		t.Fatal("invalid OAuth state was accepted")
	}
}
