# 🖥️ 後端模組

---

## BE-01｜API Gateway / HTTP Layer

**模式**：REST 入口、中介軟體層、統一請求/回應處理

### 細部功能
- 統一 request_id / correlation_id 注入
- 統一錯誤 envelope（error.code, error.message, error.requestId）
- JWT / session 驗證 middleware
- CORS、CSRF、Rate Limit middleware
- OpenTelemetry trace 注入
- DTO binding 與驗證

### 邊界個案
- JWT 簽名錯誤 → 401，不洩漏簽名細節
- Rate limit 達上限 → 429 並回傳 Retry-After header
- 請求 body 格式錯誤 → 400 with field-level validation error
- Middleware panic → Recovery middleware 捕捉，回 500 並記 log

### 資料結構（Go）

```go
// 統一錯誤 envelope
type ErrorEnvelope struct {
    Error ErrorBody `json:"error"`
}

type ErrorBody struct {
    Code      string          `json:"code"`
    Message   string          `json:"message"`
    Details   json.RawMessage `json:"details,omitempty"`
    RequestID string          `json:"requestId"`
}

// 通用 Response Wrapper
type Response[T any] struct {
    Data T `json:"data"`
}

// Common Headers 解析
type CommonHeaders struct {
    Authorization  string `header:"Authorization"`
    IdempotencyKey string `header:"Idempotency-Key"`
    ClientVersion  string `header:"X-Client-Version"`
    ReqTimezone    string `header:"X-Request-Timezone"`
    IfMatchVersion int    `header:"If-Match-Version"`
}
```

---

## BE-02｜Auth

**模式**：Session 管理、帳號驗證、Invite Token、LLM Key 加密存取

### 細部功能
- Email 登入與 OAuth 擴充
- Refresh token rotation
- Invite token 驗證（含 expiry / single-use 控制）
- LLM provider API key 加密存放（KMS / Secret Manager）
- RBAC 權限集中檢查

### 邊界個案
- Refresh token 重放攻擊 → 偵測 token reuse，撤銷整個 session family
- Invite token 過期 → 回傳 410 Gone
- LLM key 加密失敗 → 不存入，回 500 並告警
- 同帳號並發登入 → session 互不干擾

### 資料結構（DB Schema）

```sql
-- sessions table
CREATE TABLE sessions (
  id                UUID PRIMARY KEY,
  user_id           UUID NOT NULL REFERENCES users(id),
  refresh_token_hash TEXT NOT NULL UNIQUE,
  family_id         UUID NOT NULL,          -- for rotation family tracking
  is_revoked        BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at        TIMESTAMPTZ NOT NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at      TIMESTAMPTZ
);

-- llm_provider_configs table
CREATE TABLE llm_provider_configs (
  id                      UUID PRIMARY KEY,
  user_id                 UUID NOT NULL REFERENCES users(id),
  provider                TEXT NOT NULL,    -- openai | anthropic | google | custom
  label                   TEXT NOT NULL,
  encrypted_key           TEXT NOT NULL,    -- AES-256-GCM encrypted
  encrypted_key_kms_ref   TEXT,             -- KMS key reference
  base_url                TEXT,
  model                   TEXT NOT NULL,
  is_active               BOOLEAN NOT NULL DEFAULT TRUE,
  last_validated_at       TIMESTAMPTZ,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## BE-03｜User / Preference

**模式**：使用者資料 CRUD、旅遊偏好管理、AI Prompt 資料源

### 細部功能
- Profile CRUD
- 偏好設定：旅行步調 / 起床習慣 / 交通偏好 / 食物偏好 / 避開標籤
- Preference version 追蹤
- 區分 explicit（用戶設定）vs inferred（系統推論）偏好
- 偏好資料作為 AI prompt context 的組裝來源

### 邊界個案
- locale / timezone 不合法 → 400 validation error
- Preference 版本衝突 → optimistic lock 409
- 刪除帳號時 → 清除個人偏好與 LLM key，保留旅程 audit log

### 資料結構（DB Schema）

```sql
CREATE TABLE users (
  id               UUID PRIMARY KEY,
  email            CITEXT NOT NULL UNIQUE,
  display_name     TEXT NOT NULL,
  locale           TEXT NOT NULL DEFAULT 'zh-TW',
  timezone         TEXT NOT NULL DEFAULT 'Asia/Taipei',
  default_currency CHAR(3) NOT NULL DEFAULT 'TWD',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at       TIMESTAMPTZ
);

