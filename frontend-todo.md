# 📦 前端開發進度 Todo List

> 依據 `frontend-construction.md` 規格文件，追蹤各模組開發進度。
> 更新日期：2026-03-24

---

## FE-01｜App Shell

- [x] Session hydration（啟動時檢查登入狀態）
- [x] Route guard（依登入狀態決定可訪問路由）
- [x] Shell navigation active state isolation（無 active trip 時不再共用 `/` 路由）
- [x] Global error boundary
- [x] Toast 元件（右上角 3s 自動消失）
- [x] Modal 元件（全螢幕遮罩確認 dialog）
- [x] Bottom sheet 元件（行動裝置次要操作選單）
- [x] Loading overlay
- [x] Mobile-first 響應式排版
- [x] PWA 安裝提示
- [x] Service worker 更新通知
- [x] Zustand global UI state（`AppUIState`）

### 邊界個案
- [x] Session hydration 失敗 → 跳登入頁
- [x] Token 過期但路由已渲染 → 捕捉 401 觸發登出
- [x] 離線時開啟 app → 顯示離線 banner
- [x] Service worker 更新失敗 → 靜默重試

註記：Global error boundary 目前覆蓋 render tree 內的未處理錯誤與路由畫面崩潰；事件處理中的錯誤仍需各模組自行捕捉與提示。
註記：Toast 已符合右上角與 3 秒自動消失規格，並保留 `pushToast("message")` 舊介面相容；後續可逐步把成功/警告/錯誤情境補上對應 `type`。
註記：Modal 與 Loading overlay 已接入 `AppShell`，目前先用於登出確認與非同步登出流程；`Bottom sheet` 與完整 `AppUIState` 收斂仍待後續整合。
註記：Bottom sheet 已作為手機版主導覽入口，桌機仍維持 header pills；整體 mobile-first 版面細修仍未完成，因此 `Mobile-first 響應式排版` 暫不打勾。

---

## FE-02｜Auth

- [x] Email Magic Link 登入 UI
- [x] 等待 Magic Link 頁面（含重發按鈕）
- [x] OAuth provider 登入 UI（預留介面）
- [x] Access token 自動 refresh（靜默換新）
- [x] Refresh token 過期安全登出
- [x] 邀請連結 pre-auth / post-auth 流程
- [x] 跨 tab session 同步（多 tab 同步登出）
- [x] Role-based permission hook（`TripPermission`）

### 邊界個案
- [x] Magic link 過期 → 提示重發
- [x] 多 tab 同時登入 → session 共享
- [x] 邀請連結未登入 → 記錄 context，登入後繼續
- [x] Refresh 失敗 → 清 session，保留 redirect hint

註記：Auth 頁已拆成輸入 email 與等待 Magic Link 兩段式流程，並保留開發模式驗證碼輸入；token refresh 與 invite context 仍未完成。
註記：跨 tab session 同步目前採 `localStorage` 廣播登入/登出事件；access token refresh、401 全域攔截與 refresh 失敗安全登出仍未完成。

---

## FE-03｜Workspace / Membership

- [x] Trip 清單頁（含角色標示）
- [x] 每個 trip 邀請狀態顯示
- [x] Role-based UI 邏輯封裝
- [x] 管理員修改成員角色 UI
- [x] 撤銷邀請 / 移除成員 UI
- [x] 空狀態「建立旅程」CTA

### 邊界個案
- [x] 無任何 trip → 顯示「建立旅程」CTA
- [x] 被踢出 trip → 403 處理，回首頁
- [x] 邀請過期 → 標示過期 + 重新邀請按鈕
- [x] 角色降級後停留編輯頁 → mutation 403 後更新 UI

---

## FE-04｜Trip

- [x] Trip 建立 Wizard（名稱、目的地、日期、時區、人數、幣別、風格）
- [x] Trip Overview 頁（基本資訊、封面、成員列表）
- [x] Date range 修改（含縮短旅程衝突警告）
- [x] Timezone 正確顯示與跨日邏輯
- [x] Destination metadata 展示
- [x] Trip 狀態管理 UI（draft / active / archived）
- [x] Trip name 前端即時驗證（≤200 字）

### 邊界個案
- [x] Date range 縮短致 item 越界 → 顯示衝突清單
- [x] Timezone 跨夏令時間 → 正確計算
- [x] 修改 trip 版本衝突 → 提示重新整理

---

## FE-05｜Itinerary

- [x] 按天顯示 itinerary days / items
- [x] 拖拉變更順序（含跨日移動）
- [x] 設定開始時間、結束時間、全天
- [x] 顯示交通時間與預估移動成本
- [x] 顯示時間衝突 / 重疊警告
- [x] 顯示 AI 草案與正式版差異 highlight
- [x] Optimistic UI 更新
- [x] Item 新增 / 編輯 / 刪除 UI

