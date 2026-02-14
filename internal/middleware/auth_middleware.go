package middleware

import (
	"net/http"

	"finance-manage/internal/auth"
	"finance-manage/internal/database"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware verifies JWT and loads user into context (key: "user").
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if authz == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		// Expect "Bearer <token>"
		var token string
		if len(authz) > 7 && authz[:7] == "Bearer " {
			token = authz[7:]
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		claims, err := auth.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Load user from DB
		svc := database.New()
		db := svc.DB()
		u, err := auth.GetUserByID(db, claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.Set("user", u)
		c.Next()
	}
}

// RequireRole ensures the current user has one of allowed roles and is approved.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		u := v.(*auth.User)
		if !u.Approved {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user not approved"})
			return
		}
		if _, ok := allowed[u.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}

// RequireAdmin is a convenience wrapper for ADMIN-only endpoints.
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(auth.RoleAdmin)
}
