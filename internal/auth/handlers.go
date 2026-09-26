package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/httpx"
)

type Register struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

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
		}
		if name == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Error: "name is required"})
		}
		if password == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "password", Error: "password is required"})
		}

		if len(fieldErrors) > 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "bad request", fieldErrors)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "account created", nil)
	}
}
