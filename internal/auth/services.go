package auth

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InsertUserData(db *pgxpool.Pool, claims claims) error {
	fmt.Println(claims)
	fmt.Println("Inserted user into db")
	return nil
}
