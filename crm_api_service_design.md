# CRM Monitoring Backend API Design

เอกสารนี้สรุป API ที่ควรมีสำหรับทำ CRM monitoring ระบบเทรด โดยอ่านข้อมูลหลักจาก `d:\platform`

เป้าหมายของ backend API:

- ดูสถานะระบบเทรดแบบรวม
- ดู allocation ที่กำลังทำงาน อยู่ state ไหน และ cancel เพราะอะไร
- ดู historical data เช่น funding, open interest, market quality
- ดู trade log, order/execution/position, PnL และ equity curve รวม 2 exchange
- รองรับ dashboard และหน้า drill-down ราย allocation / pair / exchange

## Implementation Checklist

สถานะนี้อิงจากโค้ด backend ปัจจุบันใน `backend/cmd/api/main.go`

### Foundation

- [x] Go Fiber API scaffold
- [x] `.env` config loader
- [x] PostgreSQL/Bun connection
- [x] Swagger docs route
- [x] Health check with database ping
- [ ] Graceful shutdown
- [ ] Standard pagination response with cursor
- [ ] Central error response/middleware

### System / Health API

- [x] `GET /api/v1/health`
- [x] `GET /api/v1/system/status`
- [x] `GET /api/v1/system/exchanges`

### Dashboard Summary API

- [ ] `GET /api/v1/dashboard/summary`
- [ ] `GET /api/v1/dashboard/live-state`
- [ ] `GET /api/v1/dashboard/risk`

### Allocation API

- [x] `GET /api/v1/allocations`
- [x] `GET /api/v1/allocations/summary`
- [x] `GET /api/v1/allocations/active`
- [x] `GET /api/v1/allocations/running`
- [x] `GET /api/v1/allocations/cancelled`
- [x] `GET /api/v1/allocations/cancelled/reasons`
- [x] `GET /api/v1/allocations/{id}`
- [x] `GET /api/v1/allocations/{id}/timeline`

### Allocation Event API

- [ ] Create table `allocation_events`
- [ ] `GET /api/v1/allocation-events`
- [ ] `GET /api/v1/allocation-events/summary`

### Instrument / Pair API

- [ ] `GET /api/v1/instruments`
- [ ] `GET /api/v1/pairs`
- [ ] `GET /api/v1/pairs/{base}-{quote}/latest`

### Funding API

- [x] `GET /api/v1/funding/latest`
- [x] `GET /api/v1/funding/history`
- [x] `GET /api/v1/funding/spread`
- [x] `GET /api/v1/funding/top-spreads`

### Open Interest API

- [ ] `GET /api/v1/open-interest/latest`
- [ ] `GET /api/v1/open-interest/history`
- [ ] `GET /api/v1/open-interest/skew`

### Market Quality API

- [x] `GET /api/v1/market-quality/latest`
- [x] `GET /api/v1/market-quality/history`
- [x] `GET /api/v1/market-quality/alerts`

### Order Routing / Execution Flow API

- [ ] `GET /api/v1/order-routing/{plan_id}`
- [ ] `GET /api/v1/order-routing/{plan_id}/progress`
- [ ] `GET /api/v1/orders`

### Trade Log / Account Event API

- [ ] Create table `account_events`
- [ ] `GET /api/v1/account-events`
- [ ] `GET /api/v1/trades`
- [ ] `GET /api/v1/positions`

### PnL API

- [x] `GET /api/v1/pnl/events`
- [x] `GET /api/v1/pnl/summary`
- [x] `GET /api/v1/pnl/by-pair`
- [x] `GET /api/v1/pnl/by-exchange`
- [x] `GET /api/v1/pnl/by-component`

### Equity Curve API

- [ ] Create table `wallet_snapshots`
- [x] `GET /api/v1/equity/latest`
- [x] `GET /api/v1/equity/live`
- [ ] `GET /api/v1/equity/curve`
- [ ] `GET /api/v1/equity/curve/combined`
- [ ] `GET /api/v1/equity/drawdown`

### Alert / Incident API

- [ ] Create table `alerts`
- [ ] `GET /api/v1/alerts`
- [ ] `GET /api/v1/alerts/active`
- [ ] `POST /api/v1/alerts/{id}/ack`
- [ ] `POST /api/v1/alerts/{id}/resolve`

### Realtime API

- [ ] `GET /api/v1/sse/events`
- [ ] `GET /api/v1/ws`

## Implemented So Far

Last updated: 2026-04-29

Current backend route source:

- `backend/cmd/api/main.go`
- Swagger: `GET /swagger/*`
- API base path: `/api/v1`

### Backend Foundation

- [x] วางโครงสร้าง Go Fiber backend ใน `backend/`
- [x] โหลด config จาก `.env`
- [x] เชื่อม PostgreSQL ด้วย Bun และ `pgdriver`
- [x] เพิ่ม route Swagger ที่ `GET /swagger/*`
- [x] เพิ่ม `.gitignore` เพื่อกัน `.env`, binary, log, temp file และ swagger build artifact ที่ไม่ควร commit
- [x] เพิ่ม `backend/README.md` สำหรับวิธี setup, run, swagger และ health check

### Health Check

- [x] `GET /api/v1/health`
- [x] ตรวจสถานะ API process
- [x] ตรวจ database ด้วย `PingContext`
- [x] คืน `503 Service Unavailable` ถ้า database ไม่พร้อม

Current response fields:

- `status`
- `time`
- `checks.api`
- `checks.database`

### System API

- [x] `GET /api/v1/system/status`
  - ตรวจ API version, database, TimescaleDB extension, latest market data, allocation count และ exchange status รวม
  - คืน `status = ok | degraded | unhealthy`
