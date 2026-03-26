# 🎨 前端架構文檔

> 依據 `frontend-construction.md` 與實際程式碼分析，記錄前端各頁面的實作狀態與剩餘任務。
> 更新日期：2026-03-26

---

## 概覽

前端採用 **Vite + React + TypeScript + TailwindCSS** 建構，資料層使用 **TanStack Query + Zustand**，路由使用 **React Router v6**。

### 頁面清單（ShellNav 導覽列）

| # | 頁面 | 路由 | 實作狀態 | 說明 |
|---|------|------|---------|------|
| 1 | Dashboard | `/` | ✅ 已實作 | Trip 列表、建立表單（Zod 驗證）、即將出發 countdown、最近活動、快速存取 |
| 2 | Trip Overview | `/trips/:tripId` | ✅ 已實作 | Trip info、成員管理 (RBAC)、edit 表單、add member (Zod)、角色修改、移除成員 |
| 3 | Itinerary | `/trips/:tripId/itinerary` | ✅ 已實作 | 按天顯示、@dnd-kit 拖拉排序（跨日）、inline 編輯、新增/刪除、optimistic UI |
| 4 | Budget | `/trips/:tripId/budget` | ✅ 已實作 | 預算 profile 設定、支出 CRUD、分類條狀圖、gauge 圓環圖、per-day trend 折線圖、over-budget 警告 |
| 5 | Map | `/trips/:tripId/map` | ✅ 已實作 | Mapbox GL JS SDK、POI 搜尋、marker clustering、雙向聯動、route estimation、polyline 繪製 |
| 6 | AI Planner | `/trips/:tripId/ai-planner` | ✅ 已實作 | 條件表單（含 user preference hydration）、tag input、滑桿、planning job 進度輪詢、draft 比較、adopt dialog |
| 7 | Notifications | `/notifications` | ✅ 已實作 | 通知清單、已讀/未讀切換、批次已讀、deep-link 導航、旅程已刪除提示、清理已讀通知 |
| 8 | Settings | `/settings` | ✅ 已實作 | Profile CRUD、偏好設定、通知偏好、LLM Provider 管理（含連線測試）、OAuth binding、帳號刪除、密碼重設 |

### 基礎設施

| 模組 | 狀態 | 說明 |
|------|------|------|
| App Shell | ✅ | BottomSheet、Toast、Modal、LoadingOverlay、OfflineBanner、SyncStatusBar、NotificationBell |
| Auth | ✅ | Magic Link 登入、token refresh、跨 tab 同步登出、invite context、RBAC hook |
| Offline / Sync | ✅ | IndexedDB 持久化、mutation queue、衝突提示、bootstrap 差異同步 |
| FCM Push | ✅ | Firebase SDK、權限請求、token 上傳、前景/背景推播 |
| Analytics | ✅ | 統一 client、離線緩存補送 |
| E2E Tests | ✅ | Playwright 完整流程覆蓋 |

---

## Phase 3：待實作功能

> Phase 1（基礎 UI + Mock API）及 Phase 2（真實整合 + UX 打磨）均已完成。
> Phase 3 著重在：前端與**真實後端 API**的對接、生產環境打磨、以及用戶回饋後的功能補強。

### FE-P3-01｜真實後端 API 對接

所有 API 目前走 in-memory backend，需待後端 Phase 2 持久化完成後逐步切換。

- **Trip / Itinerary / Budget** — 後端已部分 PostgreSQL 化（Trip CRUD 完成），但 Itinerary、AI Plan、Users 等仍為 in-memory
- **Auth** — sessions 仍為 in-memory，需等後端 PostgreSQL sessions table
- **Map** — 後端 mock adapter，需等真實 Google Maps / Mapbox provider
- **AI Planner** — 後端 mock LLM provider，需等真實 OpenAI / Anthropic / Google 整合
- **Notification / FCM** — 後端 mock FCM，需等真實 Firebase Admin SDK
- **Email** — Magic Link / Invite email 目前 console log，需等真實 Email provider

### FE-P3-02｜Itinerary 功能細化

- 設定 item 的 startAt / endAt（目前僅有 allDay toggle）
- 顯示交通時間卡片（連接 Map route estimation）
- 時間衝突警告 UI（橙色邊框 + tooltip）
- AI 草案 vs 正式版差異 highlight

### FE-P3-03｜Map 與 Itinerary 聯動

- 地圖上的 marker 應顯示 itinerary item，非 POI 搜尋結果
- 單日 / 全旅程 POI 切換
- 點擊 itinerary item → 地圖平移
- 點擊 marker → 聯動 itinerary 高亮

### FE-P3-04｜Budget 進階功能

- 幣別轉換 UI（使用後端匯率快照）
- per-person 分攤計算（結合 travelersCount）
- expense 連結 itinerary item（linkedItemId）
- 匯率來源日期 tooltip

### FE-P3-05｜Trip Wizard 強化

- 步驟式 wizard（Step 1: 基本資訊 → Step 2: 偏好 → Step 3: 預算）
- 風格（pace）選擇 UI
- 封面圖片上傳

### FE-P3-06｜成員管理強化

- 邀請連結（share link）UI
- 邀請清單展示（pending / accepted / expired）
- 撤銷邀請按鈕
- 邀請 email 預覽

### FE-P3-07｜PWA 與效能優化

- Lighthouse 效能優化（LCP、CLS）
- 資源預載（字型、關鍵 CSS）
- 路由層級 code splitting 優化
- SW 快取策略調整（stale-while-revalidate → network-first for API）

### FE-P3-08｜Accessibility 補足

- 所有互動元素 aria 屬性
- 鍵盤導航（Tab / Enter / Escape）
- 焦點管理（Modal 開啟/關閉、Toast 播報）
- 色彩對比度達 WCAG AA
