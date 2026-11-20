package security_test

import (
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"strings"
	"testing"
	"time"
)

func TestAccessTokensRejectTamperingAudienceAndExpiry(t *testing.T) {
	now := time.Unix(1800000000, 0)
	tokens := security.Tokens{Key: []byte(strings.Repeat("k", 32)), Issuer: "vms", Audience: "staff"}
	proof := security.SessionClaims{SessionID: "82d2bb6d-fdcb-4ff5-a914-91e5e9ce9559", EmployeeID: 1, CredentialGeneration: 2, RoleRevision: 3}
	raw, err := tokens.Sign(proof, now, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	got, err := tokens.Verify(raw, now)
	if err != nil || got != proof {
		t.Fatal("valid token rejected")
	}
	for _, bad := range []string{raw + "x", "", strings.Replace(raw, ".", "x", 1)} {
		if _, err := tokens.Verify(bad, now); err == nil {
			t.Fatal("invalid token accepted")
		}
	}
	if _, err := tokens.Verify(raw, now.Add(5*time.Minute)); err == nil {
		t.Fatal("expiry boundary accepted")
	}
	wrong := tokens
	wrong.Audience = "visitor"
	if _, err := wrong.Verify(raw, now); err == nil {
		t.Fatal("wrong audience accepted")
	}
	long, _ := tokens.Sign(proof, now, now.Add(6*time.Minute))
	if _, err := tokens.Verify(long, now); err == nil {
		t.Fatal("excess lifetime accepted")
	}
}
func TestCSRFIsBoundMatchedAndExpiring(t *testing.T) {
	now := time.Unix(1800000000, 0)
	csrf := security.CSRF{Key: []byte(strings.Repeat("c", 32))}
	raw, err := csrf.Sign("session:first", now, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if csrf.Verify(raw, raw, "session:first", now) != nil {
		t.Fatal("valid csrf rejected")
	}
	for _, tc := range []struct {
		cookie, header, binding string
		at                      time.Time
	}{{raw, raw, "session:second", now}, {raw, "different", "session:first", now}, {raw, raw, "session:first", now.Add(time.Minute)}, {raw, raw, "session:first", now.Add(-time.Second)}} {
		if csrf.Verify(tc.cookie, tc.header, tc.binding, tc.at) == nil {
			t.Fatal("csrf boundary bypass")
		}
	}
}
func TestRecoveryEncryptionAuthenticatesPurposeAndContent(t *testing.T) {
	key := []byte(strings.Repeat("r", 32))
	data, err := security.Seal(key, "opaque-successor", "session:request")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := security.Open(key, data, "session:request")
	if err != nil || plain != "opaque-successor" {
		t.Fatal("recovery failed")
	}
	if _, err := security.Open(key, data, "another-session:request"); err == nil {
		t.Fatal("cross-session recovery accepted")
	}
	data[len(data)-1] ^= 1
	if _, err := security.Open(key, data, "session:request"); err == nil {
		t.Fatal("tampered recovery accepted")
	}
}
