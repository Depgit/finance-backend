package handlers

import (
	"net/http"
	"strconv"

	"finance-manage/internal/auth"
	"finance-manage/internal/database"
	"finance-manage/internal/middleware"
	"finance-manage/internal/notify"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers auth-related routes on the given router.
func RegisterAuthRoutes(rg *gin.Engine) {
	r := rg.Group("/api/auth")
	r.POST("/register", registerHandler)
	r.POST("/login", loginHandler)

	// Protected
	p := rg.Group("/api")
	p.Use(middleware.AuthMiddleware())
	p.GET("/me", meHandler)

	// Admin endpoints
	a := rg.Group("/api/admin")
	a.Use(middleware.AuthMiddleware(), middleware.RequireAdmin())
	a.POST("/approve/:id", approveHandler)
	a.GET("/pending", pendingHandler)
}

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func registerHandler(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// FORCE ROLE TO EMPLOYEE
	// Security: Prevent users from registering as ADMIN/Manager directly
	role := auth.RoleEmployee

	svc := database.New()
	db := svc.DB()
	// Newly created users must be approved by admin (unless ADMIN role created via CLI)
	u, err := auth.CreateUser(db, req.Email, req.Password, role, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Notify admin (no-op if SMTP not configured)
	go func() {
		_ = notify.NotifyAdminNewRegistration(u.Email, u.Role)
	}()
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email, "role": u.Role, "approved": u.Approved})
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func loginHandler(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	svc := database.New()
	db := svc.DB()
	u, err := auth.Authenticate(db, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !u.Approved {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not approved by admin"})
		return
	}
	token, err := auth.GenerateToken(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func meHandler(c *gin.Context) {
	v, _ := c.Get("user")
	u := v.(*auth.User)
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "email": u.Email, "role": u.Role, "approved": u.Approved})
}

func approveHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check for role update in body
	var req struct {
		Role string `json:"role"`
	}
	// Ignore bind error as body is optional
	_ = c.ShouldBindJSON(&req)

	svc := database.New()
	db := svc.DB()

	if req.Role != "" {
		// Update role if provided
		// Validate role?
		if _, err := db.Exec("UPDATE users SET role = $1 WHERE id = $2", req.Role, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
			return
		}
	}

	if err := auth.ApproveUser(db, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

func pendingHandler(c *gin.Context) {
	svc := database.New()
	db := svc.DB()
	users, err := auth.GetPendingUsers(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return public fields only
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		out = append(out, gin.H{"id": u.ID, "email": u.Email, "role": u.Role})
	}
	c.JSON(http.StatusOK, out)
}