### 邊界個案
- [x] 時間重疊 → 視覺標示，不阻擋儲存
- [x] 拖拉排序時離線 → mutation 排入 offline queue
- [x] 409 version conflict → revert optimistic state
- [x] startAt / endAt 為空 → 純排序卡片顯示

---

## FE-06｜Budget

- [x] 設定 total / per-person / per-day 預算 UI
- [x] 依分類統計（條狀圖 / 百分比）
- [x] 估算 vs 實際差異顯示
- [x] 幣別設定與換算來源標示
- [x] 新增實際支出 UI
- [x] 「以預算生成行程」入口（觸發 AI Planner）

### 邊界個案
- [x] 總預算未填但有每日預算 → 換算提示總預算
- [x] 幣別不支援 → 前端限制選項為白名單
- [x] 超出預算 → 顯示超支警告
- [x] 匯率快照過舊 → 顯示資料日期提醒

---

## FE-07｜Map

- [x] 地圖與 itinerary 清單雙向聯動
- [x] POI 搜尋（關鍵字、地點自動補全）
- [x] 顯示單日 / 全旅程點位
- [x] 步行 / 開車 / 大眾運輸路線候選
- [x] Provider 抽象（`MapProviderAdapter` 介面）
- [x] 大量點位 clustering

### 邊界個案
- [x] 地圖 SDK 載入失敗 → Fallback 地址清單
- [x] Provider API 無回應 → 顯示「路線暫時無法取得」
- [x] 無 GPS / 拒絕定位 → 以目的地為預設中心
- [x] 座標異常 → 過濾不繪製

---

## FE-08｜AI Planner

- [x] 條件輸入 UI（預算、天數、風格、早起/晚起、交通偏好、景點密度、必去點、禁忌）
- [x] Planning job 進度顯示
- [x] 多套 draft 清單，支援比較
- [x] Validation warnings 顯示
- [x] 手動採用 draft（confirm dialog）
- [x] 前端 debounce 防止重複提交

### 邊界個案
- [x] 規劃中途離開 → Job 後端繼續，完成後推播
- [x] Draft 為 invalid → 只能查看原因，不顯示採用按鈕
- [x] 採用 draft 時 trip 有衝突 → 提示重新驗證

---

## FE-09｜Offline / Sync

- [x] 快取最近 trip（Service Worker + IndexedDB）
- [x] 離線時排入 local mutation queue
- [x] 待同步狀態 badge
- [x] 上線後按序重送 mutation
- [x] 衝突提示 UI
- [x] Bootstrap API 差異同步
- [x] API key 絕不進入 persistent cache

### 邊界個案
- [x] 離線多人同改同一 item → 衝突解決 UI
- [x] Mutation queue 中有過期操作 → 提示部分失敗
- [x] Service worker 更新失敗 → 不阻擋 app

---

## FE-10｜Notification

- [x] In-app notification center（鈴鐺 + 未讀數量）
- [x] FCM push 接收與顯示
- [x] 點擊通知 deep-link 導航
- [x] 已讀 / 未讀狀態管理
- [x] 批次已讀

### 邊界個案
- [x] 拒絕推播權限 → 僅 in-app 通知
- [x] 通知對應 trip 已刪除 → 顯示「旅程已不存在」
- [x] FCM token 過期 → 靜默更新

---

## FE-11｜Analytics

- [x] 統一 analytics client
- [x] 事件命名格式 `<domain>.<entity>.<action>`
- [x] 最小必要 context
- [x] 離線事件緩存 → 上線後補送

### 邊界個案
- [x] 事件送出失敗 → 靜默不影響使用者
- [x] 敏感欄位 → bucketize 或 hash

---

# Phase 2：功能完善 + 真實整合 + UX 打磨

> Phase 1 所有模組都以基礎 UI + mock API 整合完成。
> Phase 2 目標：將佔位元件替換為真實功能，補足缺失的互動與整合。

---

## FE-P2-01｜真實拖拉排序（Drag & Drop）

- [ ] 導入 `@dnd-kit/core` 拖拉排序套件
- [ ] Itinerary items 支援拖拉重新排列
- [ ] 支援跨日拖拉（從 Day 1 拖到 Day 2）
- [ ] 拖拉過程中視覺 feedback（ghost card, drop indicator）
- [ ] 拖拉完成後呼叫 reorder API + optimistic UI
- [ ] 行動裝置觸控拖拉支援
- [ ] 不支援 drag 時顯示上/下排序按鈕 fallback

