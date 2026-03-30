# Proposal: weibo-scraper

## What

Build a Go CLI tool (`wtm`) that connects to a user-started real Chrome browser
via Chrome DevTools Protocol, logs into Weibo via QR code scan, and scrapes the
latest N text-containing posts from a specified user, saving each post as a
Markdown file.

This change also establishes the core architecture: CDP connector, session
persistence layer, and a multi-site adapter interface that Weibo is the first
implementation of.

## Why

Weibo uses aggressive anti-bot measures. Using a real (non-headless) Chrome
browser connected via CDP is the most reliable way to operate without detection.
A pluggable adapter interface ensures the core infrastructure can be reused for
future site targets without rearchitecting.

## Scope

**Included:**

- Go module initialization (`website-to-markdown`, target Go 1.25)
- CDP connector: connects to an already-running Chrome instance via WebSocket;
  host and port configurable via CLI flags with defaults (`localhost:9222`)
- Site adapter interface (`Adapter`) with lifecycle methods: `Login`,
  `IsLoggedIn`, `FetchPosts`
- Session persistence: save and load cookies per site+account to
  `~/.wtm/sessions/<site>-<username>.json`; if a valid session exists, skip login
- Weibo adapter:
  - Navigate to Weibo login page, wait for QR code to appear, prompt user to scan
  - Poll for login success, save session on success
  - Navigate to target user's profile page
  - Scroll and paginate to collect up to N posts
  - Include original posts and reposts; skip posts with no text content
  - Extract: post body text, repost attribution (original author + text if
    repost), publication timestamp
- Markdown output: one directory per target user (`<output-dir>/<username>/`),
  one `.md` file per post named `<YYYYMMDD>-<post-id>.md`, plus an `index.md`
  listing all collected posts with title (first line) and timestamp
- CLI: `wtm weibo --user <username> --limit <n> --output <dir>
  --cdp-host <host> --cdp-port <port>`
  - Default `--cdp-host`: `localhost`
  - Default `--cdp-port`: `9222`
  - Default `--limit`: `100`
  - Default `--output`: `./output`
- Repeated runs overwrite existing files for the same post IDs

**Excluded:**

- Other site adapters (future changes)
- Image, video, or audio content (text only)
- Comments, replies, or likes
- Incremental / delta-only scraping
- Any GUI or web interface

## Unchanged Behavior

N/A — this is a greenfield project with no existing behavior.
