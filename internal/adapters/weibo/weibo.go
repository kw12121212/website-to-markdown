package weibo

import (
	"context"
	"fmt"

	"website-to-markdown/internal/adapter"
	"website-to-markdown/internal/browser"
)

// Adapter implements adapter.Adapter for Weibo (weibo.com).
type Adapter struct {
	b      *browser.Browser
	client *client
}

// New creates a new Weibo adapter connected to the given Chrome browser.
func New(b *browser.Browser) *Adapter {
	return &Adapter{b: b}
}

// IsLoggedIn reports whether the stored session is still valid.
func (a *Adapter) IsLoggedIn(_ context.Context, username string) bool {
	return isLoggedIn(a.b, username)
}

// Login performs the QR code login flow and initialises the API client.
func (a *Adapter) Login(ctx context.Context, username string) error {
	if err := login(ctx, a.b, username); err != nil {
		return err
	}
	return a.initClient(username)
}

// FetchPosts returns up to limit text-containing posts for the given username.
func (a *Adapter) FetchPosts(ctx context.Context, username string, limit int) ([]adapter.Post, error) {
	if a.client == nil {
		if err := a.initClient(username); err != nil {
			return nil, fmt.Errorf("initialising API client: %w", err)
		}
	}
	info, err := a.client.GetUserInfo(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("resolving user %q: %w", username, err)
	}
	fmt.Printf("Fetching posts for @%s (%s), total %d posts...\n", info.ScreenName, info.UID, info.StatusesCount)
	return a.client.FetchPosts(ctx, info.UID, limit)
}

// initClient loads the persisted session and configures the HTTP API client.
func (a *Adapter) initClient(username string) error {
	cookies, err := browser.LoadSession("weibo", username)
	if err != nil {
		return fmt.Errorf("loading session: %w", err)
	}
	var sub, twm, xsrf string
	for _, c := range cookies {
		switch c.Name {
		case "SUB":
			sub = c.Value
		case "_T_WM":
			twm = c.Value
		case "XSRF-TOKEN":
			xsrf = c.Value
		}
	}
	if sub == "" {
		return fmt.Errorf("SUB cookie not found in session; please log in again")
	}
	a.client = newClient(sub, twm, xsrf)
	return nil
}