---

## FE-P2-02｜真實地圖 SDK 整合

- [ ] 整合 Mapbox GL JS 或 Google Maps JS SDK
- [ ] `MapProviderAdapter` 實作連接真實 SDK
- [ ] 地圖元件渲染 itinerary POI markers
- [ ] 點擊 marker 聚焦對應 itinerary item
- [ ] 點擊 itinerary item 平移地圖至該點
- [ ] 路線 polyline 繪製
- [ ] 大量點位 marker clustering（真實 SDK 層級）
- [ ] SDK 載入失敗 → fallback 地址清單

---

## FE-P2-03｜表單驗證強化（Zod Schema）

- [x] Trip 建立 wizard 接入 Zod schema 驗證
- [x] Trip name ≤200 字即時驗證 + 錯誤訊息
- [x] Date range 驗證（endDate ≥ startDate）
- [ ] Email 格式驗證（invite member）
- [ ] Budget amount 非負數驗證
- [ ] Expense 表單 Zod 驗證
- [x] LLM API key 格式驗證（`enc_` 前綴 + 最少 16 字）
- [x] 表單錯誤訊息 i18n 化

---

## FE-P2-04｜真實 IndexedDB 離線持久化

- [ ] 導入 `idb` 套件（IndexedDB wrapper）
- [ ] 離線時 trip 資料寫入 IndexedDB 快取
- [ ] Mutation queue 持久化到 IndexedDB（非純記憶體）
- [ ] App 重啟後從 IndexedDB 恢復 pending mutations
- [ ] IndexedDB 資料過期清理（>7 天）
- [ ] API key 絕不寫入 IndexedDB

---

## FE-P2-05｜真實 FCM Push 註冊

- [ ] Firebase SDK 初始化（`firebase/messaging`）
- [ ] 請求推播權限 dialog
- [ ] 取得 FCM token 並上傳至 backend `POST /fcm-tokens`
- [ ] Token 過期時靜默刷新上傳
- [ ] 前景推播處理（顯示 toast）
- [ ] 背景推播處理（Service Worker notification）
- [ ] 使用者拒絕推播 → 僅 in-app 通知

---

## FE-P2-06｜Budget 視覺圖表

- [x] 分類花費條狀圖（CSS bar chart 或 SVG）
- [x] 估算 vs 實際百分比對照圖
- [ ] Per-person 分攤計算顯示
- [ ] Per-day 花費趨勢折線圖
- [x] 匯率來源與快照日期顯示
- [x] Over-budget 紅色警告動畫

---

## FE-P2-07｜Dashboard 空狀態與真實內容

- [x] 移除 "Frontend foundation" 佔位 SurfaceCard
- [x] Trip 列表空狀態 → 大型「建立旅程」CTA + 插圖
- [ ] 最近活動 feed（近期 trip 變更、通知摘要）
- [x] 即將出發 trip countdown
- [ ] 快速存取 widget（最近編輯的 trip）

---

## FE-P2-08｜AI Planner 條件表單完善

- [ ] 條件輸入從 hardcoded → 真實表單（連接 user preferences）
- [ ] Budget 從 trip budget profile 自動帶入
- [ ] Must-visit / avoid 支援 tag input（新增/刪除 chips）
- [ ] Wake pattern / POI density 滑桿選擇器
- [ ] 提交後顯示真實 planning job 進度輪詢
- [ ] Draft 比較 side-by-side diff 視圖
- [ ] Adopt 確認 dialog 使用 `openAdoptDraftModal`

---

## FE-P2-09｜Notification Bell + Unread Badge

- [x] App Shell header 新增鈴鐺圖示
- [x] 鈴鐺上方 unread count badge
- [x] 點擊鈴鐺 → 下拉通知面板（或跳轉 /notifications）
- [ ] 通知 deep-link → 正確導航到 trip/itinerary/budget
- [ ] 通知刪除的 trip → 顯示「旅程已刪除」提示

---

## FE-P2-10｜Settings 頁面打磨

- [x] 帳號刪除功能（含二次確認 confirm dialog）
- [ ] Password / 社交帳號綁定管理（若啟用）
- [x] 語言切換即時生效（i18n locale 切換）
- [x] LLM provider key 儲存前先做連線測試

---

## FE-P2-11｜E2E 測試（Playwright）

- [ ] Login magic link 流程 E2E
- [ ] Trip 建立 → itinerary 編輯 → budget 設定 完整流程 E2E
- [ ] Offline → online 同步 E2E
- [ ] 推播通知 E2E（mock FCM）
- [ ] 跨裝置 responsive 截圖測試

