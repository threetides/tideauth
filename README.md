# tideauth

The threetides studio's authentication package.

`tideauth` is a Go package that handles every aspect of authentication for the
studio's services: OAuth2 and OIDC sign-in, sessions, and the user data tables
behind them. A service imports the package, fills in a `Config`, and gets back
its migrations, a set of `net/http` routes to mount, and middleware for
protecting its own routes, with no auth code of its own. It currently supports
email and password sign-in and Google sign-in, with more providers to come.

## Usage

Install the package:

```sh
go get github.com/threetides/tideauth@latest
```

Configure it, run its migrations, and mount its routes:

```go
// Connect to db
db, err := pgxpool.New(context.Background(), os.Getenv("DB_CONNECTION_STRING"))
if err != nil {
	log.Fatalln("Error connecting to db:", err)
}

// Create config for tideauth
cfg := tideauth.Config{
	DB:           db,
	CookieDomain: "threetides.dev",
	Google: tideauth.Google{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL") + "/api/auth/google/callback",
	},
}

auth := tideauth.New(cfg)

// Run migrations
err = auth.Migrate(context.Background())
if err != nil {
	log.Fatalln("Error running migrations:", err)
}

// Create default routes exported by tideauth
routes, err := auth.Routes()
if err != nil {
	log.Fatalln("Error setting up routes:", err)
}

// Handlers
mux.Handle("/api/auth/", http.StripPrefix("/api/auth", routes))

// Protect your own routes with the session middleware
mux.HandleFunc("GET /api/me", auth.Protected(meHandler))
```

## API

- `New(cfg Config) *Auth` creates an `Auth` from the config. It does no I/O;
  the pgx pool is created, owned, and closed by the caller.
- `Migrate(ctx context.Context) error` applies the embedded SQL migrations,
  creating and maintaining the user data tables: `users`, `oauth_accounts`,
  `passwords`, and `sessions`. Safe to run on every startup.
- `Routes() (http.Handler, error)` returns a handler serving `POST /sign-up`,
  `POST /sign-in`, `GET /google/sign-in`, `GET /google/callback`,
  `GET /session`, and `POST /sign-out`. Sign-up validates the email address
  and stores the password as a bcrypt hash.
- `Protected(next http.HandlerFunc) http.HandlerFunc` wraps a handler with
  session validation: it checks the session cookie against the database,
  slides the session's expiry another 30 days, and puts the user's id in the
  request context. Requests without a valid session get a 401.
- `UserIDFromContext(ctx context.Context) (string, bool)` reads the user id
  that `Protected` stored in the request context; the bool reports whether it
  was there.

## Development

`cmd/main.go` is an example server that mounts the package the way a consuming
service would, with CORS, a health route, and a protected test route. It reads
`DB_CONNECTION_STRING`, `CORS_ORIGIN`, `REDIRECT_URL`, `TEST_CLIENT_ID`, and
`TEST_CLIENT_SECRET` from a `.env` file (or the environment).

With Go, Postgres, and [golangci-lint](https://golangci-lint.run) installed:

```sh
make dev     # run the example server on :8080
make lint    # lint the package
```

## Project structure

```
tideauth/
├── tideauth.go       the package: Config, New, Migrate, Routes, Protected,
│                     and UserIDFromContext
├── cmd/              an example server that mounts the package
└── internal/
    ├── auth/         Google OAuth2/OIDC, session handlers, and middleware
    ├── httpx/        HTTP client and JSON response helpers
    └── migrations/   SQL migrations for the user data tables
```

## License

See [LICENSE](LICENSE).
