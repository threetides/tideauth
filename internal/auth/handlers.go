package auth

import (
	"net/http"

	"github.com/threetides/tideauth/internal/httpx"
)

func GoogleSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Signed in with Google", nil)
	}
}

func SignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Signed out", nil)
	}
}
