import { BarChart3, Copy, Loader2, Power } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDateTime } from "@/lib/format";
import type { RecentLink } from "@/types";

type RecentLinksCardProps = {
  links: RecentLink[];
  disablingCode: string;
  onCopy: (shortUrl: string) => Promise<void>;
  onDisable: (code: string) => Promise<void>;
  onLoadAnalytics: (code: string) => void;
};

export function RecentLinksCard({
  links,
  disablingCode,
  onCopy,
  onDisable,
  onLoadAnalytics,
}: RecentLinksCardProps) {
  return (
    <aside>
      <Card className="w-full">
        <CardHeader className="border-b border-border">
          <CardTitle className="text-base">Recent Links</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {links.length > 0 ? (
            <ul className="divide-y divide-border">
              {links.map((link) => (
                <li className="p-4" key={link.code}>
                  <div className="mb-2 flex items-center justify-between gap-3">
                    <div className="flex min-w-0 flex-wrap items-center gap-2">
                      <span className="rounded-sm bg-muted px-2 py-1 text-xs font-medium text-muted-foreground">
                        {link.code}
                      </span>
                      <span className="rounded-sm bg-card px-2 py-1 text-xs text-muted-foreground">
                        {link.isCustom ? "custom" : "random"}
                      </span>
                      {link.disabledAt ? (
                        <span className="rounded-sm bg-destructive/10 px-2 py-1 text-xs text-destructive">
                          disabled
                        </span>
                      ) : null}
                      {link.expiresAt ? (
                        <span className="rounded-sm bg-amber-100 px-2 py-1 text-xs text-amber-800">
                          expires {formatDateTime(link.expiresAt)}
                        </span>
                      ) : null}
                    </div>
                    <span className="shrink-0 text-xs text-muted-foreground">{formatDateTime(link.createdAt)}</span>
                  </div>
                  <a
                    className="block break-all text-sm font-medium text-primary underline-offset-4 hover:underline"
                    href={link.shortUrl}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {link.shortUrl}
                  </a>
                  <p className="mt-1 line-clamp-2 break-all text-xs text-muted-foreground">{link.originalUrl}</p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-8"
                      onClick={() => void onCopy(link.shortUrl)}
                    >
                      <Copy className="h-3.5 w-3.5" aria-hidden="true" />
                      Copy
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-8"
                      onClick={() => onLoadAnalytics(link.code)}
                    >
                      <BarChart3 className="h-3.5 w-3.5" aria-hidden="true" />
                      Analytics
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-8"
                      disabled={Boolean(link.disabledAt) || disablingCode === link.code}
                      onClick={() => void onDisable(link.code)}
                    >
                      {disablingCode === link.code ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                      ) : (
                        <Power className="h-3.5 w-3.5" aria-hidden="true" />
                      )}
                      Disable
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <div className="p-4 text-sm text-muted-foreground">No links yet</div>
          )}
        </CardContent>
      </Card>
    </aside>
  );
}
