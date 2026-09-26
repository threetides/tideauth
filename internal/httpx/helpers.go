package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/threetides/tideauth/internal/apperr"
)

func WriteJSON(w http.ResponseWriter, statusCode int, message string, data any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(Response{Message: message, Data: data}); err != nil {
		log.Println("error encoding json:", err)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := errors.AsType[*apperr.Apperr](err)
	if !ok || appErr.Kind == apperr.KindInternalServerError {
		log.Printf("%s: %v", r.URL.Path, err)
		WriteJSON(w, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	WriteJSON(w, appErr.Kind.HTTPStatus(), appErr.Message, appErr.Data)
}
