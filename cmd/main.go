package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/threetides/tideauth"
	"github.com/threetides/tideauth/internal/httpx"
)

const port = 8080

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	mux := http.NewServeMux()

	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(os.Getenv("CORS_ORIGIN"), ","),
		AllowCredentials: true,
	})

	// Insert the middleware
	handler := c.Handler(mux)

	cfg := tideauth.Config{
		DatabaseURL:  "postgresql://neondb_owner:npg_8diGkXRB9SuT@ep-tiny-dawn-b2ceanjt-pooler.c-6.eu-central-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require",
		CookieDomain: "threetides.dev",
		Google:       tideauth.Google{ClientID: "130842386503-76h06e8652uq9nc7nptpc8puth4dvljp.apps.googleusercontent.com", ClientSecret: "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", RedirectURL: fmt.Sprintf("%v/api/auth/google/callback", os.Getenv("REDIRECT_URL"))},
	}

	auth, err := tideauth.New(cfg)
	if err != nil {
		log.Fatalln("Error configuring tideauth:", err)
	}

	err = auth.Migrate(context.Background())
	if err != nil {
		log.Fatalln("Error migrating:", err)
	}

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Healthy", nil)
	})

	authRoutes, err := auth.Routes()
	if err != nil {
		log.Fatalln("Error setting up routes:", err)
	}

	// Handlers
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", authRoutes))

	log.Println("Server started on port", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), handler))
}
