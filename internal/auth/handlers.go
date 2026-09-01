package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/threetides/tideauth/internal/httpx"
)

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
		state := base64.RawURLEncoding.EncodeToString(b)

		// Set short lived httpOnlyCookie
		cookie := http.Cookie{
			Name:     "state",
			Value:    state,
			Path:     "/",
			Expires:  time.Now().Add(5 * time.Minute),
			HttpOnly: true,
			Secure:   os.Getenv("COOKIE_SECURE") == "true",
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

		if state != cookie.Value {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		// Clear cookie
		clearedCookie := &http.Cookie{
			Name:     "state",
			Value:    state,
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   os.Getenv("COOKIE_SECURE") == "true",
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, clearedCookie)

		// Get OAuth2 token
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
		fmt.Println(claims)

		http.Redirect(w, r, os.Getenv("CORS_ORIGIN")+"/", http.StatusTemporaryRedirect)
	}
}

func SignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Signed out", nil)
	}
}
