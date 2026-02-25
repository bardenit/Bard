package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bardenit/Bard/db"
)

type Bill struct {
	ID         int
	Name       string
	Amount     int
	DueDate    string
	Recurrence string
	IsActive   bool
	CreatedAt  string
	UpdatedAt  string
}

func (b Bill) DueDateFormatted() string {
	t, err := time.Parse("2006-01-02", b.DueDate)
	if err != nil {
		return b.DueDate
	}
	return t.Format("Jan 2, 2006")
}

func ListBills() ([]Bill, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, amount, due_date, recurrence, is_active, created_at, updated_at
		FROM bills ORDER BY due_date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bills []Bill
	for rows.Next() {
		var b Bill
		if err := rows.Scan(&b.ID, &b.Name, &b.Amount, &b.DueDate, &b.Recurrence, &b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bills = append(bills, b)
	}
	return bills, rows.Err()
}

func ListActiveBills() ([]Bill, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, amount, due_date, recurrence, is_active, created_at, updated_at
		FROM bills WHERE is_active = 1 ORDER BY due_date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bills []Bill
	for rows.Next() {
		var b Bill
		if err := rows.Scan(&b.ID, &b.Name, &b.Amount, &b.DueDate, &b.Recurrence, &b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bills = append(bills, b)
	}
	return bills, rows.Err()
}

func GetBill(id int) (Bill, error) {
	var b Bill
	err := db.DB.QueryRow(`
		SELECT id, name, amount, due_date, recurrence, is_active, created_at, updated_at
		FROM bills WHERE id = ?
	`, id).Scan(&b.ID, &b.Name, &b.Amount, &b.DueDate, &b.Recurrence, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return b, fmt.Errorf("bill not found")
	}
	return b, err
}

func CreateBill(name string, amount int, dueDate, recurrence string) error {
	_, err := db.DB.Exec(`
		INSERT INTO bills (name, amount, due_date, recurrence) VALUES (?, ?, ?, ?)
	`, name, amount, dueDate, recurrence)
	return err
}

func UpdateBill(id int, name string, amount int, dueDate, recurrence string, isActive bool) error {
	active := 0
	if isActive {
		active = 1
	}
	_, err := db.DB.Exec(`
		UPDATE bills SET name = ?, amount = ?, due_date = ?, recurrence = ?, is_active = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, amount, dueDate, recurrence, active, id)
	return err
}

func DeleteBill(id int) error {
	_, err := db.DB.Exec(`DELETE FROM bills WHERE id = ?`, id)
	return err
}

// BillPayment records the actual amount paid for a specific bill occurrence.
type BillPayment struct {
	ID             int
	BillID         int
	OccurrenceDate string
	AmountPaid     int
	Notes          string
	CreatedAt      string
}

// RecordBillPayment upserts a payment for a given bill occurrence date.
func RecordBillPayment(billID int, occurrenceDate string, amountPaid int, notes string) error {
	_, err := db.DB.Exec(`
		INSERT INTO bill_payments (bill_id, occurrence_date, amount_paid, notes)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(bill_id, occurrence_date) DO UPDATE SET amount_paid = excluded.amount_paid, notes = excluded.notes
	`, billID, occurrenceDate, amountPaid, notes)
	return err
}

// GetBillPayment retrieves the payment record for a specific bill and occurrence date.
func GetBillPayment(billID int, occurrenceDate string) (BillPayment, bool, error) {
	var p BillPayment
	err := db.DB.QueryRow(`
		SELECT id, bill_id, occurrence_date, amount_paid, notes, created_at
		FROM bill_payments WHERE bill_id = ? AND occurrence_date = ?
	`, billID, occurrenceDate).Scan(&p.ID, &p.BillID, &p.OccurrenceDate, &p.AmountPaid, &p.Notes, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return p, false, nil
	}
	return p, true, err
}

// GetBillPaymentsInRange loads all payments within a date range as a map keyed by "billID-date".
func GetBillPaymentsInRange(start, end string) (map[string]BillPayment, error) {
	rows, err := db.DB.Query(`
		SELECT id, bill_id, occurrence_date, amount_paid, notes, created_at
		FROM bill_payments WHERE occurrence_date >= ? AND occurrence_date <= ?
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make(map[string]BillPayment)
	for rows.Next() {
		var p BillPayment
		if err := rows.Scan(&p.ID, &p.BillID, &p.OccurrenceDate, &p.AmountPaid, &p.Notes, &p.CreatedAt); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%d-%s", p.BillID, p.OccurrenceDate)
		payments[key] = p
	}
	return payments, rows.Err()
}
