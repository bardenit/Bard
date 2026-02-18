# Bard

A self-hosted personal budgeting web application. Single-user, lightweight, no external dependencies. Installable as a PWA on desktop and mobile.

**Current version: 2.4**

## Features

- **Dashboard** — monthly summary of income, bills, net available funds, upcoming events, and budget status totals
- **Bills** — track recurring and one-time bills with due dates; recurrence: once, weekly, biweekly, monthly, yearly
- **Income** — manage income sources with recurrence; mark one as the primary paycheck for bill-due calculations
- **Budgets** — hierarchical categories (parent → subcategories) with allocated amounts; see spent vs. remaining per category
- **Expenditures** — log individual spending entries against budget categories
- **Transactions** — upload monthly CSV bank exports; auto-categorize by keyword rules; review, confirm, or dismiss; confirmed transactions become expenditures automatically; search confirmed history by description or category
- **Calendar** — monthly view showing bills, income, and net daily impact
- **Backup & Restore** — export all data (or everything except transactions) as a JSON file; restore from a previous backup

## Tech Stack

- Go (standard library + `mattn/go-sqlite3`)
- SQLite
- Server-rendered HTML templates
- Vanilla CSS/JS
- PWA (installable on desktop and mobile via HTTPS)

## Quick Start

### Local Development

Requires Go 1.21+ and a C compiler (for SQLite).

```bash
go run .
```

Open http://localhost:8080

The database file (`budget.db`) is created in the current directory.

### Docker

```bash
docker run -d -p 8080:8080 -v budget-data:/data jbarden75/budget:latest
```

Open http://localhost:8080

Data persists in the `budget-data` Docker volume.

### Docker Compose (with HTTPS — recommended)

The repo includes a `docker-compose.yml` and `Caddyfile` that add a Caddy reverse
proxy so the app is reachable over **HTTPS** from any device on your network.
HTTPS is required for the PWA "Install App" prompt to work in Chrome and Edge.

```bash
docker compose up -d
```

- `http://localhost:8080` — direct access on the Docker host (no HTTPS, still works)
- `https://YOUR-PC-IP` — HTTPS via Caddy (required for PWA install on other devices)

**First visit over HTTPS:** your browser will show "Your connection isn't private"
because Caddy uses its own internal certificate authority. Click
**Advanced → Continue to site** once. The warning won't appear again on that browser.

**Remove the warning permanently (optional, Windows):**

```powershell
# Run in PowerShell as Administrator
docker cp budget-proxy:/data/caddy/pki/authorities/local/root.crt C:\caddy-root.crt
certutil -addstore -f "ROOT" C:\caddy-root.crt
```

Run this on each Windows PC. After that, no more warning and Edge will offer the
Install App button automatically on every visit.

**iPhone:** In Safari, visit `https://YOUR-PC-IP` → accept the warning → go to
Settings → General → VPN & Device Management → install the Caddy profile →
Settings → General → About → Certificate Trust Settings → enable it.
Then "Add to Home Screen" works with no warning.

### Upgrading an existing Docker install

If you already have a `budget-data` volume from a previous `docker run`, the
compose file preserves it automatically (`name: budget-data` in the volumes
section). No data loss.

```bash
docker compose pull   # grab latest images
docker compose up -d  # restart with new version
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `DB_PATH` | `budget.db` | Path to SQLite database file |
| `TEMPLATE_DIR` | `templates` | Path to HTML templates |
| `STATIC_DIR` | `static` | Path to static assets |
| `AUTH_USERNAME` | _(unset)_ | Username for password login |
| `AUTH_PASSWORD_HASH` | _(unset)_ | bcrypt hash of password (see Authentication section) |
| `SESSION_SECRET` | _(random)_ | HMAC key for session cookies; if unset, sessions are lost on restart |
| `OIDC_ISSUER` | _(unset)_ | Pocket ID base URL e.g. `https://id.example.com` |
| `OIDC_CLIENT_ID` | _(unset)_ | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | _(unset)_ | OAuth2 client secret |
| `OIDC_REDIRECT_URL` | _(unset)_ | Full callback URL e.g. `https://budget.example.com/login/oidc/callback` |

## Authentication

Auth is fully opt-in. If none of the auth env vars are set, the app works exactly as before — no login required.

### Password Login

1. Generate a bcrypt hash of your password:
   ```bash
   htpasswd -bnBC 12 "" yourpassword | tr -d ':\n' | sed 's/$2y/$2a/'
   ```
2. Set env vars:
   ```
   AUTH_USERNAME=admin
   AUTH_PASSWORD_HASH=<hash from above>
   SESSION_SECRET=<random 32+ char string>
   ```

### OIDC / Pocket ID (SSO)

1. In Pocket ID, create a new OAuth2 client.
2. Set the **Redirect URI** to `https://budget.example.com/login/oidc/callback` (your actual domain). This URL is also displayed on the login page itself when OIDC is configured.
3. Copy the **Client ID** and **Client Secret**.
4. Set env vars:
   ```
   OIDC_ISSUER=https://id.example.com
   OIDC_CLIENT_ID=<client id>
   OIDC_CLIENT_SECRET=<client secret>
   OIDC_REDIRECT_URL=https://budget.example.com/login/oidc/callback
   SESSION_SECRET=<random 32+ char string>
   ```
5. At startup, the redirect URI is also printed to the container logs so you can confirm it:
   ```
   Auth: OIDC enabled — register this redirect URI in Pocket ID: https://budget.example.com/login/oidc/callback
   ```

Both methods can be enabled simultaneously — the login page will show a password form and an SSO button.

## PWA Install

When accessed over HTTPS (or localhost), the app can be installed as a standalone
app on desktop (Chrome, Edge) and mobile (Chrome on Android, Safari on iOS).

- **Desktop (Chrome/Edge):** look for the install icon in the address bar, or use
  the install banner that appears at the top of the page.
- **Android:** tap the browser menu → "Add to Home Screen" or "Install App".
- **iOS (Safari):** tap Share → "Add to Home Screen".

## Transactions: CSV Import

The Transactions page accepts CSV exports from your bank. Expected columns:

| # | Column | Example |
|---|---|---|
| 0 | Account name | Checking |
| 1 | Date (YYYY-MM-DD) | 2025-01-15 |
| 2 | Description | WITHDRAWAL VISA DEBIT MEIJER |
| 3 | Check # (ignored) | — |
| 4 | Credit or Debit | Debit |
| 5 | Amount (decimal) | 47.83 |

On import the app:
1. Strips common bank prefixes from descriptions and extracts a keyword
2. Matches the keyword against your saved rules to auto-select a category
3. Flags potential duplicates (same amount already in expenditures)
4. Skips rows already imported (same description + date + amount)

On the review table you can confirm (creates an expenditure), dismiss, or change
the category before confirming. Changing the category saves a new rule so future
imports auto-categorize that merchant correctly.

Bulk actions: **Confirm All Auto-Categorized** and **Dismiss All Credits** save
time when importing a full month at once.

## Data Model

All monetary values are stored as integer cents. Dates are ISO 8601 strings (`YYYY-MM-DD`).

- **bills** — name, amount, due date, recurrence (once/weekly/biweekly/monthly/yearly)
- **income** — name, amount, deposit date, recurrence, is_primary flag
- **budget_categories** — hierarchical categories (one level of nesting)
- **budgets** — allocated amount per category per recurrence period
- **expenditures** — individual spending entries linked to a budget category
- **transaction_rules** — keyword → category mappings for auto-categorization
- **imported_transactions** — raw CSV rows with review status and confirmed flag
