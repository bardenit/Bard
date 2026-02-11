package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bardenit/Bard/db"
)

type Expenditure struct {
	ID           int
	Description  string
	Amount       int
	CategoryID   int
	CategoryName string
	Date         string
	CreatedAt    string
	UpdatedAt    string
}

func (e Expenditure) DateFormatted() string {
	t, err := time.Parse("2006-01-02", e.Date)
	if err != nil {
		return e.Date
	}
	return t.Format("Jan 2, 2006")
}

func ListExpenditures() ([]Expenditure, error) {
	rows, err := db.DB.Query(`
		SELECT e.id, e.description, e.amount, e.category_id, bc.name, e.date, e.created_at, e.updated_at
		FROM expenditures e
		JOIN budget_categories bc ON bc.id = e.category_id
		ORDER BY e.date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Expenditure
	for rows.Next() {
		var e Expenditure
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.CategoryID, &e.CategoryName, &e.Date, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func GetExpenditure(id int) (Expenditure, error) {
	var e Expenditure
	err := db.DB.QueryRow(`
		SELECT e.id, e.description, e.amount, e.category_id, bc.name, e.date, e.created_at, e.updated_at
		FROM expenditures e
		JOIN budget_categories bc ON bc.id = e.category_id
		WHERE e.id = ?
	`, id).Scan(&e.ID, &e.Description, &e.Amount, &e.CategoryID, &e.CategoryName, &e.Date, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return e, fmt.Errorf("expenditure not found")
	}
	return e, err
}

func CreateExpenditure(description string, amount, categoryID int, date string) error {
	_, err := db.DB.Exec(`
		INSERT INTO expenditures (description, amount, category_id, date) VALUES (?, ?, ?, ?)
	`, description, amount, categoryID, date)
	return err
}

func UpdateExpenditure(id int, description string, amount, categoryID int, date string) error {
	_, err := db.DB.Exec(`
		UPDATE expenditures SET description = ?, amount = ?, category_id = ?, date = ?, updated_at = datetime('now')
		WHERE id = ?
	`, description, amount, categoryID, date, id)
	return err
}

func DeleteExpenditure(id int) error {
	_, err := db.DB.Exec(`DELETE FROM expenditures WHERE id = ?`, id)
	return err
}

// GetAllCategorySpending returns a map of category_id -> total cents spent for a given month
func GetAllCategorySpending(year int, month time.Month) (map[int]int, error) {
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-01", year, month+1)
	if month == 12 {
		endDate = fmt.Sprintf("%04d-01-01", year+1)
	}

	rows, err := db.DB.Query(`
		SELECT category_id, SUM(amount)
		FROM expenditures
		WHERE date >= ? AND date < ?
		GROUP BY category_id
	`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spending := make(map[int]int)
	for rows.Next() {
		var catID, total int
		if err := rows.Scan(&catID, &total); err != nil {
			return nil, err
		}
		spending[catID] = total
	}
	return spending, rows.Err()
}
