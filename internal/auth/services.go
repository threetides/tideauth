package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/apperr"
	"golang.org/x/crypto/bcrypt"
)

func RegisterService(db *pgxpool.Pool, ctx context.Context, register Register) (u User, s Session, err error) {
	var user User
	var session Session

	expiresAt := time.Now().AddDate(0, 0, 30)

	// * Hash password using bcrypt
	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(register.Password), bcrypt.DefaultCost)
	if err != nil {
		return user, session, apperr.InternalServerError("error hashing password", err)
	}
	passwordHash := string(passwordBytes)

	// * Generate random token and token hash
	tokenBytesbytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytesbytes); err != nil {
		return user, session, apperr.InternalServerError("error generating random token", err)
	}
	token := hex.EncodeToString(tokenBytesbytes)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	query := `
		WITH new_user AS (
			INSERT INTO users (name, email)
			VALUES ($1, $2)
			RETURNING id, name, email, email_verified, created_at, updated_at
		),
		new_password AS (
			INSERT INTO passwords (user_id, password_hash)
			SELECT id, $3
			FROM new_user
		),
		new_session AS (
			INSERT INTO sessions (user_id, token_hash, expires_at)
			SELECT id, $4, $5
			FROM new_user
		)
		SELECT id, email, name, email_verified, created_at, updated_at
		FROM new_user;
	`

	err = db.QueryRow(ctx, query, register.Name, register.Email, passwordHash, tokenHash, expiresAt).Scan(&user.ID, &user.Name, &user.Email, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return user, session, apperr.InternalServerError("error registering new user", err)
	}

	session = Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	return user, session, nil
}