- [x] `GET /api/v1/system/exchanges`
  - list exchange ที่ CRM backend รองรับ: `bitget`, `bybit`
  - แสดง `credential_configured`, `demo`, `last_funding_at`, `last_open_interest_at`, `last_market_quality_at`, `last_market_data_at`
  - `last_account_event_at` และ `last_wallet_snapshot_at` ยังว่าง เพราะยังไม่มี persistence table สำหรับ account events / wallet snapshots

### Allocation API

- [x] `GET /api/v1/allocations`
  - ใช้สำหรับตาราง CRM หน้า Active/History
  - filter ได้ด้วย `status`, `base`, `quote`, `role`, `limit`
- [x] `GET /api/v1/allocations/summary`
  - สรุปจำนวน allocation ตาม `status`
  - สรุป cancel reason โดย normalize note ให้เหลือ reason หลัก เช่น `insufficient_net_edge_after_cost`
- [x] `GET /api/v1/allocations/active`
  - ดู allocation ที่ยัง active หรือยังไม่จบ flow
- [x] `GET /api/v1/allocations/running`
  - ดู allocation ที่อยู่สถานะ `running`
- [x] `GET /api/v1/allocations/cancelled`
  - ดู allocation ที่ถูก cancel
  - filter ได้ด้วย `base`, `quote`, `role`, `reason`, `limit`
  - `reason` ใช้ partial match แบบ `ILIKE '%reason%'` เพื่อให้ค้น `insufficient_net_edge_after_cost` เจอ note แบบเต็ม เช่น `execution_rejected: insufficient_net_edge_after_cost; signal_bps=...`
- [x] `GET /api/v1/allocations/cancelled/reasons`
  - group cancel reason หลักจาก `note`
  - ตัดรายละเอียด metric เช่น `signal_bps`, `cost_bps`, `net_bps`, `required_bps` ออกจาก summary
- [x] `GET /api/v1/allocations/{id}`
  - ดูรายละเอียด allocation ตาม id
- [x] `GET /api/v1/allocations/{id}/timeline`
  - รวม timeline จาก `allocations`, `scaling_plans`, `recovery_decisions`, `order_management_routing_states`, `order_management_progress_events`

### Funding API

- [x] `GET /api/v1/funding/latest`
  - อ่าน latest funding rate ต่อ `exchange/base/quote`
  - filter ได้ด้วย `exchange`, `base`, `quote`, `limit`
- [x] `GET /api/v1/funding/history`
  - อ่าน historical funding rate
  - filter ได้ด้วย `exchange`, `base`, `quote`, `from`, `to`, `limit`
- [x] `GET /api/v1/funding/spread`
  - คำนวณ latest spread ระหว่าง Bybit และ Bitget สำหรับ pair เดียว
  - คืน `bybit_rate`, `bitget_rate`, `spread`, `spread_bps`, `abs_spread_bps`, `direction_hint`
- [x] `GET /api/v1/funding/top-spreads`
  - จัดอันดับ pair ที่มี absolute funding spread สูงสุด
  - filter ได้ด้วย `base`, `quote`, `min_abs_spread_bps`, `limit`

### Market Quality API

- [x] `GET /api/v1/market-quality/latest`
  - อ่าน latest market quality ต่อ `exchange/base/quote`
  - filter ได้ด้วย `exchange`, `base`, `quote`, `limit`
- [x] `GET /api/v1/market-quality/history`
  - อ่าน historical market quality metrics
  - filter ได้ด้วย `exchange`, `base`, `quote`, `from`, `to`, `limit`
- [x] `GET /api/v1/market-quality/alerts`
  - คืน row ล่าสุดที่ผิด threshold แบบง่ายสำหรับ CRM
  - alert reason ปัจจุบัน: `low_samples`, `wide_spread`, `fast_mid_price`, `unstable_depth`

### PnL API

- [x] `GET /api/v1/pnl/events`
  - อ่าน ledger จาก `pnl_events`
  - filter ได้ด้วย `exchange`, `base`, `quote`, `component`, `source_type`, `source_id`, `from`, `to`, `limit`
- [x] `GET /api/v1/pnl/summary`
  - สรุป `total_amount`, `funding_amount`, `trading_fee_amount`, `trading_pnl_amount`
  - คืน breakdown ตาม component, exchange และ pair
- [x] `GET /api/v1/pnl/by-pair`
  - group PnL ตาม `base/quote`
- [x] `GET /api/v1/pnl/by-exchange`
  - group PnL ตาม exchange
- [x] `GET /api/v1/pnl/by-component`
  - group PnL ตาม `funding`, `trading_fee`, `trading_pnl`

### Equity API

- [x] `GET /api/v1/equity/latest`
  - temporary live pull จาก Bybit/Bitget exchange API โดยตรง
  - ใช้ credential จาก `backend/.env`
  - มี backend cache สั้นๆ ตาม `EQUITY_CACHE_TTL_SECONDS`
- [x] `GET /api/v1/equity/live`
  - alias ของ `GET /api/v1/equity/latest`
- [ ] `GET /api/v1/equity/curve`
  - ยังทำไม่ได้จนกว่าจะมี `wallet_snapshots`
- [ ] `GET /api/v1/equity/curve/combined`
  - ยังทำไม่ได้จนกว่าจะมี `wallet_snapshots`
- [ ] `GET /api/v1/equity/drawdown`
  - ยังทำไม่ได้จนกว่าจะมี `wallet_snapshots`

### Verified

