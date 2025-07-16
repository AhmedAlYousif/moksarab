package database

import (
	"database/sql"
	"moksarab/config"
	"moksarab/models"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2/log"
	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB

func InitilizeDatabase() {
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = ":memory:"
		log.Debug("Opening sqlite in-memory database (default, SQLITE_DB_PATH not set)")
	} else if strings.Contains(dbPath, ":memory:") {
		log.Debugf("Opening sqlite in-memory database (SQLITE_DB_PATH=%s)", dbPath)
	} else {
		log.Debugf("Opening sqlite database at path: %s", dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Could not open sqlite database: %v", err)
	}

	log.Debug("Initilizing database schema")
	transaction, err := db.Begin()
	if err != nil {
		log.Fatalf("Could not Begin a database transaction during initilization: %v", err)
	}
	_, err = transaction.Exec(models.CreateQueries)
	if err != nil {
		log.Fatalf("Could not create schema: %v", err)
	}

	if !config.WorkspaceEnabled {
		_, insertError := transaction.Exec("INSERT INTO workspace (id, name, description) VALUES (?, ?, ?)",
			4269,
			"default",
			"Default workspace since workspace feature is disabled!",
		)
		if insertError != nil {
			log.Fatalf("Could not create default workspace while since workspace feature is disabled: %v", insertError)
		}
	}

	transaction.Commit()

	Db = db
}
