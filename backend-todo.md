# 🖥️ 後端開發進度 Todo List

> 依據 `backend-contruction.md` 規格文件，追蹤各模組開發進度。
> 更新日期：2026-03-27

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
- [x] Itinerary days/items：in-memory → PostgreSQL `itinerary_days` + `itinerary_items`
- [x] Budget profiles/expenses：in-memory → PostgreSQL `budget_profiles` + `expenses`
- [x] Notifications：in-memory → PostgreSQL `notifications` table
- [x] AI plan requests/drafts/validations：in-memory → PostgreSQL 三表
- [x] Users/preferences：in-memory → PostgreSQL `users` + `user_preferences`
- [x] LLM provider configs：in-memory → PostgreSQL `llm_provider_configs`
- [x] Sessions：in-memory → PostgreSQL `sessions` table
- [x] Outbox events：in-memory → PostgreSQL `outbox_events`
- [x] Audit logs：in-memory → PostgreSQL `audit_logs`
- [x] FCM tokens：in-memory → PostgreSQL `fcm_tokens`
- [x] Idempotency keys：in-memory → PostgreSQL `trip_idempotency_keys`

### 邊界個案
- [x] 連線池耗盡 → graceful error + 503
- [x] Migration 版本不符 → 啟動時自動跑 migrate
- [x] Transaction deadlock → retry 機制

---

## BE-P2-02｜Supabase Row-Level Security (RLS)

- [x] 為所有 table 啟用 RLS
- [x] 定義 policy：trips 只能 owner/member 存取
- [x] 定義 policy：expenses 只能 trip member 存取
- [x] 定義 policy：notifications 只能本人存取
- [x] Service role key 用於 backend API → bypass RLS
- [ ] 前端 Supabase client 使用 anon key → 受 RLS 限制

---

## BE-P2-03｜Redis 快取層

> 註：此區分佈式能力僅在 `RUNTIME_MODE=distributed` 啟用；預設 `single` 走單主機行為。

- [x] 建立 Redis client + connection pool
- [x] Session 快取（避免每次查 DB）
- [x] Rate limit 從 in-memory → Redis（分布式）
- [x] AI planning distributed lock → Redis
- [x] Budget 匯率快取 → Redis（TTL 1h）
- [x] Idempotency key 快取 → Redis（TTL 24h）

---

## BE-P2-04｜真實 LLM Provider 整合

- [x] OpenAI API 真實呼叫（gpt-4.1-mini / gpt-4.1）
- [x] Anthropic API 真實呼叫（claude-sonnet-4-20250514）
- [x] Google Gemini API 真實呼叫
- [x] API key 從 `llm_provider_configs` 解密後使用
- [x] Token 用量計算寫入 `ai_plan_requests.token_usage`
- [x] Cost 估算寫入 `ai_plan_requests.estimated_cost`
- [x] Provider error mapping → 統一 error code
- [x] 超時保護（30s context deadline）

---

## BE-P2-05｜真實 Map Provider 整合

- [x] Google Maps / Mapbox API 真實呼叫
- [x] Geocode API 真實呼叫
- [x] Place search 真實呼叫
- [x] Route estimation 真實呼叫
- [x] API key 管理（環境變數注入）
- [x] Quota 保護（每日用量限制）
- [x] Provider fallback（主 provider 失敗 → 備援）

---

## BE-P2-06｜真實 FCM Push 推播

- [x] Firebase Admin SDK 初始化
- [x] FCM token 儲存真實寫入 PostgreSQL
- [x] Push notification 真實透過 FCM 發送
- [x] Token refresh 流程（client 上傳新 token）
- [x] Push 失敗的 retry + DLQ 處理

---

## BE-P2-07｜真實 Email 發送

- [x] Email provider 整合（SendGrid / SES / Resend）
- [x] Magic link email 真實發送
- [x] Invite email 真實發送
- [x] Trip update digest email
- [x] Email template 管理

---

## BE-P2-08｜Outbox Worker 真實消費

- [x] Worker 從 PostgreSQL outbox_events 輪詢
- [x] 消費後寫入 notification
- [x] 消費後同步 Firebase shadow
- [x] 消費後發送 analytics event
- [x] DLQ → 手動重試 API
- [x] Worker graceful shutdown

---

## BE-P2-09｜Docker Compose 本地開發

- [x] docker-compose.yml（Go API + PostgreSQL + Redis）
- [ ] 配合 Supabase local dev（supabase start）
- [x] Hot reload（air 或 CompileDaemon）
- [x] .env.local 範例檔
- [x] Seed data 腳本

---

## BE-P2-10｜CI/CD Production Pipeline

- [x] GitHub Actions：lint + test + build
- [x] Docker image build + push to registry
- [x] Database migration 自動化
- [ ] Staging 環境部署
- [ ] Production 環境部署（Blue-green / Canary）
- [ ] Rollback playbook

# 🖥️ 後端開發進度 Todo List — Phase 2（持久化 + 真實整合）

