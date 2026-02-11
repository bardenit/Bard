package models

import (
	"database/sql"
	"fmt"

	"github.com/bardenit/Bard/db"
)

type BudgetCategory struct {
	ID        int
	Name      string
	ParentID  *int
	SortOrder int
	CreatedAt string
	Children  []BudgetCategory
	Budget    *Budget
}

type Budget struct {
	ID            int
	CategoryID    int
	Amount        int
	Recurrence    string
	EffectiveDate string
	IsActive      bool
	CreatedAt     string
	UpdatedAt     string
}

// MonthlyAmount normalizes the budget amount to a monthly equivalent
func (b Budget) MonthlyAmount() int {
	switch b.Recurrence {
	case "weekly":
		return int(float64(b.Amount) * 52.0 / 12.0)
	case "biweekly":
		return int(float64(b.Amount) * 26.0 / 12.0)
	case "monthly":
		return b.Amount
	}
	return b.Amount
}

func ListCategories() ([]BudgetCategory, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, parent_id, sort_order, created_at
		FROM budget_categories ORDER BY sort_order ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []BudgetCategory
	for rows.Next() {
		var c BudgetCategory
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &parentID, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			c.ParentID = &pid
		}
		all = append(all, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return all, nil
}

// ListCategoryTree returns top-level categories with children nested
func ListCategoryTree() ([]BudgetCategory, error) {
	all, err := ListCategories()
	if err != nil {
		return nil, err
	}

	// Load active budgets for each category
	budgets, err := listActiveBudgets()
	if err != nil {
		return nil, err
	}
	budgetMap := make(map[int]*Budget)
	for i := range budgets {
		budgetMap[budgets[i].CategoryID] = &budgets[i]
	}

	// Build tree
	var topLevel []BudgetCategory
	childMap := make(map[int][]BudgetCategory)

	for _, c := range all {
		c.Budget = budgetMap[c.ID]
		if c.ParentID == nil {
			topLevel = append(topLevel, c)
		} else {
			childMap[*c.ParentID] = append(childMap[*c.ParentID], c)
		}
	}

	for i := range topLevel {
		topLevel[i].Children = childMap[topLevel[i].ID]
	}

	return topLevel, nil
}

func ListTopLevelCategories() ([]BudgetCategory, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, parent_id, sort_order, created_at
		FROM budget_categories WHERE parent_id IS NULL ORDER BY sort_order ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []BudgetCategory
	for rows.Next() {
		var c BudgetCategory
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &parentID, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func GetCategory(id int) (BudgetCategory, error) {
	var c BudgetCategory
	var parentID sql.NullInt64
	err := db.DB.QueryRow(`
		SELECT id, name, parent_id, sort_order, created_at
		FROM budget_categories WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &parentID, &c.SortOrder, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return c, fmt.Errorf("category not found")
	}
	if parentID.Valid {
		pid := int(parentID.Int64)
		c.ParentID = &pid
	}
	return c, err
}

func CreateCategory(name string, parentID *int) error {
	if parentID != nil {
		_, err := db.DB.Exec(`
			INSERT INTO budget_categories (name, parent_id) VALUES (?, ?)
		`, name, *parentID)
		return err
	}
	_, err := db.DB.Exec(`
		INSERT INTO budget_categories (name) VALUES (?)
	`, name)
	return err
}

func UpdateCategory(id int, name string, parentID *int) error {
	if parentID != nil {
		_, err := db.DB.Exec(`
			UPDATE budget_categories SET name = ?, parent_id = ? WHERE id = ?
		`, name, *parentID, id)
		return err
	}
	_, err := db.DB.Exec(`
		UPDATE budget_categories SET name = ?, parent_id = NULL WHERE id = ?
	`, name, id)
	return err
}

func DeleteCategory(id int) error {
	_, err := db.DB.Exec(`DELETE FROM budget_categories WHERE id = ?`, id)
	return err
}

// Budget operations

func listActiveBudgets() ([]Budget, error) {
	rows, err := db.DB.Query(`
		SELECT id, category_id, amount, recurrence, effective_date, is_active, created_at, updated_at
		FROM budgets WHERE is_active = 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Amount, &b.Recurrence, &b.EffectiveDate, &b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

func ListActiveBudgets() ([]Budget, error) {
	return listActiveBudgets()
}

func GetBudgetForCategory(categoryID int) (Budget, error) {
	var b Budget
	err := db.DB.QueryRow(`
		SELECT id, category_id, amount, recurrence, effective_date, is_active, created_at, updated_at
		FROM budgets WHERE category_id = ? AND is_active = 1
		ORDER BY effective_date DESC LIMIT 1
	`, categoryID).Scan(&b.ID, &b.CategoryID, &b.Amount, &b.Recurrence, &b.EffectiveDate, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return b, fmt.Errorf("no active budget for category")
	}
	return b, err
}

func SetBudget(categoryID, amount int, recurrence, effectiveDate string) error {
	// Deactivate existing budget for this category
	_, err := db.DB.Exec(`
		UPDATE budgets SET is_active = 0, updated_at = datetime('now')
		WHERE category_id = ? AND is_active = 1
	`, categoryID)
	if err != nil {
		return err
	}

	// Create new budget
	_, err = db.DB.Exec(`
		INSERT INTO budgets (category_id, amount, recurrence, effective_date) VALUES (?, ?, ?, ?)
	`, categoryID, amount, recurrence, effectiveDate)
	return err
}

// TotalMonthlyBudgeted returns the sum of all active budgets normalized to monthly
func TotalMonthlyBudgeted() (int, error) {
	budgets, err := listActiveBudgets()
	if err != nil {
		return 0, err
	}

	total := 0
	for _, b := range budgets {
		total += b.MonthlyAmount()
	}
	return total, nil
}
