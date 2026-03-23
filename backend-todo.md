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
- [ ] JWT / session 驗證 middleware
- [ ] CSRF middleware
- [ ] OpenTelemetry trace 注入
- [ ] DTO binding 與驗證（通用 struct tag 驗證器）

### 邊界個案
- [ ] JWT 簽名錯誤 → 401，不洩漏簽名細節
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
- [ ] Refresh token rotation 實作
- [ ] Refresh token reuse 偵測（撤銷整個 session family）
- [ ] Invite token 驗證（含 expiry / single-use）
- [ ] RBAC 權限集中檢查 middleware

### 邊界個案
- [ ] Refresh token 重放攻擊 → 偵測 reuse，撤銷 session family
- [ ] Invite token 過期 → 410 Gone
- [ ] LLM key 加密失敗 → 不存入，回 500 並告警
- [ ] 同帳號並發登入 → session 互不干擾

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
- [ ] 刪除帳號時清除偏好與 LLM key

### 邊界個案
- [x] locale / timezone / wakeTime / sleepTime 格式驗證
- [x] Preference 版本衝突 → optimistic lock 409
- [ ] 刪除帳號 → 清除個人偏好與 LLM key，保留 audit log

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
- [ ] 接受邀請後自動建立 membership（with auth context）
- [ ] Share link 存取驗證（token hash 比對）

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
- [ ] Date range 縮短導致 itinerary 越界 → 回傳衝突清單
- [ ] Destination metadata 寫入 / 更新

### 邊界個案
- [x] 版本衝突 → 409 Conflict
- [x] 建立帶重複 Idempotency-Key → 回傳已存在的 trip
- [x] Trip 封存後編輯 → 403 Forbidden
- [ ] Date range 縮短導致 item 越界 → 衝突清單

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
- [ ] item 時間設定在 trip 範圍外 → 400
- [ ] Route snapshot 綁定
- [ ] Place snapshot 綁定

### 邊界個案
- [x] 版本衝突 → 409
- [x] 時間重疊 → warning 回應
- [ ] Bulk reorder 中途失敗 → transaction rollback
- [ ] item 時間超出 trip 範圍 → 400

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
- [ ] Currency conversion snapshot（匯率快照）
- [ ] 匯率 API 整合（fallback 機制）

### 邊界個案
- [x] 幣別長度不為 3 → 400
- [x] amount 負數 → 400
- [ ] 匯率 API 失敗 → 保留上次快照

---

## BE-08｜Place / Map

- [x] Place search API（mock 實作）
- [x] Route estimate API（mock 實作）
- [x] mock adapter 帶 provider timeout / error 模擬
- [x] Provider adapter interface 抽象化
- [ ] Geocode / reverse geocode
- [ ] Place detail normalization
- [ ] Route estimate normalization
- [ ] External API quota 保護（rate limit + circuit breaker）

### 邊界個案
- [x] Provider timeout → MAP_PROVIDER_TIMEOUT
- [ ] Quota 耗盡 → MAP_PROVIDER_QUOTA_EXCEEDED
- [ ] Provider 回傳缺少必填欄位 → partial warning

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
- [ ] Provider adapter（OpenAI / Anthropic / Google / custom）
- [ ] Prompt 三層組裝（System / Context / User）
- [ ] Structured JSON output parsing
- [ ] Token / cost usage accounting（DB 寫入）
- [ ] Distributed lock 防止重複 job
- [ ] Validation pipeline 串接（BE-10）

### 邊界個案
- [ ] Provider 逾時 → job failed + failure_code
- [ ] LLM 回傳非法 JSON → AI_PROVIDER_INVALID_OUTPUT
- [ ] Prompt injection → validation engine 標記 invalid

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
- [ ] 主交易 commit 後寫入 outbox_events（同 transaction）
- [ ] Worker 消費 outbox → Firebase / analytics / notification
- [ ] Idempotent consumer + dedupe key 處理
- [ ] DLQ 處理與告警
- [ ] Client 版本過舊 → full re-sync

---

## BE-12｜Notification

- [x] Notification CRUD（List / Mark Read / Mark Unread / Mark All Read / Delete）
- [x] Pagination（cursor-based）
- [x] `notifications` table migration（含 `link` 欄位）
- [ ] Event-driven 通知觸發（由 outbox 消費產生）
- [ ] Per-user delivery preference（in-app / push / email）
- [ ] Dedupe rule（同事件短時間不重複推送）
- [x] FCM token 管理（`fcm_tokens` table migration）
- [ ] Push retry 與失敗記錄

---

## BE-13｜Search / Recommendation（Beta+）

- [ ] Full-text + tag based lookup
- [ ] Recent / favorite / similar place suggestions
- [ ] Candidate generation pipeline

---

## BE-14｜Admin / Ops

- [x] `audit_logs` table migration
- [x] Job retry / cancel API
- [ ] 可疑用量審查
- [x] Provider health dashboard
- [ ] Feature flag toggle
- [ ] Emergency provider disable
- [ ] Admin endpoint 權限保護（分離 route + 雙因子）
