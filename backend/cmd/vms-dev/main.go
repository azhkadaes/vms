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
	contracts := flag.String("contracts", defaultContracts, "swagger: directory holding manifest.json and source-baseline.json")
	flag.Parse()
	if err := runWith(*mode, *contracts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// defaultContracts is the shared contract pipeline seen from the backend checkout or a
// sibling worktree.
var defaultContracts = filepath.Join("..", "docs", "contracts")

func run(mode string) error { return runWith(mode, defaultContracts) }

// runWith runs a mode; swagger reads its contract files from contracts, so a worktree can
// generate from a draft copy before the shared pipeline is updated.
func runWith(mode, contracts string) error {
	switch mode {
	case "env-setup":
		return setupEnvironment()
	case "swagger":
		return generateOpenAPI(contracts)
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

// generateOpenAPI writes docs/openapi.json from the contract pipeline in contracts:
// manifest.json lists all 40 operations with their status, and source-baseline.json holds
// the OpenAPI shapes of the implemented ones. Only implemented operations are served,
// and the two files must agree exactly, so an unregistered or stale operation fails here.
func generateOpenAPI(contracts string) error {
	manifestData, err := os.ReadFile(filepath.Join(contracts, "manifest.json"))
	if err != nil {
		return fmt.Errorf("cannot read %s; run from the backend directory", filepath.Join(contracts, "manifest.json"))
	}
	var manifest struct {
		SourceBaseline string `json:"sourceBaseline"`
		Operations     []struct {
			Number int    `json:"number"`
			ID     string `json:"id"`
			Method string `json:"method"`
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return errors.New("invalid contract manifest")
	}
	if manifest.SourceBaseline != "source-baseline.json" {
		return errors.New("unsupported source baseline; review documentation generation")
	}
	baselineData, err := os.ReadFile(filepath.Join(contracts, manifest.SourceBaseline))
	if err != nil {
		return errors.New("cannot read the contract source baseline")
	}
	var baseline struct {
		Spec map[string]any `json:"spec"`
	}
	if err := json.Unmarshal(baselineData, &baseline); err != nil || baseline.Spec == nil {
		return errors.New("invalid contract source baseline")
	}
	document := baseline.Spec
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return errors.New("source baseline paths are missing")
	}
	documented := 0
	for _, item := range paths {
		operations, ok := item.(map[string]any)
		if !ok {
			return errors.New("invalid source baseline path item")
		}
		for method := range operations {
			if httpMethods[method] {
				documented++
			}
		}
	}
	implemented := 0
	for _, operation := range manifest.Operations {
		if operation.Status != "implemented" {
			continue
		}
		item, _ := paths[operation.Path].(map[string]any)
		definition, ok := item[strings.ToLower(operation.Method)].(map[string]any)
		if !ok {
			return fmt.Errorf("implemented operation %d (%s %s) is missing from the source baseline", operation.Number, operation.Method, operation.Path)
		}
		number, _ := definition["x-operation-number"].(float64)
		if definition["operationId"] != operation.ID || int(number) != operation.Number {
			return fmt.Errorf("source baseline operation %s %s does not match manifest operation %d %s", operation.Method, operation.Path, operation.Number, operation.ID)
		}
		implemented++
	}
	if implemented == 0 {
		return errors.New("no implemented operations")
	}
	if documented != implemented {
		return errors.New("source baseline documents operations the manifest does not mark implemented")
	}
	info, ok := document["info"].(map[string]any)
	if !ok {
		return errors.New("contract version metadata is missing")
	}
	version, ok := info["version"].(string)
	if !ok || version == "" {
		return errors.New("contract version metadata is missing")
	}
	document["info"] = map[string]any{
		"title":       "VMS API",
		"version":     version,
		"description": fmt.Sprintf("Implemented operations of the frontend-driven REST contract: %d of %d. Numbers (x-operation-number) follow docs/contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md; docs/contracts/manifest.json lists every operation and its status.", implemented, len(manifest.Operations)),
	}
	output, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("docs", "openapi.json"), append(output, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Generated docs/openapi.json: %d of %d contract operations implemented, OpenAPI 3.1.1.\n", implemented, len(manifest.Operations))
	return nil
}

var httpMethods = map[string]bool{"get": true, "put": true, "post": true, "delete": true, "options": true, "head": true, "patch": true, "trace": true}
