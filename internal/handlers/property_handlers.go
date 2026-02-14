package handlers

import (
	"net/http"
	"strconv"

	"finance-manage/internal/database"
	"finance-manage/internal/middleware"
	"finance-manage/internal/properties"

	"github.com/gin-gonic/gin"
)

func RegisterPropertyRoutes(rg *gin.Engine) {
	r := rg.Group("/api/properties")
	r.Use(middleware.AuthMiddleware())

	r.GET("", listPropertiesHandler)

	// Admin only routes
	admin := r.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("", createPropertyHandler)
	admin.DELETE("/:id", deletePropertyHandler)
}

func listPropertiesHandler(c *gin.Context) {
	svc := database.New()
	db := svc.DB()

	props, err := properties.List(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, props)
}

func createPropertyHandler(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := database.New()
	db := svc.DB()

	prop, err := properties.Create(db, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, prop)
}

func deletePropertyHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	svc := database.New()
	db := svc.DB()

	if err := properties.Delete(db, id); err != nil {
		if err == properties.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
