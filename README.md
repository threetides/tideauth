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
	DB:             db,
	RedirectOrigin: os.Getenv("REDIRECT_ORIGIN"),
	CookieDomain:   os.Getenv("COOKIE_DOMAIN"),
	SecureCookie:   os.Getenv("COOKIE_SECURE") == "true",
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

### New

```go
func New(cfg Config) *Auth
```

Creates an `Auth` from the config. It does no I/O; the pgx pool is created,
owned, and closed by the caller.

| Field            | Type            | Description                                                                                             |
| ---------------- | --------------- | ------------------------------------------------------------------------------------------------------- |
| `RedirectOrigin` | `string`        | The frontend origin the Google callback redirects back to.                                              |
| `DB`             | `*pgxpool.Pool` | A pgx pool the caller creates, owns, and closes.                                                        |
| `CookieDomain`   | `string`        | The `Domain` attribute on every cookie tideauth sets. Leave empty for a host-only cookie (localhost).   |
| `SecureCookie`   | `bool`          | Marks every cookie tideauth sets as `Secure`. Use `true` behind https.                                  |
| `Google`         | `Google`        | Client id, client secret, and the callback url.                                                         |

### Migrate

```go
func (a *Auth) Migrate(ctx context.Context) error
```

Applies the embedded SQL migrations, creating and maintaining the user data
tables: `users`, `oauth_accounts`, `passwords`, and `sessions`. Safe to run on
every startup.

### Routes

```go
func (a *Auth) Routes() (http.Handler, error)
```

Returns a handler serving the auth endpoints:

| Route                   | Description                                    | Input                                                    |
| ----------------------- | ---------------------------------------------- | -------------------------------------------------------- |
| `POST /sign-up`         | Create a user with email and password          | Body: `{ "email", "name", "password" }`                  |
| `POST /sign-in`         | Sign in with email and password                | Body: `{ "email", "password" }`                          |
| `GET /google/sign-in`   | Redirect to Google's consent screen            | Query: `return_success`, `return_error` (both optional)  |
| `GET /google/callback`  | Complete the Google sign-in, then redirect back |                                                         |
| `GET /session`          | The signed-in user's profile                   |                                                          |
| `POST /sign-out`        | End the session                                |                                                          |

Bodies are JSON and every field is a string. Sign-up validates the email
address and stores the password as a bcrypt hash.

#### Google redirects

`/google/sign-in` takes two optional query parameters. Both are paths on your
frontend; they ride along in the OAuth state and decide where
`/google/callback` sends the browser once Google returns. The base is
`Config.RedirectOrigin`, and with a parameter omitted its redirect goes to the
origin's root.

| Parameter        | Redirects to                           | When                            |
| ---------------- | -------------------------------------- | ------------------------------- |
| `return_success` | `<origin>/<return_success>`            | The sign-in succeeded           |
| `return_error`   | `<origin>/<return_error>?error=<code>` | The sign-in failed              |

The `error` codes:

| Code                        | Meaning                                                              |
| --------------------------- | -------------------------------------------------------------------- |
| `unauthorized`              | The OAuth state did not match (expired or forged)                    |
| `internal_server_error`     | Exchanging or verifying the Google token failed                      |
| `google_email_not_verified` | Google reports the account's email as unverified                     |
| `local_email_not_verified`  | A password account with this email exists but has not verified it    |

For example, a frontend served at `RedirectOrigin` would start the flow with:

```
GET /api/auth/google/sign-in?return_success=/dashboard&return_error=/sign-in
```

### Protected

```go
func (a *Auth) Protected(next http.HandlerFunc) http.HandlerFunc
```

Wraps a handler with session validation: it checks the session cookie against
the database, slides the session's expiry another 30 days, and puts the user's
id in the request context. Requests without a valid session get a 401.

### UserIDFromContext

```go
func (a *Auth) UserIDFromContext(ctx context.Context) (string, bool)
```

Reads the user id that `Protected` stored in the request context; the bool
reports whether it was there.

## Development

`cmd/main.go` is an example server that mounts the package the way a consuming
service would, with CORS, a health route, and a protected test route. It reads
`DB_CONNECTION_STRING`, `CORS_ORIGIN`, `REDIRECT_ORIGIN`, `REDIRECT_URL`,
`COOKIE_DOMAIN`, `COOKIE_SECURE`, `TEST_CLIENT_ID`, and `TEST_CLIENT_SECRET`
from a `.env` file (or the environment).

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
