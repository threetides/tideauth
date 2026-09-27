package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/httpx"
	passwordvalidator "github.com/wagslane/go-password-validator"
)

var verifier = emailverifier.NewVerifier()

func RegisterHandler(db *pgxpool.Pool, secure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// * Decode request
		var register Register
		err := json.NewDecoder(r.Body).Decode(&register)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "invalid json", nil)
			return
		}

		email := strings.TrimSpace(register.Email)
		name := strings.TrimSpace(register.Name)
		password := register.Password

		var fieldErrors []httpx.FieldError

		// * Validate and append to fieldErrors
		if email == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "email", Error: "email is required"})
		} else {
			if len(email) > 254 {
				fieldErrors = append(fieldErrors, httpx.FieldError{Field: "email", Error: "email cannot contain more than 254 characters"})
			}

			res, err := verifier.Verify(email)
			if err != nil || !res.Syntax.Valid {
				fieldErrors = append(fieldErrors, httpx.FieldError{Field: "email", Error: "email is invalid"})
			}
		}

		if name == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Error: "name is required"})
		} else {
			if len(name) > 100 {
				fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Error: "name cannot contain more than 100 characters"})
			}
		}

		if password == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "password", Error: "password is required"})
		} else {
			const minEntropyBits = 60
			err := passwordvalidator.Validate(password, minEntropyBits)
			if err != nil {
				fieldErrors = append(fieldErrors, httpx.FieldError{Field: "password", Error: fmt.Sprintf("password is too weak, strenght: %f", passwordvalidator.GetEntropy(password))})
			}
		}

		// * Return early if len(fieldErrors) > 0
		if len(fieldErrors) > 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "bad request", fieldErrors)
			return
		}

		// * Insert the user into db
		user, session, err := RegisterService(db, r.Context(), register)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}

		// * Store token in httpOnlyCookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    session.Token,
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			Secure:   secure, // Set to true in production (HTTPS)
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		httpx.WriteJSON(w, http.StatusCreated, "signed up", user)
	}
}

func LoginHandler(db *pgxpool.Pool, secure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// * Decode request
		var login Login
		err := json.NewDecoder(r.Body).Decode(&login)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "invalid json", nil)
			return
		}

		email := strings.TrimSpace(login.Email)
		password := login.Password

		var fieldErrors []httpx.FieldError

		// * Validate and append to fieldErrors
		if email == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "email", Error: "email is required"})
		}
		if password == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "password", Error: "password is required"})
		}

		// * Return early if len(fieldErrors) > 0
		if len(fieldErrors) > 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "bad request", fieldErrors)
			return
		}

		// * Insert the user into db
		user, session, err := LoginService(db, r.Context(), login)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}

		// * Store token in httpOnlyCookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    session.Token,
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			Secure:   secure, // Set to true in production (HTTPS)
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		httpx.WriteJSON(w, http.StatusCreated, "signed in", user)
	}
}