- [x] รัน `go test ./...` ผ่าน
- [x] ทดสอบ live endpoint กับ database แล้วสำหรับ health, allocation summary, cancelled list, detail และ timeline
- [x] ทดสอบ `GET /api/v1/allocations/cancelled?reason=insufficient_net_edge_after_cost&limit=3` แล้ว query แบบ partial match คืน row ได้
- [x] ทดสอบ live endpoint กับ database แล้วสำหรับ funding latest, funding spread, funding top-spreads, market-quality latest, market-quality history และ market-quality alerts
- [x] รัน `go test ./...` หลังเพิ่ม PnL API ผ่าน
- [x] Live test PnL API กับ database จริงผ่านแล้ว: health ok, PnL endpoints ตอบ 200 แต่ `pnl_events` ยังไม่มี row จึงคืน `count: 0`
- [x] Live test `GET /api/v1/equity/latest?quote=USDT&refresh=true` ผ่าน ได้ snapshot จาก `bitget` และ `bybit`
- [x] ทดสอบ cache ของ equity API ผ่าน: request ถัดไปคืน `cached=true`
- [x] รัน `go test ./...` หลังเพิ่ม System API ผ่าน
- [x] Swagger มี `GET /api/v1/system/status` และ `GET /api/v1/system/exchanges`
- [ ] Live test System API รอบล่าสุดยังติด local DB config: service ฟังที่ port `5435` แต่ credential ที่มีใน `.env` / `D:\platform\.env` auth ไม่ผ่าน

## 1. System / Health API

ใช้สำหรับดูว่า API, database, exchange connector และ trading engine ยังทำงานปกติหรือไม่

### [x] `GET /api/v1/health`

ตรวจว่า API service ยัง alive

Response example:

```json
{
  "status": "ok",
  "time": "2026-04-29T02:00:00Z"
}
```

### [x] `GET /api/v1/system/status`

ดูสถานะรวมของ service สำคัญ

แสดง:

- database connected หรือไม่
- TimescaleDB ใช้งานได้หรือไม่
- exchange ที่รองรับ เช่น `bybit`, `bitget`
- last sync time ของ funding, open interest, market quality
- API version
- allocation total / running count

Data source:

- ตรวจ DB connection โดยตรง
- ตาราง `funding`
- ตาราง `open_interest`
- ตาราง `market_quality_metrics_1m`
- ตาราง `allocations`

Response fields:

- `status`
- `time`
- `api_version`
- `database`
- `timescale`
- `market_data.funding_last_at`
- `market_data.open_interest_last_at`
- `market_data.market_quality_last_at`
- `allocations.total`
- `allocations.running`
- `allocations.last_updated_at`
- `exchanges`

### [x] `GET /api/v1/system/exchanges`

ดู exchange ที่ระบบรองรับและสถานะล่าสุด

Response fields:

- `exchange`
- `enabled`
- `supported`
- `credential_configured`
- `demo`
- `last_market_data_at`
- `last_funding_at`
- `last_open_interest_at`
- `last_market_quality_at`
- `last_account_event_at`
- `last_wallet_snapshot_at`
- `status`
- `notes`

Status rules:

- `ok`: มี market data ล่าสุดและ credential configured
- `degraded`: credential ยังไม่ครบ แต่ยังมี market data
- `stale`: market data เก่ากว่า 10 นาที
- `no_data`: ยังไม่เห็น market data ใน DB

หมายเหตุ: `last_account_event_at` และ `last_wallet_snapshot_at` ยังว่าง เพราะต้องมี persistence เพิ่ม ถ้าต้องการ historical/restart-safe

## 2. Dashboard Summary API

ใช้ทำหน้าแรกของ CRM

### [ ] `GET /api/v1/dashboard/summary`

สรุปภาพรวมระบบเทรด

ควรแสดง:

- total equity รวมทุก exchange
- available balance รวม
- unrealized PnL รวม
- realized PnL รวม
- funding PnL รวม
- active allocations
- running allocations
- cancelled allocations วันนี้
- failed allocations วันนี้
- active alerts

Data source:

- `allocations`
- `pnl_events`
- future table: `wallet_snapshots`
- future table: `alerts`

### [ ] `GET /api/v1/dashboard/live-state`

สถานะ live แบบสั้นสำหรับ auto-refresh

ควรแสดง:

- exchange health
- active allocation count
- current pair exposure
- latest PnL
- latest equity
- stale data flags

### [ ] `GET /api/v1/dashboard/risk`

ดู risk รวมของระบบ

ควรแสดง:

- gross notional
- active budget USD
- pair concentration
- exchange concentration
- stale market data count
- hedge imbalance count
- drawdown หรือ equity drop

Data source:

- `allocations`
- `scaling_plans`
- `order_management_routing_states`
- future table: `wallet_snapshots`
- future table: `position_snapshots`

## 3. Allocation API

เป็น API สำคัญของ CRM เพราะใช้ตอบคำถามว่า allocation มีกี่ตัว run อยู่ state ไหน และ cancel เพราะอะไร

### Allocation status ที่มีใน `d:\platform`

```text
created      = ผ่าน runtime gate แล้ว รอ worker claim
running      = worker claim แล้ว กำลังทำงาน
completed    = worker จบปกติ
failed       = worker หรือ handoff fail อาจถูก retry
cancelled    = ถูก reject/cancel หลัง persist
superseded   = ถูก dedupe เพราะมี active allocation คู่เดียวกัน
paused       = legacy status
```

Data source:

- table: `allocations`
- code: `internal/types/allocation.go`
- repo: `internal/repo/allocation.go`

### [x] `GET /api/v1/allocations/summary`

สรุปจำนวน allocation ตาม state

Query params:

- `from`
- `to`
- `base`
- `quote`
- `role`

Response example:

```json
{
  "total": 128,
  "active": 3,
  "by_status": {
    "created": 1,
    "running": 2,
    "failed": 4,
    "cancelled": 35,
    "completed": 80,
    "superseded": 6
  },
  "active_budget_usd": "1500",
  "cancelled": {
    "total": 35,
    "by_reason": {
      "execution_rejected: insufficient_net_edge_after_cost": 12,
      "sizing_blocked: initial_slice_below_pair_min_notional": 7,
      "missing_market_data": 5
    }
  }
}
```

SQL idea:

