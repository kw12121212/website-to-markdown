package browser

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Browser wraps a go-rod browser connected to an existing Chrome instance.
type Browser struct {
	browser *rod.Browser
	page    *rod.Page
}

// Connect attaches to an already-running Chrome instance via its DevTools URL.
// Chrome must be started with --remote-debugging-port=<port>.
func Connect(host string, port int) (*Browser, error) {
	url := fmt.Sprintf("ws://%s:%d", host, port)
	b := rod.New().ControlURL(url)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to Chrome at %s: %w", url, err)
	}
	return &Browser{browser: b}, nil
}

// Page returns the current active page, opening a new blank tab if none exists.
func (b *Browser) Page() (*rod.Page, error) {
	if b.page != nil {
		return b.page, nil
	}
	page, err := b.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("opening new page: %w", err)
	}
	b.page = page
	return page, nil
}

// GetCookies returns all cookies from the connected browser that are in scope
// for any of the given URLs. Pass multiple URLs to cover cookies from related
// domains (e.g. both "https://weibo.com" and "https://m.weibo.cn").
func (b *Browser) GetCookies(urls ...string) ([]*proto.NetworkCookie, error) {
	page, err := b.Page()
	if err != nil {
		return nil, err
	}
	cookies, err := page.Cookies(urls)
	if err != nil {
		return nil, fmt.Errorf("getting cookies: %w", err)
	}
	return cookies, nil
}

// SetCookies injects a slice of cookies into the browser session.
func (b *Browser) SetCookies(cookies []*proto.NetworkCookie) error {
	page, err := b.Page()
	if err != nil {
		return err
	}
	params := make([]*proto.NetworkCookieParam, len(cookies))
	for i, c := range cookies {
		params[i] = &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
		}
	}
	return page.SetCookies(params)
}

// Disconnect closes the WebSocket connection to Chrome without sending a
// Browser.close command. Chrome itself keeps running.
func (b *Browser) Disconnect() {
	// go-rod does not expose a pure disconnect; abandoning the reference is
	// sufficient — the WebSocket connection is closed when the process exits.
	// We explicitly do NOT call b.browser.Close() since that would terminate
	// the user's Chrome process via a CDP Browser.close command.
}
