package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/Atonolo/AtlasMail/internal/model"
)

type review struct {
	version string
	license string
}

var reviewedModules = map[string]review{
	"al.essio.dev/pkg/shellescape":  {version: "v1.5.1", license: "MIT"},
	"github.com/danieljoos/wincred": {version: "v1.2.2", license: "MIT"},
	"github.com/davecgh/go-spew":    {version: "v1.1.1", license: "ISC"},
	"github.com/godbus/dbus/v5":     {version: "v5.1.0", license: "BSD-2-Clause"},
	"github.com/google/shlex":       {version: "v0.0.0-20191202100458-e7afc7fbc510", license: "Apache-2.0"},
	"github.com/pmezard/go-difflib": {version: "v1.0.0", license: "BSD-3-Clause"},
	"github.com/stretchr/objx":      {version: "v0.5.2", license: "MIT"},
	"github.com/stretchr/testify":   {version: "v1.9.0", license: "MIT"},
	"github.com/zalando/go-keyring": {version: "v0.2.6", license: "MIT AND Apache-2.0"},
	"golang.org/x/sys":              {version: "v0.26.0", license: "BSD-3-Clause"},
	"gopkg.in/yaml.v3":              {version: "v3.0.1", license: "MIT AND Apache-2.0"},
}

var reviewedLicenseHashes = map[string]string{
	"al.essio.dev/pkg/shellescape":  "01a4609ac3c7ac0a26d65a967c26c0deb146aa01bdcc8531048a70be52695556",
	"github.com/danieljoos/wincred": "1702a813fa31858c0d6c7788216fa3dd202472265716998024654c252fc7bb41",
	"github.com/davecgh/go-spew":    "1b93a317849ee09d3d7e4f1d20c2b78ddb230b4becb12d7c224c927b9d470251",
	"github.com/godbus/dbus/v5":     "e4646a82a976369d7ae8f6ed5c11d35dc0af18433a8ccc24c85b459ad8b95128",
	"github.com/google/shlex":       "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30",
	"github.com/pmezard/go-difflib": "2eb550be6801c1ea434feba53bf6d12e7c71c90253e0a9de4a4f46cf88b56477",
	"github.com/stretchr/objx":      "b2663894033a05fd80261176cd8da1d72546e25842d5c1abcc852ca23b6b61b0",
	"github.com/stretchr/testify":   "f8e536c1c7b695810427095dc85f5f80d44ff7c10535e8a9486cf393e2599189",
	"github.com/zalando/go-keyring": "14bc00f1a298eb6af250216b9e60027a4386c8a84fa2c7ddc2b5c10908e06d0d",
	"golang.org/x/sys":              "911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad",
	"gopkg.in/yaml.v3":              "d18f6323b71b0b768bb5e9616e36da390fbd39369a81807cca352de4e4e6aa0b",
}

const goLicenseHash = "911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad"

var reviewedToolchainModules = map[string]string{
	"golang.org/x/crypto": "v0.52.1-0.20260526024921-9beb694f9766",
	"golang.org/x/net":    "v0.55.1-0.20260731170536-c1d18010be90",
	"golang.org/x/sys":    "v0.45.0",
	"golang.org/x/text":   "v0.37.0",
}

var reviewedBuildModules = map[string]string{
	"github.com/google/pprof":            "v0.0.0-20260507013755-92041b743c96",
	"github.com/ianlancetaylor/demangle": "v0.0.0-20250417193237-f615e6bd150b",
	"golang.org/x/arch":                  "v0.27.1-0.20260521044007-9c1a596a2c97",
	"golang.org/x/build":                 "v0.0.0-20260522210304-d55d0041b921",
	"golang.org/x/mod":                   "v0.36.1-0.20260813213634-8569e2639ca1",
	"golang.org/x/sync":                  "v0.20.0",
	"golang.org/x/sys":                   "v0.45.0",
	"golang.org/x/telemetry":             "v0.0.0-20260519152614-eab6ae52b5e2",
	"golang.org/x/term":                  "v0.43.0",
	"golang.org/x/text":                  "v0.37.0",
	"golang.org/x/tools":                 "v0.45.1-0.20260917185718-3dd9077b8066",
	"rsc.io/markdown":                    "v0.0.0-20240306144322-0bf8f97ee8ef",
}

var toolchainLicenseHashes = map[string]string{
	"github.com/google/pprof":            "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30",
	"github.com/ianlancetaylor/demangle": "b73277f7730993ef8e7f80aed83e5309762fed4d77eced982abb2c00657c074d",
	"golang.org/x/arch":                  "8cda009bd927676a95ee7ce2dae921442ac4d9584d041b56ecfcba6839032f0b",
	"rsc.io/markdown":                    "2d36597f7117c38b006835ae7f537487207d8ec407aa9d9980794b2030cbc067",
}

const svgpanLicenseHash = "154c946c17de61ca71d28e13673d76a5798e6e25995a7397fbae157afc8f62b4"

type module struct {
	Path    string  `json:"Path"`
	Version string  `json:"Version"`
	Dir     string  `json:"Dir"`
	Main    bool    `json:"Main"`
	Replace *module `json:"Replace"`
}

