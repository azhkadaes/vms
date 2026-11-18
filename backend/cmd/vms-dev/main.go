package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	mode := flag.String("mode", "run", "env-setup, build, run, start or swagger")
	flag.Parse()
	if err := run(*mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode string) error {
	switch mode {
	case "env-setup":
		return setupEnvironment()
	case "swagger":
		return generateOpenAPI()
	case "run":
		return command("go", "run", "./cmd")
	case "build", "start":
		if mode == "start" {
			if err := setupEnvironment(); err != nil {
				return err
			}
		}
		if err := os.MkdirAll("bin", 0755); err != nil {
			return err
		}
		binary := filepath.Join("bin", "vms-api")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		if err := command("go", "build", "-o", binary, "./cmd"); err != nil {
			return err
		}
		if mode == "start" {
			path, err := filepath.Abs(binary)
			if err != nil {
				return err
			}
			return command(path)
		}
		return nil
	default:
		return errors.New("unknown VMS development mode")
	}
}

func command(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func setupEnvironment() error {
	data, err := os.ReadFile(".env.example")
	if err != nil {
		return errors.New("run from the backend directory containing .env.example")
	}
	file, err := os.OpenFile(".env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		fmt.Println("Existing .env preserved; check its VMS settings before startup.")
		return nil
	}
	if err != nil {
		return errors.New("cannot create .env")
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.New("cannot finish writing .env; inspect it before startup")
	}
	fmt.Println("Created .env. Configure the database, rate key and optional Swagger credentials before startup.")
	return nil
}

func generateOpenAPI() error {
	data, err := os.ReadFile(filepath.Join("..", "docs", "BACKEND-OPENAPI-2026-11-12.json"))
	if err != nil {
		return errors.New("cannot read the approved VMS planning contract from ../docs")
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return errors.New("invalid approved OpenAPI document")
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return errors.New("approved operation paths are missing")
	}
	manifestData, err := os.ReadFile(filepath.Join("..", "docs", "contracts", "manifest.json"))
	if err != nil {
		return errors.New("cannot read the reviewed contract implementation manifest")
	}
	var manifest struct {
		SourceBaseline string         `json:"sourceBaseline"`
		SourceSpec     map[string]any `json:"spec"`
		Operations     []struct {
			ID          string `json:"id"`
			Method      string `json:"method"`
			Path        string `json:"path"`
			Implemented bool   `json:"implemented"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return errors.New("invalid contract implementation manifest")
	}
	// Contract-only phase changes must not advertise unimplemented auth routes or
	// hide legacy routes that still exist. The reviewed source snapshot is evidence
	// for the current binary, separate from the Phase 1 target document.
	if manifest.SourceBaseline != "" {
		if manifest.SourceBaseline != "source-baseline.json" {
			return errors.New("unsupported source baseline; review documentation generation")
		}
		baselineData, err := os.ReadFile(filepath.Join("..", "docs", "contracts", manifest.SourceBaseline))
		if err != nil {
			return errors.New("cannot read the reviewed current-source baseline")
		}
		if err := json.Unmarshal(baselineData, &manifest); err != nil || manifest.SourceSpec == nil {
			return errors.New("invalid current-source baseline")
		}
		document = manifest.SourceSpec
		paths, ok = document["paths"].(map[string]any)
		if !ok {
			return errors.New("current-source baseline paths are missing")
		}
	}
	implemented := map[string]any{}
	count := 0
	for _, operation := range manifest.Operations {
		if !operation.Implemented {
			continue
		}
		item, ok := paths[operation.Path].(map[string]any)
		if !ok {
			return errors.New("implemented contract path is missing")
		}
		method := strings.ToLower(operation.Method)
		definition, ok := item[method].(map[string]any)
		if !ok || definition["operationId"] != operation.ID {
			return errors.New("implemented contract operation does not match its manifest")
		}
		if implemented[operation.Path] == nil {
			implemented[operation.Path] = map[string]any{}
		}
		selected := implemented[operation.Path].(map[string]any)
		if selected[method] != nil {
			return errors.New("duplicate implemented contract operation")
		}
		selected[method] = definition
		count++
	}
	if count == 0 {
		return errors.New("no reviewed implemented operations")
	}
	document["paths"] = implemented
	info, ok := document["info"].(map[string]any)
	if !ok {
		return errors.New("contract version metadata is missing")
	}
	version, ok := info["version"].(string)
	if !ok || version == "" {
		return errors.New("contract version metadata is missing")
	}
	document["info"] = map[string]any{"title": "VMS current-source API", "version": version, "description": "Current-source evidence only, including superseded session routes. The Phase 1 target in docs/contracts/API-CONTRACT.md is separate; this document does not claim Phase 1 auth is implemented."}
	output, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile("docs/openapi.json", append(output, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Generated docs/openapi.json: %d source-registered operations, OpenAPI 3.1.1.\n", count)
	return nil
}
