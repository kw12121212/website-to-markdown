package weibo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"time"

	"website-to-markdown/internal/adapter"
)

const (
	userAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	apiBase    = "https://m.weibo.cn/api/container/getIndex"
	detailBase = "https://m.weibo.cn/detail"
	pageSize   = 10
	maxRetries = 3
)

// client is a lightweight HTTP client for the m.weibo.cn JSON API.
type client struct {
	http    *http.Client
	cookies []*http.Cookie
}

func newClient(sub, twm, xsrf string) *client {
	jar := make([]*http.Cookie, 0, 3)
	if sub != "" {
		jar = append(jar, &http.Cookie{Name: "SUB", Value: sub, Domain: ".weibo.cn"})
	}
	if twm != "" {
		jar = append(jar, &http.Cookie{Name: "_T_WM", Value: twm, Domain: ".weibo.cn"})
	}
	if xsrf != "" {
		jar = append(jar, &http.Cookie{Name: "XSRF-TOKEN", Value: xsrf, Domain: ".weibo.cn"})
	}
	return &client{
		http:    &http.Client{Timeout: 30 * time.Second},
		cookies: jar,
	}
}

// userInfo holds the fields we need from the user info endpoint.
type userInfo struct {
	UID           string
	ScreenName    string
	StatusesCount int
}

// rawCard is a minimal representation of a Weibo API response card.
type rawCard struct {
	CardType  int     `json:"card_type"`
	Mblog     rawPost `json:"mblog"`
	CardGroup []struct {
		CardType int     `json:"card_type"`
		Mblog    rawPost `json:"mblog"`
	} `json:"card_group"`
}

type rawPost struct {
	ID              json.Number `json:"id"`
	Text            string      `json:"text"`
	CreatedAt       string      `json:"created_at"`
	IsLongText      bool        `json:"isLongText"`
	MblogType       int         `json:"mblogtype"`
	RetweetedStatus *rawPost    `json:"retweeted_status"`
	User            struct {
		ScreenName string `json:"screen_name"`
	} `json:"user"`
}

type apiResponse struct {
	OK   int `json:"ok"`
	Data struct {
		Cards        []rawCard `json:"cards"`
		CardlistInfo struct {
			Total int `json:"total"`
		} `json:"cardlistInfo"`
	} `json:"data"`
}

