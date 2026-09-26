package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/apperr"
)

func RegisterService(db *pgxpool.Pool, ctx context.Context, register Register) (u *User, err error) {
	var user *User

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
			SELECT id, $4, (SELECT NOW() + INTERVAL '30 days')
			FROM new_user
		)
		SELECT id, email, name, email_verified, created_at, updated_at
		FROM new_user;
	`

	err = db.QueryRow(ctx, query, register.Name, register.Email, "password_hash", "token_hash").Scan(&user)
	if err != nil {
		return nil, apperr.InternalServerError("error registering new user", err)
	}

	return user, nil
}
