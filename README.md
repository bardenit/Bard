# Bard

A self-hosted personal budgeting web application. Single-user, lightweight, no external dependencies.

## Features

- **Bills** — track recurring and one-time bills with due dates
- **Income** — manage recurring deposits and calculate available funds
- **Budgets** — create categories with allocated amounts, see what's left
- **Calendar** — monthly view showing bills, income, and net daily impact
- **Dashboard** — summary of monthly income, bills, net available, and upcoming events

## Tech Stack

- Go (standard library + `mattn/go-sqlite3`)
- SQLite
- Server-rendered HTML templates
- Vanilla CSS/JS

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

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `DB_PATH` | `budget.db` | Path to SQLite database file |
| `TEMPLATE_DIR` | `templates` | Path to HTML templates |
| `STATIC_DIR` | `static` | Path to static assets |

## Data Model

All monetary values are stored as integer cents. Dates are ISO 8601 strings (`YYYY-MM-DD`).

- **bills** — name, amount, due date, recurrence (once/weekly/biweekly/monthly)
- **income** — name, amount, deposit date, recurrence
- **budget_categories** — hierarchical categories (one level of nesting)
- **budgets** — allocated amount per category per recurrence period
