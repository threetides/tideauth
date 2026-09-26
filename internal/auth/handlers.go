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

type Register struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

var verifier = emailverifier.NewVerifier()

func RegisterHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			if len(email) > 100 {
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

		if len(fieldErrors) > 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "bad request", fieldErrors)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "account created", nil)
	}
}
