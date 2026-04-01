# website-to-markdown

`website-to-markdown` is a Go CLI that logs into websites with a real Chrome
session, fetches site content, and writes the result as Markdown.

The current implementation ships one adapter: `weibo`. It connects to a
user-started Chrome instance over the Chrome DevTools Protocol (CDP), performs
Weibo QR-code login when needed, reuses persisted cookies from
`~/.wtm/sessions/`, fetches posts through `m.weibo.cn`, and writes one Markdown
file per post plus an `index.md`.

## Current Scope

- Uses a real Chrome browser started by the user with remote debugging enabled
- Persists login cookies per site and account in `~/.wtm/sessions/`
- Fetches text posts and repost text from Weibo
- Writes Markdown output to `<output>/<username>/`
- Overwrites existing Markdown files on repeated runs for the same post IDs

Out of scope in the current implementation:

- Sites other than Weibo
- Media download
- Comments, replies, likes, or incremental sync
- A web UI or background service

## Requirements

- Go `1.25.4` or compatible Go `1.25.x`
- A locally installed Chrome or Chromium
- A Chrome instance started with remote debugging enabled

Example:

```bash
google-chrome \
  --remote-debugging-port=9222 \
  --user-data-dir=/tmp/wtm-chrome
```

If you use Chromium on macOS or Linux, replace `google-chrome` with the local
browser executable.

## Quick Start

1. Build the CLI:

```bash
go build -o bin/wtm ./cmd/wtm
```

2. Start Chrome with remote debugging enabled:

```bash
google-chrome \
  --remote-debugging-port=9222 \
  --user-data-dir=/tmp/wtm-chrome
```

3. Scrape a Weibo account:

```bash
./bin/wtm weibo --user <screen-name> --limit 20 --output ./output
```

4. On the first run, scan the QR code shown in Chrome. Later runs reuse the
saved session when it is still valid.

## CLI

```text
wtm weibo --user <name> [--limit 100] [--output ./output] \
  [--cdp-host localhost] [--cdp-port 9222]
```

Flags:

- `--user`: Weibo username or screen name, required
- `--limit`: maximum number of text posts to collect, default `100`
- `--output`: output directory, default `./output`
- `--cdp-host`: Chrome DevTools host, default `localhost`
- `--cdp-port`: Chrome DevTools port, default `9222`

## Output Layout

Example output tree:

```text
output/
  some-user/
    20240315-1234567890.md
    20240314-1234567888.md
    index.md
```

Per-post file format:

```md
**Date:** 2024-03-15 10:30:00 +0000

Post body text

> **@original_author:** Original repost text
```

`index.md` contains a Markdown table with timestamp, first-line title, and a
relative link to each generated file.

## Project Layout

```text
cmd/wtm/                    Cobra CLI entrypoint
internal/adapter/           Shared adapter interface and Post model
internal/adapters/weibo/    Weibo login flow and API client
internal/browser/           CDP connection and session persistence
internal/markdown/          Markdown rendering and index generation
.spec-driven/               Change proposals, design notes, and tasks
```

## Development

Run the standard checks:

```bash
go test ./...
go build ./cmd/wtm
go run ./cmd/wtm weibo --help
```

## Notes

- The tool does not launch Chrome for you; it attaches to an existing browser.
- Session files are stored at `~/.wtm/sessions/<site>-<username>.json`.
- The implementation uses CDP for login and plain HTTP requests for Weibo data
  fetching after login.
- Manual end-to-end verification still requires a real Weibo account and an
  interactive Chrome session.
