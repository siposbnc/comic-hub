package domain

import "context"

// UserRole is a member's permission level. Ordered owner > admin > member > restricted.
type UserRole string

const (
	RoleOwner      UserRole = "owner"
	RoleAdmin      UserRole = "admin"
	RoleMember     UserRole = "member"
	RoleRestricted UserRole = "restricted"
)

// Valid reports whether r is a known role.
func (r UserRole) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleRestricted:
		return true
	}
	return false
}

// Rank orders roles for permission checks (owner highest). Unknown roles rank lowest.
func (r UserRole) Rank() int {
	switch r {
	case RoleOwner:
		return 3
	case RoleAdmin:
		return 2
	case RoleMember:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether r has at least min's permission level.
func (r UserRole) AtLeast(min UserRole) bool { return r.Rank() >= min.Rank() }

// User is an account. In embedded mode a single implicit owner exists (id OwnerUserID,
// no password). In server mode users have password hashes and log in.
type User struct {
	ID           string
	Username     string
	DisplayName  string
	Role         UserRole
	PasswordHash string // argon2id; empty for the passwordless implicit owner
	AgeRatingMax string // content ceiling for restricted users; empty = unrestricted
	CreatedAt    int64
}

// Session is a stored refresh token: an opaque token's sha256 hash plus its expiry, so
// refresh tokens are revocable (logout/rotation) and a DB leak exposes no usable tokens.
type Session struct {
	ID          string
	UserID      string
	RefreshHash string
	ExpiresAt   int64
	CreatedAt   int64
}

// UserRepository persists accounts.
type UserRepository interface {
	Create(ctx context.Context, u User) (User, error)
	Get(ctx context.Context, id string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	List(ctx context.Context) ([]User, error)
	Count(ctx context.Context) (int, error)
	// Update changes profile/role/restriction fields (not the password).
	Update(ctx context.Context, u User) error
	// SetPasswordHash sets (or clears) a user's password hash.
	SetPasswordHash(ctx context.Context, id, hash string) error
	Delete(ctx context.Context, id string) error
}

// SessionRepository persists refresh-token sessions.
type SessionRepository interface {
	Create(ctx context.Context, s Session) error
	// GetByHash returns the session for a refresh-token hash (ErrNotFound if absent/revoked).
	GetByHash(ctx context.Context, refreshHash string) (Session, error)
	Delete(ctx context.Context, id string) error
	// DeleteForUser revokes all of a user's sessions (e.g. password change, delete).
	DeleteForUser(ctx context.Context, userID string) error
	// DeleteExpired prunes sessions past their expiry, returning the count removed.
	DeleteExpired(ctx context.Context, now int64) (int, error)
}
