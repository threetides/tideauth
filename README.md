# tideauth

The threetides studio's authentication package.

`tideauth` is a Go package that handles every aspect of authentication for the
studio's services: OAuth2 and OIDC sign-in, sessions, and the user data tables
behind them. A service imports the package, fills in a `Config`, and gets back
its migrations and a set of `net/http` routes to mount, with no auth code of
its own. It currently supports Google sign-in, with more providers to come, and
will eventually replace the built-in auth in multe.

## Usage

Install the package:

```sh
go get github.com/threetides/tideauth
```

Configure it, run its migrations, and mount its routes:

```go
cfg := tideauth.Config{
	DatabaseURL:  os.Getenv("DB_CONNECTION_STRING"),
	CookieDomain: "threetides.dev",
	Google: tideauth.Google{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL") + "/api/auth/google/callback",
	},
}

auth, err := tideauth.New(cfg)
// ...
err = auth.Migrate(context.Background())
// ...
routes, err := auth.Routes()
// ...
mux.Handle("/api/auth/", http.StripPrefix("/api/auth", routes))
```

`Migrate` creates and maintains the user data tables (`users`,
`oauth_accounts`, `passwords`, and `sessions`) in the configured Postgres
database. `Routes` returns a handler serving `GET /google/sign-in`,
`GET /google/callback`, and `POST /sign-out`.

## Development

`cmd/main.go` is an example server that mounts the package, with CORS and a
health route, the way a consuming service would. It reads `CORS_ORIGIN` and
`REDIRECT_URL` from a `.env` file (or the environment).

With Go, Postgres, and [golangci-lint](https://golangci-lint.run) installed:

```sh
make dev     # run the example server on :8080
make lint    # lint the package
```

## Project structure

```
tideauth/
├── tideauth.go       the package: Config, New, Migrate, Routes
├── cmd/main.go       an example server that mounts the package
└── internal/
    ├── auth/         Google OAuth2/OIDC configuration and handlers
    ├── httpx/        HTTP client and JSON response helpers
    └── migrations/   SQL migrations for the user data tables
```

## License

See [LICENSE](LICENSE).
