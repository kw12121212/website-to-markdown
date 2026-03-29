package weibo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"website-to-markdown/internal/browser"
)

const (
	loginURL   = "https://passport.weibo.com/sso/signin"
	qrSelector = ".qrcode img, img.qr, [class*='qrcode']"
)

// login performs the Weibo QR code login flow using a real Chrome browser.
// It navigates to the login page, waits for the QR code, prompts the user to
// scan, polls until login is confirmed, then saves the session.
func login(ctx context.Context, b *browser.Browser, username string) error {
	page, err := b.Page()
	if err != nil {
		return err
	}

	fmt.Println("Navigating to Weibo login page...")
	if err := page.Navigate(loginURL); err != nil {
		return fmt.Errorf("navigating to login page: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("waiting for login page load: %w", err)
	}

	// Wait up to 15 seconds for the QR code element to appear.
	fmt.Println("Waiting for QR code...")
	el, err := page.Timeout(15 * time.Second).Element(qrSelector)
	if err != nil {
		return fmt.Errorf("QR code did not appear: %w", err)
	}
	if err := el.WaitVisible(); err != nil {
		return fmt.Errorf("QR code not visible: %w", err)
	}

	fmt.Println(">>> Please scan the QR code in Chrome to log in. Waiting up to 3 minutes...")

	// Poll every 2 seconds until logged in (URL leaves passport domain).
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
		info, err := page.Info()
		if err != nil {
			continue
		}
		url := info.URL
		if strings.Contains(url, "weibo.com") && !strings.Contains(url, "passport") {
			fmt.Println("Login detected. Saving session...")
			return saveSessionCookies(b, username)
		}
	}
	return fmt.Errorf("login timed out after 3 minutes")
}

// isLoggedIn checks whether the stored session is still valid.
func isLoggedIn(b *browser.Browser, username string) bool {
	cookies, err := browser.LoadSession("weibo", username)
	if err != nil || len(cookies) == 0 {
		return false
	}
	if err := b.SetCookies(cookies); err != nil {
		return false
	}
	page, err := b.Page()
	if err != nil {
		return false
	}
	if err := page.Navigate("https://m.weibo.cn/api/config"); err != nil {
		return false
	}
	// WaitLoad error is intentionally ignored: the API endpoint may not fire a
	// standard load event, but the body content is still readable.
	_ = page.WaitLoad()
	el, err := page.Element("body")
	if err != nil {
		return false
	}
	content, err := el.Text()
	if err != nil {
		return false
	}
	return strings.Contains(content, `"login":true`)
}

func saveSessionCookies(b *browser.Browser, username string) error {
	// Collect cookies from both weibo.com and weibo.cn domains so we capture
	// the SUB token regardless of which domain Weibo sets it on.
	cookies, err := b.GetCookies("https://weibo.com", "https://m.weibo.cn")
	if err != nil {
		return fmt.Errorf("extracting cookies: %w", err)
	}
	// Deduplicate by name, preferring the first occurrence.
	seen := make(map[string]bool, len(cookies))
	deduped := cookies[:0]
	for _, c := range cookies {
		if !seen[c.Name] {
			seen[c.Name] = true
			deduped = append(deduped, c)
		}
	}
	return browser.SaveSession("weibo", username, deduped)
}
