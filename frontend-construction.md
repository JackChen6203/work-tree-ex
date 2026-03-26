# 📦 前端模組

---

## FE-01｜App Shell

**模式**：全域容器、路由守衛、PWA 安裝入口

### 細部功能
- 啟動時執行 session hydration
- 依登入狀態決定可訪問 route（route guard）
- 提供 global error boundary
- 提供 toast / modal / bottom sheet / loading overlay
- 支援 mobile-first 響應式排版
- PWA 安裝提示與 service worker 更新通知

### 邊界個案
- Session hydration 失敗 → 跳登入頁，不阻塞 UI 渲染
- Token 過期但路由已渲染 → 捕捉 401，觸發登出流程
- 離線時開啟 app → Shell 正常顯示，頂部顯示離線 banner
- Service worker 更新失敗 → 不阻擋 app，後台靜默重試

### 資料結構

```typescript
// Global UI State (Zustand store)
interface AppUIState {
  isOnline: boolean;
  pendingSyncCount: number;
  toasts: Toast[];
  activeModal: ModalType | null;
}

interface Toast {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  durationMs?: number;
}

type ModalType =
  | { type: 'confirm'; payload: ConfirmModalPayload }
  | { type: 'invite'; payload: InviteModalPayload }
  | { type: 'adopt_draft'; payload: AdoptDraftModalPayload };
```

---

## FE-02｜Auth

**模式**：登入/登出流程、Session 管理、Token 自動刷新

### 細部功能
- Email Magic Link / OTP 登入
- OAuth provider 擴充能力（預留介面）
- Access token 自動 refresh（失效前靜默換新）
- Refresh token 過期時安全登出並清 session
- 接受 trip 邀請的 pre-auth / post-auth 流程
- 跨頁面 session 一致化（多 tab 同步登出）
- 集中 role-based permission 判斷（auth policy hook）

### 邊界個案
- Magic link token 過期 → 提示重發，不顯示技術錯誤
- 多 tab 同時登入同帳號 → session 共享，不產生衝突
- 邀請連結點擊時未登入 → 記錄邀請 context，登入後自動繼續
- Refresh 失敗 → 清 session，重導登入，保留 redirect hint

### 資料結構

```typescript
interface Session {
  accessToken: string;          // 短效 JWT，15m～60m
  refreshToken: string;         // httpOnly cookie
  expiresAt: number;            // unix timestamp
  user: UserProfile;
}

interface UserProfile {
  id: string;
  email: string;
  displayName: string;
  locale: string;
  timezone: string;             // IANA timezone string
  currency: string;             // ISO-4217 e.g. "TWD"
}

// 權限 hook 回傳型別
interface TripPermission {
  canEdit: boolean;
  canComment: boolean;
  canViewOnly: boolean;
  canManageMembers: boolean;
  role: 'owner' | 'editor' | 'commenter' | 'viewer';
}
```

---

## FE-03｜Workspace / Membership

**模式**：Trip 清單、成員角色展示、邀請狀態管理

### 細部功能
- 顯示使用者可見 trip 清單（含角色標示）
- 顯示每個 trip 的邀請狀態
- 封裝 role-based UI 邏輯
- 管理員可修改成員角色
- 撤銷邀請 / 移除成員

### 邊界個案
- 使用者無任何 trip → 顯示「建立旅程」CTA，不空白
- 被踢出 trip → 403 處理，清除 trip 快取並回首頁
- 邀請過期 → 標示過期，提供重新邀請按鈕（僅 owner 可見）
- 角色降級後仍停留在編輯頁 → 次次 mutation 回 403 後更新 UI

### 資料結構

```typescript
interface TripListItem {
  id: string;
  name: string;
  destinationText: string;
  startDate: string;            // YYYY-MM-DD
  endDate: string;
  status: 'draft' | 'active' | 'archived';
  coverImageUrl?: string;
  myRole: 'owner' | 'editor' | 'commenter' | 'viewer';
  membersCount: number;
}

interface TripMember {
  userId: string;
  email: string;
  displayName: string;
  role: 'owner' | 'editor' | 'commenter' | 'viewer';
  status: 'active' | 'pending' | 'removed';
  joinedAt: string;
}

interface TripInvitation {
  id: string;
  email: string;
  role: 'editor' | 'commenter' | 'viewer';
  status: 'pending' | 'accepted' | 'revoked' | 'expired';
  expiresAt: string;
}
```

---

## FE-04｜Trip

