export type UserRole = "owner" | "editor" | "commenter" | "viewer";

export interface SessionUser {
  id: string;
  name: string;
  email: string;
  avatar: string;
}

export interface TripSummary {
  id: string;
  name: string;
  destination: string;
  dateRange: string;
  timezone: string;
  coverGradient: string;
  status: "draft" | "active" | "archived";
  role: UserRole;
  pendingInvites: number;
  members: number;
  currency: string;
  travelersCount: number;
  version: number;
  startDate: string;
  endDate: string;
}

export interface ItineraryItem {
  id: string;
  title: string;
  time: string;
  location: string;
  transit: string;
  cost: string;
  warning?: string;
  draftDiff?: string;
}

export interface ItineraryDay {
  id: string;
  label: string;
  date: string;
  summary: string;
  items: ItineraryItem[];
}

export interface BudgetCategory {
  name: string;
  estimated: number;
  actual: number;
}

export interface PlanDraft {
  id: string;
  name: string;
  summary: string;
  warnings: string[];
  score: number;
}

export interface NotificationItem {
  id: string;
  title: string;
  detail: string;
  time: string;
  unread: boolean;
  href: string;
}
