# 🖥️ 後端開發進度 Todo List

> 依據 `backend-contruction.md` 規格文件，追蹤各模組開發進度。
> 更新日期：2026-03-23

---

## BE-01｜API Gateway / HTTP Layer

- [x] 統一 request_id / correlation_id 注入（`requestIDMiddleware`）
- [x] 統一錯誤 envelope（`ErrorEnvelope` / `APIError`）
- [x] CORS middleware（含 PUT、custom headers）
- [x] Recovery middleware（panic → 500 + log）
- [x] Access log middleware
- [x] Rate limit middleware（token bucket, 429 + Retry-After）
- [x] JWT / session 驗證 middleware
- [x] CSRF middleware
- [x] OpenTelemetry trace 注入
- [x] DTO binding 與驗證（通用 struct tag 驗證器）

### 邊界個案
- [x] JWT 簽名錯誤 → 401，不洩漏簽名細節
- [x] Rate limit 達上限 → 429 + Retry-After header
- [x] 請求 body 格式錯誤 → 400 with validation error
- [x] Middleware panic → Recovery middleware 捕捉，回 500

---

## BE-02｜Auth

- [x] Email Magic Link 登入流程
- [x] OAuth provider 擴充架構（Google, Apple, Facebook, X, GitHub, LINE, Kakao, WeChat, TripAdvisor, Booking.com）
- [x] OAuth redirect + callback 流程
- [x] Sessions table migration（`000003_create_sessions`）
- [x] JWT config（access TTL、refresh TTL）
- [x] LLM provider API key 加密存放（`llm_provider_configs` table）
- [x] Refresh token rotation 實作
- [x] Refresh token reuse 偵測（撤銷整個 session family）
- [x] Invite token 驗證（含 expiry / single-use）
- [x] RBAC 權限集中檢查 middleware

### 邊界個案
- [x] Refresh token 重放攻擊 → 偵測 reuse，撤銷 session family
- [x] Invite token 過期 → 410 Gone
- [x] LLM key 加密失敗 → 不存入，回 500 並告警
- [x] 同帳號並發登入 → session 互不干擾

---

## BE-03｜User / Preference

- [x] Profile CRUD（GET / PATCH）
- [x] Preference CRUD（GET / PATCH explicit / inferred）
- [x] Notification preference CRUD
- [x] LLM provider config CRUD
- [x] `users` table migration
- [x] `user_preferences` table migration
- [x] `llm_provider_configs` table migration
- [x] Preference version 衝突（optimistic lock 409）
- [x] 刪除帳號時清除偏好與 LLM key

### 邊界個案
- [x] locale / timezone / wakeTime / sleepTime 格式驗證
- [x] Preference 版本衝突 → optimistic lock 409
- [x] 刪除帳號 → 清除個人偏好與 LLM key，保留 audit log

---

## BE-04｜Workspace / Membership

- [x] 成員清單查詢（含 role filter）
- [x] 新增成員（含 idempotency）
- [x] 移除成員
- [x] 修改成員角色
- [x] 確保每個 trip 至少 1 位 owner（last-owner 保護）
- [x] Invitation CRUD（建立 / 列表 / 撤銷 / 接受）
- [x] 邀請同一 email 不重複建立
- [x] Invitation 過期處理（410 Gone）
- [x] Share link 建立 / 列表 / 撤銷
- [x] `trip_memberships` table migration
- [x] `trip_invitations` table migration
- [x] `share_links` table migration
- [x] 接受邀請後自動建立 membership（with auth context）
- [x] Share link 存取驗證（token hash 比對）

### 邊界個案
- [x] 嘗試降級最後一位 owner → 阻擋 400
- [x] Share link 已被 revoke → 403
- [x] 邀請同一 email 兩次 → 回傳現有邀請
- [x] 大量成員同時接受邀請 → idempotency 保護

---

## BE-05｜Trip

- [x] Trip CRUD（List / Create / Get / Patch）
- [x] `If-Match-Version` 樂觀鎖
- [x] Idempotency-Key 防重複建立
- [x] 建立 trip 時自動產生 day skeleton
- [x] Status 狀態機：draft → active → archived
- [x] Trip 封存後嘗試編輯 → 403 Forbidden
- [x] `name` maxLength: 200 驗證
- [x] `travelersCount` 上限 50 驗證
- [x] Timezone / date range 驗證
- [x] `trips` table migration
- [x] Date range 縮短導致 itinerary 越界 → 回傳衝突清單
- [x] Destination metadata 寫入 / 更新

