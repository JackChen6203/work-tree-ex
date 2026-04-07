# Manual Todo 操作手冊

本手冊對應 [`manual-todo.md`](../manual-todo.md) 每一項待辦，提供實際操作路徑與申請網址。

---

## 1) GitHub Actions deployment secrets

### 1.1 `ORACLE_SSH_KEY`

用途：讓 GitHub Actions 可 SSH 進入 `opc@217.142.247.83` 執行部署。

步驟：

1. 在本機產生 deploy key（建議專用）：
   ```bash
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy
   ```
2. 把公鑰內容加到伺服器 `~/.ssh/authorized_keys`：
   ```bash
   cat ~/.ssh/github_actions_deploy.pub
   ```
3. 把私鑰內容貼到 GitHub Secret `ORACLE_SSH_KEY`：
   - Repo Secrets 頁面：
     [https://github.com/JackChen6203/work-tree-ex/settings/secrets/actions](https://github.com/JackChen6203/work-tree-ex/settings/secrets/actions)
   - 內容來源：
     ```bash
     cat ~/.ssh/github_actions_deploy
     ```

### 1.2 `APP_ENV_FILE`

用途：部署流程會把這個值完整寫到伺服器的 `.env`。

步驟：

1. 以專案根目錄 `.env.example` 為基礎建立 production 版 `.env`。
2. 逐項替換敏感值（DB/Redis/JWT/API keys）。
3. 將完整檔案內容貼到 GitHub Secret `APP_ENV_FILE`。
   - URL：
     [https://github.com/JackChen6203/work-tree-ex/settings/secrets/actions](https://github.com/JackChen6203/work-tree-ex/settings/secrets/actions)

### 1.3 `MIGRATE_DATABASE_URL`

用途：GitHub Actions 在部署前跑 migration。

步驟：

1. 準備完整 PostgreSQL 連線字串：
   ```text
   postgresql://<user>:<url-encoded-password>@<host>:<port>/<db>?sslmode=require
   ```
2. 若密碼含 `@:/?`，先 URL encode。
3. 將字串貼到 `MIGRATE_DATABASE_URL`（同上 Secrets 頁）。

---

## 2) Staging / Production 分流 Secrets（建議）

對應欄位：
- `STAGING_SSH_KEY`
- `STAGING_APP_ENV_FILE`
- `STAGING_MIGRATE_DATABASE_URL`
- `PRODUCTION_SSH_KEY`
- `PRODUCTION_APP_ENV_FILE`
- `PRODUCTION_MIGRATE_DATABASE_URL`

步驟：

1. 到同一 Secrets 頁面建立上述 key：
   [https://github.com/JackChen6203/work-tree-ex/settings/secrets/actions](https://github.com/JackChen6203/work-tree-ex/settings/secrets/actions)
2. staging 與 production 使用不同主機金鑰與不同 DB URL。
3. 若不設定 staging/prod 專用 SSH key 或 app env，workflow 會 fallback 到 `ORACLE_SSH_KEY / APP_ENV_FILE`。`PRODUCTION_MIGRATE_DATABASE_URL` 可 fallback 到 `MIGRATE_DATABASE_URL`；`STAGING_MIGRATE_DATABASE_URL` 不設定時 staging migration 會略過。

---

## 3) Supabase（RLS + 前端 anon key）

### 3.1 取得 `VITE_SUPABASE_URL` / `VITE_SUPABASE_ANON_KEY`

步驟：

1. 登入 Supabase Dashboard：
   [https://supabase.com/dashboard](https://supabase.com/dashboard)
2. 進入你的 Project。
3. `Project Settings` → `API`。
4. 讀取：
   - `Project URL` → `VITE_SUPABASE_URL`
   - `anon public` key → `VITE_SUPABASE_ANON_KEY`
5. 寫入 `apps/web/.env.local`（本機）或前端部署環境變數。

### 3.2 `SUPABASE_SERVICE_ROLE_KEY`（後端專用）

步驟：

1. 同樣在 `Project Settings` → `API` 取得 `service_role` key。
2. 只放在後端 `.env` / Secret Manager。
3. 禁止放在任何 `VITE_*` 或前端 bundle 可讀位置。

---

## 4) 外部服務金鑰（Map / Push / Email）

### 4.1 Google Maps API (`GOOGLE_MAPS_API_KEY`)

申請入口：
- [https://console.cloud.google.com/](https://console.cloud.google.com/)

步驟：

1. 建立或選擇 GCP Project。
2. 啟用 APIs（至少）：
   - Places API
   - Geocoding API
   - Directions API / Routes API（依你實作）
3. `APIs & Services` → `Credentials` → `Create credentials` → `API key`。
4. 設定 key restrictions（HTTP referrer / IP / API allow list）。
5. 寫入根目錄 `.env` 的 `GOOGLE_MAPS_API_KEY`。

### 4.2 Mapbox API (`MAPBOX_API_KEY`)

申請入口：
- [https://account.mapbox.com/](https://account.mapbox.com/)

步驟：

1. 登入 Mapbox。
2. 進 `Access tokens`。
3. 建立 token（建議建立專用 token + scope 限制）。
4. 寫入根目錄 `.env`：`MAPBOX_API_KEY`。

### 4.3 Firebase Push（後端 + 前端）

入口：
- Firebase Console: [https://console.firebase.google.com/](https://console.firebase.google.com/)
- GCP Service Accounts: [https://console.cloud.google.com/iam-admin/serviceaccounts](https://console.cloud.google.com/iam-admin/serviceaccounts)

步驟（後端）：

1. 在 Firebase 專案啟用 Cloud Messaging。
2. 建立 service account（或使用既有 account），下載 JSON key。
3. 擇一設定後端 `.env`：
   - `FCM_SERVICE_ACCOUNT_FILE`（檔案路徑），或
   - `FCM_SERVICE_ACCOUNT_JSON`（JSON 字串）
4. 視需要設定 `FCM_PROJECT_ID`。

步驟（前端）：

1. Firebase 專案 `Project settings` → `General` → Web app config。
2. 取得並填入 `apps/web/.env.local`：
   - `VITE_FIREBASE_API_KEY`
   - `VITE_FIREBASE_AUTH_DOMAIN`
   - `VITE_FIREBASE_PROJECT_ID`
   - `VITE_FIREBASE_STORAGE_BUCKET`
   - `VITE_FIREBASE_MESSAGING_SENDER_ID`
   - `VITE_FIREBASE_APP_ID`
   - `VITE_FIREBASE_VAPID_KEY`

### 4.4 Email Provider（Resend / SendGrid）

Resend：
- 註冊/控制台：[https://resend.com/](https://resend.com/)

SendGrid：
- 註冊/控制台：[https://app.sendgrid.com/](https://app.sendgrid.com/)

步驟（共通）：

1. 建立 API key。
2. 驗證寄件網域（SPF/DKIM）。
3. 在根 `.env` 設定：
   - `EMAIL_PROVIDER_PRIMARY=resend` 或 `sendgrid`
   - `RESEND_API_KEY` 或 `SENDGRID_API_KEY`
   - `EMAIL_FROM=<verified-sender>`
4. 若要備援，設定 `EMAIL_PROVIDER_FALLBACK`。

---

## 5) Rollback readiness（演練）

### 5.1 確認 production 路徑

1. SSH 到主機：
   ```bash
   ssh opc@217.142.247.83
   ```
2. 確認專案存在：
   ```bash
   ls -la /home/opc/apps/work-tree-ex
   ```

### 5.2 確認 Docker Compose / 連線

```bash
docker compose version
docker network ls
```

### 5.3 執行一次 production deploy dry-run

1. 到 GitHub Actions：
   [https://github.com/JackChen6203/work-tree-ex/actions/workflows/deploy.yml](https://github.com/JackChen6203/work-tree-ex/actions/workflows/deploy.yml)
2. `Run workflow`：
   - `target=production`
   - `strategy=rolling`

### 5.4 執行一次 rollback dry-run

1. 同一 workflow `Run workflow`：
   - `target=rollback`
   - `rollback_ref` 可先留空（使用 previous successful sha）
2. 觀察 workflow 完成且 `/healthz` 正常。
