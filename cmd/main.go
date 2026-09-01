package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/threetides/tideauth"
	"github.com/threetides/tideauth/cmd/internal/test"
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

	// Connect to db
	db, err := pgxpool.New(context.Background(), os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		log.Fatalln("Error connecting to db:", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		log.Fatalln("Error pinging db:", err)
	}
	log.Println("Connected to db")

	// Create config for tideauth
	cfg := tideauth.Config{
		DB:           db,
		CookieDomain: "threetides.dev",
		Google:       tideauth.Google{ClientID: os.Getenv("TEST_CLIENT_ID"), ClientSecret: os.Getenv("TEST_CLIENT_SECRET"), RedirectURL: fmt.Sprintf("%v/api/auth/google/callback", os.Getenv("REDIRECT_URL"))},
	}

	auth := tideauth.New(cfg)

	// Run migrations
	err = auth.Migrate(context.Background())
	if err != nil {
		log.Fatalln("Error migrating:", err)
	}

	// Create default routes exported by tideauth
	authRoutes, err := auth.Routes()
	if err != nil {
		log.Fatalln("Error setting up routes:", err)
	}

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Healthy", nil)
	})

	// Handlers
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", authRoutes))

	mux.HandleFunc("GET /api/test", auth.Protected(test.TestHandler()))

	log.Println("Server started on port", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), handler))
}
