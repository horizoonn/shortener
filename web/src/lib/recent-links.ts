import type { RecentLink } from "@/types";

export const recentLinksLimit = 20;

const recentLinksKey = "shortener.recentLinks";

export function loadRecentLinks(): RecentLink[] {
  try {
    const raw = window.localStorage.getItem(recentLinksKey);
    if (!raw) {
      return [];
    }

    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.filter(isRecentLink).slice(0, recentLinksLimit);
  } catch {
    return [];
  }
}

export function saveRecentLinks(links: RecentLink[]) {
  window.localStorage.setItem(recentLinksKey, JSON.stringify(links.slice(0, recentLinksLimit)));
}

function isRecentLink(value: unknown): value is RecentLink {
  if (!value || typeof value !== "object") {
    return false;
  }

  const link = value as Record<string, unknown>;

  return (
    typeof link.shortUrl === "string" &&
    typeof link.code === "string" &&
    typeof link.createdAt === "string" &&
    typeof link.originalUrl === "string" &&
    typeof link.isCustom === "boolean" &&
    (typeof link.disabledAt === "string" || typeof link.disabledAt === "undefined") &&
    (typeof link.expiresAt === "string" || typeof link.expiresAt === "undefined")
  );
}