### 邊界個案
- [x] 版本衝突 → 409 Conflict
- [x] 建立帶重複 Idempotency-Key → 回傳已存在的 trip
- [x] Trip 封存後編輯 → 403 Forbidden
- [x] Date range 縮短導致 item 越界 → 衝突清單

---

## BE-06｜Itinerary

- [x] Item CRUD（Create / Patch / Delete）
- [x] ListDays（含 seed data）
- [x] Bulk reorder（含 idempotency）
- [x] `If-Match-Version` 樂觀鎖
- [x] `itemType` enum 驗證
- [x] `title` maxLength: 200 / `note` maxLength: 5000
- [x] 時間重疊檢測（warning response）
- [x] `itinerary_days` table migration
- [x] `itinerary_items` table migration
- [x] `place_snapshots` table migration
- [x] `route_snapshots` table migration
- [x] 跨日移動（transaction 更新兩個 day）
- [x] item 時間設定在 trip 範圍外 → 400
- [x] Route snapshot 綁定
- [x] Place snapshot 綁定

### 邊界個案
- [x] 版本衝突 → 409
- [x] 時間重疊 → warning 回應
- [x] Bulk reorder 中途失敗 → transaction rollback
- [x] item 時間超出 trip 範圍 → 400

---

## BE-07｜Budget

- [x] Budget profile CRUD（GET / PUT upsert）
- [x] Expense CRUD（List / Create / Patch / Delete）
- [x] Over-budget rule evaluation（>10% warning）
- [x] `category` enum 驗證（lodging/transit/food/attraction/shopping/misc）
- [x] `linkedItemId` 支援
- [x] Currency ISO-4217 驗證
- [x] Idempotency-Key 防重複
- [x] `budget_profiles` table migration
- [x] `expenses` table migration
- [x] Currency conversion snapshot（匯率快照）
- [x] 匯率 API 整合（fallback 機制）

### 邊界個案
- [x] 幣別長度不為 3 → 400
- [x] amount 負數 → 400
- [x] 匯率 API 失敗 → 保留上次快照

---

## BE-08｜Place / Map

- [x] Place search API（mock 實作）
- [x] Route estimate API（mock 實作）
- [x] mock adapter 帶 provider timeout / error 模擬
- [x] Provider adapter interface 抽象化
- [x] Geocode / reverse geocode
- [x] Place detail normalization
- [x] Route estimate normalization
- [x] External API quota 保護（rate limit + circuit breaker）

### 邊界個案
- [x] Provider timeout → MAP_PROVIDER_TIMEOUT
- [x] Quota 耗盡 → MAP_PROVIDER_QUOTA_EXCEEDED
- [x] Provider 回傳缺少必填欄位 → partial warning

---

## BE-09｜AI Planner

- [x] Planning request 建立（202 async job）
- [x] Draft 列表 / 取得
- [x] Draft adopt（含 warning 確認機制 `X-Confirm-Warnings`）
- [x] Draft status 驗證（invalid 不可 adopt）
- [x] Cost/token 估算基本邏輯
- [x] Idempotency-Key 防重複
- [x] `wakePattern` / `poiDensity` constraints 欄位
- [x] `ai_plan_requests` table migration
- [x] `ai_plan_drafts` table migration
- [x] `ai_plan_validation_results` table migration
- [x] Provider adapter（OpenAI / Anthropic / Google / custom）
- [x] Prompt 三層組裝（System / Context / User）
- [x] Structured JSON output parsing
- [x] Token / cost usage accounting（DB 寫入）
- [x] Distributed lock 防止重複 job
- [x] Validation pipeline 串接（BE-10）

### 邊界個案
- [x] Provider 逾時 → job failed + failure_code
- [x] LLM 回傳非法 JSON → AI_PROVIDER_INVALID_OUTPUT
- [x] Prompt injection → validation engine 標記 invalid

---

## BE-10｜Validation Engine

- [x] Schema Validation（JSON parse、必填、enum、日期）
- [x] Business Validation（days 範圍、每日總時長、預算超出、重複 POI）
- [x] Geographic Validation（座標合理、路線矛盾、跨城市瞬移）
- [x] Trust Validation（place id 不在候選集、opening hours 缺失）
- [x] Safety Validation（違規內容、secret-like pattern、HTML/JS script）
- [x] 結果分級：valid / warning / invalid

