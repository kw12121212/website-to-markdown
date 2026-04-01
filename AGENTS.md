# AGENTS

This repository is a small, explicit Go CLI. Keep it that way.

## Purpose

- Build a CLI that converts site content into Markdown
- Current supported site: Weibo
- Current runtime model: use a real Chrome session for login, then use HTTP APIs
  for data retrieval when possible

## Mandatory Workflow

- Route every non-trivial change through `.spec-driven/` first
- Treat repository documents as navigation surfaces for humans and agents
- Prefer explicit code paths over hidden conventions or framework magic
- Avoid adding new dependencies unless the current approach is clearly
  insufficient

## Repository Map

- `cmd/wtm/`: CLI wiring with Cobra
- `internal/adapter/`: shared interface and `Post` type
- `internal/adapters/weibo/`: Weibo-specific login, session bootstrap, and API logic
- `internal/browser/`: CDP connection and cookie persistence under `~/.wtm/sessions/`
- `internal/markdown/`: Markdown file rendering and index generation
- `.spec-driven/changes/`: proposals, design notes, open questions, task lists

## Working Rules

- Preserve the current architecture split:
  CDP/browser automation for login, HTTP client for content fetching
- Keep adapter boundaries clean so future sites can reuse browser/session and
  markdown modules
- Do not hardcode machine-specific paths beyond the documented session location
- Do not assume headless browsing is acceptable for Weibo
- Repeated runs should remain safe and overwrite generated Markdown deterministically

## Before Editing

- Read the relevant spec-driven artifacts in `.spec-driven/changes/`
- Read the package you are touching end to end before changing it
- Check whether the change affects user-facing CLI behavior, output format, or
  session semantics; if it does, update docs and specs together

## Verification

Run the smallest relevant set first, then the broader baseline when possible:

```bash
go test ./...
go build ./cmd/wtm
go run ./cmd/wtm weibo --help
```

For login or scraping changes, manual verification is part of the work:

- Start Chrome with `--remote-debugging-port`
- Run `wtm weibo --user <name>`
- Confirm first-run QR login
- Confirm a session file is written under `~/.wtm/sessions/`
- Confirm a later run can skip login when the session is still valid

## Documentation Expectations

- Keep `README.md` aligned with the real command-line behavior
- Document new adapters, flags, output files, or environment prerequisites
- Prefer short, concrete instructions over broad product language
