package tideauth

import "log"

type Config struct {
	DBConn string
}

type Auth struct {
	Config Config
}

func (a *Auth) Migrate() {
	log.Println("Connection string is:", a.Config.DBConn)
}

func (a *Auth) New(cfg Config) {
	a.Config = cfg
}
