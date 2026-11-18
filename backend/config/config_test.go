package config_test

import (
	"encoding/base64"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azhkadaes/vms/backend/config"
)

func TestConfigurationFailsClosed(t *testing.T) {
	base := map[string]string{
		"APP_ENVIRONMENT": "development",
		"DB_HOST":         "127.0.0.1", "DB_PORT": "55450", "DB_USER": "vms_api", "DB_PASSWORD": "private-password", "DB_NAME": "vms_stage1", "DB_SSL_MODE": "disable",
		"RATE_LIMIT_KEY": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
	}
	for _, tc := range []struct{ name, key, value string }{
		{"missing database", "DB_NAME", ""},
		{"missing user", "DB_USER", ""},
		{"missing password", "DB_PASSWORD", ""},
		{"remote plaintext", "DB_HOST", "example.org"},
		{"host injection", "DB_HOST", "localhost?host=example.org"},
		{"TLS injection", "DB_SSL_MODE", "disable&host=example.org"},
		{"invalid database port", "DB_PORT", "65536"},
		{"invalid app port", "APP_PORT", "0"},
		{"host includes port", "APP_HOST", "localhost:8080"},
		{"missing limiter key", "RATE_LIMIT_KEY", ""},
		{"short limiter key", "RATE_LIMIT_KEY", "eA=="},
		{"unknown environment", "APP_ENVIRONMENT", "prodution"},
		{"incomplete privacy notice", "VMS_NOTICE_VERSION", "v1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := make(map[string]string)
			for k, v := range base {
				values[k] = v
			}
			values[tc.key] = tc.value
			_, err := config.LoadFrom(func(key string) string { return values[key] })
			if err == nil {
				t.Fatal("invalid configuration accepted")
			}
			if strings.Contains(err.Error(), "private-password") {
				t.Fatal("error leaked connection secret")
			}
		})
	}
}

func TestDotEnvPreservesProcessOverrides(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("APP_ENVIRONMENT", "test")
	if err := os.WriteFile(".env", []byte("APP_ENVIRONMENT=development\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadEnvironment(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("APP_ENVIRONMENT") != "test" {
		t.Fatal(".env overrode process configuration")
	}
}

func TestSwaggerRequiresExplicitCredentials(t *testing.T) {
	values := map[string]string{
		"DB_HOST": "localhost", "DB_USER": "vms_api", "DB_PASSWORD": "secret", "DB_NAME": "vms_stage1", "DB_SSL_MODE": "disable",
		"RATE_LIMIT_KEY":   base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
		"SWAGGER_USERNAME": "docs",
	}
	get := func(key string) string { return values[key] }
	if _, err := config.LoadFrom(get); err == nil {
		t.Fatal("enabled Swagger accepted absent credentials")
	}
	values["SWAGGER_PASSWORD"] = "private-docs-password"
	if cfg, err := config.LoadFrom(get); err != nil || !cfg.SwaggerEnabled {
		t.Fatal("explicit Swagger configuration rejected")
	}
}

func TestConfigurationUsesSelectedBounds(t *testing.T) {
	values := map[string]string{
		"DB_HOST": "db", "DB_USER": "vms_api", "DB_PASSWORD": "secret", "DB_NAME": "vms_stage1", "DB_SSL_MODE": "disable",
		"RATE_LIMIT_KEY":     base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
		"VMS_NOTICE_VERSION": "v1", "VMS_NOTICE_ID": "Pemberitahuan yang disetujui", "VMS_NOTICE_EN": "Approved notice",
	}
	cfg, err := config.LoadFrom(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JSONMaxBytes != 256*1024 || cfg.ExecutionTimeout.Seconds() != 15 {
		t.Fatal("request bounds drifted")
	}
	if cfg.Notice == nil || cfg.Notice.Version != "v1" {
		t.Fatal("configured notice omitted")
	}
	values["APP_ENVIRONMENT"] = "production"
	if _, err := config.LoadFrom(func(key string) string { return values[key] }); err == nil {
		t.Fatal("production plaintext database accepted")
	}
}

func TestOriginalSettingsBuildConnectionWithoutCredentialEscaping(t *testing.T) {
	values := map[string]string{
		"APP_HOST": "::1", "APP_PORT": "8090",
		"DB_HOST": "::1", "DB_PORT": "55450", "DB_USER": "vms_api",
		"DB_PASSWORD": "p@ss:/?#% &", "DB_NAME": "vms_stage1", "DB_SSL_MODE": "disable",
		"RATE_LIMIT_KEY": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
	}
	get := func(key string) string { return values[key] }
	cfg, err := config.LoadFrom(get)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		t.Fatal("generated connection is invalid")
	}
	password, _ := u.User.Password()
	if password != values["DB_PASSWORD"] || u.Host != "[::1]:55450" || cfg.ListenAddress != "[::1]:8090" || cfg.Environment != "development" || cfg.SwaggerEnabled {
		t.Fatal("original settings or credential encoding were not preserved")
	}
	migrationURL, err := config.DatabaseURLFrom(get)
	if err != nil || migrationURL != cfg.DatabaseURL {
		t.Fatal("API and migration database settings differ")
	}
	// Only synthetic temporary files are used; never read local credentials.
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte(values["DB_PASSWORD"]+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	values["DB_PASSWORD_FILE"] = path
	if _, err := config.LoadFrom(get); err == nil {
		t.Fatal("conflicting password sources accepted")
	}
	values["DB_PASSWORD"] = ""
	fromFile, err := config.LoadFrom(get)
	if err != nil || fromFile.DatabaseURL != cfg.DatabaseURL {
		t.Fatal("optional Docker password file differs from direct configuration")
	}
	values["DB_SSL_MODE"] = "verify-full"
	values["APP_ENVIRONMENT"] = "production"
	if _, err := config.LoadFrom(get); err != nil {
		t.Fatal("verified TLS production configuration rejected")
	}
}
