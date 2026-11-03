package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	SessionCookieName   = "siato_session"
	LoginCSRFCookieName = "siato_login_csrf"
	maxFailedAttempts   = 5
)

var (
	ErrInvalidCredentials    = errors.New("email atau kata sandi tidak sesuai")
	ErrLockedOut             = errors.New("akun dikunci sementara karena terlalu banyak percobaan login")
	ErrPasswordResetRequired = errors.New("user wajib mengatur ulang password")
	ErrInvalidResetToken     = errors.New("token reset password tidak valid atau kedaluwarsa")
)

type Principal struct {
	UserID      string
	UserName    string
	Email       string
	CSRFToken   string
	Roles       []string
	Permissions map[string]struct{}
}

func (p *Principal) HasRole(role string) bool {
	for _, item := range p.Roles {
		if strings.EqualFold(item, role) {
			return true
		}
	}
	return false
}

func (p *Principal) HasPermission(permission string) bool {
	_, ok := p.Permissions[permission]
	return ok
}

type Service struct {
	db             *sql.DB
	secureCookie   bool
	passwordPepper string
	now            func() time.Time
}

func NewService(db *sql.DB, secureCookie bool, passwordPepper string) *Service {
	return &Service{db: db, secureCookie: secureCookie, passwordPepper: passwordPepper, now: func() time.Time { return time.Now().UTC() }}
}

type loginUser struct {
	ID                      string
	UserName                sql.NullString
	Email                   sql.NullString
	GoPasswordHash          sql.NullString
	GoPasswordResetRequired bool
	SecurityStamp           sql.NullString
	LockoutEnd              sql.NullTime
	LockoutEnabled          bool
	AccessFailedCount       int
}

