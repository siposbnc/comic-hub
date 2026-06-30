package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Access tokens are stateless HS256 JWTs verified on every request; refresh tokens are
// opaque random strings stored (hashed) server-side so they can be revoked and rotated.
// HS256 is implemented directly on crypto/hmac to avoid a third-party JWT dependency on
// this security-sensitive surface — the format is the standard compact JWS.

// ErrInvalidToken is returned for any malformed, mis-signed, or expired access token.
var ErrInvalidToken = errors.New("auth: invalid token")

// Claims is the access-token payload. Kept minimal: identity + role + validity window.
type Claims struct {
	Subject string `json:"sub"`  // user id
	Role    string `json:"role"` // role at issue time
	Issued  int64  `json:"iat"`  // unix seconds
	Expires int64  `json:"exp"`  // unix seconds
}

var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// signAccessToken issues an HS256 JWT for the claims, signed with secret.
func signAccessToken(secret []byte, c Claims) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	signingInput := jwtHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	sig := hmacSHA256(secret, signingInput)
	return signingInput + "." + sig, nil
}

// parseAccessToken verifies the signature and expiry of an access token, returning its claims.
func parseAccessToken(secret []byte, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != jwtHeader {
		return Claims{}, ErrInvalidToken
	}
	expected := hmacSHA256(secret, parts[0]+"."+parts[1])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return Claims{}, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().Unix() >= c.Expires {
		return Claims{}, ErrInvalidToken
	}
	return c, nil
}

func hmacSHA256(secret []byte, msg string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// newRefreshToken returns a fresh opaque refresh token (the secret handed to the client) and
// its sha256 hash (what we store, so a DB leak doesn't expose usable tokens).
func newRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth: read refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashRefreshToken(token), nil
}

// hashRefreshToken is the stable one-way mapping from a refresh token to its stored form.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