```sql
SELECT status, COUNT(*) AS count
FROM allocations
GROUP BY status
ORDER BY status;
```

### [x] `GET /api/v1/allocations`

List allocation แบบ filter ได้

Query params:

- `status`
- `role`
- `base`
- `quote`
- `direction`
- `from`
- `to`
- `cursor`
- `limit`

Response fields:

- `id`
- `base`
- `quote`
- `direction`
- `rank`
- `score`
- `role`
- `status`
- `budget_usd`
- `worker_pid`
- `note`
- `created_at`
- `updated_at`

### [x] `GET /api/v1/allocations/active`

ดู allocation ที่ถือว่ายัง active ในระบบ

Active statuses:

```text
created, running, failed, paused
```

เหตุผลที่ `failed` ยังถือเป็น active ในระบบปัจจุบัน:

- trading engine มี logic sync/retry สำหรับ `failed`
- live budget นับ `failed` เป็น active state ในบาง flow

SQL idea:

```sql
SELECT *
FROM allocations
WHERE status IN ('created', 'running', 'failed', 'paused')
ORDER BY updated_at DESC;
```

### [x] `GET /api/v1/allocations/running`

ดู allocation ที่กำลัง run อยู่จริง

SQL idea:

```sql
SELECT id, base, quote, direction, role, budget_usd, worker_pid, note, created_at, updated_at
FROM allocations
WHERE status = 'running'
ORDER BY updated_at DESC;
```

### [x] `GET /api/v1/allocations/cancelled`

ดู allocation ที่ถูก cancel

Query params:

- `reason`
- `base`
- `quote`
- `from`
- `to`
- `cursor`
- `limit`

หมายเหตุ:

- เหตุผล cancel ปัจจุบันอยู่ใน `allocations.note`
- note อาจเป็น prefix เช่น `execution_rejected: ...` หรือ `sizing_blocked: ...`

### [x] `GET /api/v1/allocations/cancelled/reasons`

สรุป cancel reason

SQL idea:

```sql
SELECT COALESCE(note, 'unknown') AS reason, COUNT(*) AS count
FROM allocations
WHERE status = 'cancelled'
GROUP BY COALESCE(note, 'unknown')
ORDER BY count DESC;
```

### [x] `GET /api/v1/allocations/{id}`

ดูรายละเอียด allocation เดียว

ควรรวม:

- allocation row
- latest scaling plan
- latest recovery decision
- latest order routing state
- progress events
- PnL events ของ pair นั้นในช่วง allocation

### [x] `GET /api/v1/allocations/{id}/timeline`

แสดง timeline ของ allocation ตั้งแต่ created จนจบ

ควรมี event เช่น:

- allocation created
- worker claimed
- execution accepted/rejected
- sizing active/blocked
- order submitted
- order filled/cancelled
- recovery required
- completed/failed/cancelled

Data source ปัจจุบัน:

- `allocations`
- `scaling_plans`
- `recovery_decisions`
- `order_management_routing_states`
- `order_management_progress_events`

Gap:

- ควรเพิ่ม `allocation_events` เพื่อทำ timeline ให้ครบและไม่ต้อง parse จาก note

## 4. Allocation Event API

ควรเพิ่ม table นี้เพื่อให้ CRM เห็น reject/cancel ได้ครบ

เหตุผล:

- runtime reject บางเคสใน trading engine ถูก log แล้ว drop ก่อน insert เข้า `allocations`
- ถ้าไม่มี event table จะนับ cancel/reject ได้ไม่ครบ
- `allocations.note` เหมาะกับ latest note แต่ไม่เหมาะกับ full audit trail

### Recommended table: `allocation_events`

```sql
CREATE TABLE IF NOT EXISTS allocation_events (
    id BIGSERIAL PRIMARY KEY,
    allocation_id BIGINT,
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    base TEXT NOT NULL,
    quote TEXT NOT NULL,
    from_status TEXT NOT NULL DEFAULT '',
    to_status TEXT NOT NULL DEFAULT '',
    stage TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_allocation_events_time
    ON allocation_events (event_time DESC);

CREATE INDEX idx_allocation_events_allocation_time
    ON allocation_events (allocation_id, event_time ASC);

CREATE INDEX idx_allocation_events_reason_time
    ON allocation_events (reason, event_time DESC);
```

### [ ] `GET /api/v1/allocation-events`

List allocation events

Query params:

- `allocation_id`
- `stage`
- `reason`
- `base`
- `quote`
- `from`
- `to`
- `cursor`
- `limit`

### [ ] `GET /api/v1/allocation-events/summary`

สรุปจำนวน event ตาม stage/reason

ควรใช้ตอบ:

- reject เยอะสุดที่ stage ไหน
- allocation ถูก cancel เพราะอะไรเยอะสุด
- วันนี้มี runtime reject กี่ครั้ง

## 5. Instrument / Pair API

ใช้ดู pair ที่ระบบรู้จัก และ metadata สำหรับเทรด

Data source:

- table: `instruments`
- repo: `internal/repo/instrument.go`

### [ ] `GET /api/v1/instruments`

Query params:

- `exchange`
- `base`
- `quote`
- `limit`

Response fields:

- `exchange`
- `base`
- `quote`
- `margin_asset`
- `contract_multiplier`
- `tick_size`
- `lot_size`
- `min_qty`
- `max_qty`
- `min_notional`
- `max_leverage`
- `funding_interval`
- `launch_time`

### [ ] `GET /api/v1/pairs`

List pair รวมข้าม exchange

ควรแสดง:

- `base`
- `quote`
- exchanges ที่มี pair นี้
- metadata completeness
- latest funding spread
- latest OI skew
- latest market quality status

### [ ] `GET /api/v1/pairs/{base}-{quote}/latest`

ดู snapshot ล่าสุดของ pair เดียว

ควรรวม:

