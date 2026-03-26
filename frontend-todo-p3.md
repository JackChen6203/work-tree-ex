# 📦 前端開發進度 Todo List — Phase 3

> Phase 1（基礎 UI + Mock）及 Phase 2（真實整合 + UX 打磨）均已完成。
> Phase 3 目標：真實後端對接、功能細化、生產環境打磨。
> 更新日期：2026-03-26

---

## FE-P3-01｜真實後端 API 對接

> 待後端 Phase 2 各模組持久化完成後逐步切換。

### Itinerary API 對接
- [ ] 確認後端 Itinerary days/items PostgreSQL 遷移完成
- [ ] 前端 `itinerary-api.ts` base URL 切換至真實 API
- [ ] 驗證 CRUD 操作（create / patch / delete / bulk reorder）
- [ ] 驗證樂觀鎖 409 衝突處理
- [ ] 驗證時間衝突 warning response 顯示

### Auth / Session API 對接
- [ ] 確認後端 Sessions PostgreSQL 遷移完成
- [ ] 前端 refresh token 真實 rotation
- [ ] 確認 401 全域攔截 → 安全登出

### AI Planner API 對接
- [ ] 確認後端 AI Plan 三表 PostgreSQL 遷移完成
- [ ] 確認後端 LLM Provider 真實呼叫可用
- [ ] 前端 planning job 進度輪詢 → 真實 job status
- [ ] Draft 採用 → 真實寫入 itinerary

### Users / Preferences API 對接
- [ ] 確認後端 Users/Preferences PostgreSQL 遷移完成
- [ ] Settings 頁 profile CRUD 切換至真實 API
- [ ] Settings 頁偏好設定切換至真實 API
- [ ] 帳號刪除 → 真實 API（目前僅 toast）

### Map / Place API 對接
- [ ] 確認後端 Map Provider 真實呼叫（Google Maps / Mapbox）
- [ ] 前端 POI 搜尋 → 真實搜尋結果
- [ ] Route estimation → 真實路線資料

### FCM Push 對接
- [ ] 確認後端 FCM tokens PostgreSQL 寫入
- [ ] 確認後端 Firebase Admin SDK 初始化
- [ ] 前端 token 上傳 → 真實 `POST /fcm-tokens`

### Email 對接
- [ ] 確認後端 Email provider 整合完成
- [ ] Magic Link email → 真實發送（非 console log）
- [ ] Invite email → 真實發送

---

## FE-P3-02｜Itinerary 功能細化

- [ ] Item startAt / endAt 時間選擇 UI（time picker）
- [ ] 顯示 item 持續時間計算
- [ ] 交通時間卡片（連接 Map route estimation）
- [ ] 時間衝突 / 重疊警告 UI
  - [ ] 橙色邊框 + ⚠️ icon
  - [ ] Hover tooltip 顯示衝突 item 名稱
  - [ ] 儲存前 warning 確認
- [ ] AI 草案 vs 正式版差異 highlight
  - [ ] 新增項目綠色標示
  - [ ] 刪除項目紅色標示
  - [ ] 修改項目藍色標示
- [ ] item 營業時間顯示（來自 place_snapshots）

---

## FE-P3-03｜Map 與 Itinerary 深度聯動

- [ ] 地圖 marker 顯示當前 trip 的 itinerary items（非 POI 搜尋結果）
- [ ] 單日 / 全旅程 POI 切換 toggle
- [ ] 點擊 itinerary item → 地圖平移並高亮對應 marker
- [ ] 點擊 map marker → itinerary 清單高亮對應 item
- [ ] Route polyline 連接 itinerary item 之間
- [ ] 地圖上新增 POI → 直接建立 itinerary item

---

## FE-P3-04｜Budget 進階功能

- [ ] 幣別轉換 UI（使用後端匯率快照 API）
- [ ] Per-person 分攤計算（自動帶入 travelersCount）
- [ ] Expense 連結 itinerary item（linkedItemId 選擇器）
- [ ] 匯率來源 + 快照日期 tooltip
- [ ] 多幣別支出自動換算為 trip 幣別
- [ ] 預算建議（AI Planner 入口優化）

---

## FE-P3-05｜Trip Wizard 強化

- [x] 步驟式 wizard UI（Step indicator）
  - [x] Step 1：名稱 + 目的地 + 出發地
  - [x] Step 2：日期 + 時區 + 人數
  - [x] Step 3：幣別 + 預算 + 風格
- [x] 風格（pace）視覺選擇卡片
- [ ] 封面圖片上傳（或生成）
- [x] Wizard 完成後自動導向 trip overview

---

## FE-P3-06｜成員管理強化

- [ ] Share link 建立 / 列表 / 撤銷 UI
- [ ] 邀請清單展示（pending / accepted / expired）
- [ ] 撤銷邀請按鈕（含確認 dialog）
- [ ] 邀請 email 預覽 UI
- [ ] 重新邀請已過期的邀請

---

## FE-P3-07｜PWA 與效能優化

- [ ] Lighthouse 效能評分 ≥ 90
- [ ] 字型預載（Google Fonts: Inter）
- [ ] 關鍵 CSS inline
- [ ] 路由層級 lazy loading 確認
- [ ] SW 快取策略：API → network-first，靜態資源 → cache-first
- [ ] Image lazy loading（封面圖、地圖預覽）
- [ ] Bundle size 分析 + tree shaking 確認

---

## FE-P3-08｜Accessibility 補足

- [ ] 所有互動元素 `aria-label` / `aria-describedby`
- [ ] Modal / BottomSheet 焦點管理（開啟時 focus trap、關閉時恢復）
- [ ] Toast 使用 `role="status"` + `aria-live="polite"`
- [ ] 色彩對比度 WCAG AA（4.5:1 for text）
- [ ] 鍵盤導航：Tab / Shift+Tab / Enter / Escape
- [ ] 拖拉排序 accessibility（screen reader 播報排序結果）
- [ ] Skip to main content link
