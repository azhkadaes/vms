package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvironmentSetupPreservesExistingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".env.example", []byte("APP_ENVIRONMENT=development\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := setupEnvironment(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("APP_ENVIRONMENT=test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := setupEnvironment(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(".env")
	if err != nil || string(data) != "APP_ENVIRONMENT=test\n" {
		t.Fatal("existing configuration overwritten")
	}
}

// contractWorkspace lays out <root>/docs/contracts and <root>/backend (the working
// directory) the way the nested repository and its worktrees see them.
func contractWorkspace(t *testing.T, manifest, baseline string) string {
	t.Helper()
	root := t.TempDir()
	contracts := filepath.Join(root, "docs", "contracts")
	backend := filepath.Join(root, "backend")
	for _, dir := range []string{contracts, filepath.Join(backend, "docs")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(contracts, "manifest.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contracts, "source-baseline.json"), []byte(baseline), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(backend)
	return backend
}

const contractManifest = `{
  "sourceBaseline": "source-baseline.json",
  "operations": [
    {"number": 1, "id": "publicCatalogs", "method": "GET", "path": "/public/catalogs", "status": "implemented"},
    {"number": 2, "id": "publicUploadAttachment", "method": "POST", "path": "/public/attachments", "status": "missing"},
    {"number": 35, "id": "deleteSession", "method": "DELETE", "path": "/staff/session", "status": "implemented"}
  ]
}`

func baselineWith(paths string) string {
	return `{"spec": {
  "openapi": "3.1.1",
  "info": {"title": "VMS API", "version": "2.0.0-test"},
  "servers": [{"url": "/v1"}],
  "paths": ` + paths + `,
  "components": {"schemas": {"Catalogs": {"type": "object"}}}
}}`
}

const implementedPaths = `{
  "/public/catalogs": {"get": {"operationId": "publicCatalogs", "x-operation-number": 1, "responses": {"200": {"description": "ok"}}}},
  "/staff/session": {"delete": {"operationId": "deleteSession", "x-operation-number": 35, "responses": {"204": {"description": "signed out"}}}}
}`

func TestSwaggerPublishesOnlyImplementedOperationsFromTheBaseline(t *testing.T) {
	backend := contractWorkspace(t, contractManifest, baselineWith(implementedPaths))
	if err := run("swagger"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(backend, "docs", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		OpenAPI    string                               `json:"openapi"`
		Info       map[string]string                    `json:"info"`
		Paths      map[string]map[string]map[string]any `json:"paths"`
		Components map[string]any                       `json:"components"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.OpenAPI != "3.1.1" || document.Info["version"] != "2.0.0-test" || document.Components["schemas"] == nil {
		t.Fatalf("baseline metadata or components not carried over: %s", data)
	}
	if !strings.Contains(document.Info["description"], "2 of 3") {
		t.Fatalf("description does not state implemented coverage: %q", document.Info["description"])
	}
	if len(document.Paths) != 2 || document.Paths["/public/catalogs"]["get"]["operationId"] != "publicCatalogs" || document.Paths["/staff/session"]["delete"]["x-operation-number"] != float64(35) {
		t.Fatalf("served paths differ from implemented operations: %v", document.Paths)
	}
	if document.Paths["/public/attachments"] != nil {
		t.Fatal("missing operation advertised as served")
	}
}

func TestSwaggerReadsADraftContractsDirectory(t *testing.T) {
	backend := contractWorkspace(t, `{}`, `{}`)
	draft := t.TempDir()
	if err := os.WriteFile(filepath.Join(draft, "manifest.json"), []byte(contractManifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(draft, "source-baseline.json"), []byte(baselineWith(implementedPaths)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runWith("swagger", draft); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(backend, "docs", "openapi.json"))
	if err != nil || !strings.Contains(string(data), `"x-operation-number": 35`) {
		t.Fatalf("draft contracts not used: %v %s", err, data)
	}
}

func TestSwaggerRejectsImplementedOperationAbsentFromBaseline(t *testing.T) {
	contractWorkspace(t, contractManifest, baselineWith(`{
  "/public/catalogs": {"get": {"operationId": "publicCatalogs", "x-operation-number": 1, "responses": {}}}
}`))
	if err := run("swagger"); err == nil {
		t.Fatal("generated a document without the implemented DELETE /staff/session")
	}
}

func TestSwaggerRejectsBaselineOperationNotMarkedImplemented(t *testing.T) {
	contractWorkspace(t, contractManifest, baselineWith(`{
  "/public/catalogs": {"get": {"operationId": "publicCatalogs", "x-operation-number": 1, "responses": {}}},
  "/public/attachments": {"post": {"operationId": "publicUploadAttachment", "x-operation-number": 2, "responses": {}}},
  "/staff/session": {"delete": {"operationId": "deleteSession", "x-operation-number": 35, "responses": {}}}
}`))
	if err := run("swagger"); err == nil {
		t.Fatal("generated a document advertising an operation the manifest marks missing")
	}
}

func TestSwaggerRejectsBaselineNumberingDrift(t *testing.T) {
	contractWorkspace(t, contractManifest, baselineWith(`{
  "/public/catalogs": {"get": {"operationId": "publicCatalogs", "x-operation-number": 6, "responses": {}}},
  "/staff/session": {"delete": {"operationId": "deleteSession", "x-operation-number": 35, "responses": {}}}
}`))
	if err := run("swagger"); err == nil {
		t.Fatal("generated a document whose operation number differs from the manifest")
	}
}