- latest funding by exchange
- latest open interest by exchange
- latest market quality by exchange
- active allocation ของ pair นี้ ถ้ามี

## 6. Funding API

ใช้ดู funding rate และ spread ระหว่าง exchange

Data source:

- table: `funding`
- repo: `internal/repo/funding_repo.go`

### [x] `GET /api/v1/funding/latest`

ดู funding ล่าสุด

Query params:

- `exchange`
- `base`
- `quote`
- `limit`

Response fields:

- `time`
- `exchange`
- `base`
- `quote`
- `pair`
- `funding_rate`

### [x] `GET /api/v1/funding/history`

ดู historical funding

Query params:

- `exchange`
- `base`
- `quote`
- `from`
- `to`
- `limit`

Response fields:

- `time`
- `exchange`
- `base`
- `quote`
- `pair`
- `funding_rate`

### [x] `GET /api/v1/funding/spread`

ดู spread ระหว่าง Bybit และ Bitget

Query params:

- `base` required
- `quote` default `USDT`

Current response example:

```json
{
  "data": {
    "time": "2026-04-29T05:30:00Z",
    "base": "BTC",
    "quote": "USDT",
    "pair": "BTCUSDT",
    "bybit_rate": "-0.00000113",
    "bitget_rate": "-0.000103",
    "spread": "0.00010187",
    "spread_bps": "1.01870000",
    "abs_spread_bps": "1.01870000",
    "direction_hint": "short_bybit_long_bitget",
    "bybit_time": "2026-04-29T05:30:00Z",
    "bitget_time": "2026-04-29T05:29:59Z"
  }
}
```

หมายเหตุ: `direction_hint` เป็นมุมมอง trade direction คร่าวๆ จาก funding spread ล่าสุด ไม่ใช่คำสั่งเทรดจริง เพราะยังต้องผ่าน market quality, OI, allocation budget และ execution gate

### [x] `GET /api/v1/funding/top-spreads`

หา pair ที่ funding spread สูงสุด

Query params:

- `quote`
- `base`
- `limit`
- `min_abs_spread_bps`

Response fields:

- `time`
- `base`
- `quote`
- `pair`
- `bybit_rate`
- `bitget_rate`
- `spread`
- `spread_bps`
- `abs_spread_bps`
- `direction_hint`
- `bybit_time`
- `bitget_time`

## 7. Open Interest API

ใช้ดู OI และ crowding risk

Data source:

- table: `open_interest`
- repo: `internal/repo/open_interest.go`

### [ ] `GET /api/v1/open-interest/latest`

ดู OI ล่าสุด

Query params:

- `exchange`
- `base`
- `quote`

### [ ] `GET /api/v1/open-interest/history`

ดู historical OI

Query params:

- `exchange`
- `base`
- `quote`
- `from`
- `to`
- `limit`

### [ ] `GET /api/v1/open-interest/skew`

ดู OI skew ระหว่างสอง exchange

ควรใช้ช่วยอธิบาย crowding:

- exchange ไหน crowded กว่า
- OI เปลี่ยนเร็วผิดปกติหรือไม่
- funding spread น่าเชื่อถือแค่ไหน

## 8. Market Quality API

ใช้ดูคุณภาพตลาดก่อนส่ง order

Data source:

- table: `market_quality_metrics_1m`
- repo: `internal/repo/market_quality_repo.go`

### [x] `GET /api/v1/market-quality/latest`

ดู market quality ล่าสุด

Query params:

- `exchange`
- `base`
- `quote`
- `limit`

Response fields:

- `time`
- `exchange`
- `base`
- `quote`
- `pair`
- `samples`
- `spread_bps_p50`
- `mid_speed_bps_per_sec_p95`
- `depth_stability_ratio`

หมายเหตุ: field ข้างบนคือ field ที่มีจริงใน table ปัจจุบัน ยังไม่มี `spread_bps_p95`, depth notional, quote gap, ticker gap หรือ mark-index deviation ใน `market_quality_metrics_1m`

### [x] `GET /api/v1/market-quality/history`

ดู historical market quality

Query params:

- `exchange`
- `base`
- `quote`
- `from`
- `to`
- `limit`

Response fields:

- `time`
- `exchange`
- `base`
- `quote`
- `pair`
- `samples`
- `spread_bps_p50`
- `mid_speed_bps_per_sec_p95`
- `depth_stability_ratio`

### [x] `GET /api/v1/market-quality/alerts`

หา pair ที่ตลาดไม่ดี

Query params:

- `exchange`
- `base`
- `quote`
- `min_samples`
- `max_spread_bps_p50`
- `max_mid_speed_bps_per_sec_p95`
- `min_depth_stability_ratio`
- `limit`

ตัวอย่าง condition ที่ใช้ตอนนี้:

- sample ต่ำ
- spread p50 สูง
- mid speed p95 สูงผิดปกติ
- depth stability ratio ต่ำ

Default thresholds:

- `min_samples = 10`
- `max_spread_bps_p50 = 10`
- `max_mid_speed_bps_per_sec_p95 = 50`
- `min_depth_stability_ratio = 0.10`

Alert reasons:

- `low_samples`
- `wide_spread`
- `fast_mid_price`
- `unstable_depth`

## 9. Order Routing / Execution Flow API

ใช้ดูว่า allocation ส่ง order อย่างไร และติดตรงไหน

Data source:

- table: `order_management_routing_states`
- table: `order_management_routing_orders`
- table: `order_management_routing_executions`
- table: `order_management_progress_events`
- repo: `internal/repo/order.go`
- repo: `internal/repo/order_progress.go`

### [ ] `GET /api/v1/order-routing/{plan_id}`

ดู routing state ของ plan

ควรแสดง:

- allocation id
- pair
- requested notional
- lead exchange
- follow exchange
- direction
- execution mode
- follow policy
- cancel requested
- cancel reason
- terminal status
- orders
- executions

