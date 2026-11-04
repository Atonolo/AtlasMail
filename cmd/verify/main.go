// Command verify runs the local release gate without GitHub workflows.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type artifact struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

func main() {
	output := flag.String("output", "dist/local", "directory for local evidence and portable packages")
	allPlatforms := flag.Bool("all-platforms", false, "cross-build Windows/macOS/Linux on amd64 and arm64")
	race := flag.Bool("race", false, "enable the Go race detector (requires supported host and C compiler)")
	flag.Parse()
	if err := verify(*output, *allPlatforms, *race); err != nil {
		fmt.Fprintln(os.Stderr, "Local verification failed:", err)
		os.Exit(1)
	}
}

func verify(output string, allPlatforms, race bool) error {
	if err := os.MkdirAll(output, 0o700); err != nil {
		return err
	}
	summaryFile := filepath.Join(output, "verification.json")
	if err := os.Remove(summaryFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	logFile, err := os.Create(filepath.Join(output, "verification.log"))
	if err != nil {
		return err
	}
	defer logFile.Close()
	log := io.MultiWriter(os.Stdout, logFile)
	run := func(arguments ...string) error {
		fmt.Fprintln(log, "go", arguments)
		command := exec.Command("go", arguments...)
		command.Stdout, command.Stderr = log, log
		if err := command.Run(); err != nil {
			return fmt.Errorf("go %v: %w", arguments, err)
		}
		return nil
	}
	if err := run("mod", "download", "all"); err != nil {
		return err
	}
	testArgs := []string{"test", "-count=1"}
	if race {
		testArgs = append(testArgs, "-race")
	}
	if err := run(append(testArgs, "./...")...); err != nil {
		return err
	}
	if err := run("vet", "./..."); err != nil {
		return err
	}
	for _, report := range []struct {
		name string
		args []string
	}{
		{"licenses.json", nil},
		{"sbom.cdx.json", []string{"-sbom"}},
		{"DEPENDENCY-LICENSES.txt", []string{"-notices"}},
	} {
		command := exec.Command("go", append([]string{"run", "./cmd/licenseaudit"}, report.args...)...)
		command.Stderr = log
		data, err := command.Output()
		if err != nil {
			return fmt.Errorf("license gate (%s): %w", report.name, err)
		}
		if err := os.WriteFile(filepath.Join(output, report.name), data, 0o600); err != nil {
			return err
		}
	}
	targets := [][2]string{{runtime.GOOS, runtime.GOARCH}}
	if allPlatforms {
		targets = [][2]string{
			{"windows", "amd64"}, {"windows", "arm64"},
			{"darwin", "amd64"}, {"darwin", "arm64"},
			{"linux", "amd64"}, {"linux", "arm64"},
		}
	}
	temporary, err := os.MkdirTemp("", "atlas-mail-package-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	var artifacts []artifact
	for _, target := range targets {
		name := "atlas-mail"
		if target[0] == "windows" {
			name += ".exe"
		}
		binary := filepath.Join(temporary, name)
		command := exec.Command("go", "build", "-trimpath", "-o", binary, ".")
		command.Env = append(os.Environ(), "GOOS="+target[0], "GOARCH="+target[1], "CGO_ENABLED=0")
		command.Stdout, command.Stderr = log, log
		fmt.Fprintf(log, "Building %s/%s (cross-build is not a native execution test)\n", target[0], target[1])
		if err := command.Run(); err != nil {
			return fmt.Errorf("build %s/%s: %w", target[0], target[1], err)
		}
		archive := filepath.Join(output, fmt.Sprintf("atlas-mail-%s-%s.zip", target[0], target[1]))
		files := map[string]string{
			name: binary, "LICENSE": "LICENSE", "README.md": "README.md",
			"THIRD_PARTY_NOTICES.md":  "THIRD_PARTY_NOTICES.md",
			"DEPENDENCY-LICENSES.txt": filepath.Join(output, "DEPENDENCY-LICENSES.txt"),
			"sbom.cdx.json":           filepath.Join(output, "sbom.cdx.json"),
			"go.mod":                  "go.mod", "go.sum": "go.sum",
			"CONTRIBUTING.md":             "CONTRIBUTING.md",
			"docs/adr/0001-foundation.md": "docs/adr/0001-foundation.md",
			"docs/contracts-v1.md":        "docs/contracts-v1.md",
			"docs/security.md":            "docs/security.md",
			"docs/verification.md":        "docs/verification.md",
		}
		if err := writeArchive(archive, files); err != nil {
			return err
		}
		entry, err := fingerprint(archive)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, entry)
	}
	for _, name := range []string{"licenses.json", "sbom.cdx.json", "DEPENDENCY-LICENSES.txt"} {
		entry, err := fingerprint(filepath.Join(output, name))
		if err != nil {
			return err
		}
		artifacts = append(artifacts, entry)
	}
	revision, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("identify source revision: %w", err)
	}
	status, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return err
	}
	document := map[string]any{
		"timestamp": time.Now().UTC(), "goVersion": runtime.Version(),
		"hostOS": runtime.GOOS, "hostArch": runtime.GOARCH,
		"sourceRevision": strings.TrimSpace(string(revision)), "dirtyWorktree": len(status) != 0,
		"localGate": "passed", "raceEnabled": race, "allPlatforms": allPlatforms,
		"artifacts": artifacts,
		"notEstablished": []string{
			"native execution on other operating systems", "native keyring availability",
			"native installers", "browser matrix and screen-reader acceptance",
		},
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(summaryFile, append(data, '\n'), 0o600)
}

func fingerprint(path string) (artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return artifact{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return artifact{}, err
	}
	return artifact{File: filepath.Base(path), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func writeArchive(destination string, files map[string]string) (result error) {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		result = errors.Join(result, file.Close())
		if result != nil {
			result = errors.Join(result, os.Remove(destination))
		}
	}()
	writer := zip.NewWriter(file)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content, err := os.ReadFile(files[name])
		if err != nil {
			return errors.Join(err, writer.Close())
		}
		header := &zip.FileHeader{Name: "AtlasMail/" + name, Method: zip.Deflate}
		header.SetMode(0o644)
		if name == "atlas-mail" || name == "atlas-mail.exe" {
			header.SetMode(0o755)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return errors.Join(err, writer.Close())
		}
		if _, err := entry.Write(content); err != nil {
			return errors.Join(err, writer.Close())
		}
	}
	return writer.Close()
}
