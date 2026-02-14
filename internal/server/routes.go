package server

import (
	"net/http"
	"os"

	"finance-manage/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	clientURL := os.Getenv("CLIENT_URL")
	allowOrigins := []string{"http://localhost:5173", "http://localhost:5174"}
	if clientURL != "" {
		allowOrigins = append(allowOrigins, clientURL)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/", s.HelloWorldHandler)

	r.GET("/health", s.healthHandler)

	// Auth routes
	handlers.RegisterAuthRoutes(r)
	// Reimbursement routes
	handlers.RegisterReimburseRoutes(r)
	// File routes
	handlers.RegisterFileRoutes(r)
	// Report routes
	handlers.RegisterReportRoutes(r)
	// Property routes
	handlers.RegisterPropertyRoutes(r)

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}
