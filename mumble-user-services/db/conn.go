package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
)

var db *pgxpool.Pool
var dbContext context.Context

func InitDb() error {
	dbContext = context.Background()
	var err error
	db, err = pgxpool.Connect(dbContext, os.Getenv("DB_CONN"))

	return err
}

func CloseDb() {
	db.Close()
}
