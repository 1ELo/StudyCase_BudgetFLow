# BudgetFlow

Backend sistem pengelolaan budget dan reimbursement karyawan.

## Tech Stack

- **Language:** Go 1.22+
- **Framework:** Gin
- **ORM:** GORM + PostgreSQL
- **Auth:** JWT (HS256)
- **Migration:** golang-migrate

## Quick Start

\```bash
# 1. Clone repository
git clone https://github.com/1ELo/StudyCase_BudgetFLow.git
cd StudyCase_BudgetFLow

# 2. Setup environment
cp .env.example .env
# Edit .env sesuai konfigurasi lokal

# 3. Jalankan migrasi
make migrate-up

# 4. Seed data awal
make seed

# 5. Jalankan server
make run
\```

## API Documentation

Lihat file `api-collection.http` atau import `budgetflow.postman_collection.json`.

## Testing

\```bash
make test
\```

## Architecture

Lihat `DESIGN.md` untuk penjelasan keputusan arsitektur [On_Progress].

## Status

> [!NOTE]
> On Progress.