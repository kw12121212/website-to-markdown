# Questions: weibo-scraper

## Open

<!-- No open questions -->

## Resolved

- [x] Q: Should CDP host and port be configurable?
  Context: Different users may run Chrome on different ports or remote machines.
  A: Yes — both `--cdp-host` and `--cdp-port` are CLI flags with defaults
     `localhost` and `9222`.

- [x] Q: Where should session files be stored — configurable or fixed?
  Context: Determines whether users can manage multiple credential locations.
  A: Fixed to `~/.wtm/sessions/`.

- [x] Q: Should repeated runs skip already-downloaded posts or overwrite them?
  Context: Affects whether stale content can accumulate vs. repeated network cost.
  A: Overwrite existing files on repeated runs.

- [x] Q: What should the per-post filename format be?
  Context: Needs to be sortable and unique.
  A: `<YYYYMMDD>-<post-id>.md` (e.g. `20240315-1234567890.md`).
