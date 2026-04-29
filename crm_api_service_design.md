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
- [ ] `GET /api/v1/system/status`
- [ ] `GET /api/v1/system/exchanges`

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

- [ ] `GET /api/v1/funding/latest`
- [ ] `GET /api/v1/funding/history`
- [ ] `GET /api/v1/funding/spread`
- [ ] `GET /api/v1/funding/top-spreads`

### Open Interest API

- [ ] `GET /api/v1/open-interest/latest`
- [ ] `GET /api/v1/open-interest/history`
- [ ] `GET /api/v1/open-interest/skew`

### Market Quality API

- [ ] `GET /api/v1/market-quality/latest`
- [ ] `GET /api/v1/market-quality/history`
- [ ] `GET /api/v1/market-quality/alerts`

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

- [ ] `GET /api/v1/pnl/events`
- [ ] `GET /api/v1/pnl/summary`
- [ ] `GET /api/v1/pnl/by-pair`
- [ ] `GET /api/v1/pnl/by-exchange`
- [ ] `GET /api/v1/pnl/by-component`

### Equity Curve API

- [ ] Create table `wallet_snapshots`
- [ ] `GET /api/v1/equity/latest`
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

### [ ] `GET /api/v1/system/status`

ดูสถานะรวมของ service สำคัญ

ควรแสดง:

- database connected หรือไม่
- TimescaleDB ใช้งานได้หรือไม่
- exchange ที่รองรับ เช่น `bybit`, `bitget`
- last sync time ของ funding, open interest, market quality
- API version

Data source:

- ตรวจ DB connection โดยตรง
- ตาราง `funding`
- ตาราง `open_interest`
- ตาราง `market_quality_metrics_1m`

### [ ] `GET /api/v1/system/exchanges`

ดู exchange ที่ระบบรองรับและสถานะล่าสุด

Response fields:

- `exchange`
- `enabled`
- `last_market_data_at`
- `last_account_event_at`
- `last_wallet_snapshot_at`
- `status`

หมายเหตุ: `last_account_event_at` และ `last_wallet_snapshot_at` ต้องมี persistence เพิ่ม ถ้าต้องการ historical/restart-safe

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
- repo: `internal/repo/funding.go`

### [ ] `GET /api/v1/funding/latest`

ดู funding ล่าสุด

Query params:

- `exchange`
- `base`
- `quote`

### [ ] `GET /api/v1/funding/history`

ดู historical funding

Query params:

- `exchange`
- `base`
- `quote`
- `from`
- `to`
- `limit`

### [ ] `GET /api/v1/funding/spread`

ดู spread ระหว่าง Bybit และ Bitget

Response idea:

```json
{
  "base": "BTC",
  "quote": "USDT",
  "bybit_rate": "0.0001",
  "bitget_rate": "-0.0002",
  "spread": "0.0003",
  "direction_hint": "long_bitget_short_bybit",
  "time": "2026-04-29T02:00:00Z"
}
```

### [ ] `GET /api/v1/funding/top-spreads`

หา pair ที่ funding spread สูงสุด

Query params:

- `quote`
- `limit`
- `min_abs_spread_bps`

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
- repo: `internal/repo/market_quality_metric.go`

### [ ] `GET /api/v1/market-quality/latest`

ดู market quality ล่าสุด

Query params:

- `exchange`
- `base`
- `quote`

Response fields:

- `samples`
- `spread_bps_p50`
- `spread_bps_p95`
- `top_book_depth_notional_p05`
- `top_book_depth_notional_p50`
- `quote_gap_sec_p95`
- `ticker_gap_sec_p95`
- `mark_index_deviation_bps_p95`
- `mid_speed_bps_per_sec_p95`
- `depth_stability_ratio`

### [ ] `GET /api/v1/market-quality/history`

ดู historical market quality

Query params:

- `exchange`
- `base`
- `quote`
- `from`
- `to`
- `limit`

### [ ] `GET /api/v1/market-quality/alerts`

หา pair ที่ตลาดไม่ดี

ตัวอย่าง condition:

- spread p95 สูง
- depth ต่ำ
- quote gap สูง
- ticker gap สูง
- mark-index deviation สูง
- mid speed สูงผิดปกติ

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
- repo: `internal/repo/pnl_event.go`

Components ปัจจุบัน:

```text
funding
trading_fee
trading_pnl
```

### [ ] `GET /api/v1/pnl/events`

List PnL events

Query params:

- `exchange`
- `base`
- `quote`
- `component`
- `from`
- `to`

### [ ] `GET /api/v1/pnl/summary`

สรุป PnL รวม

ควร group ได้ตาม:

- exchange
- pair
- component
- day
- allocation id ถ้ามีในอนาคต

Response example:

```json
{
  "from": "2026-04-29T00:00:00Z",
  "to": "2026-04-29T23:59:59Z",
  "total": "12.35",
  "by_component": {
    "funding": "18.20",
    "trading_fee": "-2.10",
    "trading_pnl": "-3.75"
  },
  "by_exchange": {
    "bybit": "6.50",
    "bitget": "5.85"
  }
}
```

### [ ] `GET /api/v1/pnl/by-pair`

ดู PnL ราย pair

### [ ] `GET /api/v1/pnl/by-exchange`

ดู PnL ราย exchange

### [ ] `GET /api/v1/pnl/by-component`

ดู PnL แยก funding / trading fee / trading pnl

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

### [ ] `GET /api/v1/equity/latest`

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
- [ ] `GET /api/v1/funding/latest`
- [ ] `GET /api/v1/funding/history`
- [ ] `GET /api/v1/open-interest/latest`
- [ ] `GET /api/v1/market-quality/latest`
- [ ] `GET /api/v1/pnl/events`
- [ ] `GET /api/v1/pnl/summary`

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
cmd/crm-api/main.go
internal/crmapi/handler
internal/crmapi/service
internal/crmapi/repo
internal/crmapi/dto
internal/crmapi/httpserver
```

แนะนำใช้ repo เดิมจาก `internal/repo` เท่าที่ใช้ได้ และเพิ่ม query เฉพาะ CRM ใน package ใหม่

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
