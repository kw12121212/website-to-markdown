package adapter

import (
	"context"
	"time"
)

// Post represents a single scraped post with its text content.
type Post struct {
	ID             string
	Timestamp      time.Time
	Text           string
	IsRepost       bool
	OriginalAuthor string
	OriginalText   string
}

// Adapter is implemented by each site-specific scraper.
type Adapter interface {
	// IsLoggedIn reports whether a valid session exists for the given username.
	// The username is used as the session key (see session persistence).
	IsLoggedIn(ctx context.Context, username string) bool
	// Login performs the site-specific login flow and saves the session under username.
	Login(ctx context.Context, username string) error
	// FetchPosts returns up to limit text-containing posts for the given username.
	FetchPosts(ctx context.Context, username string, limit int) ([]Post, error)
}