type component struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	PURL     string `json:"purl"`
	Licenses []struct {
		Expression string `json:"expression"`
	} `json:"licenses"`
}

type bom struct {
	Schema       string `json:"$schema"`
	BOMFormat    string `json:"bomFormat"`
	SpecVersion  string `json:"specVersion"`
	SerialNumber string `json:"serialNumber"`
	Version      int    `json:"version"`
	Metadata     struct {
		Timestamp string    `json:"timestamp"`
		Component component `json:"component"`
	} `json:"metadata"`
	Components []component `json:"components"`
}

func main() {
	sbom := flag.Bool("sbom", false, "write a CycloneDX JSON SBOM after verifying module licenses")
	notices := flag.Bool("notices", false, "write redistribution license texts after checking dependencies")
	flag.Parse()
	if *sbom && *notices {
		fail(errors.New("choose only one output format"))
	}
	verify := exec.Command("go", "mod", "verify")
	verify.Stderr = os.Stderr
	if output, err := verify.Output(); err != nil {
		fail(fmt.Errorf("module content verification failed: %s: %w", output, err))
	}
	modules, err := listModules()
	if err != nil {
		fail(err)
	}
	components, err := verifyModules(modules)
	if err != nil {
		fail(err)
	}
	toolchain, err := verifyToolchain()
	if err != nil {
		fail(err)
	}
	components, err = uniqueComponents(append(components, toolchain...))
	if err != nil {
		fail(err)
	}
	if *notices {
		if err := writeNotices(modules); err != nil {
			fail(err)
		}
		return
	}
	if *sbom {
		if err := writeSBOM(components); err != nil {
			fail(err)
		}
		return
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"status":     "passed",
		"project":    "Atlas Mail",
		"license":    "MPL-2.0",
		"components": components,
	}); err != nil {
		fail(err)
	}
}

func listModules() ([]module, error) {
	command := exec.Command("go", "list", "-m", "-json", "all")
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list locked Go modules: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	var modules []module
	for {
		var item module
		err := decoder.Decode(&item)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode Go module graph: %w", err)
		}
		if !item.Main {
			modules = append(modules, item)
		}
	}
	return modules, nil
}

func uniqueComponents(entries []component) ([]component, error) {
	result := make([]component, 0, len(entries))
	seen := make(map[string]component)
	for _, entry := range entries {
		if previous, exists := seen[entry.PURL]; exists {
			if !reflect.DeepEqual(previous, entry) {
				return nil, fmt.Errorf("conflicting component reviews for %s", entry.PURL)
			}
			continue
		}
		seen[entry.PURL] = entry
		result = append(result, entry)
	}
	return result, nil
}

func verifyModules(modules []module) ([]component, error) {
	components := make([]component, 0, len(modules))
	for _, item := range modules {
		if item.Replace != nil {
			return nil, fmt.Errorf("unreviewed module replacement for %s", item.Path)
		}
		reviewed, exists := reviewedModules[item.Path]
		if !exists {
			return nil, fmt.Errorf("unreviewed Go module %s@%s", item.Path, item.Version)
		}
		if item.Version != reviewed.version {
			return nil, fmt.Errorf("unreviewed Go module version %s@%s (reviewed %s)", item.Path, item.Version, reviewed.version)
		}
		if item.Dir == "" {
			return nil, fmt.Errorf("module source is unavailable for %s@%s", item.Path, item.Version)
		}
		if err := verifyLicenseHash(item.Dir, reviewedLicenseHashes[item.Path]); err != nil {
			return nil, fmt.Errorf("%s@%s: %w", item.Path, item.Version, err)
		}
		pkgURL := "pkg:golang/" + item.Path + "@" + item.Version
		components = append(components, component{
			Type:    "library",
			Name:    item.Path,
			Version: item.Version,
			PURL:    pkgURL,
			Licenses: []struct {
				Expression string `json:"expression"`
			}{{Expression: reviewed.license}},
		})
	}
	return components, nil
}

func verifyLicenseHash(directory, expected string) error {
	if expected == "" {
		return errors.New("missing license fingerprint review")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := strings.ToUpper(entry.Name())
		if entry.IsDir() || !(strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "COPYING")) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(content)) == expected {
			return nil
		}
	}
	return errors.New("license text does not match the reviewed SHA-256 fingerprint")
}

