package main

import (
	"os"
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
