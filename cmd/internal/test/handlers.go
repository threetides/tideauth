package test

import (
	"net/http"

	"github.com/threetides/tideauth/internal/httpx"
)

func TestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Ok", nil)
	}
}
