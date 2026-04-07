# 📦 前端模組

---

## FE-12｜Menu 尚未實作功能補齊

**模式**：完成頂部 Menu 各入口的可用行為與對應頁面互動

### 頁面欄位/按鈕矩陣（前端必做）

#### A. `總覽`（`/`）

主要顯示欄位  
- 旅程卡片：`name`、`destination`、`dateRange`、`timezone`、`members`、`currency`
- 近期活動：`title`、`body`、`createdAt`
- 即將出發：以 `startDate` 計算倒數天數
- 快速存取：從最近通知關聯 trip 或 trips 前三筆

欄位來源  
- `useTripsQuery()` → `GET /api/v1/trips`
- `useNotificationsQuery()` → `GET /api/v1/notifications`
- DB 最終來源：`trips`、`notifications`

按鈕功能  
- `建立旅程`：開啟 Wizard（本地 UI state）
- Wizard `下一步/上一步/送出`：驗證後 `useCreateTripMutation()` → `POST /api/v1/trips`
- `前往 Google Maps / Gemini`：外連，不打 API
- Trip 卡片點擊：導向 `/trips/:tripId`

#### B. `旅程`（`/trips/:tripId`）

主要顯示欄位  
- Trip metadata：`name/destination/dateRange/timezone/version/status`
- 成員清單：`displayName/email/role/status`
- 邀請清單：`inviteeEmail/role/status/expiresAt/createdAt`
- 分享連結：`accessScope/expiresAt/revokedAt`

欄位來源  
- `useTripQuery`、`useTripMembersQuery`、`useTripInvitationsQuery`、`useTripShareLinksQuery`
- DB 最終來源：`trips`、`trip_memberships`、`users`、`trip_invitations`、`share_links`

按鈕功能  
- `更新旅程`：`PATCH /api/v1/trips/:tripId`
- `新增成員`：`POST /api/v1/trips/:tripId/members`
- `移除成員`：`DELETE /api/v1/trips/:tripId/members/:memberId`
- `更新角色`：`PATCH /api/v1/trips/:tripId/members/:memberId`
- `送出邀請`：`POST /api/v1/trips/:tripId/invitations`
- `撤銷邀請`：`POST /api/v1/trips/:tripId/invitations/:invitationId/revoke`
- `重新邀請`：同送出邀請（同 email）
- `建立分享連結`：`POST /api/v1/trips/:tripId/share-links`
- `撤銷分享連結`：`POST /api/v1/trips/:tripId/share-links/:linkId/revoke`

#### C. `行程`（`/trips/:tripId/itinerary`）

主要顯示欄位  
- Days：`dayId/date/dayIndex`
- Items：`title/itemType/startAt/endAt/allDay/note/sortOrder/version`
- 路線摘要：`distanceKm/durationMin/estimatedCost`

欄位來源  
- `useItineraryDaysQuery`（含 items）
- DB 最終來源：`itinerary_days`、`itinerary_items`、`place_snapshots`、`route_snapshots`

按鈕功能  
- `新增行程項目`：`POST /api/v1/trips/:tripId/items`
- `編輯/儲存`：`PATCH /api/v1/trips/:tripId/items/:itemId`
- `刪除`：`DELETE /api/v1/trips/:tripId/items/:itemId`
- `拖拉排序`：`POST /api/v1/trips/:tripId/items/reorder`
- `上移/下移 fallback`：同 reorder API

#### D. `預算`（`/trips/:tripId/budget`）

主要顯示欄位  
- `totalBudget/perPersonBudget/perDayBudget/currency`
- 分類預算：`categories[].plannedAmount`
- 支出列表：`category/amount/currency/expenseAt/note/linkedItemId`
- 匯率資訊：`source/fetchedAt/rates`

欄位來源  
- `useBudgetProfileQuery`、`useExpensesQuery`、`useBudgetRatesQuery`、`useItineraryDaysQuery`
- DB 最終來源：`budget_profiles`、`expenses`、`itinerary_items`

按鈕功能  
- `儲存預算`：`PUT /api/v1/trips/:tripId/budget`
- `新增支出`：`POST /api/v1/trips/:tripId/expenses`
- `儲存支出編輯`：`PATCH /api/v1/trips/:tripId/expenses/:expenseId`
- `刪除支出`：`DELETE /api/v1/trips/:tripId/expenses/:expenseId`
- `更新匯率`：`POST /api/v1/trips/:tripId/budget/rates/refresh`
- `以預算生成行程`：導向 `/trips/:tripId/ai-planner`

