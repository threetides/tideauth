package auth

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/httpx"
)

func RegisterHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(db)
		httpx.WriteJSON(w, http.StatusCreated, "account created", nil)
	}
}
