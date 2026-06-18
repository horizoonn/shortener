import { BarChart3, CalendarDays, Loader2, Search } from "lucide-react";
import type { Dispatch, FormEvent, SetStateAction } from "react";

import { EmptyPanel, ErrorMessage } from "@/components/feedback";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { formatDateTime } from "@/lib/format";
import type { AnalyticsResult } from "@/lib/api";
import type { AnalyticsState } from "@/types";

type AnalyticsCardProps = {
  state: AnalyticsState;
  analytics: AnalyticsResult | null;
  error: string;
  isLoading: boolean;
  canSubmit: boolean;
  onStateChange: Dispatch<SetStateAction<AnalyticsState>>;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
};

export function AnalyticsCard({
  state,
  analytics,
  error,
  isLoading,
  canSubmit,
  onStateChange,
  onSubmit,
}: AnalyticsCardProps) {
  return (
    <Card>
      <CardHeader className="border-b border-border">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-accent text-accent-foreground">
            <BarChart3 className="h-5 w-5" aria-hidden="true" />
          </div>
          <div>
            <CardTitle>Analytics</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">Clicks, buckets, agents, and recent events</p>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 pt-6">
        <form className="grid gap-4 lg:grid-cols-[180px_1fr_1fr_120px_auto]" onSubmit={onSubmit}>
          <AnalyticsInput
            id="analytics-code"
            label="Code"
            value={state.code}
            disabled={isLoading}
            required
            onChange={(value) => onStateChange((current) => ({ ...current, code: value }))}
          />
          <AnalyticsInput
            id="analytics-from"
            label="From"
            type="date"
            value={state.from}
            disabled={isLoading}
            onChange={(value) => onStateChange((current) => ({ ...current, from: value }))}
          />
          <AnalyticsInput
            id="analytics-to"
            label="To"
            type="date"
            value={state.to}
            disabled={isLoading}
            onChange={(value) => onStateChange((current) => ({ ...current, to: value }))}
          />
          <AnalyticsInput
            id="recent-limit"
            label="Recent"
            type="number"
            min={1}
            max={100}
            value={state.recentLimit}
            disabled={isLoading}
            onChange={(value) => onStateChange((current) => ({ ...current, recentLimit: value }))}
          />
          <div className="flex items-end">
            <Button className="w-full lg:w-auto" type="submit" disabled={!canSubmit}>
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <Search className="h-4 w-4" aria-hidden="true" />
              )}
              Load
            </Button>
          </div>
        </form>

        {error ? <ErrorMessage message={error} /> : null}

        {analytics ? <AnalyticsResultPanel analytics={analytics} /> : null}
      </CardContent>
    </Card>
  );
}

function AnalyticsInput({
  id,
  label,
  type = "text",
  value,
  disabled,
  required = false,
  min,
  max,
  onChange,
}: {
  id: string;
  label: string;
  type?: string;
  value: string;
  disabled: boolean;
  required?: boolean;
  min?: number;
  max?: number;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-2">
      <label className="text-sm font-medium" htmlFor={id}>
        {label}
      </label>
      <Input
        id={id}
        type={type}
        min={min}
        max={max}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        required={required}
      />
    </div>
  );
}

function AnalyticsResultPanel({ analytics }: { analytics: AnalyticsResult }) {
  return (
    <div className="space-y-4 rounded-md border border-border bg-card p-4">
      <div className="grid gap-3 sm:grid-cols-3">
        <Metric label="Total Clicks" value={analytics.totalClicks.toLocaleString()} />
        <Metric label="Code" value={analytics.code} />
        <Metric label="Original" value={analytics.originalUrl} wrap />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <CountList title="Daily" items={analytics.clicksByDay} />
        <CountList title="Monthly" items={analytics.clicksByMonth} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <UserAgentList items={analytics.clicksByUserAgent} />
        <RecentClicksList analytics={analytics} />
      </div>
    </div>
  );
}

function Metric({ label, value, wrap = false }: { label: string; value: string; wrap?: boolean }) {
  return (
    <div className="rounded-md border border-border bg-muted/40 p-3">
      <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
      <div className={`mt-1 text-lg font-semibold ${wrap ? "break-all text-sm" : ""}`}>{value}</div>
    </div>
  );
}

function CountList({ title, items }: { title: string; items: Array<{ label: string; clicks: number }> }) {
  return (
    <div className="rounded-md border border-border">
      <div className="flex items-center gap-2 border-b border-border px-3 py-2 text-sm font-semibold">
        <CalendarDays className="h-4 w-4" aria-hidden="true" />
        {title}
      </div>
      {items.length > 0 ? (
        <ul className="divide-y divide-border">
          {items.map((item) => (
            <li className="flex items-center justify-between gap-3 px-3 py-2 text-sm" key={item.label}>
              <span className="text-muted-foreground">{item.label}</span>
              <span className="font-semibold">{item.clicks}</span>
            </li>
          ))}
        </ul>
      ) : (
        <EmptyPanel />
      )}
    </div>
  );
}

function UserAgentList({ items }: { items: AnalyticsResult["clicksByUserAgent"] }) {
  return (
    <div className="rounded-md border border-border">
      <div className="border-b border-border px-3 py-2 text-sm font-semibold">User Agents</div>
      {items.length > 0 ? (
        <ul className="divide-y divide-border">
          {items.map((item) => (
            <li className="grid grid-cols-[minmax(0,1fr)_auto] gap-3 px-3 py-2 text-sm" key={item.userAgent}>
              <span className="truncate text-muted-foreground" title={item.userAgent}>
                {item.userAgent}
              </span>
              <span className="font-semibold">{item.clicks}</span>
            </li>
          ))}
        </ul>
      ) : (
        <EmptyPanel />
      )}
    </div>
  );
}

function RecentClicksList({ analytics }: { analytics: AnalyticsResult }) {
  return (
    <div className="rounded-md border border-border">
      <div className="border-b border-border px-3 py-2 text-sm font-semibold">Recent Clicks</div>
      {analytics.recentClicks.length > 0 ? (
        <ul className="divide-y divide-border">
          {analytics.recentClicks.map((click, index) => (
            <li className="space-y-1 px-3 py-2 text-sm" key={`${click.clickedAt}-${index}`}>
              <div className="font-medium">{formatDateTime(click.clickedAt)}</div>
              <div className="break-all text-xs text-muted-foreground">{click.userAgent}</div>
              <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                <span>{click.ip || "no ip"}</span>
                <span>{click.referer || "no referer"}</span>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <EmptyPanel />
      )}
    </div>
  );
}
