# Spec: Weibo Adapter

## ADDED Requirements

### Requirement: QR code login

When no valid session exists, the Weibo adapter MUST navigate to the Weibo login
page in the connected Chrome browser and wait for the QR code to be visible
before prompting the user to scan.

The adapter MUST poll until login is confirmed. Once confirmed, it MUST persist
the session cookies (at minimum the `SUB` token) and proceed without further
user interaction.

### Requirement: Session reuse

If a persisted session is available, the adapter MUST verify it against the
Weibo API before using it. If the session is still valid, the login flow MUST
be skipped entirely — the Chrome browser is not required.

### Requirement: User timeline fetch

The adapter MUST accept a Weibo screen name and resolve it to a numeric user ID.
It MUST then fetch the user's posts in reverse-chronological order using the
`m.weibo.cn` JSON API, stopping when the requested limit of text-containing
posts has been reached, or no further posts are available.

Pinned posts MUST be skipped and MUST NOT count toward the limit.

### Requirement: Long text expansion

If a post is marked as truncated (long text), the adapter MUST fetch the full
text before returning the post. The same rule applies to the original post
within a repost.

### Requirement: Post extraction

For each post the adapter MUST extract:
- A stable numeric post ID
- The publication timestamp
- The full plain-text body (HTML tags stripped; `<br>` converted to newline)

For reposts, the adapter MUST additionally extract:
- The original author's screen name
- The original post's full plain-text body

### Requirement: Content filtering

Posts that contain no text content after HTML stripping MUST be skipped and
MUST NOT count toward the `--limit`.

### Requirement: Rate limiting

The adapter MUST insert a randomized delay between page requests to avoid
triggering Weibo's rate limits. The delay MUST increase as more posts are
collected. On request failure, the adapter MUST retry with exponential backoff
before aborting.
