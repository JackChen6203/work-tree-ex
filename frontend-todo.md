# 📦 前端開發進度 Todo List

> 依據 `frontend-construction.md` 規格文件，追蹤各模組開發進度。
> 更新日期：2026-03-23

---

## FE-01｜App Shell

- [x] Session hydration（啟動時檢查登入狀態）
- [x] Route guard（依登入狀態決定可訪問路由）
- [ ] Global error boundary
- [ ] Toast 元件（右上角 3s 自動消失）
- [ ] Modal 元件（全螢幕遮罩確認 dialog）
- [ ] Bottom sheet 元件（行動裝置次要操作選單）
- [ ] Loading overlay
- [ ] Mobile-first 響應式排版
- [ ] PWA 安裝提示
- [ ] Service worker 更新通知
- [ ] Zustand global UI state（`AppUIState`）

### 邊界個案
- [ ] Session hydration 失敗 → 跳登入頁
- [ ] Token 過期但路由已渲染 → 捕捉 401 觸發登出
- [ ] 離線時開啟 app → 顯示離線 banner
- [ ] Service worker 更新失敗 → 靜默重試

---

## FE-02｜Auth

- [ ] Email Magic Link 登入 UI
- [ ] 等待 Magic Link 頁面（含重發按鈕）
- [ ] OAuth provider 登入 UI（預留介面）
- [ ] Access token 自動 refresh（靜默換新）
- [ ] Refresh token 過期安全登出
- [ ] 邀請連結 pre-auth / post-auth 流程
- [ ] 跨 tab session 同步（多 tab 同步登出）
- [ ] Role-based permission hook（`TripPermission`）

### 邊界個案
- [ ] Magic link 過期 → 提示重發
- [ ] 多 tab 同時登入 → session 共享
- [ ] 邀請連結未登入 → 記錄 context，登入後繼續
- [ ] Refresh 失敗 → 清 session，保留 redirect hint

---

## FE-03｜Workspace / Membership

- [ ] Trip 清單頁（含角色標示）
- [ ] 每個 trip 邀請狀態顯示
- [ ] Role-based UI 邏輯封裝
- [ ] 管理員修改成員角色 UI
- [ ] 撤銷邀請 / 移除成員 UI
- [ ] 空狀態「建立旅程」CTA

### 邊界個案
- [ ] 無任何 trip → 顯示「建立旅程」CTA
- [ ] 被踢出 trip → 403 處理，回首頁
- [ ] 邀請過期 → 標示過期 + 重新邀請按鈕
- [ ] 角色降級後停留編輯頁 → mutation 403 後更新 UI

---

## FE-04｜Trip

- [ ] Trip 建立 Wizard（名稱、目的地、日期、時區、人數、幣別、風格）
- [ ] Trip Overview 頁（基本資訊、封面、成員列表）
- [ ] Date range 修改（含縮短旅程衝突警告）
- [ ] Timezone 正確顯示與跨日邏輯
- [ ] Destination metadata 展示
- [ ] Trip 狀態管理 UI（draft / active / archived）
- [ ] Trip name 前端即時驗證（≤200 字）

### 邊界個案
- [ ] Date range 縮短致 item 越界 → 顯示衝突清單
- [ ] Timezone 跨夏令時間 → 正確計算
- [ ] 修改 trip 版本衝突 → 提示重新整理

---

## FE-05｜Itinerary

- [ ] 按天顯示 itinerary days / items
- [ ] 拖拉變更順序（含跨日移動）
- [ ] 設定開始時間、結束時間、全天
- [ ] 顯示交通時間與預估移動成本
- [ ] 顯示時間衝突 / 重疊警告
- [ ] 顯示 AI 草案與正式版差異 highlight
- [ ] Optimistic UI 更新
- [ ] Item 新增 / 編輯 / 刪除 UI

### 邊界個案
- [ ] 時間重疊 → 視覺標示，不阻擋儲存
- [ ] 拖拉排序時離線 → mutation 排入 offline queue
- [ ] 409 version conflict → revert optimistic state
- [ ] startAt / endAt 為空 → 純排序卡片顯示

---

## FE-06｜Budget

- [ ] 設定 total / per-person / per-day 預算 UI
- [ ] 依分類統計（條狀圖 / 百分比）
- [ ] 估算 vs 實際差異顯示
- [ ] 幣別設定與換算來源標示
- [ ] 新增實際支出 UI
- [ ] 「以預算生成行程」入口（觸發 AI Planner）

### 邊界個案
- [ ] 總預算未填但有每日預算 → 換算提示總預算
- [ ] 幣別不支援 → 前端限制選項為白名單
- [ ] 超出預算 → 顯示超支警告
- [ ] 匯率快照過舊 → 顯示資料日期提醒

---

## FE-07｜Map

- [ ] 地圖與 itinerary 清單雙向聯動
- [ ] POI 搜尋（關鍵字、地點自動補全）
- [ ] 顯示單日 / 全旅程點位
- [ ] 步行 / 開車 / 大眾運輸路線候選
- [ ] Provider 抽象（`MapProviderAdapter` 介面）
- [ ] 大量點位 clustering

### 邊界個案
- [ ] 地圖 SDK 載入失敗 → Fallback 地址清單
- [ ] Provider API 無回應 → 顯示「路線暫時無法取得」
- [ ] 無 GPS / 拒絕定位 → 以目的地為預設中心
- [ ] 座標異常 → 過濾不繪製

---

## FE-08｜AI Planner

- [ ] 條件輸入 UI（預算、天數、風格、早起/晚起、交通偏好、景點密度、必去點、禁忌）
- [ ] Planning job 進度顯示
- [ ] 多套 draft 清單，支援比較
- [ ] Validation warnings 顯示
- [ ] 手動採用 draft（confirm dialog）
- [ ] 前端 debounce 防止重複提交

### 邊界個案
- [ ] 規劃中途離開 → Job 後端繼續，完成後推播
- [ ] Draft 為 invalid → 只能查看原因，不顯示採用按鈕
- [ ] 採用 draft 時 trip 有衝突 → 提示重新驗證

---

## FE-09｜Offline / Sync

- [ ] 快取最近 trip（Service Worker + IndexedDB）
- [ ] 離線時排入 local mutation queue
- [ ] 待同步狀態 badge
- [ ] 上線後按序重送 mutation
- [ ] 衝突提示 UI
- [ ] Bootstrap API 差異同步
- [ ] API key 絕不進入 persistent cache

### 邊界個案
- [ ] 離線多人同改同一 item → 衝突解決 UI
- [ ] Mutation queue 中有過期操作 → 提示部分失敗
- [ ] Service worker 更新失敗 → 不阻擋 app

---

## FE-10｜Notification

- [ ] In-app notification center（鈴鐺 + 未讀數量）
- [ ] FCM push 接收與顯示
- [ ] 點擊通知 deep-link 導航
- [ ] 已讀 / 未讀狀態管理
- [ ] 批次已讀

### 邊界個案
- [ ] 拒絕推播權限 → 僅 in-app 通知
- [ ] 通知對應 trip 已刪除 → 顯示「旅程已不存在」
- [ ] FCM token 過期 → 靜默更新

---

## FE-11｜Analytics

- [ ] 統一 analytics client
- [ ] 事件命名格式 `<domain>.<entity>.<action>`
- [ ] 最小必要 context
- [ ] 離線事件緩存 → 上線後補送

### 邊界個案
- [ ] 事件送出失敗 → 靜默不影響使用者
- [ ] 敏感欄位 → bucketize 或 hash
