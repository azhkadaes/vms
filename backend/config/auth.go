package config

import (
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Auth struct {
	JWTKey                      []byte
	Issuer, Audience            string
	CSRFKey, RecoveryKey        []byte
	Origin                      string
	GatewayKey                  string
	GatewayNetworks             []*net.IPNet
	BcryptCost, HashConcurrency int
}

func splitNetworks(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
func authFrom(get func(string) string, environment string) (*Auth, error) {
	if get("JWT_SECRET_KEY") == "" && get("JWT_SECRET_KEY_FILE") == "" && get("CSRF_KEY") == "" && get("CSRF_KEY_FILE") == "" && get("STAFF_ORIGIN") == "" {
		return nil, nil
	}
	jwt, err := secret(get, "JWT_SECRET_KEY")
	if err != nil || len(jwt) < 32 {
		return nil, errors.New("JWT_SECRET_KEY requires at least 32 random bytes")
	}
	csrf, err := secret(get, "CSRF_KEY")
	if err != nil {
		return nil, err
	}
	csrfKey, err := base64.StdEncoding.DecodeString(csrf)
	if err != nil || len(csrfKey) != 32 {
		return nil, errors.New("CSRF_KEY must encode 32 random bytes as base64")
	}
	recovery, err := secret(get, "REFRESH_RECOVERY_KEY")
	if err != nil {
		return nil, err
	}
	recoveryKey, err := base64.StdEncoding.DecodeString(recovery)
	if err != nil || len(recoveryKey) != 32 {
		return nil, errors.New("REFRESH_RECOVERY_KEY must encode 32 random bytes as base64")
	}
	gateway, err := secret(get, "GATEWAY_INIT_KEY")
	if err != nil || len(gateway) < 32 {
		return nil, errors.New("GATEWAY_INIT_KEY requires at least 32 random bytes")
	}
	if jwt == gateway || jwt == csrf || jwt == recovery || gateway == csrf || gateway == recovery || csrf == recovery {
		return nil, errors.New("authentication keys must be independent")
	}
	origin := get("STAFF_ORIGIN")
	u, err := url.Parse(origin)
	if err != nil || u == nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || (u.Scheme != "https" && !(environment != "production" && u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback()))) {
		return nil, errors.New("STAFF_ORIGIN must be an exact HTTPS origin (loopback HTTP is development-only)")
	}
	a := &Auth{JWTKey: []byte(jwt), CSRFKey: csrfKey, RecoveryKey: recoveryKey, Origin: origin, GatewayKey: gateway, Issuer: get("JWT_TOKEN_ISSUER"), Audience: get("JWT_TOKEN_AUDIENCE"), BcryptCost: 12, HashConcurrency: 2}
	if a.Issuer == "" {
		a.Issuer = "vms"
	}
	if a.Audience == "" {
		a.Audience = "vms-staff"
	}
	if raw := get("PASSWORD_BCRYPT_COST"); raw != "" {
		a.BcryptCost, err = strconv.Atoi(raw)
		if err != nil || a.BcryptCost < 10 || a.BcryptCost > 16 {
			return nil, errors.New("PASSWORD_BCRYPT_COST must be 10-16")
		}
	}
	if raw := get("PASSWORD_HASH_CONCURRENCY"); raw != "" {
		a.HashConcurrency, err = strconv.Atoi(raw)
		if err != nil || a.HashConcurrency < 1 || a.HashConcurrency > 64 {
			return nil, errors.New("PASSWORD_HASH_CONCURRENCY must be 1-64")
		}
	}
	for _, cidr := range splitNetworks(get("GATEWAY_INIT_NETWORKS")) {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, errors.New("invalid GATEWAY_INIT_NETWORKS")
		}
		a.GatewayNetworks = append(a.GatewayNetworks, network)
	}
	if len(a.GatewayNetworks) == 0 {
		return nil, errors.New("GATEWAY_INIT_NETWORKS must identify the private gateway peers")
	}
	return a, nil
}
