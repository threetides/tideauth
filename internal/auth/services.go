package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func InsertUserData(ctx context.Context, db *pgxpool.Pool, claims claims) (cookie *http.Cookie, error error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("error reading byte: %v", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	expiresAt := time.Now().AddDate(0, 0, 30)

	// Begin a transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Println("error rolling back transaction:", err)
		}
	}()

	// Insert into users table
	usersQuery :=
		`INSERT INTO users (name, email, email_verified) VALUES ($1, $2, $3)
			ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
			RETURNING id`

	var userID string

	err = tx.QueryRow(ctx, usersQuery, claims.Name, claims.Email, claims.EmailVerified).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("error inserting claims into users: %v", err)
	}

	// Insert into oauth_accounts table
	oauthAccountsQuery :=
		`INSERT INTO oauth_accounts (user_id, provider, provider_user_id) VALUES ($1, $2, $3)
			ON CONFLICT (provider, provider_user_id) DO UPDATE SET provider_user_id = EXCLUDED.provider_user_id`

	_, err = tx.Exec(ctx, oauthAccountsQuery, userID, "google", claims.Sub)
	if err != nil {
		return nil, fmt.Errorf("error inserting claims into oauth_accounts: %v", err)
	}

	// Insert into sessions table
	sessionsQuery :=
		`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`

	_, err = tx.Exec(ctx, sessionsQuery, userID, tokenHash, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("error inserting claims into sessions: %v", err)
	}

	// Commit if nothing wrong
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("error commiting transaction: %v", err)
	}

	cookie = &http.Cookie{
		Name:     "session",
		Value:    token, // raw token to browser
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  expiresAt,
	}

	return cookie, nil
}