> Phase 1 所有 14 個模組均已完成（in-memory store）。
> Phase 2 目標：將 in-memory 切換至 PostgreSQL 持久化，整合真實外部服務。
> 更新日期：2026-03-27

---

## BE-P2-01｜PostgreSQL 持久化遷移

### 已完成 ✅
- [x] 建立 `database/postgres.go` 連線池（pgx）
- [x] Trips：`repository_memory.go` → `repository_postgres.go`
- [x] Trip Membership：PostgreSQL `trip_memberships`
- [x] Invitations：PostgreSQL `trip_invitations`
- [x] Share Links：PostgreSQL `share_links`
- [x] Budget Profiles / Expenses：PostgreSQL
- [x] Notifications：PostgreSQL
- [x] LLM Provider Configs：PostgreSQL
- [x] Outbox Events：PostgreSQL
- [x] Idempotency Keys：PostgreSQL `trip_idempotency_keys`

### 未完成 ❌
- [x] Itinerary Days / Items → PostgreSQL `itinerary_days` + `itinerary_items`
  - [x] repository_postgres.go 實作 CRUD
  - [x] Bulk reorder transaction 實作
  - [x] 跨日移動 transaction 實作
  - [x] version bump + optimistic lock 實作
  - [x] place_snapshot / route_snapshot 綁定
- [x] AI Plan Requests / Drafts / Validations → PostgreSQL 三表
  - [x] ai_plan_requests repository_postgres.go
  - [x] ai_plan_drafts repository_postgres.go
  - [x] ai_plan_validation_results repository_postgres.go
  - [x] Planning job status 持久化
  - [x] Draft adopt transaction（寫入 itinerary_items）
- [x] Users / Preferences → PostgreSQL `users` + `user_preferences`
  - [x] users repository_postgres.go
  - [x] user_preferences repository_postgres.go（含 version + optimistic lock）
  - [x] 帳號刪除清理（cascade + audit log）
- [x] Sessions → PostgreSQL `sessions`
  - [x] sessions repository_postgres.go
  - [x] Refresh token rotation 持久化
  - [x] Family tracking + revocation 持久化
  - [x] Session 過期清理 cron
- [x] Audit Logs → PostgreSQL `audit_logs`
  - [x] audit_logs repository_postgres.go
  - [x] 關鍵操作自動寫入 audit log（trip 修改、成員變更、AI draft adopt）
- [x] FCM Tokens → PostgreSQL `fcm_tokens`
  - [x] fcm_tokens repository_postgres.go
  - [x] Token 過期清理

### 邊界個案
- [x] 連線池耗盡 → graceful error + 503
- [x] Migration 版本不符 → 啟動時自動跑 migrate
- [x] Transaction deadlock → retry 機制

---

## BE-P2-02｜Supabase Row-Level Security (RLS)

- [x] 為所有 table 啟用 RLS
- [x] 定義 policy：trips 只能 owner / member 存取
- [x] 定義 policy：expenses 只能 trip member 存取
- [x] 定義 policy：notifications 只能本人存取
- [x] 定義 policy：user_preferences 只能本人存取
- [x] Service role key 用於 backend API → bypass RLS
- [ ] 前端 Supabase client 使用 anon key → 受 RLS 限制

---

## BE-P2-03｜Redis 快取層

> 註：此區分佈式能力僅在 `RUNTIME_MODE=distributed` 啟用；預設 `single` 走單主機行為。

- [x] 建立 Redis client + connection pool
- [x] Session 快取（避免每次查 DB）
  - [x] Login → 寫入 Redis
  - [x] Refresh → 更新 Redis
  - [x] Logout → 刪除 Redis
- [x] Rate limit 從 in-memory → Redis（分布式）
  - [x] Token bucket 算法 Redis 實作
  - [x] 按 user_id + endpoint 限速
- [x] AI planning distributed lock → Redis
  - [x] SETNX + TTL 實作
  - [x] Lock 超時自動釋放
- [x] Budget 匯率快取 → Redis（TTL 1h）
  - [x] GET 時先查 Redis
  - [x] Cache miss → 查外部 API → 寫入 Redis
- [x] Idempotency key 快取 → Redis（TTL 24h）
  - [x] 快速查重（不走 DB）
  - [x] Fallback 至 PostgreSQL

---

## BE-P2-04｜真實 LLM Provider 整合

- [x] OpenAI API
  - [x] HTTP client 實作
  - [x] Chat Completion 呼叫（gpt-4.1-mini / gpt-4.1）
  - [x] JSON mode 結構化輸出
  - [x] Token 用量解析
- [x] Anthropic API
  - [x] HTTP client 實作
  - [x] Messages API 呼叫（claude-sonnet-4-20250514）
  - [x] JSON mode 結構化輸出
  - [x] Token 用量解析
- [x] Google Gemini API
  - [x] HTTP client 實作
  - [x] Generate Content 呼叫
  - [x] JSON mode 結構化輸出
