package database

import (
	"database/sql"
	"fmt"
	"rockets-backend/pkg"

	_ "github.com/lib/pq"
)

func NewConnection() (*sql.DB, error) {
	host := pkg.GetEnv("DB_HOST", "localhost")
	port := pkg.GetEnv("DB_PORT", "5432")
	user := pkg.GetEnv("DB_USER", "postgres")
	password := pkg.GetEnv("DB_PASSWORD", "postgres")
	dbname := pkg.GetEnv("DB_NAME", "rockets")
	sslmode := pkg.GetEnv("DB_SSLMODE", "disable")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}
