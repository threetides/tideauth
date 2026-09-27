package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/apperr"
	"golang.org/x/crypto/bcrypt"
)

func RegisterService(db *pgxpool.Pool, ctx context.Context, register Register) (User, Session, error) {
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

	// * Create CTE query
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

	// * Perform query and check for PgError
	err = db.QueryRow(ctx, query, register.Name, register.Email, passwordHash, tokenHash, expiresAt).Scan(&user.ID, &user.Name, &user.Email, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			if pgErr.Code == "23505" {
				return user, session, apperr.Conflict("email is already registered", nil)
			}
			return user, session, apperr.InternalServerError("unknown PgError;", pgErr)
		}
		return user, session, apperr.InternalServerError("error registering new user", err)
	}

	// * Create session cookie
	session = Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	return user, session, nil
}

func LoginService(db *pgxpool.Pool, ctx context.Context, login Login) (User, Session, error) {
	var user User
	var session Session
	expiresAt := time.Now().AddDate(0, 0, 30)

	// * Generate random token and token hash
	tokenBytesbytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytesbytes); err != nil {
		return user, session, apperr.InternalServerError("error generating random token", err)
	}
	token := hex.EncodeToString(tokenBytesbytes)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// * Create CTE query
	selectPasswordHashQuery := `
		SELECT user_id, password_hash
		FROM passwords
		INNER JOIN users 
		ON passwords.user_id = users.id
		WHERE users.email = $1
	`

	var userID string
	var passwordHash string

	// * Perform query and check for PgError
	err := db.QueryRow(ctx, selectPasswordHashQuery, login.Email).Scan(&userID, &passwordHash)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			return user, session, apperr.InternalServerError("unknown PgError;", pgErr)
		}
		return user, session, apperr.InternalServerError("error getting password_hash", err)
	}

	log.Println("password hash:", passwordHash)

	insertSessionQuery := `
			INSERT INTO sessions (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)
	`

	// * Perform query and check for PgError
	_, err = db.Exec(ctx, insertSessionQuery, userID, tokenHash, expiresAt)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			return user, session, apperr.InternalServerError("unknown PgError;", pgErr)
		}
		return user, session, apperr.InternalServerError("error getting password_hash", err)
	}

	// * Create session cookie
	session = Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	return user, session, nil
}