- [x] 共用邏輯
  - [x] API key 從 `llm_provider_configs` AES-256-GCM 解密
  - [x] Token / Cost 計算 → 寫入 `ai_plan_requests`
  - [x] Provider error mapping → 統一 error code
  - [x] 超時保護（30s context deadline）
  - [x] Circuit breaker（連續失敗 → 暫停接單）

---

## BE-P2-05｜真實 Map Provider 整合

- [x] Google Maps API
  - [x] Places API（搜尋 + 詳情）
  - [x] Geocoding API
  - [x] Directions API（route estimation）
  - [x] API key 環境變數管理
- [x] Mapbox API（備援 provider）
  - [x] Geocoding API
  - [x] Directions API
- [x] 共用邏輯
  - [x] Quota 保護（每日用量上限）
  - [x] Provider fallback（主 provider 失敗 → 備援）
  - [x] Rate limit（requests per second）
  - [x] Place / Route 結果正規化（NormalizedPlace / NormalizedRoute）

---

## BE-P2-06｜真實 FCM Push 推播

- [x] Firebase Admin SDK 初始化
  - [x] Service account key 管理
  - [x] Go SDK (`firebase.google.com/go/v4`)
- [x] FCM token 管理
  - [x] `POST /fcm-tokens` 真實寫入 PostgreSQL
  - [x] Token refresh 更新
  - [x] 無效 token 清理（FCM 回傳 invalid token error）
- [x] Push notification 真實發送
  - [x] `messaging.SendEachForMulticast()` 實作
  - [x] 依 user notification preferences 過濾
  - [x] Payload 組裝（title / body / data / click_action）
- [x] 錯誤處理
  - [x] Push 失敗 → retry（指數退避）
  - [x] 重試超限 → DLQ + 告警
  - [x] Invalid token → 標記 is_active = false

---

## BE-P2-07｜真實 Email 發送

- [x] Email provider 整合
  - [x] Provider adapter interface 定義
  - [x] SendGrid adapter 實作
  - [x] AWS SES adapter 實作（或 Resend）
  - [x] Provider fallback 機制
- [x] Magic Link email
  - [x] HTML template（含 magic link URL）
  - [x] 純文字 fallback
  - [x] Rate limit（同一 email 60s 內不重發）
- [x] Invite email
  - [x] HTML template（含邀請者名稱、trip 名稱、接受連結）
  - [x] 邀請過期前到期提醒
- [x] Trip update digest email
  - [x] 每日 / 每週摘要 cron
  - [x] HTML template（變更清單）
- [x] Email template 管理
  - [x] Go template 或 MJML
  - [x] i18n 支援（依 user locale）

---

## BE-P2-08｜Outbox Worker 真實消費

- [x] Worker 從 PostgreSQL outbox_events 輪詢
  - [x] Polling interval 配置（預設 1s）
  - [x] Batch size 配置（預設 50）
  - [x] Status = 'pending' AND available_at <= now()
- [x] 消費處理
  - [x] 寫入 notification（in-app 通知）
  - [x] 觸發 FCM push（呼叫 BE-P2-06）
  - [x] 觸發 email 發送（呼叫 BE-P2-07）
  - [x] 同步 Firebase shadow（若需要）
  - [x] 發送 analytics event
- [x] 錯誤處理
  - [x] 消費失敗 → retry_count + 1 + 指數退避
  - [x] 超過 max retry → 進 DLQ（status = 'dead'）
  - [x] DLQ 告警 + 手動重試 API
- [x] Worker lifecycle
  - [x] Graceful shutdown（SIGTERM → 完成目前 batch → 退出）
  - [x] Health check endpoint
  - [x] Metrics（消費速率、queue 深度、DLQ 數量）

---

## BE-P2-09｜Docker Compose 本地開發

- [ ] docker-compose.yml
  - [x] Go API service（含 air hot reload）
  - [x] PostgreSQL 15+
  - [x] Redis 7+
  - [ ] Supabase local dev（`supabase start`）
- [x] 開發配置
  - [x] .env.local 範例檔
  - [x] Volume mount for hot reload
  - [x] Health check 配置
- [x] Seed data 腳本
  - [x] 測試用 user + trip + itinerary
  - [x] 測試用 budget + expenses
  - [x] 測試用 notifications

---

## BE-P2-10｜CI/CD Production Pipeline

- [x] GitHub Actions workflow
  - [x] Go lint（golangci-lint）
  - [x] Go test（含 coverage report）
  - [x] Go build
  - [x] gofmt check
- [x] Docker 建構
  - [x] multi-stage Dockerfile
  - [x] Docker image build + push to registry
  - [x] Image tag 策略（commit SHA + semver）
- [x] Database migration
  - [x] 自動化 migration（golang-migrate）
  - [x] Migration dry-run 驗證
  - [x] Rollback 腳本
- [ ] 部署
  - [ ] Staging 環境部署
  - [ ] Production 環境部署（Blue-green / Canary）
  - [ ] Rollback playbook
  - [x] Health check + readiness probe
