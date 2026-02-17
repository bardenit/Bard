package db

import "log"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS bills (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL,
		amount     INTEGER NOT NULL,
		due_date   TEXT    NOT NULL,
		recurrence TEXT    NOT NULL CHECK (recurrence IN ('once', 'weekly', 'biweekly', 'monthly')),
		is_active  INTEGER NOT NULL DEFAULT 1,
		created_at TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT    NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS income (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		name         TEXT    NOT NULL,
		amount       INTEGER NOT NULL,
		deposit_date TEXT    NOT NULL,
		recurrence   TEXT    NOT NULL CHECK (recurrence IN ('once', 'weekly', 'biweekly', 'monthly')),
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
}

func RunMigrations() {
	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v\nSQL: %s", err, m)
		}
	}
	log.Println("Migrations complete")
}
