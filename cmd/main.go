package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
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

	cfg := tideauth.Config{
		DatabaseURL:  "postgresql://neondb_owner:npg_8diGkXRB9SuT@ep-tiny-dawn-b2ceanjt-pooler.c-6.eu-central-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require",
		CookieDomain: "threetides.dev",
	}

	auth, err := tideauth.New(cfg)
	if err != nil {
		log.Fatalln("Error configuring tideauth:", err)
		return
	}

	err = auth.Migrate(context.Background())
	if err != nil {
		log.Fatalln("Error migrating:", err)
		return
	}

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Healthy", nil)
	})

	// Handlers
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", auth.Routes()))

	log.Println("Server started on port", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), mux))
}
