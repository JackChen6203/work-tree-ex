import type {
  BudgetCategory,
  ItineraryDay,
  NotificationItem,
  PlanDraft,
  SessionUser,
  TripSummary
} from "../types/domain";

export const sessionUser: SessionUser = {
  id: "u_01",
  name: "Ariel Chen",
  email: "ariel@example.com",
  avatar: "AC"
};

export const trips: TripSummary[] = [
  {
    id: "kyoto-2026",
    name: "Kyoto Slow Spring",
    destination: "Kyoto, Japan",
    dateRange: "2026/04/14 - 2026/04/19",
    timezone: "Asia/Tokyo",
    coverGradient: "from-[#24403a] via-[#376052] to-[#b4cdc2]",
    status: "active",
    role: "owner",
    pendingInvites: 2,
    members: 4,
    currency: "JPY",
    travelersCount: 4,
    version: 1,
    startDate: "2026-04-14",
    endDate: "2026-04-19"
  },
  {
    id: "seoul-food",
    name: "Seoul Food Sprint",
    destination: "Seoul, Korea",
    dateRange: "2026/06/02 - 2026/06/06",
    timezone: "Asia/Seoul",
    coverGradient: "from-[#36243a] via-[#6e4d63] to-[#f0d6ce]",
    status: "draft",
    role: "editor",
    pendingInvites: 1,
    members: 3,
    currency: "KRW",
    travelersCount: 3,
    version: 2,
    startDate: "2026-06-02",
    endDate: "2026-06-06"
  }
];

export const itineraryDays: ItineraryDay[] = [
  {
    id: "day-1",
    label: "Day 1",
    date: "04/14 Tue",
    summary: "東山區散步與晚間鴨川用餐",
    items: [
      {
        id: "i-1",
        title: "清水寺晨間參拜",
        time: "08:00 - 10:00",
        location: "Kiyomizu-dera",
        transit: "步行 14 分鐘",
        cost: "JPY 500",
        warning: "預估人潮高峰 09:30"
      },
      {
        id: "i-2",
        title: "二年坂咖啡與散策",
        time: "10:20 - 12:00",
        location: "Ninenzaka",
        transit: "步行 9 分鐘",
        cost: "JPY 1,400",
        draftDiff: "AI draft 建議縮短 20 分鐘"
      },
      {
        id: "i-3",
        title: "先斗町晚餐",
        time: "18:30 - 20:00",
        location: "Pontocho Alley",
        transit: "巴士 22 分鐘",
        cost: "JPY 4,800"
      }
    ]
  },
  {
    id: "day-2",
    label: "Day 2",
    date: "04/15 Wed",
    summary: "嵐山竹林與河岸野餐",
    items: [
      {
        id: "i-4",
        title: "嵐山竹林晨拍",
        time: "07:30 - 09:30",
        location: "Arashiyama Bamboo Grove",
        transit: "JR 34 分鐘",
        cost: "JPY 0"
      },
      {
        id: "i-5",
        title: "渡月橋河岸野餐",
        time: "11:00 - 12:30",
        location: "Togetsukyo Bridge",
        transit: "步行 12 分鐘",
        cost: "JPY 2,100",
        warning: "雨備方案待確認"
      }
    ]
  }
];

export const budgetCategories: BudgetCategory[] = [
  { name: "住宿", estimated: 28500, actual: 28500 },
  { name: "交通", estimated: 12000, actual: 10400 },
  { name: "餐飲", estimated: 15000, actual: 8600 },
  { name: "門票", estimated: 7000, actual: 500 },
  { name: "購物", estimated: 8000, actual: 0 }
];

export const aiDrafts: PlanDraft[] = [
  {
    id: "draft-1",
    name: "Budget Balanced",
    summary: "控制每日交通與餐飲支出，保留寺院與市場密度。",
    warnings: ["Day 3 夜間移動轉乘較多"],
    score: 88
  },
  {
    id: "draft-2",
    name: "Slow Morning",
    summary: "減少早起點位，拉長咖啡與逛街停留時間。",
    warnings: ["總預算高於目標 6%"],
    score: 81
  }
];

export const notifications: NotificationItem[] = [
  {
    id: "n-1",
    title: "AI draft 已完成",
    detail: "Kyoto Slow Spring 的 2 套候選方案可比較。",
    time: "3 分鐘前",
    unread: true,
    href: "/trips/kyoto-2026/ai-planner"
  },
  {
    id: "n-2",
    title: "成員接受邀請",
    detail: "Mina 已加入行程並取得 editor 權限。",
    time: "1 小時前",
    unread: false,
    href: "/trips/kyoto-2026"
  }
];
