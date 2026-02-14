package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"finance-manage/internal/auth"
	"finance-manage/internal/database"
	"finance-manage/internal/properties"
	"finance-manage/internal/reimburse"
	"finance-manage/internal/storage"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port int

	db database.Service
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,

		db: database.New(),
	}

	// Ensure auth migrations (users table)
	if err := auth.Migrate(NewServer.db.DB()); err != nil {
		panic(err)
	}
	// Ensure reimbursement migrations
	if err := reimburse.Migrate(NewServer.db.DB()); err != nil {
		panic(err)
	}
	// Ensure storage migrations (files table)
	filesSvc := database.NewFiles()
	if err := storage.Migrate(filesSvc.DB()); err != nil {
		panic(err)
	}
	// Ensure properties migrations
	if err := properties.Migrate(NewServer.db.DB()); err != nil {
		panic(err)
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
