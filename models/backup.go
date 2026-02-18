package models

import (
	"database/sql"

	"github.com/bardenit/Bard/db"
)

// BackupBill is the flat DB representation of a bill for backup/restore.
type BackupBill struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Amount     int    `json:"amount"`
	DueDate    string `json:"due_date"`
	Recurrence string `json:"recurrence"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// BackupIncome is the flat DB representation of an income item for backup/restore.
type BackupIncome struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Amount      int    `json:"amount"`
	DepositDate string `json:"deposit_date"`
	Recurrence  string `json:"recurrence"`
	IsPrimary   bool   `json:"is_primary"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// BackupCategory is the flat DB representation of a budget category.
type BackupCategory struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ParentID  *int   `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

// BackupBudget is the flat DB representation of a budget.
type BackupBudget struct {
	ID            int    `json:"id"`
	CategoryID    int    `json:"category_id"`
	Amount        int    `json:"amount"`
	Recurrence    string `json:"recurrence"`
	EffectiveDate string `json:"effective_date"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// BackupExpenditure is the flat DB representation of an expenditure.
type BackupExpenditure struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	CategoryID  int    `json:"category_id"`
	Date        string `json:"date"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// BackupRule is the flat DB representation of a transaction rule.
type BackupRule struct {
	ID         int    `json:"id"`
	Keyword    string `json:"keyword"`
	CategoryID int    `json:"category_id"`
	CreatedAt  string `json:"created_at"`
}

// BackupTransaction is the flat DB representation of an imported transaction.
type BackupTransaction struct {
	ID              int    `json:"id"`
	AccountName     string `json:"account_name"`
	ProcessedDate   string `json:"processed_date"`
	Description     string `json:"description"`
	CreditOrDebit   string `json:"credit_or_debit"`
	Amount          int    `json:"amount"`
	Balance         *int   `json:"balance"`
	CategoryID      *int   `json:"category_id"`
	AutoCategoryID  *int   `json:"auto_category_id"`
	IsDuplicateFlag bool   `json:"is_duplicate_flag"`
	IsConfirmed     bool   `json:"is_confirmed"`
	IsDismissed     bool   `json:"is_dismissed"`
	SourceFile      string `json:"source_file"`
	ImportedAt      string `json:"imported_at"`
}

// ExportAllBills returns all bills for backup.
func ExportAllBills() ([]BackupBill, error) {
	rows, err := db.DB.Query(`SELECT id,name,amount,due_date,recurrence,is_active,created_at,updated_at FROM bills ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupBill
	for rows.Next() {
		var b BackupBill
		if err := rows.Scan(&b.ID, &b.Name, &b.Amount, &b.DueDate, &b.Recurrence, &b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

// ExportAllIncome returns all income items for backup.
func ExportAllIncome() ([]BackupIncome, error) {
	rows, err := db.DB.Query(`SELECT id,name,amount,deposit_date,recurrence,is_primary,is_active,created_at,updated_at FROM income ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupIncome
	for rows.Next() {
		var i BackupIncome
		if err := rows.Scan(&i.ID, &i.Name, &i.Amount, &i.DepositDate, &i.Recurrence, &i.IsPrimary, &i.IsActive, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ExportAllCategories returns all budget categories (flat) for backup.
func ExportAllCategories() ([]BackupCategory, error) {
	rows, err := db.DB.Query(`SELECT id,name,parent_id,sort_order,created_at FROM budget_categories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupCategory
	for rows.Next() {
		var c BackupCategory
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &parentID, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			c.ParentID = &pid
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// ExportAllBudgets returns all budgets (including inactive) for backup.
func ExportAllBudgets() ([]BackupBudget, error) {
	rows, err := db.DB.Query(`SELECT id,category_id,amount,recurrence,effective_date,is_active,created_at,updated_at FROM budgets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupBudget
	for rows.Next() {
		var b BackupBudget
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Amount, &b.Recurrence, &b.EffectiveDate, &b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

// ExportAllExpenditures returns all expenditures for backup.
func ExportAllExpenditures() ([]BackupExpenditure, error) {
	rows, err := db.DB.Query(`SELECT id,description,amount,category_id,date,created_at,updated_at FROM expenditures ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupExpenditure
	for rows.Next() {
		var e BackupExpenditure
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.CategoryID, &e.Date, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

// ExportAllRules returns all transaction rules for backup.
func ExportAllRules() ([]BackupRule, error) {
	rows, err := db.DB.Query(`SELECT id,keyword,category_id,created_at FROM transaction_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupRule
	for rows.Next() {
		var r BackupRule
		if err := rows.Scan(&r.ID, &r.Keyword, &r.CategoryID, &r.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ExportAllTransactions returns all imported transactions for backup.
func ExportAllTransactions() ([]BackupTransaction, error) {
	rows, err := db.DB.Query(`SELECT id,account_name,processed_date,description,credit_or_debit,amount,balance,category_id,auto_category_id,is_duplicate_flag,is_confirmed,is_dismissed,source_file,imported_at FROM imported_transactions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BackupTransaction
	for rows.Next() {
		var t BackupTransaction
		var balance, catID, autoCatID sql.NullInt64
		if err := rows.Scan(&t.ID, &t.AccountName, &t.ProcessedDate, &t.Description, &t.CreditOrDebit, &t.Amount, &balance, &catID, &autoCatID, &t.IsDuplicateFlag, &t.IsConfirmed, &t.IsDismissed, &t.SourceFile, &t.ImportedAt); err != nil {
			return nil, err
		}
		if balance.Valid {
			v := int(balance.Int64)
			t.Balance = &v
		}
		if catID.Valid {
			v := int(catID.Int64)
			t.CategoryID = &v
		}
		if autoCatID.Valid {
			v := int(autoCatID.Int64)
			t.AutoCategoryID = &v
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// ExportAllSettings returns all settings as a map for backup.
func ExportAllSettings() (map[string]string, error) {
	rows, err := db.DB.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// RestoreData replaces all database content with the provided backup data.
// Runs in a single transaction. If includeTxns is false, imported_transactions
// are not restored.
func RestoreData(
	cats []BackupCategory,
	budgets []BackupBudget,
	bills []BackupBill,
	income []BackupIncome,
	exps []BackupExpenditure,
	rules []BackupRule,
	txns []BackupTransaction,
	settings map[string]string,
	includeTxns bool,
) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	// Delete in reverse-dependency order.
	deletes := []string{
		`DELETE FROM imported_transactions`,
		`DELETE FROM transaction_rules`,
		`DELETE FROM expenditures`,
		`DELETE FROM budgets`,
		// Delete child categories before parents to avoid FK cascade surprises.
		`DELETE FROM budget_categories WHERE parent_id IS NOT NULL`,
		`DELETE FROM budget_categories`,
		`DELETE FROM bills`,
		`DELETE FROM income`,
		`DELETE FROM settings`,
	}
	for _, d := range deletes {
		if _, err := tx.Exec(d); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Reset SQLite autoincrement sequences.
	seqTables := []string{"bills", "income", "budget_categories", "budgets", "expenditures", "transaction_rules", "imported_transactions"}
	for _, tbl := range seqTables {
		tx.Exec(`DELETE FROM sqlite_sequence WHERE name = ?`, tbl)
	}

	// Insert categories — parents (NULL parent_id) first, then children.
	for _, c := range cats {
		if c.ParentID == nil {
			if _, err := tx.Exec(`INSERT INTO budget_categories (id,name,parent_id,sort_order,created_at) VALUES (?,?,NULL,?,?)`,
				c.ID, c.Name, c.SortOrder, c.CreatedAt); err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	for _, c := range cats {
		if c.ParentID != nil {
			if _, err := tx.Exec(`INSERT INTO budget_categories (id,name,parent_id,sort_order,created_at) VALUES (?,?,?,?,?)`,
				c.ID, c.Name, *c.ParentID, c.SortOrder, c.CreatedAt); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	for _, b := range budgets {
		active := 0
		if b.IsActive {
			active = 1
		}
		if _, err := tx.Exec(`INSERT INTO budgets (id,category_id,amount,recurrence,effective_date,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
			b.ID, b.CategoryID, b.Amount, b.Recurrence, b.EffectiveDate, active, b.CreatedAt, b.UpdatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, b := range bills {
		active := 0
		if b.IsActive {
			active = 1
		}
		if _, err := tx.Exec(`INSERT INTO bills (id,name,amount,due_date,recurrence,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
			b.ID, b.Name, b.Amount, b.DueDate, b.Recurrence, active, b.CreatedAt, b.UpdatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, i := range income {
		active := 0
		if i.IsActive {
			active = 1
		}
		primary := 0
		if i.IsPrimary {
			primary = 1
		}
		if _, err := tx.Exec(`INSERT INTO income (id,name,amount,deposit_date,recurrence,is_primary,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
			i.ID, i.Name, i.Amount, i.DepositDate, i.Recurrence, primary, active, i.CreatedAt, i.UpdatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, e := range exps {
		if _, err := tx.Exec(`INSERT INTO expenditures (id,description,amount,category_id,date,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
			e.ID, e.Description, e.Amount, e.CategoryID, e.Date, e.CreatedAt, e.UpdatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, r := range rules {
		if _, err := tx.Exec(`INSERT INTO transaction_rules (id,keyword,category_id,created_at) VALUES (?,?,?,?)`,
			r.ID, r.Keyword, r.CategoryID, r.CreatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}

	if includeTxns {
		for _, t := range txns {
			var balVal, catVal, autoCatVal interface{}
			if t.Balance != nil {
				balVal = *t.Balance
			}
			if t.CategoryID != nil {
				catVal = *t.CategoryID
			}
			if t.AutoCategoryID != nil {
				autoCatVal = *t.AutoCategoryID
			}
			dup, conf, dism := 0, 0, 0
			if t.IsDuplicateFlag {
				dup = 1
			}
			if t.IsConfirmed {
				conf = 1
			}
			if t.IsDismissed {
				dism = 1
			}
			if _, err := tx.Exec(`INSERT INTO imported_transactions (id,account_name,processed_date,description,credit_or_debit,amount,balance,category_id,auto_category_id,is_duplicate_flag,is_confirmed,is_dismissed,source_file,imported_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				t.ID, t.AccountName, t.ProcessedDate, t.Description, t.CreditOrDebit, t.Amount, balVal, catVal, autoCatVal, dup, conf, dism, t.SourceFile, t.ImportedAt); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	for k, v := range settings {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES (?,?)`, k, v); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