**模式**：旅程建立 Wizard、旅程 Overview 管理

### 細部功能
- Trip 建立 Wizard（名稱、目的地、日期、時區、人數、幣別、風格）
- Trip overview 頁（基本資訊、封面、成員列表）
- Date range 修改（含縮短旅程的衝突警告）
- Timezone 正確顯示與跨日邏輯
- Destination metadata 展示
- Trip 狀態管理：draft / active / archived

### 邊界個案
- Date range 縮短導致 itinerary item 落在範圍外 → 顯示衝突清單，需確認
- Timezone 跨夏令時間 → 正確計算本地時間
- Trip 名稱超過 200 字 → 前端即時驗證阻擋
- 修改 trip 版本衝突 → 提示重新整理

### 資料結構

```typescript
interface Trip {
  id: string;
  name: string;                 // maxLength: 200
  destinationText: string;
  startDate: string;            // YYYY-MM-DD
  endDate: string;
  timezone: string;             // IANA e.g. "Asia/Tokyo"
  currency: string;             // ISO-4217
  travelersCount: number;       // 1～50
  status: 'draft' | 'active' | 'archived';
  version: number;              // optimistic lock
  createdAt: string;
  updatedAt: string;
}

// Wizard 表單型別
interface TripCreateForm {
  name: string;
  destinationText: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
  pace?: 'relaxed' | 'balanced' | 'packed';
}
```

---

## FE-05｜Itinerary

**模式**：每日行程清單、拖拉排序、時間安排、草案對比

### 細部功能
- 按天顯示 itinerary days / items
- 拖拉變更順序（含跨日移動）
- 設定開始時間、結束時間、是否全天
- 顯示交通時間與預估移動成本
- 顯示時間衝突 / 重疊 / 營業時間衝突警告
- 顯示 AI 草案與正式版差異 highlight
- Optimistic UI 更新

### 邊界個案
- 兩個 item 時間重疊 → 視覺標示衝突，不阻擋儲存但顯示警告
- 拖拉排序時失去連線 → Mutation 排入 offline queue
- Server 回傳 409 version conflict → Revert optimistic state，提示重整
- item startAt / endAt 為空 → 以純排序卡片顯示

### 資料結構

```typescript
type ItineraryItemType =
  | 'place_visit'
  | 'meal'
  | 'transit'
  | 'hotel'
  | 'free_time'
  | 'custom';

interface ItineraryDay {
  dayId: string;
  date: string;                 // YYYY-MM-DD
  sortOrder: number;
  items: ItineraryItem[];
}

interface ItineraryItem {
  id: string;
  dayId: string;
  title: string;                // maxLength: 200
  itemType: ItineraryItemType;
  startAt?: string;             // ISO 8601 datetime
  endAt?: string;
  allDay: boolean;
  sortOrder: number;
  note?: string;                // maxLength: 5000
  placeId?: string;
  lat?: number;
  lng?: number;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
  routeSnapshotId?: string;
  version: number;
  // UI only
  hasConflict?: boolean;
  conflictWith?: string[];
  isPending?: boolean;          // optimistic, not yet confirmed
}

interface ReorderOperation {
  itemId: string;
  targetDayId: string;
  targetSortOrder: number;
}
```

---

## FE-06｜Budget

**模式**：預算設定、分類統計、實際支出紀錄、AI 預算規劃入口

### 細部功能
- 設定 total / per-person / per-day 預算
- 依 lodging / transit / food / attraction / shopping / misc 分類統計
- 顯示估算 vs 實際差異（條狀圖 / 百分比）
- 幣別設定與換算來源標示
- 新增實際支出
- 「以預算生成行程」入口（觸發 AI Planner）

### 邊界個案
- 總預算未填但填了每日預算 → 系統以天數換算提示總預算
- 幣別不支援 → 前端限制選項為白名單清單
- 實際支出超出預算 → 顯示超支比例警告，不阻擋新增
- 匯率快照過舊 → 顯示「匯率資料日期」提醒用戶核對

### 資料結構

