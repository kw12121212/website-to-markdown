# Tasks: weibo-scraper

## Implementation

- [x] Initialize Go module (`go mod init website-to-markdown`) targeting Go 1.25;
      add dependencies: `go-rod/rod`, `spf13/cobra`
- [x] Define `Adapter` interface in `internal/adapter/adapter.go`:
      `Login(ctx) error`, `IsLoggedIn(ctx) bool`, `FetchPosts(ctx, username, limit int) ([]Post, error)`;
      define shared `Post` struct (ID, Timestamp, Text, IsRepost, OriginalAuthor, OriginalText)
- [x] Implement CDP connector in `internal/browser/browser.go`: connect to
      existing Chrome via `rod.New().ControlURL("ws://<host>:<port>")`, expose
      `GetCookies() []Cookie` and `SetCookies([]Cookie)` helpers
- [x] Implement session manager in `internal/browser/session.go`: load and save
      cookie JSON files at `~/.wtm/sessions/<site>-<username>.json`
- [x] Implement Weibo login in `internal/adapters/weibo/login.go`:
      - Navigate to `https://passport.weibo.com/sso/signin`
      - Wait for QR code element to be visible, print prompt to terminal
      - Poll for login success (check for logged-in DOM indicator)
      - Extract `SUB`, `_T_WM`, `XSRF-TOKEN` cookies via CDP and save session
- [x] Implement Weibo API client in `internal/adapters/weibo/api.go`:
      - `GetUserID(screenName)` via `containerid=100505{uid}` to resolve handle → numeric UID and `statuses_count`
      - `FetchPage(uid, page)` via `containerid=230413{uid}&page={n}`, parse `cards` array:
        filter `card_type == 9`; unwrap `card_type == 11` → `card_group[0]`;
        skip pinned posts (`mblogtype == 2`)
      - `FetchLongText(weiboID)` via `m.weibo.cn/detail/{id}`, extract embedded
        `"status"` JSON object, return full `text` field
      - Strip HTML tags from `text`; convert `<br>` to `\n`
      - Detect `isLongText == true` and call `FetchLongText` for both post and
        nested `retweeted_status`
      - Apply rate limiting: base 8s delay between pages, scaled to 13s after
        100 posts and 18s after 300 posts, randomized `uniform(base, base+4)`;
        exponential backoff up to 3 retries on error
- [x] Wire Weibo adapter in `internal/adapters/weibo/weibo.go` implementing the
      `Adapter` interface: check session → skip or trigger login → fetch posts
- [x] Implement Markdown writer in `internal/markdown/writer.go`:
      - Write `<output-dir>/<username>/<YYYYMMDD>-<post-id>.md` per post
      - Format: timestamp metadata line, blank line, post text; for reposts append
        blank line + `> **@original_author**: original text`
      - Write/overwrite `<output-dir>/<username>/index.md` listing all posts with
        first-line title and timestamp
- [x] Implement CLI in `cmd/wtm/main.go` with `weibo` subcommand and flags:
      `--user` (required), `--limit` (default 100), `--output` (default `./output`),
      `--cdp-host` (default `localhost`), `--cdp-port` (default `9222`)

## Testing

- [x] Unit tests for `internal/markdown/writer.go`: correct filename format,
      correct Markdown for original post, correct Markdown for repost (blockquote),
      correct `index.md` content, overwrite behavior
- [x] Unit tests for `internal/browser/session.go`: save/load round-trip using
      a temp directory; missing file returns empty session without error
- [x] Unit tests for HTML-to-text stripping in the Weibo API client: `<br>`→`\n`,
      tags stripped, repost blockquote formatted correctly
- [x] `go vet ./...` passes with no errors
- [x] `golangci-lint run` (or `go build` + `go test`) passes

## Verification

- [x] `go build ./cmd/wtm` succeeds
- [x] `wtm weibo --help` prints usage with all flags and their defaults
- [ ] Manual end-to-end: start Chrome with `--remote-debugging-port=9222`,
      run `wtm weibo --user <test-account> --limit 10`,
      confirm QR code prompt appears, scan, confirm posts written to
      `./output/<username>/`
- [ ] Session file exists at `~/.wtm/sessions/weibo-<username>.json` after login
- [ ] Second run overwrites existing files without error
- [ ] Third run (valid session) skips QR code entirely
- [ ] `index.md` lists all collected posts with correct titles and timestamps
- [ ] Implementation matches all scope items in `proposal.md`
