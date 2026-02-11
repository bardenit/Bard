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
