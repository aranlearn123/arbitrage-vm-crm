# Arbitrage VM CRM Backend

Go Fiber backend API สำหรับ CRM monitoring ระบบ arbitrage trading

เป้าหมายแรกของ service นี้คือเป็น read-only API สำหรับอ่านข้อมูลจาก database ของ `d:\platform` เพื่อใช้ทำ dashboard, allocation monitoring, historical data, trade log, PnL และ equity curve

## Tech Stack

- Go
- Fiber
- PostgreSQL / TimescaleDB
- Bun ORM
- Swagger ผ่าน `swaggo`

## Project Structure

```text
backend/
  cmd/
    api/
      main.go
  internal/
    config/
      config.go
    database/
      postgres.go
    handler/
      health_handler.go
      allocation_handler.go
    repo/
      allocation_repo.go
    response/
      health.go
      allocation.go
  .env
  go.mod
```

## Environment

สร้างไฟล์ `.env` ใน root ของ `backend`

```env
APP_PORT=8080
DATABASE_URL=postgres://postgres:password@localhost:5432/bidsize?sslmode=disable
CORS_ORIGINS=http://localhost:3000,http://localhost:5173
```

หมายเหตุ:

- `DATABASE_URL` ต้องชี้ไป database เดียวกับระบบ `d:\platform`
- phase แรกให้ API นี้เป็น read-only ก่อน ยังไม่ควรมี endpoint ที่แก้ state trading engine

## Run

```powershell
cd D:\arbitrage_vm_crm\backend
go run .\cmd\api
```

API จะ start ที่:

```text
http://localhost:8080
```

## Health Check

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/health
```

Expected response:

```json
{
  "status": "ok",
  "time": "2026-04-29T03:25:48Z"
}
```

## Test

```powershell
go test ./...
```

## Swagger

ติดตั้ง tool:

```powershell
go install github.com/swaggo/swag/cmd/swag@latest
```

ติดตั้ง dependency:

```powershell
go get github.com/swaggo/fiber-swagger
go get github.com/swaggo/swag
```

Generate docs:

```powershell
swag init -g .\cmd\api\main.go -o .\docs
```

หลัง register Swagger route แล้ว เปิด:

```text
http://localhost:8080/swagger/
```

ทุกครั้งที่แก้ Swagger annotation ให้รัน `swag init` ใหม่

## Current API

### `GET /api/v1/health`

ตรวจว่า API service ทำงานอยู่

Response:

```json
{
  "status": "ok",
  "time": "2026-04-29T03:25:48Z"
}
```

## Recommended Development Order

### Phase 1: Health Baseline

- config loader
- Fiber server
- middleware: recover, logger, cors
- health endpoint
- Swagger setup

### Phase 2: Database

เพิ่ม `internal/database/postgres.go`

หน้าที่:

- ต่อ PostgreSQL ด้วย `DATABASE_URL`
- ping database ตอน start
- return `*bun.DB`

### Phase 3: Allocation API

เพิ่ม endpoint:

```text
GET /api/v1/allocations/summary
GET /api/v1/allocations
GET /api/v1/allocations/active
GET /api/v1/allocations/running
GET /api/v1/allocations/cancelled
GET /api/v1/allocations/cancelled/reasons
```

Data source:

- `allocations`

Active statuses:

```text
created
running
failed
paused
```

Important SQL:

```sql
SELECT status, COUNT(*) AS count
FROM allocations
GROUP BY status
ORDER BY status;
```

```sql
SELECT *
FROM allocations
WHERE status IN ('created', 'running', 'failed', 'paused')
ORDER BY updated_at DESC;
```

```sql
SELECT COALESCE(note, 'unknown') AS reason, COUNT(*) AS count
FROM allocations
WHERE status = 'cancelled'
GROUP BY COALESCE(note, 'unknown')
ORDER BY count DESC;
```

### Phase 4: Market Data API

เพิ่ม endpoint:

```text
GET /api/v1/funding/latest
GET /api/v1/funding/history
GET /api/v1/funding/spread
GET /api/v1/open-interest/latest
GET /api/v1/open-interest/history
GET /api/v1/market-quality/latest
GET /api/v1/market-quality/history
```

Data source:

- `funding`
- `open_interest`
- `market_quality_metrics_1m`

### Phase 5: PnL / Equity

เพิ่ม endpoint:

```text
GET /api/v1/pnl/events
GET /api/v1/pnl/summary
GET /api/v1/equity/latest
GET /api/v1/equity/curve
GET /api/v1/equity/curve/combined
```

Data source:

- `pnl_events`
- future table: `wallet_snapshots`

## Known Gaps

### Allocation Events

ตอนนี้ runtime reject บางเคสใน `d:\platform` ถูก log แล้ว drop ก่อน insert เข้า `allocations`

ถ้าต้องการให้ CRM นับ cancel/reject ได้ครบ ควรเพิ่ม table:

```text
allocation_events
```

### Equity Curve

ตอนนี้มี wallet balance ใน runtime แต่ยังไม่มี historical table สำหรับ equity curve

ควรเพิ่ม:

```text
wallet_snapshots
```

### Trade Log

มี normalized account event struct ใน `d:\platform` แล้ว แต่ยังไม่มี table สำหรับ CRM trade log

ควรเพิ่ม:

```text
account_events
```

## Engineering Notes

- response ที่เป็นเงินหรือ decimal ควรส่งเป็น string เพื่อกัน precision loss
- list endpoint ทุกตัวต้องมี `limit`
- historical endpoint ทุกตัวควรรับ `from` และ `to`
- dashboard endpoint ควรระวัง query หนัก ควร cache หรือ aggregate เมื่อข้อมูลเยอะ
- phase แรกควรเป็น read-only เท่านั้น
