export type ShortenResult = {
  shortUrl: string;
  code: string;
  originalUrl: string;
  isCustom: boolean;
  createdAt: string;
  expiresAt?: string;
};

type ErrorResponse = {
  message?: string;
  code?: string;
};

type ShortenResponse = {
  shortUrl?: string;
  short_url?: string;
  code?: string;
  original_url?: string;
  is_custom?: boolean;
  created_at?: string;
  expires_at?: string;
};

export type CreateLinkInput = {
  url: string;
  customAlias?: string;
  expiresAt?: string;
};

export type AnalyticsQuery = {
  code: string;
  from?: string;
  to?: string;
  recentLimit?: number;
};

export type AnalyticsResult = {
  code: string;
  originalUrl: string;
  totalClicks: number;
  clicksByDay: TimeBucketCount[];
  clicksByMonth: TimeBucketCount[];
  clicksByUserAgent: UserAgentCount[];
  recentClicks: RecentClick[];
};

export type TimeBucketCount = {
  label: string;
  clicks: number;
};

export type UserAgentCount = {
  userAgent: string;
  clicks: number;
};

export type RecentClick = {
  clickedAt: string;
  userAgent: string;
  referer: string | null;
  ip: string | null;
};

type AnalyticsResponse = {
  code?: string;
  original_url?: string;
  total_clicks?: number;
  clicks_by_day?: Array<{ day?: string; clicks?: number }>;
  clicks_by_month?: Array<{ month?: string; clicks?: number }>;
  clicks_by_user_agent?: Array<{ user_agent?: string; clicks?: number }>;
  recent_clicks?: Array<{
    clicked_at?: string;
    user_agent?: string;
    referer?: string | null;
    ip?: string | null;
  }>;
};

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export async function shortenUrl(input: CreateLinkInput): Promise<ShortenResult> {
  const response = await postJSON("/api/v1/shorten", {
    original_url: input.url,
    custom_alias: input.customAlias || undefined,
    expires_at: input.expiresAt || undefined,
  });
  return parseShortenResponse(response, input.url);
}

export async function getAnalytics(query: AnalyticsQuery): Promise<AnalyticsResult> {
  const params = new URLSearchParams();

  if (query.from) {
    params.set("from", query.from);
  }
  if (query.to) {
    params.set("to", query.to);
  }
  if (query.recentLimit) {
    params.set("recent_limit", String(query.recentLimit));
  }

  const path = `/api/v1/analytics/${encodeURIComponent(query.code)}${
    params.size > 0 ? `?${params.toString()}` : ""
  }`;
  const response = await request(path);
  const data = await readJSON<AnalyticsResponse & ErrorResponse>(response);

  if (!response.ok) {
    throw apiError(response, data, "Failed to load analytics");
  }

  if (!data.code || !data.original_url || typeof data.total_clicks !== "number") {
    throw new Error("Analytics response is missing required fields");
  }

  return {
    code: data.code,
    originalUrl: data.original_url,
    totalClicks: data.total_clicks,
    clicksByDay: (data.clicks_by_day ?? []).map((item) => ({
      label: item.day ?? "",
      clicks: item.clicks ?? 0,
    })),
    clicksByMonth: (data.clicks_by_month ?? []).map((item) => ({
      label: item.month ?? "",
      clicks: item.clicks ?? 0,
    })),
    clicksByUserAgent: (data.clicks_by_user_agent ?? []).map((item) => ({
      userAgent: item.user_agent ?? "unknown",
      clicks: item.clicks ?? 0,
    })),
    recentClicks: (data.recent_clicks ?? []).map((item) => ({
      clickedAt: item.clicked_at ?? "",
      userAgent: item.user_agent ?? "unknown",
      referer: item.referer ?? null,
      ip: item.ip ?? null,
    })),
  };
}

export async function disableLink(code: string): Promise<void> {
  const response = await request(`/api/v1/links/${encodeURIComponent(code)}`, {
    method: "DELETE",
  });
  const data = await readJSON<ErrorResponse>(response);

  if (!response.ok) {
    throw apiError(response, data, "Failed to disable link");
  }
}

export function qrCodeUrl(code: string, size = 256) {
  const params = new URLSearchParams({ size: String(size) });
  return `/api/v1/links/${encodeURIComponent(code)}/qr?${params.toString()}`;
}

async function postJSON(path: string, body: unknown) {
  try {
    return await request(path, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });
  } catch {
    throw new ApiError("Backend is unavailable. Check that the API container is running.", 0, "network_error");
  }
}

async function request(path: string, init?: RequestInit) {
  try {
    return await fetch(path, init);
  } catch {
    throw new ApiError("Backend is unavailable. Check that the API container is running.", 0, "network_error");
  }
}

async function parseShortenResponse(response: Response, fallbackOriginalUrl: string): Promise<ShortenResult> {
  const data = await readJSON<ShortenResponse & ErrorResponse>(response);

  if (!response.ok) {
    throw apiError(response, data, "Failed to shorten URL");
  }

  const shortUrl = data.shortUrl || data.short_url;
  if (!shortUrl || !data.code) {
    throw new Error("Shortener response is missing required fields");
  }

  return {
    shortUrl,
    code: data.code,
    originalUrl: data.original_url || fallbackOriginalUrl,
    isCustom: data.is_custom ?? false,
    createdAt: data.created_at || new Date().toISOString(),
    expiresAt: data.expires_at,
  };
}

async function readJSON<T>(response: Response): Promise<T> {
  try {
    return (await response.json()) as T;
  } catch {
    return {} as T;
  }
}

function apiError(response: Response, data: ErrorResponse, fallbackMessage: string) {
  const code = data.code || "unknown_error";
  const message = messageForError(response.status, code, data.message || fallbackMessage);
  return new ApiError(message, response.status, code);
}

function messageForError(status: number, code: string, fallbackMessage: string) {
  if (status === 409 || code === "conflict") {
    return "Custom alias is already taken.";
  }
  if (status === 400 || code === "invalid_argument") {
    return "Check the URL and custom alias format.";
  }
  if (status === 404 || code === "not_found") {
    return "Link was not found or has been disabled.";
  }
  if (status >= 500) {
    return "Server error. Try again later.";
  }

  return fallbackMessage;
}
