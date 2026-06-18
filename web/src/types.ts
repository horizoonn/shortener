import type { ShortenResult } from "@/lib/api";

export type RecentLink = ShortenResult & {
  disabledAt?: string;
};

export type CreateState = {
  url: string;
  customAlias: string;
  expiresAt: string;
};

export type CreateFieldError = "url" | "customAlias" | "expiresAt" | null;

export type AnalyticsState = {
  code: string;
  from: string;
  to: string;
  recentLimit: string;
};
