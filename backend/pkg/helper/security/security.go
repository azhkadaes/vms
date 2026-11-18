package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
)

var ErrInvalid = errors.New("invalid credential")
var encoding = base64.RawURLEncoding

func Random() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return encoding.EncodeToString(b[:]), nil
}
func Hash(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }
func Digest(key []byte, purpose, value string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(purpose + "\x00" + value))
	return hex.EncodeToString(h.Sum(nil))
}
func Equal(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }

type Tokens struct {
	Key              []byte
	Issuer, Audience string
}
type claims struct {
	Subject    int32  `json:"sub"`
	Session    string `json:"sid"`
	Generation int64  `json:"gen"`
	Revision   int64  `json:"rev"`
	Issuer     string `json:"iss"`
	Audience   string `json:"aud"`
	Issued     int64  `json:"iat"`
	Expires    int64  `json:"exp"`
}

// Sign issues only HS256 access tokens with an explicit, fixed token header.
func (t Tokens) Sign(proof entity.SessionProof, now, expiry time.Time) (string, error) {
	if len(t.Key) < 32 || !expiry.After(now) {
		return "", ErrInvalid
	}
	data, err := json.Marshal(claims{proof.EmployeeID, proof.SessionID, proof.CredentialGeneration, proof.RoleRevision, t.Issuer, t.Audience, now.Unix(), expiry.Unix()})
	if err != nil {
		return "", err
	}
	head := encoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := head + "." + encoding.EncodeToString(data)
	mac := hmac.New(sha256.New, t.Key)
	mac.Write([]byte(body))
	return body + "." + encoding.EncodeToString(mac.Sum(nil)), nil
}

func (t Tokens) Verify(raw string, now time.Time) (entity.SessionProof, error) {
	parts := strings.Split(raw, ".")
	if len(t.Key) < 32 || len(raw) > 4096 || len(parts) != 3 {
		return entity.SessionProof{}, ErrInvalid
	}
	if parts[0] != encoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) {
		return entity.SessionProof{}, ErrInvalid
	}
	mac := hmac.New(sha256.New, t.Key)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := encoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return entity.SessionProof{}, ErrInvalid
	}
	data, err := encoding.DecodeString(parts[1])
	if err != nil {
		return entity.SessionProof{}, ErrInvalid
	}
	var c claims
	if json.Unmarshal(data, &c) != nil || c.Issuer != t.Issuer || c.Audience != t.Audience || c.Expires <= now.Unix() || c.Issued > now.Unix() || c.Expires <= c.Issued || c.Expires-c.Issued > 300 || c.Subject <= 0 || c.Generation <= 0 || c.Revision <= 0 {
		return entity.SessionProof{}, ErrInvalid
	}
	return entity.SessionProof{SessionID: c.Session, EmployeeID: c.Subject, CredentialGeneration: c.Generation, RoleRevision: c.Revision}, nil
}

type CSRF struct{ Key []byte }
type csrfPayload struct {
	Nonce   string `json:"n"`
	Issued  int64  `json:"i"`
	Expires int64  `json:"e"`
}

func (s CSRF) Sign(binding string, now, expiry time.Time) (string, error) {
	if len(s.Key) < 32 || binding == "" || !expiry.After(now) {
		return "", ErrInvalid
	}
	nonce, err := Random()
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(csrfPayload{nonce, now.Unix(), expiry.Unix()})
	body := encoding.EncodeToString(data)
	return body + "." + Digest(s.Key, "csrf", binding+"\x00"+body), nil
}
func (s CSRF) Verify(cookie, header, binding string, now time.Time) error {
	if len(s.Key) < 32 || binding == "" || len(cookie) > 1024 || !Equal(cookie, header) {
		return ErrInvalid
	}
	parts := strings.Split(cookie, ".")
	if len(parts) != 2 || !Equal(parts[1], Digest(s.Key, "csrf", binding+"\x00"+parts[0])) {
		return ErrInvalid
	}
	data, err := encoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalid
	}
	var p csrfPayload
	if json.Unmarshal(data, &p) != nil || p.Nonce == "" || p.Issued > now.Unix() || p.Expires <= now.Unix() || p.Expires <= p.Issued {
		return ErrInvalid
	}
	return nil
}
func Seal(key []byte, plaintext, aad string) ([]byte, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	return g.Seal(nonce, nonce, []byte(plaintext), []byte(aad)), nil
}
func Open(key, data []byte, aad string) (string, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(b)
	if err != nil || len(data) < g.NonceSize() {
		return "", ErrInvalid
	}
	plain, err := g.Open(nil, data[:g.NonceSize()], data[g.NonceSize():], []byte(aad))
	return string(plain), err
}
