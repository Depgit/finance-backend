package storage

import (
	"database/sql"
	"errors"
	"time"
)

const (
	// MaxFileSize is the maximum allowed file size (10MB)
	MaxFileSize = 10 * 1024 * 1024
)

var (
	ErrFileTooLarge    = errors.New("file size exceeds maximum allowed size")
	ErrFileNotFound    = errors.New("file not found")
	ErrInvalidFileType = errors.New("invalid file type")
)

// AllowedContentTypes defines the permitted file types
var AllowedContentTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/jpg":       true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // .docx
	"application/msword": true, // .doc
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true, // .xlsx
	"application/vnd.ms-excel": true, // .xls
	"text/plain":               true,
}

// File represents a stored file
type File struct {
	ID          int64     `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Data        []byte    `json:"-"` // Don't serialize data in JSON
	UploadedAt  time.Time `json:"uploaded_at"`
}

// Migrate creates the files table
func Migrate(db *sql.DB) error {
	q := `CREATE TABLE IF NOT EXISTS files (
		id SERIAL PRIMARY KEY,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		data BYTEA NOT NULL,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(q)
	return err
}

// SaveFile saves a file to the database
func SaveFile(db *sql.DB, filename string, contentType string, data []byte) (int64, error) {
	// Validate file size
	if len(data) > MaxFileSize {
		return 0, ErrFileTooLarge
	}

	// Validate content type
	if !AllowedContentTypes[contentType] {
		return 0, ErrInvalidFileType
	}

	var id int64
	q := `INSERT INTO files (filename, content_type, size_bytes, data) VALUES ($1, $2, $3, $4) RETURNING id`
	err := db.QueryRow(q, filename, contentType, int64(len(data)), data).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetFile retrieves a file from the database
func GetFile(db *sql.DB, fileID int64) (*File, error) {
	f := &File{}
	q := `SELECT id, filename, content_type, size_bytes, data, uploaded_at FROM files WHERE id = $1`
	err := db.QueryRow(q, fileID).Scan(&f.ID, &f.Filename, &f.ContentType, &f.SizeBytes, &f.Data, &f.UploadedAt)
	if err == sql.ErrNoRows {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// GetFileMetadata retrieves file metadata without the data blob
func GetFileMetadata(db *sql.DB, fileID int64) (*File, error) {
	f := &File{}
	q := `SELECT id, filename, content_type, size_bytes, uploaded_at FROM files WHERE id = $1`
	err := db.QueryRow(q, fileID).Scan(&f.ID, &f.Filename, &f.ContentType, &f.SizeBytes, &f.UploadedAt)
	if err == sql.ErrNoRows {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// DeleteFile removes a file from the database
func DeleteFile(db *sql.DB, fileID int64) error {
	q := `DELETE FROM files WHERE id = $1`
	res, err := db.Exec(q, fileID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrFileNotFound
	}

	return nil
}
