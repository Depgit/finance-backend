package properties

import (
	"database/sql"
	"errors"
)

var (
	ErrDuplicateName = errors.New("property name already exists")
	ErrNotFound      = errors.New("property not found")
)

type Property struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func Migrate(db *sql.DB) error {
	q := `CREATE TABLE IF NOT EXISTS properties (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE
	);`
	_, err := db.Exec(q)
	return err
}

func Create(db *sql.DB, name string) (*Property, error) {
	var id int64
	q := `INSERT INTO properties (name) VALUES ($1) RETURNING id`
	// Use QueryRow for RETURNING support in Postgres
	err := db.QueryRow(q, name).Scan(&id)
	if err != nil {
		// Checks for duplicate constraint?
		// For postgres, it will be a pq error.
		return nil, err
	}
	return &Property{ID: id, Name: name}, nil
}

func List(db *sql.DB) ([]*Property, error) {
	rows, err := db.Query(`SELECT id, name FROM properties ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var props []*Property
	for rows.Next() {
		var p Property
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		props = append(props, &p)
	}
	if props == nil {
		props = []*Property{}
	}
	return props, nil
}

func Delete(db *sql.DB, id int64) error {
	res, err := db.Exec(`DELETE FROM properties WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