func verifyToolchain() ([]component, error) {
	const reviewedVersion = "go1.27.3"
	if runtime.Version() != reviewedVersion {
		return nil, fmt.Errorf("unreviewed Go toolchain %s (reviewed %s)", runtime.Version(), reviewedVersion)
	}
	content, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		return nil, fmt.Errorf("read Go toolchain license: %w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != goLicenseHash {
		return nil, errors.New("Go toolchain does not match the reviewed BSD-3-Clause license")
	}
	version := runtime.Version()
	components := []component{{
		Type:    "framework",
		Name:    "Go standard library and toolchain",
		Version: version,
		PURL:    "pkg:generic/go-toolchain@" + version,
		Licenses: []struct {
			Expression string `json:"expression"`
		}{{Expression: "BSD-3-Clause"}},
	}}

	for _, inventory := range []struct {
		root     string
		versions map[string]string
	}{
		{filepath.Join(runtime.GOROOT(), "src", "vendor"), reviewedToolchainModules},
		{filepath.Join(runtime.GOROOT(), "src", "cmd", "vendor"), reviewedBuildModules},
	} {
		entries, err := verifyVendorInventory(inventory.root, inventory.versions)
		if err != nil {
			return nil, err
		}
		components = append(components, entries...)
	}
	svgpan := filepath.Join(runtime.GOROOT(), "src", "cmd", "vendor", "github.com", "google", "pprof", "third_party", "svgpan")
	if err := verifyLicenseHash(svgpan, svgpanLicenseHash); err != nil {
		return nil, fmt.Errorf("Go pprof SVGpan license: %w", err)
	}
	components = append(components, component{
		Type: "library", Name: "Go pprof SVGpan", Version: reviewedBuildModules["github.com/google/pprof"],
		PURL: "pkg:generic/svgpan@" + version,
		Licenses: []struct {
			Expression string `json:"expression"`
		}{{Expression: "BSD-3-Clause"}},
	})
	return components, nil
}

func verifyVendorInventory(vendorRoot string, versions map[string]string) ([]component, error) {
	var components []component
	moduleFile, err := os.Open(filepath.Join(vendorRoot, "modules.txt"))
	if err != nil {
		return nil, fmt.Errorf("read Go toolchain vendor list: %w", err)
	}
	defer moduleFile.Close()
	found := make(map[string]bool)
	scanner := bufio.NewScanner(moduleFile)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || fields[0] != "#" {
			continue
		}
		modulePath := fields[1]
		expectedVersion, reviewed := versions[modulePath]
		if !reviewed {
			return nil, fmt.Errorf("unreviewed Go toolchain module %s@%s", modulePath, fields[2])
		}
		if fields[2] != expectedVersion {
			return nil, fmt.Errorf("unreviewed Go toolchain module version %s@%s (reviewed %s)", modulePath, fields[2], expectedVersion)
		}
		hash := toolchainLicenseHashes[modulePath]
		if hash == "" {
			hash = goLicenseHash
		}
		if err := verifyLicenseHash(filepath.Join(vendorRoot, filepath.FromSlash(modulePath)), hash); err != nil {
			return nil, fmt.Errorf("Go toolchain module %s@%s: %w", modulePath, fields[2], err)
		}
		found[modulePath] = true
		license := "BSD-3-Clause"
		if modulePath == "github.com/google/pprof" {
			license = "Apache-2.0"
		}
		components = append(components, component{
			Type:    "library",
			Name:    "Go toolchain vendor: " + modulePath,
			Version: fields[2],
			PURL:    "pkg:golang/" + modulePath + "@" + fields[2],
			Licenses: []struct {
				Expression string `json:"expression"`
			}{{Expression: license}},
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read Go toolchain vendor list: %w", err)
	}
	for modulePath := range versions {
		if !found[modulePath] {
			return nil, fmt.Errorf("Go toolchain module %s is missing from vendor list", modulePath)
		}
	}
	return components, nil
}

func writeNotices(modules []module) error {
	paths := []string{filepath.Join(runtime.GOROOT(), "LICENSE")}
	for _, item := range modules {
		entries, err := os.ReadDir(item.Dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := strings.ToUpper(entry.Name())
			if !entry.IsDir() && (strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "COPYING") || name == "NOTICE") {
				paths = append(paths, filepath.Join(item.Dir, entry.Name()))
			}
		}
		if item.Path == "github.com/zalando/go-keyring" {
			// This file retains Google's Apache-2.0 copyright and license notice.
			paths = append(paths, filepath.Join(item.Dir, "keyring_darwin.go"))
		}
	}
	for _, root := range []string{
		filepath.Join(runtime.GOROOT(), "src", "vendor"),
		filepath.Join(runtime.GOROOT(), "src", "cmd", "vendor"),
	} {
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := strings.ToUpper(entry.Name())
			if !entry.IsDir() && (name == "LICENSE" || name == "COPYING" || name == "NOTICE" || name == "PATENTS") {
				paths = append(paths, path)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, "\n--- %s ---\n%s\n", path, content); err != nil {
			return err
		}
	}
	return nil
}
func writeSBOM(components []component) error {
	serial, err := model.NewIdentifier()
	if err != nil {
		return fmt.Errorf("generate SBOM identifier: %w", err)
	}
	document := bom{
		Schema:       "http://cyclonedx.org/schema/bom-1.5.schema.json",
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: "urn:uuid:" + serial,
		Version:      1,
		Components:   components,
	}
	document.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	document.Metadata.Component = component{
		Type:    "application",
		Name:    "Atlas Mail",
		Version: "0.1.0",
		Licenses: []struct {
			Expression string `json:"expression"`
		}{{Expression: "MPL-2.0"}},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "License audit failed: %v\n", err)
	os.Exit(1)
}
