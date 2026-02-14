package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"finance-manage/internal/auth"
	"finance-manage/internal/database"
	"finance-manage/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterReportRoutes registers routes for reports and analytics.
func RegisterReportRoutes(rg *gin.Engine) {
	r := rg.Group("/api/reports")
	r.Use(middleware.AuthMiddleware()) // Protected

	r.GET("/summary", reportSummaryHandler)
	r.GET("/monthly", monthlyReportHandler)
	r.GET("/breakdown", typeBreakdownHandler)
	r.GET("/trends", trendsHandler)
	r.GET("/details", detailsHandler)
}

// ReportSummary contains high-level statistics.
type ReportSummary struct {
	TotalRequests       int     `json:"total_requests"`
	TotalAmount         float64 `json:"total_amount"`
	TotalApproved       float64 `json:"total_approved"`
	TotalPending        float64 `json:"total_pending"`
	TotalRejected       float64 `json:"total_rejected"`
	AdvanceTotal        float64 `json:"advance_total"`
	ExpenseTotal        float64 `json:"expense_total"`
	OutstandingAdvances float64 `json:"outstanding_advances"`
}

// MonthlyData represents aggregated data for a month.
type MonthlyData struct {
	Month   string  `json:"month"`   // Format: "2026-01"
	Count   int     `json:"count"`   // Number of requests
	Amount  float64 `json:"amount"`  // Total amount requested
	Advance float64 `json:"advance"` // Advance amount
	Expense float64 `json:"expense"` // Expense amount
}

// TypeBreakdown represents breakdown by type.
type TypeBreakdown struct {
	Type   string  `json:"type"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

// TrendData represents time-series data.
type TrendData struct {
	Date   string  `json:"date"` // Format: "2026-01-27"
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

// reportSummaryHandler returns summary statistics.
func reportSummaryHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	// Parse date filters
	startDate := c.Query("start_date") // Format: "2026-01-01"
	endDate := c.Query("end_date")     // Format: "2026-01-31"

	summary, err := getReportSummary(db, u, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// monthlyReportHandler returns monthly aggregated data.
func monthlyReportHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	data, err := getMonthlyReport(db, u, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// typeBreakdownHandler returns breakdown by type.
func typeBreakdownHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	data, err := getTypeBreakdown(db, u, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// trendsHandler returns daily trend data.
func trendsHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	data, err := getTrends(db, u, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// detailsHandler returns detailed transaction list for export.
func detailsHandler(c *gin.Context) {
	u := c.MustGet("user").(*auth.User)
	svc := database.New()
	db := svc.DB()

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	data, err := getDetails(db, u, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// Helper functions

func buildWhereClause(u *auth.User, startDate, endDate string) (string, []interface{}) {
	where := "WHERE 1=1"
	args := []interface{}{}
	paramIdx := 1

	// Role-based filtering
	if u.Role == auth.RoleEmployee {
		where += fmt.Sprintf(" AND user_id = $%d", paramIdx)
		paramIdx++
		args = append(args, u.ID)
	}
	// Admin, Director, Finance Manager, Op Head see all

	// Date filtering
	if startDate != "" {
		where += fmt.Sprintf(" AND DATE(request_date) >= $%d", paramIdx)
		paramIdx++
		args = append(args, startDate)
	}
	if endDate != "" {
		where += fmt.Sprintf(" AND DATE(request_date) <= $%d", paramIdx)
		paramIdx++
		args = append(args, endDate)
	}

	return where, args
}

func getReportSummary(db *sql.DB, u *auth.User, startDate, endDate string) (*ReportSummary, error) {
	where, args := buildWhereClause(u, startDate, endDate)

	summary := &ReportSummary{}

	// Total requests and amount
	query := `SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM reimbursements ` + where
	err := db.QueryRow(query, args...).Scan(&summary.TotalRequests, &summary.TotalAmount)
	if err != nil {
		return nil, err
	}

	// Approved amount
	query = `SELECT COALESCE(SUM(approved_by_finance_amount), 0) FROM reimbursements ` + where + ` AND status = 'APPROVED_BY_FINANCE'`
	err = db.QueryRow(query, args...).Scan(&summary.TotalApproved)
	if err != nil {
		return nil, err
	}

	// Pending amount
	query = `SELECT COALESCE(SUM(amount), 0) FROM reimbursements ` + where + ` AND status IN ('PENDING', 'APPROVED_BY_OP')`
	err = db.QueryRow(query, args...).Scan(&summary.TotalPending)
	if err != nil {
		return nil, err
	}

	// Rejected amount
	query = `SELECT COALESCE(SUM(amount), 0) FROM reimbursements ` + where + ` AND status = 'REJECTED'`
	err = db.QueryRow(query, args...).Scan(&summary.TotalRejected)
	if err != nil {
		return nil, err
	}

	// Advance total
	query = `SELECT COALESCE(SUM(amount), 0) FROM reimbursements ` + where + ` AND type = 'ADVANCE'`
	err = db.QueryRow(query, args...).Scan(&summary.AdvanceTotal)
	if err != nil {
		return nil, err
	}

	// Expense total
	query = `SELECT COALESCE(SUM(amount), 0) FROM reimbursements ` + where + ` AND type = 'EXPENSE'`
	err = db.QueryRow(query, args...).Scan(&summary.ExpenseTotal)
	if err != nil {
		return nil, err
	}

	// Outstanding advances (approved advances - approved expenses)
	advQuery := `SELECT COALESCE(SUM(approved_by_finance_amount), 0) FROM reimbursements ` + where + ` AND type = 'ADVANCE' AND status = 'APPROVED_BY_FINANCE'`
	var totalAdv float64
	err = db.QueryRow(advQuery, args...).Scan(&totalAdv)
	if err != nil {
		return nil, err
	}

	expQuery := `SELECT COALESCE(SUM(approved_by_finance_amount), 0) FROM reimbursements ` + where + ` AND type = 'EXPENSE' AND status = 'APPROVED_BY_FINANCE'`
	var totalExp float64
	err = db.QueryRow(expQuery, args...).Scan(&totalExp)
	if err != nil {
		return nil, err
	}

	summary.OutstandingAdvances = totalAdv - totalExp
	if summary.OutstandingAdvances < 0 {
		summary.OutstandingAdvances = 0
	}

	return summary, nil
}

func getMonthlyReport(db *sql.DB, u *auth.User, startDate, endDate string) ([]MonthlyData, error) {
	where, args := buildWhereClause(u, startDate, endDate)

	query := `
		SELECT 
			TO_CHAR(request_date, 'YYYY-MM') as month,
			COUNT(*) as count,
			COALESCE(SUM(CASE WHEN status IN ('APPROVED_BY_FINANCE', 'SETTLED') THEN approved_by_finance_amount ELSE amount END), 0) as amount,
			COALESCE(SUM(CASE WHEN type = 'ADVANCE' THEN (CASE WHEN status IN ('APPROVED_BY_FINANCE', 'SETTLED') THEN approved_by_finance_amount ELSE amount END) ELSE 0 END), 0) as advance,
			COALESCE(SUM(CASE WHEN type = 'EXPENSE' THEN (CASE WHEN status IN ('APPROVED_BY_FINANCE', 'SETTLED') THEN approved_by_finance_amount ELSE amount END) ELSE 0 END), 0) as expense
		FROM reimbursements
		` + where + `
		GROUP BY month
		ORDER BY month DESC
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []MonthlyData
	for rows.Next() {
		var m MonthlyData
		if err := rows.Scan(&m.Month, &m.Count, &m.Amount, &m.Advance, &m.Expense); err != nil {
			return nil, err
		}
		data = append(data, m)
	}

	if data == nil {
		data = []MonthlyData{}
	}

	return data, nil
}

