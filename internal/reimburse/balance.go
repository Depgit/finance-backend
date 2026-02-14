package reimburse

import (
	"database/sql"
)

// UserBalance represents the advance balance for a user.
type UserBalance struct {
	TotalAdvance   float64 `json:"total_advance"`   // Total advance approved by Finance
	ProofSubmitted float64 `json:"proof_submitted"` // Total expense proofs submitted
	Outstanding    float64 `json:"outstanding"`     // Remaining balance to submit proof for
}

// GetUserBalance calculates the advance balance for a user.
// Total Advance = Sum of ADVANCE requests approved by Finance (approved_by_finance_amount)
// Proof Submitted = Sum of EXPENSE requests approved by Finance (approved_by_finance_amount)
// Outstanding = Total Advance - Proof Submitted
func GetUserBalance(db *sql.DB, userID int64) (*UserBalance, error) {
	balance := &UserBalance{}

	// Calculate total advance (ONLY Finance-approved advances)
	err := db.QueryRow(`
		SELECT COALESCE(SUM(approved_by_finance_amount), 0) 
		FROM reimbursements 
		WHERE user_id = $1 AND type = 'ADVANCE' AND status = 'APPROVED_BY_FINANCE'
	`, userID).Scan(&balance.TotalAdvance)
	if err != nil {
		return nil, err
	}

	// Calculate proof submitted (ONLY Finance-approved expenses)
	// This ensures only approved expense amounts count toward reducing the advance
	err = db.QueryRow(`
		SELECT COALESCE(SUM(approved_by_finance_amount), 0) 
		FROM reimbursements 
		WHERE user_id = $1 AND type = 'EXPENSE' AND status = 'APPROVED_BY_FINANCE'
	`, userID).Scan(&balance.ProofSubmitted)
	if err != nil {
		return nil, err
	}

	balance.Outstanding = balance.TotalAdvance - balance.ProofSubmitted
	if balance.Outstanding < 0 {
		balance.Outstanding = 0
	}

	return balance, nil
}
