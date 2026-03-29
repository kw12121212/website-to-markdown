package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"website-to-markdown/internal/adapter"
)

var fixedTime = time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

func makePost(id, text string) adapter.Post {
	return adapter.Post{
		ID:        id,
		Timestamp: fixedTime,
		Text:      text,
	}
}

func TestWritePost_Filename(t *testing.T) {
	dir := t.TempDir()
	post := makePost("1234567890", "Hello world")
	if err := WritePost(dir, "testuser", post); err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(dir, "testuser", "20240315-1234567890.md")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("expected file %s not created", expected)
	}
}

func TestWritePost_OriginalPost(t *testing.T) {
	dir := t.TempDir()
	post := makePost("111", "This is the post text.")
	if err := WritePost(dir, "u", post); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "u", "20240315-111.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "2024-03-15") {
		t.Error("expected date in output")
	}
	if !strings.Contains(s, "This is the post text.") {
		t.Error("expected post text in output")
	}
	if strings.Contains(s, ">") {
		t.Error("original post should not contain blockquote")
	}
}

func TestWritePost_Repost(t *testing.T) {
	dir := t.TempDir()
	post := adapter.Post{
		ID:             "222",
		Timestamp:      fixedTime,
		Text:           "My comment on this",
		IsRepost:       true,
		OriginalAuthor: "originaluser",
		OriginalText:   "The original content here.",
	}
	if err := WritePost(dir, "u", post); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "u", "20240315-222.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "My comment on this") {
		t.Error("expected repost comment in output")
	}
	if !strings.Contains(s, "> **@originaluser:**") {
		t.Error("expected blockquote with original author")
	}
	if !strings.Contains(s, "The original content here.") {
		t.Error("expected original text in blockquote")
	}
}

func TestWritePost_Overwrite(t *testing.T) {
	dir := t.TempDir()
	post := makePost("333", "First version")
	if err := WritePost(dir, "u", post); err != nil {
		t.Fatal(err)
	}
	post.Text = "Second version"
	if err := WritePost(dir, "u", post); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "u", "20240315-333.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Second version") {
		t.Error("expected overwrite to replace content")
	}
}

func TestWriteIndex(t *testing.T) {
	dir := t.TempDir()
	posts := []adapter.Post{
		makePost("1", "First post content"),
		makePost("2", "Second post content"),
	}
	if err := WriteIndex(dir, "u", posts); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "u", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "20240315-1.md") {
		t.Error("expected first post filename in index")
	}
	if !strings.Contains(s, "20240315-2.md") {
		t.Error("expected second post filename in index")
	}
	if !strings.Contains(s, "First post content") {
		t.Error("expected first post title in index")
	}
	if !strings.Contains(s, "2024-03-15") {
		t.Error("expected date in index")
	}
}