func (s *Service) Login(ctx context.Context, identifier, password string, remember bool, ipAddress, userAgent string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", fmt.Errorf("memulai transaksi login: %w", err)
	}
	defer tx.Rollback()

	var user loginUser
	normalized := strings.ToUpper(identifier)
	err = tx.QueryRowContext(ctx, `
SELECT Id, UserName, Email, GoPasswordHash, GoPasswordResetRequired, SecurityStamp, LockoutEnd, LockoutEnabled, AccessFailedCount
FROM users
WHERE NormalizedEmail = ? OR NormalizedUserName = ?
LIMIT 1 FOR UPDATE`, normalized, normalized).Scan(
		&user.ID, &user.UserName, &user.Email, &user.GoPasswordHash, &user.GoPasswordResetRequired, &user.SecurityStamp,
		&user.LockoutEnd, &user.LockoutEnabled, &user.AccessFailedCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		// Keep the missing-user path expensive enough to reduce account enumeration by timing.
		dummyPasswordWork(password, s.passwordPepper)
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("mencari user: %w", err)
	}

	now := s.now()
	if user.LockoutEnabled && user.LockoutEnd.Valid && user.LockoutEnd.Time.After(now) {
		return "", ErrLockedOut
	}
	if user.GoPasswordResetRequired || !user.GoPasswordHash.Valid || user.GoPasswordHash.String == "" {
		dummyPasswordWork(password, s.passwordPepper)
		return "", ErrPasswordResetRequired
	}

	valid := false
	if user.GoPasswordHash.Valid {
		valid, err = VerifyPassword(user.GoPasswordHash.String, password, s.passwordPepper)
		if err != nil && !errors.Is(err, errInvalidPasswordHash) {
			return "", fmt.Errorf("verifikasi password: %w", err)
		}
	}
	if !valid {
		failedCount := user.AccessFailedCount + 1
		var lockoutEnd any
		if user.LockoutEnabled && failedCount >= maxFailedAttempts {
			lockoutEnd = now.Add(5 * time.Minute)
			failedCount = 0
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET AccessFailedCount = ?, LockoutEnd = ? WHERE Id = ?`, failedCount, lockoutEnd, user.ID); err != nil {
			return "", fmt.Errorf("mencatat login gagal: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("menyimpan login gagal: %w", err)
		}
		if lockoutEnd != nil {
			return "", ErrLockedOut
		}
		return "", ErrInvalidCredentials
	}

	if _, err := tx.ExecContext(ctx, `UPDATE users SET AccessFailedCount = 0, LockoutEnd = NULL WHERE Id = ?`, user.ID); err != nil {
		return "", fmt.Errorf("mereset lockout: %w", err)
	}

	rawToken, err := randomToken(32)
	if err != nil {
		return "", err
	}
	csrfToken, err := randomToken(32)
	if err != nil {
		return "", err
	}
	expiresAt := now.Add(8 * time.Hour)
	if remember {
		expiresAt = now.Add(30 * 24 * time.Hour)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM go_auth_sessions WHERE expires_at <= ?`, now); err != nil {
		return "", fmt.Errorf("membersihkan session: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO go_auth_sessions
    (token_hash, user_id, csrf_token, security_stamp, expires_at, created_at, last_seen_at, remember_me, ip_address, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hashToken(rawToken), user.ID, csrfToken, nullStringValue(user.SecurityStamp), expiresAt, now, now,
		remember, truncateUTF8(ipAddress, 45), truncateUTF8(userAgent, 500),
	); err != nil {
		return "", fmt.Errorf("membuat session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("menyimpan session: %w", err)
	}
	return rawToken, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*Principal, error) {
	if rawToken == "" {
		return nil, ErrInvalidCredentials
	}
	var principal Principal
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx, `
SELECT s.user_id, COALESCE(u.UserName, ''), COALESCE(u.Email, ''), s.csrf_token, s.expires_at
FROM go_auth_sessions s
JOIN users u ON u.Id = s.user_id
WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP(6)
  AND (s.security_stamp <=> u.SecurityStamp)`, hashToken(rawToken)).Scan(
		&principal.UserID, &principal.UserName, &principal.Email, &principal.CSRFToken, &expiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("membaca session: %w", err)
	}
	principal.Permissions = make(map[string]struct{})
	if err := s.loadAuthorization(ctx, &principal); err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE go_auth_sessions SET last_seen_at = UTC_TIMESTAMP(6) WHERE token_hash = ?`, hashToken(rawToken))
	return &principal, nil
}

func (s *Service) loadAuthorization(ctx context.Context, principal *Principal) error {
	roleRows, err := s.db.QueryContext(ctx, `
SELECT r.Name FROM roles r
JOIN user_roles ur ON ur.RoleId = r.Id
WHERE ur.UserId = ? ORDER BY r.Name`, principal.UserID)
	if err != nil {
		return fmt.Errorf("membaca role: %w", err)
	}
	defer roleRows.Close()
	for roleRows.Next() {
		var role sql.NullString
		if err := roleRows.Scan(&role); err != nil {
			return err
		}
		if role.Valid {
			principal.Roles = append(principal.Roles, role.String)
		}
	}
	if err := roleRows.Err(); err != nil {
		return err
	}

	permissionRows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT p.name
FROM permissions p
LEFT JOIN role_permissions rp ON rp.permission_id = p.id
LEFT JOIN user_roles ur ON ur.RoleId = rp.role_id AND ur.UserId = ?
LEFT JOIN user_permissions up ON up.permission_id = p.id AND up.user_id = ?
WHERE ur.UserId IS NOT NULL OR up.user_id IS NOT NULL`, principal.UserID, principal.UserID)
	if err != nil {
		return fmt.Errorf("membaca permission: %w", err)
	}
	defer permissionRows.Close()
	for permissionRows.Next() {
		var permission string
		if err := permissionRows.Scan(&permission); err != nil {
			return err
		}
		principal.Permissions[permission] = struct{}{}
	}
	return permissionRows.Err()
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM go_auth_sessions WHERE token_hash = ?`, hashToken(rawToken))
	return err
}

func (s *Service) CreatePasswordResetToken(ctx context.Context, identifier string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(identifier))
	var userID string
	if err := s.db.QueryRowContext(ctx, `SELECT Id FROM users WHERE NormalizedEmail = ? OR NormalizedUserName = ? LIMIT 1`, normalized, normalized).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	rawToken, err := randomToken(32)
	if err != nil {
		return "", err
	}
	now := s.now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM go_password_reset_tokens WHERE user_id = ? OR expires_at <= ?`, userID, now); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO go_password_reset_tokens (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`, hashToken(rawToken), userID, now.Add(30*time.Minute), now); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *Service) PasswordResetTokenValid(ctx context.Context, rawToken string) bool {
	if rawToken == "" {
		return false
	}
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM go_password_reset_tokens WHERE token_hash = ? AND used_at IS NULL AND expires_at > UTC_TIMESTAMP(6)`, hashToken(rawToken)).Scan(&count)
	return err == nil && count == 1
}

