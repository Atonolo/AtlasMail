package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	"github.com/zalando/go-keyring": {version: "v0.2.6", license: "MIT"},
	"golang.org/x/sys":              {version: "v0.26.0", license: "BSD-3-Clause"},
	"gopkg.in/yaml.v3":              {version: "v3.0.1", license: "MIT OR Apache-2.0"},
}

var reviewedToolchainModules = map[string]string{
	"golang.org/x/crypto": "v0.52.1-0.20260526024921-9beb694f9766",
	"golang.org/x/net":    "v0.55.1-0.20260731170536-c1d18010be90",
	"golang.org/x/sys":    "v0.45.0",
	"golang.org/x/text":   "v0.37.0",
}

type module struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
	Main    bool   `json:"Main"`
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
	flag.Parse()
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
	components = append(components, toolchain...)
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

func verifyModules(modules []module) ([]component, error) {
	components := make([]component, 0, len(modules))
	for _, item := range modules {
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
		license, err := verifyLicenseFile(item.Dir, reviewed.license)
		if err != nil {
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
			}{{Expression: license}},
		})
	}
	return components, nil
}

func verifyLicenseFile(directory, expression string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	choices := strings.Split(expression, " OR ")
	for _, entry := range entries {
		name := strings.ToUpper(entry.Name())
		if entry.IsDir() || !(strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "COPYING")) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return "", err
		}
		text := strings.ToLower(string(content))
		for _, license := range choices {
			switch license {
			case "MIT":
				if strings.Contains(text, "permission is hereby granted") || strings.Contains(text, "mit license") {
					return license, nil
				}
			case "ISC":
				if strings.Contains(text, "isc license") && strings.Contains(text, "permission to use, copy, modify") {
					return license, nil
				}
			case "BSD-2-Clause", "BSD-3-Clause":
				if strings.Contains(text, "redistribution and use in source and binary forms") {
					return license, nil
				}
			case "Apache-2.0":
				if strings.Contains(text, "apache license") && strings.Contains(text, "version 2.0") {
					return license, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no license text matches %s", expression)
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
	if !strings.Contains(strings.ToLower(string(content)), "redistribution and use in source and binary forms") {
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

	vendorRoot := filepath.Join(runtime.GOROOT(), "src", "vendor")
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
		expectedVersion, reviewed := reviewedToolchainModules[modulePath]
		if !reviewed {
			return nil, fmt.Errorf("unreviewed Go toolchain module %s@%s", modulePath, fields[2])
		}
		if fields[2] != expectedVersion {
			return nil, fmt.Errorf("unreviewed Go toolchain module version %s@%s (reviewed %s)", modulePath, fields[2], expectedVersion)
		}
		license, err := verifyLicenseFile(filepath.Join(vendorRoot, filepath.FromSlash(modulePath)), "BSD-3-Clause")
		if err != nil {
			return nil, fmt.Errorf("Go toolchain module %s@%s: %w", modulePath, fields[2], err)
		}
		found[modulePath] = true
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
	for modulePath := range reviewedToolchainModules {
		if !found[modulePath] {
			return nil, fmt.Errorf("Go toolchain module %s is missing from vendor list", modulePath)
		}
	}
	return components, nil
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