### [ ] `GET /api/v1/order-routing/{plan_id}/progress`

ดู progress events ของ plan

Response fields:

- `progress_seq`
- `allocation_id`
- `slice_index`
- `status`
- `submitted_notional`
- `funding_filled_delta_notional`
- `hedge_filled_delta_notional`
- `reason`
- `occurred_at`

### [ ] `GET /api/v1/orders`

List order จาก routing table

Query params:

- `exchange`
- `base`
- `quote`
- `status`
- `plan_id`
- `allocation_id`

หมายเหตุ:

- table ปัจจุบันเก็บ order routing ไม่ใช่ full exchange order history
- ถ้าต้องการ trade log จริง ควรเพิ่ม `account_events` หรือ `trade_events`

## 10. Trade Log / Account Event API

ใช้ดู order, execution, position แบบ normalized

ใน code มี struct แล้ว:

- `types.Order`
- `types.Execution`
- `types.Position`
- `types.AccountEvent`

แต่ยังไม่เห็น table สำหรับ persist account events โดยตรง

### Recommended table: `account_events`

```sql
CREATE TABLE IF NOT EXISTS account_events (
    id BIGSERIAL PRIMARY KEY,
    event_time TIMESTAMPTZ NOT NULL,
    exchange TEXT NOT NULL,
    event_type TEXT NOT NULL,
    base TEXT NOT NULL DEFAULT '',
    quote TEXT NOT NULL DEFAULT '',
    event_id TEXT NOT NULL DEFAULT '',
    order_id TEXT NOT NULL DEFAULT '',
    client_order_id TEXT NOT NULL DEFAULT '',
    exec_id TEXT NOT NULL DEFAULT '',
    side TEXT NOT NULL DEFAULT '',
    trade_side TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    qty NUMERIC NOT NULL DEFAULT 0,
    price NUMERIC NOT NULL DEFAULT 0,
    value NUMERIC NOT NULL DEFAULT 0,
    fee NUMERIC NOT NULL DEFAULT 0,
    pnl NUMERIC NOT NULL DEFAULT 0,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_account_events_time
    ON account_events (event_time DESC);

CREATE INDEX idx_account_events_exchange_pair_time
    ON account_events (exchange, base, quote, event_time DESC);

CREATE INDEX idx_account_events_order_id
    ON account_events (order_id)
    WHERE order_id <> '';
```

### [ ] `GET /api/v1/account-events`

List normalized account events

Query params:

- `exchange`
- `event_type`
- `base`
- `quote`
- `order_id`
- `exec_id`
- `from`
- `to`
- `cursor`
- `limit`

### [ ] `GET /api/v1/trades`

Trade log สำหรับ CRM

ควรดึงจาก execution events

Response fields:

- `time`
- `exchange`
- `pair`
- `side`
- `trade_side`
- `price`
- `qty`
- `value`
- `fee`
- `fee_currency`
- `pnl`
- `order_id`
- `exec_id`

### [ ] `GET /api/v1/positions`

ดู position events หรือ latest positions

ควรมี:

- `exchange`
- `pair`
- `side`
- `size`
- `entry_price`
- `mark_price`
- `unrealized_pnl`
- `realized_pnl`
- `funding_pnl`
- `position_value`
- `is_closed`

## 11. PnL API

ใช้ดู PnL แยก component

Data source:

- table: `pnl_events`
- repo: `internal/repo/pnl_repo.go`

Environment note:

- CRM PnL API ใช้ `DATABASE_URL` จาก `.env` / `.env.example` เพื่ออ่าน Postgres
- ค่า `BITGET_CREDENTIAL_*` และ `BYBIT_CREDENTIAL_*` ใน `.env.example` ไม่ถูกใช้ใน endpoint ชุดนี้ เพราะ API เป็น read-only จาก `pnl_events`

Components ปัจจุบัน:

```text
funding
trading_fee
trading_pnl
```

### [x] `GET /api/v1/pnl/events`

List PnL events

Query params:

- `exchange`
- `base`
- `quote`
- `component`
- `source_type`
- `source_id`
- `from`
- `to`
- `limit`

Response fields:

- `event_time`
- `exchange`
- `base`
- `quote`
- `pair`
- `component`
- `amount`
- `currency`
- `source_type`
- `source_id`
- `created_at`

### [x] `GET /api/v1/pnl/summary`

สรุป PnL รวม

Query params:

- `exchange`
- `base`
- `quote`
- `component`
- `source_type`
- `source_id`
- `from`
- `to`
- `limit`

Response example:

```json
{
  "count": 18,
  "total_amount": "12.35",
  "funding_amount": "18.20",
  "trading_fee_amount": "-2.10",
  "trading_pnl_amount": "-3.75",
  "by_component": [],
  "by_exchange": [],
  "by_pair": []
}
```

### [x] `GET /api/v1/pnl/by-pair`

ดู PnL ราย pair

Group response fields:

- `base`
- `quote`
- `pair`
- `count`
- `total_amount`
- `funding_amount`
- `trading_fee_amount`
- `trading_pnl_amount`
- `first_event_time`
- `last_event_time`

### [x] `GET /api/v1/pnl/by-exchange`

ดู PnL ราย exchange

Group response fields:

- `exchange`
- `count`
- `total_amount`
- `funding_amount`
- `trading_fee_amount`
- `trading_pnl_amount`
- `first_event_time`
- `last_event_time`

### [x] `GET /api/v1/pnl/by-component`

ดู PnL แยก funding / trading fee / trading pnl

Group response fields:

- `component`
- `count`
- `total_amount`
- `funding_amount`
- `trading_fee_amount`
- `trading_pnl_amount`
- `first_event_time`
- `last_event_time`

หมายเหตุด้านความถูกต้อง:

