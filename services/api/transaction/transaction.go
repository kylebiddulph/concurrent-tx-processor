package transaction

import (
	"database/sql"
	"time"
)

type Transaction struct {
	ID          string
	Amount      float64
	Status      string
	DateUpdated time.Time
	DateCreated time.Time
}

func UpdateStatus(db *sql.DB, txID string, status string) error {
	_, err := db.Exec(`
		UPDATE transactions
		SET status = ?
		where transaction_id = ?`,
		status, txID)
	return err
}

func CreateTransaction(db *sql.DB, txID string, amount float64, status string) error {
	_, err := db.Exec(`
	INSERT INTO transactions(transaction_id, amount, status)
	VALUES(?, ?, ?)`,
		txID, amount, status)
	return err
}
