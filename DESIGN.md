# BudgetFlow — Design Document

---

## 1. Atomicity & Anti Double-Spend

### Masalah yang harus diselesaikan

Sistem ini adalah financial ledger — setiap operasi melibatkan mutasi saldo di beberapa tabel sekaligus. Tanpa mekanisme yang tepat, dua skenario berbahaya bisa terjadi:

1. **Partial update:** Satu tabel berhasil diupdate, tabel lain gagal. Contoh: `envelope_remaining` berkurang tapi `reimburse_available` employee tidak bertambah — uang "hilang" dari sistem.
2. **Double-spend / race condition:** Dua request concurrent approve klaim bersamaan pada project dengan sisa envelope yang menipis, keduanya membaca nilai yang sama sebelum salah satu commit, dan keduanya berhasil — menghasilkan `envelope_remaining` negatif.

### Solusi: db.Transaction + WithTx pattern

Semua operasi yang melibatkan lebih dari satu tabel dibungkus dalam `db.Transaction()`. Setiap repository method menerima `*gorm.DB` yang sama melalui pola `WithTx(tx)` — bukan koneksi terpisah — sehingga semua mutasi berada dalam satu database transaction yang sama. Jika salah satu operasi gagal, seluruh transaction rollback otomatis.

```
usecase.ApproveClaim():
  db.Transaction(func(tx) {
      projectRepo.WithTx(tx).DecrementEnvelope(...)   // table 1
      managerRepo.WithTx(tx).DecrementLocked(...)     // table 2
      employeeRepo.WithTx(tx).IncrementAvailable(...) // table 3
      claimRepo.WithTx(tx).UpdateStatus(...)          // table 4
  })
  // Semua berhasil → commit. Salah satu gagal → rollback semua.
```

### Solusi: Conditional UPDATE untuk anti race-condition

Untuk mencegah double-spend saat concurrent requests, digunakan pola `UPDATE ... WHERE kondisi_masih_terpenuhi` dan kemudian memeriksa `RowsAffected`:

```sql
-- Claim approval: hanya berhasil jika envelope masih cukup
UPDATE projects
   SET envelope_remaining = envelope_remaining - ?
 WHERE id = ? AND envelope_remaining >= ?
```

Jika dua goroutine menjalankan query ini bersamaan, PostgreSQL akan menjalankan keduanya secara serial di level row lock. Goroutine pertama berhasil dan commit. Goroutine kedua menemukan `envelope_remaining` sudah tidak cukup, `RowsAffected = 0`, dan transaction-nya rollback dengan error `ENVELOPE_EXHAUSTED`. Tidak ada window di mana keduanya bisa lolos sekaligus.

Pola yang sama diterapkan pada:
- **Budget lock (buka project):** `UPDATE managers WHERE budget_available >= envelope_total`
- **Payout lock:** `UPDATE employees WHERE reimburse_available >= amount`

### Solusi: Idempotent guard pada review

Semua operasi review menggunakan `UPDATE ... WHERE status = 'pending'`. Jika sebuah klaim sudah berstatus `approved`, query ini tidak akan mengupdate row manapun (`RowsAffected = 0`), dan sistem mengembalikan error `INVALID_STATUS_TRANSITION`. Ini mencegah double-approval bahkan tanpa mekanisme locking tambahan.

---

## 2. Representasi Mata Uang & Kebijakan Pembulatan Fee

### Mengapa int64, bukan float64

Representasi uang menggunakan `int64` dalam satuan rupiah (terkecil). Alasannya bukan preferensi — ini adalah keharusan untuk sistem finansial.

`float64` mengikuti standar IEEE 754 yang merepresentasikan bilangan dalam basis 2 (binary). Sebagian besar bilangan desimal tidak dapat direpresentasikan secara tepat dalam binary, sehingga terjadi *precision loss*:

```
0.1 + 0.2 = 0.30000000000000004  // bukan 0.3
```

Dalam konteks keuangan, error sekecil apapun tidak dapat diterima karena:
- Akumulasi error kecil dalam jutaan transaksi menghasilkan selisih yang signifikan
- Audit trail tidak akan balance
- Regulasi keuangan mensyaratkan perhitungan yang deterministik dan reproducible

Dengan `int64` (satuan rupiah), `Rp 10.000` direpresentasikan sebagai `10000`. Semua operasi aritmatika adalah integer math yang exact — tidak ada precision loss.

### Kebijakan pembulatan fee 2.5%

Fee dihitung dengan formula: `fee = amount * 25 / 1000`

Ini menggunakan integer division bawaan Go, yang secara natural melakukan **floor** (pembulatan ke bawah):

| Amount (Rp) | Perhitungan | Fee | Net |
|---|---|---|---|
| 1.000.000 | 1000000 * 25 / 1000 | 25.000 | 975.000 |
| 1.000.001 | 25000.025 → floor | 25.000 | 975.001 |
| 100 | 2.5 → floor | 2 | 98 |
| 1 | 0.025 → floor | 0 | 1 |

**Mengapa floor, bukan round atau ceiling?**

Floor dipilih karena platform mengambil sedikit *kurang* dari 2.5% — bukan lebih. Ini adalah prinsip *favor the user*:
- Overcollecting fee (ceiling) berisiko melanggar regulasi dan merusak kepercayaan user
- Undercollecting fee (floor) adalah business decision yang aman — selisihnya maksimal Rp 0 s/d Rp 0,999 per transaksi
- Kebijakan ini deterministik dan mudah diaudit: `fee + net_amount` selalu sama dengan `amount`