- funding PnL ต้องแยกจาก trading PnL
- Bybit funding ควรมาจาก transaction log type `SETTLEMENT`
- Bitget closed funding ใช้ `settleFee`
- Bitget live funding ใช้ delta ของ `totalFee`

## 12. Equity Curve API

ใช้ดู equity curve รวมสอง exchange

Gap ปัจจุบัน:

- code มี `WalletBalance` struct
- portfolio sync wallet ทุก 1 นาที
- แต่ยังไม่เห็น table persist wallet snapshots
- ถ้าต้องการ historical equity curve ต้องเพิ่ม table

Temporary CRM implementation:

- `GET /api/v1/equity/latest`
- `GET /api/v1/equity/live`
- data source: direct Bybit/Bitget exchange API
- config source: `backend/.env`
- cache: `EQUITY_CACHE_TTL_SECONDS`
- historical curve ยังทำไม่ได้ เพราะยังไม่ persist ลง DB

Query params:

- `exchange` เช่น `bybit`, `bitget` หรือ comma-separated
- `quote` default `USDT`
- `refresh=true` เพื่อ bypass cache แล้วดึง exchange สด

Response fields:

- `data[].exchange`
- `data[].time`
- `data[].account_equity`
- `data[].wallet_balance`
- `data[].available_balance`
- `data[].unrealized_pnl`
- `data[].initial_margin`
- `data[].maintenance_margin`
- `data[].coins`
- `combined`
- `errors`
- `cache_ttl_seconds`

### Recommended table: `wallet_snapshots`

```sql
CREATE TABLE IF NOT EXISTS wallet_snapshots (
    time TIMESTAMPTZ NOT NULL,
    exchange TEXT NOT NULL,
    account_equity NUMERIC NOT NULL DEFAULT 0,
    wallet_balance NUMERIC NOT NULL DEFAULT 0,
    available_balance NUMERIC NOT NULL DEFAULT 0,
    unrealized_pnl NUMERIC NOT NULL DEFAULT 0,
    initial_margin NUMERIC NOT NULL DEFAULT 0,
    maintenance_margin NUMERIC NOT NULL DEFAULT 0,
    payload JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (time, exchange)
);

SELECT create_hypertable('wallet_snapshots', 'time', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_wallet_snapshots_exchange_time
    ON wallet_snapshots (exchange, time DESC);
```

### [x] `GET /api/v1/equity/latest`

ดู equity ล่าสุดราย exchange และรวม

Response example:

```json
{
  "time": "2026-04-29T02:00:00Z",
  "total_equity": "10500.25",
  "exchanges": [
    {
      "exchange": "bybit",
      "account_equity": "5200.10",
      "available_balance": "4100.00",
      "unrealized_pnl": "12.50"
    },
    {
      "exchange": "bitget",
      "account_equity": "5300.15",
      "available_balance": "4200.00",
      "unrealized_pnl": "-3.25"
    }
  ]
}
```

### [x] `GET /api/v1/equity/live`

Alias ของ `GET /api/v1/equity/latest` สำหรับช่วง temporary live pull จาก exchange API

### [ ] `GET /api/v1/equity/curve`

ดู equity curve ราย exchange

Query params:

- `exchange`
- `from`
- `to`
- `interval`

### [ ] `GET /api/v1/equity/curve/combined`

ดู equity curve รวมสอง exchange

Logic:

- group snapshot ตามเวลา
- sum `account_equity` ของทุก exchange
- ควร bucket เวลา เช่น 1m, 5m, 1h, 1d

### [ ] `GET /api/v1/equity/drawdown`

ดู max drawdown จาก equity curve

ควรใช้สำหรับ risk dashboard

## 13. Alert / Incident API

ใช้เก็บเหตุการณ์ผิดปกติที่ CRM ต้องโชว์

ควรเพิ่ม table `alerts`

### Recommended table: `alerts`

```sql
CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    reason TEXT NOT NULL,
    message TEXT NOT NULL,
    allocation_id BIGINT,
    exchange TEXT NOT NULL DEFAULT '',
    base TEXT NOT NULL DEFAULT '',
    quote TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_alerts_status_created_at
    ON alerts (status, created_at DESC);
```

### [ ] `GET /api/v1/alerts`

List alerts

Query params:

- `status`
- `severity`
- `source`
- `reason`
- `from`
- `to`

### [ ] `GET /api/v1/alerts/active`

ดู alert ที่ยังไม่ resolve

### [ ] `POST /api/v1/alerts/{id}/ack`

acknowledge alert

### [ ] `POST /api/v1/alerts/{id}/resolve`

resolve alert

Alert ที่ควรมี:

- market data stale
- exchange disconnected
- account listener degraded
- allocation cancelled
- allocation failed
- order submit failed
- hedge imbalance
- funding flip
- position alignment broken
- equity drawdown
- market quality degraded

## 14. Realtime API

ใช้ส่งข้อมูลไป frontend แบบ live

### [ ] `GET /api/v1/sse/events`

Server-Sent Events สำหรับ dashboard

Event types:

- `allocation.updated`
- `allocation.cancelled`
- `order.updated`
- `execution.created`
- `position.updated`
- `pnl.updated`
- `equity.updated`
- `alert.created`
- `system.status_changed`

### [ ] `GET /api/v1/ws`

WebSocket สำหรับ realtime monitoring

เหมาะเมื่อ frontend ต้อง subscribe หลาย channel

Subscribe message example:

```json
{
  "type": "subscribe",
  "channels": [
    "allocations",
    "orders",
    "equity",
    "alerts"
  ]
}
```

## 15. Recommended MVP Build Order

### Phase 1: Read-only API จาก table ที่มีแล้ว

ทำก่อนได้ทันทีโดยไม่แก้ `d:\platform` มาก

