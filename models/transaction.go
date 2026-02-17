package models

import (
	"database/sql"
	"strings"

	"github.com/bardenit/Bard/db"
)

// ImportedTransaction represents a bank transaction imported from a CSV file.
type ImportedTransaction struct {
	ID              int
	AccountName     string
	ProcessedDate   string
	Description     string
	CreditOrDebit   string
	Amount          int
	Balance         *int
	CategoryID      *int
	AutoCategoryID  *int
	CategoryName    string
	AutoCatName     string
	IsDuplicateFlag bool
	IsConfirmed     bool
	IsDismissed     bool
	SourceFile      string
	ImportedAt      string
}

// AutoCatIDOrZero returns the AutoCategoryID value or 0 if nil (safe for template use).
func (t ImportedTransaction) AutoCatIDOrZero() int {
	if t.AutoCategoryID == nil {
		return 0
	}
	return *t.AutoCategoryID
}

// TransactionRule maps a keyword to a budget category for auto-categorization.
type TransactionRule struct {
	ID           int
	Keyword      string
	CategoryID   int
	CategoryName string
	CreatedAt    string
}

// ListPendingTransactions returns all unconfirmed, undismissed transactions.
func ListPendingTransactions() ([]ImportedTransaction, error) {
	rows, err := db.DB.Query(`
		SELECT
			t.id, t.account_name, t.processed_date, t.description, t.credit_or_debit,
			t.amount, t.balance, t.category_id, t.auto_category_id,
			COALESCE(bc1.name, ''), COALESCE(bc2.name, ''),
			t.is_duplicate_flag, t.is_confirmed, t.is_dismissed,
			t.source_file, t.imported_at
		FROM imported_transactions t
		LEFT JOIN budget_categories bc1 ON bc1.id = t.category_id
		LEFT JOIN budget_categories bc2 ON bc2.id = t.auto_category_id
		WHERE t.is_confirmed = 0 AND t.is_dismissed = 0
		ORDER BY t.processed_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ImportedTransaction
	for rows.Next() {
		var t ImportedTransaction
		var balance, catID, autoCatID sql.NullInt64
		if err := rows.Scan(
			&t.ID, &t.AccountName, &t.ProcessedDate, &t.Description, &t.CreditOrDebit,
			&t.Amount, &balance, &catID, &autoCatID,
			&t.CategoryName, &t.AutoCatName,
			&t.IsDuplicateFlag, &t.IsConfirmed, &t.IsDismissed,
			&t.SourceFile, &t.ImportedAt,
		); err != nil {
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

// ListConfirmedTransactions returns the most recently confirmed imported transactions.
func ListConfirmedTransactions(limit int) ([]ImportedTransaction, error) {
	rows, err := db.DB.Query(`
		SELECT
			t.id, t.account_name, t.processed_date, t.description, t.credit_or_debit,
			t.amount, t.balance, t.category_id, t.auto_category_id,
			COALESCE(bc1.name, ''), COALESCE(bc2.name, ''),
			t.is_duplicate_flag, t.is_confirmed, t.is_dismissed,
			t.source_file, t.imported_at
		FROM imported_transactions t
		LEFT JOIN budget_categories bc1 ON bc1.id = t.category_id
		LEFT JOIN budget_categories bc2 ON bc2.id = t.auto_category_id
		WHERE t.is_confirmed = 1
		ORDER BY t.imported_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ImportedTransaction
	for rows.Next() {
		var t ImportedTransaction
		var balance, catID, autoCatID sql.NullInt64
		if err := rows.Scan(
			&t.ID, &t.AccountName, &t.ProcessedDate, &t.Description, &t.CreditOrDebit,
			&t.Amount, &balance, &catID, &autoCatID,
			&t.CategoryName, &t.AutoCatName,
			&t.IsDuplicateFlag, &t.IsConfirmed, &t.IsDismissed,
			&t.SourceFile, &t.ImportedAt,
		); err != nil {
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

// GetTransaction returns a single imported transaction by ID.
func GetTransaction(id int) (ImportedTransaction, error) {
	var t ImportedTransaction
	var balance, catID, autoCatID sql.NullInt64
	err := db.DB.QueryRow(`
		SELECT
			t.id, t.account_name, t.processed_date, t.description, t.credit_or_debit,
			t.amount, t.balance, t.category_id, t.auto_category_id,
			COALESCE(bc1.name, ''), COALESCE(bc2.name, ''),
			t.is_duplicate_flag, t.is_confirmed, t.is_dismissed,
			t.source_file, t.imported_at
		FROM imported_transactions t
		LEFT JOIN budget_categories bc1 ON bc1.id = t.category_id
		LEFT JOIN budget_categories bc2 ON bc2.id = t.auto_category_id
		WHERE t.id = ?
	`, id).Scan(
		&t.ID, &t.AccountName, &t.ProcessedDate, &t.Description, &t.CreditOrDebit,
		&t.Amount, &balance, &catID, &autoCatID,
		&t.CategoryName, &t.AutoCatName,
		&t.IsDuplicateFlag, &t.IsConfirmed, &t.IsDismissed,
		&t.SourceFile, &t.ImportedAt,
	)
	if err == sql.ErrNoRows {
		return t, sql.ErrNoRows
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
	return t, err
}

// GetLatestBalance returns the account balance from the most recent imported transaction
// that has a balance value. Returns 0, false if none is available.
func GetLatestBalance() (int, bool, error) {
	var balance int
	err := db.DB.QueryRow(`
		SELECT balance FROM imported_transactions
		WHERE balance IS NOT NULL
		ORDER BY processed_date DESC, id DESC
		LIMIT 1
	`).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return balance, true, nil
}

// CreateImportedTransaction inserts a new imported transaction row.
func CreateImportedTransaction(accountName, processedDate, description, creditOrDebit string, amount int, balance *int, autoCatID *int, isDuplicate bool, sourceFile string) (int64, error) {
	dupFlag := 0
	if isDuplicate {
		dupFlag = 1
	}

	var balanceVal interface{}
	if balance != nil {
		balanceVal = *balance
	}

	var autoCatVal interface{}
	if autoCatID != nil {
		autoCatVal = *autoCatID
	}

	result, err := db.DB.Exec(`
		INSERT INTO imported_transactions
			(account_name, processed_date, description, credit_or_debit, amount, balance, auto_category_id, is_duplicate_flag, source_file)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, accountName, processedDate, description, creditOrDebit, amount, balanceVal, autoCatVal, dupFlag, sourceFile)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ConfirmTransaction creates an expenditure and marks the transaction as confirmed.
// Both operations run in a single DB transaction.
func ConfirmTransaction(id, categoryID int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	// Fetch the transaction for its description, amount, and date.
	var description, processedDate, creditOrDebit string
	var amount int
	if err = tx.QueryRow(`
		SELECT description, amount, processed_date, credit_or_debit
		FROM imported_transactions WHERE id = ?
	`, id).Scan(&description, &amount, &processedDate, &creditOrDebit); err != nil {
		tx.Rollback()
		return err
	}

	// Only create an expenditure for debits.
	if creditOrDebit == "Debit" {
		if _, err = tx.Exec(`
			INSERT INTO expenditures (description, amount, category_id, date)
			VALUES (?, ?, ?, ?)
		`, description, amount, categoryID, processedDate); err != nil {
			tx.Rollback()
			return err
		}
	}

	if _, err = tx.Exec(`
		UPDATE imported_transactions
		SET is_confirmed = 1, category_id = ?
		WHERE id = ?
	`, categoryID, id); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// DismissTransaction marks a transaction as dismissed.
func DismissTransaction(id int) error {
	_, err := db.DB.Exec(`UPDATE imported_transactions SET is_dismissed = 1 WHERE id = ?`, id)
	return err
}

// CheckDuplicate returns true if an expenditure with the same amount already exists.
func CheckDuplicate(amountCents int) (bool, error) {
	var c int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM expenditures WHERE amount = ?`, amountCents).Scan(&c)
	return c > 0, err
}

// CheckImportDuplicate returns true if an imported_transaction with the same
// description+date+amount already exists (any status), preventing double-imports.
func CheckImportDuplicate(description, processedDate string, amount int) (bool, error) {
	var c int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM imported_transactions
		WHERE description = ? AND processed_date = ? AND amount = ?
	`, description, processedDate, amount).Scan(&c)
	return c > 0, err
}

// ConfirmAllAuto confirms all pending debit transactions that have an auto_category_id.
// Returns the count of confirmed transactions.
func ConfirmAllAuto() (int, error) {
	// Fetch eligible transactions.
	rows, err := db.DB.Query(`
		SELECT id, description, amount, processed_date, auto_category_id
		FROM imported_transactions
		WHERE is_confirmed = 0 AND is_dismissed = 0
		  AND credit_or_debit = 'Debit'
		  AND auto_category_id IS NOT NULL
	`)
	if err != nil {
		return 0, err
	}

	type pending struct {
		id, amount, catID int
		desc, date        string
	}
	var items []pending
	for rows.Next() {
		var p pending
		var catID int
		if err := rows.Scan(&p.id, &p.desc, &p.amount, &p.date, &catID); err != nil {
			rows.Close()
			return 0, err
		}
		p.catID = catID
		items = append(items, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return 0, err
	}

	for _, p := range items {
		if _, err := tx.Exec(`
			INSERT INTO expenditures (description, amount, category_id, date) VALUES (?, ?, ?, ?)
		`, p.desc, p.amount, p.catID, p.date); err != nil {
			tx.Rollback()
			return 0, err
		}
		if _, err := tx.Exec(`
			UPDATE imported_transactions SET is_confirmed = 1, category_id = ? WHERE id = ?
		`, p.catID, p.id); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(items), nil
}

// DismissAllCredits dismisses all pending credit transactions.
// Returns the count dismissed.
func DismissAllCredits() (int, error) {
	result, err := db.DB.Exec(`
		UPDATE imported_transactions
		SET is_dismissed = 1
		WHERE is_confirmed = 0 AND is_dismissed = 0 AND credit_or_debit = 'Credit'
	`)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// ListRules returns all transaction categorization rules.
func ListRules() ([]TransactionRule, error) {
	rows, err := db.DB.Query(`
		SELECT r.id, r.keyword, r.category_id, bc.name, r.created_at
		FROM transaction_rules r
		JOIN budget_categories bc ON bc.id = r.category_id
		ORDER BY r.keyword ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []TransactionRule
	for rows.Next() {
		var r TransactionRule
		if err := rows.Scan(&r.ID, &r.Keyword, &r.CategoryID, &r.CategoryName, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// SaveRule inserts or replaces a transaction rule (keyword stored as UPPERCASE).
func SaveRule(keyword string, categoryID int) error {
	keyword = strings.ToUpper(strings.TrimSpace(keyword))
	if keyword == "" {
		return nil
	}
	_, err := db.DB.Exec(`INSERT OR REPLACE INTO transaction_rules (keyword, category_id) VALUES (?, ?)`, keyword, categoryID)
	return err
}

// UpdateRule changes the category of an existing rule.
func UpdateRule(id, categoryID int) error {
	_, err := db.DB.Exec(`UPDATE transaction_rules SET category_id = ? WHERE id = ?`, categoryID, id)
	return err
}

// DeleteRule removes a transaction rule by ID.
func DeleteRule(id int) error {
	_, err := db.DB.Exec(`DELETE FROM transaction_rules WHERE id = ?`, id)
	return err
}

// AutoCategorize finds the first matching rule for a description and returns its category ID.
// Rules are checked longest-keyword-first to prefer more specific matches.
func AutoCategorize(description string) (*int, error) {
	rows, err := db.DB.Query(`
		SELECT category_id, keyword FROM transaction_rules
		ORDER BY LENGTH(keyword) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	upper := strings.ToUpper(description)
	for rows.Next() {
		var catID int
		var keyword string
		if err := rows.Scan(&catID, &keyword); err != nil {
			return nil, err
		}
		if strings.Contains(upper, strings.ToUpper(keyword)) {
			return &catID, nil
		}
	}
	return nil, rows.Err()
}

// ExtractKeyword strips common bank prefixes from a transaction description
// and returns the first 1-2 meaningful words as an uppercase keyword.
func ExtractKeyword(description string) string {
	s := strings.TrimSpace(description)
	upper := strings.ToUpper(s)

	// Strip prefixes (longest first to avoid partial matches).
	// "WITHDRAWAL POS #" needs digit-stripping after prefix removal.
	posPrefix := "WITHDRAWAL POS #"
	prefixes := []string{
		"WITHDRAWAL BILL PAYMENT BILL PAID-",
		"WITHDRAWAL VISA DEBIT ",
		"WITHDRAWAL ACH ",
		"WITHDRAWAL TRANSFER TO ",
		posPrefix,
		"WITHDRAWAL ",
		"DEPOSIT ACH ",
		"DEPOSIT ",
	}

	matchedPOS := false
	for _, p := range prefixes {
		if strings.HasPrefix(upper, p) {
			s = s[len(p):]
			if p == posPrefix {
				matchedPOS = true
			}
			break
		}
	}

	// After stripping "WITHDRAWAL POS #", strip any remaining leading digits.
	if matchedPOS {
		i := 0
		for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
			i++
		}
		s = strings.TrimSpace(s[i:])
	}

	// Collect up to 2 meaningful words.
	tokens := strings.Fields(s)
	var words []string
	for _, tok := range tokens {
		if len(words) >= 2 {
			break
		}
		// Stop on: 2-letter alpha (state code), starts with digit, ends with ':'
		if strings.HasSuffix(tok, ":") {
			break
		}
		if len(tok) == 2 && isAlpha(tok) {
			break
		}
		if len(tok) > 0 && tok[0] >= '0' && tok[0] <= '9' {
			break
		}
		words = append(words, tok)
	}

	return strings.ToUpper(strings.Join(words, " "))
}

func isAlpha(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}