---

## 3. Keputusan desain dan Trade-off

### Yang sudah diimplementasikan

Semua requirement wajib terpenuhi, beberapa fitur optional:

| Fitur | Status | Keterangan |
|---|---|---|
| Atomicity + WithTx | ✅ | Semua multi-table ops dalam satu TX |
| Anti double-spend | ✅ | Conditional UPDATE + RowsAffected |
| Idempotent review | ✅ | UPDATE WHERE status='pending' |
| JWT authentication | ✅ | HS256, 24h access token |
| Refresh token rotation | ✅ | Sessions table, blocked on reuse, 7-day expiry |
| RBAC authorization | ✅ | Per-endpoint role check |
| Rate limiting | ✅ | 100 req/min per IP, x/time/rate |
| Idempotency-Key | ✅ | Middleware + idempotent_requests table |
| Soft delete + restore | ✅ | Projects, ownership-validated |
| Structured logging | ✅ | log/slog, JSON format |
| Graceful shutdown | ✅ | SIGINT/SIGTERM, 5s drain |
| Docker + Compose | ✅ | App + Postgres + auto-migrate |
| Concurrent test | ✅ | 5 goroutines, real Postgres |

### Keputusan desain yang disengaja

**1. Tidak ada audit log**

Tidak diimplementasikan karena desain audit log yang benar memerlukan keputusan arsitektur tambahan yang di luar scope study case ini. Membuat audit log setengah-setengah lebih berbahaya dari tidak ada karena memberikan false confidence.

**2. Rate limiter in-memory**

In-memory dipilih secara sadar karena Redis menambahkan operational complexity yang tidak justified tanpa requirement multi-instance yang eksplisit. Untuk single-instance deployment sesuai scope ini, in-memory sudah sufficient.

**3. Tidak ada pagination pada `/me/topups`, `/me/claims`, `/payouts`**

List endpoints yang ada (/me/topups, /me/claims, /payouts) mengembalikan semua records milik user. Acceptable untuk internal company tool dengan volume rendah. Cursor-based pagination akan diprioritaskan saat ada kebutuhan scaling.

**4. CI tidak diaktifkan**

GitHub Actions workflow template tersedia di `.github/workflows/ci.yml` namun belum diaktifkan. Prioritas pengerjaan difokuskan pada correctness business logic dan test coverage untuk financial invariants — CI adalah enforcement mechanism untuk tim, bukan prerequisite untuk kualitas kode itu sendiri.

### Apa yang akan dikerjakan untuk production

**Prioritas tinggi:**
- **Audit log table** — setiap mutasi saldo direkam dengan `user_id`, `action`, `before`, `after`, `timestamp`. Immutable, append-only.
- **Redis-backed rate limiter** — untuk horizontal scaling dan persistence across restarts
- **Database advisory lock atau Redis distributed lock** — untuk skenario high-concurrency payout yang lebih ekstrem dari yang bisa di-handle conditional UPDATE saja
- **Monitoring + alerting** — alert otomatis jika ada balance negatif terdeteksi di database (seharusnya tidak pernah terjadi, tapi defense in depth)

**Prioritas menengah:**
- **OpenTelemetry tracing** — distributed tracing untuk debug performance bottleneck
- **Pagination cursor-based** — untuk list endpoints, lebih scalable dari offset pagination
- **File upload integration** — S3/GCS untuk `receipt_url` dengan presigned URL dan virus scanning
- **CI/CD pipeline** — GitHub Actions yang run test, build Docker image, dan deploy ke staging otomatis

---

## 4. Idempotency untuk Webhook External

### Masalah

Webhook dari sistem eksternal (payment gateway, ERP) menggunakan pola *at-least-once delivery* — request yang sama bisa dikirim lebih dari sekali karena network timeout, retry logic, atau failure di sisi pengirim. Tanpa mekanisme khusus, sebuah payout bisa di-approve dua kali jika webhook retry terjadi setelah response pertama tidak diterima pengirim.

### Mengapa idempotent guard saja tidak cukup

Guard `UPDATE WHERE status='pending'` mencegah double-processing pada level bisnis, tapi response yang dikembalikan untuk request kedua akan berbeda — `409 INVALID_STATUS_TRANSITION`. Ini masalah untuk webhook system yang mengekspektasikan response sukses sebagai konfirmasi bahwa operasinya berhasil diproses.

### Solusi yang diimplementasikan

Middleware `Idempotency-Key` bekerja di level HTTP, sebelum request mencapai handler:

```
Request masuk dengan header Idempotency-Key: <uuid>
         │
         ▼
Middleware cek tabel idempotent_requests
         │
    ┌────┴────┐
  Ada?       Tidak ada?
    │              │
    ▼              ▼
Return cached   Jalankan handler
response        Simpan response ke tabel
(status + body) Return response normal
```

Tabel `idempotent_requests` menyimpan: `key`, `status_code`, `response_body`, `created_at`. TTL 24 jam — setelah itu key dianggap expired dan bisa digunakan ulang.

Pengirim webhook yang retry dengan key yang sama akan mendapat response yang identik dengan response pertama — termasuk status code dan body — tanpa operasi dijalankan ulang. Ini memenuhi kontrak *exactly-once semantics* dari sisi penerima.

Diterapkan pada: `POST /topups/:id/review`, `POST /claims/:id/review`, `POST /payouts/:id/review`.