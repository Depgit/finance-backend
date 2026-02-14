package main

import (
	"log"
	"os"
	"time"

	"database/sql"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Println("Starting database separation migration...")

	// 1. Connect to both databases
	mainDBUrl := os.Getenv("BLUEPRINT_DB_URL")
	if mainDBUrl == "" {
		mainDBUrl = "./data/finance.db"
	}
	mainDB, err := sql.Open("sqlite3", mainDBUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer mainDB.Close()

	filesDBUrl := os.Getenv("FILES_DB_URL")
	if filesDBUrl == "" {
		filesDBUrl = "./data/files.db"
	}
	filesDB, err := sql.Open("sqlite3", filesDBUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer filesDB.Close()

	// 2. Ensure files table exists in files.db
	log.Println("Ensuring files table exists in files.db...")
	_, err = filesDB.Exec(`CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		data BLOB NOT NULL,
		uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Read files from main DB
	log.Println("Reading files from main database...")
	rows, err := mainDB.Query(`SELECT id, filename, content_type, size_bytes, data, uploaded_at FROM files`)
	if err != nil {
		// If table doesn't exist, it might have been already dropped or not created
		log.Printf("Could not query files from main DB (might not exist): %v", err)
		log.Println("Migration might have already run.")
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var filename, contentType string
		var sizeBytes int64
		var data []byte
		var uploadedAt time.Time

		if err := rows.Scan(&id, &filename, &contentType, &sizeBytes, &data, &uploadedAt); err != nil {
			log.Fatal(err)
		}

		// 4. Insert into files DB (preserving ID)
		_, err := filesDB.Exec(`INSERT INTO files (id, filename, content_type, size_bytes, data, uploaded_at) VALUES (?, ?, ?, ?, ?, ?)`,
			id, filename, contentType, sizeBytes, data, uploadedAt)

		if err != nil {
			log.Printf("Failed to insert file %d (might already exist): %v", id, err)
		} else {
			count++
		}
	}

	log.Printf("Migrated %d files to files.db", count)

	// 5. Drop files table from main DB
	log.Println("Dropping files table from main database...")
	_, err = mainDB.Exec(`DROP TABLE files`)
	if err != nil {
		log.Printf("Failed to drop files table: %v", err)
	} else {
		log.Println("Dropped files table from main DB.")
	}

	log.Println("Migration completed successfully!")
}
