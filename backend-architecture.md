# 🖥️ 後端架構文檔

> 依據 `backend-contruction.md` 與實際程式碼分析，記錄後端各模組的實作狀態與剩餘任務。
> 更新日期：2026-03-26

---

## 概覽

後端採用 **Go (net/http + chi router)** 建構，資料層規劃使用 **PostgreSQL (pgx) + Redis**。目前 Phase 1 所有模組均已完成（in-memory store），Phase 2 持久化與真實外部服務整合進行中。

目前執行模式採 **single-host 優先**：

- 預設 `RUNTIME_MODE=single`，分佈式 Redis 行為關閉（保留本地行為）。
- 只有在 `RUNTIME_MODE=distributed` 時才啟用分佈式能力（共享 rate limit / lock / cache）。

---

## Phase 1 模組實作狀態（全部完成 ✅）

| # | 模組 | 說明 |
|---|------|------|
| BE-01 | API Gateway / HTTP Layer | 統一錯誤 envelope、CORS、CSRF、Rate Limit、JWT、Recovery、OTel |
| BE-02 | Auth | Magic Link、OAuth（10 providers）、Refresh Token Rotation、RBAC |
| BE-03 | User / Preference | Profile CRUD、偏好 CRUD、LLM config CRUD |
| BE-04 | Workspace / Membership | 成員管理、邀請、Share Link、last-owner 保護 |
| BE-05 | Trip | CRUD、樂觀鎖、day skeleton、狀態機、idempotency |
| BE-06 | Itinerary | Item CRUD、Bulk Reorder、時間衝突、跨日移動 |
| BE-07 | Budget | Profile CRUD、Expense CRUD、over-budget rule、匯率快照 |
| BE-08 | Place / Map | Provider adapter、mock 實作、geocode、route estimate |
| BE-09 | AI Planner | Planning request、Draft 管理、Prompt 組裝、Token accounting |
| BE-10 | Validation Engine | 五層驗證（Schema / Business / Geo / Trust / Safety） |
| BE-11 | Sync / Outbox | Bootstrap API、Flush mutations、outbox worker |
| BE-12 | Notification | CRUD、FCM token 管理、event-driven 觸發 |
| BE-13 | Search / Recommendation | Full-text lookup、suggestions（Beta+） |
| BE-14 | Admin / Ops | Audit logs、Job control、Feature flags |

---

## Phase 2 模組實作狀態

### BE-P2-01｜PostgreSQL 持久化遷移

| 子模組 | 狀態 | 說明 |
|--------|------|------|
| 連線池（pgx） | ✅ | `database/postgres.go` |
| Trips | ✅ | `repository_postgres.go` |
| Trip Membership | ✅ | PostgreSQL `trip_memberships` |
| Invitations | ✅ | PostgreSQL `trip_invitations` |
| Share Links | ✅ | PostgreSQL `share_links` |
| Itinerary Days/Items | ❌ | 仍為 in-memory |
| Budget Profiles/Expenses | ✅ | PostgreSQL |
| Notifications | ✅ | PostgreSQL |
| AI Plan (requests/drafts/validations) | ❌ | 仍為 in-memory 三表 |
| Users / Preferences | ❌ | 仍為 in-memory |
| LLM Provider Configs | ✅ | PostgreSQL |
| Sessions | ❌ | 仍為 in-memory |
| Outbox Events | ✅ | PostgreSQL |
| Audit Logs | ❌ | 仍為 in-memory |
| FCM Tokens | ❌ | 仍為 in-memory |
| Idempotency Keys | ✅ | PostgreSQL `trip_idempotency_keys` |

**邊界個案待處理：**
- 連線池耗盡 → graceful error + 503
- Migration 版本不符 → 啟動時自動 migrate
- Transaction deadlock → retry 機制

### BE-P2-02｜Supabase RLS（全部未開始 ❌）

- 為所有 table 啟用 RLS
- 定義 trips / expenses / notifications policy
- Service role key bypass
- 前端 anon key 受限

### BE-P2-03｜Redis 快取層（全部未開始 ❌）

- Redis client + connection pool
- Session 快取
- Rate limit 分布式化
- AI planning distributed lock
- 匯率快取（TTL 1h）
- Idempotency key 快取（TTL 24h）

### BE-P2-04｜真實 LLM Provider 整合（已完成 ✅）

- OpenAI Chat Completions（`gpt-4.1-mini` / `gpt-4.1`）
- Anthropic Messages API（`claude-sonnet-4-20250514`）
- Google Gemini Generate Content
- API key 解密（AES-256-GCM envelope）
- Token / Cost 計算與寫入 `ai_plan_requests`
- Provider error mapping + circuit breaker
- 超時保護（30s deadline）

### BE-P2-05｜真實 Map Provider 整合（已完成 ✅）

- Google Maps Places / Geocoding / Directions
- Mapbox Geocoding / Directions（備援 provider）
- Geocode / Place search / Route estimation 真實呼叫
- API key 環境變數管理
- 每日 quota + per-second rate limit + provider fallback

### BE-P2-06｜真實 FCM Push 推播（部分完成 ⚠️）

- FCM token PostgreSQL 寫入（upsert/refresh）✅
- Trigger notification 時真實 FCM HTTP 發送（可選，需設定 `FCM_SERVER_KEY`）✅
- Push 失敗 retry + DLQ 標記 ✅
- Invalid token 自動失效化（`is_active=false`）✅
- Firebase Admin SDK 初始化（Service Account + Go SDK）❌

### BE-P2-07｜真實 Email 發送（全部未開始 ❌）

- Email provider 整合（SendGrid / SES / Resend）
- Magic link email
- Invite email
- Trip update digest
- Email template

### BE-P2-08｜Outbox Worker 真實消費（部分完成 ⚠️）

- PostgreSQL outbox_events 輪詢（interval / batch size 可配置）✅
- 寫入 notification + FCM push + analytics dispatch hook ✅
- retry/backoff + DLQ 狀態轉移 ✅
- Worker graceful shutdown ✅
- Firebase shadow、Email dispatch、health/metrics 仍待補齊 ❌

### BE-P2-09｜Docker Compose 本地開發（全部未開始 ❌）

- docker-compose.yml（Go API + PostgreSQL + Redis）
- Hot reload + Supabase local dev
- .env.local + Seed data

### BE-P2-10｜CI/CD Production Pipeline（全部未開始 ❌）

- GitHub Actions：lint + test + build
- Docker image build + push
- Migration 自動化
- Staging / Production 部署
- Rollback playbook