---

## BE-11｜Sync / Outbox

- [x] Bootstrap API（sinceVersion 差異同步）
- [x] Flush mutations API（idempotency）
- [x] `outbox_events` table migration（含 `dedupe_key`）
- [x] 主交易 commit 後寫入 outbox_events（同 transaction）
- [x] Worker 消費 outbox → Firebase / analytics / notification
- [x] Idempotent consumer + dedupe key 處理
- [x] DLQ 處理與告警
- [x] Client 版本過舊 → full re-sync

---

## BE-12｜Notification

- [x] Notification CRUD（List / Mark Read / Mark Unread / Mark All Read / Delete）
- [x] Pagination（cursor-based）
- [x] `notifications` table migration（含 `link` 欄位）
- [x] Event-driven 通知觸發（由 outbox 消費產生）
- [x] Per-user delivery preference（in-app / push / email）
- [x] Dedupe rule（同事件短時間不重複推送）
- [x] FCM token 管理（`fcm_tokens` table migration）
- [x] Push retry 與失敗記錄

---

## BE-13｜Search / Recommendation（Beta+）

- [x] Full-text + tag based lookup
- [x] Recent / favorite / similar place suggestions
- [x] Candidate generation pipeline

---

## BE-14｜Admin / Ops

- [x] `audit_logs` table migration
- [x] Job retry / cancel API
- [x] 可疑用量審查
- [x] Provider health dashboard
- [x] Feature flag toggle
- [x] Emergency provider disable
- [x] Admin endpoint 權限保護（分離 route + 雙因子）

---

# Phase 2：In-memory → 持久化 + 真實整合

> Phase 1 所有模組都以 `sync.RWMutex` in-memory store 實作，Supabase 表已建立但為空。
> Phase 2 目標：將所有模組切換至 PostgreSQL 持久化，並整合真實外部服務。

---

## BE-P2-01｜PostgreSQL 持久化遷移

- [x] 建立 `database/postgres.go` 連線池（pgx + connection pool config）
- [x] Trips 模組：`repository_memory.go` → `repository_postgres.go`（CRUD + version bump）
- [x] Trip membership：in-memory map → PostgreSQL `trip_memberships` table
- [x] Invitations：in-memory map → PostgreSQL `trip_invitations` table
- [x] Share links：in-memory map → PostgreSQL `share_links` table
- [ ] Itinerary days/items：in-memory → PostgreSQL `itinerary_days` + `itinerary_items`
- [x] Budget profiles/expenses：in-memory → PostgreSQL `budget_profiles` + `expenses`
- [x] Notifications：in-memory → PostgreSQL `notifications` table
- [ ] AI plan requests/drafts/validations：in-memory → PostgreSQL 三表
- [ ] Users/preferences：in-memory → PostgreSQL `users` + `user_preferences`
- [x] LLM provider configs：in-memory → PostgreSQL `llm_provider_configs`
- [ ] Sessions：in-memory → PostgreSQL `sessions` table
- [x] Outbox events：in-memory → PostgreSQL `outbox_events`
- [ ] Audit logs：in-memory → PostgreSQL `audit_logs`
- [ ] FCM tokens：in-memory → PostgreSQL `fcm_tokens`
- [x] Idempotency keys：in-memory → PostgreSQL `trip_idempotency_keys`

### 邊界個案
- [ ] 連線池耗盡 → graceful error + 503
- [ ] Migration 版本不符 → 啟動時自動跑 migrate
- [ ] Transaction deadlock → retry 機制

---

## BE-P2-02｜Supabase Row-Level Security (RLS)

- [ ] 為所有 table 啟用 RLS
- [ ] 定義 policy：trips 只能 owner/member 存取
- [ ] 定義 policy：expenses 只能 trip member 存取
- [ ] 定義 policy：notifications 只能本人存取
- [ ] Service role key 用於 backend API → bypass RLS
- [ ] 前端 Supabase client 使用 anon key → 受 RLS 限制

---

## BE-P2-03｜Redis 快取層