- [x] `GET /api/v1/health`
- [ ] `GET /api/v1/dashboard/summary`
- [x] `GET /api/v1/allocations/summary`
- [x] `GET /api/v1/allocations`
- [x] `GET /api/v1/allocations/active`
- [x] `GET /api/v1/allocations/running`
- [x] `GET /api/v1/allocations/cancelled/reasons`
- [x] `GET /api/v1/funding/latest`
- [x] `GET /api/v1/funding/history`
- [x] `GET /api/v1/funding/spread`
- [x] `GET /api/v1/funding/top-spreads`
- [ ] `GET /api/v1/open-interest/latest`
- [x] `GET /api/v1/market-quality/latest`
- [x] `GET /api/v1/market-quality/history`
- [x] `GET /api/v1/market-quality/alerts`
- [x] `GET /api/v1/pnl/events`
- [x] `GET /api/v1/pnl/summary`
- [x] `GET /api/v1/pnl/by-pair`
- [x] `GET /api/v1/pnl/by-exchange`
- [x] `GET /api/v1/pnl/by-component`

### Phase 2: เพิ่ม persistence ที่ CRM ต้องใช้

เพิ่ม table:

- [ ] `allocation_events`
- [ ] `wallet_snapshots`
- [ ] `account_events`
- [ ] `alerts`

### Phase 3: ทำ timeline และ equity curve

- [x] `GET /api/v1/allocations/{id}/timeline`
- [ ] `GET /api/v1/equity/latest`
- [ ] `GET /api/v1/equity/curve`
- [ ] `GET /api/v1/equity/curve/combined`
- [ ] `GET /api/v1/trades`
- [ ] `GET /api/v1/positions`

### Phase 4: Realtime

- [ ] `GET /api/v1/sse/events`
- [ ] `GET /api/v1/ws`

## 16. Backend Implementation Notes

แนะนำทำ API service เป็น read-only service ก่อน

เหตุผล:

- ลดความเสี่ยงกับ trading engine
- CRM ไม่ควรเปลี่ยน state การเทรดใน phase แรก
- debug ง่าย เพราะอ่านจาก Postgres/TimescaleDB

Suggested Go structure:

```text
backend/
  cmd/
    api/
      main.go
  internal/
    config/
    database/
    handler/
    repo/
    response/
  docs/
```

โครงสร้างปัจจุบันใช้ `internal/response` แทนชื่อ `dto` เพื่อให้ชื่ออ่านง่ายขึ้นสำหรับ response JSON ของ CRM

Current implemented files:

- `backend/internal/handler/allocation_handler.go`
- `backend/internal/handler/funding_handler.go`
- `backend/internal/handler/market_quality_handler.go`
- `backend/internal/handler/pnl_handler.go`
- `backend/internal/handler/equity_handler.go`
- `backend/internal/exchange/equity.go`
- `backend/internal/repo/allocation_repo.go`
- `backend/internal/repo/funding_repo.go`
- `backend/internal/repo/market_quality_repo.go`
- `backend/internal/repo/pnl_repo.go`
- `backend/internal/repo/time_series.go`
- `backend/internal/response/allocation.go`
- `backend/internal/response/market_data.go`
- `backend/internal/response/pnl.go`
- `backend/internal/response/equity.go`

สำคัญ:

- ใช้ `decimal.Decimal` สำหรับเงินและ notional
- response JSON ควรส่ง decimal เป็น string เพื่อลดปัญหา precision
- ทุก endpoint ที่เป็น list ต้องมี `limit`
- historical endpoint ต้องมี `from` / `to`
- dashboard endpoint ต้องระวัง query หนักเกินไป ควร aggregate หรือ cache

## 17. Known Gaps จาก `d:\platform`

### Gap 1: Runtime reject ก่อน persist

บาง allocation ถูก reject ก่อน insert เข้า `allocations`

ผลกระทบ:

- CRM นับ cancel/reject ได้ไม่ครบ

วิธีแก้:

- เพิ่ม `allocation_events`
- เขียน event ทุกครั้งที่ runtime reject

### Gap 2: Equity curve ยังไม่มี historical wallet table

ผลกระทบ:

- ทำ equity curve รวม 2 exchange ย้อนหลังไม่ได้ครบ

วิธีแก้:

- เพิ่ม `wallet_snapshots`
- persist ตอน sync wallet balance ทุก 1 นาที

### Gap 3: Trade log ยังไม่มี account event table

ผลกระทบ:

- ดู order/execution/position history แบบ CRM ยังไม่ครบ

วิธีแก้:

- เพิ่ม `account_events`
- persist normalized `types.AccountEvent`

### Gap 4: Alert ยังไม่มี table

ผลกระทบ:

- frontend ต้องเดา alert จากหลาย table

วิธีแก้:

- เพิ่ม `alerts`
- ให้ trading engine / CRM job สร้าง alert จาก condition สำคัญ

### Gap 5: Market quality table ยังมี metric ไม่ครบ

ตอนนี้ `market_quality_metrics_1m` มี field ที่ CRM อ่านได้จริงคือ:

- `samples`
- `spread_bps_p50`
- `mid_speed_bps_per_sec_p95`
- `depth_stability_ratio`

ผลกระทบ:

- ยังทำ alert จาก `spread_bps_p95`, top-book depth, quote gap, ticker gap หรือ mark-index deviation ไม่ได้โดยตรง
- Market quality alert ตอนนี้จึงเป็น threshold แบบง่ายจาก field ที่มีจริงก่อน

วิธีแก้:

- เพิ่ม column หรือ table สำหรับ metric ที่ต้องใช้เพิ่ม
- persist `spread_bps_p95`, `top_book_depth_notional`, `quote_gap_sec`, `ticker_gap_sec`, `mark_index_deviation_bps`
- หลังมี field แล้วค่อยขยาย `GET /api/v1/market-quality/alerts` ให้ใช้ condition ครบขึ้น
