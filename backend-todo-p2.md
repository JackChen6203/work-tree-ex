# 🖥️ 後端開發進度 Todo List — Phase 2（持久化 + 真實整合）

> Phase 1 所有 14 個模組均已完成（in-memory store）。
> Phase 2 目標：將 in-memory 切換至 PostgreSQL 持久化，整合真實外部服務。
> 更新日期：2026-03-26

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
- [ ] Itinerary Days / Items → PostgreSQL `itinerary_days` + `itinerary_items`
  - [ ] repository_postgres.go 實作 CRUD
  - [ ] Bulk reorder transaction 實作
  - [ ] 跨日移動 transaction 實作
  - [ ] version bump + optimistic lock 實作
  - [ ] place_snapshot / route_snapshot 綁定
- [ ] AI Plan Requests / Drafts / Validations → PostgreSQL 三表
  - [ ] ai_plan_requests repository_postgres.go
  - [ ] ai_plan_drafts repository_postgres.go
  - [ ] ai_plan_validation_results repository_postgres.go
  - [ ] Planning job status 持久化
  - [ ] Draft adopt transaction（寫入 itinerary_items）
- [ ] Users / Preferences → PostgreSQL `users` + `user_preferences`
  - [ ] users repository_postgres.go
  - [ ] user_preferences repository_postgres.go（含 version + optimistic lock）
  - [ ] 帳號刪除清理（cascade + audit log）
- [ ] Sessions → PostgreSQL `sessions`
  - [ ] sessions repository_postgres.go
  - [ ] Refresh token rotation 持久化
  - [ ] Family tracking + revocation 持久化
  - [ ] Session 過期清理 cron
- [ ] Audit Logs → PostgreSQL `audit_logs`
  - [ ] audit_logs repository_postgres.go
  - [ ] 關鍵操作自動寫入 audit log（trip 修改、成員變更、AI draft adopt）
- [ ] FCM Tokens → PostgreSQL `fcm_tokens`
  - [ ] fcm_tokens repository_postgres.go
  - [ ] Token 過期清理

### 邊界個案
- [ ] 連線池耗盡 → graceful error + 503
- [ ] Migration 版本不符 → 啟動時自動跑 migrate
- [ ] Transaction deadlock → retry 機制

---

## BE-P2-02｜Supabase Row-Level Security (RLS)

- [ ] 為所有 table 啟用 RLS
- [ ] 定義 policy：trips 只能 owner / member 存取
- [ ] 定義 policy：expenses 只能 trip member 存取
- [ ] 定義 policy：notifications 只能本人存取
- [ ] 定義 policy：user_preferences 只能本人存取
- [ ] Service role key 用於 backend API → bypass RLS
- [ ] 前端 Supabase client 使用 anon key → 受 RLS 限制

---

## BE-P2-03｜Redis 快取層

- [ ] 建立 Redis client + connection pool
- [ ] Session 快取（避免每次查 DB）
  - [ ] Login → 寫入 Redis
  - [ ] Refresh → 更新 Redis
  - [ ] Logout → 刪除 Redis
- [ ] Rate limit 從 in-memory → Redis（分布式）
  - [ ] Token bucket 算法 Redis 實作
  - [ ] 按 user_id + endpoint 限速
- [ ] AI planning distributed lock → Redis
  - [ ] SETNX + TTL 實作
  - [ ] Lock 超時自動釋放
- [ ] Budget 匯率快取 → Redis（TTL 1h）
  - [ ] GET 時先查 Redis
  - [ ] Cache miss → 查外部 API → 寫入 Redis
- [ ] Idempotency key 快取 → Redis（TTL 24h）
  - [ ] 快速查重（不走 DB）
  - [ ] Fallback 至 PostgreSQL

---

## BE-P2-04｜真實 LLM Provider 整合

- [ ] OpenAI API
  - [ ] HTTP client 實作
  - [ ] Chat Completion 呼叫（gpt-4.1-mini / gpt-4.1）
  - [ ] JSON mode 結構化輸出
  - [ ] Token 用量解析
- [ ] Anthropic API
  - [ ] HTTP client 實作
  - [ ] Messages API 呼叫（claude-sonnet-4-20250514）
  - [ ] JSON mode 結構化輸出
  - [ ] Token 用量解析
- [ ] Google Gemini API
  - [ ] HTTP client 實作
  - [ ] Generate Content 呼叫
  - [ ] JSON mode 結構化輸出
- [ ] 共用邏輯
  - [ ] API key 從 `llm_provider_configs` AES-256-GCM 解密
  - [ ] Token / Cost 計算 → 寫入 `ai_plan_requests`
  - [ ] Provider error mapping → 統一 error code
  - [ ] 超時保護（30s context deadline）
  - [ ] Circuit breaker（連續失敗 → 暫停接單）

---

## BE-P2-05｜真實 Map Provider 整合

