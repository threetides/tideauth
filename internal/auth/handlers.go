package auth

import (
	"log"
	"net/http"

	"github.com/threetides/tideauth/internal/httpx"
)

func (cfg *GoogleCFG) GoogleSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := ""

		http.Redirect(w, r, cfg.Config.AuthCodeURL(state), http.StatusFound)
	}
}

func (cfg *GoogleCFG) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify state and errors.

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

		httpx.WriteJSON(w, http.StatusOK, "Callback successful", claims)
	}
}

func SignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Signed out", nil)
	}
}
