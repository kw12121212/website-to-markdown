# Spec: Browser & Session

## ADDED Requirements

### Requirement: CDP connection

The tool MUST connect to an already-running Chrome browser instance via Chrome
DevTools Protocol WebSocket. It MUST NOT launch a new browser process.

The CDP host and port MUST be configurable via CLI flags. Default host is
`localhost`; default port is `9222`.

### Requirement: Session persistence

After a successful site login, the tool MUST save the browser cookies for that
site and account to `~/.wtm/sessions/<site>-<username>.json`.

On subsequent runs, if a valid session file exists, the tool MUST inject those
cookies into the browser and verify login status before attempting any login
flow. If the session is still valid, the login flow MUST be skipped.

If the session file is missing or the injected session is no longer valid, the
tool MUST fall through to the normal login flow.
