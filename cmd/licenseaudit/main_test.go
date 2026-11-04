package main

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestLicenseAuditRejectsReplacedModulesAndModifiedLicense(t *testing.T) {
	directory := t.TempDir()
	item := module{Path: "github.com/zalando/go-keyring", Version: "v0.2.6", Dir: directory,
		Replace: &module{Path: "../unreviewed"}}
	if _, err := verifyModules([]module{item}); err == nil {
		t.Fatal("local replacements must not inherit upstream reviews")
	}
	item.Replace = nil
	if err := os.WriteFile(filepath.Join(directory, "LICENSE"), []byte("MIT License\nnon-commercial only"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyModules([]module{item}); err == nil {
		t.Fatal("matching a license title must not approve changed terms")
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

func TestSBOMComponentsAreUniqueAndConsistent(t *testing.T) {
	entry := component{Name: "Same component", PURL: "pkg:generic/example@1.0", Version: "1.0"}
	components, err := uniqueComponents([]component{entry, entry})
	if err != nil || len(components) != 1 {
		t.Fatalf("shared vendor entries must occur only once: %#v, %v", components, err)
	}
	conflict := entry
	conflict.Version = "2.0"
	if _, err := uniqueComponents([]component{entry, conflict}); err == nil {
		t.Fatal("inconsistent reviews for one component identity must fail")
	}
}

func TestVendorInventoryRequiresExactReviewedVersions(t *testing.T) {
	root := t.TempDir()
	path := "golang.org/x/example"
	directory := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "LICENSE"), license, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		inventory string
		valid     bool
	}{
		{"# " + path + " v1.0.0\n", true},
		{"# " + path + " v1.0.1\n", false},
		{"# example.invalid/unreviewed v1.0.0\n", false},
		{"", false},
	} {
		if err := os.WriteFile(filepath.Join(root, "modules.txt"), []byte(item.inventory), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := verifyVendorInventory(root, map[string]string{path: "v1.0.0"})
		if (err == nil) != item.valid {
			t.Fatalf("inventory %q: %v", item.inventory, err)
		}
	}
}
