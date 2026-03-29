package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"website-to-markdown/internal/adapter"
)

// WritePost writes a single post to <outputDir>/<username>/<YYYYMMDD>-<id>.md.
// Existing files are overwritten.
func WritePost(outputDir, username string, post adapter.Post) error {
	dir := filepath.Join(outputDir, username)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	filename := fmt.Sprintf("%s-%s.md", post.Timestamp.Format("20060102"), post.ID)
	path := filepath.Join(dir, filename)
	content := renderPost(post)
	return os.WriteFile(path, []byte(content), 0o644)
}

// WriteIndex writes (or overwrites) an index.md in <outputDir>/<username>/
// listing all given posts with first-line title and timestamp.
func WriteIndex(outputDir, username string, posts []adapter.Post) error {
	dir := filepath.Join(outputDir, username)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Posts by @%s\n\n", username)
	sb.WriteString("| Date | Title | File |\n")
	sb.WriteString("|------|-------|------|\n")
	for _, p := range posts {
		title := firstLine(p.Text)
		if title == "" {
			title = "(no text)"
		}
		filename := fmt.Sprintf("%s-%s.md", p.Timestamp.Format("20060102"), p.ID)
		fmt.Fprintf(&sb, "| %s | %s | [link](%s) |\n",
			p.Timestamp.Format("2006-01-02 15:04"),
			escapeMD(title),
			filename,
		)
	}
	path := filepath.Join(dir, "index.md")
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// renderPost formats a post as Markdown.
func renderPost(post adapter.Post) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Date:** %s\n\n", post.Timestamp.Format("2006-01-02 15:04:05 -0700"))
	sb.WriteString(post.Text)
	if post.IsRepost && post.OriginalText != "" {
		sb.WriteString("\n\n")
		if post.OriginalAuthor != "" {
			fmt.Fprintf(&sb, "> **@%s:** ", post.OriginalAuthor)
		} else {
			sb.WriteString("> ")
		}
		// Indent each line of the original text as a blockquote.
		lines := strings.SplitSeq(post.OriginalText, "\n")
		first := true
		for l := range lines {
			if first {
				sb.WriteString(l)
				first = false
			} else {
				sb.WriteString("\n> ")
				sb.WriteString(l)
			}
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// firstLine returns the first non-empty line of s, truncated to 60 runes.
func firstLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len([]rune(line)) > 60 {
				return string([]rune(line)[:60]) + "…"
			}
			return line
		}
	}
	return ""
}

// escapeMD escapes pipe characters in Markdown table cells.
func escapeMD(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