// GetUserInfo resolves a Weibo screen name to numeric UID and post count.
func (c *client) GetUserInfo(ctx context.Context, screenName string) (*userInfo, error) {
	searchURL := fmt.Sprintf("https://m.weibo.cn/n/%s", screenName)
	resp, err := c.doGet(ctx, searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// The redirect URL contains the numeric UID.
	finalURL := resp.Request.URL.String()
	// Extract UID from URL like https://m.weibo.cn/u/1234567890
	parts := strings.Split(strings.TrimRight(finalURL, "/"), "/")
	uid := parts[len(parts)-1]
	if uid == "" || uid == "u" {
		return nil, fmt.Errorf("could not resolve UID for user %q from URL %s", screenName, finalURL)
	}

	infoURL := fmt.Sprintf("%s?containerid=100505%s", apiBase, uid)
	var apiResp struct {
		Data struct {
			UserInfo struct {
				ID            json.Number `json:"id"`
				ScreenName    string      `json:"screen_name"`
				StatusesCount int         `json:"statuses_count"`
			} `json:"userInfo"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, infoURL, &apiResp); err != nil {
		return nil, fmt.Errorf("fetching user info for %s: %w", uid, err)
	}
	return &userInfo{
		UID:           uid,
		ScreenName:    apiResp.Data.UserInfo.ScreenName,
		StatusesCount: apiResp.Data.UserInfo.StatusesCount,
	}, nil
}

// FetchPosts retrieves up to limit text-containing posts for the given UID.
func (c *client) FetchPosts(ctx context.Context, uid string, limit int) ([]adapter.Post, error) {
	totalPages := (500 / pageSize) + 1 // safe upper bound; stop early on limit
	posts := make([]adapter.Post, 0, limit)
	collected := 0

	for page := 1; page <= totalPages && collected < limit; page++ {
		if page > 1 {
			if err := c.rateDelay(ctx, collected); err != nil {
				return posts, err
			}
		}
		url := fmt.Sprintf("%s?containerid=230413%s&page=%d", apiBase, uid, page)
		var apiResp apiResponse
		if err := c.getJSON(ctx, url, &apiResp); err != nil {
			return posts, fmt.Errorf("fetching page %d: %w", page, err)
		}
		if apiResp.OK != 1 || len(apiResp.Data.Cards) == 0 {
			break // no more pages
		}
		for _, card := range apiResp.Data.Cards {
			if collected >= limit {
				break
			}
			raw := extractRawPost(card)
			if raw == nil {
				continue
			}
			// Skip pinned posts (appear out of chronological order).
			if raw.MblogType == 2 {
				continue
			}
			post, err := c.convertPost(ctx, raw)
			if err != nil {
				return posts, err // propagate context cancellation
			}
			if post == nil {
				continue // no text content
			}
			posts = append(posts, *post)
			collected++
		}
	}
	return posts, nil
}

// extractRawPost unwraps card_type 11 wrappers and returns the inner post.
func extractRawPost(card rawCard) *rawPost {
	switch card.CardType {
	case 9:
		return &card.Mblog
	case 11:
		if len(card.CardGroup) > 0 && card.CardGroup[0].CardType == 9 {
			p := card.CardGroup[0].Mblog
			return &p
		}
	}
	return nil
}

// convertPost converts a rawPost to an adapter.Post, fetching long text if needed.
// Returns nil if the post has no text content at all (neither comment nor original).
func (c *client) convertPost(ctx context.Context, raw *rawPost) (*adapter.Post, error) {
	text := raw.Text
	if raw.IsLongText {
		full, err := c.fetchLongText(ctx, raw.ID.String())
		if err == nil {
			text = full
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	text = stripHTML(text)

	var rtText, rtAuthor string
	if raw.RetweetedStatus != nil {
		rt := raw.RetweetedStatus
		rtRaw := rt.Text
		if rt.IsLongText {
			full, err := c.fetchLongText(ctx, rt.ID.String())
			if err == nil {
				rtRaw = full
			} else if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		rtText = stripHTML(rtRaw)
		rtAuthor = rt.User.ScreenName
	}

	// Skip posts with no text content anywhere (neither comment nor original).
	if strings.TrimSpace(text) == "" && strings.TrimSpace(rtText) == "" {
		return nil, nil
	}

	ts, err := parseWeiboTime(raw.CreatedAt)
	if err != nil {
		ts = time.Now()
	}

	post := &adapter.Post{
		ID:        raw.ID.String(),
		Timestamp: ts,
		Text:      text,
	}
	if rtText != "" {
		post.IsRepost = true
		post.OriginalAuthor = rtAuthor
		post.OriginalText = rtText
	}
	return post, nil
}

// fetchLongText retrieves the full text of a long post from the detail page.
func (c *client) fetchLongText(ctx context.Context, weiboID string) (string, error) {
	url := fmt.Sprintf("%s/%s", detailBase, weiboID)
	resp, err := c.doGet(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Weibo detail pages embed JSON in a $render_data script block.
	re := regexp.MustCompile(`\$render_data\s*=\s*(\[.+);\s*\(function\(`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return extractLongTextFallback(body)
	}
	var renderData []json.RawMessage
	if err := json.Unmarshal(matches[1], &renderData); err != nil || len(renderData) == 0 {
		return extractLongTextFallback(body)
	}
	var wrapper struct {
		Status struct {
			Text string `json:"text"`
		} `json:"status"`
	}
	if err := json.Unmarshal(renderData[0], &wrapper); err != nil {
		return extractLongTextFallback(body)
	}
	if wrapper.Status.Text == "" {
		return extractLongTextFallback(body)
	}
	return wrapper.Status.Text, nil
}

// extractLongTextFallback attempts to find a "text" value adjacent to an "isLongText"
// marker in the raw HTML body. Used when the $render_data structure is not found.
func extractLongTextFallback(body []byte) (string, error) {
	re := regexp.MustCompile(`"isLongText"\s*:\s*true[^}]*?"text"\s*:\s*"((?:[^"\\]|\\.)*)"`)
	m := re.FindSubmatch(body)
	if len(m) < 2 {
		re2 := regexp.MustCompile(`"text"\s*:\s*"((?:[^"\\]|\\.)*)"[^}]*?"isLongText"\s*:\s*true`)
		m = re2.FindSubmatch(body)
	}
	if len(m) < 2 {
		return "", fmt.Errorf("long text not found in detail page")
	}
	var text string
	if err := json.Unmarshal(append([]byte{'"'}, append(m[1], '"')...), &text); err != nil {
		return string(m[1]), nil
	}
	return text, nil
}

// rateDelay sleeps for a randomized delay that grows with the number of posts
// collected. Returns ctx.Err() if the context is cancelled during the sleep.
func (c *client) rateDelay(ctx context.Context, collected int) error {
	base := 8.0
	if collected >= 300 {
		base = 18.0
	} else if collected >= 100 {
		base = 13.0
	}
	delay := base + rand.Float64()*4.0
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(delay * float64(time.Second))):
		return nil
	}
}

func (c *client) doGet(ctx context.Context, url string) (*http.Response, error) {
	var resp *http.Response
	var err error
	for attempt := range maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(5*(1<<attempt)) * time.Second):
			}
		}
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Referer", "https://m.weibo.cn/")
		for _, ck := range c.cookies {
			req.AddCookie(ck)
		}
		resp, err = c.http.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("HTTP %d after %d retries", resp.StatusCode, maxRetries)
}

func (c *client) getJSON(ctx context.Context, url string, dest any) error {
	resp, err := c.doGet(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// stripHTML removes HTML tags from Weibo post text and normalises whitespace.
var brRe = regexp.MustCompile(`(?i)<br\s*/?>`)
var tagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = brRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimSpace(l))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// parseWeiboTime parses Weibo's created_at format: "Mon Jan 02 15:04:05 +0800 2006"
func parseWeiboTime(s string) (time.Time, error) {
	return time.Parse("Mon Jan 02 15:04:05 +0800 2006", s)
}
