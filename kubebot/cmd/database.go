package cmd

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library for SQLite database interaction
	"log"
	"os"
)

var db *sql.DB // Global variable to hold the database connection

// Ensures the data directory exists, creates it if not.
func ensureDataDir() {
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		err := os.Mkdir("./data", 0755)
		if err != nil {
			log.Fatalf("Failed to create ./data directory: %v", err)
		}
	}
}

// Initializes the SQLite database, creates it if it doesn't exist.
func initDatabase() {
	ensureDataDir()
	var err error
	db, err = sql.Open("sqlite3", "./data/history.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create the release_history table if it doesn't exist
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS release_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		namespace TEXT,
		version TEXT,
		label TEXT,
		release_time DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)

	if err != nil {
		log.Fatal(err)
	}
}

// Adds a new entry to the release_history table in the database.
func AddReleaseHistory(namespace, version, label string) error {
	_, err := db.Exec(`
        INSERT INTO release_history (namespace, version, label) VALUES (?, ?, ?);`,
		namespace, version, label)
	if err != nil {
		return fmt.Errorf("failed to add release history to database: %w", err)
	}
	return nil
}

// Checks if the release_history table exists and has all required columns.
func checkReleaseHistoryTable() error {
	// Check if the release_history table exists
	var tableExists int
	queryTableExists := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='release_history';"
	err := db.QueryRow(queryTableExists).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("error checking for release_history table existence: %w", err)
	}
	if tableExists == 0 {
		return fmt.Errorf("release_history table does not exist")
	}

	// Check for the existence of required columns in the table
	requiredColumns := []string{"namespace", "version", "label", "release_time"}
	for _, column := range requiredColumns {
		var columnExists int
		queryColumnExists := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('release_history') WHERE name='%s';", column)
		err := db.QueryRow(queryColumnExists).Scan(&columnExists)
		if err != nil {
			return fmt.Errorf("error checking for column %s existence: %w", column, err)
		}
		if columnExists == 0 {
			return fmt.Errorf("column %s does not exist in release_history table", column)
		}
	}

	return nil // Return nil if the table exists with all required columns
}

// Retrieves the version prior to the current version from the release_history table.
func getPreviousVersion(namespace, currentVersion, label string) (string, error) {
	var previousVersion string
	err := db.QueryRow(`
        SELECT version FROM release_history
        WHERE namespace = ? AND version != ? AND label = ? AND id < (SELECT id FROM release_history WHERE version = ? AND label = ? ORDER BY id DESC LIMIT 1)
        ORDER BY id DESC LIMIT 1
    `, namespace, currentVersion, label, currentVersion, label).Scan(&previousVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Return nil if no previous version is found
		}
		return "", fmt.Errorf("failed to get previous version from database: %w", err)
	}
	return previousVersion, nil // Return the found previous version
}
