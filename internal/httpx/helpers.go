package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, statusCode int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(Response{
		Message: message,
		Data:    data,
	}); err != nil {
		log.Println("Error encoding json:", err)
	}
}
