package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/threetides/tideauth/internal/httpx"
)

func (cfg *GoogleCFG) GoogleSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate random state string
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			log.Println("Error creating random state:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
		state := base64.RawURLEncoding.EncodeToString(b)

		// Set short lived httpOnlyCookie
		cookie := http.Cookie{
			Name:     "state",
			Value:    state,
			Path:     "/",
			Expires:  time.Now().Add(5 * time.Minute),
			HttpOnly: true,
			Secure:   os.Getenv("COOKIE_SECURE") == "true",
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, &cookie)

		// Redirect to the login flow with state in url
		http.Redirect(w, r, cfg.Config.AuthCodeURL(state), http.StatusFound)
	}
}

func (cfg *GoogleCFG) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify state and errors.
		state := r.URL.Query().Get("state")
		cookie, err := r.Cookie("state")
		if err != nil {
			log.Println("Error retrieving cookie:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		if state != cookie.Value {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		// Clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "state",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		})

		// Get OAuth2 token
		oauth2Token, err := cfg.Config.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			log.Println("Error creating oauth2Token:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		// Extract the ID Token from OAuth2 token.
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			log.Println("Error extracting rawIDToken:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		// Parse and verify ID Token payload.
		idToken, err := cfg.IDTokenVerifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			log.Println("Error parsing idToken payload:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		// Extract custom claims
		var claims claims
		if err := idToken.Claims(&claims); err != nil {
			log.Println("Error extracting custom claims:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		// Insert into db
		sessionCookie, err := InsertUserData(r.Context(), cfg.DB, claims)
		if err != nil {
			log.Println("Error inserting user into db:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		http.SetCookie(w, sessionCookie)

		http.Redirect(w, r, os.Getenv("CORS_ORIGIN")+"/", http.StatusTemporaryRedirect)
	}
}

func (Auth *AuthHandler) GetSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		hash := sha256.Sum256([]byte(cookie.Value))
		tokenHash := hex.EncodeToString(hash[:])

		userIDQuery :=
			`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`

		var userID string

		err = Auth.DB.QueryRow(r.Context(), userIDQuery, tokenHash).Scan(&userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Println("Error quering database:", err)
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		if userID == "" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		var user User
		userQuery :=
			`SELECT id, email, name, email_verified, created_at FROM users WHERE id = $1`

		err = Auth.DB.QueryRow(r.Context(), userQuery, userID).Scan(&user.ID, &user.Email, &user.Name, &user.EmailVerified, &user.CreatedAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Println("Error quering database:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, "Authorized", user)
	}
}

func (Auth *AuthHandler) SignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		hash := sha256.Sum256([]byte(cookie.Value))
		tokenHash := hex.EncodeToString(hash[:])

		query :=
			`DELETE from sessions WHERE token_hash = $1`

		_, err = Auth.DB.Exec(r.Context(), query, tokenHash)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error deleting session:", err)
			return
		}

		// Clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		})

		httpx.WriteJSON(w, http.StatusOK, "Logged out", nil)
	}
}