```typescript
type BudgetCategory =
  | 'lodging'
  | 'transit'
  | 'food'
  | 'attraction'
  | 'shopping'
  | 'misc';

interface BudgetProfile {
  tripId: string;
  totalBudget?: number;
  perPersonBudget?: number;
  perDayBudget?: number;
  currency: string;
  categories: BudgetCategoryPlan[];
  version: number;
}

interface BudgetCategoryPlan {
  category: BudgetCategory;
  plannedAmount: number;
  actualAmount: number;         // computed from expenses
}

interface Expense {
  id: string;
  category: BudgetCategory;
  amount: number;
  currency: string;
  expenseAt?: string;
  note?: string;
  linkedItemId?: string;
}

// UI 計算統計
interface BudgetSummary {
  totalPlanned: number;
  totalActual: number;
  overrunPercent: number;
  byCategory: { category: BudgetCategory; planned: number; actual: number }[];
  exchangeRateSource: string;
  exchangeRateAt: string;
}
```

---

## FE-07｜Map

**模式**：地圖與清單雙向聯動、POI 搜尋、路線預覽

### 細部功能
- 地圖與 itinerary 清單雙向聯動
- POI 搜尋（輸入關鍵字、地點自動補全）
- 顯示單日 / 全旅程點位
- 顯示步行 / 開車 / 大眾運輸候選路線
- Provider 抽象：不綁死單一地圖 SDK
- 大量點位啟用 clustering

### 邊界個案
- 地圖 SDK 載入失敗 → Fallback 顯示地址清單，不崩潰
- Provider API 無回應 → 顯示「路線暫時無法取得」，保留舊快照
- 無 GPS / 拒絕定位 → 以目的地為預設中心
- 點位座標異常（0,0 或極端值）→ 過濾，不繪製

### 資料結構

```typescript
interface PlaceSearchResult {
  providerPlaceId: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  categories: string[];
}

type TransportMode = 'walk' | 'transit' | 'drive' | 'taxi';

interface RouteEstimate {
  mode: TransportMode;
  distanceMeters: number;
  durationSeconds: number;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
  provider: string;
  snapshotToken: string;        // 快取 token，可綁到 itinerary item
}

// Provider Adapter Interface（前端）
interface MapProviderAdapter {
  searchPlaces(query: string, lat?: number, lng?: number): Promise<PlaceSearchResult[]>;
  estimateRoute(origin: LatLng, destination: LatLng, mode: TransportMode): Promise<RouteEstimate>;
  renderMap(containerId: string, options: MapRenderOptions): MapInstance;
}
```

---

## FE-08｜AI Planner

**模式**：條件輸入、規劃 Job 追蹤、草案比較、手動採用

### 細部功能
- 條件輸入：預算、天數、風格、早起/晚起、交通偏好、景點密度、必去點、禁忌條件
- 顯示 planning job 進度
- 顯示多套 draft 清單，支援比較
- 顯示 validation warnings
- 僅允許用戶手動採用 draft（不自動寫入正式 itinerary）

### 邊界個案
- LLM 規劃中途離開頁面 → Job 繼續在後端執行，完成後發推播通知
- Draft 狀態為 invalid → 只能查看失敗原因，不顯示採用按鈕
- 採用 draft 時 trip 已有衝突變更 → 提示重新驗證
- 同一條件短時間重複提交 → 前端 debounce 防止爆量

### 資料結構

```typescript
interface AiPlanConstraints {
  totalBudget?: number;
  currency: string;
  travelersCount: number;
  pace: 'relaxed' | 'balanced' | 'packed';
  wakePattern: 'early' | 'normal' | 'late';
  transportPreference: 'walk' | 'transit' | 'taxi' | 'mixed';
  poiDensity: 'sparse' | 'medium' | 'dense';
  mustVisit: string[];          // place names or IDs
  avoid: string[];              // tags or place names
}

type AiJobStatus = 'queued' | 'running' | 'succeeded' | 'failed';
type AiDraftStatus = 'valid' | 'warning' | 'invalid';

interface AiPlanJob {
  jobId: string;
  status: AiJobStatus;
  acceptedAt: string;
  finishedAt?: string;
  failureCode?: string;
}

interface AiPlanDraft {
  id: string;
  tripId: string;
  title: string;
  status: AiDraftStatus;
  summary: string;
  warnings: string[];
  budgetEstimated?: number;
  daysCount: number;
  itemsCount: number;
  draft: AiDraftPayload;        // 詳見後端 BE-09
  createdAt: string;
}
```

---

## FE-09｜Offline / Sync

**模式**：本地快取、Mutation Queue、衝突提示、重新同步

### 細部功能
- 快取最近瀏覽的 trip（Service Worker + IndexedDB）
- 離線時新增/修改排入 local mutation queue
- 顯示待同步狀態 badge
- 上線後按序重送 mutation，顯示衝突提示
- Bootstrap API：依 sinceVersion 拉取差異