func getTypeBreakdown(db *sql.DB, u *auth.User, startDate, endDate string) ([]TypeBreakdown, error) {
	where, args := buildWhereClause(u, startDate, endDate)

	query := `
		SELECT 
			type,
			COUNT(*) as count,
			COALESCE(SUM(CASE WHEN status IN ('APPROVED_BY_FINANCE', 'SETTLED') THEN approved_by_finance_amount ELSE amount END), 0) as amount
		FROM reimbursements
		` + where + `
		GROUP BY type
		ORDER BY amount DESC
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []TypeBreakdown
	for rows.Next() {
		var t TypeBreakdown
		if err := rows.Scan(&t.Type, &t.Count, &t.Amount); err != nil {
			return nil, err
		}
		data = append(data, t)
	}

	if data == nil {
		data = []TypeBreakdown{}
	}

	return data, nil
}

func getTrends(db *sql.DB, u *auth.User, startDate, endDate string) ([]TrendData, error) {
	where, args := buildWhereClause(u, startDate, endDate)

	query := `
		SELECT 
			DATE(request_date) as date,
			COUNT(*) as count,
			COALESCE(SUM(CASE WHEN status IN ('APPROVED_BY_FINANCE', 'SETTLED') THEN approved_by_finance_amount ELSE amount END), 0) as amount
		FROM reimbursements
		` + where + `
		GROUP BY date
		ORDER BY date ASC
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []TrendData
	for rows.Next() {
		var t TrendData
		if err := rows.Scan(&t.Date, &t.Count, &t.Amount); err != nil {
			return nil, err
		}
		data = append(data, t)
	}

	if data == nil {
		data = []TrendData{}
	}

	return data, nil
}

// DetailRow represents a single transaction for export.
type DetailRow struct {
	ID                      int64     `json:"id"`
	UserID                  int64     `json:"user_id"`
	Type                    string    `json:"type"`
	Amount                  float64   `json:"amount"`
	Description             string    `json:"description"`
	Status                  string    `json:"status"`
	ApprovedByOpHeadAmount  *float64  `json:"approved_by_op_head_amount"`
	ApprovedByFinanceAmount *float64  `json:"approved_by_finance_amount"`
	CreatedAt               time.Time `json:"created_at"`
}

func getDetails(db *sql.DB, u *auth.User, startDate, endDate string) ([]DetailRow, error) {
	where, args := buildWhereClause(u, startDate, endDate)

	query := `
		SELECT 
			id, user_id, type, amount, description, status,
			approved_by_op_amount, approved_by_finance_amount, request_date
		FROM reimbursements
		` + where + `
		ORDER BY request_date DESC
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []DetailRow
	for rows.Next() {
		var d DetailRow
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Type, &d.Amount, &d.Description, &d.Status,
			&d.ApprovedByOpHeadAmount, &d.ApprovedByFinanceAmount, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	if data == nil {
		data = []DetailRow{}
	}

	return data, nil
}
