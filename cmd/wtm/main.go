package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"website-to-markdown/internal/adapters/weibo"
	"website-to-markdown/internal/browser"
	"website-to-markdown/internal/markdown"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "wtm",
		Short: "website-to-markdown: convert web pages to Markdown",
	}
	root.AddCommand(weiboCmd())
	return root
}

func weiboCmd() *cobra.Command {
	var (
		user    string
		limit   int
		output  string
		cdpHost string
		cdpPort int
	)

	cmd := &cobra.Command{
		Use:   "weibo",
		Short: "Scrape text posts from a Weibo user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWeibo(cmd.Context(), user, limit, output, cdpHost, cdpPort)
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "Weibo username or screen name (required)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of text posts to collect")
	cmd.Flags().StringVar(&output, "output", "./output", "Output directory")
	cmd.Flags().StringVar(&cdpHost, "cdp-host", "localhost", "Chrome DevTools host")
	cmd.Flags().IntVar(&cdpPort, "cdp-port", 9222, "Chrome DevTools port")
	_ = cmd.MarkFlagRequired("user")

	return cmd
}

func runWeibo(ctx context.Context, username string, limit int, outputDir, cdpHost string, cdpPort int) error {
	if strings.ContainsAny(username, `/\`) || strings.Contains(username, "..") {
		return fmt.Errorf("invalid username %q: must not contain path separators or '..'", username)
	}
	b, err := browser.Connect(cdpHost, cdpPort)
	if err != nil {
		return fmt.Errorf("connecting to Chrome at %s:%d — make sure Chrome is running with --remote-debugging-port=%d: %w",
			cdpHost, cdpPort, cdpPort, err)
	}
	defer b.Disconnect()

	a := weibo.New(b)

	if !a.IsLoggedIn(ctx, username) {
		fmt.Println("No valid session found. Starting login flow...")
		if err := a.Login(ctx, username); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	} else {
		fmt.Println("Session valid. Skipping login.")
	}

	fmt.Printf("Collecting up to %d posts from @%s...\n", limit, username)
	posts, err := a.FetchPosts(ctx, username, limit)
	if err != nil {
		return fmt.Errorf("fetching posts: %w", err)
	}
	fmt.Printf("Collected %d posts. Writing Markdown...\n", len(posts))

	for _, post := range posts {
		if err := markdown.WritePost(outputDir, username, post); err != nil {
			return fmt.Errorf("writing post %s: %w", post.ID, err)
		}
	}
	if err := markdown.WriteIndex(outputDir, username, posts); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}

	fmt.Printf("Done. Output written to %s/%s/\n", outputDir, username)
	return nil
}