- [ ] 建立 Redis client + connection pool
- [ ] Session 快取（避免每次查 DB）
- [ ] Rate limit 從 in-memory → Redis（分布式）
- [ ] AI planning distributed lock → Redis
- [ ] Budget 匯率快取 → Redis（TTL 1h）
- [ ] Idempotency key 快取 → Redis（TTL 24h）

---

## BE-P2-04｜真實 LLM Provider 整合

- [ ] OpenAI API 真實呼叫（gpt-4.1-mini / gpt-4.1）
- [ ] Anthropic API 真實呼叫（claude-sonnet-4-20250514）
- [ ] Google Gemini API 真實呼叫
- [ ] API key 從 `llm_provider_configs` 解密後使用
- [ ] Token 用量計算寫入 `ai_plan_requests.token_usage`
- [ ] Cost 估算寫入 `ai_plan_requests.estimated_cost`
- [ ] Provider error mapping → 統一 error code
- [ ] 超時保護（30s context deadline）

---

## BE-P2-05｜真實 Map Provider 整合

- [ ] Google Maps / Mapbox API 真實呼叫
- [ ] Geocode API 真實呼叫
- [ ] Place search 真實呼叫
- [ ] Route estimation 真實呼叫
- [ ] API key 管理（環境變數注入）
- [ ] Quota 保護（每日用量限制）
- [ ] Provider fallback（主 provider 失敗 → 備援）

---

## BE-P2-06｜真實 FCM Push 推播

- [ ] Firebase Admin SDK 初始化
- [ ] FCM token 儲存真實寫入 PostgreSQL
- [ ] Push notification 真實透過 FCM 發送
- [ ] Token refresh 流程（client 上傳新 token）
- [ ] Push 失敗的 retry + DLQ 處理

---

## BE-P2-07｜真實 Email 發送

- [ ] Email provider 整合（SendGrid / SES / Resend）
- [ ] Magic link email 真實發送
- [ ] Invite email 真實發送
- [ ] Trip update digest email
- [ ] Email template 管理

---

## BE-P2-08｜Outbox Worker 真實消費

- [ ] Worker 從 PostgreSQL outbox_events 輪詢
- [ ] 消費後寫入 notification
- [ ] 消費後同步 Firebase shadow
- [ ] 消費後發送 analytics event
- [ ] DLQ → 手動重試 API
- [ ] Worker graceful shutdown

---

## BE-P2-09｜Docker Compose 本地開發

- [ ] docker-compose.yml（Go API + PostgreSQL + Redis）
- [ ] 配合 Supabase local dev（supabase start）
- [ ] Hot reload（air 或 CompileDaemon）
- [ ] .env.local 範例檔
- [ ] Seed data 腳本

---

## BE-P2-10｜CI/CD Production Pipeline

- [ ] GitHub Actions：lint + test + build
- [ ] Docker image build + push to registry
- [ ] Database migration 自動化
- [ ] Staging 環境部署
- [ ] Production 環境部署（Blue-green / Canary）
- [ ] Rollback playbook

---

# Quest 衍生任務

> 來源：`quest.md` — 用戶反饋產生的後端支援需求

---

## BE-Q1｜目的地 / 出發地搜尋 API

- [ ] `GET /api/v1/places/autocomplete?q={query}&lang={locale}` 代理 API
- [ ] Response 標準化為 `[]PlaceAutocompleteResult`（placeId, label, lat, lng）
- [ ] `GET /api/v1/places/timezone?lat=&lng=` 座標 → timezone 推算 API
- [ ] 搜尋結果快取 24h（Redis / in-memory cache）
- [ ] Rate limit per user：100 req/min
- [ ] 外部 API 不可用 → 500 + fallback 訊息
- [ ] query 輸入清洗（XSS / injection protection）

---

## BE-Q2｜Trip Schema 擴展

- [ ] Migration：trips 表新增 `departure_text` 欄位
- [ ] Migration：trips 表新增 `departure_lat` / `departure_lng` 欄位
- [ ] Migration：trips 表新增 `destination_lat` / `destination_lng` 欄位
- [ ] Migration：trips 表新增 `initial_budget` 欄位
- [ ] Trip Create API 接受新欄位（departure, coordinates, budget）
- [ ] Trip Patch API 接受新欄位
- [ ] 預算金額寫入時同步更新 `budget_profiles.total_budget`
- [ ] 新欄位全部 nullable（backward compatible）
- [ ] 預算金額 ≤ 0 → 400 validation error