CREATE TABLE user_preferences (
  id                    UUID PRIMARY KEY,
  user_id               UUID NOT NULL REFERENCES users(id) UNIQUE,
  explicit_preferences  JSONB NOT NULL DEFAULT '{}',
  -- { tripPace, wakePattern, transportPreference, foodPreference[], avoidTags[] }
  inferred_preferences  JSONB NOT NULL DEFAULT '{}',
  version               INT NOT NULL DEFAULT 1,
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```go
// Go struct
type UserPreference struct {
    TripPace            string   `json:"tripPace"`   // relaxed | balanced | packed
    WakePattern         string   `json:"wakePattern"` // early | normal | late
    TransportPreference string   `json:"transportPreference"` // walk | transit | taxi | mixed
    FoodPreference      []string `json:"foodPreference"`
    AvoidTags           []string `json:"avoidTags"`
}
```

---

## BE-04｜Workspace / Membership

**模式**：Trip 成員管理、角色控制、邀請與分享連結

### 細部功能
- 建立與撤銷 invite
- Role change
- 成員清單查詢
- Public share token 建立（read-only）
- Share link revoke
- 確保每個 trip 至少 1 位 owner

### 邊界個案
- 嘗試降級自己為 viewer（最後一位 owner）→ 阻擋並回 400
- Share link 已被 revoke → 403
- 大量成員同時接受邀請 → idempotency 保護
- 邀請同一 email 兩次 → 回傳現有邀請，不重複建立

### 資料結構（DB Schema）

```sql
CREATE TABLE trip_memberships (
  id         UUID PRIMARY KEY,
  trip_id    UUID NOT NULL REFERENCES trips(id),
  user_id    UUID NOT NULL REFERENCES users(id),
  role       TEXT NOT NULL CHECK (role IN ('owner','editor','commenter','viewer')),
  status     TEXT NOT NULL DEFAULT 'active',
  joined_at  TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(trip_id, user_id)
);

CREATE TABLE trip_invitations (
  id                  UUID PRIMARY KEY,
  trip_id             UUID NOT NULL REFERENCES trips(id),
  invited_by_user_id  UUID NOT NULL REFERENCES users(id),
  invitee_email       CITEXT NOT NULL,
  role                TEXT NOT NULL,
  token_hash          TEXT NOT NULL UNIQUE,  -- sha256 of raw token
  status              TEXT NOT NULL DEFAULT 'pending',
  expires_at          TIMESTAMPTZ NOT NULL,
  accepted_at         TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE share_links (
  id           UUID PRIMARY KEY,
  trip_id      UUID NOT NULL REFERENCES trips(id),
  token_hash   TEXT NOT NULL UNIQUE,
  access_scope TEXT NOT NULL DEFAULT 'read_only',
  expires_at   TIMESTAMPTZ,
  revoked_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## BE-05｜Trip

**模式**：旅程聚合根管理、跨模組協調、版本控制

### 細部功能
- 建立 / 更新 trip（含 If-Match-Version 樂觀鎖）
- 建立 trip 時自動產生 day skeleton
- Trip lifecycle 狀態機：draft → active → archived
- Timezone 與 date range 一致性驗證
- Trip version bump
- Destination metadata 寫入

### 邊界個案
- Date range 縮短導致 itinerary item 越界 → 回傳衝突清單，需確認處理
- 版本衝突（If-Match-Version 不符）→ 409 Conflict
- Trip 封存後嘗試編輯 → 403 Forbidden
- 建立 trip 帶重複 Idempotency-Key → 回傳已存在的 trip，不重複建立

### 資料結構（DB Schema）

```sql
CREATE TABLE trips (
  id               UUID PRIMARY KEY,
  owner_user_id    UUID NOT NULL REFERENCES users(id),
  name             TEXT NOT NULL,              -- maxLength: 200
  destination_text TEXT,
  start_date       DATE NOT NULL,
  end_date         DATE NOT NULL,
  timezone         TEXT NOT NULL,
  currency         CHAR(3) NOT NULL,
  travelers_count  INT NOT NULL CHECK (travelers_count > 0 AND travelers_count <= 50),
  status           TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','archived')),
  version          INT NOT NULL DEFAULT 1,
  tags             JSONB DEFAULT '[]',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  archived_at      TIMESTAMPTZ
);

CREATE INDEX idx_trips_owner_user_id ON trips(owner_user_id);
CREATE INDEX idx_trips_status ON trips(status);
```

---

## BE-06｜Itinerary

**模式**：行程 CRUD、Bulk Reorder、時間衝突驗證、樂觀鎖

### 細部功能
- Item CRUD（title, itemType, startAt, endAt, allDay, note, placeId, lat/lng, estimatedCost）
- Bulk reorder（transaction 保證排序一致性）
- 跨日移動（同時更新 source day + target day sort index）
- 時間重疊驗證（可配置為 warning 或 block）
- Row-level locking + version check
- Route snapshot 綁定

### 交易原則
- 單 item 更新：row-level locking + version check
- Bulk reorder：transaction
- 跨日移動：transaction 更新兩個 day

### 邊界個案
- 兩位 editor 同時改同 item → 一方 409 version conflict
- Bulk reorder 中途 DB 失敗 → transaction rollback
- item 時間設定在 trip 範圍外 → 400 ITINERARY_INVALID_DATE_RANGE
- endAt < startAt → 400

### 資料結構（DB Schema）

```sql
CREATE TABLE itinerary_days (
  id          UUID PRIMARY KEY,
  trip_id     UUID NOT NULL REFERENCES trips(id),
  trip_date   DATE NOT NULL,
  day_index   INT NOT NULL,
  sort_order  INT NOT NULL,
  version     INT NOT NULL DEFAULT 1,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(trip_id, trip_date)
);

CREATE TABLE itinerary_items (
  id                       UUID PRIMARY KEY,
  trip_id                  UUID NOT NULL REFERENCES trips(id),
  day_id                   UUID NOT NULL REFERENCES itinerary_days(id),
  title                    TEXT NOT NULL,
  item_type                TEXT NOT NULL CHECK (item_type IN (
                             'place_visit','meal','transit','hotel','free_time','custom')),
  start_at                 TIMESTAMPTZ,
  end_at                   TIMESTAMPTZ,
  all_day                  BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order               INT NOT NULL,
  note                     TEXT,
  provider_place_id        TEXT,
  lat                      NUMERIC(10,7),
  lng                      NUMERIC(10,7),
  place_snapshot_id        UUID REFERENCES place_snapshots(id),
  route_snapshot_id        UUID REFERENCES route_snapshots(id),
  estimated_cost_amount    NUMERIC(12,2),
  estimated_cost_currency  CHAR(3),
  source_type              TEXT DEFAULT 'manual', -- manual | ai_draft
  source_draft_id          UUID,
  version                  INT NOT NULL DEFAULT 1,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at               TIMESTAMPTZ,
  CONSTRAINT chk_time_order CHECK (end_at IS NULL OR start_at IS NULL OR end_at >= start_at)
);
```

---

## BE-07｜Budget

**模式**：預算配置、成本估算、實際支出、幣別快照

### 細部功能
- Budget profile CRUD
- Cost estimate line items
- Actual expense records
- Currency conversion snapshot（含匯率來源與 timestamp）
- Over-budget rule evaluation（>10% warning / >20% invalid for AI draft）

### 邊界個案
- 幣別不在支援清單 → 400 BUDGET_INVALID_CURRENCY
- 匯率 API 失敗 → 保留上次快照，標示資料日期，不阻擋操作
- 同時新增多筆支出 → 各自帶 Idempotency-Key

### 資料結構（DB Schema）

```sql
CREATE TABLE budget_profiles (
  id               UUID PRIMARY KEY,
  trip_id          UUID NOT NULL REFERENCES trips(id) UNIQUE,
  total_budget     NUMERIC(14,2),
  per_person_budget NUMERIC(14,2),
  per_day_budget   NUMERIC(14,2),
  currency         CHAR(3) NOT NULL,
  category_plan    JSONB NOT NULL DEFAULT '[]',
  -- [{ category, plannedAmount }]
  version          INT NOT NULL DEFAULT 1,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE expenses (
  id                  UUID PRIMARY KEY,
  trip_id             UUID NOT NULL REFERENCES trips(id),
  created_by_user_id  UUID NOT NULL REFERENCES users(id),
  category            TEXT NOT NULL CHECK (category IN (
                        'lodging','transit','food','attraction','shopping','misc')),
  amount              NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
  currency            CHAR(3) NOT NULL,
  expense_at          TIMESTAMPTZ,
  note                TEXT,
  linked_item_id      UUID REFERENCES itinerary_items(id),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at          TIMESTAMPTZ
);
```

---

## BE-08｜Place / Map

**模式**：Provider Adapter 包裝、地點搜尋、路線估算、外部 API 保護

### 細部功能
- Provider adapter interface（支援多供應商切換）
- Geocode / reverse geocode
- Place detail normalization（標準化 DTO）
- Route estimate normalization
- Provider error handling / fallback
- External API quota 保護（rate limit + circuit breaker）

### 邊界個案
- Provider 回傳缺少必填欄位 → Partial warning，不崩潰
- Provider timeout → MAP_PROVIDER_TIMEOUT
- Quota 耗盡 → MAP_PROVIDER_QUOTA_EXCEEDED，告警
- 切換 provider 後座標系偏差 → mapping layer 統一處理

### 資料結構（DB Schema + Go Interface）

```sql
CREATE TABLE place_snapshots (
  id                     UUID PRIMARY KEY,
  provider               TEXT NOT NULL,
  provider_place_id      TEXT NOT NULL,
  name                   TEXT NOT NULL,
  address                TEXT,
  lat                    NUMERIC(10,7) NOT NULL,
  lng                    NUMERIC(10,7) NOT NULL,
  categories             JSONB DEFAULT '[]',
  opening_hours          JSONB,
  raw_normalized_payload JSONB,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(provider, provider_place_id)
);

CREATE TABLE route_snapshots (
  id                       UUID PRIMARY KEY,
  provider                 TEXT NOT NULL,
  mode                     TEXT NOT NULL,
  distance_meters          INT NOT NULL,
  duration_seconds         INT NOT NULL,
  estimated_cost_amount    NUMERIC(12,2),
  estimated_cost_currency  CHAR(3),
  raw_normalized_payload   JSONB,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```go
// Provider Adapter Interface（Go）
type MapProvider interface {
    SearchPlaces(ctx context.Context, req PlaceSearchRequest) ([]NormalizedPlace, error)
    GetPlaceDetail(ctx context.Context, providerPlaceID string) (*NormalizedPlace, error)
    EstimateRoute(ctx context.Context, req RouteEstimateRequest) (*NormalizedRoute, error)
    Name() string
}

type NormalizedPlace struct {
    ProviderPlaceID string
    Name            string
    Address         string
    Lat, Lng        float64
    Categories      []string
    OpeningHours    *OpeningHours
}

type NormalizedRoute struct {
    Mode                   string
    DistanceMeters         int
    DurationSeconds        int
    EstimatedCostAmount    *decimal.Decimal
    EstimatedCostCurrency  *string
}
```

---

## BE-09｜AI Planner

**模式**：Planning Job 管理、Prompt 組裝、結構化輸出解析、Draft 版本管理

### 細部功能
- Provider adapter（OpenAI / Anthropic / Google / custom）
- Planning request 建立 → 202 async job
- Prompt 三層組裝：System / Context / User
- Structured JSON output parsing
- Validation pipeline（詳見 BE-10）
- Draft versioning 與 explainability metadata
- Token / cost usage accounting
- Distributed lock 防止重複 job

### 邊界個案
- Provider 逾時 → job 狀態標為 failed，存 failure_code
- LLM 回傳非法 JSON → AI_PROVIDER_INVALID_OUTPUT
- Prompt injection 嘗試 → system prompt boundary 生效，validation engine 標記 invalid
- 採用 draft 但 trip 已有不相容變更 → 要求重新驗證

### 資料結構（DB Schema）

```sql
CREATE TABLE ai_plan_requests (
  id                    UUID PRIMARY KEY,
  trip_id               UUID NOT NULL REFERENCES trips(id),
  requested_by_user_id  UUID NOT NULL REFERENCES users(id),
  provider_config_id    UUID NOT NULL REFERENCES llm_provider_configs(id),
  status                TEXT NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued','running','succeeded','failed')),
  prompt_context        JSONB NOT NULL,       -- redacted/hashed for audit
  prompt_tokens         INT,
  completion_tokens     INT,
  estimated_cost        NUMERIC(10,6),
  queued_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at            TIMESTAMPTZ,
  finished_at           TIMESTAMPTZ,
  failure_code          TEXT,
  failure_message       TEXT
);

CREATE TABLE ai_plan_drafts (
  id              UUID PRIMARY KEY,
  trip_id         UUID NOT NULL REFERENCES trips(id),
  request_id      UUID NOT NULL REFERENCES ai_plan_requests(id),
  title           TEXT NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('valid','warning','invalid')),
  draft_payload   JSONB NOT NULL,             -- full structured itinerary
  summary_payload JSONB NOT NULL,             -- for list preview
  version         INT NOT NULL DEFAULT 1,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_plan_validation_results (
  id         UUID PRIMARY KEY,
  draft_id   UUID NOT NULL REFERENCES ai_plan_drafts(id),
  severity   TEXT NOT NULL CHECK (severity IN ('error','warning','info')),
  rule_code  TEXT NOT NULL,
  message    TEXT NOT NULL,
  details    JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```go
// Draft JSON payload 核心結構（Go struct）
type AiDraftPayload struct {
    Title    string        `json:"title"`
    Summary  string        `json:"summary"`
    Days     []AiDraftDay  `json:"days"`
    BudgetSummary AiBudgetSummary `json:"budgetSummary"`
    GlobalWarnings []string `json:"globalWarnings"`
}

type AiDraftDay struct {
    Date      string        `json:"date"`
    Theme     string        `json:"theme"`
    Items     []AiDraftItem `json:"items"`
    DailyBudgetEstimate AmountWithCurrency `json:"dailyBudgetEstimate"`
}

type AiDraftItem struct {
    Title          string     `json:"title"`
    ItemType       string     `json:"itemType"`
    StartAt        *time.Time `json:"startAt"`
    EndAt          *time.Time `json:"endAt"`
    Place          *AiDraftPlace `json:"place"`
    EstimatedCost  AmountWithCurrency `json:"estimatedCost"`
    Confidence     string     `json:"confidence"` // high | medium | low
    Warnings       []string   `json:"warnings"`
}
```

---

## BE-10｜Validation Engine

**模式**：AI 草案驗證管線、五層驗證、結果分級

### 細部功能
1. **Schema Validation**：JSON parse、必填欄位、enum 合法、日期格式
2. **Business Validation**：days 落於 trip 範圍、每日總時長、預算超出比例、重複 POI
3. **Geographic Validation**：座標合理性、路線時間矛盾、跨城市瞬移偵測
4. **Trust Validation**：place id 不在候選集 → unverified_place、opening hours 缺失警告
5. **Safety Validation**：違規內容、secret-like pattern、HTML/JS script 片段

### 結果分級

| 狀態 | 說明 |
|------|------|
| `valid` | 可採用 |
| `warning` | 可採用但需顯示醒目提示 |
| `invalid` | 不可採用，只能查看失敗原因 |

### 邊界個案
- LLM 自述驗證通過 → validation engine 不信任 LLM 自述，仍執行全部規則
- 所有 days 有效但 budget 超出 25% → 整體標為 invalid
- 地點座標精度不足 → 降為 warning，不直接 invalid

### 資料結構（Go）

```go
type ValidationResult struct {
    Status   string            // valid | warning | invalid
    Results  []ValidationIssue
}

type ValidationIssue struct {
    Severity string // error | warning | info
    RuleCode string
    Message  string
    Details  map[string]any
}

// 規則碼清單
const (
    RuleSchemaInvalid         = "SCHEMA_INVALID"
    RuleItemOutOfRange        = "ITEM_OUT_OF_DATE_RANGE"
    RuleTimeOverlap           = "TIME_OVERLAP"
    RuleBudgetWarning         = "BUDGET_OVER_10_PCT"
    RuleBudgetInvalid         = "BUDGET_OVER_20_PCT"
    RuleDuplicatePOI          = "DUPLICATE_POI"
    RuleGeoImpossibleTravel   = "GEO_IMPOSSIBLE_TRAVEL"
    RuleUnverifiedPlace       = "UNVERIFIED_PLACE"
    RuleMissingOpeningHours   = "MISSING_OPENING_HOURS"
    RuleSafetyPromptInjection = "SAFETY_PROMPT_INJECTION"
)
```

---

## BE-11｜Sync / Outbox

**模式**：Transactional Outbox、衍生系統同步、Retry / DLQ

### 細部功能
- 主交易 commit 後寫入 outbox_events（同 transaction）
- Worker 消費 outbox → Firebase shadow sync / analytics / notification
- Exactly-once illusion（idempotent consumer + dedupe key）
- DLQ 處理與告警
- Bootstrap API 支援 sinceVersion 差異同步

### 邊界個案
- Outbox 寫入失敗 → 與主交易同 rollback，不產生幽靈事件
- Worker 重試仍失敗 → 進 DLQ，告警 SRE
- Firebase shadow sync 延遲 → 正式 DB 正常，告知用戶可能有同步延遲
- Client 版本過舊 → server 要求 full re-sync

### 資料結構（DB Schema）

```sql
CREATE TABLE outbox_events (
  id             UUID PRIMARY KEY,
  trip_id        UUID REFERENCES trips(id),
  aggregate_type TEXT NOT NULL,    -- trips | itinerary_items | ai_plan_drafts
  aggregate_id   TEXT NOT NULL,
  event_type     TEXT NOT NULL,    -- e.g. "itinerary_item.updated"
  payload        JSONB NOT NULL,
  dedupe_key     TEXT UNIQUE,      -- for idempotent consumer
  status         TEXT NOT NULL DEFAULT 'pending',
  retry_count    INT NOT NULL DEFAULT 0,
  available_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at   TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_outbox_events_status_available ON outbox_events(status, available_at)
  WHERE status = 'pending';
```

---

## BE-12｜Notification

**模式**：Event-driven 通知、多管道派送、Dedupe、Retry

### 細部功能
- Event-driven 通知觸發（由 outbox 消費後產生）
- Per-user delivery preference（in-app / push / email 各自開關）
- Dedupe rule（同事件短時間不重複推送）
- Push retry 與失敗記錄
- FCM token 管理與刷新

### 邊界個案
- FCM 送達失敗 → 記錄失敗，不影響正式資料流
- 使用者關閉推播 → 降為 in-app only
- 通知爆量（多人同時編輯）→ dedupe rule 合併通知
- 通知發送時 trip 已封存 → 仍可發送

### 資料結構（DB Schema）

```sql
CREATE TABLE notifications (
  id         UUID PRIMARY KEY,
  user_id    UUID NOT NULL REFERENCES users(id),
  trip_id    UUID REFERENCES trips(id),
  type       TEXT NOT NULL,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL,
  payload    JSONB DEFAULT '{}',
  link       TEXT,
  read_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id_read ON notifications(user_id, read_at)
  WHERE read_at IS NULL;

CREATE TABLE fcm_tokens (
  id          UUID PRIMARY KEY,
  user_id     UUID NOT NULL REFERENCES users(id),
  token       TEXT NOT NULL UNIQUE,
  platform    TEXT NOT NULL,  -- web | android | ios
  is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## BE-13｜Search / Recommendation（Beta+）

**模式**：景點搜尋、偏好排序、推薦候選產生

### 細部功能
- Full-text + tag based lookup
- Recent / favorite / similar place suggestions
- 可與 AI Planner 共用 candidate generation pipeline

### 邊界個案
- 搜尋結果為空 → 回傳空陣列 + suggestion hint，不 404
- 偏好資料不足 → fallback 為熱門景點

---

## BE-14｜Admin / Ops

**模式**：內部管理介面、Job 控制、Provider 健康、Feature Flag

### 細部功能
- Job retry / cancel
- 可疑用量審查
- Provider health dashboard
- Feature flag toggle
- Share link revoke
- Emergency provider disable

### 邊界個案
- Admin endpoint 被一般使用者存取 → 分離 route + 雙因子保護，強制 403
- Provider 緊急下線 → circuit breaker 觸發，新 job 停止接單
- Feature flag 誤關閉主功能 → 快速 toggle 回復，記錄操作 audit log

### 資料結構（DB Schema）

```sql
CREATE TABLE audit_logs (
  id            UUID PRIMARY KEY,
  actor_user_id UUID REFERENCES users(id),
  action        TEXT NOT NULL,          -- e.g. "adopt_ai_draft"
  resource_type TEXT NOT NULL,
  resource_id   TEXT NOT NULL,
  before_state  JSONB,
  after_state   JSONB,
  request_id    TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_user_id);
```

---
}
---

# 🎨 UI/UX 規格

---
{
## UX-01｜設計系統原則

| 原則 | 實作要求 |
|------|---------|
| Mobile-first | 主要流程在 375px 寬完整可用 |
| Offline feedback | 任何離線狀態需有視覺標示，不讓使用者陷入無回應 |
| Optimistic UI | 本地先行顯示變更，失敗時明確 rollback 提示 |
| Accessibility | 所有互動元素可 Tab 導航，aria 屬性正確，焦點管理完整 |
| Error clarity | 錯誤訊息區分「用戶操作問題」vs「系統問題」，給出明確 CTA |

---

## UX-02｜互動模式規格

### Drag & Drop（行程排序）
- 視覺拖曳手把（Desktop: hover 顯示 / Mobile: 長按觸發）
- 拖曳時顯示放置 placeholder
- 跨日移動需有明確的 day 分隔線視覺引導
- 邊界個案：拖曳中離線 → 動畫完成但標示待同步

### 時間衝突顯示
- 重疊 items 以橙色邊框 + 警告 icon 標示
- Tooltip 說明衝突的 item 名稱與時間
- 不強制阻擋儲存，但在 save 前需確認 warning

### AI Draft 比較
- 側滑面板顯示多套 draft
- 差異 highlight（新增 / 刪除 / 修改 以顏色區分）
- 每套 draft 顯示：預算估算、警告數量、景點數量
- 採用按鈕只在 `valid` / `warning` 狀態顯示

### 預算進度
- 各分類使用環形或條狀圖
- 超預算分類以紅色標示
- 金額顯示統一幣別（含換算來源與日期）

---

## UX-03｜主要頁面流程

### 登入流程
```
首頁 → 輸入 email → 送出 Magic Link → 等待頁（含重發按鈕）
→ 點擊 mail 連結 → 驗證 token → 登入成功 → 回到原頁或首頁
```

### 建立旅程（Wizard）
```
Step 1: 目的地 + 日期
Step 2: 人數 + 幣別 + 時區
Step 3: 旅行風格偏好（可跳過）
→ 建立成功 → 進入 Trip Overview
```

### AI 規劃流程
```
進入 AI Planner → 填寫條件 → 提交 → 等待 Job（可離開）
→ 收到推播通知 → 查看 Draft 清單 → 比較差異 → 手動採用
→ 採用確認 dialog → 成功寫入正式 itinerary
```

### 離線同步流程
```
離線操作 → pending badge 顯示 → 恢復上線 → 自動重送
→ 成功：badge 消失 → 衝突：顯示衝突解決 UI
```

---

## UX-04｜通知 & 回饋元件

| 元件 | 觸發時機 | 顯示位置 |
|------|---------|---------|
| Toast | 操作成功 / 輕量警告 | 右上角，3s 自動消失 |
| Modal | 需確認的危險操作（刪除、採用 draft）| 全螢幕遮罩 |
| Bottom Sheet | 行動裝置次要操作選單 | 底部滑出 |
| Inline Warning | 時間衝突、超預算 | 卡片內嵌 |
| Offline Banner | 偵測到離線 | 頂部固定條 |
| Sync Badge | Mutation queue 有待送出項目 | 導覽列圖示 |

---

## UX-05｜邊界個案 UX 處理總表

| 情境 | UX 處理方式 |
|------|------------|
| 首次開啟無旅程 | 空狀態插圖 + 明顯 CTA「建立第一個旅程」 |
| 請求 loading 超過 2s | 顯示 skeleton screen，不顯示空白 |
| API 500 錯誤 | 顯示「系統暫時無法使用」+ request_id 供回報 |
| 長時間 AI Job | 進度動畫 + 「完成後會通知您」 |
| Trip 被踢出 | 全頁覆蓋提示「您已不在此旅程中」，3s 後回首頁 |
| Token 過期 | 靜默 refresh 失敗後 → 顯示「登入已過期」+ 一鍵重新登入 |
| 地圖 SDK 失敗 | 降級顯示地址清單，標示「地圖暫時無法載入」 |
| PWA 更新可用 | 底部 snackbar「有新版本」+ 「立即更新」按鈕 |
| Drag & drop 不支援 | 提供排序按鈕（↑↓）作為 fallback |

---

# Phase 2：持久化 + 真實整合

## BE-P2-01｜PostgreSQL 持久化遷移

**模式**：將所有 `sync.RWMutex` in-memory store 替換為 PostgreSQL repository

### 細部功能
- 建立 pgx connection pool（max_conns, idle_timeout, health check）
- 每個模組提供 `RepositoryPostgres` 實作替換 `RepositoryMemory`
- 啟動時自動執行 `golang-migrate` 到最新版本
- 所有 CRUD 走 `BEGIN/COMMIT` transaction
- 樂觀鎖用 `version` column + `WHERE version = $1`

### 邊界個案
- 連線池耗盡 → 503 + Retry-After
- Migration 失敗 → 啟動中止，rollback 到上一版
- Deadlock → 自動 retry（最多 3 次）

---

## BE-P2-04｜真實 LLM Provider 整合

**模式**：替換 mock adapter 為真實 HTTP client

### 細部功能
- 從 `llm_provider_configs.encrypted_key` 解密取得真實 API key
- 組裝 system/context/user 三層 prompt 發送到真實 API
- 解析 structured JSON output
- Token 用量與 cost 寫入 `ai_plan_requests`
- 30s timeout + context cancellation

---

## BE-P2-05｜真實 Map Provider 整合

**模式**：替換 mock map adapter 為 Google Maps / Mapbox

### 細部功能
- Place Search → 真實 API 呼叫，response normalize 為 `PlaceSnapshot`
- Route Estimation → 真實 API 呼叫，response normalize 為 `RouteSnapshot`
- Geocode / Reverse Geocode → 真實 API
- Quota 監控 + 每日用量上限

---

## BE-P2-06｜真實 FCM / Email

**模式**：替換 mock notification delivery 為 Firebase + Email provider

### 細部功能
- Firebase Admin SDK 初始化
- `messaging.Send()` 真實推播
- SendGrid / Resend API 真實發信
- Email template rendering（Go `html/template`）

