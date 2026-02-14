package auth

import (
	"database/sql"
	"errors"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// Role constants
const (
	RoleAdmin         = "ADMIN"
	RoleDirector      = "DIRECTOR"
	RoleFinance       = "FINANCE_MANAGER"
	RoleOperationHead = "OP_HEAD"
	RoleEmployee      = "EMPLOYEE"
)

// User represents an application user.
type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
}

var ErrNotFound = errors.New("not found")

// Migrate creates the users table if it doesn't exist.
func Migrate(db *sql.DB) error {
	q := `CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        role TEXT NOT NULL,
        approved BOOLEAN DEFAULT FALSE
    );`
	_, err := db.Exec(q)
	return err
}

// hashPassword hashes a plaintext password.
func hashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

// checkPassword compares hash with plaintext.
func checkPassword(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

// CreateUser inserts a new user. Approved should be false unless ADMIN.
func CreateUser(db *sql.DB, email, password, role string, approved bool) (*User, error) {
	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	var id int64
	err = db.QueryRow(`INSERT INTO users (email, password, role, approved) VALUES ($1, $2, $3, $4) RETURNING id`,
		email, hashed, role, approved).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Email: email, Role: role, Approved: approved}, nil
}

// CreateAdmin creates an ADMIN user and marks approved true.
func CreateAdmin(db *sql.DB, email, password string) (*User, error) {
	return CreateUser(db, email, password, RoleAdmin, true)
}

// GetUserByEmail fetches a user by email.
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	row := db.QueryRow(`SELECT id, email, password, role, approved FROM users WHERE email = $1`, email)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.Approved); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// GetUserByID fetches user by id.
func GetUserByID(db *sql.DB, id int64) (*User, error) {
	row := db.QueryRow(`SELECT id, email, password, role, approved FROM users WHERE id = $1`, id)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.Approved); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// GetPendingUsers returns users that are not yet approved.
func GetPendingUsers(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`SELECT id, email, password, role, approved FROM users WHERE approved = FALSE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.Approved); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

// ApproveUser sets approved = true for given user id.
func ApproveUser(db *sql.DB, id int64) error {
	_, err := db.Exec(`UPDATE users SET approved = TRUE WHERE id = $1`, id)
	return err
}

// Authenticate checks credentials and returns user if successful.
func Authenticate(db *sql.DB, email, password string) (*User, error) {
	u, err := GetUserByEmail(db, email)
	if err != nil {
		return nil, err
	}
	if err := checkPassword(u.Password, password); err != nil {
		return nil, err
	}
	return u, nil
}

// EnvJWTSecret returns JWT secret from env or a default.
func EnvJWTSecret() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-secret"
	}
	return s
}
