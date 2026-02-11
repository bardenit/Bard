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

### Docker Compose

Create a `docker-compose.yml`:

```yaml
services:
  budget:
    image: jbarden75/budget:latest
    container_name: budget
    ports:
      - "8080:8080"
    volumes:
      - budget-data:/data
    restart: unless-stopped

volumes:
  budget-data:
```

Then run:

```bash
docker compose up -d
```

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
