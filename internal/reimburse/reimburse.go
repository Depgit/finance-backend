package reimburse

import (
	"database/sql"
	"time"
)

const (
	TypeExpense = "EXPENSE"
	TypeAdvance = "ADVANCE"

	StatusPending           = "PENDING"
	StatusApprovedByOp      = "APPROVED_BY_OP"
	StatusApprovedByFinance = "APPROVED_BY_FINANCE"
	StatusSettled           = "SETTLED" // For advances that returned
	StatusRejected          = "REJECTED"
)

type Reimbursement struct {
	ID                      int64     `json:"id"`
	UserID                  int64     `json:"user_id"`
	UserEmail               string    `json:"user_email,omitempty"` // Joined field
	Type                    string    `json:"type"`
	Amount                  float64   `json:"amount"`                                // Requested amount
	ApprovedByOpAmount      float64   `json:"approved_by_op_amount"`                 // Amount Op Head approved
	ApprovedByFinanceAmount float64   `json:"approved_by_finance_amount"`            // Final amount Finance approved
	ApprovedByOpUserID      *int64    `json:"approved_by_op_user_id,omitempty"`      // Who approved at Op level
	ApprovedByFinanceUserID *int64    `json:"approved_by_finance_user_id,omitempty"` // Who approved at Finance level
	Description             string    `json:"description"`
	Proof                   string    `json:"proof"`         // URL or text note
	ProofFileID             *int64    `json:"proof_file_id"` // Reference to uploaded file
	Status                  string    `json:"status"`
	Property                string    `json:"property"` // New field
	RequestDate             time.Time `json:"request_date"`
}

func Migrate(db *sql.DB) error {
	q := `CREATE TABLE IF NOT EXISTS reimbursements (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		approved_by_op_amount DOUBLE PRECISION DEFAULT 0,
		approved_by_finance_amount DOUBLE PRECISION DEFAULT 0,
		approved_by_op_user_id INTEGER,
		approved_by_finance_user_id INTEGER,
		description TEXT,
		proof TEXT,
		proof_file_id INTEGER,
		status TEXT NOT NULL,
		property TEXT,
		request_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(approved_by_op_user_id) REFERENCES users(id),
		FOREIGN KEY(approved_by_finance_user_id) REFERENCES users(id),
		FOREIGN KEY(approved_by_finance_user_id) REFERENCES users(id)
	);`
	_, err := db.Exec(q)
	if err != nil {
		return err
	}

	// Add columns to existing table if they don't exist (for migration)
	alterQueries := []string{
		`ALTER TABLE reimbursements ADD COLUMN IF NOT EXISTS approved_by_op_amount DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE reimbursements ADD COLUMN IF NOT EXISTS approved_by_finance_amount DOUBLE PRECISION DEFAULT 0`,
		`ALTER TABLE reimbursements ADD COLUMN IF NOT EXISTS approved_by_op_user_id INTEGER`,
		`ALTER TABLE reimbursements ADD COLUMN IF NOT EXISTS approved_by_finance_user_id INTEGER`,
		`ALTER TABLE reimbursements ADD COLUMN IF NOT EXISTS proof_file_id INTEGER`,
		`ALTER TABLE reimbursements ADD COLUMN IF NOT EXISTS property TEXT`,
	}

	for _, q := range alterQueries {
		// Ignore errors if column already exists (Postgres supports IF NOT EXISTS in newer versions, but if older, we might need a different approach.
		// However, standard postgres usually errors if column exists without IF NOT EXISTS.
		// Let's assume Postgres 9.6+ which supports IF NOT EXISTS.
		// If fails, we ignore as before.
		_, _ = db.Exec(q)
	}

	return nil
}

