package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/jackc/pgx/v5"
	"github.com/threetides/tideauth/internal/httpx"
)

var verifier = emailverifier.NewVerifier()

var hasLower = regexp.MustCompile(`\p{Ll}`)
var hasUpper = regexp.MustCompile(`\p{Lu}`)
var hasDigit = regexp.MustCompile(`[0-9]`)

var returnErrors = rErrors{
	InternalServerError:    "internal_server_error",
	Unauthorized:           "unauthorized",
	GoogleEmailNotVerified: "google_email_not_verified",
	LocalEmailNotVerified:  "local_email_not_verified",
}

func (Auth *AuthHandler) SignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var signUpData signUp
		err := json.NewDecoder(r.Body).Decode(&signUpData)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "Invalid json", nil)
			return
		}

		var fieldErrors []fieldError

		email := strings.TrimSpace(signUpData.Email)
		name := strings.TrimSpace(signUpData.Name)
		password := strings.TrimSpace(signUpData.Password)

		if email == "" {
			fieldErrors = append(fieldErrors, fieldError{Field: "email", Error: "Email is required"})
		} else {
			verified, err := verifier.Verify(email)
			if err != nil || !verified.Syntax.Valid {
				fieldErrors = append(fieldErrors, fieldError{Field: "email", Error: "Email is invalid"})
			}
		}

		if name == "" {
			fieldErrors = append(fieldErrors, fieldError{Field: "name", Error: "Name is required"})
		} else if len(name) > 255 {
			fieldErrors = append(fieldErrors, fieldError{Field: "name", Error: "Name can't contain more than 255 characters"})
		}

		if password == "" {
			fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password is required"})
		} else {
			if len(password) < 8 {
				fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password must contain at least 8 characters"})
			} else {
				if len(password) > 128 {
					fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password can't contain more than 128 characters"})
				}
				if !hasLower.MatchString(password) {
					fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password must contain at least one lowercase letter"})
				}
				if !hasUpper.MatchString(password) {
					fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password must contain at least one uppercase letter"})
				}
				if !hasDigit.MatchString(password) {
					fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password must contain at least one digit"})
				}
			}
		}

		if len(fieldErrors) > 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "Bad request", fieldErrors)
			return
		}

		expiresAt := time.Now().AddDate(0, 0, 30)
		token, err := InsertUserData(r.Context(), Auth.DB, signUp{
			Email:    email,
			Name:     name,
			Password: password,
		}, expiresAt)

		if err != nil {
			if errors.Is(err, ErrEmailExists) {
				httpx.WriteJSON(w, http.StatusConflict, "An account with this email is already registered", nil)
				return
			}

			log.Println("Error inserting user into db:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Domain:   Auth.CookieDomain,
			Value:    token, // raw token to browser
			HttpOnly: true,
			Secure:   Auth.SecureCookie,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  expiresAt,
		})

		httpx.WriteJSON(w, http.StatusCreated, "Signed up", nil)
	}
}

func (Auth *AuthHandler) SignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var signInData signIn
		err := json.NewDecoder(r.Body).Decode(&signInData)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "Invalid json", nil)
			return
		}

		var fieldErrors []fieldError

		email := strings.TrimSpace(signInData.Email)
		password := strings.TrimSpace(signInData.Password)

		if email == "" {
			fieldErrors = append(fieldErrors, fieldError{Field: "email", Error: "Email is required"})
		}

		if password == "" {
			fieldErrors = append(fieldErrors, fieldError{Field: "password", Error: "Password is required"})
		}

		if len(fieldErrors) > 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "Bad request", fieldErrors)
			return
		}

		expiresAt := time.Now().AddDate(0, 0, 30)
		token, err := SignIn(r.Context(), Auth.DB, signIn{
			Email:    email,
			Password: password,
		}, expiresAt)

		if err != nil {
			if errors.Is(err, ErrInvalidEmailOrPassword) {
				httpx.WriteJSON(w, http.StatusConflict, "Invalid email or password", nil)
				return
			}

			log.Println("Error signing in:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Domain:   Auth.CookieDomain,
			Value:    token, // raw token to browser
			HttpOnly: true,
			Secure:   Auth.SecureCookie,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  expiresAt,
		})

		httpx.WriteJSON(w, http.StatusOK, "Signed in", nil)
	}
}

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

		token := base64.RawURLEncoding.EncodeToString(b)
		returnError := base64.StdEncoding.EncodeToString([]byte(r.URL.Query().Get("return_error")))
		returnSuccess := base64.StdEncoding.EncodeToString([]byte(r.URL.Query().Get("return_success")))

		state := fmt.Sprintf("%s|%s|%s", token, returnError, returnSuccess)

		// Set short lived httpOnlyCookie
		cookie := http.Cookie{
			Name:     "state",
			Domain:   cfg.CookieDomain,
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(5 * time.Minute),
			HttpOnly: true,
			Secure:   cfg.SecureCookie,
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

		parts := strings.SplitN(state, "|", 3)
		token := parts[0]

		rawReturnError, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			log.Println("Error decoding rawReturnError:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
		rawReturnSuccess, err := base64.StdEncoding.DecodeString(parts[2])
		if err != nil {
			log.Println("Error decoding rawReturnSuccess:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		returnError := string(rawReturnError)
		returnSuccess := string(rawReturnSuccess)

		if token != cookie.Value {
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.Unauthorized)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// Clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "state",
			Domain:   cfg.CookieDomain,
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
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.InternalServerError)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// Extract the ID Token from OAuth2 token.
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			log.Println("Error extracting rawIDToken:", err)
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.InternalServerError)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// Parse and verify ID Token payload.
		idToken, err := cfg.IDTokenVerifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			log.Println("Error parsing idToken payload:", err)
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.InternalServerError)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// Extract custom claims
		var claims claims
		if err := idToken.Claims(&claims); err != nil {
			log.Println("Error extracting claims:", err)
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.InternalServerError)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		if !claims.EmailVerified {
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.GoogleEmailNotVerified)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// Check for existing user
		var user struct {
			Email         string
			EmailVerified bool
		}
		query := `SELECT email, email_verified FROM users WHERE email = ($1)`

		err = cfg.DB.QueryRow(r.Context(), query, claims.Email).Scan(&user.Email, &user.EmailVerified)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Println("Error querying users table:", err)
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.InternalServerError)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// User exists, but email is NOT verified
		if user.Email != "" && !user.EmailVerified {
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.LocalEmailNotVerified)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		// Insert into db
		expiresAt := time.Now().AddDate(0, 0, 30)
		sessionToken, err := InsertGoogleData(r.Context(), cfg.DB, claims, expiresAt)
		if err != nil {
			log.Println("Error inserting into db:", err)
			redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnError, returnErrors.InternalServerError)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
			http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Domain:   cfg.CookieDomain,
			Value:    sessionToken, // raw token to browser
			HttpOnly: true,
			Secure:   cfg.SecureCookie,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  expiresAt,
		})

		redirect, err := generateRedirectURL(cfg.RedirectOrigin, returnSuccess, "")
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		http.Redirect(w, r, redirect.String(), http.StatusTemporaryRedirect)
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
			Domain:   Auth.CookieDomain,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		})

		httpx.WriteJSON(w, http.StatusOK, "Logged out", nil)
	}
}
