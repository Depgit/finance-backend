package main

import (
	"flag"
	"fmt"
	"os"

	"finance-manage/internal/auth"
	"finance-manage/internal/database"
)

func main() {
	email := flag.String("email", "", "admin email")
	password := flag.String("password", "", "admin password")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Println("usage: go run cmd/user.create-admin -email you@host -password secret")
		os.Exit(2)
	}

	svc := database.New()
	db := svc.DB()
	// Ensure users table exists before creating admin
	if err := auth.Migrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "failed to migrate auth schema: %v\n", err)
		os.Exit(1)
	}
	if _, err := auth.CreateAdmin(db, *email, *password); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create admin: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ADMIN user created")
}