#### E. `地圖`（`/trips/:tripId/map`）

主要顯示欄位  
- 搜尋結果：`name/address/categories/lat/lng/providerPlaceId`
- 行程點位：由 itinerary items 聚合
- 路線估算：`distanceKm/durationMin/provider/estimatedCost`

欄位來源  
- `useMapPlacesQuery`、`useItineraryDaysQuery`
- DB 最終來源：搜尋/路線多為 provider；加入行程後落地於 `itinerary_items`

按鈕功能  
- `搜尋`：`GET /api/v1/maps/search`
- `估算路線`：`POST /api/v1/maps/routes`
- `加入行程`：`POST /api/v1/trips/:tripId/items`
- `聚焦點位`：本地 map state 更新

#### F. `AI 規劃`（`/trips/:tripId/ai-planner`）

主要顯示欄位  
- Constraints form：`totalBudget/currency/pace/transportPreference/wakePattern/poiDensity/mustVisit/avoid`
- Draft list：`id/title/status/summary/warnings`
- Job progress：`queued/running/succeeded/failed`

欄位來源  
- `useAiPlansQuery`、`useAiPlanQuery`、`useBudgetProfileQuery`、`useMyPreferencesQuery`
- DB 最終來源：`ai_plan_requests`、`ai_plan_drafts`、`ai_plan_validation_results`、`budget_profiles`、`user_preferences`

按鈕功能  
- `產生規劃`：`POST /api/v1/trips/:tripId/ai/plans`
- `採用草案`：`POST /api/v1/trips/:tripId/ai/plans/:planId/adopt`
- `新增/移除 mustVisit/avoid tag`：本地 form state

#### G. `收件匣`（`/notifications`）

主要顯示欄位  
- `title/body/type/link/readAt/createdAt`
- unread 狀態 badge

欄位來源  
- `useNotificationsQuery(unreadOnly)`、`useTripsQuery`（檢查 deep-link trip 是否存在）
- DB 最終來源：`notifications`、`trips`

按鈕功能  
- `全部已讀`：`POST /api/v1/notifications/read-all`
- `僅看未讀`：本地 state 過濾（重新 query）
- `標記已讀/未讀`：`POST /api/v1/notifications/:id/read|unread`
- `刪除`：`DELETE /api/v1/notifications/:id`
- `清除已讀`：`POST /api/v1/notifications/cleanup-read`
- `點擊通知`：先標已讀再導頁；失效 trip 顯示 toast

#### H. `設定`（`/settings`）

主要顯示欄位  
- Profile：`displayName/locale/timezone/currency`
- Preferences：`tripPace/wakePattern/transportPreference/foodPreference/avoidTags`
- Notification prefs：`pushEnabled/emailEnabled/digestFrequency/quietHoursStart/quietHoursEnd/tripUpdates/budgetAlerts/aiPlanReadyAlerts`
- LLM provider list：`provider/label/model/maskedKey/createdAt`

欄位來源  
- `useMyProfileQuery`、`useMyPreferencesQuery`、`useMyNotificationPreferencesQuery`、`useMyLlmProvidersQuery`
- DB 最終來源：`users`、`user_preferences`、`llm_provider_configs`

按鈕功能  
- `儲存個人資料`：`PATCH /api/v1/users/me`
- `儲存偏好`：`PUT /api/v1/users/me/preferences`
- `儲存通知設定`：`PUT /api/v1/users/me/notification-preferences`
- `新增 provider`：`POST /api/v1/users/me/llm-providers`
- `刪除 provider`：`DELETE /api/v1/users/me/llm-providers/:providerId`
- `測試連線`：目前本地模擬（後續可接真實測試 API）
- `刪除帳號`：`DELETE /api/v1/users/me`

### 邊界個案（Menu 共通）
- 初次登入無 trip：`旅程/行程/預算/地圖/AI` 路由按鈕導向 `/?openCreateTrip=1`
- route 直入不存在 trip：顯示錯誤卡，不使整頁崩潰
- 行動版 bottom-sheet 導覽與桌機 header 導覽行為一致
- 語系切換後 label 與按鈕功能不可脫鉤

### 資料結構

```typescript
interface MenuRouteState {
  activeTripId?: string;
  fallbackCreateTripPath: string; // "/?openCreateTrip=1"
}
```

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
