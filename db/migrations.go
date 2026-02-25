package db

import (
	"log"
	"strings"
)

var migrations = []string{
	// Migration 0: bills — no CHECK constraint so 'yearly' is a valid recurrence
	`CREATE TABLE IF NOT EXISTS bills (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL,
		amount     INTEGER NOT NULL,
		due_date   TEXT    NOT NULL,
		recurrence TEXT    NOT NULL DEFAULT 'monthly',
		is_active  INTEGER NOT NULL DEFAULT 1,
		created_at TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	// Migration 1: income — no CHECK constraint, is_primary added by later migration
	`CREATE TABLE IF NOT EXISTS income (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		name         TEXT    NOT NULL,
		amount       INTEGER NOT NULL,
		deposit_date TEXT    NOT NULL,
		recurrence   TEXT    NOT NULL DEFAULT 'biweekly',
		is_active    INTEGER NOT NULL DEFAULT 1,
		created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at   TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS budget_categories (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL,
		parent_id  INTEGER REFERENCES budget_categories(id) ON DELETE CASCADE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS budgets (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id    INTEGER NOT NULL REFERENCES budget_categories(id) ON DELETE CASCADE,
		amount         INTEGER NOT NULL,
		recurrence     TEXT    NOT NULL CHECK (recurrence IN ('weekly', 'biweekly', 'monthly')),
		effective_date TEXT    NOT NULL,
		is_active      INTEGER NOT NULL DEFAULT 1,
		created_at     TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at     TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS expenditures (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT    NOT NULL,
		amount      INTEGER NOT NULL,
		category_id INTEGER NOT NULL REFERENCES budget_categories(id) ON DELETE CASCADE,
		date        TEXT    NOT NULL,
		created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS transaction_rules (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		keyword     TEXT    NOT NULL UNIQUE,
		category_id INTEGER NOT NULL REFERENCES budget_categories(id) ON DELETE CASCADE,
		created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS imported_transactions (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		account_name      TEXT    NOT NULL DEFAULT '',
		processed_date    TEXT    NOT NULL,
		description       TEXT    NOT NULL,
		credit_or_debit   TEXT    NOT NULL,
		amount            INTEGER NOT NULL,
		category_id       INTEGER REFERENCES budget_categories(id) ON DELETE SET NULL,
		auto_category_id  INTEGER REFERENCES budget_categories(id) ON DELETE SET NULL,
		is_duplicate_flag INTEGER NOT NULL DEFAULT 0,
		is_confirmed      INTEGER NOT NULL DEFAULT 0,
		is_dismissed      INTEGER NOT NULL DEFAULT 0,
		source_file       TEXT    NOT NULL DEFAULT '',
		imported_at       TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_expenditures_amount ON expenditures(amount)`,
	`ALTER TABLE imported_transactions ADD COLUMN balance INTEGER`,
	`CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	// Add is_primary to income (idempotent — "duplicate column name" is tolerated)
	`ALTER TABLE income ADD COLUMN is_primary INTEGER NOT NULL DEFAULT 0`,
	// Bill payments — tracks actual amounts paid for each bill occurrence
	`CREATE TABLE IF NOT EXISTS bill_payments (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		bill_id         INTEGER NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
		occurrence_date TEXT    NOT NULL,
		amount_paid     INTEGER NOT NULL,
		notes           TEXT    NOT NULL DEFAULT '',
		created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
		UNIQUE(bill_id, occurrence_date)
	)`,
}

func RunMigrations() {
	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			s := err.Error()
			// Tolerate expected non-fatal migration conditions.
			if strings.Contains(s, "duplicate column name") ||
				strings.Contains(s, "no such table") ||
				strings.Contains(s, "there is already a table named") {
				continue
			}
			log.Fatalf("Migration failed: %v\nSQL: %s", err, m)
		}
	}

	// Fix bills and income table schemas for existing databases that have
	// the old CHECK constraint (which blocked 'yearly' recurrence).
	fixBillsTable()
	fixIncomeTable()

	log.Println("Migrations complete")
}

// fixBillsTable rebuilds the bills table without the old CHECK constraint on
// recurrence, allowing 'yearly' entries. Uses sqlite_master to detect whether
// a rebuild is needed, making this safe to call on every startup.
func fixBillsTable() {
	var schema string
	DB.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='bills'`).Scan(&schema)
	if schema == "" || !strings.Contains(schema, "CHECK") {
		return // already fixed or table doesn't exist
	}

	steps := []string{
		`ALTER TABLE bills RENAME TO bills_old_ck`,
		`CREATE TABLE IF NOT EXISTS bills (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL,
			amount     INTEGER NOT NULL,
			due_date   TEXT    NOT NULL,
			recurrence TEXT    NOT NULL DEFAULT 'monthly',
			is_active  INTEGER NOT NULL DEFAULT 1,
			created_at TEXT    NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT    NOT NULL DEFAULT (datetime('now'))
		)`,
		`INSERT OR IGNORE INTO bills (id,name,amount,due_date,recurrence,is_active,created_at,updated_at)
		 SELECT id,name,amount,due_date,recurrence,is_active,created_at,updated_at FROM bills_old_ck`,
		`DROP TABLE IF EXISTS bills_old_ck`,
	}
	for _, s := range steps {
		if _, err := DB.Exec(s); err != nil {
			log.Printf("fixBillsTable step failed (non-fatal): %v", err)
		}
	}
	log.Println("bills table rebuilt without CHECK constraint")
}

// fixIncomeTable rebuilds the income table without the old CHECK constraint on
// recurrence and with the is_primary column. Safe to call on every startup.
func fixIncomeTable() {
	var schema string
	DB.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='income'`).Scan(&schema)
	if schema == "" {
		return
	}
	needsRebuild := strings.Contains(schema, "CHECK") || !strings.Contains(schema, "is_primary")
	if !needsRebuild {
		return
	}

	steps := []string{
		`ALTER TABLE income RENAME TO income_old_ck`,
		`CREATE TABLE IF NOT EXISTS income (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			name         TEXT    NOT NULL,
			amount       INTEGER NOT NULL,
			deposit_date TEXT    NOT NULL,
			recurrence   TEXT    NOT NULL DEFAULT 'biweekly',
			is_primary   INTEGER NOT NULL DEFAULT 0,
			is_active    INTEGER NOT NULL DEFAULT 1,
			created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
			updated_at   TEXT    NOT NULL DEFAULT (datetime('now'))
		)`,
		`INSERT OR IGNORE INTO income (id,name,amount,deposit_date,recurrence,is_active,created_at,updated_at)
		 SELECT id,name,amount,deposit_date,recurrence,is_active,created_at,updated_at FROM income_old_ck`,
		`DROP TABLE IF EXISTS income_old_ck`,
	}
	for _, s := range steps {
		if _, err := DB.Exec(s); err != nil {
			log.Printf("fixIncomeTable step failed (non-fatal): %v", err)
		}
	}
	log.Println("income table rebuilt with is_primary column")
}
