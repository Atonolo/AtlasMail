package main

import (
	"testing"
)

func TestLicenseAuditFailsClosedOnNewModuleVersions(t *testing.T) {
	_, err := verifyModules([]module{{
		Path:    "github.com/zalando/go-keyring",
		Version: "v0.2.7",
		Dir:     t.TempDir(),
	}})
	if err == nil {
		t.Fatal("an unreviewed module version must block the audit")
	}
}

func TestLicenseAuditFailsClosedOnUnknownModules(t *testing.T) {
	_, err := verifyModules([]module{{
		Path:    "example.invalid/unreviewed",
		Version: "v1.0.0",
		Dir:     t.TempDir(),
	}})
	if err == nil {
		t.Fatal("an unreviewed module must block the audit")
	}
}
