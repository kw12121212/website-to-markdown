package weibo

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// ---- stripHTML tests ----

func TestStripHTML_BrToNewline(t *testing.T) {
	input := "line one<br>line two<br/>line three"
	got := stripHTML(input)
	if !strings.Contains(got, "line one\nline two\nline three") {
		t.Errorf("expected <br> converted to newline, got: %q", got)
	}
}

func TestStripHTML_TagsRemoved(t *testing.T) {
	input := `<span class="surl-text">hello</span> <a href="#">world</a>`
	got := stripHTML(input)
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Errorf("expected HTML tags stripped, got: %q", got)
	}
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("expected text preserved, got: %q", got)
	}
}

func TestStripHTML_EmptyResult(t *testing.T) {
	input := "<img src='x'/><span></span>"
	got := stripHTML(input)
	if strings.TrimSpace(got) != "" {
		t.Errorf("expected empty result for tag-only input, got: %q", got)
	}
}

func TestStripHTML_Whitespace(t *testing.T) {
	input := "  hello  <br>  world  "
	got := stripHTML(input)
	for l := range strings.SplitSeq(got, "\n") {
		if l != strings.TrimSpace(l) {
			t.Errorf("expected lines trimmed, got line: %q", l)
		}
	}
}

// ---- parseWeiboTime tests ----

func TestParseWeiboTime(t *testing.T) {
	ts, err := parseWeiboTime("Mon Jan 02 15:04:05 +0800 2006")
	if err != nil {
		t.Fatalf("parseWeiboTime failed: %v", err)
	}
	if ts.Year() != 2006 {
		t.Errorf("expected year 2006, got %d", ts.Year())
	}
	if ts.Month().String() != "January" {
		t.Errorf("expected January, got %s", ts.Month())
	}
}

func TestParseWeiboTime_Invalid(t *testing.T) {
	_, err := parseWeiboTime("not a date")
	if err == nil {
		t.Error("expected error for invalid date string")
	}
}

// ---- extractRawPost tests ----

func makeCard(cardType int, postText string) rawCard {
	return rawCard{
		CardType: cardType,
		Mblog:    rawPost{ID: json.Number("1"), Text: postText},
	}
}

func TestExtractRawPost_Type9(t *testing.T) {
	card := makeCard(9, "hello")
	got := extractRawPost(card)
	if got == nil {
		t.Fatal("expected non-nil for card_type 9")
	}
	if got.Text != "hello" {
		t.Errorf("expected text 'hello', got %q", got.Text)
	}
}

func TestExtractRawPost_Type11_WithInnerType9(t *testing.T) {
	card := rawCard{
		CardType: 11,
		CardGroup: []struct {
			CardType int     `json:"card_type"`
			Mblog    rawPost `json:"mblog"`
		}{
			{CardType: 9, Mblog: rawPost{ID: json.Number("2"), Text: "inner"}},
		},
	}
	got := extractRawPost(card)
	if got == nil {
		t.Fatal("expected non-nil for card_type 11 with inner type 9")
	}
	if got.Text != "inner" {
		t.Errorf("expected text 'inner', got %q", got.Text)
	}
}

func TestExtractRawPost_Type11_EmptyGroup(t *testing.T) {
	card := rawCard{CardType: 11}
	got := extractRawPost(card)
	if got != nil {
		t.Errorf("expected nil for card_type 11 with empty group, got %+v", got)
	}
}

func TestExtractRawPost_UnknownType(t *testing.T) {
	card := makeCard(5, "ignored")
	got := extractRawPost(card)
	if got != nil {
		t.Errorf("expected nil for unknown card_type, got %+v", got)
	}
}

// ---- convertPost tests ----

func newTestClient() *client {
	return newClient("SUB_VALUE", "", "")
}

func TestConvertPost_OriginalPost(t *testing.T) {
	c := newTestClient()
	raw := &rawPost{
		ID:        json.Number("100"),
		Text:      "Hello <b>world</b>",
		CreatedAt: "Mon Jan 02 15:04:05 +0800 2006",
	}
	post, err := c.convertPost(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if post == nil {
		t.Fatal("expected non-nil post")
	}
	if post.Text != "Hello world" {
		t.Errorf("expected HTML stripped, got %q", post.Text)
	}
	if post.IsRepost {
		t.Error("expected IsRepost=false for original post")
	}
}

func TestConvertPost_RepostWithComment(t *testing.T) {
	c := newTestClient()
	raw := &rawPost{
		ID:        json.Number("200"),
		Text:      "My comment",
		CreatedAt: "Mon Jan 02 15:04:05 +0800 2006",
		RetweetedStatus: &rawPost{
			ID:   json.Number("99"),
			Text: "Original post text",
			User: struct{ ScreenName string `json:"screen_name"` }{ScreenName: "author"},
		},
	}
	post, err := c.convertPost(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if post == nil {
		t.Fatal("expected non-nil post")
	}
	if !post.IsRepost {
		t.Error("expected IsRepost=true")
	}
	if post.Text != "My comment" {
		t.Errorf("expected repost comment as Text, got %q", post.Text)
	}
	if post.OriginalAuthor != "author" {
		t.Errorf("expected OriginalAuthor 'author', got %q", post.OriginalAuthor)
	}
	if post.OriginalText != "Original post text" {
		t.Errorf("expected original text, got %q", post.OriginalText)
	}
}

func TestConvertPost_RepostWithNoComment(t *testing.T) {
	// Repost where the sharing user wrote no comment — should NOT be skipped
	// as long as the original post has text.
	c := newTestClient()
	raw := &rawPost{
		ID:        json.Number("300"),
		Text:      "", // no comment
		CreatedAt: "Mon Jan 02 15:04:05 +0800 2006",
		RetweetedStatus: &rawPost{
			ID:   json.Number("98"),
			Text: "Original content is here",
			User: struct{ ScreenName string `json:"screen_name"` }{ScreenName: "orig"},
		},
	}
	post, err := c.convertPost(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if post == nil {
		t.Fatal("expected non-nil post for repost-with-no-comment when original has text")
	}
	if post.OriginalText != "Original content is here" {
		t.Errorf("expected original text, got %q", post.OriginalText)
	}
}

func TestConvertPost_SkippedWhenNoTextAnywhere(t *testing.T) {
	c := newTestClient()
	raw := &rawPost{
		ID:        json.Number("400"),
		Text:      "<img src='x'/>", // becomes empty after strip
		CreatedAt: "Mon Jan 02 15:04:05 +0800 2006",
	}
	post, err := c.convertPost(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if post != nil {
		t.Errorf("expected nil post for image-only content, got %+v", post)
	}
}