### 邊界個案
- 離線期間多人同改同一 item → 上線後顯示衝突解決 UI
- Mutation queue 中有過期操作 → 提示用戶部分操作失敗
- API key 絕不進入 persistent cache
- Service worker 更新失敗 → 不阻擋 app

### 資料結構

```typescript
interface MutationQueueItem {
  id: string;                   // client-generated UUID
  idempotencyKey: string;
  method: 'POST' | 'PATCH' | 'DELETE';
  endpoint: string;
  payload: unknown;
  version?: number;             // for optimistic lock
  enqueuedAt: number;           // unix ms
  retryCount: number;
  status: 'pending' | 'syncing' | 'failed';
  failureReason?: string;
}

interface SyncState {
  isOnline: boolean;
  pendingCount: number;
  lastSyncedAt?: number;
  conflictItems: ConflictItem[];
}

interface ConflictItem {
  resourceType: 'itinerary_item' | 'trip';
  resourceId: string;
  localVersion: number;
  serverVersion: number;
  description: string;
}

interface SyncBootstrapPayload {
  serverTime: string;
  trips: Trip[];
  changedDays: ItineraryDay[];
  changedNotifications: Notification[];
}
```

---

## FE-10｜Notification

**模式**：In-app 通知中心、FCM 推播接收、Deep-link 導航

### 細部功能
- In-app notification center（鈴鐺圖示 + 未讀數量）
- FCM push 接收與顯示
- 點擊通知 deep-link 到對應 trip / item / draft
- 已讀 / 未讀狀態管理

### 邊界個案
- 使用者拒絕推播權限 → 僅使用 in-app 通知，不強制
- 通知對應的 trip 已被刪除 → 點擊顯示「旅程已不存在」
- 大量通知堆積 → 支援批次已讀
- FCM token 過期 → 後端重新訂閱，前端靜默更新

### 資料結構

```typescript
type NotificationType =
  | 'trip.member.invited'
  | 'trip.itinerary_item.updated'
  | 'ai.plan.succeeded'
  | 'ai.plan.failed'
  | 'sync.conflict.detected';

interface AppNotification {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  link: string;                 // deep link path e.g. "/trips/xxx/ai/plans/yyy"
  readAt?: string;
  createdAt: string;
}
```

---

## FE-11｜Analytics

**模式**：統一事件追蹤、漏斗分析、曝光與轉換

### 細部功能
- 所有事件透過統一 analytics client 發送
- 事件命名格式：`<domain>.<entity>.<action>`
- 每個事件帶最小必要 context
- 避免單次操作重複觸發事件

### 邊界個案
- 事件送出失敗 → 靜默不影響使用者
- 敏感欄位（金額等）→ bucketize 或 hash
- 離線時觸發事件 → 緩存於 queue，上線後補送

### 資料結構

```typescript
interface AnalyticsEvent {
  event_name: string;           // e.g. "ai.plan.adopted"
  event_id: string;             // UUID
  occurred_at: string;
  actor_user_id?: string;
  session_id: string;
  trip_id?: string;
  request_id?: string;
  platform: 'web' | 'pwa' | 'mobile_web';
  app_version: string;
  locale: string;
  timezone: string;
  // event-specific payload
  [key: string]: unknown;
}
```

---
}
---

# Phase 2：真實整合 + UX 打磨

## FE-P2-01｜真實拖拉排序

**模式**：`@dnd-kit/core` + `@dnd-kit/sortable` 實作行程拖拉

### 細部功能
- `DndContext` 包裹 itinerary list
- `SortableItem` 包裹每個 itinerary card
- 跨 day container 拖拉移動
- 拖拉結束 → call `reorderItineraryItems` API
- Optimistic sort → 失敗時 revert

### 資料結構
```typescript
interface DragEndPayload {
  itemId: string;
  sourceDayId: string;
  targetDayId: string;
  targetSortOrder: number;
}
```

---

## FE-P2-02｜真實地圖 SDK

**模式**：Mapbox GL JS wrapper 實作 `MapProviderAdapter`

### 細部功能
- `MapboxAdapter implements MapProviderAdapter`
- `renderMap()` → 真實 mapboxgl.Map instance
- `addMarker()` → mapboxgl.Marker
- `fitBounds()` → map.fitBounds()
- Marker clustering → mapboxgl source + layer

---

## FE-P2-03｜Zod 表單驗證

**模式**：`@hookform/resolvers/zod` 整合所有表單

