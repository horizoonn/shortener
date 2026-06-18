import { Check, Copy, Download, ExternalLink, Link2, Loader2, Power, QrCode } from "lucide-react";
import type { Dispatch, FormEvent, SetStateAction } from "react";
import { useState } from "react";

import { ErrorMessage } from "@/components/feedback";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { qrCodeUrl, type ShortenResult } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { CreateFieldError, CreateState } from "@/types";

type ShortenCardProps = {
  createState: CreateState;
  createdLink: ShortenResult | null;
  error: string;
  fieldError: CreateFieldError;
  isCreating: boolean;
  canCreate: boolean;
  onStateChange: Dispatch<SetStateAction<CreateState>>;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onCopy: (shortUrl: string) => Promise<void>;
  onDisable: (code: string) => Promise<void>;
};

export function ShortenCard({
  createState,
  createdLink,
  error,
  fieldError,
  isCreating,
  canCreate,
  onStateChange,
  onSubmit,
  onCopy,
  onDisable,
}: ShortenCardProps) {
  return (
    <Card>
      <CardHeader className="border-b border-border">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <Link2 className="h-5 w-5" aria-hidden="true" />
          </div>
          <div>
            <CardTitle>URL Shortener</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">Create and manage short links</p>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 pt-6">
        <form className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px_200px_auto]" onSubmit={onSubmit}>
          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="long-url">
              Long URL
            </label>
            <Input
              id="long-url"
              aria-invalid={fieldError === "url"}
              className={cn(fieldError === "url" && "border-destructive focus-visible:ring-destructive")}
              inputMode="url"
              placeholder="https://example.com/articles/release-notes"
              type="url"
              value={createState.url}
              onChange={(event) => onStateChange((current) => ({ ...current, url: event.target.value }))}
              disabled={isCreating}
              required
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="custom-alias">
              Custom Alias
            </label>
            <Input
              id="custom-alias"
              aria-invalid={fieldError === "customAlias"}
              className={cn(fieldError === "customAlias" && "border-destructive focus-visible:ring-destructive")}
              inputMode="text"
              pattern="[A-Za-z0-9_-]{3,64}"
              placeholder="release-notes"
              value={createState.customAlias}
              onChange={(event) => onStateChange((current) => ({ ...current, customAlias: event.target.value }))}
              disabled={isCreating}
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium" htmlFor="expires-at">
              Expires At
            </label>
            <Input
              id="expires-at"
              aria-invalid={fieldError === "expiresAt"}
              className={cn(fieldError === "expiresAt" && "border-destructive focus-visible:ring-destructive")}
              type="datetime-local"
              value={createState.expiresAt}
              onChange={(event) => onStateChange((current) => ({ ...current, expiresAt: event.target.value }))}
              disabled={isCreating}
            />
          </div>

          <div className="flex items-end">
            <Button className="w-full lg:w-auto" type="submit" disabled={!canCreate}>
              {isCreating ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <Link2 className="h-4 w-4" aria-hidden="true" />
              )}
              Shorten
            </Button>
          </div>
        </form>

        {error ? <ErrorMessage message={error} /> : null}

        {createdLink ? <LinkResult link={createdLink} onCopy={onCopy} onDisable={onDisable} /> : null}
      </CardContent>
    </Card>
  );
}

function LinkResult({
  link,
  onCopy,
  onDisable,
}: {
  link: ShortenResult;
  onCopy: (shortUrl: string) => Promise<void>;
  onDisable: (code: string) => Promise<void>;
}) {
  const qrUrl = qrCodeUrl(link.code);
  const [qrFailed, setQRFailed] = useState(false);

  return (
    <div className="rounded-md border border-border bg-muted/40 p-4">
      <div className="mb-3 flex flex-wrap items-center gap-2 text-sm font-medium text-muted-foreground">
        <Check className="h-4 w-4 text-emerald-600" aria-hidden="true" />
        <span>Created</span>
        <span className="rounded-sm bg-card px-2 py-1 text-xs">{link.isCustom ? "custom" : "random"}</span>
        {link.expiresAt ? (
          <span className="rounded-sm bg-amber-100 px-2 py-1 text-xs text-amber-800">
            expires {new Date(link.expiresAt).toLocaleString()}
          </span>
        ) : null}
      </div>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <a
            className="break-all text-base font-semibold text-primary underline-offset-4 hover:underline"
            href={link.shortUrl}
            target="_blank"
            rel="noreferrer"
          >
            {link.shortUrl}
          </a>
          <p className="mt-1 break-all text-sm text-muted-foreground">{link.originalUrl}</p>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="Copy short URL"
            onClick={() => void onCopy(link.shortUrl)}
          >
            <Copy className="h-4 w-4" aria-hidden="true" />
          </Button>
          <Button asChild variant="outline" size="icon">
            <a href={link.shortUrl} target="_blank" rel="noreferrer" aria-label="Open short URL">
              <ExternalLink className="h-4 w-4" aria-hidden="true" />
            </a>
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="Disable short URL"
            onClick={() => void onDisable(link.code)}
          >
            <Power className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
      </div>
      <div className="mt-4 flex flex-col gap-3 rounded-md border border-border bg-card p-3 sm:flex-row sm:items-center">
        {qrFailed ? (
          <div className="flex h-32 w-32 items-center justify-center rounded-sm border border-border bg-muted p-3 text-center text-xs text-muted-foreground">
            QR unavailable
          </div>
        ) : (
          <img
            className="h-32 w-32 rounded-sm border border-border bg-white p-2"
            src={qrUrl}
            width={128}
            height={128}
            alt={`QR code for ${link.shortUrl}`}
            onError={() => setQRFailed(true)}
          />
        )}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 text-sm font-medium">
            <QrCode className="h-4 w-4" aria-hidden="true" />
            QR code
          </div>
          <p className="mt-1 break-all text-xs text-muted-foreground">{link.shortUrl}</p>
          {qrFailed ? null : (
            <Button className="mt-3 h-8" variant="outline" size="sm" asChild>
              <a href={qrUrl} download={`${link.code}-qr.png`}>
                <Download className="h-3.5 w-3.5" aria-hidden="true" />
                Download QR
              </a>
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
