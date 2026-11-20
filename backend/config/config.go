package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
)

// Notice contains approved, published notice text; absence disables genuine registration.
type Notice struct{ Version, ID, EN string }

// Config is private process configuration. It must never be serialized or logged.
type Config struct {
	Environment      string
	ListenAddress    string
	DatabaseURL      string
	RateKey          []byte
	TrustedProxies   []string
	Notice           *Notice
	JSONMaxBytes     int
	ExecutionTimeout time.Duration
	SwaggerEnabled   bool
	SwaggerUsername  string
	SwaggerPassword  string
	Auth             *Auth
	Uploads          *Uploads
	AdmissionQRKey   []byte
}

// Load reads VMS configuration from the environment with optional .env defaults.
func Load() (Config, error) {
	if err := LoadEnvironment(); err != nil {
		return Config{}, err
	}
	return LoadFrom(os.Getenv)
}

// LoadEnvironment preserves process variables and treats a missing .env as optional.
func LoadEnvironment() error {
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot load .env; check its syntax and permissions")
	}
	return nil
}

// LoadFrom supports isolated configuration tests without changing process state.
func LoadFrom(get func(string) string) (Config, error) {
	cfg := Config{Environment: get("APP_ENVIRONMENT"), JSONMaxBytes: 256 * 1024, ExecutionTimeout: 15 * time.Second}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.Environment != "development" && cfg.Environment != "test" && cfg.Environment != "production" {
		return Config{}, errors.New("APP_ENVIRONMENT must be development, test or production")
	}
	host, port := get("APP_HOST"), get("APP_PORT")
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "8080"
	}
	if !validHost(host) || !validPort(port) {
		return Config{}, errors.New("APP_HOST must be a hostname/IP without a port; APP_PORT must be 1-65535")
	}
	cfg.ListenAddress = net.JoinHostPort(host, port)
	dsn, err := databaseURL(get, cfg.Environment)
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseURL = dsn
	cfg.SwaggerUsername = get("SWAGGER_USERNAME")
	cfg.SwaggerEnabled = cfg.SwaggerUsername != "" || get("SWAGGER_PASSWORD") != "" || get("SWAGGER_PASSWORD_FILE") != ""
	if cfg.SwaggerEnabled {
		cfg.SwaggerPassword, err = secret(get, "SWAGGER_PASSWORD")
		if err != nil || cfg.SwaggerUsername == "" || cfg.SwaggerPassword == "" {
			return Config{}, errors.New("configure both SWAGGER_USERNAME and SWAGGER_PASSWORD or its _FILE variant")
		}
	}
	key, err := secret(get, "RATE_LIMIT_KEY")
	if err != nil {
		return Config{}, err
	}
	cfg.RateKey, err = base64.StdEncoding.DecodeString(key)
	if err != nil || len(cfg.RateKey) < 32 {
		return Config{}, errors.New("RATE_LIMIT_KEY must be base64 encoding at least 32 random bytes")
	}
	for _, raw := range strings.Split(get("VMS_TRUSTED_PROXIES"), ",") {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if net.ParseIP(value) == nil {
			if _, _, err := net.ParseCIDR(value); err != nil {
				return Config{}, errors.New("VMS_TRUSTED_PROXIES contains an invalid IP/CIDR")
			}
		}
		cfg.TrustedProxies = append(cfg.TrustedProxies, value)
	}
	version, id, en := get("VMS_NOTICE_VERSION"), get("VMS_NOTICE_ID"), get("VMS_NOTICE_EN")
	if version != "" || id != "" || en != "" {
		if !utf8.ValidString(version+id+en) || strings.TrimSpace(version) == "" || utf8.RuneCountInString(version) > 120 || strings.TrimSpace(id) == "" || strings.TrimSpace(en) == "" {
			return Config{}, errors.New("published notice requires a valid version and both ID/EN texts")
		}
		cfg.Notice = &Notice{Version: version, ID: id, EN: en}
	}
	cfg.Auth, err = authFrom(get, cfg.Environment)
	if err != nil {
		return Config{}, err
	}
	others := [][]byte{cfg.RateKey}
	if cfg.Auth != nil {
		others = append(others, cfg.Auth.JWTKey, cfg.Auth.CSRFKey, cfg.Auth.RecoveryKey, []byte(cfg.Auth.GatewayKey))
	}
	cfg.Uploads, err = uploadsFrom(get, others...)
	if err != nil {
		return Config{}, err
	}
	if cfg.Uploads != nil {
		others = append(others, cfg.Uploads.TokenKey)
	}
	cfg.AdmissionQRKey, err = admissionQRKeyFrom(get, others)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// DatabaseURLFrom loads only the connection settings needed by explicit migration tooling.
func DatabaseURLFrom(get func(string) string) (string, error) {
	environment := get("APP_ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}
	if environment != "development" && environment != "test" && environment != "production" {
		return "", errors.New("invalid APP_ENVIRONMENT")
	}
	return databaseURL(get, environment)
}

func databaseURL(get func(string) string, environment string) (string, error) {
	host, port := get("DB_HOST"), get("DB_PORT")
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if !validHost(host) || !validPort(port) {
		return "", errors.New("DB_HOST must be a hostname/IP without a port; DB_PORT must be 1-65535")
	}
	user, name := get("DB_USER"), get("DB_NAME")
	if strings.TrimSpace(user) == "" || strings.TrimSpace(name) == "" {
		return "", errors.New("DB_USER and DB_NAME are required")
	}
	password, err := secret(get, "DB_PASSWORD")
	if err != nil {
		return "", err
	}
	mode := get("DB_SSL_MODE")
	if mode == "" {
		mode = "verify-full"
	}
	local := host == "db" || host == "localhost" || net.ParseIP(host).IsLoopback()
	if mode != "verify-full" && !(environment != "production" && local && mode == "disable") {
		return "", errors.New("database requires verify-full TLS; plaintext is restricted to isolated development loopback/db")
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(user, password), Host: net.JoinHostPort(host, port), Path: "/" + name}
	u.RawQuery = url.Values{"sslmode": {mode}}.Encode()
	return u.String(), nil
}

func validPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}

func validHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	if host == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-') {
				return false
			}
		}
	}
	return true
}

func secret(get func(string) string, name string) (string, error) {
	value, path := get(name), get(name+"_FILE")
	if value != "" && path != "" {
		return "", fmt.Errorf("configure only %s or %s_FILE", name, name)
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("cannot read %s_FILE", name)
		}
		value = strings.TrimSpace(string(data))
	}
	if value == "" {
		return "", fmt.Errorf("%s or %s_FILE is required", name, name)
	}
	return value, nil
}