### 資料結構
```typescript
const createTripSchema = z.object({
  name: z.string().min(1).max(200),
  destinationText: z.string().min(1),
  startDate: z.string().date(),
  endDate: z.string().date(),
  timezone: z.string().min(1),
  currency: z.string().length(3),
  travelersCount: z.number().int().min(1).max(50)
}).refine(d => d.endDate >= d.startDate, {
  message: "End date must be on or after start date"
});
```

---

## FE-P2-04｜IndexedDB 離線持久化

**模式**：`idb` 套件封裝 mutation queue + trip cache

### 資料結構
```typescript
interface OfflineDB {
  mutations: {
    key: string;
    value: MutationQueueItem;
  };
  tripCache: {
    key: string;
    value: { trip: Trip; cachedAt: number };
  };
}
```

---

## FE-P2-06｜Budget 圖表

**模式**：純 CSS + SVG 實作，不引入圖表庫

### 細部功能
- 橫向 bar chart：category → planned vs actual
- Gauge chart：total spend / total budget %
- 數字 summary card：每人/每日平均

---

## FE-Q1｜Trip 建立 Wizard UX 強化（Quest #3–#7）

**模式**：Trip 建立表單增強，目的地搜尋整合、出發地欄位、下拉選單優化

> 來源：`quest.md` #3 #4 #5 #6 #7

### 細部功能
- 建立旅程表單新增「預算金額」欄位（數字輸入，帶幣別前綴）
- 建立旅程表單新增「出發地點」欄位（文字輸入 + 未來可串接地圖搜尋）
- 「目的地」欄位改為自動完成下拉（呼叫後端 `/api/v1/places/autocomplete`）
  - 最後一個選項顯示「沒有您想要的地方嗎？」→ 連結至 Google Maps / Gemini AI 推薦
  - 選取後自動帶入座標（經緯度）回填到 `destinationLat` / `destinationLng`
- 「時區」欄位改為下拉選單（依 IANA timezone list 產生選項）
  - 選取目的地後自動推算時區（from 座標 → timezone API）
- 「幣別」欄位改為下拉選單（ISO-4217 幣別清單，支援搜尋過濾）

### 邊界個案
- 目的地搜尋 API 失敗 → fallback 為純文字手動輸入，不阻擋建立
- 目的地搜尋結果為空 → 顯示「沒有找到結果，請手動輸入」
- 自動推算時區失敗 → 保留手動選擇功能
- 幣別清單過長 → 支援搜尋過濾 + 常用幣別置頂（TWD, USD, JPY, EUR）

### 資料結構

```typescript
// 加強版 Trip 建立表單
interface TripCreateFormEnhanced {
  name: string;
  departureText: string;           // 出發地（新增）
  departureLat?: number;
  departureLng?: number;
  destinationText: string;
  destinationLat?: number;         // 目的地座標（新增）
  destinationLng?: number;
  startDate: string;
  endDate: string;
  timezone: string;                // IANA，改為下拉
  currency: string;                // ISO-4217，改為下拉
  travelersCount: number;
  budgetAmount?: number;           // 預算金額（新增）
}

// 目的地搜尋結果
interface PlaceAutocompleteOption {
  placeId: string;
  label: string;                   // "京都, 日本"
  lat: number;
  lng: number;
  timezone?: string;
}

// 幣別選項
interface CurrencyOption {
  code: string;     // "TWD"
  label: string;    // "新台幣 (TWD)" | "New Taiwan Dollar (TWD)"
  symbol: string;   // "NT$"
}

// 時區選項
interface TimezoneOption {
  value: string;    // "Asia/Taipei"
  label: string;    // "(UTC+8) 台北" | "(UTC+8) Taipei"
  utcOffset: string;
}
```

---

## FE-Q2｜功能路由啟用與錯誤狀態（Quest #1 #2）

**模式**：修復導覽啟用邏輯與旅程載入錯誤處理

> 來源：`quest.md` #1 #2

### 細部功能
- 行程、預算、地圖、AI 規劃功能在建立 trip 後啟用導覽連結
- Trip detail 載入失敗時顯示友善錯誤訊息 + 重試按鈕

### 邊界個案
- 無 trip → itinerary / budget / map / ai 導覽 disabled（灰化 + tooltip 說明）
- Trip detail API 回 404 → 顯示「旅程不存在或已刪除」
- Trip detail API 回 403 → 顯示「無權限存取此旅程」
- Trip detail API timeout → 顯示「載入逾時，請重試」


