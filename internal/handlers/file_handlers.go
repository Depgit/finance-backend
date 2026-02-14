package handlers

import (
	"io"
	"net/http"
	"strconv"

	"finance-manage/internal/database"
	"finance-manage/internal/storage"

	"finance-manage/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterFileRoutes registers routes for file operations
func RegisterFileRoutes(rg *gin.Engine) {
	r := rg.Group("/api/files")
	r.Use(middleware.AuthMiddleware()) // Protected

	r.GET("/:id", getFileHandler)
}

func getFileHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	svc := database.NewFiles() // Changed to NewFiles()
	db := svc.DB()

	file, err := storage.GetFile(db, id)
	if err == storage.ErrFileNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: Add permission check - verify user has access to this file
	// For now, any authenticated user can access any file
	// In production, you should verify the file belongs to a reimbursement
	// that the user has permission to view

	// Set appropriate headers
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Disposition", "inline; filename=\""+file.Filename+"\"")
	c.Header("Content-Length", strconv.FormatInt(file.SizeBytes, 10))

	// Write file data
	c.Data(http.StatusOK, file.ContentType, file.Data)
}

// Helper function to handle file upload from multipart form
func handleFileUpload(c *gin.Context) (*int64, error) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		// No file uploaded is not an error - it's optional
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Save to database
	svc := database.NewFiles()
	db := svc.DB()

	fileID, err := storage.SaveFile(db, header.Filename, contentType, data)
	if err != nil {
		return nil, err
	}

	return &fileID, nil
}
