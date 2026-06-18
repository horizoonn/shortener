import { FormEvent, useMemo, useState } from "react";
import { toast, Toaster } from "sonner";

import { AnalyticsCard } from "@/components/analytics-card";
import { RecentLinksCard } from "@/components/recent-links-card";
import { ShortenCard } from "@/components/shorten-card";
import { ApiError, AnalyticsResult, disableLink, getAnalytics, shortenUrl, ShortenResult } from "@/lib/api";
import { normalizeLimit } from "@/lib/format";
import { loadRecentLinks, recentLinksLimit, saveRecentLinks } from "@/lib/recent-links";
import type { AnalyticsState, CreateFieldError, CreateState, RecentLink } from "@/types";

export default function App() {
  const [createState, setCreateState] = useState<CreateState>({ url: "", customAlias: "", expiresAt: "" });
  const [createdLink, setCreatedLink] = useState<ShortenResult | null>(null);
  const [recentLinks, setRecentLinks] = useState<RecentLink[]>(() => loadRecentLinks());
  const [createError, setCreateError] = useState("");
  const [createFieldError, setCreateFieldError] = useState<CreateFieldError>(null);
  const [isCreating, setIsCreating] = useState(false);

  const [analyticsState, setAnalyticsState] = useState<AnalyticsState>({
    code: "",
    from: "",
    to: "",
    recentLimit: "20",
  });
  const [analytics, setAnalytics] = useState<AnalyticsResult | null>(null);
  const [analyticsError, setAnalyticsError] = useState("");
  const [isLoadingAnalytics, setIsLoadingAnalytics] = useState(false);
  const [disablingCode, setDisablingCode] = useState("");

  const canCreate = useMemo(
    () => createState.url.trim().length > 0 && !isCreating,
    [createState.url, isCreating],
  );
  const canLoadAnalytics = useMemo(
    () => analyticsState.code.trim().length > 0 && !isLoadingAnalytics,
    [analyticsState.code, isLoadingAnalytics],
  );

  function persistRecentLinks(links: RecentLink[]) {
    setRecentLinks(links);
    saveRecentLinks(links);
  }

  async function handleCreateSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const url = createState.url.trim();
    const customAlias = createState.customAlias.trim();
    if (!url) {
      return;
    }

    setIsCreating(true);
    setCreateError("");
    setCreateFieldError(null);

    try {
      const nextLink = await shortenUrl({
        url,
        customAlias: customAlias || undefined,
        expiresAt: createState.expiresAt || undefined,
      });
      const nextRecentLinks = [
        nextLink,
        ...recentLinks.filter((link) => link.code !== nextLink.code),
      ].slice(0, recentLinksLimit);

      setCreatedLink(nextLink);
      setAnalyticsState((current) => ({ ...current, code: nextLink.code }));
      persistRecentLinks(nextRecentLinks);
      toast.success("Link created");
    } catch (error) {
      const feedback = createFeedback(error, customAlias);
      setCreateError(feedback.message);
      setCreateFieldError(feedback.field);
      setCreatedLink(null);
    } finally {
      setIsCreating(false);
    }
  }

  async function handleAnalyticsSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await loadAnalytics(analyticsState.code);
  }

  async function loadAnalytics(code: string) {
    const trimmedCode = code.trim();
    if (!trimmedCode) {
      return;
    }

    setIsLoadingAnalytics(true);
    setAnalyticsError("");

    try {
      const result = await getAnalytics({
        code: trimmedCode,
        from: analyticsState.from || undefined,
        to: analyticsState.to || undefined,
        recentLimit: normalizeLimit(analyticsState.recentLimit),
      });
      setAnalytics(result);
      setAnalyticsState((current) => ({ ...current, code: result.code }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to load analytics";
      setAnalyticsError(message);
      setAnalytics(null);
    } finally {
      setIsLoadingAnalytics(false);
    }
  }

  async function handleDisable(code: string) {
    setDisablingCode(code);

    try {
      await disableLink(code);
      const disabledAt = new Date().toISOString();
      const nextRecentLinks = recentLinks.map((link) =>
        link.code === code ? { ...link, disabledAt } : link,
      );

      persistRecentLinks(nextRecentLinks);
      toast.success("Link disabled");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to disable link";
      toast.error(message);
    } finally {
      setDisablingCode("");
    }
  }

  async function copyShortUrl(shortUrl: string) {
    await navigator.clipboard.writeText(shortUrl);
    toast.success("Copied");
  }

  return (
    <main className="min-h-screen bg-background text-foreground">
      <Toaster richColors position="top-right" />

      <div className="mx-auto grid min-h-screen w-full max-w-7xl gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[minmax(0,1fr)_380px] lg:py-8">
        <section className="space-y-6">
          <ShortenCard
            createState={createState}
            createdLink={createdLink}
            error={createError}
            fieldError={createFieldError}
            isCreating={isCreating}
            canCreate={canCreate}
            onStateChange={setCreateState}
            onSubmit={handleCreateSubmit}
            onCopy={copyShortUrl}
            onDisable={handleDisable}
          />

          <AnalyticsCard
            state={analyticsState}
            analytics={analytics}
            error={analyticsError}
            isLoading={isLoadingAnalytics}
            canSubmit={canLoadAnalytics}
            onStateChange={setAnalyticsState}
            onSubmit={handleAnalyticsSubmit}
          />
        </section>

        <RecentLinksCard
          links={recentLinks}
          disablingCode={disablingCode}
          onCopy={copyShortUrl}
          onDisable={handleDisable}
          onLoadAnalytics={(code) => {
            setAnalyticsState((current) => ({ ...current, code }));
            void loadAnalytics(code);
          }}
        />
      </div>
    </main>
  );
}

function createFeedback(error: unknown, customAlias: string): { message: string; field: CreateFieldError } {
  if (error instanceof ApiError) {
    if (error.status === 409 || error.code === "conflict") {
      return { message: error.message, field: "customAlias" };
    }
    if (error.status === 400 || error.code === "invalid_argument") {
      return { message: error.message, field: customAlias ? "customAlias" : "url" };
    }
    return { message: error.message, field: null };
  }

  if (error instanceof Error) {
    return { message: error.message, field: null };
  }

  return { message: "Failed to shorten URL", field: null };
}
