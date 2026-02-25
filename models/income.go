package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bardenit/Bard/db"
)

type Income struct {
	ID          int
	Name        string
	Amount      int
	DepositDate string
	Recurrence  string
	IsPrimary   bool
	IsActive    bool
	CreatedAt   string
	UpdatedAt   string
}

func (i Income) DepositDateFormatted() string {
	t, err := time.Parse("2006-01-02", i.DepositDate)
	if err != nil {
		return i.DepositDate
	}
	return t.Format("Jan 2, 2006")
}

func ListIncome() ([]Income, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, amount, deposit_date, recurrence, is_primary, is_active, created_at, updated_at
		FROM income ORDER BY deposit_date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Income
	for rows.Next() {
		var i Income
		if err := rows.Scan(&i.ID, &i.Name, &i.Amount, &i.DepositDate, &i.Recurrence, &i.IsPrimary, &i.IsActive, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func ListActiveIncome() ([]Income, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, amount, deposit_date, recurrence, is_primary, is_active, created_at, updated_at
		FROM income WHERE is_active = 1 ORDER BY deposit_date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Income
	for rows.Next() {
		var i Income
		if err := rows.Scan(&i.ID, &i.Name, &i.Amount, &i.DepositDate, &i.Recurrence, &i.IsPrimary, &i.IsActive, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func GetIncome(id int) (Income, error) {
	var i Income
	err := db.DB.QueryRow(`
		SELECT id, name, amount, deposit_date, recurrence, is_primary, is_active, created_at, updated_at
		FROM income WHERE id = ?
	`, id).Scan(&i.ID, &i.Name, &i.Amount, &i.DepositDate, &i.Recurrence, &i.IsPrimary, &i.IsActive, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return i, fmt.Errorf("income not found")
	}
	return i, err
}

func CreateIncome(name string, amount int, depositDate, recurrence string) error {
	_, err := db.DB.Exec(`
		INSERT INTO income (name, amount, deposit_date, recurrence) VALUES (?, ?, ?, ?)
	`, name, amount, depositDate, recurrence)
	return err
}

func UpdateIncome(id int, name string, amount int, depositDate, recurrence string, isActive bool) error {
	active := 0
	if isActive {
		active = 1
	}
	_, err := db.DB.Exec(`
		UPDATE income SET name = ?, amount = ?, deposit_date = ?, recurrence = ?, is_active = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, amount, depositDate, recurrence, active, id)
	return err
}

func DeleteIncome(id int) error {
	_, err := db.DB.Exec(`DELETE FROM income WHERE id = ?`, id)
	return err
}

// IncomeActual records the actual amount received for a specific income occurrence.
type IncomeActual struct {
	ID             int
	IncomeID       int
	OccurrenceDate string
	AmountActual   int
	Notes          string
	CreatedAt      string
}

// RecordIncomeActual upserts an actual amount for a given income occurrence date.
func RecordIncomeActual(incomeID int, occurrenceDate string, amountActual int, notes string) error {
	_, err := db.DB.Exec(`
		INSERT INTO income_actuals (income_id, occurrence_date, amount_actual, notes)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(income_id, occurrence_date) DO UPDATE SET amount_actual = excluded.amount_actual, notes = excluded.notes
	`, incomeID, occurrenceDate, amountActual, notes)
	return err
}

// GetIncomeActual retrieves the recorded actual for a specific income and occurrence date.
func GetIncomeActual(incomeID int, occurrenceDate string) (IncomeActual, bool, error) {
	var a IncomeActual
	err := db.DB.QueryRow(`
		SELECT id, income_id, occurrence_date, amount_actual, notes, created_at
		FROM income_actuals WHERE income_id = ? AND occurrence_date = ?
	`, incomeID, occurrenceDate).Scan(&a.ID, &a.IncomeID, &a.OccurrenceDate, &a.AmountActual, &a.Notes, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return a, false, nil
	}
	return a, true, err
}

// GetIncomeActualsInRange loads all actuals within a date range as a map keyed by "incomeID-date".
func GetIncomeActualsInRange(start, end string) (map[string]IncomeActual, error) {
	rows, err := db.DB.Query(`
		SELECT id, income_id, occurrence_date, amount_actual, notes, created_at
		FROM income_actuals WHERE occurrence_date >= ? AND occurrence_date <= ?
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actuals := make(map[string]IncomeActual)
	for rows.Next() {
		var a IncomeActual
		if err := rows.Scan(&a.ID, &a.IncomeID, &a.OccurrenceDate, &a.AmountActual, &a.Notes, &a.CreatedAt); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%d-%s", a.IncomeID, a.OccurrenceDate)
		actuals[key] = a
	}
	return actuals, rows.Err()
}

// SetPrimaryIncome marks a single income item as the primary paycheck source.
// All other income items have is_primary cleared in the same transaction.
func SetPrimaryIncome(id int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE income SET is_primary = 0`); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`UPDATE income SET is_primary = 1 WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
