package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestPortablePackageRetainsExecutableAndNotices(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	if err := os.WriteFile(source, []byte("synthetic package content"), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(directory, "package.zip")
	if err := writeArchive(destination, map[string]string{
		"atlas-mail": source, "LICENSE": source, "DEPENDENCY-LICENSES.txt": source, "sbom.cdx.json": source,
	}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 4 {
		t.Fatalf("unexpected archive entries: %d", len(archive.File))
	}
	for _, file := range archive.File {
		if file.Name == "AtlasMail/atlas-mail" && file.Mode().Perm()&0o111 != 0o111 {
			t.Fatal("portable application is missing executable permissions")
		}
	}
	entry, err := fingerprint(destination)
	if err != nil || len(entry.SHA256) != 64 {
		t.Fatalf("invalid artifact fingerprint: %#v, %v", entry, err)
	}
}

func TestIncompletePackageIsRemoved(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "package.zip")
	err := writeArchive(destination, map[string]string{"LICENSE": filepath.Join(directory, "missing")})
	if err == nil {
		t.Fatal("missing package input must fail")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("incomplete archive remains: %v", err)
	}
}
