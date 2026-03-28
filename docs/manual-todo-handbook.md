# Manual TODO 操作手冊（Frontend + Backend）

本手冊對應 `frontend-todo.md` 的 Manual TODO，逐項說明要去哪裡申請金鑰、要填到哪個檔案、以及如何驗證是否生效。

---

## 0) 先備：建立環境檔

1. 複製後端環境檔  
   - `cp .env.example .env`
2. 複製前端環境檔  
   - `cp apps/web/.env.example apps/web/.env.local`

---

## 1) 前端 Vite 參數（`apps/web/.env.local`）

### 1.1 API 與登入顯示控制

- `VITE_API_BASE_URL=http://localhost:8080`（本機）或你的 API 網域
- `VITE_OAUTH_PROVIDERS=google`（可多個，逗號分隔）
- `VITE_ENABLE_MAGIC_LINK_AUTH=false`（正式建議）

### 1.2 Mapbox（前端地圖 SDK）

申請位置：
- https://www.mapbox.com/  → Sign up / Sign in  
- Token 管理頁（Dashboard > Access tokens）

步驟：
1. 建立或使用既有 public token（pk 開頭）。
2. 將 token 填到：`VITE_MAPBOX_ACCESS_TOKEN=...`
3. 重新啟動前端：`cd apps/web && npm run dev`

驗證：
- 打開地圖頁，確認不是 fallback 地址清單，而是 Mapbox 地圖畫面。

### 1.3 Firebase Web Push（前端）

申請位置：
- Firebase Console: https://console.firebase.google.com/

步驟：
1. 建立/選擇 Firebase 專案。
2. Project Settings > General > Your apps > 新增 Web app。
3. 取得並填入：
   - `VITE_FIREBASE_API_KEY`
   - `VITE_FIREBASE_AUTH_DOMAIN`
   - `VITE_FIREBASE_PROJECT_ID`
   - `VITE_FIREBASE_STORAGE_BUCKET`
   - `VITE_FIREBASE_MESSAGING_SENDER_ID`
   - `VITE_FIREBASE_APP_ID`
4. Cloud Messaging > Web configuration 取得 VAPID key，填入：
   - `VITE_FIREBASE_VAPID_KEY`
5. 重新啟動前端。

驗證：
- 設定頁啟用 push 後可成功拿到 token，且前端可呼叫 `POST /api/v1/fcm-tokens`。

---

## 2) 後端參數（`.env`）

### 2.1 資料庫（PostgreSQL / Supabase）

填寫：
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`
- （建議）`MIGRATE_DATABASE_URL`

驗證：
1. `docker compose --profile tools run --rm migrate`
2. API 啟動後無 DB 連線錯誤。

### 2.2 JWT 與加密

填寫：
- `JWT_SECRET`（長字串隨機值）
- `LLM_ENCRYPTION_KEY`（32-byte raw 或 base64 32-byte）

建議：
- 用密碼管理器或 KMS 生成，不要把明碼 commit 到 repo。

### 2.3 Google OAuth（後端交換 code）

申請位置：
- Google Cloud Console: https://console.cloud.google.com/

步驟：
1. 建立/選擇 GCP 專案。
2. APIs & Services > OAuth consent screen 完成設定。
3. Credentials > Create Credentials > OAuth client ID（Web application）。
4. 設定 Authorized redirect URI（對應部署網域）。
5. 填入 `.env`：
   - `OAUTH_GOOGLE_CLIENT_ID=...`
   - `OAUTH_GOOGLE_CLIENT_SECRET=...`

驗證：
- 走 `/api/v1/auth/oauth/google/start`，完成登入並能回到前端。

### 2.4 地圖 Provider（後端真實呼叫）

Google Maps key 申請：
- https://console.cloud.google.com/ （啟用 Places / Geocoding / Directions 對應 API）

Mapbox key 申請：
- https://www.mapbox.com/

填寫 `.env`（至少一個）：
- `GOOGLE_MAPS_API_KEY=...`
- `MAPBOX_API_KEY=...`
- `MAP_PRIMARY_PROVIDER=google` 或 `mapbox`

驗證：
- API `GET /api/v1/maps/search`、`POST /api/v1/maps/routes` 回傳真實 provider 結果。

### 2.5 FCM 推播（後端）

建議新制（Firebase Admin SDK）：
- `FCM_SERVICE_ACCOUNT_JSON` 或 `FCM_SERVICE_ACCOUNT_FILE`
- `FCM_PROJECT_ID`

舊制 fallback（可選）：
- `FCM_SERVER_KEY`

步驟（Admin SDK）：
1. Firebase Console > Project settings > Service accounts。
2. 產生 service account key JSON（妥善保管）。
3. 將 JSON 放到安全位置（或直接填 `FCM_SERVICE_ACCOUNT_JSON`）。
4. 設定 `FCM_PROJECT_ID`。

驗證：
- API/worker log 出現 Firebase Admin gateway 啟用訊息。
- 測試通知可送達，且無效 token 會被正確處理。

### 2.6 Email Provider（Magic Link / Invite）

Resend 申請：
- https://resend.com/

SendGrid 申請：
- https://sendgrid.com/

填寫 `.env`：
- `EMAIL_PROVIDER_PRIMARY=resend` 或 `sendgrid`
- `EMAIL_PROVIDER_FALLBACK=...`（可選）
- `EMAIL_FROM=no-reply@your-domain.com`
- `RESEND_API_KEY=...` 或 `SENDGRID_API_KEY=...`

驗證：
- request magic link 會寄出真實 email（非 console log）。
- trip invite 會寄送邀請信。

---

## 3) CI/CD 與部署 Secrets

GitHub repo secrets（依 `DEPLOYMENT.md`）：
- `ORACLE_SSH_KEY`
- `APP_ENV_FILE`（完整 production `.env` 內容）
- `MIGRATE_DATABASE_URL`

步驟：
1. GitHub Repo > Settings > Secrets and variables > Actions。
2. 新增以上 secrets。
3. 觸發 workflow，確認 migration + deploy 成功。

---

## 4) 本機 CI 執行阻塞修復（目前已知）

現況：Node 啟動缺 `libicui18n.74.dylib`。

建議處理：
1. 重新安裝與當前 Homebrew ICU 相容的 Node（或重裝 Node）。
2. 確認 `node -v` 可正常執行後，再跑：
   - `cd apps/web && npm run lint`
   - `cd apps/web && npm run test`
   - `cd apps/web && npm run build`

---

## 5) 最後驗收清單

1. 前端可正常登入（Magic Link / OAuth）
2. Itinerary CRUD + 409 衝突提示正常
3. AI Planner 建立與採用正常
4. Map 搜尋/路線回傳真實資料
5. FCM token 註冊與推播可用
6. Email（magic link/invite）實際寄送成功
7. Lighthouse（FE-P3-07）分數達標（>=90）

