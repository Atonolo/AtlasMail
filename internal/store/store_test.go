package store

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Atonolo/AtlasMail/internal/model"
)

func TestStorePersistsSyntheticAccountAndBackup(t *testing.T) {
	directory := t.TempDir()
	stateStore := New(directory)
	account, err := stateStore.AddAccount(model.AccountInput{
		Label:    "Synthetic",
		Address:  "synthetic@example.invalid",
		Protocol: "jmap",
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(directory).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.Accounts) != 1 || reopened.Accounts[0] != account {
		t.Fatalf("account did not survive reopen: %#v", reopened)
	}
	backup, err := stateStore.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateStore.RestoreBackup(backup); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidRestorePreservesExistingBytes(t *testing.T) {
	stateStore := New(t.TempDir())
	if _, err := stateStore.AddAccount(model.AccountInput{
		Label:    "Keep",
		Address:  "keep@example.invalid",
		Protocol: "imap-smtp",
	}); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(stateStore.directory, dataFilename)
	before, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := stateStore.RestoreBackup([]byte(`{"schemaVersion":99}`)); err == nil {
		t.Fatal("unsupported backup schema must fail")
	}
	after, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("failed restore changed the valid state")
	}
}

func TestExistingLockFailsWithoutReplacingData(t *testing.T) {
	stateStore := New(t.TempDir())
	filename := filepath.Join(stateStore.directory, dataFilename)
	if err := os.MkdirAll(stateStore.directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename+".lock", []byte("another process"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := stateStore.AddAccount(model.AccountInput{
		Label:    "Locked",
		Address:  "locked@example.invalid",
		Protocol: "jmap",
	})
	var storeErr *Error
	if !errors.As(err, &storeErr) || storeErr.Code != "STORE_LOCKED" {
		t.Fatalf("expected STORE_LOCKED, got %v", err)
	}
}

func TestRelativeDataDirectoryIsRejected(t *testing.T) {
	t.Setenv("ATLAS_MAIL_DATA_DIR", "relative/path")
	if _, err := DataDirectory(); err == nil {
		t.Fatal("relative data directory must fail")
	}
}

func TestLinuxDataDirectoryUsesXDGDataHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux directory convention")
	}
	t.Setenv("ATLAS_MAIL_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/atlas-mail-test-data")
	directory, err := DataDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if directory != filepath.Join("/tmp/atlas-mail-test-data", "AtlasMail") {
		t.Fatalf("unexpected data directory: %s", directory)
	}
}
