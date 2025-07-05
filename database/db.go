package database

import (
	"database/sql"
	"moksarab/models"

	"github.com/gofiber/fiber/v2/log"
	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB

func InitilizeDatabase() {
	log.Debug("Opening sqlite in-memory database")
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Cloud not open sqlite database: %v", err)
	}

	log.Debug("Initilizing database schema")
	transaction, err := db.Begin()
	if err != nil {
		log.Fatalf("Cloud not Begin a database transaction during initilization: %v", err)
	}
	_, err = transaction.Exec(models.CreateQueries)
	if err != nil {
		log.Fatalf("Cloud not create schema: %v", err)
	}

	transaction.Commit()

	Db = db
}
