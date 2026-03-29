package browser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

func TestSaveLoadSession_RoundTrip(t *testing.T) {
	// Override home dir by pointing the session path to a temp dir.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cookies := []*proto.NetworkCookie{
		{Name: "SUB", Value: "abc123", Domain: ".weibo.cn", Path: "/", Secure: true},
		{Name: "_T_WM", Value: "xyz456", Domain: ".weibo.cn", Path: "/"},
	}

	if err := SaveSession("weibo", "testuser", cookies); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Verify file was created.
	expectedPath := filepath.Join(tmp, ".wtm", "sessions", "weibo-testuser.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("session file not created at %s", expectedPath)
	}

	loaded, err := LoadSession("weibo", "testuser")
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if len(loaded) != len(cookies) {
		t.Fatalf("expected %d cookies, got %d", len(cookies), len(loaded))
	}
	for i, c := range loaded {
		if c.Name != cookies[i].Name || c.Value != cookies[i].Value {
			t.Errorf("cookie[%d]: got {%s=%s}, want {%s=%s}",
				i, c.Name, c.Value, cookies[i].Name, cookies[i].Value)
		}
	}
}

func TestLoadSession_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cookies, err := LoadSession("weibo", "nonexistent")
	if err != nil {
		t.Fatalf("expected nil error for missing session, got: %v", err)
	}
	if cookies != nil {
		t.Errorf("expected nil cookies for missing session, got %v", cookies)
	}
}
