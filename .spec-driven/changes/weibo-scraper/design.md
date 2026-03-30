# Design: weibo-scraper

## Approach

The tool is structured as a thin CLI shell (`cmd/wtm/main.go`) over a core
package (`internal/browser`) and a set of site adapters (`internal/adapters/`).

**Two-phase architecture**: CDP is used *only* for the login flow. Once the
session is established, actual data fetching is done via Go's `net/http` client
against the `m.weibo.cn` JSON API. This avoids fragile DOM scraping, is faster,
and produces structured data that is easier to parse.

**Browser connection**: The user starts Chrome with `--remote-debugging-port`
before running `wtm`. The tool connects to `ws://<host>:<port>/json` to discover
open tabs, then attaches via `go-rod`. After login, the `SUB` cookie is extracted
from the browser and persisted — it is the sole required credential for all
subsequent API calls.

**Session lifecycle**:
1. On startup, attempt to load `~/.wtm/sessions/weibo-<username>.json`
2. If loaded, use the stored `SUB` cookie in the HTTP client and verify login by
   calling the user info endpoint; if valid, skip the browser entirely
3. If no session or session invalid, open Chrome via CDP, navigate to Weibo login
   page, wait for the QR code, prompt user to scan, poll for success
4. On successful login, extract cookies from the browser (especially `SUB`,
   `_T_WM`, `XSRF-TOKEN`), write to `~/.wtm/sessions/weibo-<username>.json`

**m.weibo.cn JSON API** (reference: github.com/dataabc/weibo-crawler):

| Purpose | URL |
|---|---|
| User timeline (paginated) | `https://m.weibo.cn/api/container/getIndex?containerid=230413{uid}&page={n}` |
| User info / post count | `https://m.weibo.cn/api/container/getIndex?containerid=100505{uid}` |
| Long post full text | `https://m.weibo.cn/detail/{weibo_id}` |

The `containerid` prefix `230413` is the internal Weibo constant for user
timeline cards. User info uses prefix `100505`.

**Pagination**:
- Total page count = `ceil(statuses_count / page_size)` where `statuses_count`
  comes from the user info endpoint and default `page_size` is 10.
- The JSON response contains a `cards` array. Only `card_type == 9` items are
  actual posts; `card_type == 11` items are wrapper containers whose
  `card_group[0]` element holds the actual post.
- Pinned posts (`mblogtype == 2`) appear at the top of page 1 out of
  chronological order and MUST be skipped.
- Stop when `--limit` text-containing posts have been collected, or all pages
  have been fetched.

**Long text handling**:
- If `isLongText == true` on a card, the `text` field is truncated.
- Fetch `https://m.weibo.cn/detail/{id}`, find the embedded JSON object
  (scan for `"status":` in the HTML response), extract the full `text` field.
- Same logic applies to the nested repost object if it also has `isLongText`.

**Text extraction**:
- The `text` field is raw HTML (contains `<br>`, `<a>`, `<span>` tags).
- Strip all HTML tags; convert `<br>` to `\n`; strip Weibo emoticon spans.
- Result is plain text, used as the Markdown body.

**Repost handling**:
- Reposts have a `retweeted_status` object containing the original post's
  `text`, `created_at`, and `user.screen_name`.
- Markdown format: reposting user's comment (if any) → blank line →
  `> **@original_author**: original text`

**Rate limiting**:
- Base delay: 8 seconds between page requests (matches weibo-crawler baseline).
- After 100 posts collected: increase base delay to 13s.
- After 300 posts collected: increase base delay to 18s.
- Each delay is randomized: `uniform(base, base+4)` seconds.
- On HTTP error or non-200 response: exponential backoff up to 3 retries before
  aborting.

**Package layout**:
```
cmd/wtm/              # CLI entry point (cobra)
internal/
  browser/            # CDP connector (go-rod) and session manager
  adapter/            # Adapter interface definition
  adapters/weibo/     # Weibo adapter: login + m.weibo.cn API client
  markdown/           # Markdown file writer and index builder
```

## Key Decisions

**CDP only for login, HTTP for data**: Using CDP to scrape paginated post lists
would require waiting on DOM renders for every scroll and page load. The
`m.weibo.cn` JSON API returns the same data in a single HTTP call per page.
Separating concerns (CDP = auth, HTTP = data) is faster and more testable.

**`go-rod/rod` for CDP**: Supports attaching to an existing browser via
`rod.New().ControlURL(...)`. Has clean Go API with good waiter/polling
primitives for the QR code login flow. `chromedp` was the alternative.

**`SUB` cookie as primary credential**: Both reference repos identify `SUB` as
the sole required auth token for the `m.weibo.cn` API. Secondary cookies
(`_T_WM`, `XSRF-TOKEN`) are extracted as backups but `SUB` alone is sufficient
for timeline reads.

**m.weibo.cn over weibo.cn**: `weibo.cn` (used by weiboSpider) requires
`Cookie` on every HTML request and returns paginated HTML requiring XPath
parsing. `m.weibo.cn` returns JSON, is simpler to parse, and provides more
structured metadata (numeric IDs, ISO timestamps, repost objects).

**Host + port as CLI flags**: Both `--cdp-host` and `--cdp-port` are CLI flags
with defaults `localhost` / `9222`, keeping the tool stateless and scriptable.

**Session path fixed to `~/.wtm/sessions/`**: No config option — simplifies
UX and avoids confusion about where credentials are stored.

**Overwrite on repeated runs**: Repeated runs overwrite existing post files.
Simplest behavior; avoids stale content. Future change may add incremental mode.

## Alternatives Considered

**DOM scraping with CDP for data fetching**: Rejected — fragile against layout
changes, requires waiting on dynamic renders, much slower per page than a direct
HTTP API call.

**`weibo.cn` HTML API (weiboSpider approach)**: Returns HTML that requires
XPath parsing; no structured JSON. Rejected in favor of `m.weibo.cn` JSON API.

**`chromedp` instead of `go-rod`**: More widely used, but more verbose API for
attaching to existing browsers and polling DOM conditions. Rejected in favor of
`go-rod`'s cleaner ergonomics for the narrow login-only CDP use case.

**Headless Chrome**: Easily detected by Weibo's anti-bot measures. Rejected per
the explicit requirement to use a real, user-visible browser.