- [ ] Google Maps API
  - [ ] Places API（搜尋 + 詳情）
  - [ ] Geocoding API
  - [ ] Directions API（route estimation）
  - [ ] API key 環境變數管理
- [ ] Mapbox API（備援 provider）
  - [ ] Geocoding API
  - [ ] Directions API
- [ ] 共用邏輯
  - [ ] Quota 保護（每日用量上限）
  - [ ] Provider fallback（主 provider 失敗 → 備援）
  - [ ] Rate limit（requests per second）
  - [ ] Place / Route 結果正規化（NormalizedPlace / NormalizedRoute）

---

## BE-P2-06｜真實 FCM Push 推播

- [ ] Firebase Admin SDK 初始化
  - [ ] Service account key 管理
  - [ ] Go SDK (`firebase.google.com/go/v4`)
- [ ] FCM token 管理
  - [ ] `POST /fcm-tokens` 真實寫入 PostgreSQL
  - [ ] Token refresh 更新
  - [ ] 無效 token 清理（FCM 回傳 invalid token error）
- [ ] Push notification 真實發送
  - [ ] `messaging.SendMulticast()` 實作
  - [ ] 依 user notification preferences 過濾
  - [ ] Payload 組裝（title / body / data / click_action）
- [ ] 錯誤處理
  - [ ] Push 失敗 → retry（指數退避）
  - [ ] 重試超限 → DLQ + 告警
  - [ ] Invalid token → 標記 is_active = false

---

## BE-P2-07｜真實 Email 發送

- [ ] Email provider 整合
  - [ ] Provider adapter interface 定義
  - [ ] SendGrid adapter 實作
  - [ ] AWS SES adapter 實作（或 Resend）
  - [ ] Provider fallback 機制
- [ ] Magic Link email
  - [ ] HTML template（含 magic link URL）
  - [ ] 純文字 fallback
  - [ ] Rate limit（同一 email 60s 內不重發）
- [ ] Invite email
  - [ ] HTML template（含邀請者名稱、trip 名稱、接受連結）
  - [ ] 邀請過期前到期提醒
- [ ] Trip update digest email
  - [ ] 每日 / 每週摘要 cron
  - [ ] HTML template（變更清單）
- [ ] Email template 管理
  - [ ] Go template 或 MJML
  - [ ] i18n 支援（依 user locale）

---

## BE-P2-08｜Outbox Worker 真實消費

- [ ] Worker 從 PostgreSQL outbox_events 輪詢
  - [ ] Polling interval 配置（預設 1s）
  - [ ] Batch size 配置（預設 50）
  - [ ] Status = 'pending' AND available_at <= now()
- [ ] 消費處理
  - [ ] 寫入 notification（in-app 通知）
  - [ ] 觸發 FCM push（呼叫 BE-P2-06）
  - [ ] 觸發 email 發送（呼叫 BE-P2-07）
  - [ ] 同步 Firebase shadow（若需要）
  - [ ] 發送 analytics event
- [ ] 錯誤處理
  - [ ] 消費失敗 → retry_count + 1 + 指數退避
  - [ ] 超過 max retry → 進 DLQ（status = 'dead'）
  - [ ] DLQ 告警 + 手動重試 API
- [ ] Worker lifecycle
  - [ ] Graceful shutdown（SIGTERM → 完成目前 batch → 退出）
  - [ ] Health check endpoint
  - [ ] Metrics（消費速率、queue 深度、DLQ 數量）

---

## BE-P2-09｜Docker Compose 本地開發

- [ ] docker-compose.yml
  - [ ] Go API service（含 air hot reload）
  - [ ] PostgreSQL 15+
  - [ ] Redis 7+
  - [ ] Supabase local dev（`supabase start`）
- [ ] 開發配置
  - [ ] .env.local 範例檔
  - [ ] Volume mount for hot reload
  - [ ] Health check 配置
- [ ] Seed data 腳本
  - [ ] 測試用 user + trip + itinerary
  - [ ] 測試用 budget + expenses
  - [ ] 測試用 notifications

---

## BE-P2-10｜CI/CD Production Pipeline

- [ ] GitHub Actions workflow
  - [ ] Go lint（golangci-lint）
  - [ ] Go test（含 coverage report）
  - [ ] Go build
  - [ ] gofmt check
- [ ] Docker 建構
  - [ ] multi-stage Dockerfile
  - [ ] Docker image build + push to registry
  - [ ] Image tag 策略（commit SHA + semver）
- [ ] Database migration
  - [ ] 自動化 migration（golang-migrate）
  - [ ] Migration dry-run 驗證
  - [ ] Rollback 腳本
- [ ] 部署
  - [ ] Staging 環境部署
  - [ ] Production 環境部署（Blue-green / Canary）
  - [ ] Rollback playbook
  - [ ] Health check + readiness probe