func Create(db *sql.DB, userID int64, rType string, amount float64, desc, proof string, proofFileID *int64, property string) (*Reimbursement, error) {
	var id int64
	q := `INSERT INTO reimbursements (user_id, type, amount, description, proof, proof_file_id, status, property) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err := db.QueryRow(q, userID, rType, amount, desc, proof, proofFileID, StatusPending, property).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &Reimbursement{
		ID:          id,
		UserID:      userID,
		Type:        rType,
		Amount:      amount,
		Description: desc,
		Proof:       proof,
		ProofFileID: proofFileID,
		Status:      StatusPending,
		Property:    property,
		RequestDate: time.Now(),
	}, nil
}

// GetByID returns a reimbursement by ID.
func GetByID(db *sql.DB, id int64) (*Reimbursement, error) {
	r := &Reimbursement{}
	err := db.QueryRow(`SELECT id, user_id, type, amount, approved_by_op_amount, approved_by_finance_amount, 
		approved_by_op_user_id, approved_by_finance_user_id, description, proof, proof_file_id, status, property, request_date 
		FROM reimbursements WHERE id = $1`, id).
		Scan(&r.ID, &r.UserID, &r.Type, &r.Amount, &r.ApprovedByOpAmount, &r.ApprovedByFinanceAmount,
			&r.ApprovedByOpUserID, &r.ApprovedByFinanceUserID, &r.Description, &r.Proof, &r.ProofFileID, &r.Status, &r.Property, &r.RequestDate)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UpdateStatus updates the status of a reimbursement.
func UpdateStatus(db *sql.DB, id int64, status string) error {
	_, err := db.Exec(`UPDATE reimbursements SET status = $1 WHERE id = $2`, status, id)
	return err
}

// ApproveByOp updates status and records Op Head approval amount.
func ApproveByOp(db *sql.DB, id int64, amount float64, approverID int64) error {
	_, err := db.Exec(`UPDATE reimbursements SET status = $1, approved_by_op_amount = $2, approved_by_op_user_id = $3 WHERE id = $4`,
		StatusApprovedByOp, amount, approverID, id)
	return err
}

// ApproveByFinance updates status and records Finance approval amount (final).
func ApproveByFinance(db *sql.DB, id int64, amount float64, approverID int64) error {
	_, err := db.Exec(`UPDATE reimbursements SET status = $1, approved_by_finance_amount = $2, approved_by_finance_user_id = $3 WHERE id = $4`,
		StatusApprovedByFinance, amount, approverID, id)
	return err
}

// ListByUserID returns all requests for a user.
func ListByUserID(db *sql.DB, userID int64) ([]*Reimbursement, error) {
	rows, err := db.Query(`SELECT id, user_id, type, amount, approved_by_op_amount, approved_by_finance_amount, 
		approved_by_op_user_id, approved_by_finance_user_id, description, proof, proof_file_id, status, property, request_date 
		FROM reimbursements WHERE user_id = $1 ORDER BY request_date DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// ListAll returns all requests (useful for Admin/Director).
func ListAll(db *sql.DB) ([]*Reimbursement, error) {
	q := `SELECT r.id, r.user_id, u.email, r.type, r.amount, r.approved_by_op_amount, r.approved_by_finance_amount,
		r.approved_by_op_user_id, r.approved_by_finance_user_id, r.description, r.proof, r.proof_file_id, r.status, r.property, r.request_date 
	      FROM reimbursements r JOIN users u ON r.user_id = u.id ORDER BY r.request_date DESC`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowsWithEmail(rows)
}

// ListPendingForOpHead returns requests pending OP Head approval (status=PENDING).
func ListPendingForOpHead(db *sql.DB) ([]*Reimbursement, error) {
	q := `SELECT r.id, r.user_id, u.email, r.type, r.amount, r.approved_by_op_amount, r.approved_by_finance_amount,
		r.approved_by_op_user_id, r.approved_by_finance_user_id, r.description, r.proof, r.proof_file_id, r.status, r.property, r.request_date 
	      FROM reimbursements r JOIN users u ON r.user_id = u.id 
	      WHERE r.status = 'PENDING' ORDER BY r.request_date ASC`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowsWithEmail(rows)
}

// ListPendingForFinance returns requests approved by Op Head (status=APPROVED_BY_OP).
func ListPendingForFinance(db *sql.DB) ([]*Reimbursement, error) {
	q := `SELECT r.id, r.user_id, u.email, r.type, r.amount, r.approved_by_op_amount, r.approved_by_finance_amount,
		r.approved_by_op_user_id, r.approved_by_finance_user_id, r.description, r.proof, r.proof_file_id, r.status, r.property, r.request_date 
	      FROM reimbursements r JOIN users u ON r.user_id = u.id 
	      WHERE r.status = 'APPROVED_BY_OP' ORDER BY r.request_date ASC`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowsWithEmail(rows)
}

func scanRows(rows *sql.Rows) ([]*Reimbursement, error) {
	var res []*Reimbursement
	for rows.Next() {
		r := &Reimbursement{}
		if err := rows.Scan(&r.ID, &r.UserID, &r.Type, &r.Amount, &r.ApprovedByOpAmount, &r.ApprovedByFinanceAmount,
			&r.ApprovedByOpUserID, &r.ApprovedByFinanceUserID, &r.Description, &r.Proof, &r.ProofFileID, &r.Status, &r.Property, &r.RequestDate); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func scanRowsWithEmail(rows *sql.Rows) ([]*Reimbursement, error) {
	var res []*Reimbursement
	for rows.Next() {
		r := &Reimbursement{}
		if err := rows.Scan(&r.ID, &r.UserID, &r.UserEmail, &r.Type, &r.Amount, &r.ApprovedByOpAmount, &r.ApprovedByFinanceAmount,
			&r.ApprovedByOpUserID, &r.ApprovedByFinanceUserID, &r.Description, &r.Proof, &r.ProofFileID, &r.Status, &r.Property, &r.RequestDate); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