func (s *Service) ResetPassword(ctx context.Context, rawToken, password string) error {
	if err := ValidateNewPassword(password); err != nil {
		return err
	}
	passwordHash, err := HashPassword(password, s.passwordPepper)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID string
	err = tx.QueryRowContext(ctx, `SELECT user_id FROM go_password_reset_tokens WHERE token_hash = ? AND used_at IS NULL AND expires_at > UTC_TIMESTAMP(6) FOR UPDATE`, hashToken(rawToken)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return err
	}
	securityStamp, err := randomToken(32)
	if err != nil {
		return err
	}
	now := s.now()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET GoPasswordHash = ?, GoPasswordResetRequired = 0, GoPasswordUpdatedAt = ?, SecurityStamp = ?, AccessFailedCount = 0, LockoutEnd = NULL WHERE Id = ?`, passwordHash, now, securityStamp, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE go_password_reset_tokens SET used_at = ? WHERE token_hash = ?`, now, hashToken(rawToken)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM go_auth_sessions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// SetDevelopmentPassword assigns a Go password without a reset token. It is
// intentionally used only by the explicit development seeder command.
func (s *Service) SetDevelopmentPassword(ctx context.Context, identifier, password string) error {
	if err := ValidateNewPassword(password); err != nil {
		return err
	}
	passwordHash, err := HashPassword(password, s.passwordPepper)
	if err != nil {
		return err
	}
	securityStamp, err := randomToken(32)
	if err != nil {
		return err
	}
	normalized := strings.ToUpper(strings.TrimSpace(identifier))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
UPDATE users
SET GoPasswordHash = ?, GoPasswordResetRequired = 0, GoPasswordUpdatedAt = ?,
    SecurityStamp = ?, AccessFailedCount = 0, LockoutEnd = NULL
WHERE NormalizedEmail = ? OR NormalizedUserName = ?`, passwordHash, s.now(), securityStamp, normalized, normalized)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("user %q tidak ditemukan atau tidak unik", identifier)
	}
	if _, err := tx.ExecContext(ctx, `DELETE s FROM go_auth_sessions s JOIN users u ON u.Id = s.user_id WHERE u.NormalizedEmail = ? OR u.NormalizedUserName = ?`, normalized, normalized); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE t FROM go_password_reset_tokens t JOIN users u ON u.Id = t.user_id WHERE u.NormalizedEmail = ? OR u.NormalizedUserName = ?`, normalized, normalized); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, rawToken string, remember bool) {
	cookie := &http.Cookie{Name: SessionCookieName, Value: rawToken, Path: "/", HttpOnly: true, Secure: s.secureCookie, SameSite: http.SameSiteLaxMode}
	if remember {
		cookie.MaxAge = int((30 * 24 * time.Hour).Seconds())
		cookie.Expires = s.now().Add(30 * 24 * time.Hour)
	}
	http.SetCookie(w, cookie)
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}

func (s *Service) LoginCSRF(w http.ResponseWriter, existing string) (string, error) {
	if len(existing) >= 32 {
		return existing, nil
	}
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{Name: LoginCSRFCookieName, Value: token, Path: "/panel-admin-stieww", HttpOnly: true, Secure: s.secureCookie, SameSite: http.SameSiteStrictMode, MaxAge: 900})
	return token, nil
}

func (s *Service) VerifyLoginCSRF(cookieToken, formToken string) bool {
	return len(cookieToken) >= 32 && len(cookieToken) == len(formToken) && subtle.ConstantTimeCompare([]byte(cookieToken), []byte(formToken)) == 1
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("membuat token acak: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func nullStringValue(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

func truncateUTF8(value string, max int) string {
	if len(value) <= max {
		return value
	}
	value = value[:max]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
