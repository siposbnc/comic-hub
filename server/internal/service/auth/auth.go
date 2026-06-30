// Package auth owns server-mode authentication: password login, JWT access tokens, and
// revocable refresh-token sessions. Embedded mode bypasses this entirely (a single implicit
// owner, loopback token) — auth only matters when the server runs multi-user.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/password"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

const (
	// DefaultAccessTTL is short: access tokens are stateless and can't be revoked, so they
	// expire quickly and are refreshed.
	DefaultAccessTTL = 15 * time.Minute
	// DefaultRefreshTTL is long: refresh tokens are stored server-side and revocable.
	DefaultRefreshTTL = 30 * 24 * time.Hour
	// MinPasswordLen is the minimum acceptable password length.
	MinPasswordLen = 8
)

// Tokens is the pair returned by login/refresh: a short-lived access token and the opaque
// refresh token (handed to the client; only its hash is stored).
type Tokens struct {
	Access       string
	Refresh      string
	AccessExpiry int64 // unix seconds
}

// Service authenticates users and manages sessions.
type Service struct {
	repo       domain.Repository
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// New builds an auth service. secret signs access tokens; it must be stable across restarts
// (else all access tokens are invalidated) and secret.
func New(repo domain.Repository, secret []byte) *Service {
	return &Service{repo: repo, secret: secret, accessTTL: DefaultAccessTTL, refreshTTL: DefaultRefreshTTL}
}

// Login verifies credentials and issues a token pair. Bad username or password both return
// ErrUnauthorized (no oracle for which was wrong).
func (s *Service) Login(ctx context.Context, username, plaintext string) (Tokens, domain.User, error) {
	u, err := s.repo.Users().GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Tokens{}, domain.User{}, domain.ErrUnauthorized
		}
		return Tokens{}, domain.User{}, err
	}
	if u.PasswordHash == "" {
		// Passwordless accounts (the implicit owner) can't log in over the network.
		return Tokens{}, domain.User{}, domain.ErrUnauthorized
	}
	if err := password.Verify(plaintext, u.PasswordHash); err != nil {
		return Tokens{}, domain.User{}, domain.ErrUnauthorized
	}
	toks, err := s.issue(ctx, u)
	if err != nil {
		return Tokens{}, domain.User{}, err
	}
	return toks, u, nil
}

// Refresh rotates a refresh token: it validates the presented token, deletes its session,
// and issues a fresh pair. A reused/revoked/expired token returns ErrUnauthorized.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, domain.User, error) {
	sess, err := s.repo.Sessions().GetByHash(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Tokens{}, domain.User{}, domain.ErrUnauthorized
		}
		return Tokens{}, domain.User{}, err
	}
	// Rotate regardless of expiry — a presented token is now spent.
	_ = s.repo.Sessions().Delete(ctx, sess.ID)
	if time.Now().Unix() >= sess.ExpiresAt {
		return Tokens{}, domain.User{}, domain.ErrUnauthorized
	}
	u, err := s.repo.Users().Get(ctx, sess.UserID)
	if err != nil {
		return Tokens{}, domain.User{}, domain.ErrUnauthorized
	}
	toks, err := s.issue(ctx, u)
	if err != nil {
		return Tokens{}, domain.User{}, err
	}
	return toks, u, nil
}

// Logout revokes the refresh token's session (idempotent — an unknown token is a no-op).
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	sess, err := s.repo.Sessions().GetByHash(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.repo.Sessions().Delete(ctx, sess.ID)
}

// Authenticate validates an access token and returns the current user (loaded fresh so a
// role change or deletion takes effect immediately, not only after the token expires).
func (s *Service) Authenticate(ctx context.Context, accessToken string) (domain.User, error) {
	claims, err := parseAccessToken(s.secret, accessToken)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	u, err := s.repo.Users().Get(ctx, claims.Subject)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	return u, nil
}

// EnsureAdmin bootstraps a login-capable admin for server mode: if the username exists, it
// (re)sets the password; otherwise it creates an admin. A no-op when username/password are
// empty. Lets a packaged/Docker deployment seed the first account from env on boot.
func (s *Service) EnsureAdmin(ctx context.Context, username, displayName, plaintext string) error {
	username = strings.TrimSpace(username)
	if username == "" || plaintext == "" {
		return nil
	}
	if len(plaintext) < MinPasswordLen {
		return fmt.Errorf("%w: admin password must be at least %d characters", domain.ErrValidation, MinPasswordLen)
	}
	hash, err := password.Hash(plaintext)
	if err != nil {
		return err
	}
	existing, err := s.repo.Users().GetByUsername(ctx, username)
	switch {
	case err == nil:
		return s.repo.Users().SetPasswordHash(ctx, existing.ID, hash)
	case errors.Is(err, domain.ErrNotFound):
		if displayName == "" {
			displayName = username
		}
		_, cerr := s.repo.Users().Create(ctx, domain.User{
			ID:           ulid.New(),
			Username:     username,
			DisplayName:  displayName,
			Role:         domain.RoleAdmin,
			PasswordHash: hash,
			CreatedAt:    time.Now().UnixMilli(),
		})
		return cerr
	default:
		return err
	}
}

// issue mints a token pair for a user and stores the refresh session.
func (s *Service) issue(ctx context.Context, u domain.User) (Tokens, error) {
	now := time.Now()
	accessExp := now.Add(s.accessTTL)
	access, err := signAccessToken(s.secret, Claims{
		Subject: u.ID,
		Role:    string(u.Role),
		Issued:  now.Unix(),
		Expires: accessExp.Unix(),
	})
	if err != nil {
		return Tokens{}, err
	}
	refresh, refreshHash, err := newRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	if err := s.repo.Sessions().Create(ctx, domain.Session{
		ID:          ulid.New(),
		UserID:      u.ID,
		RefreshHash: refreshHash,
		ExpiresAt:   now.Add(s.refreshTTL).Unix(),
		CreatedAt:   now.UnixMilli(),
	}); err != nil {
		return Tokens{}, err
	}
	return Tokens{Access: access, Refresh: refresh, AccessExpiry: accessExp.Unix()}, nil
}
