package handlers

import (
	"net/http"
	"strconv"

	"finance-manage/internal/auth"
	"finance-manage/internal/database"
	"finance-manage/internal/reimburse"
	"finance-manage/internal/storage"

	"finance-manage/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterReimburseRoutes registers routes for reimbursement.
func RegisterReimburseRoutes(rg *gin.Engine) {
	r := rg.Group("/api/reimbursements")
	r.Use(middleware.AuthMiddleware()) // Protected

	r.POST("", submitReimbursementHandler)
	r.GET("", listReimbursementsHandler)
	r.GET("/balance", balanceHandler)
	r.POST("/:id/approve", approveReimbursementHandler)
	r.POST("/:id/reject", rejectReimbursementHandler)
}

type submitReq struct {
	Type        string  `json:"type" binding:"required,oneof=EXPENSE ADVANCE"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description" binding:"required"`
	Proof       string  `json:"proof"`
	Property    string  `json:"property" binding:"required"`
}

func submitReimbursementHandler(c *gin.Context) {
	// Try to parse as multipart form first (for file uploads)
	contentType := c.GetHeader("Content-Type")
	var reqType, desc, proof, property string
	var amount float64
	var proofFileID *int64

	if contentType == "application/json" || contentType == "application/json; charset=utf-8" {
		// Handle JSON request (backward compatibility)
		var req submitReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		reqType = req.Type
		amount = req.Amount
		desc = req.Description
		proof = req.Proof
		property = req.Property
	} else {
		// Handle multipart form data (with file upload)
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB max
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
			return
		}

		reqType = c.PostForm("type")
		amountStr := c.PostForm("amount")
		desc = c.PostForm("description")
		proof = c.PostForm("proof")
		property = c.PostForm("property")

		// Validate required fields
		if reqType == "" || amountStr == "" || desc == "" || property == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type, amount, description, and property are required"})
			return
		}

		if reqType != "EXPENSE" && reqType != "ADVANCE" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type must be EXPENSE or ADVANCE"})
			return
		}

		var err error
		amount, err = strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a positive number"})
			return
		}

		// Handle file upload
		fileID, err := handleFileUpload(c)
		if err != nil {
			if err == storage.ErrFileTooLarge {
				c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 10MB limit"})
				return
			}
			if err == storage.ErrInvalidFileType {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
			return
		}
		proofFileID = fileID
	}

	u := c.MustGet("user").(*auth.User)

	svc := database.New()
	db := svc.DB()

	r, err := reimburse.Create(db, u.ID, reqType, amount, desc, proof, proofFileID, property)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, r)
}

func listReimbursementsHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	var list []*reimburse.Reimbursement
	var err error

	// Determine what to show based on role
	switch u.Role {
	case auth.RoleAdmin, auth.RoleDirector:
		list, err = reimburse.ListAll(db)
	case auth.RoleOperationHead:
		// Op Head sees all Pending requests for approval,
		// AND presumably their own? or just all?
		// For simplicity, let's say they see ALL Pending for approval tasks,
		// and maybe separate view for history.
		// Given the requirements, "operation head request an amount finance manager will aprove it".
		// So OpHead can also submit requests.

		// If query param 'view=approval' is set, show actionable items.
		if c.Query("view") == "approval" {
			list, err = reimburse.ListPendingForOpHead(db)
		} else {
			// Default view: Own requests
			list, err = reimburse.ListByUserID(db, u.ID)
		}
	case auth.RoleFinance:
		if c.Query("view") == "approval" {
			list, err = reimburse.ListPendingForFinance(db)
		} else {
			list, err = reimburse.ListByUserID(db, u.ID)
		}
	default: // EMPLOYEE
		list, err = reimburse.ListByUserID(db, u.ID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return empty slice instead of null
	if list == nil {
		list = []*reimburse.Reimbursement{}
	}
	c.JSON(http.StatusOK, list)
}

func balanceHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	balance, err := reimburse.GetUserBalance(db, u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, balance)
}

func approveReimbursementHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Parse approval amount from body
	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount is required and must be > 0"})
		return
	}

	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	// Get current request
	r, err := reimburse.GetByID(db, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// State machine logic with partial approval
	switch r.Status {
	case reimburse.StatusPending:
		// Op Head approval
		if u.Role != auth.RoleOperationHead && u.Role != auth.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "only op head or admin can approve pending requests"})
			return
		}
		// Validate: approved amount <= requested amount
		if req.Amount > r.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "approved amount cannot exceed requested amount"})
			return
		}
		if err := reimburse.ApproveByOp(db, id, req.Amount, u.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	case reimburse.StatusApprovedByOp:
		// Finance approval (final)
		if u.Role != auth.RoleFinance && u.Role != auth.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "only finance can approve op-approved requests"})
			return
		}
		// Validate: finance amount <= op head approved amount
		if req.Amount > r.ApprovedByOpAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "finance approved amount cannot exceed op head approved amount"})
			return
		}
		if err := reimburse.ApproveByFinance(db, id, req.Amount, u.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	case reimburse.StatusApprovedByFinance:
		// If it's an advance, maybe allow marking as Settled?
		if r.Type == reimburse.TypeAdvance && (u.Role == auth.RoleFinance || u.Role == auth.RoleAdmin) {
			if err := reimburse.UpdateStatus(db, id, reimburse.StatusSettled); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "already fully approved"})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot approve request in this status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func rejectReimbursementHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	u := c.MustGet("user").(*auth.User)
	// Any approver can reject?
	if u.Role == auth.RoleEmployee {
		c.JSON(http.StatusForbidden, gin.H{"error": "employees cannot reject"})
		return
	}

	svc := database.New()
	db := svc.DB()
	if err := reimburse.UpdateStatus(db, id, reimburse.StatusRejected); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}
